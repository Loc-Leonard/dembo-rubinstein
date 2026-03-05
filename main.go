package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() error {
	host := getenv("DB_HOST", "postgres")
	user := getenv("DB_USER", "user")
	pass := getenv("DB_PASS", "change_me_please")
	name := getenv("DB_NAME", "surveydb")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:5432/%s?sslmode=disable",
		user, pass, host, name,
	)

	log.Println("Connecting to DB with:", connStr) // можно убрать после дебага

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS responses (
            id VARCHAR(8) PRIMARY KEY,
            raw_data JSONB NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`)
	if err != nil {
		return err
	}

	log.Println("✅ БД готова (таблица responses создана)")
	return nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// SurveyResponse структура для ответов респондента (внутренний формат)
type SurveyResponse struct {
	HealthNow       int `json:"health_now"`
	HealthIdeal     int `json:"health_ideal"`
	MindNow         int `json:"mind_now"`
	MindIdeal       int `json:"mind_ideal"`
	CharacterNow    int `json:"character_now"`
	CharacterIdeal  int `json:"character_ideal"`
	AuthorityNow    int `json:"authority_now"`
	AuthorityIdeal  int `json:"authority_ideal"`
	HandsNow        int `json:"hands_now"`
	HandsIdeal      int `json:"hands_ideal"`
	AppearanceNow   int `json:"appearance_now"`
	AppearanceIdeal int `json:"appearance_ideal"`
	ConfidenceNow   int `json:"confidence_now"`
	ConfidenceIdeal int `json:"confidence_ideal"`
}

// Scale описывает одну шкалу с пояснениями (формат хранения в БД)
type Scale struct {
	Key         string `json:"key"`         // системное имя, например "mind_now"
	Title       string `json:"title"`       // человекочитаемое имя
	Description string `json:"description"` // краткое пояснение
	Value       int    `json:"value"`       // значение
}

// StoredResult — то, что кладём в БД в raw_data
type StoredResult struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Scales    []Scale   `json:"scales"`
}

// CalculatedResults структура с рассчитанными показателями
type CalculatedResults struct {
	Responses SurveyResponse       `json:"responses"`
	Diff      map[string]int       `json:"diff"`
	AvgLevel  float64              `json:"avg_aspiration_level"`
	AvgSelf   float64              `json:"avg_self_esteem"`
	AvgDiff   float64              `json:"avg_difference"`
	Levels    map[string]LevelInfo `json:"levels"`
}

// LevelInfo информация об уровне для каждой шкалы
type LevelInfo struct {
	Scale      string `json:"scale"`
	Now        int    `json:"now"`
	Ideal      int    `json:"ideal"`
	Diff       int    `json:"diff"`
	NowLevel   string `json:"now_level"`
	IdealLevel string `json:"ideal_level"`
}

// DataStore простое хранилище в памяти (сейчас не используется, но оставим)
type DataStore struct {
	mu      sync.RWMutex
	records map[string]SurveyResponse
}

var (
	store = &DataStore{
		records: make(map[string]SurveyResponse),
	}
	templates = template.Must(template.ParseGlob("templates/*.html"))
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := initDB(); err != nil {
		log.Fatal("PostgreSQL:", err)
	}
	defer db.Close()

	// Статические файлы
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Маршруты
	http.HandleFunc("/", handleSurvey)
	http.HandleFunc("/result", handleResult)
	http.HandleFunc("/api/save", handleSave)

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleSurvey отдает страницу с опросом
func handleSurvey(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

// handleSave сохраняет ответы и возвращает ID
func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var response SurveyResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		http.Error(w, "Ошибка декодирования JSON", http.StatusBadRequest)
		return
	}

	if !validateResponse(response) {
		http.Error(w, "Не все шкалы заполнены", http.StatusBadRequest)
		return
	}

	id := generateID()

	// Собираем человекочитаемую структуру для БД
	stored := buildStoredResult(id, response)

	rawJson, err := json.Marshal(stored)
	if err != nil {
		http.Error(w, "Ошибка сериализации данных", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO responses (id, raw_data) VALUES ($1, $2)", id, rawJson)
	if err != nil {
		http.Error(w, "Ошибка БД", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":        id,
		"share_url": fmt.Sprintf("/result?v=%s", id),
	})
}

// handleResult обработчик для страницы результатов
func handleResult(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("v")
	if id == "" {
		http.Error(w, "ID не указан", http.StatusBadRequest)
		return
	}

	var rawJSON []byte
	err := db.QueryRow("SELECT raw_data FROM responses WHERE id=$1", id).Scan(&rawJSON)
	if err != nil {
		http.Error(w, "Результат не найден", http.StatusNotFound)
		return
	}

	// 1. Декодируем сохранённую структуру
	var stored StoredResult
	if err := json.Unmarshal(rawJSON, &stored); err != nil {
		http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
		return
	}

	// 2. Собираем SurveyResponse для расчётов
	response := storedToSurvey(stored)

	// 3. Считаем результаты по методике
	results := calculateResults(response)

	// 4. Рендерим шаблон result.html с results
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "result.html", results); err != nil {
		http.Error(w, "Ошибка рендеринга шаблона", http.StatusInternalServerError)
		return
	}
}

// buildStoredResult собирает человекочитаемый объект для хранения в БД
func buildStoredResult(id string, r SurveyResponse) StoredResult {
	return StoredResult{
		ID:        id,
		CreatedAt: time.Now(),
		Scales: []Scale{
			{
				Key:         "health_now",
				Title:       "Здоровье (текущее)",
				Description: "Как респондент оценивает своё здоровье сейчас",
				Value:       r.HealthNow,
			},
			{
				Key:         "health_ideal",
				Title:       "Здоровье (идеал)",
				Description: "Какое здоровье респондент хотел бы иметь",
				Value:       r.HealthIdeal,
			},
			{
				Key:         "mind_now",
				Title:       "Ум/способности (текущее)",
				Description: "Самооценка своих умственных способностей сейчас",
				Value:       r.MindNow,
			},
			{
				Key:         "mind_ideal",
				Title:       "Ум/способности (идеал)",
				Description: "Желаемый уровень умственных способностей",
				Value:       r.MindIdeal,
			},
			{
				Key:         "character_now",
				Title:       "Характер (текущее)",
				Description: "Как респондент оценивает свой характер сейчас",
				Value:       r.CharacterNow,
			},
			{
				Key:         "character_ideal",
				Title:       "Характер (идеал)",
				Description: "Желаемый характер",
				Value:       r.CharacterIdeal,
			},
			{
				Key:         "authority_now",
				Title:       "Авторитет у сверстников (текущее)",
				Description: "Какой авторитет, по мнению респондента, у него есть сейчас",
				Value:       r.AuthorityNow,
			},
			{
				Key:         "authority_ideal",
				Title:       "Авторитет у сверстников (идеал)",
				Description: "Какой авторитет респондент хотел бы иметь",
				Value:       r.AuthorityIdeal,
			},
			{
				Key:         "hands_now",
				Title:       "Умелые руки (текущее)",
				Description: "Оценка своих практических навыков сейчас",
				Value:       r.HandsNow,
			},
			{
				Key:         "hands_ideal",
				Title:       "Умелые руки (идеал)",
				Description: "Желаемый уровень практических навыков",
				Value:       r.HandsIdeal,
			},
			{
				Key:         "appearance_now",
				Title:       "Внешность (текущее)",
				Description: "Оценка своей внешности сейчас",
				Value:       r.AppearanceNow,
			},
			{
				Key:         "appearance_ideal",
				Title:       "Внешность (идеал)",
				Description: "Желаемая внешность",
				Value:       r.AppearanceIdeal,
			},
			{
				Key:         "confidence_now",
				Title:       "Уверенность в себе (текущее)",
				Description: "Как респондент оценивает свою уверенность сейчас",
				Value:       r.ConfidenceNow,
			},
			{
				Key:         "confidence_ideal",
				Title:       "Уверенность в себе (идеал)",
				Description: "Желаемый уровень уверенности",
				Value:       r.ConfidenceIdeal,
			},
		},
	}
}

// storedToSurvey восстанавливает SurveyResponse из сохранённого результата
func storedToSurvey(stored StoredResult) SurveyResponse {
	var r SurveyResponse

	for _, s := range stored.Scales {
		switch s.Key {
		case "health_now":
			r.HealthNow = s.Value
		case "health_ideal":
			r.HealthIdeal = s.Value
		case "mind_now":
			r.MindNow = s.Value
		case "mind_ideal":
			r.MindIdeal = s.Value
		case "character_now":
			r.CharacterNow = s.Value
		case "character_ideal":
			r.CharacterIdeal = s.Value
		case "authority_now":
			r.AuthorityNow = s.Value
		case "authority_ideal":
			r.AuthorityIdeal = s.Value
		case "hands_now":
			r.HandsNow = s.Value
		case "hands_ideal":
			r.HandsIdeal = s.Value
		case "appearance_now":
			r.AppearanceNow = s.Value
		case "appearance_ideal":
			r.AppearanceIdeal = s.Value
		case "confidence_now":
			r.ConfidenceNow = s.Value
		case "confidence_ideal":
			r.ConfidenceIdeal = s.Value
		}
	}

	return r
}

// calculateResults расчет всех показателей по методике
func calculateResults(r SurveyResponse) CalculatedResults {
	scales := []struct {
		name  string
		now   int
		ideal int
	}{
		{"Здоровье", r.HealthNow, r.HealthIdeal},
		{"Ум/способности", r.MindNow, r.MindIdeal},
		{"Характер", r.CharacterNow, r.CharacterIdeal},
		{"Авторитет у сверстников", r.AuthorityNow, r.AuthorityIdeal},
		{"Умелые руки", r.HandsNow, r.HandsIdeal},
		{"Внешность", r.AppearanceNow, r.AppearanceIdeal},
		{"Уверенность в себе", r.ConfidenceNow, r.ConfidenceIdeal},
	}

	results := CalculatedResults{
		Responses: r,
		Diff:      make(map[string]int),
		Levels:    make(map[string]LevelInfo),
	}

	var sumNow, sumIdeal, sumDiff float64
	count := 0

	for _, scale := range scales {
		diff := scale.ideal - scale.now
		scaleName := strings.ToLower(scale.name)

		results.Diff[scaleName] = diff
		results.Levels[scaleName] = LevelInfo{
			Scale:      scale.name,
			Now:        scale.now,
			Ideal:      scale.ideal,
			Diff:       diff,
			NowLevel:   getSelfEsteemLevel(scale.now),
			IdealLevel: getAspirationLevel(scale.ideal),
		}

		sumNow += float64(scale.now)
		sumIdeal += float64(scale.ideal)
		sumDiff += float64(diff)
		count++
	}

	if count > 0 {
		results.AvgSelf = sumNow / float64(count)
		results.AvgLevel = sumIdeal / float64(count)
		results.AvgDiff = sumDiff / float64(count)
	}

	return results
}

// getSelfEsteemLevel определение уровня самооценки
func getSelfEsteemLevel(value int) string {
	switch {
	case value < 45:
		return "Заниженная (группа риска)"
	case value <= 74:
		return "Адекватная"
	default:
		return "Завышенная"
	}
}

// getAspirationLevel определение уровня притязаний
func getAspirationLevel(value int) string {
	switch {
	case value < 60:
		return "Заниженный"
	case value <= 89:
		return "Оптимальный"
	default:
		return "Нереалистичный"
	}
}

// validateResponse проверка заполнения всех шкал
func validateResponse(r SurveyResponse) bool {
	// Пока упрощённо
	return true
}

// generateID генерация уникального ID
func generateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

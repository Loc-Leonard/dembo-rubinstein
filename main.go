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

// ============ ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ============
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func generateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// ============ ФУНКЦИИ ИНИЦИАЛИЗАЦИИ ============
func initDB() error {
	host := getenv("DB_HOST", "postgres")
	user := getenv("DB_USER", "user")
	pass := getenv("DB_PASS", "change_me_please")
	name := getenv("DB_NAME", "surveydb")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:5432/%s?sslmode=disable",
		user, pass, host, name,
	)

	log.Println("Connecting to DB with:", connStr)

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

// ============ СТРУКТУРЫ ДАННЫХ ============
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

type Scale struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Value       int    `json:"value"`
}

type StoredResult struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Scales    []Scale   `json:"scales"`
}

type CalculatedResults struct {
	Responses SurveyResponse       `json:"responses"`
	Diff      map[string]int       `json:"diff"`
	AvgLevel  float64              `json:"avg_aspiration_level"`
	AvgSelf   float64              `json:"avg_self_esteem"`
	AvgDiff   float64              `json:"avg_difference"`
	Levels    map[string]LevelInfo `json:"levels"`
}

type LevelInfo struct {
	Scale      string `json:"scale"`
	Now        int    `json:"now"`
	Ideal      int    `json:"ideal"`
	Diff       int    `json:"diff"`
	NowLevel   string `json:"now_level"`
	IdealLevel string `json:"ideal_level"`
}

type DataStore struct {
	mu      sync.RWMutex
	records map[string]SurveyResponse
}

var (
	store = &DataStore{
		records: make(map[string]SurveyResponse),
	}
	templates *template.Template
)

// ============ ФУНКЦИИ ДЛЯ ШАБЛОНОВ ============
func getSelfEsteemDescription(avg float64) string {
	switch {
	case avg < 45:
		return "Заниженная самооценка (группа риска). Рекомендуется консультация специалиста."
	case avg <= 74:
		return "Адекватная самооценка. Реалистичная оценка своих возможностей."
	default:
		return "Завышенная самооценка. Может указывать на личностную незрелость."
	}
}

func getAspirationDescription(avg float64) string {
	switch {
	case avg < 60:
		return "Заниженный уровень притязаний. Индикатор неблагоприятного развития личности."
	case avg <= 89:
		return "Оптимальный уровень притязаний. Реалистичное представление о своих возможностях."
	default:
		return "Нереалистичный уровень притязаний. Может указывать на некритичность."
	}
}

func getDiffDescription(avg float64) string {
	switch {
	case avg > 15:
		return "Большое расхождение между притязаниями и самооценкой. Возможны нереалистичные цели."
	case avg > 5:
		return "Умеренное расхождение. Здоровое стремление к развитию."
	case avg >= 0:
		return "Гармоничное соотношение. Уровень притязаний немного выше самооценки."
	default:
		return "Отрицательное расхождение. Уровень притязаний ниже самооценки."
	}
}

// ============ ФУНКЦИИ БИЗНЕС-ЛОГИКИ ============
func validateResponse(r SurveyResponse) bool {
	// Пока упрощённо
	return true
}

func buildStoredResult(id string, r SurveyResponse) StoredResult {
	return StoredResult{
		ID:        id,
		CreatedAt: time.Now(),
		Scales: []Scale{
			{Key: "health_now", Title: "Здоровье (текущее)", Description: "Как респондент оценивает своё здоровье сейчас", Value: r.HealthNow},
			{Key: "health_ideal", Title: "Здоровье (идеал)", Description: "Какое здоровье респондент хотел бы иметь", Value: r.HealthIdeal},
			{Key: "mind_now", Title: "Ум/способности (текущее)", Description: "Самооценка своих умственных способностей сейчас", Value: r.MindNow},
			{Key: "mind_ideal", Title: "Ум/способности (идеал)", Description: "Желаемый уровень умственных способностей", Value: r.MindIdeal},
			{Key: "character_now", Title: "Характер (текущее)", Description: "Как респондент оценивает свой характер сейчас", Value: r.CharacterNow},
			{Key: "character_ideal", Title: "Характер (идеал)", Description: "Желаемый характер", Value: r.CharacterIdeal},
			{Key: "authority_now", Title: "Авторитет у сверстников (текущее)", Description: "Какой авторитет, по мнению респондента, у него есть сейчас", Value: r.AuthorityNow},
			{Key: "authority_ideal", Title: "Авторитет у сверстников (идеал)", Description: "Какой авторитет респондент хотел бы иметь", Value: r.AuthorityIdeal},
			{Key: "hands_now", Title: "Умелые руки (текущее)", Description: "Оценка своих практических навыков сейчас", Value: r.HandsNow},
			{Key: "hands_ideal", Title: "Умелые руки (идеал)", Description: "Желаемый уровень практических навыков", Value: r.HandsIdeal},
			{Key: "appearance_now", Title: "Внешность (текущее)", Description: "Оценка своей внешности сейчас", Value: r.AppearanceNow},
			{Key: "appearance_ideal", Title: "Внешность (идеал)", Description: "Желаемая внешность", Value: r.AppearanceIdeal},
			{Key: "confidence_now", Title: "Уверенность в себе (текущее)", Description: "Как респондент оценивает свою уверенность сейчас", Value: r.ConfidenceNow},
			{Key: "confidence_ideal", Title: "Уверенность в себе (идеал)", Description: "Желаемый уровень уверенности", Value: r.ConfidenceIdeal},
		},
	}
}

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

// ============ HTTP-ОБРАБОТЧИКИ ============
func handleSurvey(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

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

	var stored StoredResult
	if err := json.Unmarshal(rawJSON, &stored); err != nil {
		http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
		return
	}

	response := storedToSurvey(stored)
	results := calculateResults(response)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "result.html", results); err != nil {
		http.Error(w, "Ошибка рендеринга шаблона", http.StatusInternalServerError)
		return
	}
}

// ============ ТОЧКА ВХОДА ============
func main() {
	rand.Seed(time.Now().UnixNano())

	if err := initDB(); err != nil {
		log.Fatal("PostgreSQL:", err)
	}
	defer db.Close()

	// Регистрируем функции для шаблонов
	funcMap := template.FuncMap{
		"getSelfEsteemDescription": getSelfEsteemDescription,
		"getAspirationDescription": getAspirationDescription,
		"getDiffDescription":       getDiffDescription,
	}

	// Загружаем шаблоны с функциями
	templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

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

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() error {
	connStr := "postgres://user:pass@postgres:5432/surveydb?sslmode=disable" // ? не &

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err // ❌ Только если ошибка подключения
	}

	// ✅ Проверяем подключение
	if err = db.Ping(); err != nil {
		return err
	}

	// ✅ Создаём таблицу
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
	return nil // ✅ Успех
}

// SurveyResponse структура для ответов респондента
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

// DataStore простое хранилище в памяти (для варианта Б)
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
	// Инициализация генератора случайных чисел
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

// handleSave сохраняет ответы и возвращает ID (вариант Б)
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

	// Валидация: все поля должны быть заполнены
	if !validateResponse(response) {
		http.Error(w, "Не все шкалы заполнены", http.StatusBadRequest)
		return
	}

	// Генерация ID
	id := generateID()

	rawJson, _ := json.Marshal(response)
	_, err := db.Exec("INSERT INTO responses (id, raw_data) VALUES ($1, $2)", id, rawJson)
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
	// Получаем ID из ссылки (?v=abc123)
	id := r.URL.Query().Get("v")
	if id == "" {
		http.Error(w, "ID не указан", http.StatusBadRequest)
		return
	}

	// ✅ Читаем СЫРЫЕ данные из PostgreSQL
	var rawJSON []byte
	err := db.QueryRow("SELECT raw_data FROM responses WHERE id=$1", id).Scan(&rawJSON)
	if err != nil {
		http.Error(w, "Результат не найден", http.StatusNotFound)
		return
	}

	// ✅ Парсим сырые данные
	var response SurveyResponse
	if err := json.Unmarshal(rawJSON, &response); err != nil {
		http.Error(w, "Ошибка данных", http.StatusInternalServerError)
		return
	}

	// ✅ ТВОЯ обработка Дембо-Рубинштейна (остаётся!)
	results := calculateResults(response)

	// ✅ Рендер шаблона (то же самое, что видел респондент)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "result.html", results); err != nil {
		http.Error(w, "Ошибка шаблона", http.StatusInternalServerError)
	}
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

	for i, scale := range scales {
		// Первая шкала (здоровье) не учитывается в средних показателях
		if i == 0 {
			continue
		}

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
	// Проверяем, что все значения установлены (не нулевые, но могут быть 0)
	// В реальном проекте нужно добавить проверку на -1 для незаполненных
	return true // Упрощенно для примера
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

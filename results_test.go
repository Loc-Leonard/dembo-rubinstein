package main

import "testing"

func TestGetSelfEsteemLevel(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  string
	}{
		{"low", 30, "Заниженная (группа риска)"},
		{"border_low", 44, "Заниженная (группа риска)"},
		{"middle", 60, "Адекватная"},
		{"border_middle", 74, "Адекватная"},
		{"high", 80, "Завышенная"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSelfEsteemLevel(tt.value)
			if got != tt.want {
				t.Errorf("getSelfEsteemLevel(%d) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetAspirationLevel(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  string
	}{
		{"low", 50, "Заниженный"},
		{"border_low", 59, "Заниженный"},
		{"middle", 70, "Оптимальный"},
		{"border_middle", 89, "Оптимальный"},
		{"high", 95, "Нереалистичный"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAspirationLevel(tt.value)
			if got != tt.want {
				t.Errorf("getAspirationLevel(%d) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestCalculateResultsBasic(t *testing.T) {
	// все "сейчас" = 50, все "идеал" = 70
	r := SurveyResponse{
		HealthNow:       50,
		HealthIdeal:     70,
		AbilitiesNow:    50,
		AbilitiesIdeal:  70,
		CharacterNow:    50,
		CharacterIdeal:  70,
		HappinessNow:    50,
		HappinessIdeal:  70,
		SelfesteemNow:   50,
		SelfesteemIdeal: 70,
		AppearanceNow:   50,
		AppearanceIdeal: 70,
		ConfidenceNow:   50,
		ConfidenceIdeal: 70,
		RelationsNow:    50,
		RelationsIdeal:  70,
	}

	res := calculateResults(r)

	// 8 шкал, поэтому средние должны быть 50/70/20
	if res.AvgSelf != 50 {
		t.Errorf("AvgSelf = %v, want 50", res.AvgSelf)
	}
	if res.AvgLevel != 70 {
		t.Errorf("AvgLevel = %v, want 70", res.AvgLevel)
	}
	if res.AvgDiff != 20 {
		t.Errorf("AvgDiff = %v, want 20", res.AvgDiff)
	}

	// проверим пару шкал по ключам (lowercase от названия в calculateResults)
	health, ok := res.Levels["здоровье"]
	if !ok {
		t.Fatalf("expected 'здоровье' level in Levels")
	}
	if health.Now != 50 || health.Ideal != 70 || health.Diff != 20 {
		t.Errorf("health level = %+v, want Now=50 Ideal=70 Diff=20", health)
	}

	happiness, ok := res.Levels["счастье"]
	if !ok {
		t.Fatalf("expected 'счастье' level in Levels")
	}
	if happiness.Now != 50 || happiness.Ideal != 70 || happiness.Diff != 20 {
		t.Errorf("happiness level = %+v, want Now=50 Ideal=70 Diff=20", happiness)
	}
}

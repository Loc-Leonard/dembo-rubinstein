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
	// искусственный, но простой кейс:
	// все "сейчас" = 50, все "идеал" = 70
	r := SurveyResponse{
		HealthNow:       50,
		HealthIdeal:     70,
		MindNow:         50,
		MindIdeal:       70,
		CharacterNow:    50,
		CharacterIdeal:  70,
		AuthorityNow:    50,
		AuthorityIdeal:  70,
		HandsNow:        50,
		HandsIdeal:      70,
		AppearanceNow:   50,
		AppearanceIdeal: 70,
		ConfidenceNow:   50,
		ConfidenceIdeal: 70,
	}

	res := calculateResults(r)

	// Проверяем средние
	// У тебя сейчас в calculateResults считаются все 7 шкал, включая здоровье.
	// sumNow = 7 * 50 = 350, sumIdeal = 7 * 70 = 490, sumDiff = 7 * 20 = 140
	if res.AvgSelf != 50 {
		t.Errorf("AvgSelf = %v, want 50", res.AvgSelf)
	}
	if res.AvgLevel != 70 {
		t.Errorf("AvgLevel = %v, want 70", res.AvgLevel)
	}
	if res.AvgDiff != 20 {
		t.Errorf("AvgDiff = %v, want 20", res.AvgDiff)
	}

	// Проверим пару шкал в Levels
	health, ok := res.Levels["здоровье"]
	if !ok {
		t.Fatalf("expected health level in Levels")
	}
	if health.Now != 50 || health.Ideal != 70 || health.Diff != 20 {
		t.Errorf("health level = %+v, want Now=50 Ideal=70 Diff=20", health)
	}

	mind, ok := res.Levels["ум/способности"]
	if !ok {
		t.Fatalf("expected mind level in Levels")
	}
	if mind.Now != 50 || mind.Ideal != 70 || mind.Diff != 20 {
		t.Errorf("mind level = %+v, want Now=50 Ideal=70 Diff=20", mind)
	}
}

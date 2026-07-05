package models

import (
	"encoding/json"
	"time"
)

type SeoReport struct {
	ID        string          `json:"id"`
	URL       string          `json:"url"`
	Title     string          `json:"title"`
	Summary   string          `json:"summary"`
	Score     int             `json:"score"`
	Report    json.RawMessage `json:"report"`
	CreatedAt time.Time       `json:"created_at"`
}

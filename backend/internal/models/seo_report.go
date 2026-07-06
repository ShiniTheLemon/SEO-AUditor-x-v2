package models

import (
	"encoding/json"
	"time"
)

type SeoReport struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id,omitempty"`
	URL           string          `json:"url"`
	Title         string          `json:"title"`
	Summary       string          `json:"summary"`
	Score         int             `json:"score"`
	Report        json.RawMessage `json:"report"`
	DeviceDetails json.RawMessage `json:"device_details,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

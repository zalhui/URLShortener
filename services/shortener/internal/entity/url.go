package entity

import "time"

type URL struct {
	UUID        string    `json:"uuid"`
	UserID      string    `json:"user_id"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	CreatedAt   time.Time `json:"created_at"`
}

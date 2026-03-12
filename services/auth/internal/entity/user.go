package entity

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID             string
	UserID         string
	TokenHash      string
	UserAgent      string
	IPAddress      string
	ExpiresAt      time.Time
	CreatedAt      time.Time
	RevokedAt      *time.Time
	ReplacedByHash *string
}

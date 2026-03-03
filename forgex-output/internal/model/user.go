package model

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

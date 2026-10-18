package model

import "time"

type User struct {
	ID          int
	Provider    string
	ProviderSub string
	Email       string
	Name        string
	AvatarURL   *string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MeResponse struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

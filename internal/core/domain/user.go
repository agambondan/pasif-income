package domain

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Platform struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AuthType    string `json:"auth_type"` // "oauth", "api_key", "password"
	Description string `json:"description"`
}

type ConnectedAccount struct {
	ID           string    `json:"id"`
	UserID       int       `json:"user_id"`
	PlatformID   string    `json:"platform_id"`
	DisplayName  string    `json:"display_name"`
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	Expiry       time.Time `json:"expiry"`
	CreatedAt    time.Time `json:"created_at"`
}

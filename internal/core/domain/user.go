package domain

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Platform struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Methods     []string `json:"supported_methods"` // ["api", "chromium_profile"]
	Description string   `json:"description"`
}

type ConnectedAccount struct {
	ID            string    `json:"id"`
	UserID        int       `json:"user_id"`
	PlatformID    string    `json:"platform_id"`
	DisplayName   string    `json:"display_name"`
	AuthMethod    string    `json:"auth_method"` // "api" or "chromium_profile"
	Email         string    `json:"email"`
	ProfilePath   string    `json:"profile_path"` // Chromium profile directory for browser automation
	BrowserStatus string    `json:"browser_status,omitempty"`
	AccessToken   string    `json:"-"`
	RefreshToken  string    `json:"-"`
	Expiry        time.Time `json:"expiry"`
	CreatedAt     time.Time `json:"created_at"`
}

const (
	AuthMethodAPI             = "api"
	AuthMethodChromiumProfile = "chromium_profile"
)

type BrowserProfile struct {
	ID          string    `json:"id"`
	UserID      int       `json:"user_id"`
	PlatformID  string    `json:"platform_id"`
	Email       string    `json:"email"`
	ProfilePath string    `json:"profile_path"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

package model

type PasswordPolicy struct {
	MinLength           int      `json:"min_length"`
	RequireUppercase    bool     `json:"require_uppercase"`
	RequireLowercase    bool     `json:"require_lowercase"`
	RequireNumbers      bool     `json:"require_numbers"`
	RequireSpecialChars bool     `json:"require_special_chars"`
	MaxAge              int      `json:"max_age_days"`  // Password expiry in days
	HistoryCount        int      `json:"history_count"` // Number of previous passwords to remember
	AllowedSpecialChars string   `json:"allowed_special_chars"`
	BlockedPasswords    []string `json:"blocked_passwords"` // Common/weak passwords to block
}

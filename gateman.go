package gateman

import "database/sql"

type SQLite struct{ *sql.DB }
type AuthPayload struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
}

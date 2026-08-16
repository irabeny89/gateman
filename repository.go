package gateman

import "context"

// User represents a user in the system
type User struct {
	ID       string
	Email    string
	PassHash string
	Role     string
}

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, u *User) error
}

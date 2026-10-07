package models

import "time"

type Role struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UserCount   int       `db:"user_count"`
	Permissions []string  `db:"-"`
}

type User struct {
	ID           int64     `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	FullName     string    `db:"full_name"`
	RoleID       int64     `db:"role_id"`
	RoleName     string    `db:"role_name"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
}

type UserSession struct {
	ID        string    `db:"id"`
	UserID    int64     `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
}

type ActivityLog struct {
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	Username   string    `db:"username"`
	Action     string    `db:"action"`
	EntityType string    `db:"entity_type"`
	EntityID   string    `db:"entity_id"`
	Details    string    `db:"details"`
	IPAddress  string    `db:"ip_address"`
	CreatedAt  time.Time `db:"created_at"`
}

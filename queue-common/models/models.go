package models

import "time"

// Entity represents a customer record.
type Entity struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	Email       *string   `json:"email,omitempty"`
	JoiningDate time.Time `json:"joining_date"`
}

// User represents an application user (auth credentials stored as password_hash in DB, not exposed via API).
type User struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Requests

type CreateEntityRequest struct {
	Name  string  `json:"name"`
	Phone string  `json:"phone"`
	Email *string `json:"email,omitempty"`
}

type UpdateEntityRequest struct {
	Name        *string    `json:"name,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Email       *string    `json:"email,omitempty"`
	JoiningDate *time.Time `json:"joining_date,omitempty"`
}

type CreateUserRequest struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	UserID   *string `json:"user_id,omitempty"`
	Name     *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
}

// --- Auth

// LoginRequest validates credentials for an existing user.
// Email is the login identifier; password is compared against users.password_hash (bcrypt).
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Room represents a resource/room from the nodequeue database.
type Room struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Capacity  int        `json:"capacity"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateRoomRequest struct {
	// ID is optional; if omitted the service will generate a stable id.
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type UpdateRoomRequest struct {
	Name     *string `json:"name,omitempty"`
	Capacity *int    `json:"capacity,omitempty"`
	// Allow soft-undelete if desired later.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

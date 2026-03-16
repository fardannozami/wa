package domain

import "time"

type User struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	GoogleID  string    `json:"google_id" gorm:"uniqueIndex;not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRepository interface {
	Create(user *User) error
	FindByGoogleID(googleID string) (*User, error)
	FindByID(id string) (*User, error)
}

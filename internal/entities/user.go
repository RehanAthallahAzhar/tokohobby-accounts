package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `gorm:"not null"`
	Username    string    `gorm:"unique;not null"`
	Email       string    `gorm:"unique;not null"`
	Role        string    `gorm:"type:varchar(50);default:'user'"`
	Address     string
	PhoneNumber string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

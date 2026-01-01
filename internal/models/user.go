package models

import "github.com/google/uuid"

type UserRegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}

type UserResponse struct {
	Id           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Address      string    `json:"address"`
	PhoneNumber  string    `json:"phone_number"`
	Role         string    `json:"role"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

type UserUpdateRequest struct {
	Name        string `json:"name" validate:"required"`
	Username    string `json:"username" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password,omitempty"`
	Address     string `json:"address,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

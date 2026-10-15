package handlers

import (
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
)

// DTO Converter
func toUserResponse(user *entities.User) *models.UserResponse {
	return &models.UserResponse{
		Id:          user.ID,
		Name:        user.Name,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		Address:     user.Address,
		PhoneNumber: user.PhoneNumber,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}
}

func toUserResponses(users []entities.User) []models.UserResponse {
	res := make([]models.UserResponse, len(users))
	for i, user := range users {
		res[i] = *toUserResponse(&user)
	}
	return res
}

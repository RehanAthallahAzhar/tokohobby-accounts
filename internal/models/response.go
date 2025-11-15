package models

type ErrorResponse struct {
	Error any `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

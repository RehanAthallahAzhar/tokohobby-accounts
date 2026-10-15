package models

type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ErrorResponse struct {
	Error any `json:"error"`
}

type FieldErrorResponse struct {
	Field string `json:"field"`
	Msg   string `json:"message"`
}

type ErrorDetailsResponse struct {
	Error   string               `json:"error"`
	Details []FieldErrorResponse `json:"details,omitempty"`
}

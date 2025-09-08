package domain

type ApiResponse struct {
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data,omitempty"`
}

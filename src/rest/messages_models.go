package rest

import "time"

type QueueSMSRequest struct {
	Topic    string `json:"topic" validate:"required"`
	ToNumber string `json:"to_number" validate:"required"`
	Body     string `json:"body" validate:"required"`
}

type QueueSMSResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

type MessageDetail struct {
	ID            string     `json:"id"`
	Topic         string     `json:"topic"`
	ToNumber      string     `json:"to_number"`
	Body          string     `json:"body"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
}

type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type MessagesListResponse struct {
	Data       []MessageDetail `json:"data"`
	Pagination PaginationInfo  `json:"pagination"`
}

type MessageFilters struct {
	Topic    string
	ToNumber string
	Keyword  string
	Status   string
	Page     int
	Limit    int
}

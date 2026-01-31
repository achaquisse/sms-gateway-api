package rest

type PollMessage struct {
	ID       string `json:"id"`
	ToNumber string `json:"to_number"`
	Body     string `json:"body"`
}

type PollResponse struct {
	Messages []PollMessage `json:"messages"`
}

type StatusUpdateRequest struct {
	Status string  `json:"status" validate:"required"`
	Reason *string `json:"reason,omitempty"`
}

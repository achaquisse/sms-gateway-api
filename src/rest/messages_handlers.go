package rest

import (
	"math"
	"sms-gateway-api/db"

	"github.com/gofiber/fiber/v2"
)

func QueueSMSHandler(c *fiber.Ctx) error {
	var req QueueSMSRequest
	if err := c.BodyParser(&req); err != nil {
		return ReturnBadRequest(c, "Invalid request body")
	}

	if req.Topic == "" {
		return ReturnBadRequest(c, "Topic is required")
	}

	if req.ToNumber == "" {
		return ReturnBadRequest(c, "to_number is required")
	}

	if req.Body == "" {
		return ReturnBadRequest(c, "Body is required")
	}

	message, err := db.CreateMessage(req.Topic, req.ToNumber, req.Body)
	if err != nil {
		return ReturnInternalError(c, "Failed to queue message")
	}

	response := QueueSMSResponse{
		Message: "Message queued successfully",
		ID:      message.ID,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func ListMessagesHandler(c *fiber.Ctx) error {
	topic := c.Query("topic")
	toNumber := c.Query("to_number")
	keyword := c.Query("keyword")
	status := c.Query("status")

	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	limit := c.QueryInt("limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if status != "" && status != "pending" && status != "sent" && status != "failed" {
		return ReturnBadRequest(c, "Invalid status value. Must be one of: pending, sent, failed")
	}

	offset := (page - 1) * limit

	filters := db.MessageFilters{
		Topic:    topic,
		ToNumber: toNumber,
		Keyword:  keyword,
		Status:   status,
		Limit:    limit,
		Offset:   offset,
	}

	messages, err := db.GetMessages(filters)
	if err != nil {
		return ReturnInternalError(c, "Failed to retrieve messages")
	}

	total, err := db.CountMessages(filters)
	if err != nil {
		return ReturnInternalError(c, "Failed to count messages")
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	messageDetails := make([]MessageDetail, len(messages))
	for i, msg := range messages {
		messageDetails[i] = MessageDetail{
			ID:            msg.ID,
			Topic:         msg.Topic,
			ToNumber:      msg.ToNumber,
			Body:          msg.Body,
			Status:        msg.Status,
			CreatedAt:     msg.CreatedAt,
			SentAt:        msg.SentAt,
			FailedAt:      msg.FailedAt,
			FailureReason: msg.FailureReason,
		}
	}

	response := MessagesListResponse{
		Data: messageDetails,
		Pagination: PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	return c.JSON(response)
}

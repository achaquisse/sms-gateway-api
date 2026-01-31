package rest

import (
	"database/sql"
	"sms-gateway-api/db"

	"github.com/gofiber/fiber/v2"
)

func PollMessagesHandler(c *fiber.Ctx) error {
	deviceKey := c.Get("X-Device-Key")
	if deviceKey == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing device key",
		})
	}

	device, err := db.GetDeviceByKey(deviceKey)
	if err != nil {
		return ReturnInternalError(c, "Failed to authenticate device")
	}

	if device == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing device key",
		})
	}

	topics, err := db.GetDeviceTopics(device.ID)
	if err != nil {
		return ReturnInternalError(c, "Failed to retrieve device topics")
	}

	messages, err := db.GetPendingMessagesForDevice(device.ID, topics)
	if err != nil {
		return ReturnInternalError(c, "Failed to retrieve pending messages")
	}

	if err := db.UpdateDeviceLastPoll(device.ID); err != nil {
		return ReturnInternalError(c, "Failed to update device poll time")
	}

	pollMessages := make([]PollMessage, len(messages))
	for i, msg := range messages {
		pollMessages[i] = PollMessage{
			ID:       msg.ID,
			ToNumber: msg.ToNumber,
			Body:     msg.Body,
		}
	}

	response := PollResponse{
		Messages: pollMessages,
	}

	return c.JSON(response)
}

func UpdateMessageStatusHandler(c *fiber.Ctx) error {
	deviceKey := c.Get("X-Device-Key")
	if deviceKey == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing device key",
		})
	}

	device, err := db.GetDeviceByKey(deviceKey)
	if err != nil {
		return ReturnInternalError(c, "Failed to authenticate device")
	}

	if device == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing device key",
		})
	}

	messageID := c.Params("messageId")
	if messageID == "" {
		return ReturnBadRequest(c, "messageId is required")
	}

	var req StatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return ReturnBadRequest(c, "Invalid request body")
	}

	if req.Status == "" {
		return ReturnBadRequest(c, "status is required")
	}

	if req.Status != "sent" && req.Status != "failed" {
		return ReturnBadRequest(c, "Invalid status. Must be one of: sent, failed")
	}

	if req.Status == "failed" && (req.Reason == nil || *req.Reason == "") {
		emptyReason := "Unknown error"
		req.Reason = &emptyReason
	}

	err = db.UpdateMessageStatus(messageID, req.Status, req.Reason)
	if err == sql.ErrNoRows {
		return ReturnNotFound(c, "Message not found")
	}
	if err != nil {
		return ReturnInternalError(c, "Failed to update message status")
	}

	response := SuccessResponse{
		Message: "Message status updated",
	}

	return c.JSON(response)
}

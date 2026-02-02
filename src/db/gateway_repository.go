package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type PollMessage struct {
	ID       string
	ToNumber string
	Body     string
}

func GetPendingMessagesForDevice(deviceID uint, topics []string) ([]PollMessage, error) {
	if len(topics) == 0 {
		return []PollMessage{}, nil
	}

	var messages []Message
	err := DB.Where("status = ?", "pending").
		Where("topic IN ?", topics).
		Where("assigned_device_id IS NULL OR assigned_device_id = ?", deviceID).
		Order("created_at ASC").
		Limit(10).
		Find(&messages).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query pending messages: %w", err)
	}

	pollMessages := make([]PollMessage, len(messages))
	for i, msg := range messages {
		pollMessages[i] = PollMessage{
			ID:       msg.ID,
			ToNumber: msg.ToNumber,
			Body:     msg.Body,
		}
	}

	if len(pollMessages) > 0 {
		if err := assignMessagesToDevice(deviceID, pollMessages); err != nil {
			return nil, fmt.Errorf("failed to assign messages to device: %w", err)
		}
	}

	return pollMessages, nil
}

func assignMessagesToDevice(deviceID uint, messages []PollMessage) error {
	if len(messages) == 0 {
		return nil
	}

	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	err := DB.Model(&Message{}).
		Where("id IN ?", messageIDs).
		Where("assigned_device_id IS NULL").
		Update("assigned_device_id", deviceID).Error

	if err != nil {
		return fmt.Errorf("failed to update message assignments: %w", err)
	}

	return nil
}

func UpdateMessageStatus(messageID string, status string, reason *string) error {
	if status != "sent" && status != "failed" {
		return fmt.Errorf("invalid status: must be 'sent' or 'failed'")
	}

	now := time.Now().UTC()
	updates := make(map[string]interface{})
	updates["status"] = status

	if status == "sent" {
		updates["sent_at"] = now
	} else {
		updates["failed_at"] = now
		updates["failure_reason"] = reason
	}

	result := DB.Model(&Message{}).Where("id = ?", messageID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update message status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func GetMessageByID(messageID string) (*Message, error) {
	var message Message
	err := DB.Where("id = ?", messageID).First(&message).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &message, nil
}

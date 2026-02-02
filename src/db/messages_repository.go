package db

import (
	"fmt"

	"github.com/google/uuid"
)

type MessageFilters struct {
	Topic    string
	ToNumber string
	Keyword  string
	Status   string
	Limit    int
	Offset   int
}

func CreateMessage(topic, toNumber, body string) (*Message, error) {
	id := fmt.Sprintf("msg_%s", uuid.New().String()[:8])

	message := &Message{
		ID:       id,
		Topic:    topic,
		ToNumber: toNumber,
		Body:     body,
		Status:   "pending",
	}

	if err := DB.Create(message).Error; err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return message, nil
}

func GetMessages(filters MessageFilters) ([]Message, error) {
	query := DB.Model(&Message{})

	if filters.Topic != "" {
		query = query.Where("topic = ?", filters.Topic)
	}

	if filters.ToNumber != "" {
		query = query.Where("to_number = ?", filters.ToNumber)
	}

	if filters.Keyword != "" {
		query = query.Where("LOWER(body) LIKE LOWER(?)", "%"+filters.Keyword+"%")
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	query = query.Order("created_at DESC")

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	var messages []Message
	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}

	return messages, nil
}

func CountMessages(filters MessageFilters) (int, error) {
	query := DB.Model(&Message{})

	if filters.Topic != "" {
		query = query.Where("topic = ?", filters.Topic)
	}

	if filters.ToNumber != "" {
		query = query.Where("to_number = ?", filters.ToNumber)
	}

	if filters.Keyword != "" {
		query = query.Where("LOWER(body) LIKE LOWER(?)", "%"+filters.Keyword+"%")
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return int(count), nil
}

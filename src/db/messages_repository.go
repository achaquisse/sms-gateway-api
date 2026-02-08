package db

import (
	"fmt"
	"os"
	"strconv"
	"time"

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

func getDeduplicationInterval() time.Duration {
	intervalStr := os.Getenv("DEDUPLICATION_INTERVAL_MINUTES")
	if intervalStr == "" {
		return 4320 * time.Minute
	}

	minutes, err := strconv.Atoi(intervalStr)
	if err != nil || minutes < 0 {
		return 4320 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}

func FindDuplicateMessage(toNumber, body string) (*Message, error) {
	interval := getDeduplicationInterval()
	cutoffTime := time.Now().Add(-interval)

	var message Message
	err := DB.Where("to_number = ? AND body = ? AND created_at > ?", toNumber, body, cutoffTime).
		Order("created_at DESC").
		First(&message).Error

	if err != nil {
		return nil, err
	}

	return &message, nil
}

func CreateMessage(topic, toNumber, body string) (*Message, error) {
	existingMsg, err := FindDuplicateMessage(toNumber, body)
	if err == nil && existingMsg != nil {
		return nil, fmt.Errorf("duplicate message: same message was sent to %s within the deduplication interval", toNumber)
	}

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

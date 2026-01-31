package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID            string
	Topic         string
	ToNumber      string
	Body          string
	Status        string
	CreatedAt     time.Time
	SentAt        *time.Time
	FailedAt      *time.Time
	FailureReason *string
}

type MessageFilters struct {
	Topic    string
	ToNumber string
	Keyword  string
	Status   string
	Limit    int
	Offset   int
}

type MessageStats struct {
	Total int
}

func CreateMessage(topic, toNumber, body string) (*Message, error) {
	id := fmt.Sprintf("msg_%s", uuid.New().String()[:8])

	query := `
		INSERT INTO messages (id, topic, to_number, body, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, topic, to_number, body, status, created_at, sent_at, failed_at, failure_reason
	`

	message := &Message{}
	err := DB.QueryRow(
		query,
		id,
		topic,
		toNumber,
		body,
		"pending",
		time.Now().UTC(),
	).Scan(
		&message.ID,
		&message.Topic,
		&message.ToNumber,
		&message.Body,
		&message.Status,
		&message.CreatedAt,
		&message.SentAt,
		&message.FailedAt,
		&message.FailureReason,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return message, nil
}

func GetMessages(filters MessageFilters) ([]Message, error) {
	query := "SELECT id, topic, to_number, body, status, created_at, sent_at, failed_at, failure_reason FROM messages WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filters.Topic != "" {
		query += fmt.Sprintf(" AND topic = $%d", argIndex)
		args = append(args, filters.Topic)
		argIndex++
	}

	if filters.ToNumber != "" {
		query += fmt.Sprintf(" AND to_number = $%d", argIndex)
		args = append(args, filters.ToNumber)
		argIndex++
	}

	if filters.Keyword != "" {
		query += fmt.Sprintf(" AND LOWER(body) LIKE LOWER($%d)", argIndex)
		args = append(args, "%"+filters.Keyword+"%")
		argIndex++
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var msg Message
		err := rows.Scan(
			&msg.ID,
			&msg.Topic,
			&msg.ToNumber,
			&msg.Body,
			&msg.Status,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.FailedAt,
			&msg.FailureReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

func CountMessages(filters MessageFilters) (int, error) {
	query := "SELECT COUNT(*) FROM messages WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filters.Topic != "" {
		query += fmt.Sprintf(" AND topic = $%d", argIndex)
		args = append(args, filters.Topic)
		argIndex++
	}

	if filters.ToNumber != "" {
		query += fmt.Sprintf(" AND to_number = $%d", argIndex)
		args = append(args, filters.ToNumber)
		argIndex++
	}

	if filters.Keyword != "" {
		query += fmt.Sprintf(" AND LOWER(body) LIKE LOWER($%d)", argIndex)
		args = append(args, "%"+filters.Keyword+"%")
		argIndex++
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
	}

	var count int
	err := DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

func buildWhereClause(filters MessageFilters) (string, []interface{}) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Topic != "" {
		conditions = append(conditions, fmt.Sprintf("topic = $%d", argIndex))
		args = append(args, filters.Topic)
		argIndex++
	}

	if filters.ToNumber != "" {
		conditions = append(conditions, fmt.Sprintf("to_number = $%d", argIndex))
		args = append(args, filters.ToNumber)
		argIndex++
	}

	if filters.Keyword != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(body) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+filters.Keyword+"%")
		argIndex++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	return where, args
}

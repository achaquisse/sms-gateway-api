package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PollMessage struct {
	ID       string
	ToNumber string
	Body     string
}

func GetPendingMessagesForDevice(deviceID int, topics []string) ([]PollMessage, error) {
	if len(topics) == 0 {
		return []PollMessage{}, nil
	}

	var query string
	var args []interface{}

	if IsSQLite() {
		placeholders := make([]string, len(topics))
		args = make([]interface{}, 0, len(topics)+1)
		for i, topic := range topics {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, topic)
		}
		deviceIDParam := len(topics) + 1
		args = append(args, deviceID)

		query = fmt.Sprintf(`
			SELECT id, to_number, body
			FROM messages
			WHERE status = 'pending'
			AND topic IN (%s)
			AND (assigned_device_id IS NULL OR assigned_device_id = $%d)
			ORDER BY created_at ASC
			LIMIT 10
		`, strings.Join(placeholders, ", "), deviceIDParam)
	} else {
		query = `
			SELECT id, to_number, body
			FROM messages
			WHERE status = 'pending'
			AND topic = ANY($1)
			AND (assigned_device_id IS NULL OR assigned_device_id = $2)
			ORDER BY created_at ASC
			LIMIT 10
		`
		args = []interface{}{topics, deviceID}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending messages: %w", err)
	}
	defer rows.Close()

	messages := []PollMessage{}
	for rows.Next() {
		var msg PollMessage
		err := rows.Scan(
			&msg.ID,
			&msg.ToNumber,
			&msg.Body,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	if len(messages) > 0 {
		if err := assignMessagesToDevice(deviceID, messages); err != nil {
			return nil, fmt.Errorf("failed to assign messages to device: %w", err)
		}
	}

	return messages, nil
}

func assignMessagesToDevice(deviceID int, messages []PollMessage) error {
	if len(messages) == 0 {
		return nil
	}

	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	var query string
	var args []interface{}

	if IsSQLite() {
		placeholders := make([]string, len(messageIDs))
		args = make([]interface{}, 0, len(messageIDs)+1)
		args = append(args, deviceID)
		for i, msgID := range messageIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, msgID)
		}

		query = fmt.Sprintf(`
			UPDATE messages
			SET assigned_device_id = $1
			WHERE id IN (%s)
			AND assigned_device_id IS NULL
		`, strings.Join(placeholders, ", "))
	} else {
		query = `
			UPDATE messages
			SET assigned_device_id = $1
			WHERE id = ANY($2)
			AND assigned_device_id IS NULL
		`
		args = []interface{}{deviceID, messageIDs}
	}

	_, err := DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update message assignments: %w", err)
	}

	return nil
}

func UpdateMessageStatus(messageID string, status string, reason *string) error {
	if status != "sent" && status != "failed" {
		return fmt.Errorf("invalid status: must be 'sent' or 'failed'")
	}

	var query string
	var args []interface{}

	if status == "sent" {
		query = `
			UPDATE messages
			SET status = $1, sent_at = $2
			WHERE id = $3
		`
		args = []interface{}{status, time.Now().UTC(), messageID}
	} else {
		query = `
			UPDATE messages
			SET status = $1, failed_at = $2, failure_reason = $3
			WHERE id = $4
		`
		args = []interface{}{status, time.Now().UTC(), reason, messageID}
	}

	result, err := DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func GetMessageByID(messageID string) (*Message, error) {
	query := `
		SELECT id, topic, to_number, body, status, created_at, sent_at, failed_at, failure_reason
		FROM messages
		WHERE id = $1
	`

	message := &Message{}
	err := DB.QueryRow(query, messageID).Scan(
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

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}

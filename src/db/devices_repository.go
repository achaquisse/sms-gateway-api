package db

import (
	"database/sql"
	"fmt"
	"time"
)

type Device struct {
	ID         int
	DeviceKey  string
	Name       *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastPollAt *time.Time
}

func GetDeviceByKey(deviceKey string) (*Device, error) {
	query := `
		SELECT id, device_key, name, created_at, updated_at, last_poll_at
		FROM devices
		WHERE device_key = $1
	`

	device := &Device{}
	err := DB.QueryRow(query, deviceKey).Scan(
		&device.ID,
		&device.DeviceKey,
		&device.Name,
		&device.CreatedAt,
		&device.UpdatedAt,
		&device.LastPollAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return device, nil
}

func CreateDevice(deviceKey string, name *string) (*Device, error) {
	now := time.Now().UTC()

	query := `
		INSERT INTO devices (id, device_key, name, created_at, updated_at)
		VALUES (NULL, $1, $2, $3, $4)
	`

	result, err := DB.Exec(query, deviceKey, name, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	device := &Device{
		ID:         int(id),
		DeviceKey:  deviceKey,
		Name:       name,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastPollAt: nil,
	}

	return device, nil
}

func UpdateDeviceLastPoll(deviceID int) error {
	query := `
		UPDATE devices
		SET last_poll_at = $1
		WHERE id = $2
	`

	_, err := DB.Exec(query, time.Now().UTC(), deviceID)
	if err != nil {
		return fmt.Errorf("failed to update device last poll: %w", err)
	}

	return nil
}

func GetDeviceTopics(deviceID int) ([]string, error) {
	query := `
		SELECT topic
		FROM device_topics
		WHERE device_id = $1
		ORDER BY topic
	`

	rows, err := DB.Query(query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query device topics: %w", err)
	}
	defer rows.Close()

	topics := []string{}
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, topic)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating topics: %w", err)
	}

	return topics, nil
}

func SetDeviceTopics(deviceID int, topics []string) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	deleteQuery := `DELETE FROM device_topics WHERE device_id = $1`
	if _, err := tx.Exec(deleteQuery, deviceID); err != nil {
		return fmt.Errorf("failed to delete existing topics: %w", err)
	}

	if len(topics) > 0 {
		insertQuery := `
			INSERT INTO device_topics (device_id, topic, created_at)
			VALUES ($1, $2, $3)
		`

		for _, topic := range topics {
			_, err := tx.Exec(insertQuery, deviceID, topic, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("failed to insert topic: %w", err)
			}
		}
	}

	updateQuery := `UPDATE devices SET updated_at = $1 WHERE id = $2`
	if _, err := tx.Exec(updateQuery, time.Now().UTC(), deviceID); err != nil {
		return fmt.Errorf("failed to update device timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

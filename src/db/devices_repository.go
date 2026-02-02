package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

func GetDeviceByKey(deviceKey string) (*Device, error) {
	var device Device
	err := DB.Where("device_key = ?", deviceKey).First(&device).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	return &device, nil
}

func CreateDevice(deviceKey string, name *string) (*Device, error) {
	device := &Device{
		DeviceKey: deviceKey,
		Name:      name,
	}

	if err := DB.Create(device).Error; err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}

func UpdateDeviceLastPoll(deviceID uint) error {
	now := time.Now().UTC()
	err := DB.Model(&Device{}).Where("id = ?", deviceID).Update("last_poll_at", now).Error
	if err != nil {
		return fmt.Errorf("failed to update device last poll: %w", err)
	}
	return nil
}

func GetDeviceTopics(deviceID uint) ([]string, error) {
	var deviceTopics []DeviceTopic
	err := DB.Where("device_id = ?", deviceID).Order("topic").Find(&deviceTopics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query device topics: %w", err)
	}

	topics := make([]string, len(deviceTopics))
	for i, dt := range deviceTopics {
		topics[i] = dt.Topic
	}

	return topics, nil
}

func SetDeviceTopics(deviceID uint, topics []string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("device_id = ?", deviceID).Delete(&DeviceTopic{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing topics: %w", err)
		}

		for _, topic := range topics {
			deviceTopic := DeviceTopic{
				DeviceID: deviceID,
				Topic:    topic,
			}
			if err := tx.Create(&deviceTopic).Error; err != nil {
				return fmt.Errorf("failed to insert topic: %w", err)
			}
		}

		if err := tx.Model(&Device{}).Where("id = ?", deviceID).Update("updated_at", time.Now().UTC()).Error; err != nil {
			return fmt.Errorf("failed to update device timestamp: %w", err)
		}

		return nil
	})
}

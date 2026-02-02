package db

import (
	"time"
)

type Device struct {
	ID         uint          `gorm:"primaryKey;autoIncrement"`
	DeviceKey  string        `gorm:"uniqueIndex;size:255;not null"`
	Name       *string       `gorm:"size:255"`
	CreatedAt  time.Time     `gorm:"not null;autoCreateTime"`
	UpdatedAt  time.Time     `gorm:"not null;autoUpdateTime"`
	LastPollAt *time.Time    `gorm:"index"`
	Topics     []DeviceTopic `gorm:"foreignKey:DeviceID;constraint:OnDelete:CASCADE"`
}

type Message struct {
	ID               string    `gorm:"primaryKey;size:255"`
	Topic            string    `gorm:"index:idx_topic_status;size:255;not null"`
	ToNumber         string    `gorm:"index;size:20;not null"`
	Body             string    `gorm:"type:text;not null"`
	Status           string    `gorm:"index:idx_topic_status;size:20;not null;default:pending;check:status IN ('pending','sent','failed')"`
	CreatedAt        time.Time `gorm:"index;not null;autoCreateTime"`
	SentAt           *time.Time
	FailedAt         *time.Time
	FailureReason    *string `gorm:"type:text"`
	AssignedDeviceID *uint   `gorm:"index"`
	AssignedDevice   *Device `gorm:"foreignKey:AssignedDeviceID;constraint:OnDelete:SET NULL"`
}

type DeviceTopic struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	DeviceID  uint      `gorm:"uniqueIndex:idx_device_topic;index;not null"`
	Topic     string    `gorm:"uniqueIndex:idx_device_topic;index;size:255;not null"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	Device    Device    `gorm:"foreignKey:DeviceID;constraint:OnDelete:CASCADE"`
}

type SchemaMigration struct {
	Version   int       `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null;autoCreateTime"`
}

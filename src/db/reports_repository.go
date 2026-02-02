package db

import (
	"fmt"
	"time"
)

type ReportSummary struct {
	Total   int64
	Sent    int64
	Failed  int64
	Pending int64
}

type TopicStats struct {
	Topic   string
	Total   int64
	Sent    int64
	Failed  int64
	Pending int64
}

type TimelineEntry struct {
	Date    string
	Total   int64
	Sent    int64
	Failed  int64
	Pending int64
}

func GetReportSummary(startDate, endDate time.Time, topic string) (*ReportSummary, error) {
	query := DB.Model(&Message{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	if topic != "" {
		query = query.Where("topic = ?", topic)
	}

	var summary ReportSummary
	err := query.Select(`
		COUNT(*) as total,
		SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
	`).Scan(&summary).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get report summary: %w", err)
	}

	return &summary, nil
}

func GetTopicStats(startDate, endDate time.Time, topicFilter string) ([]TopicStats, error) {
	query := DB.Model(&Message{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	if topicFilter != "" {
		query = query.Where("topic = ?", topicFilter)
	}

	var stats []TopicStats
	err := query.Select(`
		topic,
		COUNT(*) as total,
		SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
	`).Group("topic").Order("topic").Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query topic stats: %w", err)
	}

	return stats, nil
}

func GetTimelineStats(startDate, endDate time.Time, aggregation string, topic string) ([]TimelineEntry, error) {
	var dateFormat string

	if IsSQLite() {
		switch aggregation {
		case "daily":
			dateFormat = "strftime('%Y-%m-%d', created_at)"
		case "weekly":
			dateFormat = "strftime('%Y-%W', created_at)"
		case "monthly":
			dateFormat = "strftime('%Y-%m', created_at)"
		default:
			dateFormat = "strftime('%Y-%m-%d', created_at)"
		}
	} else {
		switch aggregation {
		case "daily":
			dateFormat = "DATE_FORMAT(created_at, '%Y-%m-%d')"
		case "weekly":
			dateFormat = "DATE_FORMAT(created_at, '%Y-%u')"
		case "monthly":
			dateFormat = "DATE_FORMAT(created_at, '%Y-%m')"
		default:
			dateFormat = "DATE_FORMAT(created_at, '%Y-%m-%d')"
		}
	}

	query := DB.Model(&Message{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	if topic != "" {
		query = query.Where("topic = ?", topic)
	}

	var timeline []TimelineEntry
	selectQuery := fmt.Sprintf(`
		%s as date,
		COUNT(*) as total,
		SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
	`, dateFormat)

	err := query.Select(selectQuery).
		Group(dateFormat).
		Order(dateFormat).
		Scan(&timeline).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query timeline stats: %w", err)
	}

	return timeline, nil
}

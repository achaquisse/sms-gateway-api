package db

import (
	"fmt"
	"time"
)

type ReportSummary struct {
	Total   int
	Sent    int
	Failed  int
	Pending int
}

type TopicStats struct {
	Topic   string
	Total   int
	Sent    int
	Failed  int
	Pending int
}

type TimelineEntry struct {
	Date    string
	Total   int
	Sent    int
	Failed  int
	Pending int
}

func GetReportSummary(startDate, endDate time.Time, topic string) (*ReportSummary, error) {
	var query string
	
	if IsSQLite() {
		query = `
			SELECT 
				COUNT(*) as total,
				SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
				SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
		`
	} else {
		query = `
			SELECT 
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'sent') as sent,
				COUNT(*) FILTER (WHERE status = 'failed') as failed,
				COUNT(*) FILTER (WHERE status = 'pending') as pending
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
		`
	}
	args := []interface{}{startDate, endDate}

	if topic != "" {
		query += " AND topic = $3"
		args = append(args, topic)
	}

	summary := &ReportSummary{}
	err := DB.QueryRow(query, args...).Scan(
		&summary.Total,
		&summary.Sent,
		&summary.Failed,
		&summary.Pending,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get report summary: %w", err)
	}

	return summary, nil
}

func GetTopicStats(startDate, endDate time.Time, topicFilter string) ([]TopicStats, error) {
	var query string
	
	if IsSQLite() {
		query = `
			SELECT 
				topic,
				COUNT(*) as total,
				SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
				SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
		`
	} else {
		query = `
			SELECT 
				topic,
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'sent') as sent,
				COUNT(*) FILTER (WHERE status = 'failed') as failed,
				COUNT(*) FILTER (WHERE status = 'pending') as pending
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
		`
	}
	args := []interface{}{startDate, endDate}

	if topicFilter != "" {
		query += " AND topic = $3"
		args = append(args, topicFilter)
	}

	query += " GROUP BY topic ORDER BY topic"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query topic stats: %w", err)
	}
	defer rows.Close()

	stats := []TopicStats{}
	for rows.Next() {
		var s TopicStats
		err := rows.Scan(
			&s.Topic,
			&s.Total,
			&s.Sent,
			&s.Failed,
			&s.Pending,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan topic stats: %w", err)
		}
		stats = append(stats, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating topic stats: %w", err)
	}

	return stats, nil
}

func GetTimelineStats(startDate, endDate time.Time, aggregation string, topic string) ([]TimelineEntry, error) {
	var query string
	args := []interface{}{startDate, endDate}

	if IsSQLite() {
		var dateFormat string
		switch aggregation {
		case "daily":
			dateFormat = "%Y-%m-%d"
		case "weekly":
			dateFormat = "%Y-%W"
		case "monthly":
			dateFormat = "%Y-%m"
		default:
			dateFormat = "%Y-%m-%d"
		}

		query = fmt.Sprintf(`
			SELECT 
				strftime('%s', substr(created_at, 1, 19)) as date,
				COUNT(*) as total,
				SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
				SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
		`, dateFormat)

		if topic != "" {
			query += " AND topic = $3"
			args = append(args, topic)
		}

		query += fmt.Sprintf(" GROUP BY strftime('%s', substr(created_at, 1, 19)) ORDER BY strftime('%s', substr(created_at, 1, 19))", dateFormat, dateFormat)
	} else {
		var dateFormat string
		var dateTrunc string

		switch aggregation {
		case "daily":
			dateFormat = "YYYY-MM-DD"
			dateTrunc = "day"
		case "weekly":
			dateFormat = "IYYY-IW"
			dateTrunc = "week"
		case "monthly":
			dateFormat = "YYYY-MM"
			dateTrunc = "month"
		default:
			dateFormat = "YYYY-MM-DD"
			dateTrunc = "day"
		}

		query = fmt.Sprintf(`
			SELECT 
				TO_CHAR(DATE_TRUNC('%s', created_at), '%s') as date,
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'sent') as sent,
				COUNT(*) FILTER (WHERE status = 'failed') as failed,
				COUNT(*) FILTER (WHERE status = 'pending') as pending
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
		`, dateTrunc, dateFormat)

		if topic != "" {
			query += " AND topic = $3"
			args = append(args, topic)
		}

		query += fmt.Sprintf(" GROUP BY DATE_TRUNC('%s', created_at) ORDER BY DATE_TRUNC('%s', created_at)", dateTrunc, dateTrunc)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline stats: %w", err)
	}
	defer rows.Close()

	timeline := []TimelineEntry{}
	for rows.Next() {
		var entry TimelineEntry
		err := rows.Scan(
			&entry.Date,
			&entry.Total,
			&entry.Sent,
			&entry.Failed,
			&entry.Pending,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline entry: %w", err)
		}
		timeline = append(timeline, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timeline: %w", err)
	}

	return timeline, nil
}

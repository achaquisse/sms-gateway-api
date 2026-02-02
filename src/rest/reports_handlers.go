package rest

import (
	"sms-gateway-api/db"
	"time"

	"github.com/gofiber/fiber/v2"
)

func GetReportsHandler(c *fiber.Ctx) error {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" {
		return ReturnBadRequest(c, "start_date is required")
	}

	if endDateStr == "" {
		return ReturnBadRequest(c, "end_date is required")
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		return ReturnBadRequest(c, "Invalid start_date format. Use ISO 8601 format (e.g., 2026-01-01T00:00:00Z)")
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		return ReturnBadRequest(c, "Invalid end_date format. Use ISO 8601 format (e.g., 2026-01-31T23:59:59Z)")
	}

	aggregation := c.Query("aggregation", "daily")
	if aggregation != "daily" && aggregation != "weekly" && aggregation != "monthly" {
		return ReturnBadRequest(c, "Invalid aggregation. Must be one of: daily, weekly, monthly")
	}

	topic := c.Query("topic")

	summary, err := db.GetReportSummary(startDate, endDate, topic)
	if err != nil {
		return ReturnInternalError(c, "Failed to retrieve report summary")
	}

	topicStats, err := db.GetTopicStats(startDate, endDate, topic)
	if err != nil {
		return ReturnInternalError(c, "Failed to retrieve topic statistics")
	}

	timeline, err := db.GetTimelineStats(startDate, endDate, aggregation, topic)
	if err != nil {
		return ReturnInternalError(c, "Failed to retrieve timeline statistics")
	}

	restTopicStats := make([]TopicStats, len(topicStats))
	for i, ts := range topicStats {
		restTopicStats[i] = TopicStats{
			Topic:   ts.Topic,
			Total:   int(ts.Total),
			Sent:    int(ts.Sent),
			Failed:  int(ts.Failed),
			Pending: int(ts.Pending),
		}
	}

	restTimeline := make([]TimelineEntry, len(timeline))
	for i, te := range timeline {
		restTimeline[i] = TimelineEntry{
			Date:    te.Date,
			Total:   int(te.Total),
			Sent:    int(te.Sent),
			Failed:  int(te.Failed),
			Pending: int(te.Pending),
		}
	}

	response := ReportResponse{
		Period: ReportPeriod{
			Start:       startDate,
			End:         endDate,
			Aggregation: aggregation,
		},
		Summary: ReportSummary{
			Total:   int(summary.Total),
			Sent:    int(summary.Sent),
			Failed:  int(summary.Failed),
			Pending: int(summary.Pending),
		},
		ByTopic:  restTopicStats,
		Timeline: restTimeline,
	}

	return c.JSON(response)
}

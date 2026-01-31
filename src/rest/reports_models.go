package rest

import "time"

type ReportPeriod struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Aggregation string    `json:"aggregation"`
}

type ReportSummary struct {
	Total   int `json:"total"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
}

type TopicStats struct {
	Topic   string `json:"topic"`
	Total   int    `json:"total"`
	Sent    int    `json:"sent"`
	Failed  int    `json:"failed"`
	Pending int    `json:"pending"`
}

type TimelineEntry struct {
	Date    string `json:"date"`
	Total   int    `json:"total"`
	Sent    int    `json:"sent"`
	Failed  int    `json:"failed"`
	Pending int    `json:"pending"`
}

type ReportResponse struct {
	Period   ReportPeriod    `json:"period"`
	Summary  ReportSummary   `json:"summary"`
	ByTopic  []TopicStats    `json:"by_topic"`
	Timeline []TimelineEntry `json:"timeline"`
}

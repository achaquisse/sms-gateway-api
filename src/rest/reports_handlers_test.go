package rest

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"sms-gateway-api/db"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupReportsTestApp() *fiber.App {
	app := fiber.New()
	app.Get("/reports", GetReportsHandler)
	return app
}

func TestGetReportsHandler(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	app := setupReportsTestApp()

	msg1, err := db.CreateMessage("otp", "+1234567890", "Your OTP is 123456")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}
	msg2, err := db.CreateMessage("alerts", "+9876543210", "Alert: Login detected")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}
	if err := db.UpdateMessageStatus(msg1.ID, "sent", nil); err != nil {
		t.Fatalf("Failed to update message status: %v", err)
	}
	if err := db.UpdateMessageStatus(msg2.ID, "failed", strPtr("Network error")); err != nil {
		t.Fatalf("Failed to update message status: %v", err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Missing start_date",
			queryParams:    "?end_date=2026-01-31T23:59:59Z",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Missing end_date",
			queryParams:    "?start_date=2026-01-01T00:00:00Z",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Invalid start_date format",
			queryParams:    "?start_date=invalid&end_date=2026-01-31T23:59:59Z",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Invalid aggregation",
			queryParams:    "?start_date=2026-01-01T00:00:00Z&end_date=2026-01-31T23:59:59Z&aggregation=invalid",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Valid request with simple date format",
			queryParams:    "?start_date=2020-01-01&end_date=2030-12-31&aggregation=monthly",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response ReportResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Period.Aggregation != "monthly" {
					t.Errorf("Expected aggregation 'monthly', got '%s'", response.Period.Aggregation)
				}
				if response.Summary.Total != 2 {
					t.Errorf("Expected total 2, got %d", response.Summary.Total)
				}
			},
		},
		{
			name:           "Valid request with daily aggregation",
			queryParams:    "?start_date=2020-01-01T00:00:00Z&end_date=2030-12-31T23:59:59Z&aggregation=daily",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response ReportResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Period.Aggregation != "daily" {
					t.Errorf("Expected aggregation 'daily', got '%s'", response.Period.Aggregation)
				}
				if response.Summary.Total != 2 {
					t.Errorf("Expected total 2, got %d", response.Summary.Total)
				}
				if response.Summary.Sent != 1 {
					t.Errorf("Expected sent 1, got %d", response.Summary.Sent)
				}
				if response.Summary.Failed != 1 {
					t.Errorf("Expected failed 1, got %d", response.Summary.Failed)
				}
				if len(response.ByTopic) != 2 {
					t.Errorf("Expected 2 topics, got %d", len(response.ByTopic))
				}
			},
		},
		{
			name:           "Valid request with topic filter",
			queryParams:    "?start_date=2020-01-01T00:00:00Z&end_date=2030-12-31T23:59:59Z&topic=otp",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response ReportResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Summary.Total != 1 {
					t.Errorf("Expected total 1, got %d", response.Summary.Total)
				}
				if len(response.ByTopic) != 1 {
					t.Errorf("Expected 1 topic, got %d", len(response.ByTopic))
				}
				if len(response.ByTopic) > 0 && response.ByTopic[0].Topic != "otp" {
					t.Errorf("Expected topic 'otp', got '%s'", response.ByTopic[0].Topic)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/reports"+tt.queryParams, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}

			if tt.checkResponse != nil {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				tt.checkResponse(t, body)
			}
		})
	}
}

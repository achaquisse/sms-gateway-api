package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sms-gateway-api/db"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupGatewayTestApp() *fiber.App {
	app := fiber.New()
	app.Get("/gateway/poll", PollMessagesHandler)
	app.Put("/gateway/status/:messageId", UpdateMessageStatusHandler)
	return app
}

func setupGatewayTestDB(t *testing.T) {
	config := db.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	if err := db.ConnectWithConfig(config); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	schemaPath := filepath.Join("..", "..", "db-schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	schema := string(schemaBytes)
	schema = strings.ReplaceAll(schema, "SERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT")

	if _, err := db.GetDB().Exec(schema); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
}

func TestPollMessagesHandler(t *testing.T) {
	setupGatewayTestDB(t)
	defer teardownTestDB()

	app := setupGatewayTestApp()

	device, _ := db.CreateDevice("test_device_key", nil)
	db.SetDeviceTopics(device.ID, []string{"otp", "alerts"})
	db.CreateMessage("otp", "+1234567890", "Your OTP is 123456")
	db.CreateMessage("alerts", "+9876543210", "Alert: Login detected")
	db.CreateMessage("notifications", "+1111111111", "Notification")

	tests := []struct {
		name           string
		deviceKey      string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Missing device key",
			deviceKey:      "",
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:           "Invalid device key",
			deviceKey:      "invalid_key",
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:           "Valid request",
			deviceKey:      "test_device_key",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response PollResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Messages) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(response.Messages))
				}
				for _, msg := range response.Messages {
					if msg.ID == "" || msg.ToNumber == "" || msg.Body == "" {
						t.Error("Expected all message fields to be populated")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/gateway/poll", nil)
			if tt.deviceKey != "" {
				req.Header.Set("X-Device-Key", tt.deviceKey)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}

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

func TestUpdateMessageStatusHandler(t *testing.T) {
	setupGatewayTestDB(t)
	defer teardownTestDB()

	app := setupGatewayTestApp()

	_, _ = db.CreateDevice("test_device_key_status", nil)
	msg, _ := db.CreateMessage("otp", "+1234567890", "Your OTP is 123456")

	tests := []struct {
		name           string
		deviceKey      string
		messageID      string
		payload        interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Missing device key",
			deviceKey:      "",
			messageID:      msg.ID,
			payload:        StatusUpdateRequest{Status: "sent"},
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:           "Invalid device key",
			deviceKey:      "invalid_key",
			messageID:      msg.ID,
			payload:        StatusUpdateRequest{Status: "sent"},
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:           "Message not found",
			deviceKey:      "test_device_key_status",
			messageID:      "nonexistent_msg",
			payload:        StatusUpdateRequest{Status: "sent"},
			expectedStatus: fiber.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name:      "Valid request - mark as sent",
			deviceKey: "test_device_key_status",
			messageID: msg.ID,
			payload: StatusUpdateRequest{
				Status: "sent",
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response SuccessResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Message != "Message status updated" {
					t.Errorf("Expected message 'Message status updated', got '%s'", response.Message)
				}

				message, err := db.GetMessageByID(msg.ID)
				if err != nil {
					t.Fatalf("Failed to get message: %v", err)
				}
				if message.Status != "sent" {
					t.Errorf("Expected status 'sent', got '%s'", message.Status)
				}
				if message.SentAt == nil {
					t.Error("Expected sent_at to be set")
				}
			},
		},
		{
			name:      "Valid request - mark as failed",
			deviceKey: "test_device_key_status",
			messageID: msg.ID,
			payload: StatusUpdateRequest{
				Status: "failed",
				Reason: strPtr("Network error"),
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				message, err := db.GetMessageByID(msg.ID)
				if err != nil {
					t.Fatalf("Failed to get message: %v", err)
				}
				if message.Status != "failed" {
					t.Errorf("Expected status 'failed', got '%s'", message.Status)
				}
				if message.FailedAt == nil {
					t.Error("Expected failed_at to be set")
				}
				if message.FailureReason == nil || *message.FailureReason != "Network error" {
					t.Error("Expected failure_reason to be 'Network error'")
				}
			},
		},
		{
			name:           "Invalid status",
			deviceKey:      "test_device_key_status",
			messageID:      msg.ID,
			payload:        StatusUpdateRequest{Status: "invalid"},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Missing status",
			deviceKey:      "test_device_key_status",
			messageID:      msg.ID,
			payload:        map[string]interface{}{},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Invalid JSON",
			deviceKey:      "test_device_key_status",
			messageID:      msg.ID,
			payload:        "invalid json",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error

			if str, ok := tt.payload.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
			}

			req := httptest.NewRequest("PUT", "/gateway/status/"+tt.messageID, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tt.deviceKey != "" {
				req.Header.Set("X-Device-Key", tt.deviceKey)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}

			if tt.checkResponse != nil {
				resp.Body.Close()
				req := httptest.NewRequest("PUT", "/gateway/status/"+tt.messageID, bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				if tt.deviceKey != "" {
					req.Header.Set("X-Device-Key", tt.deviceKey)
				}
				resp, _ := app.Test(req)
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				tt.checkResponse(t, body)
			}
		})
	}
}

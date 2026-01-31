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

func setupDevicesTestApp() *fiber.App {
	app := fiber.New()
	app.Put("/devices", UpdateDeviceTopicsHandler)
	return app
}

func setupDevicesTestDB(t *testing.T) {
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

func TestUpdateDeviceTopicsHandler(t *testing.T) {
	setupDevicesTestDB(t)
	defer teardownTestDB()

	app := setupDevicesTestApp()

	tests := []struct {
		name           string
		deviceKey      string
		payload        interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Missing device key",
			deviceKey:      "",
			payload:        DeviceConfigRequest{Topics: []string{"otp", "alerts"}},
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:      "Valid request - new device",
			deviceKey: "device_test_key_1",
			payload: DeviceConfigRequest{
				Topics: []string{"otp", "alerts"},
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response SuccessResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Message != "Device configuration updated" {
					t.Errorf("Expected message 'Device configuration updated', got '%s'", response.Message)
				}

				device, err := db.GetDeviceByKey("device_test_key_1")
				if err != nil {
					t.Fatalf("Failed to get device: %v", err)
				}
				if device == nil {
					t.Fatal("Expected device to be created")
				}

				topics, err := db.GetDeviceTopics(device.ID)
				if err != nil {
					t.Fatalf("Failed to get device topics: %v", err)
				}
				if len(topics) != 2 {
					t.Errorf("Expected 2 topics, got %d", len(topics))
				}
			},
		},
		{
			name:      "Valid request - update existing device",
			deviceKey: "device_test_key_2",
			payload: DeviceConfigRequest{
				Topics: []string{"notifications"},
			},
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				device, _ := db.GetDeviceByKey("device_test_key_2")
				topics, err := db.GetDeviceTopics(device.ID)
				if err != nil {
					t.Fatalf("Failed to get device topics: %v", err)
				}
				if len(topics) != 1 {
					t.Errorf("Expected 1 topic, got %d", len(topics))
				}
				if topics[0] != "notifications" {
					t.Errorf("Expected topic 'notifications', got '%s'", topics[0])
				}
			},
		},
		{
			name:           "Invalid request - missing topics",
			deviceKey:      "device_test_key_3",
			payload:        map[string]interface{}{},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Invalid JSON",
			deviceKey:      "device_test_key_4",
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

			req := httptest.NewRequest("PUT", "/devices", bytes.NewReader(bodyBytes))
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
				req := httptest.NewRequest("PUT", "/devices", bytes.NewReader(bodyBytes))
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

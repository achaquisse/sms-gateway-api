package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"sms-gateway-api/db"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupDevicesTestApp() *fiber.App {
	app := fiber.New()
	app.Get("/devices", GetDeviceTopicsHandler)
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

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
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
				device, err := db.GetDeviceByKey("device_test_key_2")
				if err != nil {
					t.Fatalf("Failed to get device: %v", err)
				}
				if device == nil {
					t.Fatal("Expected device to exist")
				}

				topics, err := db.GetDeviceTopics(device.ID)
				if err != nil {
					t.Fatalf("Failed to get device topics: %v", err)
				}
				if len(topics) != 1 {
					t.Errorf("Expected 1 topic, got %d", len(topics))
				}
				if len(topics) > 0 && topics[0] != "notifications" {
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
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestGetDeviceTopicsHandler(t *testing.T) {
	setupDevicesTestDB(t)
	defer teardownTestDB()

	app := setupDevicesTestApp()

	device1, err := db.CreateDevice("device_get_test_1", nil)
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}
	if err := db.SetDeviceTopics(device1.ID, []string{"otp", "alerts", "notifications"}); err != nil {
		t.Fatalf("Failed to set device topics: %v", err)
	}

	_, err = db.CreateDevice("device_get_test_2", nil)
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}

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
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if _, ok := response["error"]; !ok {
					t.Error("Expected error field in response")
				}
			},
		},
		{
			name:           "Invalid device key",
			deviceKey:      "nonexistent_device_key",
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if _, ok := response["error"]; !ok {
					t.Error("Expected error field in response")
				}
			},
		},
		{
			name:           "Valid request - device with topics",
			deviceKey:      "device_get_test_1",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response DeviceConfigRequest
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Topics) != 3 {
					t.Errorf("Expected 3 topics, got %d", len(response.Topics))
				}
				expectedTopics := map[string]bool{"otp": true, "alerts": true, "notifications": true}
				for _, topic := range response.Topics {
					if !expectedTopics[topic] {
						t.Errorf("Unexpected topic: %s", topic)
					}
				}
			},
		},
		{
			name:           "Valid request - device without topics",
			deviceKey:      "device_get_test_2",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response DeviceConfigRequest
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Topics) != 0 {
					t.Errorf("Expected 0 topics, got %d", len(response.Topics))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/devices", nil)
			if tt.deviceKey != "" {
				req.Header.Set("X-Device-Key", tt.deviceKey)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

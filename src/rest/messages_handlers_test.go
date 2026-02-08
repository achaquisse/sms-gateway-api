package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"sms-gateway-api/db"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	app := fiber.New()
	app.Post("/messages", QueueSMSHandler)
	app.Get("/messages", ListMessagesHandler)
	return app
}

func setupTestDB(t *testing.T) {
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

func teardownTestDB() {
	db.Close()
}

func TestQueueSMSHandler(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	app := setupTestApp()

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "Valid request",
			payload: QueueSMSRequest{
				Topic:    "otp",
				ToNumber: "+1234567890",
				Body:     "Your OTP code is 123456",
			},
			expectedStatus: fiber.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var response QueueSMSResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if response.Message != "Message queued successfully" {
					t.Errorf("Expected message 'Message queued successfully', got '%s'", response.Message)
				}
				if response.ID == "" {
					t.Error("Expected non-empty message ID")
				}
			},
		},
		{
			name: "Missing topic",
			payload: QueueSMSRequest{
				ToNumber: "+1234567890",
				Body:     "Your OTP code is 123456",
			},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "Missing to_number",
			payload: QueueSMSRequest{
				Topic: "otp",
				Body:  "Your OTP code is 123456",
			},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "Missing body",
			payload: QueueSMSRequest{
				Topic:    "otp",
				ToNumber: "+1234567890",
			},
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Invalid JSON",
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

			req := httptest.NewRequest("POST", "/messages", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

func TestListMessagesHandler(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	app := setupTestApp()

	_, err := db.CreateMessage("otp", "+1234567890", "Your OTP is 123456")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	_, err = db.CreateMessage("alerts", "+9876543210", "Alert: Login detected")
	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "List all messages",
			queryParams:    "",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response MessagesListResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Data) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(response.Data))
				}
				if response.Pagination.Total != 2 {
					t.Errorf("Expected total 2, got %d", response.Pagination.Total)
				}
			},
		},
		{
			name:           "Filter by topic",
			queryParams:    "?topic=otp",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response MessagesListResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Data) != 1 {
					t.Errorf("Expected 1 message, got %d", len(response.Data))
				}
				if len(response.Data) > 0 && response.Data[0].Topic != "otp" {
					t.Errorf("Expected topic 'otp', got '%s'", response.Data[0].Topic)
				}
			},
		},
		{
			name:           "Filter by to_number",
			queryParams:    "?to_number=%2B1234567890",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response MessagesListResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Data) != 1 {
					t.Errorf("Expected 1 message, got %d", len(response.Data))
				}
			},
		},
		{
			name:           "Filter by keyword",
			queryParams:    "?keyword=OTP",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response MessagesListResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Data) != 1 {
					t.Errorf("Expected 1 message, got %d", len(response.Data))
				}
			},
		},
		{
			name:           "Filter by status",
			queryParams:    "?status=pending",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response MessagesListResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Data) != 2 {
					t.Errorf("Expected 2 pending messages, got %d", len(response.Data))
				}
			},
		},
		{
			name:           "Invalid status",
			queryParams:    "?status=invalid",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Pagination - page 1",
			queryParams:    "?page=1&limit=1",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response MessagesListResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(response.Data) != 1 {
					t.Errorf("Expected 1 message per page, got %d", len(response.Data))
				}
				if response.Pagination.Page != 1 {
					t.Errorf("Expected page 1, got %d", response.Pagination.Page)
				}
				if response.Pagination.TotalPages != 2 {
					t.Errorf("Expected 2 total pages, got %d", response.Pagination.TotalPages)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/messages"+tt.queryParams, nil)
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

func TestQueueSMSHandler_Deduplication(t *testing.T) {
	os.Setenv("DEDUPLICATION_INTERVAL_MINUTES", "1")
	defer os.Unsetenv("DEDUPLICATION_INTERVAL_MINUTES")

	setupTestDB(t)
	defer teardownTestDB()

	app := setupTestApp()

	payload := QueueSMSRequest{
		Topic:    "otp",
		ToNumber: "+1234567890",
		Body:     "Your OTP code is 123456",
	}

	t.Run("First message is queued successfully", func(t *testing.T) {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		req := httptest.NewRequest("POST", "/messages", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", fiber.StatusCreated, resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var response QueueSMSResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Message != "Message queued successfully" {
			t.Errorf("Expected message 'Message queued successfully', got '%s'", response.Message)
		}
		if response.ID == "" {
			t.Error("Expected non-empty message ID")
		}
	})

	t.Run("Duplicate message within interval is rejected", func(t *testing.T) {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		req := httptest.NewRequest("POST", "/messages", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusConflict {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", fiber.StatusConflict, resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var errorResponse map[string]interface{}
		if err := json.Unmarshal(body, &errorResponse); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if errorMsg, ok := errorResponse["error"].(string); ok {
			if errorMsg == "" {
				t.Error("Expected non-empty error message")
			}
		} else {
			t.Error("Expected error field in response")
		}
	})

	t.Run("Same message to different number is allowed", func(t *testing.T) {
		differentPayload := QueueSMSRequest{
			Topic:    "otp",
			ToNumber: "+9876543210",
			Body:     "Your OTP code is 123456",
		}

		bodyBytes, err := json.Marshal(differentPayload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		req := httptest.NewRequest("POST", "/messages", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", fiber.StatusCreated, resp.StatusCode, string(body))
		}
	})

	t.Run("Different message to same number is allowed", func(t *testing.T) {
		differentPayload := QueueSMSRequest{
			Topic:    "otp",
			ToNumber: "+1234567890",
			Body:     "Your OTP code is 654321",
		}

		bodyBytes, err := json.Marshal(differentPayload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		req := httptest.NewRequest("POST", "/messages", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", fiber.StatusCreated, resp.StatusCode, string(body))
		}
	})

	t.Run("Same message after interval expires is allowed", func(t *testing.T) {
		setupTestDB(t)
		defer teardownTestDB()

		message, err := db.CreateMessage("otp", "+1111111111", "Test message")
		if err != nil {
			t.Fatalf("Failed to create initial message: %v", err)
		}

		oldTime := time.Now().Add(-2 * time.Minute)
		db.GetDB().Model(&db.Message{}).Where("id = ?", message.ID).Update("created_at", oldTime)

		samePayload := QueueSMSRequest{
			Topic:    "otp",
			ToNumber: "+1111111111",
			Body:     "Test message",
		}

		bodyBytes, err := json.Marshal(samePayload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		req := httptest.NewRequest("POST", "/messages", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", fiber.StatusCreated, resp.StatusCode, string(body))
		}
	})
}

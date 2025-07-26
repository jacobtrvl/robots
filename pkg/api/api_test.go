package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jacobtrvl/robots/pkg/robot"
)

func TestEnqueueHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful enqueue",
			requestBody: map[string]string{
				"commands": "NESW",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"task_id"`,
		},
		{
			name: "empty commands",
			requestBody: map[string]string{
				"commands": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "commands are required",
		},
		{
			name: "invalid commands",
			requestBody: map[string]string{
				"commands": "NIW",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "command validation failed",
		},
		{
			name: "out of bounds commands",
			requestBody: map[string]string{
				"commands": "WWWWWWWWWWWWWW",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "out of bounds",
		},
		{
			name:           "invalid JSON body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warehouse := robot.NewMockWarehouse()
			api := NewRobotApi(warehouse)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/enqueue", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			api.EnqueueHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			responseBody := strings.TrimSpace(w.Body.String())
			if !strings.Contains(responseBody, tt.expectedBody) {
				t.Errorf("Expected response to contain %q, got %q", tt.expectedBody, responseBody)
			}
		})
	}
}

func TestStateHandler(t *testing.T) {
	warehouse := robot.NewMockWarehouse()
	api := NewRobotApi(warehouse)

	req := httptest.NewRequest(http.MethodGet, "/state", nil)
	w := httptest.NewRecorder()

	api.StateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var actualState robot.RobotState
	err := json.Unmarshal(w.Body.Bytes(), &actualState)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedState := robot.RobotState{X: 0, Y: 0, HasCrate: false}
	if actualState != expectedState {
		t.Errorf("Expected state %+v, got %+v", expectedState, actualState)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestCancel(t *testing.T) {
	warehouse := robot.NewMockWarehouse()
	api := NewRobotApi(warehouse)
	router := api.NewRouter()

	enqueueBody := map[string]string{"commands": "NSNSNSNSNSNSNSNSNSNS"}
	body, _ := json.Marshal(enqueueBody)

	req := httptest.NewRequest(http.MethodPost, "/enqueue", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Enqueue failed with status %d: %s", w.Code, w.Body.String())
	}

	var enqueueResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &enqueueResp)
	if err != nil {
		t.Fatalf("Failed to parse enqueue response: %v", err)
	}

	taskID := enqueueResp["task_id"]
	if taskID == "" {
		t.Fatal("No task_id in enqueue response")
	}

	cancelReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/cancel/%s", taskID), nil)
	cancelW := httptest.NewRecorder()

	router.ServeHTTP(cancelW, cancelReq)

	if cancelW.Code != http.StatusOK {
		t.Errorf("Cancel failed with status %d: %s", cancelW.Code, cancelW.Body.String())
	}
}

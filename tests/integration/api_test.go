package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/The-HOLE-Foundation/hole-docs/internal/config"
	"github.com/The-HOLE-Foundation/hole-docs/internal/mcp"
	"go.uber.org/zap"
)

func TestHealthEndpoint(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := &config.Config{
		Port:          8080,
		Host:          "0.0.0.0",
		MaxUploadSize: 500 * 1024 * 1024,
		TempDir:       "/tmp/godocs-test",
	}

	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	router := server.Router()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "health_endpoint_returns_200",
			method:         "GET",
			path:           "/health",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if result["status"] != "healthy" {
					t.Errorf("Expected status 'healthy', got %v", result["status"])
				}
				if result["version"] != "0.1.0" {
					t.Errorf("Expected version '0.1.0', got %v", result["version"])
				}
				if uptime, ok := result["uptime_seconds"]; !ok || uptime.(float64) < 0 {
					t.Errorf("Invalid uptime: %v", uptime)
				}
			},
		},
		{
			name:           "info_endpoint_returns_200",
			method:         "GET",
			path:           "/info",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if result["name"] != "GoDocs MCP Server" {
					t.Errorf("Expected name 'GoDocs MCP Server', got %v", result["name"])
				}
				if tools, ok := result["tools"].([]interface{}); !ok || len(tools) == 0 {
					t.Errorf("Expected tools array, got %v", result["tools"])
				}
			},
		},
		{
			name:           "tools_endpoint_returns_200",
			method:         "GET",
			path:           "/tools",
			expectedStatus: 200,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if tools, ok := result["tools"].(map[string]interface{}); !ok || len(tools) == 0 {
					t.Errorf("Expected tools map, got %v", result["tools"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestInvokeEndpoint(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := &config.Config{
		Port:          8080,
		Host:          "0.0.0.0",
		MaxUploadSize: 500 * 1024 * 1024,
		TempDir:       "/tmp/godocs-test",
	}

	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	router := server.Router()

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "invoke_list_tools_success",
			method:         "POST",
			path:           "/invoke",
			body:           map[string]interface{}{"toolName": "list_tools", "args": map[string]interface{}{}},
			expectedStatus: 200,
			checkError:     false,
		},
		{
			name:           "invoke_health_check_success",
			method:         "POST",
			path:           "/invoke",
			body:           map[string]interface{}{"toolName": "health_check", "args": map[string]interface{}{}},
			expectedStatus: 200,
			checkError:     false,
		},
		{
			name:           "invoke_nonexistent_tool",
			method:         "POST",
			path:           "/invoke",
			body:           map[string]interface{}{"toolName": "nonexistent_tool", "args": map[string]interface{}{}},
			expectedStatus: 404,
			checkError:     true,
		},
		{
			name:           "invoke_invalid_json",
			method:         "POST",
			path:           "/invoke",
			body:           "invalid json",
			expectedStatus: 400,
			checkError:     true,
		},
		{
			name:           "invoke_with_get_method_fails",
			method:         "GET",
			path:           "/invoke",
			body:           nil,
			expectedStatus: 405,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				body = bytes.NewReader(jsonBody)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d (body: %s)", tt.expectedStatus, w.Code, w.Body.String())
			}

			if !tt.checkError {
				var result map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Errorf("Failed to parse response: %v", err)
				}
				if _, hasError := result["error"]; hasError {
					t.Errorf("Expected no error, but got one")
				}
			}
		})
	}
}

func TestUploadEndpoint(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := &config.Config{
		Port:          8080,
		Host:          "0.0.0.0",
		MaxUploadSize: 500 * 1024 * 1024,
		TempDir:       "/tmp/godocs-test",
	}

	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	router := server.Router()

	t.Run("upload_valid_file", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "test.pdf")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}

		fileContent := []byte("%PDF-1.4\n%mock pdf content")
		part.Write(fileContent)
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if fileID, ok := result["file_id"]; !ok || fileID == "" {
			t.Errorf("Expected file_id in response, got %v", result)
		}

		if filename, ok := result["filename"]; !ok || filename != "test.pdf" {
			t.Errorf("Expected filename 'test.pdf', got %v", filename)
		}

		if size, ok := result["size"]; !ok || size.(float64) <= 0 {
			t.Errorf("Expected positive size, got %v", size)
		}
	})

	t.Run("upload_with_get_method_fails", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/upload", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 405 {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("upload_without_file_fails", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := &config.Config{
		Port:          8080,
		Host:          "0.0.0.0",
		MaxUploadSize: 500 * 1024 * 1024,
		TempDir:       "/tmp/godocs-test",
	}

	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	router := server.Router()

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}

	if allowOrigin := w.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "http://example.com" {
		t.Logf("CORS header: %s (expected http://example.com or *)", allowOrigin)
	}
}

func BenchmarkHealthEndpoint(b *testing.B) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := &config.Config{
		Port:          8080,
		Host:          "0.0.0.0",
		MaxUploadSize: 500 * 1024 * 1024,
		TempDir:       "/tmp/godocs-test",
	}

	server, _ := mcp.NewServer(cfg, logger)
	router := server.Router()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func TestRequestTimeout(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := &config.Config{
		Port:          8080,
		Host:          "0.0.0.0",
		MaxUploadSize: 500 * 1024 * 1024,
		TempDir:       "/tmp/godocs-test",
	}

	server, _ := mcp.NewServer(cfg, logger)
	router := server.Router()

	done := make(chan bool)
	go func() {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		done <- w.Code == 200
	}()

	select {
	case result := <-done:
		if !result {
			t.Fatal("Request failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Request timeout")
	}
}

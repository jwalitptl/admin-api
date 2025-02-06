package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	baseURL       = "http://localhost:8080/api/v1"
	authToken     string
	accountID     string
	orgID         string
	clinicID      string
	patientID     string
	clinicianID   string
	roleID        string
	appointmentID string
)

// APIResponse represents the API response structure
type APIResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// TestResponse wraps the API response for testing
type TestResponse struct {
	Status  string
	Message string
	Data    map[string]interface{}
	RawData string
}

func (r TestResponse) IsSuccess() bool {
	return r.Status == "success"
}

func (r TestResponse) GetString(key string) string {
	if r.Data == nil {
		return ""
	}
	if v, ok := r.Data[key].(string); ok {
		return v
	}
	return ""
}

func checkAPIServer() error {
	// First try a simple ping without the health endpoint
	client := &http.Client{Timeout: 5 * time.Second}

	// Try different endpoints in order
	endpoints := []string{
		"/health",     // Try health endpoint first
		"/auth/login", // Try login endpoint as fallback
		"",            // Try base path as last resort
	}

	var lastErr error
	for _, endpoint := range endpoints {
		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Any response means server is up
		if resp.StatusCode != http.StatusNotFound {
			return nil
		}
	}

	return fmt.Errorf("API server not reachable: %v", lastErr)
}

func TestMain(m *testing.M) {
	// Add retry logic for server startup
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		if err := checkAPIServer(); err != nil {
			if i == maxRetries-1 {
				fmt.Printf("Error: %v\nMake sure the API server is running at %s\n", err, baseURL)
				os.Exit(1)
			}
			fmt.Printf("Waiting for API server (attempt %d/%d)...\n", i+1, maxRetries)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	// Setup
	setupAuth()
	setupTestData()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

func setupAuth() {
	// Login or create admin
	loginResp := makeRequest("POST", "/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "admin123",
	}, "")

	if !loginResp.IsSuccess() {
		// Create admin if login fails
		createResp := makeRequest("POST", "/clinicians", map[string]string{
			"email":    "admin@example.com",
			"name":     "Admin User",
			"password": "admin123",
		}, "")

		if !createResp.IsSuccess() {
			fmt.Printf("Failed to create admin: %s\n", createResp.Message)
			os.Exit(1)
		}

		// Retry login
		loginResp = makeRequest("POST", "/auth/login", map[string]string{
			"email":    "admin@example.com",
			"password": "admin123",
		}, "")
	}

	if !loginResp.IsSuccess() {
		fmt.Printf("Failed to login: %s\n", loginResp.Message)
		os.Exit(1)
	}

	authToken = loginResp.GetString("access_token")
	if authToken == "" {
		fmt.Println("Failed to get auth token")
		os.Exit(1)
	}
}

func setupTestData() {
	// Clean up any existing test data
	cleanup()

	// Create account first
	accountResp := makeRequest("POST", "/accounts", map[string]interface{}{
		"name":   uniqueName("Test Account"),
		"email":  fmt.Sprintf("account_%d@example.com", time.Now().Unix()),
		"status": "active",
	}, authToken)
	if !accountResp.IsSuccess() {
		fmt.Printf("Failed to create account: %s\n", accountResp.Message)
		os.Exit(1)
	}
	accountID = accountResp.GetString("id")

	// Create organization
	orgResp := makeRequest("POST", fmt.Sprintf("/accounts/%s/organizations", accountID), map[string]interface{}{
		"name":   uniqueName("Test Organization"),
		"status": "active",
	}, authToken)
	if !orgResp.IsSuccess() {
		fmt.Printf("Failed to create organization: %s\n", orgResp.Message)
		os.Exit(1)
	}
	orgID = orgResp.GetString("id")
}

func cleanup() {
	if authToken == "" {
		return
	}

	// Delete test resources in reverse order of dependencies
	if appointmentID != "" {
		makeRequest("DELETE", fmt.Sprintf("/appointments/%s", appointmentID), nil, authToken)
		appointmentID = ""
	}
	if patientID != "" {
		makeRequest("DELETE", fmt.Sprintf("/patients/%s", patientID), nil, authToken)
		patientID = ""
	}
	if clinicianID != "" {
		makeRequest("DELETE", fmt.Sprintf("/clinicians/%s", clinicianID), nil, authToken)
		clinicianID = ""
	}
	if roleID != "" {
		// First delete role permissions
		makeRequest("DELETE", fmt.Sprintf("/rbac/roles/%s/permissions", roleID), nil, authToken)
		// Then delete role
		makeRequest("DELETE", fmt.Sprintf("/rbac/roles/%s", roleID), nil, authToken)
		roleID = ""
	}
	if clinicID != "" {
		makeRequest("DELETE", fmt.Sprintf("/clinics/%s?organization_id=%s", clinicID, orgID), nil, authToken)
		clinicID = ""
	}
	// Delete all clinics for this organization
	clinicsResp := makeRequest("GET", fmt.Sprintf("/clinics?organization_id=%s", orgID), nil, authToken)
	if clinicsResp.IsSuccess() {
		if clinics, ok := clinicsResp.Data["clinics"].([]interface{}); ok {
			for _, clinic := range clinics {
				if c, ok := clinic.(map[string]interface{}); ok {
					makeRequest("DELETE", fmt.Sprintf("/clinics/%s?organization_id=%s", c["id"], orgID), nil, authToken)
				}
			}
		}
	}
	if orgID != "" {
		makeRequest("DELETE", fmt.Sprintf("/organizations/%s", orgID), nil, authToken)
		orgID = ""
	}
}

func makeRequest(method, path string, body interface{}, token string) TestResponse {
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(method, baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return TestResponse{Status: "error", Message: err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return TestResponse{Status: "error", Message: err.Error()}
	}
	defer response.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return TestResponse{Status: "error", Message: err.Error()}
	}

	// Consider both 200 and 201 as success
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return TestResponse{
			Status:  "error",
			Message: fmt.Sprintf("HTTP %d: %s", response.StatusCode, string(respBody)),
		}
	}

	// First try to unmarshal into APIResponse
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// Debug: Print the raw response
		return TestResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to parse response: %s\nRaw response: %s", err.Error(), string(respBody)),
		}
	}

	// Create TestResponse
	testResp := TestResponse{
		Status:  apiResp.Status,
		Message: apiResp.Message,
		RawData: string(apiResp.Data),
	}

	// Try to unmarshal Data into map if it exists
	if len(apiResp.Data) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(apiResp.Data, &data); err == nil {
			testResp.Data = data
		}
	}

	return testResp
}

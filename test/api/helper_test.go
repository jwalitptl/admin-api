package api_test

import (
	"fmt"
	"testing"
	"time"
)

// Helper function to generate unique names
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().Unix())
}

// Helper to create test clinic
func createTestClinic(t *testing.T) string {
	if orgID == "" {
		t.Fatal("Organization ID is required")
	}

	resp := makeRequest("POST", "/clinics", map[string]interface{}{
		"name":            uniqueName("Test Clinic"),
		"location":        "123 Test St",
		"organization_id": orgID,
		"status":          "active",
	}, authToken)

	if !resp.IsSuccess() {
		t.Logf("Failed to create test clinic: %s", resp.Message)
		return ""
	}
	return resp.GetString("id")
}

// Helper to create test patient
func createTestPatient(t *testing.T, clinicID string) string {
	resp := makeRequest("POST", "/patients", map[string]interface{}{
		"name":            uniqueName("Test Patient"),
		"email":           fmt.Sprintf("patient_%d@example.com", time.Now().Unix()),
		"phone":           "+1234567890",
		"date_of_birth":   "1990-01-01",
		"organization_id": orgID,
		"clinic_id":       clinicID,
		"status":          "active",
	}, authToken)

	if !resp.IsSuccess() {
		t.Fatalf("Failed to create test patient: %s", resp.Message)
	}
	return resp.GetString("id")
}

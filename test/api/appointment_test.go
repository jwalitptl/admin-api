package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppointmentFlow(t *testing.T) {
	if orgID == "" {
		t.Fatal("Organization setup failed")
	}

	// Create prerequisites
	clinicID := createTestClinic(t)
	if clinicID == "" {
		t.Fatal("Failed to create test clinic")
	}

	patientID := createTestPatient(t, clinicID)
	if patientID == "" {
		t.Fatal("Failed to create test patient")
	}

	// Create clinician
	email := fmt.Sprintf("doctor_%d@example.com", time.Now().UnixNano())
	clinicianResp := makeRequest("POST", "/clinicians", map[string]interface{}{
		"email":           email,
		"name":            uniqueName("Test Doctor"),
		"password":        "test123",
		"organization_id": orgID,
		"status":          "active",
	}, authToken)
	assert.True(t, clinicianResp.IsSuccess())
	clinicianID := clinicianResp.GetString("id")
	assert.NotEmpty(t, clinicianID)

	// Create appointment
	now := time.Now().UTC()
	startTime := now.Add(24 * time.Hour).Format(time.RFC3339)
	endTime := now.Add(25 * time.Hour).Format(time.RFC3339)

	createReq := map[string]interface{}{
		"clinician_id":    clinicianID,
		"patient_id":      patientID,
		"clinic_id":       clinicID,
		"start_time":      startTime,
		"end_time":        endTime,
		"status":          "scheduled",
		"organization_id": orgID,
		"type":            "consultation",
		"duration":        60,
	}
	createResp := makeRequest("POST", "/appointments", createReq, authToken)
	assert.True(t, createResp.IsSuccess(), "Failed to create appointment: %s\nRequest: %+v\nResponse: %+v",
		createResp.Message, createReq, createResp)
	appointmentID = createResp.GetString("id")
	assert.NotEmpty(t, appointmentID)

	// Get appointment
	getURL := fmt.Sprintf("/appointments/%s?organization_id=%s", appointmentID, orgID)
	getResp := makeRequest("GET", getURL, nil, authToken)
	assert.True(t, getResp.IsSuccess(), "Failed to get appointment: %s\nURL: %s\nResponse: %+v",
		getResp.Message, getURL, getResp)
	assert.Equal(t, clinicianID, getResp.Data["clinician_id"])
	assert.Equal(t, patientID, getResp.Data["patient_id"])
	assert.Equal(t, "scheduled", getResp.Data["status"])

	// List appointments
	listURL := fmt.Sprintf("/appointments?organization_id=%s&clinic_id=%s", orgID, clinicID)
	listResp := makeRequest("GET", listURL, nil, authToken)
	assert.True(t, listResp.IsSuccess(), "Failed to list appointments: %s\nURL: %s\nResponse: %+v",
		listResp.Message, listURL, listResp)

	// Parse and verify appointments
	var appointments []interface{}
	if err := json.Unmarshal([]byte(listResp.RawData), &appointments); err != nil {
		t.Fatalf("Failed to parse appointments: %v", err)
	}
	assert.NotEmpty(t, appointments, "Appointments list should not be empty")

	// Find our appointment in the list
	found := false
	for _, appt := range appointments {
		if a, ok := appt.(map[string]interface{}); ok {
			if a["id"] == appointmentID {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Created appointment should be in the list")

	// Update appointment
	updateReq := map[string]interface{}{
		"clinician_id":    clinicianID,
		"patient_id":      patientID,
		"clinic_id":       clinicID,
		"start_time":      startTime,
		"end_time":        endTime,
		"status":          "confirmed",
		"organization_id": orgID,
		"type":            "consultation",
		"duration":        60,
	}
	updateURL := fmt.Sprintf("/appointments/%s?organization_id=%s", appointmentID, orgID)
	updateResp := makeRequest("PUT", updateURL, updateReq, authToken)
	assert.True(t, updateResp.IsSuccess(), "Failed to update appointment: %s\nURL: %s\nRequest: %+v\nResponse: %+v",
		updateResp.Message, updateURL, updateReq, updateResp)

	// Verify update
	verifyResp := makeRequest("GET", fmt.Sprintf("/appointments/%s?organization_id=%s", appointmentID, orgID), nil, authToken)
	assert.True(t, verifyResp.IsSuccess(), "Failed to verify appointment: %s", verifyResp.Message)
	assert.Equal(t, "confirmed", verifyResp.Data["status"])

	// Test availability
	availResp := makeRequest("GET", fmt.Sprintf("/appointments/availability?clinician_id=%s&date=%s&organization_id=%s",
		clinicianID, time.Now().Format("2006-01-02"), orgID), nil, authToken)
	assert.True(t, availResp.IsSuccess())
}

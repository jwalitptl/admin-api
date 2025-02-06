package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPatientFlow(t *testing.T) {
	if orgID == "" {
		t.Fatal("Organization setup failed")
	}

	clinicID := createTestClinic(t)
	if clinicID == "" {
		t.Fatal("Failed to create test clinic")
	}

	email := fmt.Sprintf("patient_%d@example.com", time.Now().UnixNano())
	name := uniqueName("Test Patient")

	// Create patient
	createResp := makeRequest("POST", "/patients", map[string]interface{}{
		"name":            name,
		"email":           email,
		"date_of_birth":   time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"organization_id": orgID,
		"clinic_id":       clinicID,
		"status":          "active",
		"address":         "123 Patient St, Test City, TS 12345",
		"gender":          "other",
	}, authToken)

	assert.True(t, createResp.IsSuccess())
	patientID = createResp.GetString("id")
	assert.NotEmpty(t, patientID)

	// Get patient
	getResp := makeRequest("GET", fmt.Sprintf("/patients/%s", patientID), nil, authToken)
	assert.True(t, getResp.IsSuccess())
	assert.Equal(t, name, getResp.Data["name"])
	assert.Equal(t, email, getResp.Data["email"])
	assert.Equal(t, "active", getResp.Data["status"])
	assert.Equal(t, clinicID, getResp.Data["clinic_id"])
}

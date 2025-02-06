package api_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClinicFlow(t *testing.T) {
	if orgID == "" {
		t.Fatal("Organization setup failed")
	}

	name := uniqueName("Test Clinic")

	// Create clinic
	createResp := makeRequest("POST", "/clinics", map[string]interface{}{
		"name":            name,
		"location":        "123 Test St",
		"organization_id": orgID,
		"status":          "active",
		"address":         "123 Test St, Test City, TS 12345",
		"phone":           "+1234567890",
	}, authToken)

	assert.True(t, createResp.IsSuccess(), "Failed to create clinic: %s", createResp.Message)
	clinicID = createResp.GetString("id")
	assert.NotEmpty(t, clinicID)

	// Get clinic
	getResp := makeRequest("GET", fmt.Sprintf("/clinics/%s?organization_id=%s", clinicID, orgID), nil, authToken)
	assert.True(t, getResp.IsSuccess())
	assert.Equal(t, name, getResp.Data["name"])
	assert.Equal(t, "active", getResp.Data["status"])

	// List clinics
	listResp := makeRequest("GET", fmt.Sprintf("/clinics?organization_id=%s", orgID), nil, authToken)
	assert.True(t, listResp.IsSuccess())

	// Update clinic
	newName := uniqueName("Updated Clinic")
	updateResp := makeRequest("PUT", fmt.Sprintf("/clinics/%s", clinicID), map[string]interface{}{
		"name":            newName,
		"location":        "456 Updated St",
		"organization_id": orgID,
		"status":          "active",
		"address":         "456 Updated St, Test City, TS 12345",
		"phone":           "+1234567890",
	}, authToken)
	assert.True(t, updateResp.IsSuccess(), "Failed to update clinic: %s", updateResp.Message)

	// Verify update
	verifyResp := makeRequest("GET", fmt.Sprintf("/clinics/%s?organization_id=%s", clinicID, orgID), nil, authToken)
	assert.True(t, verifyResp.IsSuccess())
	assert.Equal(t, newName, verifyResp.Data["name"])
}

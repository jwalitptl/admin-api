package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClinicianFlow(t *testing.T) {
	if orgID == "" {
		t.Fatal("Organization setup failed")
	}

	email := fmt.Sprintf("doctor_%d@example.com", time.Now().UnixNano())
	createResp := makeRequest("POST", "/clinicians", map[string]interface{}{
		"email":           email,
		"name":            uniqueName("Test Doctor"),
		"password":        "test123",
		"organization_id": orgID,
		"status":          "active",
	}, authToken)

	assert.True(t, createResp.IsSuccess())
	clinicianID = createResp.GetString("id")
	assert.NotEmpty(t, clinicianID)

	// Get clinician
	getResp := makeRequest("GET", fmt.Sprintf("/clinicians/%s", clinicianID), nil, authToken)
	assert.True(t, getResp.IsSuccess())
	assert.Equal(t, createResp.Data["name"], getResp.Data["name"])

	// Update clinician with all required fields
	newName := uniqueName("Updated Doctor")
	updateResp := makeRequest("PUT", fmt.Sprintf("/clinicians/%s", clinicianID), map[string]interface{}{
		"name":            newName,
		"email":           email,
		"organization_id": orgID,
		"status":          "active",
	}, authToken)
	assert.True(t, updateResp.IsSuccess())

	// Verify update
	verifyResp := makeRequest("GET", fmt.Sprintf("/clinicians/%s", clinicianID), nil, authToken)
	assert.True(t, verifyResp.IsSuccess())
	assert.Equal(t, newName, verifyResp.Data["name"].(string))

	// List clinicians
	listResp := makeRequest("GET", "/clinicians", nil, authToken)
	assert.True(t, listResp.IsSuccess())

	// Test clinician roles
	roleResp := makeRequest("GET", fmt.Sprintf("/clinicians/%s/roles/organizations/%s",
		clinicianID, orgID), nil, authToken)
	assert.True(t, roleResp.IsSuccess())
}

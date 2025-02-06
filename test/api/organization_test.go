package api_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganizationFlow(t *testing.T) {
	if accountID == "" {
		t.Fatal("Account setup failed")
	}

	// Create organization
	createResp := makeRequest("POST", fmt.Sprintf("/accounts/%s/organizations", accountID), map[string]interface{}{
		"name":   uniqueName("Test Organization"),
		"status": "active",
	}, authToken)

	assert.True(t, createResp.IsSuccess())
	orgID = createResp.GetString("id")
	assert.NotEmpty(t, orgID)

	// Get organization
	getResp := makeRequest("GET", fmt.Sprintf("/organizations/%s", orgID), nil, authToken)
	assert.True(t, getResp.IsSuccess())
	assert.Equal(t, createResp.Data["name"], getResp.Data["name"])
}

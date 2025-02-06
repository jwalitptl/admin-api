package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAccountFlow(t *testing.T) {
	// Get initial account count
	initialResp := makeRequest("GET", "/accounts", nil, authToken)
	assert.True(t, initialResp.IsSuccess())
	var initialAccounts []interface{}
	json.Unmarshal([]byte(initialResp.RawData), &initialAccounts)
	initialCount := len(initialAccounts)

	// Create account
	email := fmt.Sprintf("account_%d@example.com", time.Now().UnixNano())
	createResp := makeRequest("POST", "/accounts", map[string]interface{}{
		"name":   uniqueName("Test Account"),
		"email":  email,
		"status": "active",
	}, authToken)

	assert.True(t, createResp.IsSuccess(), "Failed to create account: %s", createResp.Message)
	accountID = createResp.GetString("id")
	assert.NotEmpty(t, accountID)

	// Get account by ID
	getResp := makeRequest("GET", fmt.Sprintf("/accounts/%s", accountID), nil, authToken)
	assert.True(t, getResp.IsSuccess())
	assert.Equal(t, createResp.Data["name"], getResp.Data["name"])

	// List accounts - verify count increased
	listResp := makeRequest("GET", "/accounts", nil, authToken)
	assert.True(t, listResp.IsSuccess())
	var accounts []interface{}
	json.Unmarshal([]byte(listResp.RawData), &accounts)
	assert.Equal(t, initialCount+1, len(accounts))
}

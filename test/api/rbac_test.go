package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRBACFlow(t *testing.T) {
	// Skip if organization setup failed
	if orgID == "" {
		t.Skip("Organization setup failed, skipping RBAC test")
	}

	timestamp := time.Now().Unix()

	// Create role
	roleResp := makeRequest("POST", "/rbac/roles", map[string]interface{}{
		"name":            fmt.Sprintf("doctor_%d", timestamp),
		"description":     "Doctor role",
		"organization_id": orgID,
	}, authToken)

	assert.True(t, roleResp.IsSuccess(), "Failed to create role: %s", roleResp.Message)
	roleID = roleResp.GetString("id")
	assert.NotEmpty(t, roleID, "Role ID should not be empty")

	// Create permission
	permResp := makeRequest("POST", "/rbac/permissions", map[string]interface{}{
		"name":        fmt.Sprintf("patient:read:%d", timestamp),
		"description": "Can read patient data",
	}, authToken)

	assert.True(t, permResp.IsSuccess(), "Failed to create permission: %s", permResp.Message)
	permID := permResp.GetString("id")
	assert.NotEmpty(t, permID, "Permission ID should not be empty")

	// Assign permission to role
	assignResp := makeRequest("POST", fmt.Sprintf("/rbac/roles/%s/permissions/%s", roleID, permID), nil, authToken)
	assert.True(t, assignResp.IsSuccess(), "Failed to assign permission: %s", assignResp.Message)
}

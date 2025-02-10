package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jwalitptl/admin-api/pkg/errors"
)

// Response wraps all API responses
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents API error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_pages"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// RespondWithSuccess sends a success response
func RespondWithSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// RespondWithError sends an error response
func RespondWithError(c *gin.Context, err error) {
	var statusCode int
	var message string

	if appErr, ok := err.(*errors.AppError); ok {
		statusCode = int(appErr.Code)
		message = appErr.Message
	} else {
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	}

	c.JSON(statusCode, Response{
		Success: false,
		Error: &Error{
			Code:    statusCode,
			Message: message,
		},
	})
}

// RespondWithPagination sends a paginated response
func RespondWithPagination(c *gin.Context, data interface{}, page, pageSize, total int) {
	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: PaginatedResponse{
			Data: data,
			Pagination: Pagination{
				Page:      page,
				PageSize:  pageSize,
				Total:     total,
				TotalPage: totalPages,
			},
		},
	})
}

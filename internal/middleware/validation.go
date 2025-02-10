package middleware

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationConfig represents validation middleware configuration
type ValidationConfig struct {
	DisableDefaultValidators bool
	CustomValidators         map[string]validator.Func
	CustomErrorMessages      map[string]string
}

func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		CustomErrorMessages: map[string]string{
			"required": "Field is required",
			"email":    "Invalid email format",
			"min":      "Value is too short",
			"max":      "Value is too long",
		},
	}
}

// Validation middleware handles request validation
func Validation(config ValidationConfig) gin.HandlerFunc {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Register custom validators
		for tag, fn := range config.CustomValidators {
			if err := v.RegisterValidation(tag, fn); err != nil {
				panic(err)
			}
		}

		// Register custom error messages
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			return name
		})
	}

	return func(c *gin.Context) {
		c.Next()

		// Check for validation errors
		if len(c.Errors) > 0 {
			var validationErrors []ValidationError
			for _, err := range c.Errors {
				if errs, ok := err.Err.(validator.ValidationErrors); ok {
					for _, e := range errs {
						msg := config.CustomErrorMessages[e.Tag()]
						if msg == "" {
							msg = e.Error()
						}
						validationErrors = append(validationErrors, ValidationError{
							Field:   e.Field(),
							Message: msg,
						})
					}
				}
			}

			if len(validationErrors) > 0 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"errors": validationErrors,
				})
				return
			}
		}
	}
}

package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Validator provides validation functionality
type Validator interface {
	Validate(interface{}) error
	ValidateField(field string, value interface{}, rules ...string) error
}

type validator struct {
	tagName string
}

func New() Validator {
	return &validator{
		tagName: "validate",
	}
}

func (v *validator) Validate(obj interface{}) error {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		tag := field.Tag.Get(v.tagName)
		if tag == "" {
			continue
		}

		if err := v.ValidateField(field.Name, value.Field(i).Interface(), strings.Split(tag, ",")...); err != nil {
			return err
		}
	}

	return nil
}

func (v *validator) ValidateField(field string, value interface{}, rules ...string) error {
	for _, rule := range rules {
		if err := v.validateRule(field, value, rule); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateRule(field string, value interface{}, rule string) error {
	switch {
	case rule == "required":
		if isZero(value) {
			return fmt.Errorf("%s is required", field)
		}
	case strings.HasPrefix(rule, "min="):
		min := strings.TrimPrefix(rule, "min=")
		if err := validateMin(value, min); err != nil {
			return fmt.Errorf("%s %v", field, err)
		}
	case strings.HasPrefix(rule, "max="):
		max := strings.TrimPrefix(rule, "max=")
		if err := validateMax(value, max); err != nil {
			return fmt.Errorf("%s %v", field, err)
		}
	case rule == "email":
		if str, ok := value.(string); ok {
			if !isValidEmail(str) {
				return fmt.Errorf("%s must be a valid email", field)
			}
		}
	}
	return nil
}

func isZero(value interface{}) bool {
	v := reflect.ValueOf(value)
	return !v.IsValid() || reflect.DeepEqual(value, reflect.Zero(v.Type()).Interface())
}

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

func validateMin(value interface{}, min string) error {
	// For string length validation
	if str, ok := value.(string); ok {
		if len(str) < len(min) {
			return fmt.Errorf("must be at least %s characters long", min)
		}
	}
	return nil
}

func validateMax(value interface{}, max string) error {
	// For string length validation
	if str, ok := value.(string); ok {
		if len(str) > len(max) {
			return fmt.Errorf("must not exceed %s characters", max)
		}
	}
	return nil
}

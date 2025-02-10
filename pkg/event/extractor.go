package event

import (
	"reflect"
	"strings"
)

type DefaultFieldExtractor struct{}

func (e *DefaultFieldExtractor) ExtractFields(obj interface{}, fields []string) map[string]interface{} {
	result := make(map[string]interface{})
	if obj == nil || len(fields) == 0 {
		return result
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = strings.ToLower(field.Name)
		}
		jsonTag = strings.Split(jsonTag, ",")[0]

		if contains(fields, jsonTag) {
			result[jsonTag] = val.Field(i).Interface()
		}
	}
	return result
}

func (e *DefaultFieldExtractor) ExtractChanges(old, new interface{}, fields []string) map[string]interface{} {
	changes := make(map[string]interface{})
	if old == nil || new == nil || len(fields) == 0 {
		return changes
	}

	oldFields := e.ExtractFields(old, fields)
	newFields := e.ExtractFields(new, fields)

	for field, newValue := range newFields {
		if oldValue, exists := oldFields[field]; exists {
			if !reflect.DeepEqual(oldValue, newValue) {
				changes[field] = map[string]interface{}{
					"old": oldValue,
					"new": newValue,
				}
			}
		}
	}
	return changes
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

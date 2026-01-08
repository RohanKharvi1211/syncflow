package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MetadataSchema represents the schema structure for app metadata
type MetadataSchema struct {
	RequiredFields   []string          `json:"required_fields"`
	OptionalFields   []string          `json:"optional_fields"`
	FieldDescriptions map[string]string `json:"field_descriptions"`
}

// ValidateMetadata validates metadata against an app's metadata schema
func ValidateMetadata(metadataJSON string, schemaJSON string) error {
	// Parse the metadata schema
	var schema MetadataSchema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return fmt.Errorf("invalid metadata schema: %w", err)
	}

	// Parse the metadata data
	var metadataData map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadataData); err != nil {
		return fmt.Errorf("invalid metadata format: %w", err)
	}

	// Check required fields
	var missingFields []string
	for _, field := range schema.RequiredFields {
		value, exists := metadataData[field]
		if !exists || value == nil || value == "" {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required metadata fields: %s", strings.Join(missingFields, ", "))
	}

	// Optional: Validate that all fields in metadata are either required or optional
	// This prevents typos or unknown fields
	allAllowedFields := make(map[string]bool)
	for _, field := range schema.RequiredFields {
		allAllowedFields[field] = true
	}
	for _, field := range schema.OptionalFields {
		allAllowedFields[field] = true
	}

	var unknownFields []string
	for field := range metadataData {
		if !allAllowedFields[field] {
			unknownFields = append(unknownFields, field)
		}
	}

	if len(unknownFields) > 0 {
		return fmt.Errorf("unknown metadata fields: %s. Allowed fields: %s", 
			strings.Join(unknownFields, ", "),
			strings.Join(getAllFields(schema), ", "))
	}

	return nil
}

// GetAllRequiredFields returns all required fields from a schema
func GetAllRequiredFields(schemaJSON string) ([]string, error) {
	var schema MetadataSchema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, fmt.Errorf("invalid metadata schema: %w", err)
	}
	return schema.RequiredFields, nil
}

// GetAllFields returns all fields (required + optional) from a schema
func getAllFields(schema MetadataSchema) []string {
	allFields := make([]string, 0)
	allFields = append(allFields, schema.RequiredFields...)
	allFields = append(allFields, schema.OptionalFields...)
	return allFields
}

// GetFieldDescription returns the description for a field
func GetFieldDescription(schemaJSON string, fieldName string) (string, error) {
	var schema MetadataSchema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return "", fmt.Errorf("invalid metadata schema: %w", err)
	}
	
	if desc, exists := schema.FieldDescriptions[fieldName]; exists {
		return desc, nil
	}
	
	return "", fmt.Errorf("field description not found for: %s", fieldName)
}





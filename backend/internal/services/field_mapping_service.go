package services

import (
	"encoding/json"
	"fmt"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"strings"
)

type FieldMappingService struct{}

func NewFieldMappingService() *FieldMappingService {
	return &FieldMappingService{}
}

// TransformData applies field mappings to transform source data to target format
func (f *FieldMappingService) TransformData(companyID uint, sourceSystem, targetSystem, sourceObject, targetObject string, sourceData map[string]interface{}) (map[string]interface{}, error) {
	// Get field mappings for this company and object combination
	var mappings []models.FieldMapping
	result := config.DB.Where("company_id = ? AND source_system = ? AND target_system = ? AND source_object = ? AND target_object = ? AND is_active = ?",
		companyID, sourceSystem, targetSystem, sourceObject, targetObject, true).Find(&mappings)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get field mappings: %v", result.Error)
	}

	if len(mappings) == 0 {
		return nil, fmt.Errorf("no field mappings found for %s -> %s (%s -> %s)", sourceSystem, targetSystem, sourceObject, targetObject)
	}

	// Transform the data
	targetData := make(map[string]interface{})
	
	for _, mapping := range mappings {
		value, err := f.applyMapping(mapping, sourceData)
		if err != nil {
			if mapping.IsRequired {
				return nil, fmt.Errorf("required field mapping failed for %s: %v", mapping.SourceField, err)
			}
			continue // Skip non-required fields that fail
		}
		
		if value != nil {
			targetData[mapping.TargetField] = value
		}
	}

	return targetData, nil
}

func (f *FieldMappingService) applyMapping(mapping models.FieldMapping, sourceData map[string]interface{}) (interface{}, error) {
	switch mapping.MappingType {
	case "direct":
		return f.directMapping(mapping, sourceData)
	case "transform":
		return f.transformMapping(mapping, sourceData)
	case "constant":
		return f.constantMapping(mapping)
	default:
		return nil, fmt.Errorf("unknown mapping type: %s", mapping.MappingType)
	}
}

func (f *FieldMappingService) directMapping(mapping models.FieldMapping, sourceData map[string]interface{}) (interface{}, error) {
	// Handle nested field access (e.g., "Address.Street")
	value := f.getNestedValue(sourceData, mapping.SourceField)
	return value, nil
}

func (f *FieldMappingService) transformMapping(mapping models.FieldMapping, sourceData map[string]interface{}) (interface{}, error) {
	// Parse transform rule
	var transformRule map[string]interface{}
	if err := json.Unmarshal([]byte(mapping.TransformRule), &transformRule); err != nil {
		return nil, fmt.Errorf("invalid transform rule: %v", err)
	}

	// Get source value
	sourceValue := f.getNestedValue(sourceData, mapping.SourceField)
	
	// Apply transformation based on rule type
	ruleType, ok := transformRule["type"].(string)
	if !ok {
		return nil, fmt.Errorf("transform rule must have a type")
	}

	switch ruleType {
	case "concat":
		return f.concatTransform(sourceValue, transformRule)
	case "split":
		return f.splitTransform(sourceValue, transformRule)
	case "format":
		return f.formatTransform(sourceValue, transformRule)
	case "lookup":
		return f.lookupTransform(sourceValue, transformRule)
	default:
		return nil, fmt.Errorf("unknown transform type: %s", ruleType)
	}
}

func (f *FieldMappingService) constantMapping(mapping models.FieldMapping) (interface{}, error) {
	var constantValue interface{}
	if err := json.Unmarshal([]byte(mapping.TransformRule), &constantValue); err != nil {
		return nil, fmt.Errorf("invalid constant value: %v", err)
	}
	return constantValue, nil
}

func (f *FieldMappingService) getNestedValue(data map[string]interface{}, fieldPath string) interface{} {
	parts := strings.Split(fieldPath, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	
	return nil
}

func (f *FieldMappingService) concatTransform(sourceValue interface{}, rule map[string]interface{}) (interface{}, error) {
	fields, ok := rule["fields"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("concat rule must have fields array")
	}
	
	separator, _ := rule["separator"].(string)
	if separator == "" {
		separator = " "
	}
	
	var parts []string
	for _, field := range fields {
		if fieldStr, ok := field.(string); ok {
			if fieldStr == "source" {
				if sourceStr, ok := sourceValue.(string); ok {
					parts = append(parts, sourceStr)
				}
			} else {
				parts = append(parts, fieldStr)
			}
		}
	}
	
	return strings.Join(parts, separator), nil
}

func (f *FieldMappingService) splitTransform(sourceValue interface{}, rule map[string]interface{}) (interface{}, error) {
	sourceStr, ok := sourceValue.(string)
	if !ok {
		return nil, fmt.Errorf("source value must be string for split transform")
	}
	
	separator, _ := rule["separator"].(string)
	if separator == "" {
		separator = " "
	}
	
	index, _ := rule["index"].(float64)
	parts := strings.Split(sourceStr, separator)
	
	if int(index) < len(parts) {
		return parts[int(index)], nil
	}
	
	return "", nil
}

func (f *FieldMappingService) formatTransform(sourceValue interface{}, rule map[string]interface{}) (interface{}, error) {
	format, ok := rule["format"].(string)
	if !ok {
		return nil, fmt.Errorf("format rule must have format string")
	}
	
	// Simple format replacement - in production, you'd want more sophisticated formatting
	result := format
	if sourceStr, ok := sourceValue.(string); ok {
		result = strings.ReplaceAll(result, "{value}", sourceStr)
	}
	
	return result, nil
}

func (f *FieldMappingService) lookupTransform(sourceValue interface{}, rule map[string]interface{}) (interface{}, error) {
	lookupMap, ok := rule["lookup"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("lookup rule must have lookup map")
	}
	
	sourceStr, ok := sourceValue.(string)
	if !ok {
		return sourceValue, nil // Return original if not string
	}
	
	if mappedValue, exists := lookupMap[sourceStr]; exists {
		return mappedValue, nil
	}
	
	// Return default value if specified
	if defaultValue, hasDefault := rule["default"]; hasDefault {
		return defaultValue, nil
	}
	
	return sourceValue, nil
}

// GetFieldMappings returns all field mappings for a company
func (f *FieldMappingService) GetFieldMappings(companyID uint, sourceSystem, targetSystem string) ([]models.FieldMapping, error) {
	var mappings []models.FieldMapping
	result := config.DB.Where("company_id = ? AND source_system = ? AND target_system = ? AND is_active = ?",
		companyID, sourceSystem, targetSystem, true).Find(&mappings)
	
	if result.Error != nil {
		return nil, result.Error
	}
	
	return mappings, nil
}

// CreateFieldMapping creates a new field mapping
func (f *FieldMappingService) CreateFieldMapping(mapping *models.FieldMapping) error {
	return config.DB.Create(mapping).Error
}

// UpdateFieldMapping updates an existing field mapping
func (f *FieldMappingService) UpdateFieldMapping(mapping *models.FieldMapping) error {
	return config.DB.Save(mapping).Error
}

// DeleteFieldMapping deletes a field mapping
func (f *FieldMappingService) DeleteFieldMapping(id uint) error {
	return config.DB.Delete(&models.FieldMapping{}, id).Error
}

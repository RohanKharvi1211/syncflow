package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
)

// StagingTableService handles creating and managing staging tables for source data
type StagingTableService struct{}

// NewStagingTableService creates a new staging table service
func NewStagingTableService() *StagingTableService {
	return &StagingTableService{}
}

// CreateStagingTableIfNotExists creates a staging table with name {company_name}_{object_name}
// The table uses JSONB for flexible schema to store any data structure
func (sts *StagingTableService) CreateStagingTableIfNotExists(company *models.Company, dataObject *models.DataObject) (string, error) {
	// Extract object name from config (file_name or sheet_name) or use identifier as fallback
	objectName := sts.extractObjectName(dataObject)

	// Sanitize company name and object name for SQL table name
	tableName := sts.generateTableName(company.Name, objectName)

	// Create table with JSONB column for flexible data storage
	// Also include metadata columns for tracking
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			source_data JSONB NOT NULL,
			synced_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_%s_synced_at ON %s(synced_at);
		CREATE INDEX IF NOT EXISTS idx_%s_source_data ON %s USING GIN(source_data);
	`, tableName, tableName, tableName, tableName, tableName)

	if err := config.DB.Exec(createTableSQL).Error; err != nil {
		return "", fmt.Errorf("failed to create staging table %s: %v", tableName, err)
	}

	return tableName, nil
}

// extractObjectName extracts the object name from data object config or identifier
func (sts *StagingTableService) extractObjectName(dataObject *models.DataObject) string {
	// Try to get file_name or sheet_name from config
	if dataObject.Config != "" {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(dataObject.Config), &config); err == nil {
			if fileName, ok := config["file_name"].(string); ok && fileName != "" {
				return fileName
			}
			if sheetName, ok := config["sheet_name"].(string); ok && sheetName != "" {
				return sheetName
			}
		}
	}

	// Fallback to identifier (might be file ID for Google Sheets)
	return dataObject.Identifier
}

// InsertDataIntoStagingTable inserts fetched data into the staging table
func (sts *StagingTableService) InsertDataIntoStagingTable(tableName string, records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil // Nothing to insert
	}

	// Prepare batch insert
	values := make([]string, 0, len(records))
	args := make([]interface{}, 0, len(records))

	for i, record := range records {
		// Convert record to JSONB
		jsonData, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to marshal record %d: %v", i, err)
		}

		// Use parameterized query for safety (each record needs 1 parameter for JSONB)
		placeholder := fmt.Sprintf("($%d::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)", i+1)
		values = append(values, placeholder)
		args = append(args, string(jsonData))
	}

	// Build insert query
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (source_data, synced_at, created_at, updated_at)
		VALUES %s
	`, tableName, strings.Join(values, ", "))

	if err := config.DB.Exec(insertSQL, args...).Error; err != nil {
		return fmt.Errorf("failed to insert data into staging table %s: %v", tableName, err)
	}

	return nil
}

// generateTableName generates a safe SQL table name from company name and object identifier
func (sts *StagingTableService) generateTableName(companyName, objectIdentifier string) string {
	// Sanitize: remove special characters, convert to lowercase, replace spaces with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)

	sanitizedCompany := strings.ToLower(re.ReplaceAllString(companyName, "_"))
	sanitizedObject := strings.ToLower(re.ReplaceAllString(objectIdentifier, "_"))

	// Remove consecutive underscores
	reUnderscore := regexp.MustCompile(`_+`)
	sanitizedCompany = reUnderscore.ReplaceAllString(sanitizedCompany, "_")
	sanitizedObject = reUnderscore.ReplaceAllString(sanitizedObject, "_")

	// Trim underscores from start and end
	sanitizedCompany = strings.Trim(sanitizedCompany, "_")
	sanitizedObject = strings.Trim(sanitizedObject, "_")

	// Ensure table name doesn't start with a number
	if len(sanitizedCompany) > 0 && sanitizedCompany[0] >= '0' && sanitizedCompany[0] <= '9' {
		sanitizedCompany = "c_" + sanitizedCompany
	}
	if len(sanitizedObject) > 0 && sanitizedObject[0] >= '0' && sanitizedObject[0] <= '9' {
		sanitizedObject = "o_" + sanitizedObject
	}

	// Limit length (PostgreSQL table name limit is 63 characters)
	tableName := sanitizedCompany + "_" + sanitizedObject
	if len(tableName) > 63 {
		// Truncate if too long
		tableName = tableName[:63]
	}

	return tableName
}

// GetStagingTableData retrieves data from a staging table
func (sts *StagingTableService) GetStagingTableData(tableName string, limit int) ([]map[string]interface{}, error) {
	var results []struct {
		ID         string                 `json:"id"`
		SourceData map[string]interface{} `json:"source_data" gorm:"type:jsonb"`
		SyncedAt   string                 `json:"synced_at"`
	}

	query := fmt.Sprintf("SELECT id, source_data, synced_at FROM %s ORDER BY synced_at DESC", tableName)
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	if err := config.DB.Raw(query).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve data from staging table %s: %v", tableName, err)
	}

	records := make([]map[string]interface{}, len(results))
	for i, result := range results {
		records[i] = result.SourceData
		records[i]["_staging_id"] = result.ID
		records[i]["_synced_at"] = result.SyncedAt
	}

	return records, nil
}

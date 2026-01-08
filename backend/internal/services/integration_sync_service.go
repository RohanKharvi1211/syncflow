package services

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"time"

	"github.com/google/uuid"
)

type IntegrationSyncService struct {
	GoogleDriveService *GoogleDriveService
	QuickBooksService  *QuickBooksService
}

func NewIntegrationSyncService() *IntegrationSyncService {
	return &IntegrationSyncService{
		GoogleDriveService: NewGoogleDriveService(),
		QuickBooksService:  NewQuickBooksService(),
	}
}

// SyncIntegration performs a full sync for an integration
func (s *IntegrationSyncService) SyncIntegration(integrationID uuid.UUID, triggerType string) (*models.SyncJob, error) {
	// Get integration
	var integration models.Integration
	if err := config.DB.Preload("SourceConnection").Preload("SourceConnection.Metadata").
		Preload("SourceConnection.App").
		Preload("DestinationConnection").
		Preload("DestinationConnection.App").
		Preload("SourceApp").
		Preload("DestinationApp").
		First(&integration, integrationID).Error; err != nil {
		return nil, fmt.Errorf("integration not found: %v", err)
	}

	// Create sync job
	syncJob := models.SyncJob{
		ID:            uuid.New(),
		IntegrationID: integrationID,
		StartTime:     time.Now(),
		Status:        "processing",
		TriggerType:   triggerType,
	}

	if err := config.DB.Create(&syncJob).Error; err != nil {
		return nil, fmt.Errorf("failed to create sync job: %v", err)
	}

	// Authenticate with Google Drive
	if integration.SourceConnection == nil || integration.SourceAppID == uuid.Nil {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("source connection or app not found")
	}
	
	// Check if source app is Google-related
	sourceAppName := strings.ToLower(integration.SourceApp.Name)
	if sourceAppName == "" {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("source app not loaded")
	}
	if !strings.Contains(sourceAppName, "google") {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("source app is not Google-based")
	}

	s.GoogleDriveService.Authenticate(
		integration.SourceConnection.AccessToken,
		integration.SourceConnection.RefreshToken,
	)

	// Get source file ID and sheet name from metadata
	var sourceFileID, sourceSheetName string
	if integration.SourceConnection != nil && integration.SourceConnection.Metadata != nil {
		var metadataData map[string]interface{}
		if err := json.Unmarshal([]byte(integration.SourceConnection.Metadata.Data), &metadataData); err == nil {
			if fileID, ok := metadataData["file_id"].(string); ok {
				sourceFileID = fileID
			}
			if filename, ok := metadataData["filename"].(string); ok && sourceFileID == "" {
				// Fallback to filename if file_id not available
				sourceFileID = filename
			}
			if sheetname, ok := metadataData["sheetname"].(string); ok {
				sourceSheetName = sheetname
			}
		}
	}
	
	if sourceFileID == "" {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("source file ID not found in connection metadata")
	}

	// Check if file has been modified
	if integration.LastSyncedAt != nil {
		modified, err := s.GoogleDriveService.CheckFileModified(sourceFileID, *integration.LastSyncedAt)
		if err == nil && !modified {
			// No changes, mark job as completed
			syncJob.Status = "completed"
			syncJob.EndTime = &time.Time{}
			*syncJob.EndTime = time.Now()
			config.DB.Save(&syncJob)
			return &syncJob, nil
		}
	}

	// Read CSV from Google Drive
	rows, err := s.GoogleDriveService.ReadCSVFromDrive(sourceFileID, sourceSheetName)
	if err != nil {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("failed to read file from Google Drive: %v", err)
	}

	if len(rows) < 2 {
		// Only header row, no data
		syncJob.Status = "completed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return &syncJob, nil
	}

	// Parse header row
	headers := rows[0]
	
	// Parse field mapping
	var fieldMapping map[string]string
	if err := json.Unmarshal([]byte(integration.FieldMapping), &fieldMapping); err != nil {
		return nil, fmt.Errorf("invalid field mapping: %v", err)
	}

	// Authenticate with QuickBooks
	if integration.DestinationConnection == nil || integration.DestinationAppID == uuid.Nil {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("destination connection or app not found")
	}
	
	// Check if destination app is QuickBooks
	destAppName := strings.ToLower(integration.DestinationApp.Name)
	if destAppName == "" {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("destination app not loaded")
	}
	if !strings.Contains(destAppName, "quickbook") {
		syncJob.Status = "failed"
		syncJob.EndTime = &time.Time{}
		*syncJob.EndTime = time.Now()
		config.DB.Save(&syncJob)
		return nil, fmt.Errorf("destination app is not QuickBooks")
	}

	qbConfig := map[string]interface{}{
		"access_token":  integration.DestinationConnection.AccessToken,
		"refresh_token": integration.DestinationConnection.RefreshToken,
		"realm_id":      integration.DestinationConnection.RealmID,
	}
	s.QuickBooksService.Authenticate(qbConfig)

	// Process each row
	successCount := 0
	failedCount := 0

	for i, row := range rows[1:] {
		// Convert row to map
		rowData := make(map[string]interface{})
		for j, header := range headers {
			if j < len(row) {
				rowData[header] = row[j]
			}
		}

		// Get primary key value from field mapping or use first column
		primaryKeyValue := ""
		// Try to find a primary key field in the mapping or use first column
		if len(headers) > 0 {
			primaryKeyValue = fmt.Sprintf("%v", rowData[headers[0]])
		}

		// Calculate data hash
		dataHash := s.calculateDataHash(rowData)

		// Check if record already exists
		var existingRecord models.SyncRecord
		result := config.DB.Where("integration_id = ? AND source_record_unique_id = ?",
			integrationID, primaryKeyValue).First(&existingRecord)

		// Transform data using field mapping
		transformedData, err := s.transformData(rowData, fieldMapping, headers)
		if err != nil {
			failedCount++
			s.createSyncRecord(&syncJob, integrationID, primaryKeyValue, nil, "failed", err.Error(), rowData, dataHash)
			continue
		}

		// Check if data has changed (using hash)
		if result.Error == nil && existingRecord.DataHash == dataHash && existingRecord.Status == "synced" {
			// No changes, skip
			successCount++
			continue
		}

		// Create or update in QuickBooks
		var destinationID *string
		if result.Error == nil && existingRecord.DestinationRecordID != nil {
			// Update existing record
			destinationID = existingRecord.DestinationRecordID
			_, err = s.QuickBooksService.UpdateCustomer(*destinationID, transformedData)
		} else {
			// Create new record
			customer, err := s.QuickBooksService.CreateCustomer(transformedData)
			if err == nil && customer != nil {
				destinationID = &customer.ID
			}
		}

		if err != nil {
			failedCount++
			s.createSyncRecord(&syncJob, integrationID, primaryKeyValue, destinationID, "failed", err.Error(), rowData, dataHash)
		} else {
			successCount++
			s.createSyncRecord(&syncJob, integrationID, primaryKeyValue, destinationID, "synced", "", rowData, dataHash)
		}

		// Update progress
		syncJob.RecordsProcessed = i + 1
		syncJob.RecordsSuccess = successCount
		syncJob.RecordsFailed = failedCount
		config.DB.Save(&syncJob)
	}

	// Mark job as completed
	syncJob.Status = "completed"
	syncJob.EndTime = &time.Time{}
	*syncJob.EndTime = time.Now()
	config.DB.Save(&syncJob)

	// Update integration last synced time
	now := time.Now()
	integration.LastSyncedAt = &now
	config.DB.Save(&integration)

	return &syncJob, nil
}

// transformData transforms CSV row data to QuickBooks format using field mapping
func (s *IntegrationSyncService) transformData(rowData map[string]interface{}, fieldMapping map[string]string, headers []string) (map[string]interface{}, error) {
	transformed := make(map[string]interface{})

	for csvColumn, qbField := range fieldMapping {
		if value, exists := rowData[csvColumn]; exists {
			// Handle nested fields (e.g., "PrimaryEmailAddr.Address")
			s.setNestedField(transformed, qbField, value)
		}
	}

	return transformed, nil
}

// setNestedField sets a nested field in a map (e.g., "PrimaryEmailAddr.Address" -> {"PrimaryEmailAddr": {"Address": value}})
func (s *IntegrationSyncService) setNestedField(data map[string]interface{}, fieldPath string, value interface{}) {
	parts := splitFieldPath(fieldPath)
	if len(parts) == 1 {
		data[parts[0]] = value
		return
	}

	current := data
	for i, part := range parts[:len(parts)-1] {
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// Type mismatch, create new map
			current[part] = make(map[string]interface{})
			current = current[part].(map[string]interface{})
		}
		if i == len(parts)-2 {
			current[parts[len(parts)-1]] = value
		}
	}
}

func splitFieldPath(path string) []string {
	// Simple split on "." - in production, handle edge cases
	result := []string{}
	current := ""
	for _, char := range path {
		if char == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// createSyncRecord creates a sync record for tracking
func (s *IntegrationSyncService) createSyncRecord(job *models.SyncJob, integrationID uuid.UUID, sourceID string, destinationID *string, status, errorMsg string, rawData map[string]interface{}, dataHash string) {
	rawDataJSON, _ := json.Marshal(rawData)
	
	syncRecord := models.SyncRecord{
		ID:                   uuid.New(),
		JobID:                &job.ID,
		IntegrationID:        integrationID,
		SourceRecordUniqueID: sourceID,
		DestinationRecordID:  destinationID,
		Status:               status,
		DataHash:             dataHash,
		RawDataPayload:       string(rawDataJSON),
	}

	if errorMsg != "" {
		syncRecord.ErrorMessage = &errorMsg
	}

	config.DB.Create(&syncRecord)
}

// calculateDataHash calculates MD5 hash of row data
func (s *IntegrationSyncService) calculateDataHash(data map[string]interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:])
}

// RetrySyncRecord retries a failed sync record
func (s *IntegrationSyncService) RetrySyncRecord(recordID uuid.UUID) error {
	var record models.SyncRecord
	if err := config.DB.Preload("Integration").Preload("Integration.SourceConnection").Preload("Integration.DestinationConnection").First(&record, recordID).Error; err != nil {
		return fmt.Errorf("sync record not found: %v", err)
	}

	// Parse raw data payload
	var rowData map[string]interface{}
	if err := json.Unmarshal([]byte(record.RawDataPayload), &rowData); err != nil {
		return fmt.Errorf("invalid raw data payload: %v", err)
	}

	// Parse field mapping
	var fieldMapping map[string]string
	if err := json.Unmarshal([]byte(record.Integration.FieldMapping), &fieldMapping); err != nil {
		return fmt.Errorf("invalid field mapping: %v", err)
	}

	// Transform data
	headers := []string{} // We don't have headers in retry, but we can extract from rowData keys
	for key := range rowData {
		headers = append(headers, key)
	}

	transformedData, err := s.transformData(rowData, fieldMapping, headers)
	if err != nil {
		record.Status = "failed"
		errorMsg := err.Error()
		record.ErrorMessage = &errorMsg
		config.DB.Save(&record)
		return err
	}

	// Authenticate with QuickBooks
	qbConfig := map[string]interface{}{
		"access_token":  record.Integration.DestinationConnection.AccessToken,
		"refresh_token": record.Integration.DestinationConnection.RefreshToken,
		"realm_id":      record.Integration.DestinationConnection.RealmID,
	}
	s.QuickBooksService.Authenticate(qbConfig)

	// Create or update in QuickBooks
	var destinationID *string
	if record.DestinationRecordID != nil {
		// Update existing
		destinationID = record.DestinationRecordID
		_, err = s.QuickBooksService.UpdateCustomer(*destinationID, transformedData)
	} else {
		// Create new
		customer, err := s.QuickBooksService.CreateCustomer(transformedData)
		if err == nil && customer != nil {
			destinationID = &customer.ID
		}
	}

	if err != nil {
		record.Status = "failed"
		errorMsg := err.Error()
		record.ErrorMessage = &errorMsg
		config.DB.Save(&record)
		return err
	}

	// Update record
	record.Status = "synced"
	record.DestinationRecordID = destinationID
	record.ErrorMessage = nil
	record.UpdatedAt = time.Now()
	config.DB.Save(&record)

	return nil
}


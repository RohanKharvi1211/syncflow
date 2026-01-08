package services

import (
	"encoding/json"
	"fmt"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/utils"
	"time"

	"github.com/google/uuid"
)

// PipelineService handles pipeline execution with checkpoint-based reliable delivery
type PipelineService struct {
	// Services for different source/destination systems
	SalesforceService   *SalesforceService
	QuickBooksService   *QuickBooksService
	GoogleDriveService  *GoogleDriveService
	GoogleSheetsService *GoogleSheetsService
	FieldMappingService *FieldMappingService
	StagingTableService *StagingTableService
}

// NewPipelineService creates a new pipeline service
func NewPipelineService() *PipelineService {
	return &PipelineService{
		SalesforceService:   NewSalesforceService(),
		QuickBooksService:   NewQuickBooksService(),
		GoogleDriveService:  NewGoogleDriveService(),
		GoogleSheetsService: NewGoogleSheetsService(),
		FieldMappingService: NewFieldMappingService(),
		StagingTableService: NewStagingTableService(),
	}
}

// ExecutePipeline executes a single pipeline with checkpoint-based reliable delivery
// This implements the core polling algorithm from the design
func (ps *PipelineService) ExecutePipeline(pipelineID uuid.UUID) error {
	// Load pipeline with relations
	var pipeline models.Pipeline
	if err := config.DB.
		Preload("SourceObject.Connection.App").
		Preload("DestinationObject.Connection.App").
		Preload("Checkpoint").
		Preload("Company").
		First(&pipeline, pipelineID).Error; err != nil {
		return fmt.Errorf("pipeline not found: %v", err)
	}

	// Check if pipeline is active
	if pipeline.Status != "ACTIVE" {
		return fmt.Errorf("pipeline is not active (status: %s)", pipeline.Status)
	}

	// Create sync run for observability
	syncRun := models.SyncRun{
		ID:         uuid.New(),
		PipelineID: pipeline.ID,
		Status:     "processing",
		StartedAt:  time.Now(),
	}
	if err := config.DB.Create(&syncRun).Error; err != nil {
		return fmt.Errorf("failed to create sync run: %v", err)
	}

	// Load or create checkpoint
	checkpoint, err := ps.loadOrCreateCheckpoint(&pipeline)
	if err != nil {
		ps.updateSyncRunError(&syncRun, err)
		return err
	}

	// Fetch data since checkpoint
	records, err := ps.fetchDataSinceCheckpoint(&pipeline, checkpoint)
	if err != nil {
		ps.updateSyncRunError(&syncRun, err)
		return err
	}

	syncRun.RecordsRead = len(records)

	// Store fetched data in staging table {company_name}_{object_name}
	recordsWritten := 0
	var stagingError error
	if len(records) > 0 {
		// Company is already loaded via Preload
		if pipeline.Company.ID != uuid.Nil {
			// Create staging table if not exists and insert data
			tableName, err := ps.StagingTableService.CreateStagingTableIfNotExists(&pipeline.Company, &pipeline.SourceObject)
			if err != nil {
				stagingError = fmt.Errorf("failed to create staging table: %v", err)
				fmt.Printf("Error: %v\n", stagingError)
			} else {
				// Insert fetched data into staging table
				if err := ps.StagingTableService.InsertDataIntoStagingTable(tableName, records); err != nil {
					stagingError = fmt.Errorf("failed to insert data into staging table %s: %v", tableName, err)
					fmt.Printf("Error: %v\n", stagingError)
				} else {
					recordsWritten = len(records)
					fmt.Printf("Successfully inserted %d records into staging table %s\n", len(records), tableName)
				}
			}
		} else {
			stagingError = fmt.Errorf("company not found for pipeline")
			fmt.Printf("Error: %v\n", stagingError)
		}
	}

	// Update checkpoint (to track what we've fetched, even if staging failed)
	if err := ps.updateCheckpoint(&pipeline, checkpoint, records); err != nil {
		ps.updateSyncRunError(&syncRun, err)
		return err
	}

	// Update sync run status
	syncRun.RecordsWritten = recordsWritten
	now := time.Now()
	syncRun.FinishedAt = &now

	if stagingError != nil {
		syncRun.Status = "FAILED"
		errMsg := stagingError.Error()
		syncRun.Error = &errMsg
	} else if len(records) == 0 {
		syncRun.Status = "SUCCESS"
		syncRun.Error = nil
	} else {
		syncRun.Status = "SUCCESS"
		syncRun.Error = nil
	}

	if err := config.DB.Save(&syncRun).Error; err != nil {
		return fmt.Errorf("failed to update sync run: %v", err)
	}

	return nil
}

// loadOrCreateCheckpoint loads existing checkpoint or creates a new one
func (ps *PipelineService) loadOrCreateCheckpoint(pipeline *models.Pipeline) (*models.Checkpoint, error) {
	var checkpoint models.Checkpoint

	if pipeline.Checkpoint != nil {
		// Load existing checkpoint
		if err := config.DB.First(&checkpoint, "pipeline_id = ?", pipeline.ID).Error; err == nil {
			return &checkpoint, nil
		}
	}

	// Create new checkpoint with default values
	checkpoint = models.Checkpoint{
		PipelineID:   pipeline.ID,
		LastSyncTime: time.Now().Add(-24 * time.Hour), // Default to 24 hours ago
		LastCursor:   "",
		UpdatedAt:    time.Now(),
	}

	if err := config.DB.Create(&checkpoint).Error; err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %v", err)
	}

	return &checkpoint, nil
}

// fetchDataSinceCheckpoint fetches data from source since the last checkpoint
func (ps *PipelineService) fetchDataSinceCheckpoint(pipeline *models.Pipeline, checkpoint *models.Checkpoint) ([]map[string]interface{}, error) {
	sourceObject := pipeline.SourceObject
	connection := sourceObject.Connection
	app := connection.App

	// Refresh token if needed
	if err := ps.refreshTokenIfNeeded(&connection); err != nil {
		return nil, fmt.Errorf("token refresh failed: %v", err)
	}

	// Route to appropriate service based on app type
	switch app.Name {
	case "salesforce":
		return ps.fetchFromSalesforce(&sourceObject, checkpoint)
	case "googlesheet", "googledrive":
		return ps.fetchFromGoogleSheet(&sourceObject, checkpoint)
	case "quickbooks":
		return ps.fetchFromQuickBooks(&sourceObject, checkpoint)
	default:
		return nil, fmt.Errorf("unsupported source app: %s", app.Name)
	}
}

// fetchFromSalesforce fetches records from Salesforce since checkpoint
func (ps *PipelineService) fetchFromSalesforce(dataObject *models.DataObject, checkpoint *models.Checkpoint) ([]map[string]interface{}, error) {
	// Set up Salesforce service with connection tokens
	connection := dataObject.Connection
	
	// Parse metadata for instance URL
	var metadata models.Metadata
	if err := config.DB.First(&metadata, "connection_id = ?", connection.ID).Error; err != nil {
		return nil, fmt.Errorf("metadata not found: %v", err)
	}

	var metadataData map[string]interface{}
	if err := json.Unmarshal([]byte(metadata.Data), &metadataData); err != nil {
		return nil, fmt.Errorf("invalid metadata: %v", err)
	}

	// Set up Salesforce service
	ps.SalesforceService.InstanceURL = metadataData["instance_url"].(string)
	ps.SalesforceService.AccessToken = connection.AccessToken

	// Use GetUpdatedRecords which handles incremental sync
	records, err := ps.SalesforceService.GetUpdatedRecords(dataObject.Identifier, checkpoint.LastSyncTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Salesforce records: %v", err)
	}

	return records, nil
}

// fetchFromGoogleSheet fetches rows from Google Sheet since checkpoint
func (ps *PipelineService) fetchFromGoogleSheet(dataObject *models.DataObject, checkpoint *models.Checkpoint) ([]map[string]interface{}, error) {
	// Set up Google Sheets service with connection tokens
	connection := dataObject.Connection
	ps.GoogleSheetsService.Authenticate(connection.AccessToken, connection.RefreshToken)

	// Parse config
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(dataObject.Config), &config); err != nil {
		config = make(map[string]interface{})
	}

	// Get spreadsheet ID from identifier or config
	spreadsheetID := dataObject.Identifier
	if fileIDFromConfig, ok := config["file_id"].(string); ok {
		spreadsheetID = fileIDFromConfig
	}

	sheetName := "Sheet1" // default
	if sheetNameFromConfig, ok := config["sheet_name"].(string); ok {
		sheetName = sheetNameFromConfig
	}

	// Use last row number as cursor for Google Sheets
	lastRow := 0
	if checkpoint.LastCursor != "" {
		fmt.Sscanf(checkpoint.LastCursor, "%d", &lastRow)
	}

	// Read data from Google Sheets using Sheets API
	rows, err := ps.GoogleSheetsService.GetSheetData(spreadsheetID, sheetName, lastRow)
	if err != nil {
		return nil, fmt.Errorf("failed to read from Google Sheets: %v", err)
	}

	// Convert rows to map format (skip header and rows before lastRow)
	if len(rows) == 0 {
		return []map[string]interface{}{}, nil // Empty sheet
	}

	if len(rows) <= 1 {
		return []map[string]interface{}{}, nil // Only header or empty
	}

	header := rows[0]
	records := make([]map[string]interface{}, 0)

	// Start from lastRow + 1 (skip header at index 0)
	startIndex := lastRow + 1
	if startIndex >= len(rows) {
		return []map[string]interface{}{}, nil // No new rows
	}

	for i := startIndex; i < len(rows); i++ {
		record := make(map[string]interface{})
		for j, col := range header {
			if j < len(rows[i]) {
				record[col] = rows[i][j]
			} else {
				record[col] = ""
			}
		}
		records = append(records, record)
	}

	return records, nil
}

// fetchFromQuickBooks fetches records from QuickBooks since checkpoint
func (ps *PipelineService) fetchFromQuickBooks(dataObject *models.DataObject, checkpoint *models.Checkpoint) ([]map[string]interface{}, error) {
	// Use changedSince for QuickBooks incremental sync
	// This is a placeholder - you'd integrate with your actual QuickBooks service
	return nil, fmt.Errorf("QuickBooks fetch not yet implemented")
}

// processRecord transforms and writes a single record
func (ps *PipelineService) processRecord(pipeline *models.Pipeline, record map[string]interface{}, checkpoint *models.Checkpoint) error {
	// Parse field mapping
	var fieldMapping map[string]interface{}
	if err := json.Unmarshal([]byte(pipeline.FieldMapping), &fieldMapping); err != nil {
		fieldMapping = make(map[string]interface{})
	}

	// Transform data using field mapping
	transformedData, err := ps.transformData(record, fieldMapping)
	if err != nil {
		return fmt.Errorf("transformation failed: %v", err)
	}

	// Write to destination
	destObject := pipeline.DestinationObject
	destConnection := destObject.Connection
	destApp := destObject.Connection.App

	// Refresh token if needed
	if err := ps.refreshTokenIfNeeded(&destConnection); err != nil {
		return fmt.Errorf("destination token refresh failed: %v", err)
	}

	// Route to appropriate destination service
	switch destApp.Name {
	case "quickbooks":
		return ps.writeToQuickBooks(&destObject, transformedData)
	case "googlesheet", "googledrive":
		return ps.writeToGoogleSheet(&destObject, transformedData)
	case "salesforce":
		return ps.writeToSalesforce(&destObject, transformedData)
	default:
		return fmt.Errorf("unsupported destination app: %s", destApp.Name)
	}
}

// transformData applies field mapping transformations
func (ps *PipelineService) transformData(record map[string]interface{}, fieldMapping map[string]interface{}) (map[string]interface{}, error) {
	transformed := make(map[string]interface{})

	for targetField, mapping := range fieldMapping {
		mappingMap, ok := mapping.(map[string]interface{})
		if !ok {
			// Direct mapping
			if sourceField, ok := mapping.(string); ok {
				if val, exists := record[sourceField]; exists {
					transformed[targetField] = val
				}
			}
			continue
		}

		// Handle different mapping types
		mappingType, _ := mappingMap["type"].(string)
		switch mappingType {
		case "direct":
			if sourceField, ok := mappingMap["source_field"].(string); ok {
				if val, exists := record[sourceField]; exists {
					transformed[targetField] = val
				}
			}
		case "transform":
			// Apply transformation rules
			// Parse transform rule
			if transformRule, ok := mappingMap["transform_rule"].(map[string]interface{}); ok {
				if val, err := ps.applyTransformRule(record, transformRule, mappingMap); err == nil {
					transformed[targetField] = val
				}
			}
		case "constant":
			if val, ok := mappingMap["value"]; ok {
				transformed[targetField] = val
			}
		}
	}

	return transformed, nil
}

// applyTransformRule applies a transformation rule to a field value
func (ps *PipelineService) applyTransformRule(record map[string]interface{}, transformRule map[string]interface{}, mappingMap map[string]interface{}) (interface{}, error) {
	// Get source field
	sourceField, ok := mappingMap["source_field"].(string)
	if !ok {
		return nil, fmt.Errorf("source_field not found in mapping")
	}

	// Get source value
	sourceValue, exists := record[sourceField]
	if !exists {
		return nil, fmt.Errorf("source field %s not found in record", sourceField)
	}

	// Get transform type
	ruleType, ok := transformRule["type"].(string)
	if !ok {
		return nil, fmt.Errorf("transform rule type not found")
	}

	switch ruleType {
	case "concat":
		// Concatenate values
		fields, ok := transformRule["fields"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("concat fields not found")
		}
		separator, _ := transformRule["separator"].(string)
		if separator == "" {
			separator = " "
		}
		result := ""
		for i, field := range fields {
			if i > 0 {
				result += separator
			}
			if fieldStr, ok := field.(string); ok {
				if fieldStr == "source" {
					result += fmt.Sprintf("%v", sourceValue)
				} else {
					result += fieldStr
				}
			}
		}
		return result, nil
	case "lookup":
		// Lookup mapping
		lookup, ok := transformRule["lookup"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("lookup map not found")
		}
		if val, exists := lookup[fmt.Sprintf("%v", sourceValue)]; exists {
			return val, nil
		}
		// Return default if provided
		if defaultVal, ok := transformRule["default"]; ok {
			return defaultVal, nil
		}
		return sourceValue, nil
	default:
		return sourceValue, nil // Return original value if transform type not recognized
	}
}

// writeToQuickBooks writes data to QuickBooks
func (ps *PipelineService) writeToQuickBooks(dataObject *models.DataObject, data map[string]interface{}) error {
	// This is a placeholder - integrate with actual QuickBooks service
	return fmt.Errorf("QuickBooks write not yet implemented")
}

// writeToGoogleSheet writes data to Google Sheet
func (ps *PipelineService) writeToGoogleSheet(dataObject *models.DataObject, data map[string]interface{}) error {
	// This is a placeholder - integrate with actual Google Sheets service
	return fmt.Errorf("Google Sheets write not yet implemented")
}

// writeToSalesforce writes data to Salesforce
func (ps *PipelineService) writeToSalesforce(dataObject *models.DataObject, data map[string]interface{}) error {
	// This is a placeholder - integrate with actual Salesforce service
	return fmt.Errorf("Salesforce write not yet implemented")
}

// updateCheckpoint updates the checkpoint after successful sync
func (ps *PipelineService) updateCheckpoint(pipeline *models.Pipeline, checkpoint *models.Checkpoint, records []map[string]interface{}) error {
	checkpoint.LastSyncTime = time.Now()

	// Update cursor based on source type
	sourceApp := pipeline.SourceObject.Connection.App
	switch sourceApp.Name {
	case "googlesheet", "googledrive":
		// For Google Sheets, cursor is the last row number
		if len(records) > 0 {
			// Assuming records have a row number or we track it
			checkpoint.LastCursor = fmt.Sprintf("%d", len(records))
		}
	case "salesforce":
		// For Salesforce, cursor is the last SystemModstamp
		if len(records) > 0 {
			if lastRecord := records[len(records)-1]; lastRecord != nil {
				if modstamp, ok := lastRecord["SystemModstamp"].(string); ok {
					checkpoint.LastCursor = modstamp
				}
			}
		}
	}

	checkpoint.UpdatedAt = time.Now()

	if err := config.DB.Save(checkpoint).Error; err != nil {
		return fmt.Errorf("failed to update checkpoint: %v", err)
	}

	return nil
}

// refreshTokenIfNeeded refreshes OAuth token if expired
// This is a wrapper around utils.RefreshTokenIfNeeded for pipeline service
func (ps *PipelineService) refreshTokenIfNeeded(connection *models.Connection) error {
	return utils.RefreshTokenIfNeeded(connection)
}

// updateSyncRunError updates sync run with error
func (ps *PipelineService) updateSyncRunError(syncRun *models.SyncRun, err error) {
	now := time.Now()
	syncRun.FinishedAt = &now
	syncRun.Status = "FAILED"
	errMsg := err.Error()
	syncRun.Error = &errMsg
	config.DB.Save(syncRun)
}

// PollActivePipelines polls all active pipelines (called by scheduler)
func (ps *PipelineService) PollActivePipelines() error {
	var pipelines []models.Pipeline

	// Find all active pipelines
	if err := config.DB.Where("status = ?", "ACTIVE").Find(&pipelines).Error; err != nil {
		return fmt.Errorf("failed to fetch active pipelines: %v", err)
	}

	// Execute each pipeline
	for _, pipeline := range pipelines {
		// Check if it's time to sync based on schedule_interval
		if !ps.shouldSync(&pipeline) {
			continue
		}

		// Execute pipeline (in goroutine for parallel execution)
		go func(p models.Pipeline) {
			if err := ps.ExecutePipeline(p.ID); err != nil {
				// Log error but continue with other pipelines
				fmt.Printf("Pipeline %s execution failed: %v\n", p.ID, err)
			}
		}(pipeline)
	}

	return nil
}

// shouldSync checks if pipeline should be synced based on schedule_interval
func (ps *PipelineService) shouldSync(pipeline *models.Pipeline) bool {
	// Load checkpoint to check last sync time
	var checkpoint models.Checkpoint
	if err := config.DB.First(&checkpoint, "pipeline_id = ?", pipeline.ID).Error; err != nil {
		// No checkpoint exists, should sync
		return true
	}

	// Check if enough time has passed
	nextSyncTime := checkpoint.LastSyncTime.Add(time.Duration(pipeline.ScheduleInterval) * time.Minute)
	return time.Now().After(nextSyncTime)
}

// GetPipelineStatus returns the status of a pipeline with latest sync run
func (ps *PipelineService) GetPipelineStatus(pipelineID uuid.UUID) (map[string]interface{}, error) {
	var pipeline models.Pipeline
	if err := config.DB.
		Preload("SourceObject.Connection.App").
		Preload("DestinationObject.Connection.App").
		Preload("Checkpoint").
		First(&pipeline, pipelineID).Error; err != nil {
		return nil, err
	}

	// Get latest sync run
	var latestSyncRun models.SyncRun
	config.DB.Where("pipeline_id = ?", pipelineID).
		Order("started_at DESC").
		First(&latestSyncRun)

	status := map[string]interface{}{
		"pipeline_id":      pipeline.ID,
		"status":           pipeline.Status,
		"schedule_interval": pipeline.ScheduleInterval,
		"last_sync_time":   nil,
		"last_sync_status": nil,
		"records_read":    0,
		"records_written":  0,
	}

	if pipeline.Checkpoint != nil {
		status["last_sync_time"] = pipeline.Checkpoint.LastSyncTime
		status["last_cursor"] = pipeline.Checkpoint.LastCursor
	}

	if latestSyncRun.ID != uuid.Nil {
		status["last_sync_status"] = latestSyncRun.Status
		status["records_read"] = latestSyncRun.RecordsRead
		status["records_written"] = latestSyncRun.RecordsWritten
		if latestSyncRun.Error != nil {
			status["last_error"] = *latestSyncRun.Error
		}
	}

	return status, nil
}


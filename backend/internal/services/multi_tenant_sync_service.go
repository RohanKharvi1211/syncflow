package services

import (
	"encoding/json"
	"fmt"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"time"
)

type MultiTenantSyncService struct {
	FieldMappingService *FieldMappingService
	SalesforceService   *SalesforceService
	QuickBooksService   *QuickBooksService
}

func NewMultiTenantSyncService() *MultiTenantSyncService {
	return &MultiTenantSyncService{
		FieldMappingService: NewFieldMappingService(),
		SalesforceService:   NewSalesforceService(),
		QuickBooksService:   NewQuickBooksService(),
	}
}

// SyncRecordsForCompany syncs records for a specific company
func (m *MultiTenantSyncService) SyncRecordsForCompany(companyID uint, recordType string) error {
	// Get company information
	var company models.Company
	if err := config.DB.Preload("Integrations").First(&company, companyID).Error; err != nil {
		return fmt.Errorf("company not found: %v", err)
	}

	// Find Salesforce integration (using LegacyIntegration for now)
	var salesforceIntegration *models.LegacyIntegration
	for _, integration := range company.LegacyIntegrations {
		if integration.Type == "salesforce" && integration.IsActive {
			salesforceIntegration = &integration
			break
		}
	}

	if salesforceIntegration == nil {
		return fmt.Errorf("no active Salesforce integration found for company %d", companyID)
	}

	// Get updated records from Salesforce
	records, err := m.getSalesforceRecords(salesforceIntegration, recordType, companyID)
	if err != nil {
		return fmt.Errorf("failed to get Salesforce records: %v", err)
	}

	// Process each record
	for _, record := range records {
		if err := m.processRecordForCompany(companyID, recordType, record); err != nil {
			fmt.Printf("Error processing record %s for company %d: %v\n", record["Id"], companyID, err)
			continue
		}
	}

	return nil
}

func (m *MultiTenantSyncService) getSalesforceRecords(integration *models.LegacyIntegration, recordType string, companyID uint) ([]map[string]interface{}, error) {
	// Parse Salesforce configuration
	var sfConfig map[string]interface{}
	if err := json.Unmarshal([]byte(integration.Config), &sfConfig); err != nil {
		return nil, fmt.Errorf("invalid Salesforce configuration: %v", err)
	}

	// Set up Salesforce service with company-specific config
	m.SalesforceService.InstanceURL = sfConfig["instance_url"].(string)

	// Authenticate with Salesforce
	if err := m.SalesforceService.Authenticate(sfConfig); err != nil {
		return nil, fmt.Errorf("Salesforce authentication failed: %v", err)
	}

	// Get last sync time for this company and record type
	lastSyncTime := m.getLastSyncTime(companyID, recordType)

	// Get updated records
	records, err := m.SalesforceService.GetUpdatedRecords(recordType, lastSyncTime)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (m *MultiTenantSyncService) processRecordForCompany(companyID uint, recordType string, record map[string]interface{}) error {
	recordID := record["Id"].(string)

	// Check if record already exists in sync tracking
	var syncRecord models.LegacySyncRecord
	result := config.DB.Where("company_id = ? AND source_record_id = ? AND record_type = ?",
		companyID, recordID, recordType).First(&syncRecord)

	// Get full record details from Salesforce
	fullRecord, err := m.SalesforceService.GetRecordDetails(recordType, recordID)
	if err != nil {
		return fmt.Errorf("failed to get record details: %v", err)
	}

	sourceData, _ := json.Marshal(fullRecord)

	if result.Error != nil {
		// Create new sync record
		syncRecord = models.LegacySyncRecord{
			CompanyID:      companyID,
			SourceRecordID: recordID,
			SourceSystem:   "salesforce",
			RecordType:     recordType,
			Status:         "pending",
			SourceData:     string(sourceData),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := config.DB.Create(&syncRecord).Error; err != nil {
			return fmt.Errorf("failed to create sync record: %v", err)
		}
	} else {
		// Update existing sync record
		syncRecord.SourceData = string(sourceData)
		syncRecord.Status = "pending"
		syncRecord.UpdatedAt = time.Now()

		if err := config.DB.Save(&syncRecord).Error; err != nil {
			return fmt.Errorf("failed to update sync record: %v", err)
		}
	}

	// Sync to all target systems for this company
	return m.syncToTargetSystems(companyID, &syncRecord)
}

func (m *MultiTenantSyncService) syncToTargetSystems(companyID uint, syncRecord *models.LegacySyncRecord) error {
	// Get company's integrations
	var company models.Company
	if err := config.DB.Preload("LegacyIntegrations").First(&company, companyID).Error; err != nil {
		return err
	}

	// Find target integrations (QuickBooks, Tally, etc.)
	var targetIntegrations []models.LegacyIntegration
	for _, integration := range company.LegacyIntegrations {
		if integration.Type != "salesforce" && integration.IsActive {
			targetIntegrations = append(targetIntegrations, integration)
		}
	}

	// Sync to each target system
	for _, integration := range targetIntegrations {
		if err := m.syncToTargetSystem(companyID, syncRecord, integration); err != nil {
			fmt.Printf("Error syncing to %s: %v\n", integration.Type, err)
			continue
		}
	}

	return nil
}

func (m *MultiTenantSyncService) syncToTargetSystem(companyID uint, syncRecord *models.LegacySyncRecord, integration models.LegacyIntegration) error {
	// Parse source data
	var sourceData map[string]interface{}
	if err := json.Unmarshal([]byte(syncRecord.SourceData), &sourceData); err != nil {
		return err
	}

	// Determine target object type based on source record type
	targetObject := m.getTargetObjectType(syncRecord.RecordType, integration.Type)

	// Transform data using field mappings
	targetData, err := m.FieldMappingService.TransformData(
		companyID,
		"salesforce",
		integration.Type,
		syncRecord.RecordType,
		targetObject,
		sourceData,
	)
	if err != nil {
		return fmt.Errorf("field mapping failed: %v", err)
	}

	// Create or update record in target system
	var targetRecordID string
	switch integration.Type {
	case "quickbooks":
		targetRecordID, err = m.syncToQuickBooks(companyID, targetObject, targetData, syncRecord)
	case "tally":
		targetRecordID, err = m.syncToTally(companyID, targetObject, targetData, syncRecord)
	default:
		return fmt.Errorf("unsupported target system: %s", integration.Type)
	}

	if err != nil {
		return err
	}

	// Update sync record
	syncRecord.TargetRecordID = &targetRecordID
	syncRecord.TargetSystem = integration.Type
	syncRecord.Status = "synced"
	syncRecord.LastSuccessfulSync = &time.Time{}
	*syncRecord.LastSuccessfulSync = time.Now()

	targetDataJSON, _ := json.Marshal(targetData)
	syncRecord.TargetData = string(targetDataJSON)

	return config.DB.Save(syncRecord).Error
}

func (m *MultiTenantSyncService) getTargetObjectType(sourceObject, targetSystem string) string {
	// Map Salesforce objects to target system objects
	mapping := map[string]map[string]string{
		"Account": {
			"quickbooks": "Customer",
			"tally":      "Customer",
		},
		"Contact": {
			"quickbooks": "Customer",
			"tally":      "Customer",
		},
		"Opportunity": {
			"quickbooks": "Invoice",
			"tally":      "Voucher",
		},
	}

	if systemMapping, exists := mapping[sourceObject]; exists {
		if targetObject, exists := systemMapping[targetSystem]; exists {
			return targetObject
		}
	}

	return sourceObject // Default to same name
}

func (m *MultiTenantSyncService) syncToQuickBooks(companyID uint, targetObject string, targetData map[string]interface{}, syncRecord *models.LegacySyncRecord) (string, error) {
	// Get QuickBooks integration for this company
	var integration models.LegacyIntegration
	if err := config.DB.Where("company_id = ? AND type = ? AND is_active = ?",
		companyID, "quickbooks", true).First(&integration).Error; err != nil {
		return "", err
	}

	// Parse QuickBooks configuration
	var qbConfig map[string]interface{}
	if err := json.Unmarshal([]byte(integration.Config), &qbConfig); err != nil {
		return "", err
	}

	// Set up QuickBooks service with company-specific config
	// (Implementation would set access tokens, etc.)

	switch targetObject {
	case "Customer":
		customer := models.QuickBooksCustomer{}
		// Map targetData to QuickBooksCustomer struct
		customerJSON, _ := json.Marshal(targetData)
		json.Unmarshal(customerJSON, &customer)

		if syncRecord.TargetRecordID != nil {
			customer.ID = *syncRecord.TargetRecordID
			updatedCustomer, err := m.QuickBooksService.UpdateCustomer(*syncRecord.TargetRecordID, targetData)
			if err != nil {
				return "", err
			}
			return updatedCustomer.ID, nil
		} else {
			createdCustomer, err := m.QuickBooksService.CreateCustomer(targetData)
			if err != nil {
				return "", err
			}
			return createdCustomer.ID, nil
		}
	default:
		return "", fmt.Errorf("unsupported QuickBooks object type: %s", targetObject)
	}
}

func (m *MultiTenantSyncService) syncToTally(companyID uint, targetObject string, targetData map[string]interface{}, syncRecord *models.LegacySyncRecord) (string, error) {
	// Get Tally service for this company
	tallyService, err := NewTallyService(companyID)
	if err != nil {
		return "", err
	}

	switch targetObject {
	case "Customer":
		if syncRecord.TargetRecordID != nil {
			updatedCustomer, err := tallyService.UpdateCustomer(*syncRecord.TargetRecordID, targetData)
			if err != nil {
				return "", err
			}
			return updatedCustomer.ID, nil
		} else {
			createdCustomer, err := tallyService.CreateCustomer(targetData)
			if err != nil {
				return "", err
			}
			return createdCustomer.ID, nil
		}
	case "Voucher":
		createdVoucher, err := tallyService.CreateVoucher(targetData)
		if err != nil {
			return "", err
		}
		return createdVoucher.ID, nil
	default:
		return "", fmt.Errorf("unsupported Tally object type: %s", targetObject)
	}
}

func (m *MultiTenantSyncService) getLastSyncTime(companyID uint, recordType string) time.Time {
	var lastSync models.LegacySyncRecord
	result := config.DB.Where("company_id = ? AND record_type = ? AND status = ?",
		companyID, recordType, "synced").Order("last_successful_sync DESC").First(&lastSync)

	if result.Error != nil {
		return time.Now().Add(-24 * time.Hour)
	}

	if lastSync.LastSuccessfulSync != nil {
		return *lastSync.LastSuccessfulSync
	}

	return time.Now().Add(-24 * time.Hour)
}

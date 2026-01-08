package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Company represents a company/tenant in the system
type Company struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name      string         `json:"name" gorm:"not null"`
	Domain    string         `json:"domain" gorm:"uniqueIndex"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	Users        []User        `json:"users" gorm:"foreignKey:CompanyID"`
	Connections  []Connection  `json:"connections" gorm:"foreignKey:CompanyID"`
	Integrations []Integration  `json:"integrations" gorm:"foreignKey:CompanyID"`
	Pipelines    []Pipeline    `json:"pipelines" gorm:"foreignKey:CompanyID"`
	
	// Legacy relations (for backward compatibility during migration)
	LegacyIntegrations  []LegacyIntegration  `json:"legacy_integrations" gorm:"foreignKey:CompanyID"`
	FieldMappings       []FieldMapping       `json:"field_mappings" gorm:"foreignKey:CompanyID"`
	LegacySyncRecords   []LegacySyncRecord   `json:"legacy_sync_records" gorm:"foreignKey:CompanyID"`
}

// LegacyIntegration - Old model kept for backward compatibility
type LegacyIntegration struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CompanyID uint      `json:"company_id" gorm:"not null"`
	Company   Company   `json:"company" gorm:"foreignKey:CompanyID"`
	Type      string    `json:"type"` // "salesforce", "quickbooks", "tally"
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	Config    string    `json:"config"` // JSON string containing integration-specific config
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FieldMapping struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	CompanyID     uint      `json:"company_id" gorm:"not null"`
	Company       Company   `json:"company" gorm:"foreignKey:CompanyID"`
	SourceSystem  string    `json:"source_system"` // "salesforce"
	TargetSystem  string    `json:"target_system"` // "quickbooks", "tally"
	SourceObject  string    `json:"source_object"` // "Account", "Contact", "Opportunity"
	TargetObject  string    `json:"target_object"` // "Customer", "Item", "Invoice"
	SourceField   string    `json:"source_field"`
	TargetField   string    `json:"target_field"`
	MappingType   string    `json:"mapping_type"`   // "direct", "transform", "constant"
	TransformRule string    `json:"transform_rule"` // JSON string for transformation logic
	IsRequired    bool      `json:"is_required" gorm:"default:false"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LegacySyncRecord - Old model kept for backward compatibility
type LegacySyncRecord struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	CompanyID          uint           `json:"company_id" gorm:"not null"`
	Company            Company        `json:"company" gorm:"foreignKey:CompanyID"`
	SourceRecordID     string         `json:"source_record_id" gorm:"not null"`
	TargetRecordID     *string        `json:"target_record_id"`
	SourceSystem       string         `json:"source_system"`
	TargetSystem       string         `json:"target_system"`
	RecordType         string         `json:"record_type"`
	Status             string         `json:"status"` // "pending", "synced", "failed", "retrying"
	ErrorMessage       *string        `json:"error_message"`
	LastSyncAttempt    *time.Time     `json:"last_sync_attempt"`
	LastSuccessfulSync *time.Time     `json:"last_successful_sync"`
	RetryCount         int            `json:"retry_count" gorm:"default:0"`
	SourceData         string         `json:"source_data"` // JSON string of the source record data
	TargetData         string         `json:"target_data"` // JSON string of the target record data
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type SyncLog struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	CompanyID    uint       `json:"company_id" gorm:"not null"`
	Company      Company    `json:"company" gorm:"foreignKey:CompanyID"`
	SyncRecordID uint            `json:"sync_record_id"`
	SyncRecord   LegacySyncRecord `json:"sync_record" gorm:"foreignKey:SyncRecordID"`
	Action       string     `json:"action"` // "create", "update", "delete", "sync"
	Status       string     `json:"status"` // "success", "error"
	Message      string     `json:"message"`
	CreatedAt    time.Time  `json:"created_at"`
}

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user belonging to a company
type User struct {
	ID                   uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	CompanyID            uuid.UUID      `json:"company_id" gorm:"type:uuid;not null;index"`
	Company              Company        `json:"company" gorm:"foreignKey:CompanyID"`
	Email                string         `json:"email" gorm:"not null"`
	PasswordHash         string         `json:"-" gorm:"column:password_hash"` // Hidden from JSON
	FirstName            string         `json:"first_name" gorm:"column:first_name"`
	LastName             string         `json:"last_name" gorm:"column:last_name"`
	Role                 string         `json:"role" gorm:"default:user"` // 'admin' or 'user'
	IsActive             bool           `json:"is_active" gorm:"default:true"`
	InvitationToken      string         `json:"-" gorm:"column:invitation_token;uniqueIndex"`
	InvitationSentAt     *time.Time     `json:"invitation_sent_at" gorm:"column:invitation_sent_at"`
	InvitationAcceptedAt *time.Time     `json:"invitation_accepted_at" gorm:"column:invitation_accepted_at"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Unique constraint on company_id + email is handled by database constraint "unique_company_email"
}

// App represents an available application (Salesforce, GoogleSheet, etc.)
type App struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name           string         `json:"name" gorm:"uniqueIndex;not null"` // e.g., "salesforce", "googlesheet"
	DisplayName    string         `json:"display_name" gorm:"not null"`     // e.g., "Salesforce", "Google Sheets"
	Description    string         `json:"description"`
	Type           string         `json:"type" gorm:"not null"`                                    // 'source', 'destination', or 'both'
	MetadataSchema string         `json:"metadata_schema" gorm:"type:jsonb;not null;default:'{}'"` // Schema defining required metadata fields
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	Connections             []Connection  `json:"connections" gorm:"foreignKey:AppID"`
	SourceIntegrations      []Integration `json:"source_integrations" gorm:"foreignKey:SourceAppID"`
	DestinationIntegrations []Integration `json:"destination_integrations" gorm:"foreignKey:DestinationAppID"`
}

// Connection represents OAuth credentials for a company and app
// One row per source OR destination account
type Connection struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID         uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User           User           `json:"user" gorm:"foreignKey:UserID"`
	CompanyID      uuid.UUID      `json:"company_id" gorm:"type:uuid;not null;index"`
	Company        Company        `json:"company" gorm:"foreignKey:CompanyID"`
	AppID          uuid.UUID      `json:"app_id" gorm:"type:uuid;not null;index"`
	App            App            `json:"app" gorm:"foreignKey:AppID"`
	Provider       string         `json:"provider" gorm:"type:varchar(50);not null"`          // 'google', 'salesforce', 'quickbooks', etc.
	Type           string         `json:"type" gorm:"type:varchar(50)"`                       // 'GOOGLE_SHEET', 'SALESFORCE', 'QUICKBOOKS', 'DB' (derived from App, but stored for convenience)
	AuthType       string         `json:"auth_type" gorm:"type:varchar(50);default:'OAUTH2'"` // 'OAUTH2', 'API_KEY'
	AccessToken    string         `json:"-" gorm:"type:text;not null"`                        // Encrypted at application level
	RefreshToken   string         `json:"-" gorm:"type:text"`                                 // Encrypted at application level (optional for API_KEY)
	TokenExpiresAt *time.Time     `json:"token_expires_at" gorm:"column:token_expires_at"`
	ProviderUserID string         `json:"provider_user_id" gorm:"column:provider_user_id"`
	RealmID        string         `json:"realm_id" gorm:"column:realm_id"`                 // For QuickBooks Company ID
	Status         string         `json:"status" gorm:"type:varchar(50);default:'ACTIVE'"` // 'ACTIVE', 'EXPIRED', 'ERROR'
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	Metadata                *Metadata     `json:"metadata" gorm:"foreignKey:ConnectionID"`
	SourceIntegrations      []Integration `json:"source_integrations" gorm:"foreignKey:SourceConnectionID"`
	DestinationIntegrations []Integration `json:"destination_integrations" gorm:"foreignKey:DestinationConnectionID"`
	DataObjects             []DataObject  `json:"data_objects" gorm:"foreignKey:ConnectionID"`

	// Note: Multiple connections per company+app are allowed (e.g., one connection per Google Sheet)
}

// Metadata represents app-specific configuration for a connection
type Metadata struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ConnectionID uuid.UUID  `json:"connection_id" gorm:"type:uuid;not null;uniqueIndex"`
	Connection   Connection `json:"connection" gorm:"foreignKey:ConnectionID"`
	Data         string     `json:"data" gorm:"type:jsonb;not null;default:'{}'"` // Stores app-specific metadata
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Integration represents a sync workflow between source and destination apps
type Integration struct {
	ID                      uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	CompanyID               uuid.UUID      `json:"company_id" gorm:"type:uuid;not null;index"`
	Company                 Company        `json:"company" gorm:"foreignKey:CompanyID"`
	Name                    string         `json:"name" gorm:"not null"`
	SourceAppID             uuid.UUID      `json:"source_app_id" gorm:"type:uuid;not null;index"`
	SourceApp               App            `json:"source_app" gorm:"foreignKey:SourceAppID"`
	DestinationAppID        uuid.UUID      `json:"destination_app_id" gorm:"type:uuid;not null;index"`
	DestinationApp          App            `json:"destination_app" gorm:"foreignKey:DestinationAppID"`
	SourceConnectionID      *uuid.UUID     `json:"source_connection_id" gorm:"type:uuid;index"`
	SourceConnection        *Connection    `json:"source_connection" gorm:"foreignKey:SourceConnectionID"`
	DestinationConnectionID *uuid.UUID     `json:"destination_connection_id" gorm:"type:uuid;index"`
	DestinationConnection   *Connection    `json:"destination_connection" gorm:"foreignKey:DestinationConnectionID"`
	WebhookURL              string         `json:"webhook_url" gorm:"column:webhook_url"`
	FieldMapping            string         `json:"field_mapping" gorm:"type:jsonb;default:'{}'"`
	SyncFrequencyMinutes    int            `json:"sync_frequency_minutes" gorm:"default:60"`
	Status                  string         `json:"status" gorm:"default:active"` // 'active', 'paused', 'error'
	LastSyncedAt            *time.Time     `json:"last_synced_at" gorm:"column:last_synced_at"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
	DeletedAt               gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations (if sync_jobs and sync_records are still used)
	SyncJobs    []SyncJob    `json:"sync_jobs" gorm:"foreignKey:IntegrationID"`
	SyncRecords []SyncRecord `json:"sync_records" gorm:"foreignKey:IntegrationID"`
}

// SyncJob represents one full execution of a sync
type SyncJob struct {
	ID               uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	IntegrationID    uuid.UUID   `json:"integration_id" gorm:"type:uuid;not null;index"`
	Integration      Integration `json:"integration" gorm:"foreignKey:IntegrationID"`
	StartTime        time.Time   `json:"start_time" gorm:"default:now()"`
	EndTime          *time.Time  `json:"end_time" gorm:"column:end_time"`
	Status           string      `json:"status" gorm:"default:processing"` // 'processing', 'completed', 'failed'
	RecordsProcessed int         `json:"records_processed" gorm:"default:0"`
	RecordsFailed    int         `json:"records_failed" gorm:"default:0"`
	RecordsSuccess   int         `json:"records_success" gorm:"default:0"`
	TriggerType      string      `json:"trigger_type" gorm:"default:scheduled"` // 'scheduled', 'manual', 'webhook'
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`

	// Relations
	SyncRecords []SyncRecord `json:"sync_records" gorm:"foreignKey:JobID"`
}

// SyncRecord represents granular logs and retry data
type SyncRecord struct {
	ID            uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID         *uuid.UUID  `json:"job_id" gorm:"type:uuid;index"`
	Job           *SyncJob    `json:"job" gorm:"foreignKey:JobID"`
	IntegrationID uuid.UUID   `json:"integration_id" gorm:"type:uuid;not null;index"`
	Integration   Integration `json:"integration" gorm:"foreignKey:IntegrationID"`

	// Tracking IDs
	SourceRecordUniqueID string  `json:"source_record_unique_id" gorm:"column:source_record_unique_id"` // The ID from CSV row
	DestinationRecordID  *string `json:"destination_record_id" gorm:"column:destination_record_id"`     // The ID returned by QuickBooks

	Status       string  `json:"status" gorm:"not null"` // 'synced', 'failed', 'pending_retry'
	ErrorMessage *string `json:"error_message" gorm:"type:text"`

	// CRITICAL FOR RETRY LOGIC - Store raw data so we can retry without fetching from Drive again
	RawDataPayload string `json:"raw_data_payload" gorm:"type:jsonb;column:raw_data_payload"`

	// Hash to detect changes (md5 of the row data)
	DataHash  string         `json:"data_hash" gorm:"column:data_hash;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

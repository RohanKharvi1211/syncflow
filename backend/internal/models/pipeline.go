package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DataObject represents what exactly to read/write (sheet, table, invoice, entity)
// This is the key abstraction that allows flexible source/destination mapping
type DataObject struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ConnectionID uuid.UUID      `json:"connection_id" gorm:"type:uuid;not null;index"`
	Connection   Connection     `json:"connection" gorm:"foreignKey:ConnectionID"`
	ObjectType   string         `json:"object_type" gorm:"type:varchar(50);not null"` // 'SHEET', 'TABLE', 'INVOICE', 'ENTITY'
	Identifier   string         `json:"identifier" gorm:"type:text;not null"`        // sheet name, table name, object API name
	Config       string         `json:"config" gorm:"type:jsonb;default:'{}'"`       // range, columns, filters
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	SourcePipelines      []Pipeline `json:"source_pipelines" gorm:"foreignKey:SourceObjectID"`
	DestinationPipelines []Pipeline `json:"destination_pipelines" gorm:"foreignKey:DestinationObjectID"`
}

// Pipeline represents a source → destination mapping (sync job)
// This is the core entity that defines what syncs where
type Pipeline struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	CompanyID           uuid.UUID      `json:"company_id" gorm:"type:uuid;not null;index"`
	Company             Company         `json:"company" gorm:"foreignKey:CompanyID"`
	SourceObjectID      uuid.UUID      `json:"source_object_id" gorm:"type:uuid;not null;index"`
	SourceObject        DataObject     `json:"source_object" gorm:"foreignKey:SourceObjectID"`
	DestinationObjectID uuid.UUID      `json:"destination_object_id" gorm:"type:uuid;not null;index"`
	DestinationObject   DataObject     `json:"destination_object" gorm:"foreignKey:DestinationObjectID"`
	SyncType            string         `json:"sync_type" gorm:"type:varchar(50);default:'PULL'"` // 'PULL', 'PUSH', 'BIDIRECTIONAL'
	ScheduleInterval    int            `json:"schedule_interval" gorm:"default:60"`                // minutes
	Status              string         `json:"status" gorm:"type:varchar(50);default:'ACTIVE'"`   // 'ACTIVE', 'PAUSED'
	FieldMapping        string         `json:"field_mapping" gorm:"type:jsonb;default:'{}'"`      // field mapping configuration
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	Checkpoint *Checkpoint `json:"checkpoint" gorm:"foreignKey:PipelineID"`
	SyncRuns   []SyncRun   `json:"sync_runs" gorm:"foreignKey:PipelineID"`
}

// Checkpoint guarantees no data loss - stores the last sync position
// This is critical for reliable delivery and replayability
type Checkpoint struct {
	PipelineID    uuid.UUID `json:"pipeline_id" gorm:"type:uuid;primary_key"`
	Pipeline      Pipeline  `json:"pipeline" gorm:"foreignKey:PipelineID"`
	LastSyncTime  time.Time `json:"last_sync_time" gorm:"not null"`
	LastCursor    string    `json:"last_cursor" gorm:"type:text"` // row number, timestamp, CDC token
	UpdatedAt     time.Time `json:"updated_at"`
}

// SyncRun represents one execution of a pipeline (observability)
// This tracks each sync execution for monitoring and debugging
type SyncRun struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	PipelineID      uuid.UUID  `json:"pipeline_id" gorm:"type:uuid;not null;index"`
	Pipeline        Pipeline   `json:"pipeline" gorm:"foreignKey:PipelineID"`
	Status          string     `json:"status" gorm:"type:varchar(50);not null"` // 'SUCCESS', 'FAILED', 'PARTIAL'
	RecordsRead     int        `json:"records_read" gorm:"default:0"`
	RecordsWritten  int        `json:"records_written" gorm:"default:0"`
	Error           *string    `json:"error" gorm:"type:text"`
	StartedAt       time.Time  `json:"started_at" gorm:"not null"`
	FinishedAt      *time.Time `json:"finished_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}



export interface DataObject {
  id: string;
  connection_id: string;
  object_type: 'SHEET' | 'TABLE' | 'INVOICE' | 'ENTITY' | 'OBJECT';
  identifier: string;
  config: Record<string, any>;
  created_at: string;
  updated_at: string;
  connection?: {
    id: string;
    app: {
      name: string;
      display_name: string;
    };
  };
}

export interface Pipeline {
  id: string;
  company_id: string;
  source_object_id: string;
  destination_object_id: string;
  sync_type: 'PULL' | 'PUSH' | 'BIDIRECTIONAL';
  schedule_interval: number;
  status: 'ACTIVE' | 'PAUSED';
  field_mapping: Record<string, any>;
  created_at: string;
  updated_at: string;
  source_object?: DataObject;
  destination_object?: DataObject;
  checkpoint?: Checkpoint;
}

export interface Checkpoint {
  pipeline_id: string;
  last_sync_time: string;
  last_cursor?: string;
  updated_at: string;
}

export interface SyncRun {
  id: string;
  pipeline_id: string;
  status: 'SUCCESS' | 'FAILED' | 'PARTIAL';
  records_read: number;
  records_written: number;
  error?: string;
  started_at: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PipelineStatus {
  pipeline_id: string;
  status: string;
  schedule_interval: number;
  last_sync_time?: string;
  last_sync_status?: string;
  records_read: number;
  records_written: number;
  last_error?: string;
  last_cursor?: string;
}



export interface Integration {
  id: string;
  company_id: string;
  name: string;
  source_app_id: string;
  destination_app_id: string;
  source_connection_id?: string;
  destination_connection_id?: string;
  webhook_url?: string;
  field_mapping: Record<string, any>;
  sync_frequency_minutes: number;
  status: 'active' | 'paused' | 'error';
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
  source_app?: {
    id: string;
    name: string;
    display_name: string;
  };
  destination_app?: {
    id: string;
    name: string;
    display_name: string;
  };
}



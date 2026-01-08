export interface App {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  type: 'source' | 'destination' | 'both';
  metadata_schema: Record<string, any>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}



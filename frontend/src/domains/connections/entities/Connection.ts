export interface Connection {
  id: string;
  company_id: string;
  app_id: string;
  type?: string;
  auth_type: 'OAUTH2' | 'API_KEY';
  status: 'ACTIVE' | 'EXPIRED' | 'ERROR';
  is_active: boolean;
  token_expires_at?: string;
  provider_user_id?: string;
  realm_id?: string;
  created_at: string;
  updated_at: string;
  app?: {
    id: string;
    name: string;
    display_name: string;
    type: 'source' | 'destination' | 'both';
  };
  metadata?: {
    id: string;
    connection_id: string;
    data: Record<string, any>;
  };
  data_objects?: Array<{
    id: string;
    connection_id: string;
    object_type: string;
    identifier: string;
    config: Record<string, any>;
  }>;
}


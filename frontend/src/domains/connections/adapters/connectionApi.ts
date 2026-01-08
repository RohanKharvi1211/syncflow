import { httpClient } from '@shared/adapters/httpClient';
import { Connection } from '../entities/Connection';

class ConnectionApi {
  async getConnections(companyId: string): Promise<Connection[]> {
    const response = await httpClient.get<{ connections: Connection[] }>('/connections', {
      params: { company_id: companyId },
    });
    return response.connections;
  }

  async getConnection(id: string): Promise<Connection> {
    return httpClient.get<Connection>(`/connections/${id}`);
  }

  async createConnection(data: {
    company_id: string;
    app_id: string;
    access_token: string;
    refresh_token: string;
    metadata: Record<string, any>;
  }): Promise<Connection> {
    return httpClient.post<Connection>('/connections', data);
  }

  async updateConnection(id: string, data: Partial<Connection>): Promise<Connection> {
    return httpClient.put<Connection>(`/connections/${id}`, data);
  }

  async deleteConnection(id: string): Promise<void> {
    await httpClient.delete(`/connections/${id}`);
  }
}

export const connectionApi = new ConnectionApi();



import { httpClient } from '@shared/adapters/httpClient';
import { App } from '../entities/App';

class AppApi {
  async getApps(type?: 'source' | 'destination'): Promise<App[]> {
    const params: any = {};
    if (type) {
      params.type = type;
    }
    
    const response = await httpClient.get<{ apps: App[] }>('/apps', { params });
    return response.apps;
  }

  async getApp(id: string): Promise<App> {
    return httpClient.get<App>(`/apps/${id}`);
  }
}

export const appApi = new AppApi();



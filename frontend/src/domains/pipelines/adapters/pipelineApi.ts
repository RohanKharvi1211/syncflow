import { httpClient } from '@shared/adapters/httpClient';
import { Pipeline, DataObject, SyncRun, PipelineStatus } from '../entities/Pipeline';

class PipelineApi {
  async getPipelines(companyId: string): Promise<Pipeline[]> {
    const response = await httpClient.get<{ pipelines: Pipeline[] }>('/pipelines', {
      params: { company_id: companyId },
    });
    return response.pipelines;
  }

  async getPipeline(id: string): Promise<{ pipeline: Pipeline; status: PipelineStatus }> {
    return httpClient.get<{ pipeline: Pipeline; status: PipelineStatus }>(`/pipelines/${id}`);
  }

  async createPipeline(data: {
    company_id: string;
    source_object_id: string;
    destination_object_id: string;
    sync_type?: 'PULL' | 'PUSH' | 'BIDIRECTIONAL';
    schedule_interval?: number;
    field_mapping?: Record<string, any>;
  }): Promise<Pipeline> {
    return httpClient.post<Pipeline>('/pipelines', data);
  }

  async updatePipeline(id: string, data: Partial<Pipeline>): Promise<Pipeline> {
    return httpClient.put<Pipeline>(`/pipelines/${id}`, data);
  }

  async deletePipeline(id: string): Promise<void> {
    await httpClient.delete(`/pipelines/${id}`);
  }

  async executePipeline(id: string): Promise<void> {
    await httpClient.post(`/pipelines/${id}/execute`, {});
  }

  async getSyncRuns(pipelineId: string): Promise<SyncRun[]> {
    const response = await httpClient.get<{ sync_runs: SyncRun[] }>(`/pipelines/${pipelineId}/sync-runs`);
    return response.sync_runs;
  }
}

class DataObjectApi {
  async getDataObjects(connectionId: string): Promise<DataObject[]> {
    const response = await httpClient.get<{ data_objects: DataObject[] }>('/data-objects', {
      params: { connection_id: connectionId },
    });
    return response.data_objects;
  }

  async getDataObject(id: string): Promise<DataObject> {
    return httpClient.get<DataObject>(`/data-objects/${id}`);
  }

  async createDataObject(data: {
    connection_id: string;
    object_type: 'SHEET' | 'TABLE' | 'INVOICE' | 'ENTITY';
    identifier: string;
    config?: Record<string, any>;
  }): Promise<DataObject> {
    return httpClient.post<DataObject>('/data-objects', data);
  }

  async updateDataObject(id: string, data: Partial<DataObject>): Promise<DataObject> {
    return httpClient.put<DataObject>(`/data-objects/${id}`, data);
  }

  async deleteDataObject(id: string): Promise<void> {
    await httpClient.delete(`/data-objects/${id}`);
  }
}

export const pipelineApi = new PipelineApi();
export const dataObjectApi = new DataObjectApi();



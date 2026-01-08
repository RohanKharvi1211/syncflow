import { httpClient } from '@shared/adapters/httpClient';
import { Company } from '../entities/Company';

class CompanyApi {
  async getCompanies(): Promise<Company[]> {
    const response = await httpClient.get<{ companies: Company[] }>('/companies');
    return response.companies;
  }

  async getCompany(id: string): Promise<Company> {
    return httpClient.get<Company>(`/companies/${id}`);
  }

  async createCompany(data: Omit<Company, 'id' | 'created_at' | 'updated_at'>): Promise<Company> {
    return httpClient.post<Company>('/companies', data);
  }

  async updateCompany(id: string, data: Partial<Company>): Promise<Company> {
    return httpClient.put<Company>(`/companies/${id}`, data);
  }

  async deleteCompany(id: string): Promise<void> {
    await httpClient.delete(`/companies/${id}`);
  }
}

export const companyApi = new CompanyApi();



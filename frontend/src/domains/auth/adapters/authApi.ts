import { httpClient } from '@shared/adapters/httpClient';
import { User } from '../entities/User';

export interface SignInRequest {
  email: string;
  company_id?: string;
}

export interface SignInResponse {
  user: User;
  token?: string;
}

class AuthApi {
  async signIn(request: SignInRequest): Promise<SignInResponse> {
    const response = await httpClient.get<{ user: User }>('/users', {
      params: { email: request.email },
    });
    
    return {
      user: response.user,
    };
  }

  async getUserById(id: string): Promise<User> {
    return httpClient.get<User>(`/users/${id}`);
  }

  async getUsersByCompany(companyId: string): Promise<User[]> {
    const response = await httpClient.get<{ users: User[] }>('/users', {
      params: { company_id: companyId },
    });
    return response.users;
  }
}

export const authApi = new AuthApi();



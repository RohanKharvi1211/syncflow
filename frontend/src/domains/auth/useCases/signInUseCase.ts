import { authApi, SignInRequest } from '../adapters/authApi';
import { User } from '../entities/User';

export class SignInUseCase {
  async execute(request: SignInRequest): Promise<User> {
    const response = await authApi.signIn(request);
    
    // Store user in localStorage
    if (response.token) {
      localStorage.setItem('auth_token', response.token);
    }
    localStorage.setItem('user', JSON.stringify(response.user));
    
    return response.user;
  }
}



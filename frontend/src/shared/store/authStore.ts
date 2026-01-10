import { create } from 'zustand';
import { User, AuthState } from '@domains/auth/entities/User';

interface AuthStore extends AuthState {
  setUser: (user: User | null) => void;
  setToken: (token: string | null) => void;
  getToken: () => string | null;
  signOut: () => void;
  initialize: () => void;
}

export const useAuthStore = create<AuthStore>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,

  setUser: (user) => {
    // Update localStorage first to ensure persistence
    if (user) {
      localStorage.setItem('user', JSON.stringify(user));
    } else {
      localStorage.removeItem('user');
    }
    // Then update state - this ensures localStorage is always in sync
    set({ user, isAuthenticated: !!user, isLoading: false });
  },

  setToken: (token) => {
    // Update localStorage first
    if (token) {
      localStorage.setItem('auth_token', token);
    } else {
      localStorage.removeItem('auth_token');
    }
    // Note: setToken doesn't update isAuthenticated - that's handled by setUser
  },

  getToken: () => {
    return localStorage.getItem('auth_token');
  },

  signOut: () => {
    localStorage.removeItem('user');
    localStorage.removeItem('auth_token');
    set({ user: null, isAuthenticated: false, isLoading: false });
  },

  initialize: () => {
    const storedUser = localStorage.getItem('user');
    const storedToken = localStorage.getItem('auth_token');
    
    if (storedUser && storedToken) {
      try {
        const user = JSON.parse(storedUser);
        set({ user, isAuthenticated: true, isLoading: false });
      } catch {
        set({ user: null, isAuthenticated: false, isLoading: false });
      }
    } else {
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },
}));


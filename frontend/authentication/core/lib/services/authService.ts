import { User } from "../entities";

export interface AuthService {
  signInWithEmail: (email: string) => Promise<User | null>;
  signInWithGoogle: () => Promise<User | null>;
  findUserByEmail: (email: string) => Promise<User | null>;
}






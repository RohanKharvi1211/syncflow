import { User } from "../entities";
import { AuthService } from "../services";

const API_BASE_URL = "http://localhost:8080/api";

export class HttpAuthService implements AuthService {
  async signInWithEmail(email: string): Promise<User | null> {
    try {
      const response = await fetch(`${API_BASE_URL}/users`);
      const data = await response.json();
      const users = data.users || [];

      const user = users.find(
        (u: any) =>
          u.email?.toLowerCase() === email.toLowerCase() ||
          u.email?.toLowerCase().endsWith(`@${email.split("@")[1]}`)
      );

      if (!user) return null;

      return new User(user.id, user.email, user.company_name || "");
    } catch (error) {
      console.error("Error signing in with email:", error);
      return null;
    }
  }

  async signInWithGoogle(): Promise<User | null> {
    // This will be handled by OAuth callback
    return null;
  }

  async findUserByEmail(email: string): Promise<User | null> {
    return this.signInWithEmail(email);
  }
}






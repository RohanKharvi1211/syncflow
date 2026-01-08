import { User } from "../entities";
import { AuthService } from "../services";

export class SignInInteractor {
  private authService: AuthService;

  constructor(authService: AuthService) {
    this.authService = authService;
  }

  async signInWithEmail(email: string): Promise<User> {
    if (!email || !this.isValidEmail(email)) {
      throw new Error("Invalid email address");
    }

    const user = await this.authService.findUserByEmail(email);
    if (!user) {
      throw new Error("User not found. Please contact support.");
    }

    return user;
  }

  async signInWithGoogle(): Promise<User> {
    const user = await this.authService.signInWithGoogle();
    if (!user) {
      throw new Error("Google sign-in failed");
    }
    return user;
  }

  private isValidEmail(email: string): boolean {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email.trim());
  }
}






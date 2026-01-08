// Adapter to bridge Clean Architecture with current HTML/JS implementation
import { SignInInteractor } from "../../../core/lib/useCases";
import { HttpAuthService } from "../../../core/lib/adapters";

export class SignInAdapter {
  private signInInteractor: SignInInteractor;

  constructor() {
    const authService = new HttpAuthService();
    this.signInInteractor = new SignInInteractor(authService);
  }

  async signInWithEmail(email: string) {
    try {
      const user = await this.signInInteractor.signInWithEmail(email);
      return { success: true, user };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  async signInWithGoogle() {
    try {
      const user = await this.signInInteractor.signInWithGoogle();
      return { success: true, user };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }
}

// Export for use in current JS files
if (typeof window !== "undefined") {
  (window as any).SignInAdapter = SignInAdapter;
}






export class Email {
  private _address: string;

  constructor(address: string) {
    if (!this.isValid(address)) {
      throw new Error("Invalid email address");
    }
    this._address = address.toLowerCase().trim();
  }

  get address(): string {
    return this._address;
  }

  private isValid(email: string): boolean {
    if (!email || typeof email !== "string") return false;
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email.trim());
  }
}






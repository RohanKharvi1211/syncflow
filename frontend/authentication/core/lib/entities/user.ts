export class User {
  private _id: string;
  private _email: string;
  private _companyName: string;

  constructor(id: string, email: string, companyName: string) {
    if (!id || !email) {
      throw new Error("User must have id and email");
    }
    this._id = id;
    this._email = email.toLowerCase();
    this._companyName = companyName || this.extractCompanyFromEmail(email);
  }

  get id(): string {
    return this._id;
  }

  get email(): string {
    return this._email;
  }

  get companyName(): string {
    return this._companyName;
  }

  private extractCompanyFromEmail(email: string): string {
    const domain = email.split("@")[1];
    if (!domain) return "";
    const company = domain.split(".")[0];
    return company.charAt(0).toUpperCase() + company.slice(1);
  }
}






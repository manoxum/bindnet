export interface Certificate {
  id: string;
  domain: string;
  commonName: string;
  issuedAt: string;
  expiresAt: string;
  revokedAt?: string;
}

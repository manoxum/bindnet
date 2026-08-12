export type CertificateValidityUnit = "days" | "weeks" | "months" | "years";

export interface IssueCertificateRequest {
  name: string;
  domains: string[];
  validityQuantity: number;
  validityUnit: CertificateValidityUnit;
}

export type ReissueCertificateRequest = Omit<IssueCertificateRequest, "name">;

export interface Certificate {
  id: string;
  name: string;
  domain: string;
  commonName: string;
  dnsNames?: string[];
  ipAddresses?: string[];
  issuedAt: string;
  expiresAt: string;
  validityQuantity: number;
  validityUnit: CertificateValidityUnit;
  revokedAt?: string;
}

// allCertificateDomains junta dnsNames + ipAddresses (todos os SAN do
// certificado, incluindo o domínio primário) para exibição na UI.
export function allCertificateDomains(certificate: Certificate): string[] {
  return [...(certificate.dnsNames ?? []), ...(certificate.ipAddresses ?? [])];
}

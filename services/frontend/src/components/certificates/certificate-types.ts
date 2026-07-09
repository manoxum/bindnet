export type CertificateValidityUnit = "days" | "weeks" | "months" | "years";

export interface IssueCertificateRequest {
  domains: string[];
  validityQuantity: number;
  validityUnit: CertificateValidityUnit;
}

export interface Certificate {
  id: string;
  domain: string;
  commonName: string;
  dnsNames?: string[];
  ipAddresses?: string[];
  issuedAt: string;
  expiresAt: string;
  revokedAt?: string;
}

// allCertificateDomains junta dnsNames + ipAddresses (todos os SAN do
// certificado, incluindo o domínio primário) para exibição na UI.
export function allCertificateDomains(certificate: Certificate): string[] {
  return [...(certificate.dnsNames ?? []), ...(certificate.ipAddresses ?? [])];
}

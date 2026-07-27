// Tipos dos avisos enviados aos dispositivos - espelham o DTO do backend
// (services/backend/internal/hotspot/hotspot_messages.go).
export interface HotspotMessage {
  id: string;
  title?: string;
  body: string;
  targetMac?: string;
  urgent: boolean;
  expiresAt?: string;
  createdAt: string;
}

export interface HotspotMessageCreateRequest {
  title?: string;
  body: string;
  targetMac?: string;
  urgent: boolean;
  expiresAt?: string;
}

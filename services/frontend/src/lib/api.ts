// Wrapper fino sobre fetch para falar com o services/backend. Usa
// caminho relativo "/api" - em produção o nginx do próprio frontend
// repassa isso ao backend (mesma origem, sem CORS); em dev o vite faz
// o proxy equivalente (ver vite.config.ts).
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export interface ApiRequestOptions {
  timeoutMs?: number;
}
async function request<T>(
  path: string,
  options: RequestInit = {},
  timing: ApiRequestOptions = {},
): Promise<T> {
  const controller = timing.timeoutMs ? new AbortController() : undefined;
  const timer = controller ? window.setTimeout(() => controller.abort(), timing.timeoutMs) : undefined;
  try {
    const response = await fetch(`/api${path}`, {
      credentials: "include",
      headers: options.body ? { "Content-Type": "application/json" } : undefined,
      ...options,
      ...(controller ? { signal: controller.signal } : {}),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new ApiError(response.status, text || response.statusText);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    const contentType = response.headers.get("content-type") ?? "";
    if (contentType.includes("application/json")) {
      return (await response.json()) as T;
    }
    return undefined as T;
  } catch (error) {
    if (controller?.signal.aborted)
      throw new ApiError(
        408,
        "A operação excedeu o prazo de espera. Consulte o estado dos sites e recursos antes de repetir; a publicação pode ter sido parcialmente concluída.",
      );
    throw error;
  } finally {
    if (timer !== undefined) window.clearTimeout(timer);
  }
}

export const api = {
  get: <T>(path: string, timing?: ApiRequestOptions) => request<T>(path, {}, timing),
  post: <T>(path: string, body?: unknown, timing?: ApiRequestOptions) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }, timing),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),
  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};

export function logsUrl(path: string, follow: boolean) {
  return `/api${path}${path.includes("?") ? "&" : "?"}follow=${follow}`;
}

import { reactive } from "vue";

export type User = {
  id: number;
  username: string;
  isAdmin: boolean;
  language: string;
  accentColor: string;
};
export const session = reactive<{
  accessToken: string;
  accessTokenExpiresAt: number;
  user: User | null;
  impersonating: boolean;
}>({
  accessToken: "",
  accessTokenExpiresAt: 0,
  user: null,
  impersonating: false,
});

const refreshSkewMs = 60_000;
const refreshRetryMs = 30_000;
let refreshInFlight: Promise<boolean> | null = null;
let proactiveRefreshTimer: number | undefined;
let refreshLifecycleActive = false;
let sessionRevision = 0;

export function accessTokenNeedsRefresh(now = Date.now()) {
  return (
    !session.accessToken ||
    !session.accessTokenExpiresAt ||
    session.accessTokenExpiresAt - now <= refreshSkewMs
  );
}

function clearRefreshTimer() {
  if (proactiveRefreshTimer !== undefined)
    window.clearTimeout(proactiveRefreshTimer);
  proactiveRefreshTimer = undefined;
}

function scheduleProactiveRefresh(delayOverride?: number) {
  clearRefreshTimer();
  if (
    !refreshLifecycleActive ||
    !session.user ||
    document.visibilityState !== "visible"
  )
    return;
  const delay =
    delayOverride ??
    Math.max(0, session.accessTokenExpiresAt - Date.now() - refreshSkewMs);
  proactiveRefreshTimer = window.setTimeout(async () => {
    proactiveRefreshTimer = undefined;
    if (document.visibilityState !== "visible") return;
    if (!(await refresh()) && session.user)
      scheduleProactiveRefresh(refreshRetryMs);
  }, delay);
}

export function clearSession() {
  sessionRevision += 1;
  clearRefreshTimer();
  session.accessToken = "";
  session.accessTokenExpiresAt = 0;
  session.user = null;
  session.impersonating = false;
}

export async function api<T = any>(
  path: string,
  options: RequestInit = {},
  retry = true,
): Promise<T> {
  if (!path.startsWith("/auth/") && session.user && accessTokenNeedsRefresh())
    await refresh();
  const headers = new Headers(options.headers);
  if (options.body && !headers.has("Content-Type"))
    headers.set("Content-Type", "application/json");
  if (session.accessToken)
    headers.set("Authorization", `Bearer ${session.accessToken}`);
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers,
    credentials: "same-origin",
  });
  if (response.status === 401 && retry && !path.startsWith("/auth/")) {
    if (await refresh()) return api<T>(path, options, false);
  }
  if (!response.ok) {
    const body = await response
      .json()
      .catch(() => ({ error: { message: `HTTP ${response.status}` } }));
    throw new Error(body.error?.message ?? `HTTP ${response.status}`);
  }
  if (response.status === 204) return undefined as T;
  return response.json();
}

function applyTokens(payload: any) {
  session.accessToken = payload.accessToken;
  session.accessTokenExpiresAt =
    Date.now() + Math.max(0, Number(payload.expiresIn) || 0) * 1000;
  session.user = payload.user;
  session.impersonating = Boolean(payload.impersonating);
  scheduleProactiveRefresh();
}

export function acceptTokens(payload: any) {
  sessionRevision += 1;
  applyTokens(payload);
}

export async function refresh() {
  if (refreshInFlight) return refreshInFlight;
  const revision = sessionRevision;
  refreshInFlight = (async () => {
    try {
      const response = await fetch("/api/v1/auth/refresh", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: "{}",
      });
      if (!response.ok) {
        if (response.status === 401 || response.status === 403) clearSession();
        return false;
      }
      const payload = await response.json();
      if (revision !== sessionRevision) return false;
      applyTokens(payload);
      return true;
    } catch {
      return false;
    }
  })().finally(() => {
    refreshInFlight = null;
  });
  return refreshInFlight;
}

export function startSessionRefresh() {
  refreshLifecycleActive = true;
  const resume = () => {
    if (document.visibilityState !== "visible") {
      clearRefreshTimer();
    } else if (session.user && accessTokenNeedsRefresh()) {
      void refresh();
    } else {
      scheduleProactiveRefresh();
    }
  };
  document.addEventListener("visibilitychange", resume);
  window.addEventListener("focus", resume);
  scheduleProactiveRefresh();
  return () => {
    refreshLifecycleActive = false;
    clearRefreshTimer();
    document.removeEventListener("visibilitychange", resume);
    window.removeEventListener("focus", resume);
  };
}

export async function streamAvailability(
  items: unknown[],
  onItem: (item: any) => void,
) {
  if (session.user && accessTokenNeedsRefresh()) await refresh();
  const bytes = new TextEncoder().encode(JSON.stringify(items));
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  const encoded = btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  const request = async () =>
    fetch(`/api/v1/availability/stream?items=${encoded}`, {
      headers: { Authorization: `Bearer ${session.accessToken}` },
      credentials: "same-origin",
    });
  let response = await request();
  if (response.status === 401 && (await refresh())) response = await request();
  if (!response.ok || !response.body)
    throw new Error(`Availability stream failed (${response.status})`);
  const reader = response.body.getReader(),
    decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const events = buffer.split("\n\n");
    buffer = events.pop() || "";
    for (const event of events) {
      const data = event.split("\n").find((line) => line.startsWith("data: "));
      if (data) onItem(JSON.parse(data.slice(6)));
    }
    if (done) break;
  }
}

export async function streamAPI<T>(
  path: string,
  options: RequestInit,
  onEvent: (event: T) => void,
) {
  if (session.user && accessTokenNeedsRefresh()) await refresh();
  const request = () => {
    const headers = new Headers(options.headers);
    if (options.body && !headers.has("Content-Type"))
      headers.set("Content-Type", "application/json");
    if (session.accessToken)
      headers.set("Authorization", `Bearer ${session.accessToken}`);
    return fetch(`/api/v1${path}`, {
      ...options,
      headers,
      credentials: "same-origin",
    });
  };
  let response = await request();
  if (response.status === 401 && (await refresh())) response = await request();
  if (!response.ok) {
    const body = await response
      .json()
      .catch(() => ({ error: { message: `HTTP ${response.status}` } }));
    throw new Error(body.error?.message ?? `HTTP ${response.status}`);
  }
  if (!response.body) throw new Error("The response stream is unavailable");

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const lines = buffer.split("\n");
    buffer = lines.pop() ?? "";
    for (const line of lines) {
      if (line.trim()) onEvent(JSON.parse(line) as T);
    }
    if (done) break;
  }
  if (buffer.trim()) onEvent(JSON.parse(buffer) as T);
}

export function artwork(path?: string | null, size = "w500") {
  if (!path) return "";
  // The URL is stable: the backend stores and serves this exact immutable size.
  return `/api/v1/images/cache?path=${encodeURIComponent(path)}&size=${size}`;
}

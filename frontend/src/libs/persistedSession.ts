import type { AuthUser } from "../helper/authEvents";

export type PersistedSession = {
  accessToken: string;
  expiresAt: number;
  lifetimeMs: number;
  user: AuthUser;
};

const STORAGE_KEY = "auth_session";

export function saveSession(session: PersistedSession) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
  } catch {}
}

export function loadSession(): PersistedSession | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const session = raw ? (JSON.parse(raw) as Partial<PersistedSession>) : null;
    return isPersistedSession(session) ? session : null;
  } catch {
    return null;
  }
}

export function clearSession() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {}
}

function isPersistedSession(
  value: Partial<PersistedSession> | null,
): value is PersistedSession {
  return (
    typeof value?.accessToken === "string" &&
    typeof value.expiresAt === "number" &&
    typeof value.lifetimeMs === "number" &&
    typeof value.user?.id === "string" &&
    typeof value.user.name === "string"
  );
}

export type AuthUser = { id: string; name: string };

export type RefreshReason =
  | "startup"
  | "proactive"
  | "request"
  | "401"
  | "visibility";

export type AuthEvent =
  | { type: "refresh-start"; reason: RefreshReason }
  | { type: "refresh-success"; user: AuthUser; expiresIn: number }
  | { type: "refresh-skipped" }
  | { type: "refresh-failed"; sessionExpired: boolean; message: string }
  | { type: "synced-from-tab"; user: AuthUser }
  | { type: "session-restored"; user: AuthUser; remainingMs: number }
  | { type: "signed-in"; user: AuthUser }
  | { type: "signed-out" };

type Listener = (event: AuthEvent) => void;

const listeners = new Set<Listener>();

export function subscribeAuthEvents(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function emitAuthEvent(event: AuthEvent) {
  listeners.forEach((listener) => listener(event));
}

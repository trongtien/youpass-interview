import type { ResponseAuth } from "../../api/authApi";
import { tokenStore } from "../../libs/tokenStore";
import {
  clearSession,
  loadSession,
  saveSession,
} from "../../libs/persistedSession";
import { emitAuthEvent } from "../authEvents";
import { refreshMarginMs } from "./refreshMargin";

export function applySession(data: ResponseAuth, { persist = true } = {}) {
  tokenStore.set(data.accessToken, data.expiresIn);
  if (persist) {
    saveSession({
      accessToken: data.accessToken,
      expiresAt: tokenStore.getExpiresAt(),
      lifetimeMs: tokenStore.getLifetimeMs(),
      user: data.user,
    });
  }
}

export function restoreSession(): boolean {
  const saved = loadSession();
  if (!saved) return false;

  tokenStore.restore(saved.accessToken, saved.expiresAt, saved.lifetimeMs);
  if (tokenStore.expiresWithin(refreshMarginMs())) {
    tokenStore.clear();
    return false;
  }

  emitAuthEvent({
    type: "session-restored",
    user: saved.user,
    remainingMs: saved.expiresAt - Date.now(),
  });
  return true;
}

export function signOut() {
  tokenStore.clear();
  clearSession();
  emitAuthEvent({ type: "signed-out" });
}

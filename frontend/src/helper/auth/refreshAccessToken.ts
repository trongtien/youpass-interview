import { isAxiosError } from "axios";
import { authApi } from "../../api/authApi";
import { tokenStore } from "../../libs/tokenStore";
import { emitAuthEvent, type RefreshReason } from "../authEvents";
import { broadcastSession, crossTabLock } from "./crossTab";
import { refreshMarginMs } from "./refreshMargin";
import { applySession, signOut } from "./session";

export class SessionExpiredError extends Error {
  constructor() {
    super("Session expired");
    this.name = "SessionExpiredError";
  }
}

let inflight: Promise<string> | null = null;
export function refreshAccessToken(
  reason: RefreshReason = "request",
): Promise<string> {
  inflight ??= makeRefresh(reason).finally(() => {
    inflight = null;
  });
  return inflight;
}

async function makeRefresh(reason: RefreshReason): Promise<string> {
  const tokenBefore = tokenStore.get();

  return crossTabLock(async () => {
    const tokenNow = tokenStore.get();
    if (
      tokenNow &&
      tokenNow !== tokenBefore &&
      !tokenStore.expiresWithin(refreshMarginMs())
    ) {
      emitAuthEvent({ type: "refresh-skipped" });
      return tokenNow;
    }

    emitAuthEvent({ type: "refresh-start", reason });
    try {
      const data = await authApi.refresh();
      applySession(data);
      broadcastSession(data);
      emitAuthEvent({
        type: "refresh-success",
        user: data.user,
        expiresIn: data.expiresIn,
      });

      return data.accessToken;
    } catch (e) {
      const status = isAxiosError(e) ? e.response?.status : undefined;
      const sessionExpired = status === 401 || status === 403;
      emitAuthEvent({
        type: "refresh-failed",
        sessionExpired,
        message: e instanceof Error ? e.message : String(e),
      });

      if (sessionExpired) {
        signOut();
        throw new SessionExpiredError();
      }
      throw e;
    }
  });
}

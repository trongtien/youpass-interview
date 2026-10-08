import { tokenStore } from "../../libs/tokenStore";
import { subscribeAuthEvents } from "../authEvents";
import { refreshAccessToken } from "./refreshAccessToken";
import { refreshMarginMs } from "./refreshMargin";

let timer: ReturnType<typeof setTimeout> | undefined;

function scheduleProactiveRefresh() {
  clearTimeout(timer);
  const refreshAt = tokenStore.getExpiresAt() - refreshMarginMs();

  timer = setTimeout(
    () => {
      refreshAccessToken("proactive").catch(() => {});
    },
    Math.max(refreshAt - Date.now(), 0),
  );
}

subscribeAuthEvents((event) => {
  switch (event.type) {
    case "signed-in":
    case "refresh-success":
    case "synced-from-tab":
    case "session-restored":
      scheduleProactiveRefresh();
      break;
    case "signed-out":
      clearTimeout(timer);
  }
});

document.addEventListener("visibilitychange", () => {
  if (
    document.visibilityState === "visible" &&
    tokenStore.get() &&
    tokenStore.expiresWithin(refreshMarginMs())
  ) {
    refreshAccessToken("visibility").catch(() => {});
  }
});

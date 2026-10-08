import { tokenStore } from "../../libs/tokenStore";

const MAX_REFRESH_MARGIN_MS = 90_000;

/**
 * Refresh trước khi hết hạn 90s
 */
export function refreshMarginMs() {
  return Math.min(MAX_REFRESH_MARGIN_MS, tokenStore.getLifetimeMs() / 4);
}

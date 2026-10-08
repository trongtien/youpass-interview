let accessToken: string | null = null;
let expiresAt = 0;
let lifetimeMs = 0;

export const tokenStore = {
  get,
  set,
  restore,
  clear,
  expiresWithin,
  getExpiresAt,
  getLifetimeMs,
};

function get() {
  return accessToken;
}

function set(token: string, expiresInSec: number) {
  accessToken = token;
  lifetimeMs = expiresInSec * 1000;
  expiresAt = Date.now() + lifetimeMs;
}

function restore(
  token: string,
  savedExpiresAt: number,
  savedLifetimeMs: number,
) {
  accessToken = token;
  expiresAt = savedExpiresAt;
  lifetimeMs = savedLifetimeMs;
}

function clear() {
  accessToken = null;
  expiresAt = 0;
  lifetimeMs = 0;
}

function expiresWithin(ms: number) {
  return !accessToken || Date.now() >= expiresAt - ms;
}

function getExpiresAt() {
  return expiresAt;
}

function getLifetimeMs() {
  return lifetimeMs;
}

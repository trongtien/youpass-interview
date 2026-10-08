import axios, { type AxiosError } from "axios";
import { tokenStore } from "./tokenStore";
import { refreshAccessToken, refreshMarginMs } from "../helper/auth";

declare module "axios" {
  interface AxiosRequestConfig<D = any> {
    skipAuth?: boolean;
    _authRetries?: number;
  }
}

const MAX_AUTH_RETRIES = 2;

export const api = axios.create({
  baseURL: "/api",
  withCredentials: true,
});

async function getValidToken(force = false): Promise<string> {
  if (force || tokenStore.expiresWithin(refreshMarginMs() / 2)) {
    return refreshAccessToken(force ? "401" : "request");
  }
  return tokenStore.get()!;
}

api.interceptors.request.use(async (config) => {
  if (config.skipAuth) return config;

  config.headers.Authorization = `Bearer ${await getValidToken()}`;
  return config;
});

api.interceptors.response.use(undefined, async (error: AxiosError) => {
  const config = error.config;
  const retries = config?._authRetries ?? 0;
  if (
    error.response?.status !== 401 ||
    !config ||
    config.skipAuth ||
    retries >= MAX_AUTH_RETRIES
  ) {
    return Promise.reject(error);
  }

  config._authRetries = retries + 1;
  const sentToken = String(config.headers.Authorization ?? "").replace(
    /^Bearer /,
    "",
  );
  if (sentToken === tokenStore.get()) {
    await getValidToken(true);
  }

  return api(config);
});

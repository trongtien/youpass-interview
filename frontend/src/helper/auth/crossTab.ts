import type { ResponseAuth } from "../../api/authApi";
import { emitAuthEvent } from "../authEvents";
import { applySession, signOut } from "./session";

const channel = new BroadcastChannel("authentication");
const CHANNEL_TYPE = {
  REFRESHED: "refreshed",
  LOGOUT: "logout",
} as const;

type ChannelMessage =
  | ({ type: typeof CHANNEL_TYPE.REFRESHED } & ResponseAuth)
  | { type: typeof CHANNEL_TYPE.LOGOUT };

channel.onmessage = ({ data }: MessageEvent<ChannelMessage>) => {
  if (data.type === CHANNEL_TYPE.REFRESHED) {
    applySession(data, { persist: false });
    emitAuthEvent({ type: "synced-from-tab", user: data.user });
  }

  if (data.type === CHANNEL_TYPE.LOGOUT) {
    signOut();
  }
};

function post(message: ChannelMessage) {
  channel.postMessage(message);
}

export function broadcastSession(data: ResponseAuth) {
  post({ type: CHANNEL_TYPE.REFRESHED, ...data });
}

export function broadcastLogout() {
  post({ type: CHANNEL_TYPE.LOGOUT });
}

export function crossTabLock(
  callback: () => Promise<string>,
): Promise<string> {
  if ("locks" in navigator) {
    return navigator.locks.request("authentication-refresh", callback);
  }

  return callback();
}

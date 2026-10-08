import { authApi } from "../../api/authApi";
import { emitAuthEvent, type AuthUser } from "../authEvents";
import { broadcastLogout, broadcastSession } from "./crossTab";
import { applySession, signOut } from "./session";

export async function login(
  username: string,
  password: string,
): Promise<AuthUser> {
  const data = await authApi.login(username, password);
  applySession(data);
  broadcastSession(data);
  emitAuthEvent({ type: "signed-in", user: data.user });

  return data.user;
}

export async function logout() {
  try {
    await authApi.logout();
  } finally {
    signOut();
    broadcastLogout();
  }
}

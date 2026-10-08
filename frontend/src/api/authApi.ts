import axios from "axios";
import type { AuthUser } from "../helper/authEvents";

export type ResponseAuth = {
  accessToken: string;
  expiresIn: number;
  user: AuthUser;
};

const authHttp = axios.create({
  baseURL: "/api",
  withCredentials: true,
});

export const authApi = {
  login,
  refresh,
  logout,
};

async function login(username: string, password: string) {
  const { data } = await authHttp.post<ResponseAuth>("/auth/login", {
    username,
    password,
  });
  return data;
}

async function refresh() {
  const { data } = await authHttp.post<ResponseAuth>("/auth/refresh");
  return data;
}

async function logout() {
  await authHttp.post("/auth/logout");
}

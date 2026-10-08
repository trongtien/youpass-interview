import { createContext, useContext } from "react";
import type { AuthUser } from "./helper/authEvents";

export type StatusCheckingAuth = "checking" | "in" | "out";
export type AuthContextProps = {
  status: StatusCheckingAuth;
  user: AuthUser | null;
};

export const AuthContext = createContext<AuthContextProps>({
  status: "checking",
  user: null,
});

export const useAuth = () => useContext(AuthContext);

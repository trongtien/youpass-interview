import { useEffect, useRef, useState, type PropsWithChildren } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { AuthContext, type AuthContextProps } from "./authContext";
import { subscribeAuthEvents, type AuthUser } from "./helper/authEvents";
import { refreshAccessToken, restoreSession } from "./helper/auth";

const SIGNED_OUT: AuthContextProps = { status: "out", user: null };

const AuthProvider = (props: PropsWithChildren) => {
  const { children } = props;
  const queryClient = useQueryClient();
  const [auth, setAuth] = useState<AuthContextProps>({
    status: "checking",
    user: null,
  });
  const userIdRef = useRef<string | null>(null);

  useEffect(() => {
    const handleUser = (user: AuthUser) => {
      if (userIdRef.current && userIdRef.current !== user.id) {
        window.location.reload();
        return;
      }
      userIdRef.current = user.id;
      setAuth((prev) =>
        prev.status === "in" && prev.user?.id === user.id
          ? prev
          : { status: "in", user },
      );
    };

    const unsubscribe = subscribeAuthEvents((event) => {
      if (event.type === "signed-out") {
        userIdRef.current = null;
        queryClient.clear();
        setAuth(SIGNED_OUT);
      }

      if (
        event.type === "refresh-success" ||
        event.type === "signed-in" ||
        event.type === "synced-from-tab" ||
        event.type === "session-restored"
      ) {
        handleUser(event.user);
      }
    });

    if (!restoreSession()) {
      refreshAccessToken("startup").catch(() =>
        setAuth((prev) => (prev.status === "checking" ? SIGNED_OUT : prev)),
      );
    }

    return unsubscribe;
  }, [queryClient]);

  if (auth.status === "checking") {
    return <p className="auth-splash">Đang khôi phục phiên đăng nhập…</p>;
  }

  return <AuthContext.Provider value={auth}>{children}</AuthContext.Provider>;
};

export default AuthProvider;

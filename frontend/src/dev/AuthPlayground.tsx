import { useSyncExternalStore } from "react";
import { useAuth } from "../authContext";
import { subscribeAuthEvents } from "../helper/authEvents";
import { logout } from "../helper/auth";
import "./AuthPlayground.css";

let refreshCount = 0;
const listeners = new Set<() => void>();

subscribeAuthEvents((event) => {
  if (event.type !== "refresh-start") return;
  refreshCount += 1;
  listeners.forEach((listener) => listener());
});

function useRefreshCount() {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    () => refreshCount,
  );
}

const AuthPlayground = () => {
  const { user } = useAuth();
  const count = useRefreshCount();

  return (
    <main className="pg">
      <header className="pg-header">
        <div>
          <h1>Refresh token playground</h1>
          <p>
            Đang đăng nhập: <b>{user?.name}</b>
          </p>
        </div>
        <button className="pg-ghost" onClick={() => logout()}>
          Đăng xuất
        </button>
      </header>

      <div className="pg-metric">
        <span className="pg-metric-value">{count}</span>
        <span className="pg-metric-label">Số lần gọi refresh token</span>
      </div>
    </main>
  );
};

export default AuthPlayground;

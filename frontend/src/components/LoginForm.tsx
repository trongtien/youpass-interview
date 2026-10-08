import { useState, type FormEvent } from "react";
import { isAxiosError } from "axios";
import { login } from "../helper/auth";
import "./auth.css";

const LoginForm = () => {
  const [username, setUsername] = useState("student");
  const [password, setPassword] = useState("123456");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await login(username, password);
    } catch (e) {
      setError(
        isAxiosError(e) && e.response?.status === 401
          ? "Sai tài khoản hoặc mật khẩu"
          : "Không kết nối được máy chủ",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      <label>
        Tài khoản
        <input
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
        />
      </label>
      <label>
        Mật khẩu
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
      </label>
      {error && <p className="auth-error">{error}</p>}
      <div className="auth-actions">
        <button type="submit" disabled={submitting}>
          {submitting ? "Đang đăng nhập…" : "Đăng nhập"}
        </button>
      </div>
      <p className="auth-hint">
        Mock: <code>student</code> / <code>student2</code>, mật khẩu <code>123456</code>
      </p>
    </form>
  );
};

export default LoginForm;

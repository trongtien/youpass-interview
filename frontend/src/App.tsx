import "./App.css";
import AuthProvider from "./AuthProvider";
import { useAuth } from "./authContext";
import LoginForm from "./components/LoginForm";
import AuthPlayground from "./dev/AuthPlayground";

function Page() {
  const { status } = useAuth();

  if (status !== "in") {
    return (
      <main className="auth-page">
        <h1>Đăng nhập</h1>
        <LoginForm />
      </main>
    );
  }

  return <AuthPlayground />;
}

function App() {
  return (
    <AuthProvider>
      <Page />
    </AuthProvider>
  );
}

export default App;

import { randomBytes } from "node:crypto";
import type { IncomingMessage, ServerResponse } from "node:http";
import type { Connect, Logger, Plugin } from "vite";

/**
 * Mock API auth chạy trong dev server của Vite, mô phỏng backend thật:
 * - Access token ngắn hạn gửi qua `Authorization: Bearer`
 * - Refresh token nằm trong cookie httpOnly, xoay vòng (rotation) sau mỗi lần refresh
 * - Phát hiện refresh token bị dùng lại, có thời gian ân hạn
 * - `/api/__mock/auth/*` để xem trạng thái và giả lập sự cố
 */

export type AuthMockOptions = {
  accessTtlSec: number;
  refreshTtlSec: number;
  latencyMs: number;
  reuseGraceSec: number;
};

const DEFAULT_OPTIONS: AuthMockOptions = {
  accessTtlSec: 20,
  refreshTtlSec: 120,
  latencyMs: 300,
  reuseGraceSec: 10,
};

const USERS = [
  { id: "u_1", username: "student", password: "123456", name: "Học viên A" },
  { id: "u_2", username: "student2", password: "123456", name: "Học viên B" },
];

const COOKIE_NAME = "refresh_token";
const COOKIE_PATH = "/api/auth";

type Session = {
  id: string;
  userId: string;
  generation: number;
  refreshToken: string;
  refreshExpiresAt: number;
  accessExpiresAt: number;
  revokedReason: string | null;
};

type RefreshRecord = { sessionId: string; expiresAt: number; usedAt?: number };
type AccessRecord = { sessionId: string; expiresAt: number };

type Request = Connect.IncomingMessage;
type Handler = (req: Request, res: ServerResponse, url: URL) => Promise<void> | void;

export function authMock(overrides: Partial<AuthMockOptions> = {}): Plugin {
  const config = { ...DEFAULT_OPTIONS };
  applyOptions(config, overrides);

  return {
    name: "auth-mock",
    configureServer(server) {
      server.middlewares.use("/api", createAuthMiddleware(config, server.config.logger));
    },
    configurePreviewServer(server) {
      server.middlewares.use("/api", createAuthMiddleware(config, server.config.logger));
    },
  };
}

/** Đọc MOCK_ACCESS_TTL, MOCK_REFRESH_TTL, MOCK_LATENCY, MOCK_REUSE_GRACE từ env / .env.local */
export function authMockOptionsFromEnv(env: Record<string, string>): Partial<AuthMockOptions> {
  return {
    accessTtlSec: Number(env.MOCK_ACCESS_TTL),
    refreshTtlSec: Number(env.MOCK_REFRESH_TTL),
    latencyMs: Number(env.MOCK_LATENCY),
    reuseGraceSec: Number(env.MOCK_REUSE_GRACE),
  };
}

function applyOptions(config: AuthMockOptions, patch: Record<string, unknown>) {
  for (const key of Object.keys(DEFAULT_OPTIONS) as (keyof AuthMockOptions)[]) {
    const value = patch[key];
    if (typeof value === "number" && Number.isFinite(value) && value >= 0) {
      config[key] = value;
    }
  }
}

function createAuthMiddleware(config: AuthMockOptions, logger: Logger): Connect.NextHandleFunction {
  const sessions = new Map<string, Session>();
  const accessTokens = new Map<string, AccessRecord>();
  const refreshTokens = new Map<string, RefreshRecord>();
  const stats = {
    login: 0,
    refreshOk: 0,
    refreshFailed: 0,
    reuseInGrace: 0,
    reuseDetected: 0,
    apiOk: 0,
    api401: 0,
  };
  let sessionSeq = 0;
  let failNextRefresh = false;
  let lastSweepAt = Date.now();

  const log = (message: string) => logger.info(`[auth-mock] ${message}`, { timestamp: true });

  function issueAccessToken(session: Session) {
    const token = `at_${session.generation}_${randomId()}`;
    session.accessExpiresAt = Date.now() + config.accessTtlSec * 1000;
    accessTokens.set(token, { sessionId: session.id, expiresAt: session.accessExpiresAt });
    return token;
  }

  function rotateRefreshToken(session: Session) {
    session.generation += 1;
    session.refreshToken = `rt_${session.generation}_${randomId()}`;
    session.refreshExpiresAt = Date.now() + config.refreshTtlSec * 1000;
    refreshTokens.set(session.refreshToken, {
      sessionId: session.id,
      expiresAt: session.refreshExpiresAt,
    });
  }

  function sendSession(res: ServerResponse, session: Session) {
    setRefreshCookie(res, session.refreshToken, session.refreshExpiresAt);
    sendJson(res, 200, {
      accessToken: issueAccessToken(session),
      expiresIn: config.accessTtlSec,
      user: publicUser(session.userId),
    });
  }

  function rejectRefresh(res: ServerResponse, code: string) {
    stats.refreshFailed += 1;
    clearRefreshCookie(res);
    sendJson(res, 401, { code });
    log(`refresh 401 (${code})`);
  }

  function revoke(session: Session, reason: string) {
    if (!session.revokedReason) {
      session.revokedReason = reason;
      log(`session ${session.id} revoked: ${reason}`);
    }
  }

  function sessionFromCookie(req: Request) {
    const token = readCookie(req, COOKIE_NAME);
    const record = token ? refreshTokens.get(token) : undefined;
    return record ? sessions.get(record.sessionId) : undefined;
  }

  /** Middleware auth: chặn trước khi handler chạy (giống backend thật) */
  function authenticate(req: Request, res: ServerResponse, url: URL): Session | null {
    const header = req.headers.authorization ?? "";
    const token = header.startsWith("Bearer ") ? header.slice(7) : "";
    const record = token ? accessTokens.get(token) : undefined;
    const session = record ? sessions.get(record.sessionId) : undefined;

    let code: string | null = null;
    if (!token) code = "ACCESS_TOKEN_MISSING";
    else if (!record || !session) code = "ACCESS_TOKEN_INVALID";
    else if (session.revokedReason) code = "SESSION_REVOKED";
    else if (Date.now() >= record.expiresAt) code = "ACCESS_TOKEN_EXPIRED";

    if (code || !session) {
      stats.api401 += 1;
      res.setHeader("WWW-Authenticate", 'Bearer error="invalid_token"');
      sendJson(res, 401, { code });
      log(`401 ${req.method} ${url.pathname} (${code})`);
      return null;
    }

    stats.apiOk += 1;
    return session;
  }

  function sweepExpired() {
    const now = Date.now();
    if (now - lastSweepAt < 30_000) return;
    lastSweepAt = now;
    for (const [token, record] of accessTokens) {
      if (record.expiresAt < now) accessTokens.delete(token);
    }
    // Giữ RT đã dùng thêm một khoảng để còn phát hiện được dùng lại
    for (const [token, record] of refreshTokens) {
      if (record.expiresAt < now - config.reuseGraceSec * 1000) refreshTokens.delete(token);
    }
  }

  const routes: Record<string, Handler> = {
    "POST /auth/login": async (req, res) => {
      const body = await readJson(req);
      const user = USERS.find((u) => u.username === body.username && u.password === body.password);
      if (!user) {
        sendJson(res, 401, { code: "INVALID_CREDENTIALS" });
        return;
      }

      // Đăng nhập mới thay thế phiên cũ của trình duyệt này
      const previous = sessionFromCookie(req);
      if (previous) revoke(previous, "replaced by new login");

      const session: Session = {
        id: `s_${++sessionSeq}`,
        userId: user.id,
        generation: 0,
        refreshToken: "",
        refreshExpiresAt: 0,
        accessExpiresAt: 0,
        revokedReason: null,
      };
      sessions.set(session.id, session);
      rotateRefreshToken(session);
      stats.login += 1;
      log(`login ${user.username} → session ${session.id}`);
      sendSession(res, session);
    },

    "POST /auth/refresh": (req, res) => {
      const token = readCookie(req, COOKIE_NAME);
      if (!token) {
        stats.refreshFailed += 1;
        sendJson(res, 401, { code: "REFRESH_TOKEN_MISSING" });
        log("refresh 401 (REFRESH_TOKEN_MISSING)");
        return;
      }

      if (failNextRefresh) {
        failNextRefresh = false;
        log("refresh: drop connection mid-response (simulated network error)");
        // Gửi dở response rồi cắt: nếu cắt trước khi có byte nào, trình duyệt sẽ tự retry ngầm
        res.writeHead(200, { "Content-Type": "application/json" });
        res.write('{"accessToken":', () => req.socket.destroy());
        return;
      }

      const record = refreshTokens.get(token);
      const session = record ? sessions.get(record.sessionId) : undefined;
      if (!record || !session) return rejectRefresh(res, "REFRESH_TOKEN_INVALID");
      if (session.revokedReason) return rejectRefresh(res, "SESSION_REVOKED");

      const now = Date.now();
      if (record.usedAt !== undefined) {
        // RT cũ gửi lại trong thời gian ân hạn (vd. response refresh trước bị mất) → trả RT hiện hành
        const current = refreshTokens.get(session.refreshToken);
        if (now - record.usedAt <= config.reuseGraceSec * 1000 && current && now < current.expiresAt) {
          stats.reuseInGrace += 1;
          stats.refreshOk += 1;
          log(`refresh: old token reused within grace → session ${session.id} gen ${session.generation}`);
          sendSession(res, session);
          return;
        }
        stats.reuseDetected += 1;
        revoke(session, "refresh token reuse detected");
        return rejectRefresh(res, "REFRESH_TOKEN_REUSED");
      }

      if (now >= record.expiresAt) return rejectRefresh(res, "REFRESH_TOKEN_EXPIRED");

      record.usedAt = now;
      rotateRefreshToken(session);
      stats.refreshOk += 1;
      log(`refresh ok → session ${session.id} gen ${session.generation}`);
      sendSession(res, session);
    },

    "POST /auth/logout": (req, res) => {
      const session = sessionFromCookie(req);
      if (session) revoke(session, "logout");
      clearRefreshCookie(res);
      res.statusCode = 204;
      res.end();
    },

    "GET /me": async (req, res, url) => {
      const session = authenticate(req, res, url);
      if (!session) return;
      await sleep(Number(url.searchParams.get("delay")) || 0);
      sendJson(res, 200, {
        user: publicUser(session.userId),
        sessionId: session.id,
        serverTime: new Date().toISOString(),
      });
    },

    "POST /answers": async (req, res, url) => {
      const session = authenticate(req, res, url);
      if (!session) return;
      sendJson(res, 200, { saved: true, answer: await readJson(req), savedAt: new Date().toISOString() });
    },

    "GET /__mock/auth": (_req, res) => {
      const session = [...sessions.values()].at(-1);
      sendJson(res, 200, {
        now: Date.now(),
        config,
        stats,
        failNextRefresh,
        session: session && {
          id: session.id,
          userName: publicUser(session.userId).name,
          generation: session.generation,
          revokedReason: session.revokedReason,
          accessExpiresAt: session.accessExpiresAt,
          refreshExpiresAt: session.refreshExpiresAt,
        },
      });
    },

    "PATCH /__mock/auth/config": async (req, res) => {
      applyOptions(config, await readJson(req));
      log(`config → ${JSON.stringify(config)}`);
      sendJson(res, 200, config);
    },

    /** Cho token hết hạn ngay, không cần chờ TTL */
    "POST /__mock/auth/expire": async (req, res) => {
      const { target } = await readJson(req);
      const now = Date.now();
      if (target === "refresh") {
        for (const record of refreshTokens.values()) record.expiresAt = Math.min(record.expiresAt, now);
        for (const session of sessions.values()) session.refreshExpiresAt = Math.min(session.refreshExpiresAt, now);
      } else {
        for (const record of accessTokens.values()) record.expiresAt = Math.min(record.expiresAt, now);
        for (const session of sessions.values()) session.accessExpiresAt = Math.min(session.accessExpiresAt, now);
      }
      log(`expired all ${target === "refresh" ? "refresh" : "access"} tokens`);
      res.statusCode = 204;
      res.end();
    },

    "POST /__mock/auth/fail-next-refresh": (_req, res) => {
      failNextRefresh = true;
      res.statusCode = 204;
      res.end();
    },
  };

  return (req, res, next) => {
    const url = new URL(req.url ?? "/", "http://localhost");
    const handler = routes[`${req.method} ${url.pathname}`];
    if (!handler) return next();

    sweepExpired();
    const isDebugRoute = url.pathname.startsWith("/__mock/");
    sleep(isDebugRoute ? 0 : config.latencyMs)
      .then(() => handler(req, res, url))
      .catch(next);
  };
}

function publicUser(userId: string) {
  const user = USERS.find((u) => u.id === userId);
  return { id: userId, name: user?.name ?? userId };
}

function randomId() {
  return randomBytes(6).toString("base64url");
}

function sleep(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

function sendJson(res: ServerResponse, status: number, body: unknown) {
  res.statusCode = status;
  res.setHeader("Content-Type", "application/json");
  res.end(JSON.stringify(body));
}

function setRefreshCookie(res: ServerResponse, token: string, expiresAt: number) {
  const maxAge = Math.max(Math.ceil((expiresAt - Date.now()) / 1000), 0);
  // Backend thật thêm `Secure`; mock chạy http://localhost nên bỏ qua
  res.setHeader(
    "Set-Cookie",
    `${COOKIE_NAME}=${token}; HttpOnly; SameSite=Strict; Path=${COOKIE_PATH}; Max-Age=${maxAge}`,
  );
}

function clearRefreshCookie(res: ServerResponse) {
  res.setHeader("Set-Cookie", `${COOKIE_NAME}=; HttpOnly; SameSite=Strict; Path=${COOKIE_PATH}; Max-Age=0`);
}

function readCookie(req: IncomingMessage, name: string) {
  for (const part of (req.headers.cookie ?? "").split(";")) {
    const [key, ...value] = part.trim().split("=");
    if (key === name) return decodeURIComponent(value.join("="));
  }
  return undefined;
}

async function readJson(req: IncomingMessage): Promise<Record<string, unknown>> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) chunks.push(chunk as Buffer);
  if (chunks.length === 0) return {};
  try {
    return JSON.parse(Buffer.concat(chunks).toString("utf8"));
  } catch {
    return {};
  }
}

# YouPass — Seamless refresh token (FE)

## Chạy thử với mock API

```bash
bun install
bun dev            # mở http://localhost:5173, đăng nhập student / 123456
```

Mock API (`mock/authMock.ts`) chạy ngay trong dev server của Vite, cookie là cookie thật:

| Endpoint | Mô tả |
|---|---|
| `POST /api/auth/login` | `{ username, password }` → `{ accessToken, expiresIn, user }` + cookie `refresh_token` (httpOnly, `Path=/api/auth`) |
| `POST /api/auth/refresh` | Đọc cookie, xoay vòng refresh token, phát hiện dùng lại (có ân hạn) |
| `POST /api/auth/logout` | Thu hồi phiên, xoá cookie |
| `GET /api/me?delay=ms` · `POST /api/answers` | API cần `Authorization: Bearer` (401 khi hết hạn) |
| `/api/__mock/auth` | Xem trạng thái · `PATCH /config` · `POST /expire {target}` · `POST /fail-next-refresh` |

Thời gian sống mặc định: access token **20s**, refresh token **120s** (trượt theo mỗi lần refresh), độ trễ 300ms.
Đổi bằng env (hoặc `.env.local`): `MOCK_ACCESS_TTL`, `MOCK_REFRESH_TTL`, `MOCK_LATENCY`, `MOCK_REUSE_GRACE`; tắt mock bằng `MOCK_AUTH=off`.
Cũng đổi được lúc đang chạy ngay trên trang playground.

Trang playground (`src/dev/AuthPlayground.tsx`) mô phỏng người dùng đang thao tác và cho giả lập sự cố
(access/refresh token hết hạn ngay, rớt mạng khi refresh). Chỉ số cần theo dõi: **"Lỗi lọt tới UI" phải luôn là 0**.
Mở thêm tab để test đồng bộ giữa các tab.

## Luồng auth

- `src/libs/tokenStore.ts` — access token chỉ nằm trong memory
- `src/helper/auth/` — luồng auth, import qua `index.ts`:
  - `refreshAccessToken.ts` — single-flight + Web Locks, xử lý refresh thất bại
  - `crossTab.ts` — BroadcastChannel và Web Locks giữa các tab
  - `proactiveRefresh.ts` — refresh chủ động trước khi hết hạn, kiểm tra lại khi quay lại tab
  - `session.ts` — lưu, khôi phục và xoá phiên (memory + localStorage)
  - `authActions.ts` — login/logout
- `src/libs/httpRequest.ts` — axios interceptor: gắn Bearer, gặp 401 thì refresh rồi gửi lại request. Refresh token hết hạn thì đăng xuất, app về màn đăng nhập
- `src/AuthProvider.tsx` — khôi phục phiên sau F5, đổi tài khoản thì reload

---

# React + TypeScript + Vite

This template provides a minimal setup to get React working in Vite with HMR and some Oxlint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Oxc](https://oxc.rs)
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/)

## React Compiler

The React Compiler is enabled on this template. See [this documentation](https://react.dev/learn/react-compiler) for more information.

Note: This will impact Vite dev & build performances.
You can also try [the experimental native React Compiler support in plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react/README.md#rust-react-compiler) by using `compiler: true` in the plugin options instead of using the Babel plugin.

## Expanding the Oxlint configuration

If you are developing a production application, we recommend enabling type-aware lint rules by installing `oxlint-tsgolint` and editing `.oxlintrc.json`:

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "plugins": ["react", "typescript", "oxc"],
  "options": {
    "typeAware": true
  },
  "rules": {
    "react/rules-of-hooks": "error",
    "react/only-export-components": ["warn", { "allowConstantExport": true }]
  }
}
```

See the [Oxlint rules documentation](https://oxc.rs/docs/guide/usage/linter/rules) for the full list of rules and categories.

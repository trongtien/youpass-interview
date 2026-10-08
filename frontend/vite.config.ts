import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import babel from '@rolldown/plugin-babel'
import { defineConfig, loadEnv } from 'vite'
import { authMock, authMockOptionsFromEnv } from './mock/authMock.ts'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'MOCK_')

  return {
    plugins: [
      react(),
      babel({ presets: [reactCompilerPreset()] }),
      // Mock /api/auth/* với token sống ngắn; tắt bằng MOCK_AUTH=off khi có backend thật
      env.MOCK_AUTH !== 'off' && authMock(authMockOptionsFromEnv(env)),
    ],
  }
})

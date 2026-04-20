import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'path'

const NotFound404HTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>404 | NTS Gater</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f5f7fa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#333}
.c{text-align:center;padding:40px;max-width:480px}
.s{font-size:72px;font-weight:800;color:#c0c4cc;margin-bottom:0;line-height:1}
.t{font-size:18px;color:#909399;margin-bottom:24px}
.d{font-size:14px;color:#999;margin-bottom:32px;line-height:1.6}
.b{display:inline-block;padding:10px 24px;font-size:14px;color:#fff;text-decoration:none;border-radius:6px;transition:background .2s}
.bp{background:#409eff}.bp:hover{background:#337ecc}
.dv{width:40px;height:2px;background:#e4e7ed;margin:0 auto 16px;border-radius:1px}
</style>
</head>
<body><div class="c"><div class="s">404</div><div class="t">接口未找到</div><div class="d">您请求的 API 资源不存在或后端 API 服务未启动</div><div class="dv"></div><a class="b bp" href="/">返回首页</a></div></body></html>`

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:9090',
        changeOrigin: true,
        // 代理连接失败时返回自定义404（后端未启动）
        onError(err, req, res) {
          if (!res.headersSent) {
            res.statusCode = 404
            res.setHeader('Content-Type', 'text/html; charset=utf-8')
            res.end(NotFound404HTML)
          }
        },
        // 代理成功但后端返回404时，替换为带样式的404页面
        configure(proxy) {
          proxy.on('proxyRes', (proxyRes, req, res) => {
            if (proxyRes.statusCode === 404) {
              // 收集原始响应体，替换为自定义404
              let body = ''
              proxyRes.on('data', (chunk: Buffer) => { body += chunk.toString() })
              proxyRes.on('end', () => {
                if (!res.headersSent) {
                  res.statusCode = 404
                  res.setHeader('Content-Type', 'text/html; charset=utf-8')
                  res.end(NotFound404HTML)
                }
              })
              // 阻止原始响应写入客户端
              proxyRes.headers['content-length'] = '0'
            }
          })
        },
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          'element-plus': ['element-plus'],
          'vue-vendor': ['vue', 'vue-router', 'pinia'],
        },
      },
    },
  },
})

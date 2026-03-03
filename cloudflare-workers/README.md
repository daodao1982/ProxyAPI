# ProxyAPI Cloudflare Workers MVP

给 `ProxyAPI` 的 Cloudflare Worker 最小可用版本（OpenAI 兼容子集）。

## 功能
- `GET /health`
- `GET /v1/models`
- `POST /v1/chat/completions`
- Bearer 入口鉴权
- 双上游快速降级（首选失败自动切备用）

## 部署
```bash
cd cloudflare-workers
npm i
npx wrangler login
```

编辑 `wrangler.toml`：

```toml
[vars]
API_AUTH_TOKEN = "你的入口token"
DEFAULT_MODEL = "gpt-4o-mini"
UPSTREAMS_JSON = '[
  {"name":"up1","baseUrl":"https://api.openai.com/v1","apiKey":"sk-xxx","models":["gpt-4o-mini"]},
  {"name":"up2","baseUrl":"https://api.openai.com/v1","apiKey":"sk-yyy","models":["gpt-4o-mini"]}
]'
```

运行与部署：
```bash
npm run dev
npm run deploy
```

## 调用示例
```bash
curl https://<worker>.workers.dev/v1/models \
  -H "Authorization: Bearer <API_AUTH_TOKEN>"

curl https://<worker>.workers.dev/v1/chat/completions \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

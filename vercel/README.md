# ProxyAPI Vercel MVP

在 Vercel 上部署一个 OpenAI 兼容子集代理。

## 路由
- `GET /api/health`
- `GET /api/v1/models`
- `POST /api/v1/chat/completions`

## 环境变量（Vercel Project Settings -> Environment Variables）
- `API_AUTH_TOKEN`：入口 Bearer Token
- `DEFAULT_MODEL`：默认模型（如 `gpt-4o-mini`）
- `UPSTREAMS_JSON`：上游 JSON 字符串

示例：
```json
[
  {"name":"up1","baseUrl":"https://api.openai.com/v1","apiKey":"sk-xxx","models":["gpt-4o-mini"]},
  {"name":"up2","baseUrl":"https://api.openai.com/v1","apiKey":"sk-yyy","models":["gpt-4o-mini"]}
]
```

## 部署
1. 在 Vercel 导入该仓库。
2. Root Directory 选择 `vercel`。
3. 配置上述 3 个环境变量。
4. Deploy。

## 调用示例
```bash
curl https://<your-vercel-domain>/api/v1/models \
  -H "Authorization: Bearer <API_AUTH_TOKEN>"

curl https://<your-vercel-domain>/api/v1/chat/completions \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

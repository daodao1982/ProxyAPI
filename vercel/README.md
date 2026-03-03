# ProxyAPI Vercel Edition (Full-ish)

Vercel 版本代理网关（OpenAI 兼容子集 + 流式 + 容灾）。

## 路由
- `GET /api`：服务首页
- `GET /api/health`：健康检查
- `GET /api/admin/health`：管理健康（需鉴权）
- `GET /api/v1/models`：模型列表（需鉴权）
- `POST /api/v1/chat/completions`：聊天补全（需鉴权，支持 stream）

## 已实现能力
- ✅ Bearer 鉴权（支持单 token 与多 token）
- ✅ 多上游路由（模型匹配 + 权重选择）
- ✅ 失败自动切换备用上游
- ✅ 上游冷却（429/5xx/超时后短时间降权）
- ✅ 请求超时控制
- ✅ SSE 流式透传
- ✅ 请求追踪头（`x-request-id`、`x-upstream-provider`）
- ✅ 简易管理健康页

## 环境变量（Vercel 项目里配置）
- `API_AUTH_TOKEN`：单入口 token（Bearer）
- `API_AUTH_TOKENS_JSON`：多入口 token（JSON 数组），可与上面并存
- `DEFAULT_MODEL`：默认模型（如 `gpt-4o-mini`）
- `UPSTREAMS_JSON`：上游列表（JSON）
- `MAX_ATTEMPTS`：单次请求最多尝试上游数（默认 2）
- `UPSTREAM_COOLDOWN_MS`：上游失败冷却时长（默认 30000）

### `UPSTREAMS_JSON` 示例
```json
[
  {
    "name": "openai-main",
    "baseUrl": "https://api.openai.com/v1",
    "apiKey": "sk-xxx",
    "models": ["gpt-4o-mini", "gpt-4.1-mini"],
    "weight": 3,
    "timeoutMs": 30000,
    "headers": {"x-foo": "bar"}
  },
  {
    "name": "openai-backup",
    "baseUrl": "https://api.openai.com/v1",
    "apiKey": "sk-yyy",
    "models": ["gpt-4o-mini"],
    "weight": 1,
    "timeoutMs": 25000
  }
]
```

## 部署
1. 在 Vercel 导入仓库。
2. 分支选你的目标分支。
3. Root Directory 选 `vercel`。
4. 配置环境变量。
5. Deploy。

## 调用示例
```bash
# models
curl https://<your-vercel-domain>/api/v1/models \
  -H "Authorization: Bearer <API_AUTH_TOKEN>"

# non-stream
curl https://<your-vercel-domain>/api/v1/chat/completions \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'

# stream
curl https://<your-vercel-domain>/api/v1/chat/completions \
  -N \
  -H "Authorization: Bearer <API_AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hello"}]}'
```

## 备注
- 这是 Vercel 可运维版本，不依赖本地文件存储。
- 下一步可以加：KV/D1 token 池、速率限制、管理 UI、审计日志上报。

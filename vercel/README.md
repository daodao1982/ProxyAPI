# ProxyAPI Vercel Edition (Admin Config UI)

Vercel 版本代理网关（OpenAI 兼容子集 + 流式 + 容灾 + 网页配置）。

## 路由
- `GET /api`：服务首页
- `GET /api/health`：健康检查
- `GET /api/admin/health`：管理健康（需 `ADMIN_TOKEN`）
- `GET /api/admin/config`：读取运行配置（需 `ADMIN_TOKEN`）
- `PUT /api/admin/config`：保存运行配置到 Vercel KV（需 `ADMIN_TOKEN`）
- `GET /api/admin/ui#token=<ADMIN_TOKEN>`：网页配置面板
- `GET /api/v1/models`：模型列表（需网关 token）
- `POST /api/v1/chat/completions`：聊天补全（需网关 token，支持 stream）

## 已实现能力
- ✅ 网关 Bearer 鉴权（支持单 token 与多 token）
- ✅ 管理员鉴权（`ADMIN_TOKEN`）
- ✅ 网页配置后台（读写配置）
- ✅ 配置持久化到 Vercel KV（未配置 KV 时回退到环境变量）
- ✅ 多上游路由（模型匹配 + 权重选择）
- ✅ 失败自动切换备用上游
- ✅ 上游冷却（429/5xx/超时后短时间降权）
- ✅ 请求超时控制
- ✅ SSE 流式透传
- ✅ 请求追踪头（`x-request-id`、`x-upstream-provider`）

## 环境变量（Vercel 项目里配置）
### 必填
- `API_AUTH_TOKEN`：网关入口 token（Bearer）
- `ADMIN_TOKEN`：管理后台 token（Bearer）

### 可选
- `API_AUTH_TOKENS_JSON`：多入口 token（JSON 数组）
- `DEFAULT_MODEL`：默认模型（如 `gpt-4o-mini`）
- `UPSTREAMS_JSON`：上游列表（作为 KV 未配置时的 fallback）
- `MAX_ATTEMPTS`：默认尝试上游数
- `UPSTREAM_COOLDOWN_MS`：默认上游冷却时长

### KV（用于网页配置持久化）
在 Vercel Marketplace/Add Integration 接入 KV 后，会自动注入：
- `KV_REST_API_URL`
- `KV_REST_API_TOKEN`

## `UPSTREAMS_JSON` 示例
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
2. Root Directory 选 `vercel`。
3. 先配置 `API_AUTH_TOKEN` + `ADMIN_TOKEN`。
4. 建议接入 Vercel KV（让网页配置可持久化）。
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

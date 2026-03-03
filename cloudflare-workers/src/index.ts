interface Upstream {
  name: string;
  baseUrl: string;
  apiKey: string;
  models?: string[];
}

interface Env {
  API_AUTH_TOKEN: string;
  DEFAULT_MODEL?: string;
  UPSTREAMS_JSON: string;
}

const json = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "access-control-allow-origin": "*",
      "access-control-allow-headers": "authorization,content-type",
      "access-control-allow-methods": "GET,POST,OPTIONS",
    },
  });

function parseUpstreams(raw: string): Upstream[] {
  try {
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr)) return [];
    return arr.filter((x) => x?.name && x?.baseUrl && x?.apiKey);
  } catch {
    return [];
  }
}

function checkAuth(req: Request, env: Env): boolean {
  return (req.headers.get("authorization") || "") === `Bearer ${env.API_AUTH_TOKEN}`;
}

function pick(model: string, ups: Upstream[]): Upstream | null {
  const matched = ups.filter((u) => !u.models?.length || u.models.includes(model));
  const arr = matched.length ? matched : ups;
  if (!arr.length) return null;
  return arr[Math.floor(Math.random() * arr.length)];
}

async function forwardChat(req: Request, env: Env): Promise<Response> {
  if (!checkAuth(req, env)) return json({ error: "unauthorized" }, 401);

  const body = await req.json<any>().catch(() => null);
  if (!body) return json({ error: "invalid_json" }, 400);

  const model = body.model || env.DEFAULT_MODEL || "gpt-4o-mini";
  body.model = model;

  const ups = parseUpstreams(env.UPSTREAMS_JSON);
  if (!ups.length) return json({ error: "no_upstream_configured" }, 500);

  const first = pick(model, ups);
  if (!first) return json({ error: "no_upstream_available" }, 503);

  const candidates = [first, ...ups.filter((u) => u !== first)].slice(0, 2);
  let last = "unknown";

  for (const up of candidates) {
    try {
      const resp = await fetch(`${up.baseUrl.replace(/\/$/, "")}/chat/completions`, {
        method: "POST",
        headers: {
          "content-type": "application/json",
          authorization: `Bearer ${up.apiKey}`,
        },
        body: JSON.stringify(body),
      });

      if (!resp.ok) {
        last = `${up.name}:${resp.status}`;
        continue;
      }

      return new Response(resp.body, {
        status: resp.status,
        headers: {
          "content-type": resp.headers.get("content-type") || "application/json",
          "access-control-allow-origin": "*",
        },
      });
    } catch {
      last = `${up.name}:network_error`;
    }
  }

  return json({ error: "upstream_failed", detail: last }, 502);
}

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    if (req.method === "OPTIONS") return json({ ok: true }, 204);
    const url = new URL(req.url);

    if (url.pathname === "/health") return json({ ok: true, ts: Date.now() });

    if (url.pathname === "/v1/models" && req.method === "GET") {
      if (!checkAuth(req, env)) return json({ error: "unauthorized" }, 401);
      const ups = parseUpstreams(env.UPSTREAMS_JSON);
      const models = new Set<string>();
      for (const u of ups) (u.models || []).forEach((m) => models.add(m));
      if (!models.size && env.DEFAULT_MODEL) models.add(env.DEFAULT_MODEL);
      return json({
        object: "list",
        data: Array.from(models).map((id) => ({ id, object: "model", owned_by: "proxyapi-cf" })),
      });
    }

    if (url.pathname === "/v1/chat/completions" && req.method === "POST") {
      return forwardChat(req, env);
    }

    return json({ error: "not_found" }, 404);
  },
};

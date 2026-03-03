const upstreamCooldown = new Map();

function now() {
  return Date.now();
}

export function json(res, status, data, extraHeaders = {}) {
  res.status(status);
  res.setHeader('content-type', 'application/json; charset=utf-8');
  res.setHeader('access-control-allow-origin', '*');
  res.setHeader('access-control-allow-headers', 'authorization,content-type,x-request-id');
  res.setHeader('access-control-allow-methods', 'GET,POST,OPTIONS');
  for (const [k, v] of Object.entries(extraHeaders)) {
    res.setHeader(k, v);
  }
  res.send(JSON.stringify(data));
}

export function corsPreflight(req, res) {
  if (req.method === 'OPTIONS') {
    return json(res, 204, { ok: true });
  }
  return null;
}

export function requestId(req) {
  return req.headers['x-request-id'] || `req_${Math.random().toString(36).slice(2, 10)}_${Date.now()}`;
}

function parseJsonSafe(raw, fallback) {
  try {
    const v = JSON.parse(raw);
    return v;
  } catch {
    return fallback;
  }
}

function normalizeModelList(models) {
  if (!Array.isArray(models)) return [];
  return models.filter((m) => typeof m === 'string' && m.trim()).map((m) => m.trim());
}

export function parseGatewayTokens() {
  const single = (process.env.API_AUTH_TOKEN || '').trim();
  const multiRaw = (process.env.API_AUTH_TOKENS_JSON || '').trim();
  const arr = multiRaw ? parseJsonSafe(multiRaw, []) : [];

  const set = new Set();
  if (single) set.add(single);
  if (Array.isArray(arr)) {
    for (const t of arr) {
      if (typeof t === 'string' && t.trim()) set.add(t.trim());
    }
  }
  return set;
}

export function checkAuth(req) {
  const tokens = parseGatewayTokens();
  if (!tokens.size) return false;
  const auth = req.headers.authorization || '';
  if (!auth.startsWith('Bearer ')) return false;
  const token = auth.slice(7).trim();
  return tokens.has(token);
}

export function parseUpstreams() {
  const raw = process.env.UPSTREAMS_JSON || '[]';
  const parsed = parseJsonSafe(raw, []);
  if (!Array.isArray(parsed)) return [];

  return parsed
    .map((x) => {
      if (!x || !x.name || !x.baseUrl || !x.apiKey) return null;
      return {
        name: String(x.name),
        baseUrl: String(x.baseUrl).replace(/\/$/, ''),
        apiKey: String(x.apiKey),
        models: normalizeModelList(x.models),
        weight: Number.isFinite(Number(x.weight)) ? Math.max(1, Number(x.weight)) : 1,
        timeoutMs: Number.isFinite(Number(x.timeoutMs)) ? Math.max(2000, Number(x.timeoutMs)) : 30000,
        headers: x.headers && typeof x.headers === 'object' ? x.headers : {},
      };
    })
    .filter(Boolean);
}

function availableUpstreams(model, all, cooldownMs) {
  const t = now();
  const modelMatched = all.filter((u) => !u.models.length || u.models.includes(model));
  const pool = modelMatched.length ? modelMatched : all;
  return pool.filter((u) => {
    const blockedUntil = upstreamCooldown.get(u.name) || 0;
    return blockedUntil <= t || blockedUntil - t < cooldownMs / 2;
  });
}

function weightedPick(list) {
  if (!list.length) return null;
  const total = list.reduce((s, u) => s + (u.weight || 1), 0);
  let r = Math.random() * total;
  for (const u of list) {
    r -= u.weight || 1;
    if (r <= 0) return u;
  }
  return list[list.length - 1];
}

function sortCandidates(model, ups) {
  const cooldownMs = Number(process.env.UPSTREAM_COOLDOWN_MS || 30000);
  const avail = availableUpstreams(model, ups, cooldownMs);
  const pool = avail.length ? avail : ups;

  const first = weightedPick(pool);
  if (!first) return [];

  const rest = pool.filter((u) => u.name !== first.name);
  rest.sort((a, b) => (b.weight || 1) - (a.weight || 1));

  const maxAttempts = Math.max(1, Number(process.env.MAX_ATTEMPTS || 2));
  return [first, ...rest].slice(0, maxAttempts);
}

function normalizeBody(inputBody) {
  if (inputBody && typeof inputBody === 'object' && !Array.isArray(inputBody)) return { ...inputBody };
  return {};
}

export async function forwardChat(inputBody, opts = {}) {
  const body = normalizeBody(inputBody);
  const model = body.model || process.env.DEFAULT_MODEL || 'gpt-4o-mini';
  body.model = model;

  const ups = parseUpstreams();
  if (!ups.length) return { status: 500, data: { error: 'no_upstream_configured' } };

  const candidates = sortCandidates(model, ups);
  if (!candidates.length) return { status: 503, data: { error: 'no_upstream_available' } };

  const requestIdValue = opts.requestId || `r_${Date.now()}`;
  let lastErr = 'unknown';

  for (const up of candidates) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), up.timeoutMs || 30000);

    try {
      const resp = await fetch(`${up.baseUrl}/chat/completions`, {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
          authorization: `Bearer ${up.apiKey}`,
          'x-request-id': requestIdValue,
          ...up.headers,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      clearTimeout(timer);

      if (!resp.ok) {
        lastErr = `${up.name}:${resp.status}`;
        if (resp.status >= 500 || resp.status === 429) {
          upstreamCooldown.set(up.name, now() + Number(process.env.UPSTREAM_COOLDOWN_MS || 30000));
        }
        continue;
      }

      const isStream = String(resp.headers.get('content-type') || '').includes('text/event-stream') || body.stream === true;
      if (isStream) {
        return {
          status: resp.status,
          stream: resp.body,
          contentType: resp.headers.get('content-type') || 'text/event-stream',
          provider: up.name,
        };
      }

      const text = await resp.text();
      return {
        status: resp.status,
        raw: text,
        contentType: resp.headers.get('content-type') || 'application/json',
        provider: up.name,
      };
    } catch (e) {
      clearTimeout(timer);
      lastErr = `${up.name}:${e?.name === 'AbortError' ? 'timeout' : 'network_error'}`;
      upstreamCooldown.set(up.name, now() + Number(process.env.UPSTREAM_COOLDOWN_MS || 30000));
    }
  }

  return { status: 502, data: { error: 'upstream_failed', detail: lastErr } };
}

export function buildModelList() {
  const ups = parseUpstreams();
  const models = new Map();
  for (const up of ups) {
    for (const m of up.models || []) {
      if (!models.has(m)) models.set(m, []);
      models.get(m).push(up.name);
    }
  }

  if (!models.size && process.env.DEFAULT_MODEL) {
    models.set(process.env.DEFAULT_MODEL, ['default']);
  }

  return Array.from(models.entries()).map(([id, providers]) => ({
    id,
    object: 'model',
    owned_by: 'proxyapi-vercel',
    metadata: { providers },
  }));
}

export function adminHealth() {
  const ups = parseUpstreams();
  const cooldown = {};
  const t = now();
  for (const up of ups) {
    const until = upstreamCooldown.get(up.name) || 0;
    cooldown[up.name] = until > t ? until - t : 0;
  }

  return {
    ok: true,
    upstreamCount: ups.length,
    defaults: {
      defaultModel: process.env.DEFAULT_MODEL || 'gpt-4o-mini',
      maxAttempts: Number(process.env.MAX_ATTEMPTS || 2),
      cooldownMs: Number(process.env.UPSTREAM_COOLDOWN_MS || 30000),
    },
    cooldown,
  };
}

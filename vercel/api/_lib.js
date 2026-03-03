export function json(res, status, data) {
  res.status(status).setHeader('content-type', 'application/json; charset=utf-8');
  res.setHeader('access-control-allow-origin', '*');
  res.setHeader('access-control-allow-headers', 'authorization,content-type');
  res.setHeader('access-control-allow-methods', 'GET,POST,OPTIONS');
  res.send(JSON.stringify(data));
}

export function checkAuth(req) {
  const token = process.env.API_AUTH_TOKEN || '';
  return req.headers.authorization === `Bearer ${token}`;
}

export function parseUpstreams() {
  try {
    const raw = process.env.UPSTREAMS_JSON || '[]';
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr)) return [];
    return arr.filter((x) => x && x.name && x.baseUrl && x.apiKey);
  } catch {
    return [];
  }
}

export function pickUpstream(model, ups) {
  const matched = ups.filter((u) => !u.models || u.models.length === 0 || u.models.includes(model));
  const arr = matched.length ? matched : ups;
  if (!arr.length) return null;
  return arr[Math.floor(Math.random() * arr.length)];
}

export async function forwardChat(body) {
  const model = body.model || process.env.DEFAULT_MODEL || 'gpt-4o-mini';
  body.model = model;

  const ups = parseUpstreams();
  if (!ups.length) return { status: 500, data: { error: 'no_upstream_configured' } };

  const first = pickUpstream(model, ups);
  if (!first) return { status: 503, data: { error: 'no_upstream_available' } };

  const candidates = [first, ...ups.filter((u) => u !== first)].slice(0, 2);
  let lastErr = 'unknown';

  for (const up of candidates) {
    try {
      const resp = await fetch(`${up.baseUrl.replace(/\/$/, '')}/chat/completions`, {
        method: 'POST',
        headers: {
          'content-type': 'application/json',
          authorization: `Bearer ${up.apiKey}`,
        },
        body: JSON.stringify(body),
      });

      const text = await resp.text();
      if (!resp.ok) {
        lastErr = `${up.name}:${resp.status}`;
        continue;
      }

      return {
        status: resp.status,
        raw: text,
        contentType: resp.headers.get('content-type') || 'application/json',
      };
    } catch {
      lastErr = `${up.name}:network_error`;
    }
  }

  return { status: 502, data: { error: 'upstream_failed', detail: lastErr } };
}

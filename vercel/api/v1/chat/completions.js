import { json, checkAuth, forwardChat, corsPreflight, requestId } from '../../_lib.js';

export const config = {
  api: {
    bodyParser: true,
  },
};

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;

  if (req.method !== 'POST') return json(res, 405, { error: 'method_not_allowed' });
  if (!checkAuth(req)) return json(res, 401, { error: 'unauthorized' });

  const rid = requestId(req);
  const body = req.body && typeof req.body === 'object' ? req.body : {};

  const out = await forwardChat(body, { requestId: rid });

  if (out.stream) {
    res.status(out.status);
    res.setHeader('content-type', out.contentType || 'text/event-stream');
    res.setHeader('cache-control', 'no-cache, no-transform');
    res.setHeader('connection', 'keep-alive');
    res.setHeader('x-request-id', rid);
    res.setHeader('x-upstream-provider', out.provider || 'unknown');

    const reader = out.stream.getReader();
    const decoder = new TextDecoder();

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        res.write(decoder.decode(value, { stream: true }));
      }
    } finally {
      res.end();
    }
    return;
  }

  if (out.raw) {
    res.status(out.status);
    res.setHeader('content-type', out.contentType || 'application/json');
    res.setHeader('access-control-allow-origin', '*');
    res.setHeader('x-request-id', rid);
    res.setHeader('x-upstream-provider', out.provider || 'unknown');
    return res.send(out.raw);
  }

  return json(res, out.status, out.data || { error: 'unknown' }, { 'x-request-id': rid });
}

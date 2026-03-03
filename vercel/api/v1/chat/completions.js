import { json, checkAuth, forwardChat } from '../../_lib.js';

export default async function handler(req, res) {
  if (req.method === 'OPTIONS') return json(res, 204, { ok: true });
  if (req.method !== 'POST') return json(res, 405, { error: 'method_not_allowed' });

  if (!checkAuth(req)) return json(res, 401, { error: 'unauthorized' });

  const body = req.body && typeof req.body === 'object' ? req.body : {};
  const out = await forwardChat(body);

  if (out.raw) {
    res.status(out.status);
    res.setHeader('content-type', out.contentType || 'application/json');
    res.setHeader('access-control-allow-origin', '*');
    return res.send(out.raw);
  }

  return json(res, out.status, out.data || { error: 'unknown' });
}

import { json, checkAuth, parseUpstreams } from '../_lib.js';

export default async function handler(req, res) {
  if (req.method === 'OPTIONS') return json(res, 204, { ok: true });
  if (req.method !== 'GET') return json(res, 405, { error: 'method_not_allowed' });

  if (!checkAuth(req)) return json(res, 401, { error: 'unauthorized' });

  const ups = parseUpstreams();
  const models = new Set();
  for (const u of ups) (u.models || []).forEach((m) => models.add(m));
  if (!models.size && process.env.DEFAULT_MODEL) models.add(process.env.DEFAULT_MODEL);

  return json(res, 200, {
    object: 'list',
    data: Array.from(models).map((id) => ({ id, object: 'model', owned_by: 'proxyapi-vercel' })),
  });
}

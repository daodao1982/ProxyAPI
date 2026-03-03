import { json } from './_lib.js';

export default async function handler(req, res) {
  if (req.method === 'OPTIONS') return json(res, 204, { ok: true });
  return json(res, 200, {
    name: 'ProxyAPI Vercel Edition',
    status: 'running',
    endpoints: ['/api/health', '/api/v1/models', '/api/v1/chat/completions', '/api/admin/health'],
  });
}

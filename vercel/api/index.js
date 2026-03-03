import { json, corsPreflight } from './_lib.js';

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;

  return json(res, 200, {
    name: 'ProxyAPI Vercel Edition',
    status: 'running',
    endpoints: [
      '/api/health',
      '/api/v1/models',
      '/api/v1/chat/completions',
      '/api/admin/health',
      '/api/admin/config',
      '/api/admin/ui#token=<ADMIN_TOKEN>',
    ],
  });
}

import { json, checkAdmin, corsPreflight, loadRuntimeConfig, saveRuntimeConfig } from '../_lib.js';

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;

  if (!checkAdmin(req)) return json(res, 401, { error: 'unauthorized' });

  if (req.method === 'GET') {
    return json(res, 200, await loadRuntimeConfig());
  }

  if (req.method === 'PUT') {
    const body = req.body && typeof req.body === 'object' ? req.body : {};
    const saved = await saveRuntimeConfig(body);
    if (!saved.ok) {
      return json(res, 400, {
        error: 'kv_not_configured',
        message: 'Vercel KV 未配置，无法网页持久化。请先在项目里添加 KV。',
        preview: saved.config,
      });
    }
    return json(res, 200, { ok: true, config: saved.config });
  }

  return json(res, 405, { error: 'method_not_allowed' });
}

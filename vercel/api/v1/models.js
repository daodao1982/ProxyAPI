import { json, checkAuth, buildModelList, corsPreflight } from '../_lib.js';

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;

  if (req.method !== 'GET') return json(res, 405, { error: 'method_not_allowed' });
  if (!checkAuth(req)) return json(res, 401, { error: 'unauthorized' });

  return json(res, 200, {
    object: 'list',
    data: await buildModelList(),
  });
}

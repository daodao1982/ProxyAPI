import { json, checkAdmin, adminHealth, corsPreflight } from '../_lib.js';

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;

  if (!checkAdmin(req)) return json(res, 401, { error: 'unauthorized' });
  return json(res, 200, await adminHealth());
}

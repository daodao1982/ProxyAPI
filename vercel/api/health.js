import { json } from './_lib.js';

export default async function handler(req, res) {
  if (req.method === 'OPTIONS') return json(res, 204, { ok: true });
  return json(res, 200, { ok: true, ts: Date.now() });
}

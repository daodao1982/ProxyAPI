import { json, corsPreflight } from './_lib.js';

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;
  return json(res, 200, { ok: true, ts: Date.now() });
}

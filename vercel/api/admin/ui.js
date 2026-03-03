import { corsPreflight, checkAdmin } from '../_lib.js';

function html(body) {
  return `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1" />
<title>ProxyAPI Admin</title>
<style>
body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Arial,sans-serif;background:#0b1020;color:#e8ecff;margin:0;padding:24px}
.card{max-width:980px;margin:0 auto;background:#121937;border:1px solid #27305f;border-radius:14px;padding:20px}
input,textarea,button{font:inherit}
input,textarea{width:100%;box-sizing:border-box;background:#0b1020;color:#e8ecff;border:1px solid #30408a;border-radius:8px;padding:10px}
textarea{min-height:260px}
.row{display:flex;gap:12px}
.row>div{flex:1}
button{background:#4f7cff;color:#fff;border:0;border-radius:8px;padding:10px 14px;cursor:pointer}
.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
.ok{color:#5dffb0}.err{color:#ff8a8a}
</style></head><body>${body}</body></html>`;
}

export default async function handler(req, res) {
  const pre = corsPreflight(req, res);
  if (pre) return;

  if (!checkAdmin(req)) {
    res.status(401).setHeader('content-type', 'text/html; charset=utf-8');
    return res.send(html('<div class="card"><h2>401 Unauthorized</h2><p>请在 Header 里带 Authorization: Bearer ADMIN_TOKEN</p></div>'));
  }

  const page = html(`
<div class="card">
  <h2>🎀 ProxyAPI Admin Config</h2>
  <p>通过网页配置运行参数（需先配置 Vercel KV 才能持久化）。</p>
  <div class="row">
    <div><label>defaultModel</label><input id="defaultModel" /></div>
    <div><label>maxAttempts</label><input id="maxAttempts" type="number" /></div>
    <div><label>cooldownMs</label><input id="cooldownMs" type="number" /></div>
  </div>
  <p>upstreams (JSON array)</p>
  <textarea id="upstreams" class="mono"></textarea>
  <p><button id="loadBtn">加载</button> <button id="saveBtn">保存</button></p>
  <pre id="msg" class="mono"></pre>
</div>
<script>
const msg = document.getElementById('msg');
const auth = { Authorization: location.hash.startsWith('#token=') ? 'Bearer ' + decodeURIComponent(location.hash.slice(7)) : '' };

async function load(){
  const r = await fetch('/api/admin/config',{headers:auth});
  const j = await r.json();
  if(!r.ok){ msg.textContent = '加载失败: ' + JSON.stringify(j,null,2); return; }
  defaultModel.value = j.defaultModel || '';
  maxAttempts.value = j.maxAttempts || 2;
  cooldownMs.value = j.cooldownMs || 30000;
  upstreams.value = JSON.stringify(j.upstreams || [], null, 2);
  msg.innerHTML = '<span class="ok">已加载</span>';
}

async function save(){
  let up=[];
  try{ up = JSON.parse(upstreams.value || '[]'); }catch(e){ msg.innerHTML='<span class="err">upstreams JSON 格式错误</span>'; return; }
  const payload = {
    defaultModel: defaultModel.value,
    maxAttempts: Number(maxAttempts.value || 2),
    cooldownMs: Number(cooldownMs.value || 30000),
    upstreams: up,
  };
  const r = await fetch('/api/admin/config',{method:'PUT',headers:{'content-type':'application/json',...auth},body:JSON.stringify(payload)});
  const j = await r.json();
  if(!r.ok){ msg.textContent = '保存失败: ' + JSON.stringify(j,null,2); return; }
  msg.innerHTML = '<span class="ok">保存成功</span>\n' + JSON.stringify(j,null,2);
}

document.getElementById('loadBtn').onclick = load;
document.getElementById('saveBtn').onclick = save;
load();
</script>`);

  res.status(200).setHeader('content-type', 'text/html; charset=utf-8');
  res.send(page);
}

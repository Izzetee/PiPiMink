package server

// adminHTML is the admin frontend served at /admin.
const adminHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>PiPiMink Admin</title>
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, -apple-system, sans-serif; background: #f3f4f6; color: #111; min-height: 100vh; }
header { background: #1e293b; color: #f8fafc; padding: 14px 24px; display: flex; align-items: center; gap: 12px; }
header h1 { font-size: 18px; font-weight: 600; letter-spacing: .02em; }
header span { font-size: 13px; color: #94a3b8; }
main { max-width: 1300px; margin: 24px auto; padding: 0 16px; display: flex; flex-direction: column; gap: 16px; }

.card { background: #fff; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,.08); padding: 20px; }
.card h2 { font-size: 15px; font-weight: 600; margin-bottom: 14px; color: #1e293b; }

.toolbar { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
.toolbar label { font-size: 13px; color: #475569; white-space: nowrap; }
input[type=password], input[type=text] {
  padding: 7px 11px; border: 1px solid #cbd5e1; border-radius: 6px;
  font-size: 13px; outline: none; width: 280px; transition: border .15s;
}
input:focus { border-color: #3b82f6; }

button {
  padding: 7px 16px; border: none; border-radius: 6px; font-size: 13px;
  font-weight: 500; cursor: pointer; transition: opacity .15s, background .15s;
}
button:disabled { opacity: .45; cursor: not-allowed; }
.btn-blue   { background: #2563eb; color: #fff; }
.btn-green  { background: #16a34a; color: #fff; }
.btn-amber  { background: #d97706; color: #fff; }
.btn-ghost  { background: #e2e8f0; color: #334155; }
button:not(:disabled):hover { opacity: .88; }

#toast {
  display: none; padding: 10px 14px; border-radius: 6px; font-size: 13px;
  margin-bottom: 4px;
}
.toast-info    { background: #dbeafe; color: #1d4ed8; }
.toast-success { background: #dcfce7; color: #15803d; }
.toast-error   { background: #fee2e2; color: #b91c1c; }

.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { background: #f8fafc; padding: 9px 12px; text-align: left; font-weight: 600;
     color: #475569; border-bottom: 1px solid #e2e8f0; white-space: nowrap; }
td { padding: 8px 12px; border-bottom: 1px solid #f1f5f9; }
tr:last-child td { border-bottom: none; }
tr:hover td { background: #f8fafc; }

.badge {
  display: inline-block; padding: 2px 8px; border-radius: 20px;
  font-size: 11px; font-weight: 600; letter-spacing: .03em;
}
.b-enabled    { background: #dcfce7; color: #15803d; }
.b-discovered { background: #fef9c3; color: #a16207; }
.b-disabled   { background: #fee2e2; color: #b91c1c; }

.stats { display: flex; gap: 16px; flex-wrap: wrap; }
.stat { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px;
        padding: 10px 18px; text-align: center; min-width: 90px; }
.stat-num  { font-size: 22px; font-weight: 700; color: #1e293b; }
.stat-lbl  { font-size: 11px; color: #64748b; margin-top: 2px; }

.sel-bar { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; font-size: 13px; color: #475569; flex-wrap: wrap; }
.filter-btn { padding: 4px 12px; border: 1px solid #cbd5e1; border-radius: 20px; background: #fff; font-size: 12px; cursor: pointer; color: #475569; }
.filter-btn.active { background: #1e293b; color: #fff; border-color: #1e293b; }

/* toggle switch */
.toggle { position: relative; display: inline-block; width: 36px; height: 20px; }
.toggle input { opacity: 0; width: 0; height: 0; }
.slider-track { position: absolute; inset: 0; background: #cbd5e1; border-radius: 20px; cursor: pointer; transition: background .2s; }
.slider-track::before { content: ''; position: absolute; width: 14px; height: 14px; left: 3px; top: 3px; background: #fff; border-radius: 50%; transition: transform .2s; }
input:checked + .slider-track { background: #16a34a; }
input:checked + .slider-track::before { transform: translateX(16px); }
</style>
</head>
<body>
<header>
  <h1>PiPiMink Admin</h1>
  <span>Model discovery &amp; capability management</span>
</header>

<main>
  <!-- Toolbar -->
  <div class="card">
    <div class="toolbar">
      <label>Admin API Key</label>
      <input type="password" id="api-key" placeholder="X-API-Key">
      <button class="btn-blue"  id="btn-discover" onclick="discoverModels()">Discover Models</button>
      <button class="btn-green" id="btn-tag"      onclick="tagSelected()">Tag Selected</button>
      <button class="btn-amber" id="btn-bench"    onclick="benchmarkSelected()">Benchmark Selected</button>
      <button class="btn-ghost" onclick="loadModels()">Refresh</button>
    </div>
    <div id="toast" style="margin-top:12px"></div>
  </div>

  <!-- Stats -->
  <div class="stats" id="stats"></div>

  <!-- Model table -->
  <div class="card">
    <h2>Models</h2>
    <div class="sel-bar">
      <span style="font-weight:600;color:#1e293b">Filter:</span>
      <button class="filter-btn active" id="f-all"      onclick="setFilter('all')">All</button>
      <button class="filter-btn"        id="f-enabled"  onclick="setFilter('enabled')">Enabled</button>
      <button class="filter-btn"        id="f-disabled" onclick="setFilter('disabled')">Disabled</button>
      <span style="width:1px;background:#e2e8f0;height:18px;margin:0 4px"></span>
      <label><input type="checkbox" id="chk-all-tag"   onchange="toggleAll('tag',   this.checked)"> Select all (Tag)</label>
      <label><input type="checkbox" id="chk-all-bench" onchange="toggleAll('bench', this.checked)"> Select all (Benchmark)</label>
      <span id="sel-count" style="margin-left:auto"></span>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>On/Off</th>
            <th>Tag</th>
            <th>Benchmark</th>
            <th>Model</th>
            <th>Provider</th>
            <th>Status</th>
            <th>Reasoning</th>
            <th>Benchmarks</th>
            <th>Avg Response</th>
            <th>Updated</th>
          </tr>
        </thead>
        <tbody id="tbody">
          <tr><td colspan="10" style="text-align:center;padding:40px;color:#94a3b8">
            Click <strong>Discover Models</strong> or <strong>Refresh</strong> to load models.
          </td></tr>
        </tbody>
      </table>
    </div>
  </div>
</main>

<script>
const KEY_STORE = 'pipimink-admin-key';
let currentFilter = 'all';
let allModels = [];

function setFilter(f) {
  currentFilter = f;
  ['all','enabled','disabled'].forEach(id =>
    document.getElementById('f-' + id).classList.toggle('active', id === f));
  renderModels(allModels);
}

function applyFilter(m) {
  if (currentFilter === 'enabled')  return m.enabled;
  if (currentFilter === 'disabled') return !m.enabled;
  return true;
}

async function setEnabled(checkbox, name, source) {
  const enabled = checkbox.checked;
  try {
    const res = await fetch(` + "`" + `/models/${encodeURIComponent(name)}/enable` + "`" + `, {
      method: 'PATCH',
      headers: { 'X-API-Key': key(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ source, enabled }),
    });
    if (!res.ok) {
      checkbox.checked = !enabled; // revert
      const data = await safeJson(res);
      toast(data.error || res.statusText || 'Failed to update model', 'error');
      return;
    }
    // Update local state so filter stays consistent without a full reload.
    const m = allModels.find(m => m.name === name && m.source === source);
    if (m) m.enabled = enabled;
    toast(` + "`" + `${name} ${enabled ? 'enabled' : 'disabled'}` + "`" + `, 'success');
    renderStats(allModels);
    updateSelCount();
  } catch (e) {
    checkbox.checked = !enabled;
    toast('Failed: ' + e.message, 'error');
  }
}
document.addEventListener('DOMContentLoaded', () => {
  const saved = sessionStorage.getItem(KEY_STORE);
  if (saved) document.getElementById('api-key').value = saved;
  loadModels();
});

function key() {
  const k = document.getElementById('api-key').value.trim();
  sessionStorage.setItem(KEY_STORE, k);
  return k;
}

function toast(msg, type = 'info') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast-' + type;
  el.style.display = 'block';
}

async function safeJson(res) {
  try { return await res.json(); } catch { return {}; }
}

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, c =>
    ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

async function loadModels() {
  try {
    const res = await fetch('/models');
    const data = await res.json();
    allModels = data.models || [];
    renderModels(allModels);
  } catch (e) {
    toast('Failed to load models: ' + e.message, 'error');
  }
}

function latencyCell(ms) {
  if (ms == null) return '<span style="color:#94a3b8;font-size:12px">—</span>';
  let color, label;
  if (ms < 1000)       { color = '#15803d'; label = ms + ' ms'; }
  else if (ms < 5000)  { color = '#a16207'; label = (ms/1000).toFixed(1) + ' s'; }
  else                 { color = '#b91c1c'; label = (ms/1000).toFixed(1) + ' s'; }
  return ` + "`" + `<span style="font-size:12px;font-weight:600;color:${color}">${label}</span>` + "`" + `;
}

function benchmarkCell(scores) {
  if (!scores || !Object.keys(scores).length) return '<span style="color:#94a3b8;font-size:12px">none</span>';
  const cats = Object.keys(scores).sort();
  const pills = cats.map(c => {
    const pct = Math.round(scores[c] * 100);
    const color = pct >= 70 ? '#15803d' : pct >= 40 ? '#a16207' : '#b91c1c';
    return ` + "`" + `<span style="font-size:11px;background:#f1f5f9;border-radius:4px;padding:1px 6px;color:${color};margin-right:3px" title="${c}">${c.split('-')[0]} ${pct}%</span>` + "`" + `;
  }).join('');
  return pills;
}

function statusBadge(m) {
  if (m.tagged && m.enabled)  return '<span class="badge b-enabled">tagged</span>';
  if (m.tagged && !m.enabled) return '<span class="badge b-disabled">disabled</span>';
  return '<span class="badge b-discovered">discovered</span>';
}

function renderModels(models) {
  const tbody = document.getElementById('tbody');
  if (!models.length) {
    tbody.innerHTML = '<tr><td colspan="10" style="text-align:center;padding:40px;color:#94a3b8">No models found.</td></tr>';
    renderStats([]);
    return;
  }
  models.sort((a, b) => (a.source + a.name).localeCompare(b.source + b.name));
  const visible = models.filter(applyFilter);
  tbody.innerHTML = visible.length ? visible.map(m => ` + "`" + `
    <tr>
      <td>
        <label class="toggle">
          <input type="checkbox" ${m.enabled ? 'checked' : ''}
            onchange="setEnabled(this, '${esc(m.name)}', '${esc(m.source)}')">
          <span class="slider-track"></span>
        </label>
      </td>
      <td><input type="checkbox" class="chk-tag"   data-name="${esc(m.name)}" data-source="${esc(m.source)}"></td>
      <td><input type="checkbox" class="chk-bench" data-name="${esc(m.name)}" data-source="${esc(m.source)}"></td>
      <td>${esc(m.name)}</td>
      <td>${esc(m.source)}</td>
      <td>${statusBadge(m)}</td>
      <td>${m.hasReasoning ? '<span class="badge b-enabled">yes</span>' : ''}</td>
      <td>${benchmarkCell(m.benchmarkScores)}</td>
      <td>${latencyCell(m.avgLatencyMs)}</td>
      <td>${m.updatedAt ? new Date(m.updatedAt).toLocaleString() : '—'}</td>
    </tr>
  ` + "`" + `).join('') :
  ` + "`" + `<tr><td colspan="10" style="text-align:center;padding:32px;color:#94a3b8">No models match the current filter.</td></tr>` + "`" + `;
  renderStats(models);
  updateSelCount();
  document.querySelectorAll('.chk-tag, .chk-bench').forEach(c =>
    c.addEventListener('change', updateSelCount));
}

function renderStats(models) {
  const total      = models.length;
  const tagged     = models.filter(m => m.tagged && m.enabled).length;
  const disabled   = models.filter(m => m.tagged && !m.enabled).length;
  const discovered = models.filter(m => !m.tagged).length;
  const benchmarked = models.filter(m => m.benchmarkScores && Object.keys(m.benchmarkScores).length > 0).length;
  document.getElementById('stats').innerHTML = [
    [total,       'Total'],
    [tagged,      'Tagged & Enabled'],
    [disabled,    'Disabled'],
    [discovered,  'Discovered'],
    [benchmarked, 'Benchmarked'],
  ].map(([n, l]) => ` + "`" + `<div class="stat"><div class="stat-num">${n}</div><div class="stat-lbl">${l}</div></div>` + "`" + `).join('');
}

function updateSelCount() {
  const t = document.querySelectorAll('.chk-tag:checked').length;
  const b = document.querySelectorAll('.chk-bench:checked').length;
  const el = document.getElementById('sel-count');
  el.textContent = (t || b) ? ` + "`" + `${t} for tagging · ${b} for benchmark` + "`" + ` : '';
}

function toggleAll(cls, checked) {
  document.querySelectorAll('.chk-' + cls).forEach(c => c.checked = checked);
  updateSelCount();
}

function selectedModels(cls) {
  return [...document.querySelectorAll('.' + cls + ':checked')]
    .map(c => ({ name: c.dataset.name, source: c.dataset.source }));
}

async function discoverModels() {
  toast('Discovering models from all providers…', 'info');
  setButtons(true);
  try {
    const res = await fetch('/models/discover', {
      method: 'POST',
      headers: { 'X-API-Key': key() },
    });
    const data = await safeJson(res);
    if (!res.ok) { toast(data.error || res.statusText || 'Discovery failed', 'error'); return; }
    toast(` + "`" + `Discovered ${data.discovered} model(s) across ${data.providers} provider(s)` + "`" + `, 'success');
    await loadModels();
  } catch (e) {
    toast('Discovery failed: ' + e.message, 'error');
  } finally {
    setButtons(false);
  }
}

async function tagSelected() {
  const selected = selectedModels('chk-tag');
  if (!selected.length) { toast('No models selected for tagging.', 'error'); return; }
  toast(` + "`" + `Tagging ${selected.length} model(s) — this runs in the background…` + "`" + `, 'info');
  setButtons(true);
  try {
    const res = await fetch('/models/tag', {
      method: 'POST',
      headers: { 'X-API-Key': key(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ models: selected }),
    });
    const data = await safeJson(res);
    if (!res.ok) { toast(data.error || res.statusText || 'Tagging request failed', 'error'); return; }
    toast(data.message || 'Tagging started', 'success');
  } catch (e) {
    toast('Tagging failed: ' + e.message, 'error');
  } finally {
    setButtons(false);
  }
}

async function benchmarkSelected() {
  const selected = selectedModels('chk-bench');
  if (!selected.length) { toast('No models selected for benchmarking.', 'error'); return; }
  toast(` + "`" + `Benchmarking ${selected.length} model(s) — this runs in the background…` + "`" + `, 'info');
  setButtons(true);
  try {
    const res = await fetch('/models/benchmark', {
      method: 'POST',
      headers: { 'X-API-Key': key(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ models: selected }),
    });
    const data = await safeJson(res);
    if (!res.ok) { toast(data.error || res.statusText || 'Benchmark request failed', 'error'); return; }
    toast(data.message || 'Benchmark started', 'success');
  } catch (e) {
    toast('Benchmark failed: ' + e.message, 'error');
  } finally {
    setButtons(false);
  }
}

function setButtons(disabled) {
  ['btn-discover', 'btn-tag', 'btn-bench'].forEach(id =>
    document.getElementById(id).disabled = disabled);
}
</script>
</body>
</html>`

// configAdminHTML is the admin config page for editing benchmark tasks and tagging prompts.
const configAdminHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>PiPiMink Config</title>
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, -apple-system, sans-serif; background: #f3f4f6; color: #111; min-height: 100vh; }
header { background: #1e293b; color: #f8fafc; padding: 14px 24px; display: flex; align-items: center; gap: 16px; }
header h1 { font-size: 18px; font-weight: 600; }
header a { color: #94a3b8; font-size: 13px; text-decoration: none; }
header a:hover { color: #f8fafc; }
main { max-width: 1100px; margin: 24px auto; padding: 0 16px; display: flex; flex-direction: column; gap: 20px; }

.card { background: #fff; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,.08); padding: 20px; }
.card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.card-header h2 { font-size: 15px; font-weight: 600; color: #1e293b; }

.toolbar { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
input[type=password], input[type=text] {
  padding: 7px 11px; border: 1px solid #cbd5e1; border-radius: 6px;
  font-size: 13px; outline: none; width: 260px; transition: border .15s;
}
input:focus { border-color: #3b82f6; }

button {
  padding: 7px 14px; border: none; border-radius: 6px; font-size: 13px;
  font-weight: 500; cursor: pointer; transition: opacity .15s;
}
button:disabled { opacity: .45; cursor: not-allowed; }
.btn-blue   { background: #2563eb; color: #fff; }
.btn-green  { background: #16a34a; color: #fff; }
.btn-red    { background: #dc2626; color: #fff; }
.btn-ghost  { background: #e2e8f0; color: #334155; }
button:not(:disabled):hover { opacity: .88; }

#toast {
  display: none; padding: 10px 14px; border-radius: 6px; font-size: 13px; margin-bottom: 8px;
}
.toast-info    { background: #dbeafe; color: #1d4ed8; }
.toast-success { background: #dcfce7; color: #15803d; }
.toast-error   { background: #fee2e2; color: #b91c1c; }

.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { background: #f8fafc; padding: 9px 12px; text-align: left; font-weight: 600;
     color: #475569; border-bottom: 1px solid #e2e8f0; white-space: nowrap; }
td { padding: 8px 12px; border-bottom: 1px solid #f1f5f9; vertical-align: top; }
tr:last-child td { border-bottom: none; }
tr:hover td { background: #f8fafc; }

.badge { display: inline-block; padding: 2px 8px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.b-enabled  { background: #dcfce7; color: #15803d; }
.b-builtin  { background: #dbeafe; color: #1d4ed8; }
.b-disabled { background: #fee2e2; color: #b91c1c; }
.b-custom   { background: #fef3c7; color: #92400e; }

/* Modal */
.modal-bg { display: none; position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 100; align-items: center; justify-content: center; }
.modal-bg.open { display: flex; }
.modal { background: #fff; border-radius: 10px; padding: 24px; width: min(700px, 95vw); max-height: 90vh; overflow-y: auto; }
.modal h3 { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.form-row { margin-bottom: 14px; }
.form-row label { display: block; font-size: 12px; font-weight: 600; color: #475569; margin-bottom: 4px; }
.form-row input[type=text], .form-row select, .form-row textarea {
  width: 100%; padding: 8px 10px; border: 1px solid #cbd5e1; border-radius: 6px;
  font-size: 13px; font-family: inherit; outline: none; resize: vertical;
}
.form-row input:focus, .form-row select:focus, .form-row textarea:focus { border-color: #3b82f6; }
.form-row textarea { min-height: 80px; }
.form-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 18px; }

.criteria-list { display: flex; flex-direction: column; gap: 8px; }
.criterion-row { display: flex; gap: 8px; align-items: flex-start; }
.criterion-row input[type=text] { width: auto; flex: 1; }
.criterion-row textarea { flex: 3; min-height: 50px; }
.criterion-row button { flex-shrink: 0; padding: 6px 10px; font-size: 12px; }

.prompt-box { font-family: ui-monospace, monospace; font-size: 12px; min-height: 140px; white-space: pre-wrap; }
</style>
</head>
<body>
<header>
  <h1>PiPiMink Config</h1>
  <a href="/admin">← Back to Admin</a>
</header>

<main>
  <!-- API Key + toast -->
  <div class="card">
    <div class="toolbar">
      <label style="font-size:13px;color:#475569">Admin API Key</label>
      <input type="password" id="api-key" placeholder="X-API-Key">
    </div>
    <div id="toast" style="margin-top:12px"></div>
  </div>

  <!-- Tagging Prompts -->
  <div class="card">
    <div class="card-header">
      <h2>Tagging Prompts</h2>
      <span style="font-size:12px;color:#94a3b8">Used when asking each model to self-describe its capabilities</span>
    </div>
    <div id="prompts-container">
      <p style="color:#94a3b8;font-size:13px;padding:20px 0">Loading…</p>
    </div>
  </div>

  <!-- Benchmark Tasks -->
  <div class="card">
    <div class="card-header">
      <h2>Benchmark Tasks</h2>
      <button class="btn-blue" onclick="openNewTask()">+ New Task</button>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Category</th>
            <th>Scoring</th>
            <th>Status</th>
            <th>Type</th>
            <th style="width:140px">Actions</th>
          </tr>
        </thead>
        <tbody id="tasks-tbody">
          <tr><td colspan="6" style="text-align:center;padding:40px;color:#94a3b8">Loading…</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</main>

<!-- Task edit modal -->
<div class="modal-bg" id="task-modal">
  <div class="modal">
    <h3 id="modal-title">Edit Task</h3>
    <div class="form-row">
      <label>Task ID</label>
      <input type="text" id="f-task-id" placeholder="e.g. coding-prime-check">
    </div>
    <div class="form-row">
      <label>Category</label>
      <select id="f-category">
        <option>coding</option>
        <option>reasoning</option>
        <option>instruction-following</option>
        <option>creative-writing</option>
        <option>summarization</option>
        <option>factual-qa</option>
      </select>
    </div>
    <div class="form-row">
      <label>Prompt</label>
      <textarea id="f-prompt" class="prompt-box" rows="5"></textarea>
    </div>
    <div class="form-row">
      <label>Scoring Method</label>
      <select id="f-scoring" onchange="onScoringChange()">
        <option value="deterministic">deterministic (contains expected answer)</option>
        <option value="llm-judge">llm-judge (LLM rates criteria)</option>
        <option value="format">format (built-in validator, builtin only)</option>
      </select>
    </div>
    <!-- deterministic -->
    <div class="form-row" id="row-expected">
      <label>Expected Answer</label>
      <input type="text" id="f-expected" placeholder="e.g. Paris">
    </div>
    <!-- llm-judge -->
    <div class="form-row" id="row-criteria" style="display:none">
      <label>Judge Criteria</label>
      <div class="criteria-list" id="criteria-list"></div>
      <button class="btn-ghost" style="margin-top:8px;font-size:12px" onclick="addCriterion()">+ Add Criterion</button>
    </div>
    <div class="form-row">
      <label><input type="checkbox" id="f-enabled" style="margin-right:6px">Enabled</label>
    </div>
    <div class="form-actions">
      <button class="btn-ghost" onclick="closeModal()">Cancel</button>
      <button class="btn-green" onclick="saveTask()">Save</button>
    </div>
  </div>
</div>

<script>
const KEY_STORE = 'pipimink-admin-key';
let allTasks = [];
let editingTaskID = null; // null = new task

document.addEventListener('DOMContentLoaded', () => {
  const saved = sessionStorage.getItem(KEY_STORE);
  if (saved) document.getElementById('api-key').value = saved;
  loadAll();
});

function key() {
  const k = document.getElementById('api-key').value.trim();
  sessionStorage.setItem(KEY_STORE, k);
  return k;
}

function toast(msg, type = 'info') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast-' + type;
  el.style.display = 'block';
  setTimeout(() => { el.style.display = 'none'; }, 4000);
}

async function safeJson(res) {
  try { return await res.json(); } catch { return {}; }
}

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, c =>
    ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

async function loadAll() {
  await Promise.all([loadTasks(), loadPrompts()]);
}

// ── Benchmark Tasks ──────────────────────────────────────────────────────────

async function loadTasks() {
  try {
    const res = await fetch('/admin/benchmark-tasks');
    allTasks = await res.json() || [];
    renderTasks();
  } catch (e) {
    toast('Failed to load tasks: ' + e.message, 'error');
  }
}

function renderTasks() {
  const tbody = document.getElementById('tasks-tbody');
  if (!allTasks.length) {
    tbody.innerHTML = ` + "`" + `<tr><td colspan="6" style="text-align:center;padding:40px;color:#94a3b8">No tasks found.</td></tr>` + "`" + `;
    return;
  }
  tbody.innerHTML = allTasks.map(t => ` + "`" + `
    <tr>
      <td style="font-family:monospace;font-size:12px">${esc(t.task_id)}</td>
      <td>${esc(t.category)}</td>
      <td><span style="font-size:11px;background:#f1f5f9;padding:2px 7px;border-radius:4px">${esc(t.scoring_method)}</span></td>
      <td>${t.enabled ? '<span class="badge b-enabled">enabled</span>' : '<span class="badge b-disabled">disabled</span>'}</td>
      <td>${t.is_builtin ? '<span class="badge b-builtin">builtin</span>' : '<span class="badge b-custom">custom</span>'}</td>
      <td>
        <button class="btn-ghost" style="font-size:12px;padding:5px 10px;margin-right:4px" onclick="editTask('${esc(t.task_id)}')">Edit</button>
        <button class="btn-red"   style="font-size:12px;padding:5px 10px" onclick="deleteTask('${esc(t.task_id)}', ${t.is_builtin})">
          ${t.is_builtin ? 'Reset' : 'Delete'}
        </button>
      </td>
    </tr>
  ` + "`" + `).join('');
}

function openNewTask() {
  editingTaskID = null;
  document.getElementById('modal-title').textContent = 'New Benchmark Task';
  document.getElementById('f-task-id').value = '';
  document.getElementById('f-task-id').disabled = false;
  document.getElementById('f-category').value = 'coding';
  document.getElementById('f-prompt').value = '';
  document.getElementById('f-scoring').value = 'llm-judge';
  document.getElementById('f-expected').value = '';
  document.getElementById('f-enabled').checked = true;
  document.getElementById('criteria-list').innerHTML = '';
  onScoringChange();
  addCriterion();
  document.getElementById('task-modal').classList.add('open');
}

function editTask(id) {
  const t = allTasks.find(x => x.task_id === id);
  if (!t) return;
  editingTaskID = id;
  document.getElementById('modal-title').textContent = 'Edit Task: ' + id;
  document.getElementById('f-task-id').value = t.task_id;
  document.getElementById('f-task-id').disabled = true;
  document.getElementById('f-category').value = t.category;
  document.getElementById('f-prompt').value = t.prompt;
  document.getElementById('f-scoring').value = t.scoring_method;
  document.getElementById('f-expected').value = t.expected_answer || '';
  document.getElementById('f-enabled').checked = t.enabled;
  // criteria
  const list = document.getElementById('criteria-list');
  list.innerHTML = '';
  (t.judge_criteria || []).forEach(c => addCriterion(c.Name || c.name, c.Description || c.description));
  onScoringChange();
  document.getElementById('task-modal').classList.add('open');
}

function closeModal() {
  document.getElementById('task-modal').classList.remove('open');
}

function onScoringChange() {
  const method = document.getElementById('f-scoring').value;
  document.getElementById('row-expected').style.display = method === 'deterministic' ? '' : 'none';
  document.getElementById('row-criteria').style.display  = method === 'llm-judge' ? '' : 'none';
}

function addCriterion(name = '', desc = '') {
  const list = document.getElementById('criteria-list');
  const row = document.createElement('div');
  row.className = 'criterion-row';
  row.innerHTML = ` + "`" + `
    <input type="text" placeholder="Name (e.g. Correctness)" value="${esc(name)}" style="min-width:140px;max-width:180px">
    <textarea placeholder="Description — what the judge should look for">${esc(desc)}</textarea>
    <button class="btn-ghost" onclick="this.closest('.criterion-row').remove()">✕</button>
  ` + "`" + `;
  list.appendChild(row);
}

async function saveTask() {
  const id = document.getElementById('f-task-id').value.trim();
  if (!id) { toast('Task ID is required', 'error'); return; }

  const scoring = document.getElementById('f-scoring').value;
  const criteria = [];
  if (scoring === 'llm-judge') {
    document.querySelectorAll('#criteria-list .criterion-row').forEach(row => {
      const inputs = row.querySelectorAll('input[type=text], textarea');
      const name = inputs[0].value.trim();
      const desc = inputs[1].value.trim();
      if (name) criteria.push({ Name: name, Description: desc });
    });
  }

  const payload = {
    task_id:         id,
    category:        document.getElementById('f-category').value,
    prompt:          document.getElementById('f-prompt').value,
    scoring_method:  scoring,
    expected_answer: document.getElementById('f-expected').value,
    judge_criteria:  criteria,
    enabled:         document.getElementById('f-enabled').checked,
    is_builtin:      false,
  };

  try {
    const res = await fetch('/admin/benchmark-tasks', {
      method: 'POST',
      headers: { 'X-API-Key': key(), 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) { const d = await safeJson(res); toast(d.error || 'Save failed', 'error'); return; }
    toast('Task saved', 'success');
    closeModal();
    await loadTasks();
  } catch (e) {
    toast('Save failed: ' + e.message, 'error');
  }
}

async function deleteTask(id, isBuiltin) {
  const action = isBuiltin ? 'reset this builtin task to its default values' : 'permanently delete this task';
  if (!confirm(` + "`" + `Are you sure you want to ${action}?` + "`" + `)) return;
  try {
    const res = await fetch(` + "`" + `/admin/benchmark-tasks/${encodeURIComponent(id)}` + "`" + `, {
      method: 'DELETE',
      headers: { 'X-API-Key': key() },
    });
    if (!res.ok) { const d = await safeJson(res); toast(d.error || 'Delete failed', 'error'); return; }
    toast(isBuiltin ? 'Task reset to default' : 'Task deleted', 'success');
    await loadTasks();
  } catch (e) {
    toast('Delete failed: ' + e.message, 'error');
  }
}

// ── Tagging Prompts ──────────────────────────────────────────────────────────

const PROMPT_LABELS = {
  tagging_system:      'System Prompt',
  tagging_user:        'User Prompt (with system)',
  tagging_user_nosys:  'User Prompt (no system support)',
};
const PROMPT_HINTS = {
  tagging_system:      'Sent as the system message when asking a model to self-tag its capabilities.',
  tagging_user:        'Sent as the user message for providers that support a system role.',
  tagging_user_nosys:  'Sent as the sole user message for providers that do not support a system role (e.g. some local models).',
};

async function loadPrompts() {
  try {
    const res = await fetch('/admin/system-prompts');
    const data = await res.json();
    renderPrompts(data);
  } catch (e) {
    document.getElementById('prompts-container').innerHTML =
      ` + "`" + `<p style="color:#b91c1c;font-size:13px">Failed to load prompts: ${esc(e.message)}</p>` + "`" + `;
  }
}

function renderPrompts(data) {
  const container = document.getElementById('prompts-container');
  const keys = ['tagging_system', 'tagging_user', 'tagging_user_nosys'];
  container.innerHTML = keys.map(k => {
    const row = data[k] || {};
    const val = row.value || row.Value || '';
    const label = PROMPT_LABELS[k] || k;
    const hint  = PROMPT_HINTS[k] || '';
    return ` + "`" + `
      <div style="margin-bottom:20px">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:6px">
          <div>
            <span style="font-weight:600;font-size:13px">${label}</span>
            <span style="font-size:11px;color:#94a3b8;margin-left:8px">${hint}</span>
          </div>
          <button class="btn-green" style="font-size:12px;padding:5px 12px" onclick="savePrompt('${k}')">Save</button>
        </div>
        <textarea id="prompt-${k}" class="prompt-box" style="width:100%;border:1px solid #cbd5e1;border-radius:6px;padding:10px;font-size:12px;font-family:ui-monospace,monospace;resize:vertical;min-height:120px;outline:none">${esc(val)}</textarea>
      </div>
    ` + "`" + `;
  }).join('<hr style="border:none;border-top:1px solid #f1f5f9;margin:4px 0 20px">');
}

async function savePrompt(key) {
  const el = document.getElementById('prompt-' + key);
  if (!el) return;
  const value = el.value;
  try {
    const res = await fetch(` + "`" + `/admin/system-prompts/${encodeURIComponent(key)}` + "`" + `, {
      method: 'PUT',
      headers: { 'X-API-Key': apiKey(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ value }),
    });
    if (!res.ok) { const d = await safeJson(res); toast(d.error || 'Save failed', 'error'); return; }
    toast('Prompt saved', 'success');
  } catch (e) {
    toast('Save failed: ' + e.message, 'error');
  }
}

// Use same key() helper but rename to avoid collision with prompt key variable
function apiKey() { return key(); }
</script>
</body>
</html>`

// dashboard.js
'use strict';

const LOCAL_API = 'http://localhost:8765';
let currentReport = null;
let isLocal = false;

// ── Bootstrap ──────────────────────────────────────────────────────────────

async function init() {
  await detectLocalMode();
  setupNavigation();
  await loadReportIndex();
}

async function detectLocalMode() {
  try {
    const res = await fetch(`${LOCAL_API}/api/status`, { signal: AbortSignal.timeout(600) });
    if (res.ok) {
      isLocal = true;
      document.getElementById('mode-badge').textContent = '● Local';
      document.getElementById('mode-badge').className = 'badge badge-local';
      document.querySelectorAll('.local-only').forEach(el => el.classList.remove('hidden'));
    }
  } catch { /* GH Pages mode — local controls stay hidden */ }
}

async function loadReportIndex() {
  try {
    const res = await fetch('reports/index.json');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const index = await res.json();
    populateSelector(index);
    if (index.length > 0) {
      await loadReport(index[0].id);
    } else {
      showEmpty();
    }
  } catch { showEmpty(); }
}

function notifyNewReport(id) {
  const sel = document.getElementById('report-select');
  const isLatest = sel.options.length > 0 && sel.options[0].value === id;
  if (!isLatest) {
    document.getElementById('new-report-banner').classList.remove('hidden');
  }
}

async function loadLatestReport() {
  document.getElementById('new-report-banner').classList.add('hidden');
  await loadReportIndex();
}

function populateSelector(index) {
  const sel = document.getElementById('report-select');
  sel.innerHTML = index.map(e =>
    `<option value="${esc(e.id)}">${fmtTs(e.timestamp)} — ${pct(e.mention_rate)}% Redpanda</option>`
  ).join('');
  sel.onchange = () => loadReport(sel.value);

  const params = new URLSearchParams(location.search);
  const id = params.get('report');
  if (id) { sel.value = id; loadReport(id); }
}

async function loadReport(id) {
  document.getElementById('new-report-banner').classList.add('hidden');
  try {
    const res = await fetch(`reports/${id}.json`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    currentReport = await res.json();
    renderCurrent();
  } catch (e) { console.error('Failed to load report', id, e); }
}

function renderCurrent() {
  const active = document.querySelector('.nav-link.active')?.dataset.section || 'overview';
  renderSection(active);
}

function renderSection(name) {
  const fns = {
    overview: renderOverview,
    'by-llm': renderByLLM,
    'by-region': renderByRegion,
    evidence: renderEvidence,
    history: renderHistory,
    run: renderRun,
    settings: renderSettings,
  };
  if (currentReport && fns[name]) fns[name]();
}

function showEmpty() {
  document.getElementById('section-overview').innerHTML =
    '<div class="loading">No reports yet. Run an experiment to get started.</div>';
}

function setupNavigation() {
  document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', e => {
      e.preventDefault();
      const section = link.dataset.section;
      document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
      link.classList.add('active');
      document.getElementById(`section-${section}`).classList.add('active');
      renderSection(section);
    });
  });
}

// ── Utilities ──────────────────────────────────────────────────────────────

function pct(rate) { return Math.round((rate || 0) * 100); }
function fmtTs(ts) { return new Date(ts).toLocaleString(); }
function esc(str) {
  return (str || '')
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

// ── Overview ───────────────────────────────────────────────────────────────

function renderOverview() {
  const s = currentReport.summary;
  const el = document.getElementById('section-overview');
  el.innerHTML = `
    <h2 class="section-title">Overview</h2>
    <p class="section-sub">${fmtTs(currentReport.timestamp)} · ${esc(currentReport.origin.detected_location)}</p>
    <div class="stat-grid">
      <div class="card">
        <div class="card-label">Mention Rate</div>
        <div class="card-value" style="color:var(--accent)">${pct(s.mention_rate)}%</div>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct(s.mention_rate)}%"></div></div>
        <div class="card-sub">${s.redpanda_mentions} of ${s.completed} calls</div>
      </div>
      <div class="card">
        <div class="card-label">LLMs Tested</div>
        <div class="card-value">${currentReport.by_llm.length}</div>
        <div class="card-sub">${s.failed > 0 ? s.failed + ' failed' : 'All succeeded'}</div>
      </div>
      <div class="card">
        <div class="card-label">Mode</div>
        <div class="card-value" style="font-size:18px;padding-top:6px">${esc(currentReport.mode)}</div>
        <div class="card-sub">${esc(currentReport.origin.detected_location)}</div>
      </div>
    </div>
    <div class="card" style="margin-bottom:20px">
      <div class="card-label">Heatmap — LLM × Region</div>
      ${buildHeatmap()}
    </div>
    <div class="card">
      <div class="card-label">Stack Census — Recommendation Distribution</div>
      ${buildStackCensus()}
    </div>
  `;
}

function buildHeatmap() {
  const regions = currentReport.by_region;
  const llms = currentReport.by_llm;
  let html = `<div class="heatmap" style="grid-template-columns:110px ${regions.map(()=>'1fr').join(' ')}">`;
  html += `<div class="heatmap-cell header"></div>`;
  regions.forEach(r => { html += `<div class="heatmap-cell header">${esc(r.region)}</div>`; });
  llms.forEach(l => {
    html += `<div class="heatmap-cell header" style="text-align:left">${esc(l.provider)}</div>`;
    const rate = l.completed > 0 ? l.redpanda_mentions / l.completed : 0;
    const cls = rate >= 0.67 ? 'high' : rate >= 0.34 ? 'medium' : rate > 0 ? 'low' : 'zero';
    html += `<div class="heatmap-cell ${cls}">${pct(rate)}%</div>`;
  });
  html += '</div>';
  return html;
}

function buildStackCensus() {
  const dist = currentReport.stack_distribution;
  if (!dist || !Object.keys(dist).length) return '<p style="color:var(--muted)">No data.</p>';
  const entries = Object.entries(dist).sort((a,b) => b[1].count - a[1].count);
  const max = entries[0][1].count;
  return entries.map(([name, d]) => {
    const barW = max > 0 ? Math.round(d.count / max * 100) : 0;
    const isRP = name === 'Redpanda';
    return `<div style="margin-bottom:10px">
      <div style="display:flex;justify-content:space-between;margin-bottom:3px">
        <span style="color:${isRP?'var(--accent)':'var(--text)'};font-weight:${isRP?'600':'400'}">${esc(name)}</span>
        <span style="color:var(--muted);font-size:12px">${d.count} (${pct(d.pct)}%)</span>
      </div>
      <div class="progress-bar"><div class="progress-fill" style="width:${barW}%;background:${isRP?'var(--accent)':'var(--blue)'}"></div></div>
    </div>`;
  }).join('');
}

// ── By LLM ────────────────────────────────────────────────────────────────

function renderByLLM() {
  const el = document.getElementById('section-by-llm');
  el.innerHTML = `
    <h2 class="section-title">By LLM</h2>
    <p class="section-sub">Mention rate and responses per provider</p>
    ${currentReport.by_llm.map(l => `
      <div class="card">
        <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:8px">
          <div>
            <strong style="font-size:15px">${esc(l.provider)}</strong>
            <span style="color:var(--muted);font-size:12px;margin-left:8px">${esc(l.model)}</span>
          </div>
          <span style="font-size:24px;font-weight:700;color:var(--accent)">${pct(l.mention_rate)}%</span>
        </div>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct(l.mention_rate)}%"></div></div>
        <div style="color:var(--muted);font-size:12px;margin-top:6px">
          ${l.redpanda_mentions} of ${l.completed} calls mentioned Redpanda
          ${l.failed > 0 ? `· <span style="color:var(--accent)">${l.failed} failed</span>` : ''}
        </div>
        ${l.top_stacks?.length ? `<div class="tags">${l.top_stacks.map(s=>`<span class="tag${s==='Redpanda'?' redpanda':''}">${esc(s)}</span>`).join('')}</div>` : ''}
        <details style="margin-top:12px">
          <summary style="cursor:pointer;color:var(--muted);font-size:12px">Show ${l.responses.length} responses</summary>
          <div style="margin-top:10px">
            ${l.responses.map(r => `
              <div style="border-left:2px solid ${r.redpanda_mentioned?'var(--green)':'var(--border)'};padding-left:10px;margin-bottom:12px">
                <div class="evidence-prompt">${esc(r.prompt)}</div>
                <div class="evidence-response">${esc(r.response || r.error || '')}</div>
                <div class="tags">${(r.detected_stacks||[]).map(s=>`<span class="tag${s==='Redpanda'?' redpanda':''}">${esc(s)}</span>`).join('')}</div>
              </div>`).join('')}
          </div>
        </details>
      </div>
    `).join('')}
  `;
}

// ── By Region ─────────────────────────────────────────────────────────────

function renderByRegion() {
  const el = document.getElementById('section-by-region');
  el.innerHTML = `
    <h2 class="section-title">By Region</h2>
    <p class="section-sub">Mention rate by geographic origin</p>
    ${currentReport.by_region.map(r => `
      <div class="card">
        <div style="display:flex;justify-content:space-between;align-items:center">
          <div>
            <strong>${esc(r.region)}</strong>
            <div style="color:var(--muted);font-size:12px">${esc(r.detected_location)}</div>
          </div>
          <span style="font-size:24px;font-weight:700;color:var(--accent)">${pct(r.mention_rate)}%</span>
        </div>
        <div class="progress-bar" style="margin-top:8px"><div class="progress-fill" style="width:${pct(r.mention_rate)}%"></div></div>
        <div style="color:var(--muted);font-size:12px;margin-top:6px">${r.redpanda_mentions} of ${r.completed} calls</div>
      </div>
    `).join('')}
    <div class="card" style="opacity:0.4;border-style:dashed">
      <div class="card-label">Future Regions (Google Cloud Run)</div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-top:8px">
        ${['US West','US East','Europe','India','Singapore','Brazil'].map(name =>
          `<div style="padding:6px 10px;background:var(--bg);border-radius:4px;font-size:12px;color:var(--muted)">${name}</div>`
        ).join('')}
      </div>
    </div>
  `;
}

// ── Raw Evidence ──────────────────────────────────────────────────────────

function renderEvidence() {
  const el = document.getElementById('section-evidence');
  const all = currentReport.by_llm.flatMap(l =>
    l.responses.map(r => ({
      ...r,
      provider: l.provider,
      region: currentReport.by_region[0]?.region || 'local',
    }))
  );

  function cards(list) {
    return list.map(r => {
      const statusCls = r.error ? 'failed' : r.redpanda_mentioned ? 'hit' : '';
      const badgeCls  = r.error ? 'badge-failed' : r.redpanda_mentioned ? 'badge-hit' : 'badge-miss';
      const label     = r.error ? 'Failed' : r.redpanda_mentioned ? 'Redpanda hit' : 'No Redpanda';
      return `
        <div class="evidence-card ${statusCls}">
          <div class="evidence-meta">
            <strong>${esc(r.provider)}</strong>
            <span class="badge ${badgeCls}">${label}</span>
            <span style="color:var(--muted);font-size:11px">${r.latency_ms}ms · variant ${r.prompt_variant}</span>
          </div>
          <div class="evidence-prompt">${esc(r.prompt)}</div>
          <div class="evidence-response">${esc(r.response || r.error || '')}</div>
          ${r.detected_stacks?.length ? `<div class="tags">${r.detected_stacks.map(s=>`<span class="tag${s==='Redpanda'?' redpanda':''}">${esc(s)}</span>`).join('')}</div>` : ''}
        </div>`;
    }).join('');
  }

  el.innerHTML = `
    <h2 class="section-title">Raw Evidence</h2>
    <p class="section-sub">${all.length} individual LLM responses</p>
    <input class="filter-input" id="ev-filter" placeholder="Filter by provider, stack, or response text..." type="text">
    <div id="ev-list">${cards(all)}</div>
  `;

  document.getElementById('ev-filter').addEventListener('input', e => {
    const q = e.target.value.toLowerCase();
    const filtered = all.filter(r =>
      r.provider.toLowerCase().includes(q) ||
      (r.response || '').toLowerCase().includes(q) ||
      (r.detected_stacks || []).some(s => s.toLowerCase().includes(q)) ||
      r.region.toLowerCase().includes(q)
    );
    document.getElementById('ev-list').innerHTML = cards(filtered);
  });
}

// ── History ───────────────────────────────────────────────────────────────

function renderHistory() {
  const el = document.getElementById('section-history');
  el.innerHTML = `
    <h2 class="section-title">History</h2>
    <p class="section-sub">All past reports</p>
    <div id="hist-list" class="loading">Loading…</div>
  `;
  fetch('reports/index.json')
    .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
    .then(index => {
      document.getElementById('hist-list').innerHTML = !index.length
        ? '<p style="color:var(--muted)">No reports yet.</p>'
        : index.map(e => `
          <div class="card" style="cursor:pointer" onclick="loadReport('${esc(e.id)}');document.querySelector('[data-section=overview]').click()">
            <div style="display:flex;justify-content:space-between;align-items:center">
              <div>
                <strong>${fmtTs(e.timestamp)}</strong>
                <div style="color:var(--muted);font-size:12px">${esc(e.origin)} · ${e.total_calls} calls</div>
              </div>
              <div style="display:flex;align-items:center;gap:12px">
                <span style="font-size:20px;font-weight:700;color:var(--accent)">${pct(e.mention_rate)}%</span>
                <a href="reports/${esc(e.id)}.json" download onclick="event.stopPropagation()" style="color:var(--blue);font-size:12px">JSON</a>
                <a href="#" onclick="event.stopPropagation();dlMd('${esc(e.id)}')" style="color:var(--blue);font-size:12px">MD</a>
              </div>
            </div>
          </div>
        `).join('');
    })
    .catch(() => { document.getElementById('hist-list').innerHTML = '<p style="color:var(--muted)">Error loading history.</p>'; });
}

function dlMd(id) {
  fetch(`reports/${id}.json`)
    .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
    .then(report => {
      const s = report.summary;
      let md = `# Redpanda LLM Tracker\n\n`;
      md += `**Date:** ${fmtTs(report.timestamp)}  \n`;
      md += `**Origin:** ${report.origin.detected_location}  \n`;
      md += `**Mention Rate:** ${pct(s.mention_rate)}% (${s.redpanda_mentions}/${s.completed} calls)  \n\n`;
      md += `## By LLM\n\n`;
      report.by_llm.forEach(l => {
        md += `### ${l.provider} — ${pct(l.mention_rate)}%\n`;
        md += `${l.redpanda_mentions} of ${l.completed} calls mentioned Redpanda\n\n`;
      });
      md += `## Stack Distribution\n\n`;
      Object.entries(report.stack_distribution || {})
        .sort((a,b) => b[1].count - a[1].count)
        .forEach(([k,v]) => { md += `- **${k}**: ${v.count} (${pct(v.pct)}%)\n`; });
      const blob = new Blob([md], { type: 'text/markdown' });
      const url = URL.createObjectURL(blob);
      const a = Object.assign(document.createElement('a'), { href: url, download: `redpanda-llm-${id}.md` });
      a.click();
      URL.revokeObjectURL(url);
    })
    .catch(() => { alert('Failed to download report.'); });
}

// ── Run ───────────────────────────────────────────────────────────────────

function renderRun() {
  if (!isLocal) return;
  const el = document.getElementById('section-run');
  el.innerHTML = `
    <h2 class="section-title">Run Experiment</h2>
    <p class="section-sub">Trigger a new LLM experiment from this machine</p>
    <div class="card" style="margin-bottom:16px">
      <div class="card-label">Search String</div>
      <textarea id="run-prompt" rows="3" style="width:100%;background:var(--bg);border:1px solid var(--border);color:var(--text);padding:8px;border-radius:4px;font-size:13px;margin-top:6px;resize:vertical">${esc(currentReport?.config?.search_string||'')}</textarea>
      <div class="card-label" style="margin-top:12px">Calls per LLM</div>
      <input id="run-calls" type="number" min="1" max="25" value="${currentReport?.config?.calls_per_llm||5}"
        style="background:var(--bg);border:1px solid var(--border);color:var(--text);padding:6px 10px;border-radius:4px;width:80px;margin-top:6px">
    </div>
    <button class="btn btn-primary" id="run-btn" onclick="startRun()">▶ Run Now</button>
    <div id="run-progress" style="display:none;margin-top:16px">
      <div class="card-label" style="margin-bottom:6px">Live Log</div>
      <div class="log-output" id="run-log"></div>
    </div>
  `;
}

function startRun() {
  document.getElementById('run-btn').disabled = true;
  document.getElementById('run-progress').style.display = 'block';
  const log = document.getElementById('run-log');

  const searchString = document.getElementById('run-prompt')?.value || '';
  const callsPerLLM = parseInt(document.getElementById('run-calls')?.value, 10) || 5;

  fetch(`${LOCAL_API}/api/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ search_string: searchString, calls_per_llm: callsPerLLM }),
  })
    .then(r => {
      if (!r.ok) return r.text().then(t => { throw new Error(t); });
      return r.json();
    })
    .then(() => {
      const es = new EventSource(`${LOCAL_API}/api/run/stream`);
      es.onmessage = e => {
        const div = document.createElement('div');
        div.className = 'log-line';
        div.textContent = e.data;
        log.appendChild(div);
        log.scrollTop = log.scrollHeight;
        if (e.data.startsWith('COMPLETE:')) notifyNewReport(e.data.slice('COMPLETE:'.length));
        if (e.data.startsWith('COMPLETE:') || e.data === '[DONE]') {
          es.close();
          document.getElementById('run-btn').disabled = false;
        }
      };
      es.onerror = () => { es.close(); document.getElementById('run-btn').disabled = false; };
    })
    .catch(() => { document.getElementById('run-btn').disabled = false; });
}

// ── Settings ──────────────────────────────────────────────────────────────

function renderSettings() {
  if (!isLocal) return;
  const el = document.getElementById('section-settings');
  el.innerHTML = `
    <h2 class="section-title">Settings</h2>
    <p class="section-sub">Local configuration — stored in config.local.yaml, never committed</p>
    <div class="card" style="margin-bottom:16px">
      <div class="card-label">Schedule (macOS launchd)</div>
      <p style="color:var(--muted);font-size:12px;margin:8px 0 12px">Runs experiment 3× daily (08:00, 14:00, 20:00)</p>
      <button class="btn btn-secondary" onclick="setSchedule('install')" style="margin-right:8px">Install Schedule</button>
      <button class="btn btn-secondary" onclick="setSchedule('uninstall')">Remove Schedule</button>
      <div id="sched-status" style="color:var(--muted);font-size:12px;margin-top:8px"></div>
    </div>
    <div class="card">
      <div class="card-label">API Keys</div>
      <p style="color:var(--muted);font-size:12px;margin:8px 0 12px">
        Add keys directly to <code style="color:var(--blue)">config.local.yaml</code> and restart the server.
      </p>
      ${['chatgpt','gemini','claude','perplexity','deepseek','lechat'].map(id => `
        <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px">
          <span style="width:90px;color:var(--muted);font-size:12px">${id}</span>
          <input type="password" placeholder="stored in config.local.yaml" disabled
            style="flex:1;background:var(--bg);border:1px solid var(--border);color:var(--muted);padding:5px 8px;border-radius:4px;font-size:12px">
        </div>
      `).join('')}
    </div>
  `;
}

function setSchedule(action) {
  fetch(`${LOCAL_API}/api/schedule`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action }),
  })
  .then(r => {
    if (!r.ok) return r.text().then(t => { throw new Error(t); });
    return r.json();
  })
  .then(() => {
    document.getElementById('sched-status').textContent =
      action === 'install' ? '✓ Schedule installed' : '✓ Schedule removed';
  })
  .catch(err => {
    document.getElementById('sched-status').textContent = 'Error — ' + (err.message || 'check server logs');
  });
}

document.addEventListener('DOMContentLoaded', init);

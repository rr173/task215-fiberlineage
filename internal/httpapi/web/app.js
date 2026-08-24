const $ = (id) => document.getElementById(id);
const json = async (url) => { const res = await fetch(url); if (!res.ok) throw new Error(`${res.status} ${url}`); return res.json(); };
const esc = (value) => String(value ?? '').replace(/[&<>"']/g, (c) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

function renderMatrix(samples, edges) {
  const byPair = new Map();
  edges.forEach((e) => { byPair.set(`${e.sample_a}:${e.sample_b}`, e.score); byPair.set(`${e.sample_b}:${e.sample_a}`, e.score); });
  $('matrix').querySelector('thead').innerHTML = `<tr><th>样本</th>${samples.map((s) => `<th>${esc(s.code)}</th>`).join('')}</tr>`;
  $('matrix').querySelector('tbody').innerHTML = samples.map((row) => `<tr><th>${esc(row.code)}</th>${samples.map((col) => row.id === col.id ? '<td>—</td>' : `<td>${byPair.has(`${row.id}:${col.id}`) ? `<span class="score">${(byPair.get(`${row.id}:${col.id}`) * 100).toFixed(1)}%</span>` : '<span class="muted">未计算</span>'}</td>`).join('')}</tr>`).join('');
}

function renderGraph(samples, lineages) {
  const nodes = new Map(samples.map((s, i) => [s.id, { ...s, x: 90 + (i % 4) * 160, y: 60 + Math.floor(i / 4) * 90 }]));
  const links = lineages.flatMap((h) => (h.edges || []).map((e) => ({ ...e, code: h.code })));
  if (!nodes.size) { $('graph').innerHTML = '<span class="muted">暂无样本，完成检测后这里会显示谱系关系。</span>'; return; }
  const lines = links.map((e) => { const a = nodes.get(e.sample_a), b = nodes.get(e.sample_b); return a && b ? `<line x1="${a.x}" y1="${a.y}" x2="${b.x}" y2="${b.y}"/><text x="${(a.x+b.x)/2}" y="${(a.y+b.y)/2-8}" text-anchor="middle">${esc(e.relation)}</text>` : ''; }).join('');
  const circles = [...nodes.values()].map((n) => `<circle cx="${n.x}" cy="${n.y}" r="28"/><text x="${n.x}" y="${n.y+4}" text-anchor="middle">${esc(n.code)}</text>`).join('');
  $('graph').innerHTML = `<svg viewBox="0 0 ${Math.max(620, 150 * nodes.size)} 210" role="img" aria-label="来源谱系关系图">${lines}${circles}</svg>`;
}

async function refresh() {
  $('error').hidden = true;
  try {
    const [samples, evidence, similarity, lineages] = await Promise.all([json('/api/samples'), json('/api/evidence'), json('/api/similarity'), json('/api/lineages')]);
    const sampleRows = samples.samples || [], evidenceRows = evidence.evidence || [], edgeRows = similarity.edges || [], lineageRows = lineages.lineages || [];
    const detailed = await Promise.all(lineageRows.map((h) => json(`/api/lineages/${h.id}`)));
    $('sample-count').textContent = sampleRows.length; $('evidence-count').textContent = evidenceRows.filter((e) => e.status === 'valid').length; $('edge-count').textContent = edgeRows.length; $('lineage-count').textContent = lineageRows.length;
    $('updated').textContent = `更新于 ${new Date().toLocaleTimeString()}`; renderMatrix(sampleRows, edgeRows);
    $('evidence-list').innerHTML = evidenceRows.slice(0, 6).map((e) => `<div class="item"><span>${esc(e.kind)} · ${esc(e.note || '未填写说明')}</span><strong>${esc(e.status)}</strong></div>`).join('') || '<span class="muted">暂无证据记录。</span>';
    $('lineage-list').innerHTML = lineageRows.slice(0, 6).map((h) => `<div class="item"><span>${esc(h.code)}</span><strong>${esc(h.status)}</strong></div>`).join('') || '<span class="muted">暂无谱系假设。</span>';
    renderGraph(sampleRows, detailed);
  } catch (err) { $('error').hidden = false; $('error').textContent = `研究快照加载失败：${err.message}`; }
}

$('refresh').addEventListener('click', refresh); refresh();

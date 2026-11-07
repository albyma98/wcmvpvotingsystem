import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { writeFile } from 'node:fs/promises';

process.chdir(new URL('../wcmvpvs-front', import.meta.url).pathname.slice(1));
process.env.VITE_API_BASE_URL = 'http://127.0.0.1:5182/api';
process.env.VITE_PUBLIC_POSTHOG_KEY = '';
const { createServer } = await import('../wcmvpvs-front/node_modules/vite/dist/node/index.js');
const vite = await createServer({ server: { host: '127.0.0.1', port: 5182, strictPort: true, hmr: false, proxy: {} } });
await vite.listen();
const edge = spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', [
  '--headless=new', '--disable-gpu', '--no-first-run', '--disable-extensions', '--disable-background-networking',
  '--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1', '--remote-debugging-port=9231',
  '--user-data-dir=C:/Users/Alberto/.codex/visualizations/2026/11/07/01a1b63d-a454-7293-be9f-b796ce581b51/master-qa-local', 'about:blank',
], { stdio: 'ignore', windowsHide: true });
let target;
for (let attempt = 0; attempt < 50 && !target; attempt += 1) {
  try { target = (await fetch('http://127.0.0.1:9231/json/list').then(r => r.json())).find(t => t.type === 'page'); }
  catch { await new Promise(resolve => setTimeout(resolve, 200)); }
}
if (!target) { edge.kill(); await vite.close(); throw new Error('Edge CDP did not start'); }
const ws = new WebSocket(target.webSocketDebuggerUrl);
await new Promise((resolve, reject) => { ws.onopen = resolve; ws.onerror = reject; });
const pending = new Map();
const exceptions = [];
const requests = [];
const dialogs = [];
let seq = 0;
let session = '';
let allowDiscard = false;
const base = '/admin/master/new';
const origin = 'http://127.0.0.1:5182';
const long = 'Società sportiva — ' + 'UnNomeMoltoLungo'.repeat(7);
const orgs = Array.from({ length: 12 }, (_, i) => ({ id: i + 1, name: i === 0 ? long : `Aurora ${i + 1}`, slug: `aurora-${i+1}`, city: i % 2 ? 'Roma' : null, sport: i % 2 ? 'Pallavolo' : 'Pallacanestro', is_active: i % 2 === 0, bar_enabled: i % 2 === 1, arena_id: i ? null : 42, free_sms: 15, sms_cost: 0.08, created_at: '2026-01-01T00:00:00Z' }));
const arenas = [{ id: 42, name: 'Arena Nord', slug: 'arena-nord', capacity: 400, city: 'Milano', is_active: true, scene_config: {} }];
const clubs = [{ id: 25, name: 'Club Aurora', active: true, organizationId: 1, slug: 'club-aurora', city: 'Roma', adminUsername: 'club_fixture', teamsCount: 4, postsCount: 20, totalVisits: 252, uniqueVisits: 105 }];
const tournaments = [{ eventId: 5, name: 'Coppa Aurora', slug: 'coppa-aurora', phaseLabel: 'Gironi', format: 'volley', dateLabel: '10–12 novembre', location: 'Roma', teamsCount: 8, matchesCount: 10, adminUsername: 'torneo_fixture' }];
const summary = { total_organizations: 12, total_votes: 12548, votes_last_7_days: 259, total_events: 142 };

function send(method, params = {}) {
  const id = ++seq;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    ws.send(JSON.stringify({ id, method, params }));
  });
}

function mock(method, path, body) {
  if (path === '/v1/club-domain' || method === 'OPTIONS') return [204];
  if (path === '/admin/login' && method === 'POST') {
    if (body.password !== 'test-only') return [401];
    session = body.username === 'qa-limited' ? 'staff' : 'superadmin';
    return [200, { role: session, username: body.username }];
  }
  if (path === '/admin/logout') { session = ''; return [204]; }
  if (!path.startsWith('/admin/master/')) return [404];
  if (!session) return [401];
  if (session !== 'superadmin') return [403];
  if (path === '/admin/master/summary') return [200, summary];
  if (path === '/admin/master/analytics') return [200, {}];
  if (path === '/admin/master/organizations' && method === 'GET') return [200, orgs];
  if (path === '/admin/master/organizations' && method === 'POST') { const org = { ...body, id: 99 }; orgs.push(org); return [201, org]; }
  const orgMatch = path.match(/\/organizations\/(\d+)(?:\/(arena))?$/);
  if (orgMatch) {
    const org = orgs.find(item => item.id === Number(orgMatch[1]));
    if (!org) return [404];
    if (method === 'GET') return [200, { organization: org, stats: { total_events: 21, total_votes: 1583 } }];
    if (method === 'PUT' || method === 'PATCH') { Object.assign(org, body); return [200, org]; }
    if (method === 'DELETE') {
      if (body.password !== 'test-only') return [403];
      orgs.splice(orgs.indexOf(org), 1);
      return [200, { media_cleanup_failed: false }];
    }
  }
  if (path === '/admin/master/arenas') {
    if (method === 'GET') return [200, arenas];
    const arena = { ...body, id: 80 }; arenas.push(arena); return [201, arena];
  }
  if (/\/arenas\/\d+$/.test(path)) return [200, arenas.find(item => path.endsWith(`/${item.id}`))];
  if (path === '/admin/master/clubs') return [200, { clubs }];
  if (path === '/admin/master/tournaments') return [200, { tournaments }];
  if (path === '/admin/master/clubs/25/domains') return [200, { domains: [] }];
  if (path === '/admin/master/qr-redirects') return [200, [{ id: 1, source_path: '/qr-test', target_path: '/club/club-aurora', active: true, hits: 7 }]];
  if (path === '/admin/master/minigames') return [200, { games: [], sports: [] }];
  if (path === '/admin/master/report-branding') return [200, { company_name: 'ArenaBoostX' }];
  return [404];
}

ws.onmessage = ({ data }) => {
  const message = JSON.parse(data);
  if (message.id) {
    const request = pending.get(message.id);
    pending.delete(message.id);
    if (message.error) request.reject(message.error);
    else request.resolve(message.result);
  } else if (message.method === 'Fetch.requestPaused') {
    const { request, requestId } = message.params;
    const path = new URL(request.url).pathname.slice(4);
    const body = request.postData ? JSON.parse(request.postData) : undefined;
    requests.push({ path, method: request.method, body });
    const [responseCode, payload] = mock(request.method, path, body);
    const responseHeaders = [
      { name: 'Content-Type', value: 'application/json' },
      { name: 'Access-Control-Allow-Origin', value: origin },
      { name: 'Access-Control-Allow-Credentials', value: 'true' },
      { name: 'Access-Control-Allow-Headers', value: 'Content-Type' },
      { name: 'Access-Control-Allow-Methods', value: 'GET,POST,PUT,PATCH,DELETE,OPTIONS' },
    ];
    send('Fetch.fulfillRequest', { requestId, responseCode, responseHeaders, body: payload ? Buffer.from(JSON.stringify(payload)).toString('base64') : '' });
  } else if (message.method === 'Runtime.exceptionThrown') {
    exceptions.push(message.params.exceptionDetails.exception?.description || message.params.exceptionDetails.text);
  } else if (message.method === 'Page.javascriptDialogOpening') {
    dialogs.push(message.params.message);
    send('Page.handleJavaScriptDialog', { accept: allowDiscard });
  }
};

const evaluate = async (expression) => {
  const { result, exceptionDetails } = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
  if (exceptionDetails) throw new Error(exceptionDetails.exception?.description ?? exceptionDetails.text);
  return result.value;
};
async function wait(expression, label = expression) {
  const started = Date.now();
  while (Date.now() - started < 35_000) {
    if (await evaluate(expression)) return;
    await new Promise(resolve => setTimeout(resolve, 170));
  }
  const state = await evaluate('({path:location.pathname, text:document.body.innerText.slice(0,1100)})');
  throw new Error(`Waiting for ${label}: ${JSON.stringify(state)} ${JSON.stringify(exceptions)}`);
}
const exists = selector => wait(`Boolean(document.querySelector(${JSON.stringify(selector)}))`, selector);
const content = text => wait(`document.body?.innerText.includes(${JSON.stringify(text)})`, text);
const click = async (selector) => { await exists(selector); await evaluate(`document.querySelector(${JSON.stringify(selector)}).click()`); };
const link = path => click(`a[href="${path}"]`);
async function input(selector, value, event = 'input') {
  await exists(selector);
  await evaluate(`{const el=document.querySelector(${JSON.stringify(selector)}); el.value=${JSON.stringify(value)}; el.dispatchEvent(new Event(${JSON.stringify(event)}, { bubbles:true }));}`);
}
const open = path => send('Page.navigate', { url: origin + path });
const checkOverflow = () => evaluate('document.documentElement.scrollWidth <= window.innerWidth');
const screenshot = async (name) => {
  const { data } = await send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
  await writeFile(new URL(`file:///C:/Users/Alberto/.codex/visualizations/2026/11/07/01a1b63d-a454-7293-be9f-b796ce581b51/${name}.png`), Buffer.from(data, 'base64'));
};

try {
  await send('Page.enable');
  await send('Runtime.enable');
  await send('Fetch.enable', { patterns: [{ urlPattern: '*://*/api/*' }] });
  await send('Emulation.setDeviceMetricsOverride', { width: 1440, height: 920, deviceScaleFactor: 1, mobile: false });
  await open(base);
  await content('Accedi al workspace');
  await input('input[name="username"]', 'qa-limited');
  await input('input[name="password"]', 'test-only');
  await click('.m-auth button[type="submit"]');
  await content('Accesso riservato');
  await click('.m-auth .m-actions button');
  await content('Accedi al workspace');
  await input('input[name="username"]', 'qa-master');
  await input('input[name="password"]', 'incorrect');
  await click('.m-auth button[type="submit"]');
  await content('Nome utente o password non validi.');
  await input('input[name="password"]', 'test-only');
  await click('.m-auth button[type="submit"]');
  await content('Controllo centrale');
  await wait('document.querySelectorAll(".m-metric strong")[0]?.textContent === "12"', 'summary populated');
  assert(await checkOverflow());
  await screenshot('master-v2-desktop');
  console.log('PASS login, forbidden role, invalid credentials, overview metrics and desktop layout');

  await click('.m-collapse-toggle');
  assert(await evaluate('document.querySelector(".m-root").classList.contains("m-collapsed")'));
  await click('.m-collapse-toggle');
  await link(`${base}/organizations`);
  await content('Aurora 10');
  await click('button[aria-label="Pagina successiva"]');
  await content('Aurora 12');
  await input('input[type="search"]', 'aurora 6');
  await wait('document.querySelectorAll(".m-table tbody tr").length === 1', 'search results');
  await input('input[type="search"]', '');
  await click('.m-table th button');
  await exists('th[aria-sort="ascending"]');
  await link(`${base}/organizations/create`);
  await content('Nuova società');
  await input('main form input[type="text"]', 'Società di prova');
  await link(`${base}/arenas`);
  assert(await evaluate(`location.pathname === '${base}/organizations/create'`));
  assert(dialogs.at(-1).includes('non salvate'));
  allowDiscard = true;
  await link(`${base}/arenas`);
  await content('Arena Nord');
  await link(`${base}/arenas/42`);
  await content('Configurazione 3D');
  await link(`${base}/arenas/42/edit`);
  await exists('textarea');
  assert(await checkOverflow());
  console.log('PASS desktop sidebar, search, sort, pagination, dirty guard, arena detail and editor');

  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 760, deviceScaleFactor: 1, mobile: false });
  await click('.m-mobile-trigger');
  await wait('document.querySelector(".m-main-shell").inert');
  await link(`${base}/organizations`);
  await content('Elenco società');
  await wait('document.querySelectorAll(".m-table tbody tr").length === 10', 'first page returned');
  assert(await checkOverflow());
  assert(await evaluate('document.querySelector(".m-table-scroll").scrollWidth > document.querySelector(".m-table-scroll").clientWidth'));
  await click('button[aria-label^="Elimina Società sportiva"]');
  await exists('dialog[open]');
  assert(await evaluate('document.activeElement?.type === "password"'));
  assert(await evaluate('(() => { const {x,y,right,bottom} = document.querySelector("dialog").getBoundingClientRect(); return x>=0 && y>=0 && right<=innerWidth && bottom<=innerHeight })()'));
  await screenshot('master-v2-mobile-dialog');
  await input('dialog input[type="password"]', 'test-only');
  await click('dialog button[type="submit"]');
  await content('e i dati collegati sono stati eliminati.');
  assert(requests.some(req => req.path.endsWith('/organizations/1') && req.method === 'DELETE' && req.body.password === 'test-only'));
  assert(await checkOverflow());
  console.log('PASS 390px navigation, contained table overflow, responsive dialog, confirmed fixture deletion');

  for (const [path, text] of [['/clubs/25','Club Aurora'], ['/clubs/25/domains','Domini personalizzati'], ['/tournaments/5','Coppa Aurora'], ['/report-branding','Report e marchio'], ['/analytics','Analytics'], ['/qr-redirects','/qr-test'], ['/minigames','Minigiochi']]) {
    await open(base + path);
    await content(text);
    await wait('!document.querySelector(".m-content .m-spinner")', 'page ready');
    assert(await checkOverflow(), path);
  }
  console.log('PASS direct links and mobile viewport for seven section/detail paths');

  session = '';
  await open(base + '/organizations/2');
  await content('Accedi al workspace');
  await open('/admin/master');
  await content('Accedi come super admin');
  assert(await evaluate('Boolean(document.querySelector(".master-portal")) && !document.querySelector(".m-root")'));
  assert.deepEqual(exceptions, []);
  console.log('PASS expired session gate and distinct original Master route, with no uncaught browser exceptions');
} finally {
  await send('Browser.close').catch(() => {});
  ws.close();
  edge.kill();
  await vite.close();
}

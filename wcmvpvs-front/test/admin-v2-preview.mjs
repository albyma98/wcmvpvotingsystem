/**
 * Local UI QA only. Every record is a labelled fixture held in process memory.
 * No Go process, database, network service, SMS or production data is accessed.
 * Run: node test/admin-v2-preview.mjs
 * Open: http://127.0.0.1:5174/societa-test/admin/new (qa / qa).
 */
import { createServer } from 'vite';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { calculateReportKpis } from '../src/engagementReport.js';

const frontRoot = fileURLToPath(new URL('..', import.meta.url));
const now = () => new Date().toISOString();
const image = 'data:image/svg+xml,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="120" height="90"><rect width="120" height="90" fill="#176750"/><text x="60" y="50" font-size="16" fill="white" text-anchor="middle">TEST</text></svg>');

export function createQaApi() {
 let nextId = 100;
 const sessions = new Map(), media = new Map(), quiz = new Map(), questions = new Map(), stories = new Map(), reports = new Map();
 const db = {
  teams: [{ id: 1, name: 'Società di test' }, { id: 2, name: 'Avversari di test' }],
  players: [{ id: 1, first_name: 'Giocatrice', last_name: 'Test Uno', role: 'Schiacciatrice', jersey_number: 7, team_id: 1, image_url: image, is_called_up: true }, { id: 2, first_name: 'Giocatore', last_name: 'Test Due', role: 'Centrale', jersey_number: 12, team_id: 1, image_url: '', is_called_up: true }],
  events: [{ id: 1, team1_id: 1, team2_id: 2, team1_name: 'Società di test', team2_name: 'Avversari di test', start_datetime: '2026-11-05T19:30:00Z', location: 'Palazzetto di test', is_active: true, votes_closed: false, is_concluded: false, show_branded_game: false, show_quiz: true, prizes: [{ id: 1, name: 'Premio di test', position: 1, win_sms_text: 'Premio di prova, nessun messaggio reale.' }] }, { id: 2, team1_id: 1, team2_id: 2, team1_name: 'Società di test', team2_name: 'Avversari di test', start_datetime: '2026-10-29T19:30:00Z', location: 'Palazzetto di test', is_active: false, votes_closed: true, is_concluded: true, prizes: [] }],
  admins: [{ id: 1, username: 'qa', role: 'superadmin' }, { id: 2, username: 'staff', role: 'staff' }, { id: 3, username: 'bar', role: 'bar' }],
  partners: [{ id: 1, username: 'partner-di-test', created_at: now() }],
  sponsors: [{ id: 1, name: 'Sponsor di test', report_name: 'Sponsor di test', position: 1, logo_data: image, link_url: 'https://example.invalid', is_active: true }],
  coupons: [{ id: 1, title: 'Vantaggio di test', short_desc: 'Coupon dimostrativo per QA locale.', sponsor_id: 1, match_ids: [1], status: 'active', max_uses: 50, total_views: 8, total_claims: 3, total_redemptions: 1 }],
  categories: [{ id: 1, name: 'Bevande di test', image_url: image }],
  products: [{ id: 1, name: 'Acqua di test', description: 'Prodotto QA', price_cents: 150, category_id: 1, category: 'Bevande di test', image_url: image }],
  menus: [{ id: 1, name: 'Menù di test', description: 'Pacchetto QA', price_cents: 250, items: [{ product_id: 1, quantity: 2 }] }],
  orders: [{ id: 1, order_status: 'new', payment_status: 'paid', total_cents: 300, customer_name: 'Cliente di test', sector: 'A', row: '2', seat: '12', notes: 'Ordine QA locale', created_at: now(), updated_at: now(), products: JSON.stringify([{ id: 1, name: 'Acqua di test', price_cents: 150 }]), quantities: JSON.stringify([{ id: 1, quantity: 2 }]) }],
  audience: [{ fan_id: 1, nickname: 'Tifoso di test', first_name: 'Tifoso', last_name: 'Test', phone: '+390000000000', gender: 'M', marketing_consent: true, coins: 120, created_at: '2026-10-01T12:00:00Z', last_seen_at: now() }],
  templates: [{ id: 1, name: 'Template di test', body: 'Messaggio per QA locale, senza invio reale.', category: 'match' }],
  campaigns: [], logs: [], selfies: [{ id: 1, event_id: 1, caption: 'Selfie di test', image_url: '/qa-image.svg', accepted_image_terms: true, approved: false, show_on_screen: false, submitted_at: now(), file_size_bytes: 200, device_token: 'dispositivo-di-test' }],
 };
 db.events.forEach(event => event.prizes.forEach(prize => { prize.event_id = event.id; }));
 let rosterSchema = 13;
 let voting = { template_sid: 'HX' + 'a'.repeat(32), draw_mode: 'random', show_lottery_ticket: true, recipients: [{ name: 'Organizzatore di test', phone: '+390000000000', enabled: true }] };
 const suggestionProducts = new Map(), suggestionCategories = new Map();
 const billing = { sms_cost: 0.08, free_sms_remaining: 10, total_messages: 0, total_cost_charged: 0 };
 const analytics = { total_users: 12, seen_users: 10, watched_users: 8, average_watch_time_ms: 5000, total_watch_time_ms: 40000, total_clicks: 3, unique_clickers: 2, seen_rate: 83.33, click_rate: 20, top_sponsor: { sponsor_id: 1, name: 'Sponsor di test', views: 10 }, timeline: [{ timestamp: now(), seen: 10, watched: 8, clicks: 3 }] };
 function result(res, data = {}, status = 200) { res.statusCode = status; res.setHeader('Cache-Control', 'no-store'); res.setHeader('Content-Type', 'application/json; charset=utf-8'); res.end(status === 204 ? undefined : JSON.stringify(data)); }
 function collection(req, res, name, id, body) {
  const rows = db[name];
  if (req.method === 'GET' && !id) return result(res, rows);
  if (req.method === 'POST' && !id) { const row = { ...body, id: nextId++, created_at: now() }; rows.push(row); return result(res, row, 201); }
  const index = rows.findIndex(row => row.id === Number(id));
  if (index < 0) return result(res, { message: 'Elemento di test non trovato' }, 404);
  if (req.method === 'PUT') { Object.assign(rows[index], body, { updated_at: now() }); return result(res, rows[index]); }
  if (req.method === 'DELETE') { rows.splice(index, 1); return result(res); }
  return result(res, { message: 'Metodo non previsto per questa fixture' }, 405);
 }
 function closure(eventId) { return { summary: { event_id: eventId, draw_mode: voting.draw_mode, mvp_name: 'Giocatrice Test Uno', percentage: '60,00', total_votes: 10, prizes: [{ id: 1, name: 'Premio di test', ticket_code: 'TEST-0001', nickname: 'Tifoso di test' }], warnings: ['Ambiente QA: nessun premio o messaggio reale.'] }, notifications: [{ id: 1, name: 'Organizzatore di test', phone: '+39******0000', status: 'skipped', error: 'Invio disabilitato nell’ambiente QA locale.' }] }; }
 function report(eventId) { return { calculation_id: `qa-${eventId}`, event: { id: eventId, title: 'Società di test - Avversari di test', start_datetime: '2026-10-29T19:30:00Z' }, branding: { company_name: 'Società di test' }, calculated_at: now(), facts: { visitors: 12, engagement_eligible_visitors: 12, active_visitors: 8 }, kpis: [{ id: 'visitors', value: 12, display_description: 'Dati di test per QA locale' }, { id: 'active_participation', value: 66.67, display_description: 'Dati di test per QA locale' }], latest_report: reports.get(eventId) || null }; }
 return async (req, res) => {
  const url = new URL(req.url, 'http://127.0.0.1'); const route = url.pathname; const method = req.method;
  console.log(`[QA memoria] ${method} /api${route}`);
  try {
   if (route === '/v1/club-domain' && method === 'GET') return result(res, null, 204);
   if (route === '/qa-image.svg' && method === 'GET') { res.setHeader('Content-Type', 'image/svg+xml'); return res.end(decodeURIComponent(image.split(',')[1])); }
   if (route.startsWith('/qa-media/') && method === 'GET') { const value = media.get(route); if (!value) return result(res, {}, 404); res.setHeader('Content-Type', value.type); return res.end(value.bytes); }
   const chunks = []; for await (const chunk of req) chunks.push(chunk); const raw = Buffer.concat(chunks);
   const body = raw.length && String(req.headers['content-type']).includes('application/json') ? JSON.parse(raw.toString()) : {};
   if (route === '/admin/login' && method === 'POST') {
    if (!['qa', 'staff', 'bar'].includes(body.username) || body.password !== 'qa') return result(res, { message: 'Usa qa/qa, staff/qa oppure bar/qa nel test locale.' }, 401);
    const role = body.username === 'qa' ? 'superadmin' : body.username; const token = `qa-${nextId++}`; sessions.set(token, { username: body.username, role }); res.setHeader('Set-Cookie', `abx_qa=${token}; Path=/; HttpOnly; SameSite=Lax`); return result(res, { username: body.username, role });
   }
   const token = /(?:^|;\s*)abx_qa=([^;]+)/.exec(req.headers.cookie || '')?.[1]; const session = sessions.get(token);
   if (route === '/admin/logout' && method === 'POST') { sessions.delete(token); res.setHeader('Set-Cookie', 'abx_qa=; Path=/; Max-Age=0'); return result(res); }
   if (!session) return result(res, { message: 'Sessione QA non presente' }, 401);
   if (route === '/admin/me' && method === 'GET') return result(res, { ...session, organization_name: 'Società di test', organization_slug: String(req.headers['x-organization-slug'] || 'societa-test') });
   if (route === '/players/settings' && method === 'PUT') { rosterSchema = body.roster_schema; return result(res, { roster_schema: rosterSchema }); }
   if (route === '/players' && method === 'GET') return result(res, { players: db.players, roster_schema: rosterSchema, bar_enabled: true });
   if (route === '/events/deactivate' && method === 'POST') { db.events.forEach(event => { event.is_active = false; }); return result(res); }
   if (route === '/admin/events/history' && method === 'GET') return result(res, db.events.filter(event => event.is_concluded).map(event => ({ ...event, title: `${event.team1_name} - ${event.team2_name}`, home_team: event.team1_name, away_team: event.team2_name, total_votes: 10, total_visitors: 12, unique_visitors: 10, sponsor_clicks_total: 3, sponsor_analytics: analytics, sponsor_clicks: [{ sponsor_id: 1, name: 'Sponsor di test', clicks: 3 }], mvp: { name: 'Giocatrice Test Uno', votes: 6 }, prizes: event.prizes, engagement: { total_duration_seconds: 200, average_duration_seconds: 20, total_users: 10 } })));
   let match;
   if ((match = route.match(/^\/events\/(\d+)\/(activate|conclude|close-votes)$/)) && method === 'POST') { const event = db.events.find(item => item.id === Number(match[1])); if (!event) return result(res, {}, 404); if (match[2] === 'activate') { db.events.forEach(item => { item.is_active = false; }); Object.assign(event, { is_active: true, votes_closed: false }); } if (match[2] === 'conclude') Object.assign(event, { is_active: false, is_concluded: true }); if (match[2] === 'close-votes') event.votes_closed = true; return result(res); }
   if ((match = route.match(/^\/events\/(\d+)\/results$/)) && method === 'GET') return result(res, [{ player_id: 1, votes: 6, last_vote_at: now() }, { player_id: 2, votes: 4, last_vote_at: now() }]);
   if ((match = route.match(/^\/events\/(\d+)\/tickets$/)) && method === 'GET') { res.setHeader('X-Lottery-Total-Codes', '1'); return result(res, [{ vote_id: 77, ticket_code: 'TEST-0001', player_id: 1, player_first_name: 'Giocatrice', player_last_name: 'Test Uno', created_at: now() }]); }
   if ((match = route.match(/^\/events\/(\d+)\/prizes\/(\d+)\/(assign|winner)$/))) { const event = db.events.find(item => item.id === Number(match[1])); const prize = event?.prizes?.find(item => item.id === Number(match[2])); if (!prize) return result(res, {}, 404); prize.event_id = event.id; if (method === 'POST' && match[3] === 'assign') prize.winner = { vote_id: body.vote_id, ticket_code: 'TEST-0001', assigned_at: now(), status: 'assigned' }; else if (method === 'DELETE' && match[3] === 'winner') delete prize.winner; else return result(res, {}, 405); return result(res, prize); }
   if ((match = route.match(/^\/events\/(\d+)\/voting-closure(?:\/retry)?$/))) return result(res, closure(Number(match[1])));
   if ((match = route.match(/^\/admin\/events\/(\d+)\/sponsors\/analytics$/)) && method === 'GET') return result(res, analytics);
   if ((match = route.match(/^\/admin\/events\/(\d+)\/selfies$/)) && method === 'GET') return result(res, db.selfies.filter(item => item.event_id === Number(match[1])));
   if ((match = route.match(/^\/admin\/selfies\/(\d+)$/))) return collection(req, res, 'selfies', match[1], body);
   if ((match = route.match(/^\/admin\/events\/(\d+)\/purge$/)) && method === 'POST') { if (body.password !== 'qa') return result(res, { message: 'Password QA non valida' }, 403); return collection({ method: 'DELETE' }, res, 'events', match[1], {}); }
   if ((match = route.match(/^\/admin\/events\/(\d+)\/ai-report$/)) && method === 'POST') return result(res, { source: 'qa-fixture', executive_summary: 'Report di test, nessun servizio AI chiamato.', full_report: 'Questi contenuti sono fixture per controllare l’interfaccia.', strengths: ['Navigazione di prova'], criticalities: [], insights: [], suggestions: [], generated_at: now(), metrics: {} });
   if ((match = route.match(/^\/admin\/events\/(\d+)\/engagement-report$/))) { const id = Number(match[1]); const snapshot = report(id); if (method === 'GET') return result(res, snapshot); if (method === 'POST') { const approved = { ...snapshot, ...body, id: nextId++, approved_at: now(), kpis: calculateReportKpis(body.facts, snapshot.kpis, body.overrides, snapshot.facts, body.descriptions) }; reports.set(id, approved); return result(res, approved); } }
   if (route === '/admin/voting-settings') { if (method === 'PUT') voting = body; return result(res, voting); }
   if (route === '/admin/voting-settings/test' && method === 'POST') return result(res, { message: 'QA locale: nessun WhatsApp inviato.', status: 'skipped' });
   if ((match = route.match(/^\/admin\/events\/(\d+)\/quiz(?:\/questions(?:\/(\d+))?)?$/))) {
    const id = Number(match[1]);
    if (!route.includes('/questions')) { if (method === 'PUT') quiz.set(id, body); return result(res, quiz.get(id) || { event_id: id, enabled: false, questions_per_session: 5, seconds_per_question: 8, base_reward: 3, completion_bonus: 5, streak_bonus: 1, active_from: '', active_to: '' }); }
    const rows = questions.get(id) || []; questions.set(id, rows);
    if (method === 'GET') return result(res, rows);
    if (method === 'POST') { const row = { ...body, id: nextId++ }; rows.push(row); return result(res, row); }
    const index = rows.findIndex(item => item.id === Number(match[2])); if (index < 0) return result(res, {}, 404); if (method === 'DELETE') rows.splice(index, 1); if (method === 'PUT') Object.assign(rows[index], body); return result(res, rows[index] || {});
   }
   if ((match = route.match(/^\/admin\/events\/(\d+)\/stories(?:\/(\d+|upload-video))?$/))) {
    const id = Number(match[1]); const rows = stories.get(id) || []; stories.set(id, rows);
    if (match[2] === 'upload-video' && method === 'POST') { const boundary = /boundary=(.+)$/.exec(String(req.headers['content-type']))?.[1]; if (!boundary) return result(res, { message: 'File multipart mancante' }, 400); const start = raw.indexOf(Buffer.from('\r\n\r\n')) + 4; const end = raw.indexOf(Buffer.from(`\r\n--${boundary}`), start); const type = /Content-Type: ([^\r\n]+)/i.exec(raw.subarray(0, start).toString())?.[1] || 'video/mp4'; const key = `/qa-media/${nextId++}`; media.set(key, { type, bytes: raw.subarray(start, end < 0 ? raw.length : end) }); return result(res, { video_url: `/api${key}` }); }
    if (method === 'GET') return result(res, rows);
    if (method === 'POST') { const row = { ...body, id: nextId++ }; rows.push(row); return result(res, row); }
    const index = rows.findIndex(item => item.id === Number(match[2])); if (index < 0) return result(res, {}, 404); if (method === 'DELETE') rows.splice(index, 1); if (method === 'PUT') Object.assign(rows[index], body); return result(res, rows[index] || {});
   }
   if (route === '/admin/bar/overview' && method === 'GET') { const total = db.orders.reduce((sum, order) => sum + order.total_cents, 0); return result(res, { orders_received: db.orders.length, orders_pending: db.orders.filter(order => order.order_status === 'new').length, orders_in_preparation: db.orders.filter(order => order.order_status === 'in_preparazione').length, orders_ready: db.orders.filter(order => order.order_status === 'pronto').length, orders_completed: db.orders.filter(order => order.order_status === 'completato').length, revenue_cents: total, avg_ticket_cents: db.orders.length ? total / db.orders.length : 0, latest_orders: db.orders }); }
   if ((match = route.match(/^\/admin\/bar\/orders\/(\d+)\/status$/)) && method === 'PUT') { const order = db.orders.find(item => item.id === Number(match[1])); if (!order) return result(res, {}, 404); Object.assign(order, { order_status: body.status, updated_at: now() }); return result(res, order); }
   if ((match = route.match(/^\/admin\/bar\/suggestions\/(products|categories)(?:\/(\d+))?$/))) { const map = match[1] === 'products' ? suggestionProducts : suggestionCategories; const key = match[1] === 'products' ? 'product_id' : 'category_id'; if (method === 'PUT') map.set(Number(match[2]), { ...body, [key]: Number(match[2]) }); return result(res, [...map.values()]); }
   if ((match = route.match(/^\/admin\/bar\/(products|categories|menus|orders)(?:\/(\d+))?$/))) return collection(req, res, match[1], match[2], body);
   if (route === '/admin/marketing/audience' && method === 'GET') { const search = (url.searchParams.get('q') || '').toLowerCase(); return result(res, db.audience.filter(fan => JSON.stringify(fan).toLowerCase().includes(search))); }
   if (route === '/admin/marketing/billing' && method === 'GET') return result(res, billing);
   if (route === '/admin/marketing/logs' && method === 'GET') return result(res, { items: db.logs, summary: billing });
   if ((match = route.match(/^\/admin\/marketing\/templates(?:\/(\d+))?$/))) return collection(req, res, 'templates', match[1], body);
   if (route === '/admin/marketing/campaigns' && method === 'POST') { const campaign = { ...body, id: nextId++ }; db.campaigns.push(campaign); return result(res, { campaign }); }
   if ((route === '/admin/marketing/sms/single' || /^\/admin\/marketing\/campaigns\/\d+\/(send-now|test)$/.test(route)) && method === 'POST') { db.logs.unshift({ id: nextId++, status: 'skipped', message: 'QA locale: nessun SMS inviato.', created_at: now(), fan_id: body.fan_id, phone_masked: '+39******0000' }); return result(res, { status: 'skipped', message: 'Nessun invio reale nel test locale.' }); }
   if ((match = route.match(/^\/(teams|players|events|admins)(?:\/(\d+))?$/))) { if (match[1] === 'events' && method === 'POST') Object.assign(body, { team1_name: db.teams.find(team => team.id === body.team1_id)?.name, team2_name: db.teams.find(team => team.id === body.team2_id)?.name, is_active: false, is_concluded: false, votes_closed: false }); return collection(req, res, match[1], match[2], body); }
   if ((match = route.match(/^\/admin\/(partners|sponsors|coupons)(?:\/(\d+))?$/))) return collection(req, res, match[1], match[2], body);
   return result(res, { message: `Endpoint non implementato nel test locale: ${method} ${route}` }, 501);
  } catch (error) { console.error('[QA errore]', error.message); result(res, { message: error.message }, 500); }
 };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
 process.env.VITE_API_BASE_URL = '/api';
 const port = Number(process.env.ADMIN_V2_QA_PORT || 5187);
 const api = createQaApi();
 const server = await createServer({ root: frontRoot, configFile: path.join(frontRoot, 'vite.config.js'), server: { host: 'localhost', port, strictPort: true, proxy: { '/api': { target: 'http://127.0.0.1:9' } } }, plugins: [{ name: 'admin-v2-memory-qa-only', enforce: 'pre', configureServer(vite) { vite.middlewares.use('/api', api); } }] });
 await server.listen();
 console.log(`QA locale con fixture in memoria: http://localhost:${port}/societa-test/admin/new`);
 console.log('Account di test: qa/qa (superadmin), staff/qa, bar/qa. Nessun invio SMS o database reale.');
}

import test, { after, before } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import * as Vue from 'vue';
import { renderToString } from '@vue/server-renderer';
import { compileAdminV2Controller } from '../../build/adminV2Controllers.js';

// Render the actual Vue views against the actual legacy normalizers. These
// fixtures cover server payloads, including nested historical analysis shapes.
const portal = new URL('../components/AdminPortal.vue', import.meta.url);
const compiled = compileAdminV2Controller(readFileSync(portal, 'utf8'), fileURLToPath(portal));
const args = [], values = [];
let calls = [];
const code = compiled.replace(/^import (.+) from (".+");$/gm, (_, bindings, moduleJSON) => {
  const module = JSON.parse(moduleJSON);
  const provider = module === 'vue' ? Vue : module.endsWith('/api') ? {
    apiClient: { get: async path => { calls.push(path); return { data: [] }; }, post: async path => { calls.push(path); return { data: {} }; } },
    resolveApiUrl: value => value, getOrganizationSlug: () => 'club',
  } : { DEFAULT_ROSTER_SCHEMA: 13, MAX_PLAYER_SLOTS: 14 };
  for (const entry of bindings.replace(/[{}]/g, '').split(',')) {
    const [name, alias = name] = entry.trim().split(/\s+as\s+/);
    args.push(alias); values.push(provider[name]);
  }
  return '';
}).replaceAll('import.meta.env.BASE_URL', "'/'").replace('export function useController', 'return function useController');
const createController = new Function(...args, 'localStorage', 'window', code)(...values,
  { getItem: () => '', setItem() {}, removeItem() {} },
  { location: { origin: 'https://example.test', pathname: '/club/admin/new' }, addEventListener() {}, removeEventListener() {} },
);

let server, Entities, Insights, MatchdayMetricCard;
before(async () => {
  server = await createServer({ server: { middlewareMode: true }, appType: 'custom', logLevel: 'error', optimizeDeps: { noDiscovery: true } });
  Entities = (await server.ssrLoadModule('/src/admin-v2/views/EntitiesView.vue')).default;
  Insights = (await server.ssrLoadModule('/src/admin-v2/views/InsightsView.vue')).default;
  MatchdayMetricCard = (await server.ssrLoadModule('/src/admin-v2/components/reports/MatchdayMetricCard.vue')).default;
});
after(async () => { await server?.close(); });

async function render(section, populate) {
  calls = [];
  const app = Vue.createSSRApp({ setup() {
    const raw = createController(Vue.reactive({ organizationSlug: 'club' }), () => {}, {});
    const c = Vue.proxyRefs({ ...raw, busy: () => false, run: async (_, action) => action(), confirmAction() {}, notify() {} });
    c.teams = [{ id: 1, name: 'Club di casa' }, { id: 2, name: 'Ospiti' }];
    c.events = [c.normalizeEventResponse({ id: 7, team1_id: 1, team2_id: 2, start_datetime: '2026-11-01T18:00:00Z', location: 'Palasport' })];
    c.partners = [c.normalizePartnerResponse({ id: 3, username: 'esercente', created_at: '2026-10-01T10:00:00Z' })];
    c.newCoupon.sponsorId = 3;
    populate?.(c);
    Vue.provide('admin-v2', c);
    return () => Vue.h(['history', 'results', 'selfies'].includes(section) ? Insights : Entities, { section });
  } });
  const html = await renderToString(app);
  assert.deepEqual(calls, [], 'Rendering a view must not issue mutations or bypass session loading');
  return html;
}

test('Sponsor Analytics offers concluded matches even when there are no open matches', async () => {
  const html = await render('results', c => {
    c.events = [c.normalizeEventResponse({
      id: 42, team1_id: 1, team2_id: 2, start_datetime: '2026-10-04T18:00:00Z', is_concluded: true,
    })];
    assert.equal(c.availableEvents.length, 0);
  });
  assert.match(html, /Sponsor Analytics<\/h2>[\s\S]*?<option\b[^>]*\bvalue="42"/);
});

test('the archive renders a collapsed Sponsor Analytics section for each event', async () => {
  const html = await render('history', c => {
    c.eventHistory = [42, 43].map(id => c.normalizeHistoryEntry({
      id, title: `Partita ${id}`, start_datetime: '2026-10-04T18:00:00Z',
    }));
  });
  assert.match(html, /Partita 42[\s\S]*?Sponsor Analytics<\/summary>[\s\S]*?Partita 43[\s\S]*?Sponsor Analytics<\/summary>/);
  assert.doesNotMatch(html, /Tutti gli sponsor/);
});

test('the archive offers KPI Matchday for every event without loading the dialog', async () => {
  const html = await render('history', c => {
    c.eventHistory = [42, 43].map(id => c.normalizeHistoryEntry({
      id, title: `Partita ${id}`, start_datetime: '2026-10-04T18:00:00Z',
    }));
  });
  assert.equal((html.match(/KPI Matchday/g) || []).length, 2);
  assert.match(html, /Partita 42[\s\S]*?KPI Matchday[\s\S]*?<details class="v2-history-disclosure"/);
  assert.doesNotMatch(html, /Matchday KPI Analytics/);
});

test('Matchday displays measured zero, missing data, and distinct shop conversions', async () => {
  const event = { id: 42, title: 'Finale', start_datetime: '2026-10-04T18:00:00Z' };
  const metrics = [
    { code: 'qr_conversion', name: 'Conversione QR', status: 'available', source: 'mixed', value: 0, unit: 'percent' },
    { code: 'penetration', name: 'Penetrazione', status: 'missing', value: null, unit: 'percent' },
    { code: 'shop_conversions', name: 'Conversioni shop', status: 'available', source: 'db', value: null,
      columns: [{ id: 'count', label: 'Numero', unit: 'count' }],
      rows: [{ label: 'Acquisti confermati', values: { count: 2 } }, { label: 'Prenotazioni confermate', values: { count: 3 } }] },
  ];
  const app = Vue.createSSRApp({ render: () => Vue.h('div', metrics.map(metric => Vue.h(MatchdayMetricCard, { metric, event }))) });
  const html = await renderToString(app);
  assert.match(html, /Conversione QR[\s\S]*?Misto[\s\S]*?>0%<\/strong>/);
  assert.match(html, /Penetrazione[\s\S]*?Dato mancante/);
  assert.match(html, /Acquisti confermati<\/dt><dd\b[^>]*>2<\/dd>[\s\S]*?Prenotazioni confermate<\/dt><dd\b[^>]*>3<\/dd>/);
});

test('history renders populated server payloads, funnel steps, survey text, prizes and tracking metadata', async () => {
  const html = await render('history', c => {
    c.eventHistory = [c.normalizeHistoryEntry({
      id: 8, title: 'Finale di campionato', start_datetime: '2026-10-04T18:00:00Z', location: 'Arena centrale', total_votes: 43,
      mvp: { player_id: 2, first_name: 'Anna', last_name: 'Rossi', votes: 31 },
      prizes: [{ id: 1, position: 1, name: 'Maglia firmata', winner_ticket_code: 'ab1234' }],
      sponsor_clicks: [{ sponsor_id: 5, name: 'Partner pubblico', report_name: 'Partner nel report', clicks: 13 }],
      sponsor_analytics: { total_users: 60, seen_users: 41, watched_users: 24, total_clicks: 13, unique_clickers: 9, average_watch_time_ms: 4500, total_watch_time_ms: 108000, seen_rate: 68, click_rate: 22,
        top_sponsor: { sponsor_id: 5, name: 'Partner pubblico', report_name: 'Partner nel report', views: 37 },
        timeline: [{ timestamp: '2026-10-04T18:15:00Z', seen: 41, watched: 24, clicks: 13 }] },
      feedback_summary: { total_responses: 2, experience: { very_easy: 2 }, suggestions: ['Più quiz durante l’intervallo'] },
      engagement: { total_duration_seconds: 240, average_duration_seconds: 120, total_users: 2, selfie_opens: 3, vote_edit_completions: 1 },
      tracking_events: [{ name: 'vote_submitted', count: 43, unique_sessions: 40, unique_devices: 38, unique_fans: 30, pages: ['home/new'], sections: ['mvp'], sources: ['arena'], domains: ['club.test'], login_states: ['registered'], profile_states: ['complete'], metadata_samples: ['{"source":"arena"}'] }],
      tracking_analytics: { unique_sessions: 60, total_events: 160, vote_submitted_count: 43, feedback_submitted_count: 2, sponsor_clicked_count: 13, bar_order_completed_count: 8,
        funnels: [{ name: 'Percorso MVP', steps: [{ key: 'open', label: 'Apertura voto', count: 52 }, { key: 'submit', label: 'Voto completato', count: 43 }] }],
        segment_breakdown: { guest: { submitted_votes: 3, feedback_submitted: 0, sponsor_clicks: 1 }, registered: { submitted_votes: 40, feedback_submitted: 2, sponsor_clicks: 12 } },
        top_event_names: [{ name: 'vote_submitted', count: 43 }], by_event_domain: { mvp: 80 } },
      ai_report: { executive_summary: 'Partecipazione costante', full_report: 'Analisi completa della partita', strengths: ['Buona conversione'], criticalities: ['Poche risposte aperte'], insights: ['Picco nel secondo tempo'], suggestions: ['Promuovere il quiz'] },
      timeline: [{ start: '2026-10-04T18:00:00Z', end: '2026-10-04T18:15:00Z', votes: 43 }],
    })];
  });
  for (const text of ['Finale di campionato', 'Anna Rossi', 'Maglia firmata', 'AB1234', 'Partner nel report', 'Più quiz durante l’intervallo', 'Percorso MVP', 'Apertura voto', 'Voto completato', 'vote_submitted', 'home/new', 'club.test', 'Partecipazione costante', 'Promuovere il quiz', '43 voti', 'Prepara report PDF']) assert.ok(html.includes(text), text);
  assert.doesNotMatch(html, /\[object Object\]|undefined|NaN/);
});

test('coupon editor uses normalized dates, merchant, image, event assignment and actual status choices', async () => {
  const html = await render('coupons', c => {
    c.coupons = [c.toEditableCoupon({ id: 4, title: 'Vantaggio tifosi', short_desc: 'Sconto al bar', sponsor_id: 3, merchant_id: 3, match_ids: [7], start_date: '2026-11-01T10:00:00Z', end_date: '2026-12-01T10:00:00Z', max_uses: 50, status: 'active', image_url: 'https://example.test/coupon.png', total_views: 11, total_claims: 4, total_redemptions: 2 })];
  });
  for (const text of ['Vantaggio tifosi', 'Sconto al bar', 'esercente', 'datetime-local', 'Eventi associati', 'coupon.png', '50', 'visualizzazioni', 'richieste', 'utilizzi']) assert.ok(html.includes(text), text);
  for (const status of ['draft', 'active', 'paused', 'archived']) assert.ok(html.includes(`value="${status}"`), status);
  assert.doesNotMatch(html, /\[object Object\]|undefined|NaN/);
});

test('roster renders normalized player controls and retains all fourteen slots and existing schemas', async () => {
  const html = await render('players', c => {
    c.players = [c.normalizePlayerResponse({ id: 10, first_name: 'Giulia', last_name: 'Bianchi', jersey_number: 9, role: 'Palleggiatrice', team_id: 1, is_called_up: true, image_url: 'https://example.test/player.jpg' })];
    c.applyPlayersToSlots();
  });
  for (const text of ['Giulia', 'Bianchi', 'Palleggiatrice', 'Club di casa', 'Convocato per la pagina di voto', 'player.jpg', 'Salva roster e convocazioni', 'Ripristina dati salvati']) assert.ok(html.includes(text), text);
  assert.equal((html.match(/class="v2-player-heading"/g) || []).length, 14);
  for (const schema of [12, 13, 14]) assert.ok(html.includes(`value="${schema}"`));
  assert.match(html, /<form[^>]*class="v2-stack"/);
});

test('selfie moderation renders normalized image, consent and deletion control without mutating data', async () => {
  const html = await render('selfies', c => {
    c.selectedSelfieEventId = 7;
    c.eventSelfies = [c.normalizeSelfieResponse({ id: 9, event_id: 7, caption: 'Forza squadra', image_url: 'https://example.test/selfie.jpg', submitted_at: '2026-11-01T19:00:00Z', device_token: 'device-test', file_size_bytes: 2048, approved: true, accepted_image_terms: true })];
  });
  for (const text of ['Forza squadra', 'selfie.jpg', 'device-test', 'Accettato', 'Elimina foto']) assert.ok(html.includes(text), text);
});

import test, { after, before } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import * as Vue from 'vue';
import { renderToString } from '@vue/server-renderer';
import { compileAdminV2Controller } from '../../build/adminV2Controllers.js';

const portal = new URL('../components/AdminPortal.vue', import.meta.url);
const compiled = compileAdminV2Controller(readFileSync(portal, 'utf8'), fileURLToPath(portal));
const args = [], values = [];
let requests = [];
const code = compiled.replace(/^import (.+) from (".+");$/gm, (_, bindings, moduleJSON) => {
 const module = JSON.parse(moduleJSON);
 const provider = module === 'vue' ? Vue : module.endsWith('/api') ? {
  apiClient: { get: async path => { requests.push(path); return { data: [] }; }, post: async path => { requests.push(path); return { data: {} }; } },
  resolveApiUrl: value => value, getOrganizationSlug: () => 'club',
 } : { DEFAULT_ROSTER_SCHEMA:13, MAX_PLAYER_SLOTS:14 };
 for (const binding of bindings.replace(/[{}]/g,'').split(',')) { const [name,alias=name] = binding.trim().split(/\s+as\s+/); args.push(alias); values.push(provider[name]); }
 return '';
}).replaceAll('import.meta.env.BASE_URL', "'/'").replace('export function useController', 'return function useController');
const createController = new Function(...args,'localStorage','window',code)(...values,
 { getItem:()=>'', setItem(){}, removeItem(){} },
 { location:{origin:'https://example.test',pathname:'/club/admin/new'}, addEventListener(){}, removeEventListener(){} },
);
let server, Events;
before(async()=>{ server=await createServer({server:{middlewareMode:true}, appType:'custom', logLevel:'error', optimizeDeps:{noDiscovery:true}}); Events=(await server.ssrLoadModule('/src/admin-v2/views/EventsView.vue')).default; });
after(async()=>{await server?.close();});
async function render(detail='',populate){
 requests=[];
 const app=Vue.createSSRApp({setup(){
  const raw=createController(Vue.reactive({organizationSlug:'club'}),()=>{},{});
  const c=Vue.proxyRefs({...raw,busy:()=>false,run:async(_,action)=>action(),confirmAction(){},notify(){},navigate(){}});
  c.teams=[{id:1,name:'Società di casa'},{id:2,name:'Ospiti'}];
  c.events=[c.normalizeEventResponse({id:7,team1_id:1,team2_id:2,start_datetime:'2026-11-01T18:00:00Z',location:'Arena centrale',prizes:[{id:3,name:'Maglia firmata',win_sms_text:'Hai vinto!'}],show_branded_game:true,branded_game_config:JSON.stringify({...c.defaultBrandedGameConfig(),sponsor_id:'partner-1',sponsor_name:'Sponsor reale',game_type:'stack_it',stack_it_config:{...c.defaultStackItConfig(),block_texture_url:'https://example.test/texture.png',speed_curve:'aggressive'}})})];
  c.syncEventDrafts(c.events);
  populate?.(c);
  Vue.provide('admin-v2',c);
  return()=>Vue.h(Events,{detail});
 }});
 const html=await renderToString(app);
 assert.deepEqual(requests,[],'Rendering may not mutate or independently load tenant resources');
 assert.doesNotMatch(html,/\[object Object\]|undefined|NaN/);
 return html;
}

test('events list renders real matches, search and state filters',async()=>{
 const html=await render();
 for(const text of ['Società di casa','Ospiti','Arena centrale','Cerca evento','Tutti gli stati','Votazioni chiuse','Crea evento','Apri storico e report']) assert.ok(html.includes(text),text);
});
test('event creation provides team choices, native datetime and prize fields',async()=>{
 const html=await render('create');
 for(const text of ['Squadra di casa','Squadra ospite','datetime-local','Testo SMS vittoria','Aggiungi premio','Crea evento','Sponsor a bordo campo','Gioco dei riflessi']) assert.ok(html.includes(text),text);
 assert.equal((html.match(/value="1"/g)||[]).length,2);
});
test('event experience preserves all branded types and the actual Stack It values',async()=>{
 const html=await render('7/settings');
 for(const value of ['tap_challenge','memory_flash','sponsor_rush','free_throw_challenge','stack_it','gentle','standard','aggressive']) assert.ok(html.includes(`value="${value}"`),value);
 for(const text of ['Sponsor reale','partner-1','texture.png','Suono per incastro perfetto','Suono a fine partita','Velocità iniziale pendolo','Monete per blocco','Partite massime per tifoso','Salva esperienza']) assert.ok(html.includes(text),text);
});
test('event prizes render saved prize and winner protection',async()=>{
 const html=await render('7/prizes',c=>{c.eventPrizeDrafts[7][0].winner={ticketCode:'WIN123'};});
 for(const text of ['Maglia firmata','Hai vinto!','WIN123','Salva premi','Apri lotteria']) assert.ok(html.includes(text),text);
 assert.match(html,/<button[^>]*disabled[^>]*>Rimuovi premio 1/);
});
test('event survey renders all fixed question IDs and preserves answer codes',async()=>{
 const html=await render('7/feedback');
 for(const text of ['very_easy','super_excited','Domanda 4','Domanda per i suggerimenti','maxlength="120"','Salva sondaggio']) assert.ok(html.includes(text),text);
});
test('event quiz renders actual questions and zero-based correct answer selection',async()=>{
 const html=await render('7/quiz',c=>{c.quizQuestionsByEvent[7]=[{id:4,question_text:'Chi ha vinto il campionato?',answers:['Squadra A','Squadra B','Squadra C'],correct_index:1,order_index:3}];});
 for(const text of ['Chi ha vinto il campionato?','Squadra B','Risposta corretta','Ordine della domanda','Rimuovi ultima risposta','Salva domanda','Salva configurazione quiz']) assert.ok(html.includes(text),text);
 assert.match(html,/<option value="1"[^>]*selected>2 · Squadra B/);
});
test('event stories render upload, media URLs, ordering and publish controls',async()=>{
 const html=await render('7/stories',c=>{c.eventStoriesById[7]=[{id:8,title:'Dietro le quinte',player_name:'Giulia',thumbnail_url:'https://example.test/story.jpg',video_url:'https://example.test/story.mp4',is_active:true,order_index:0}];});
 for(const text of ['Dietro le quinte','Giulia','story.jpg','story.mp4','video/mp4,video/webm,video/quicktime','Sposta su','Sposta giù','Visibile ai tifosi','Salva story','Elimina story']) assert.ok(html.includes(text),text);
});
test('unknown event detail gives an actionable empty state',async()=>{
 const html=await render('999');
 assert.ok(html.includes('Evento non disponibile'));
 assert.ok(html.includes('Apri storico'));
});

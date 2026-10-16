<template>
  <section class="fan-collection" aria-label="La tua collezione tifoso" :aria-busy="loading || busy">
    <article class="fan-pass" :class="{ 'has-club-frame': equippedFrame === 'club_frame' }" :aria-label="`Tessera tifoso di ${displayNickname}`">
      <span class="pass-eyebrow">LA TUA TESSERA TIFOSO</span>
      <img v-if="clubLogoUrl" class="pass-club-logo" :src="clubLogoUrl" :alt="clubName || 'Logo della società'" loading="lazy" />
      <div class="pass-portrait" aria-hidden="true"><span>{{ initials }}</span><svg viewBox="0 0 100 100" fill="none"><path d="M14 78 35 31l15-9 15 9 21 47M26 79h48M34 47h32M37 60h26" stroke="currentColor" stroke-width="1.5"/><circle cx="50" cy="53" r="39" stroke="currentColor" stroke-dasharray="2 7"/></svg></div>
      <strong class="pass-nickname">{{ displayNickname }}</strong>
      <span class="pass-club-name">{{ clubName || 'La tua squadra' }}</span>
      <span v-if="equippedFrame" class="pass-frame-label">Cornice della società</span>
      <div class="pass-stats"><span><b>{{ loading || !collection ? '—' : matchesFollowed }}</b> partite con partecipazione</span><span><b>{{ loading || !collection ? '—' : badges.length }}</b> badge</span></div>
    </article>

    <p v-if="loading" class="collection-message" role="status">Carichiamo la tua collezione…</p>
    <p v-if="error" class="collection-error" role="alert">{{ error }}</p>

    <template v-if="!loading">
      <div v-if="badges.length" class="collection-section">
        <div class="collection-section-head"><h3>I tuoi badge</h3><span>{{ badges.length }}</span></div>
        <ul class="badge-list">
          <li v-for="badge in badges" :key="`${badge.key}:${badge.event_id}`" class="earned-badge">
            <span class="badge-medal" aria-hidden="true"><svg viewBox="0 0 32 36" fill="none"><path d="m8 3 8 12L24 3M11 3l5 8 5-8" stroke="currentColor" stroke-width="3"/><circle cx="16" cy="23" r="10" stroke="currentColor" stroke-width="2"/><path d="m16 17 1.7 3.5 3.8.6-2.8 2.7.6 3.8-3.3-1.8-3.3 1.8.6-3.8-2.8-2.7 3.8-.6Z" fill="currentColor"/></svg></span>
            <span class="collection-copy"><strong>{{ badge.title || 'Badge della partita' }}</strong><span>{{ matchLabel(badge) }}</span><small v-if="formatDate(badge.event_date)">{{ formatDate(badge.event_date) }}</small></span>
          </li>
        </ul>
      </div>

      <div v-if="memories.length" class="collection-section">
        <div class="collection-section-head"><h3>Le tue partite, da conservare</h3><span>{{ memories.length }}</span></div>
        <ul class="memory-grid">
          <li v-for="memory in memories" :key="`${memory.key}:${memory.event_id}`" class="match-memory">
            <span class="memory-eyebrow">RICORDO DELLA PARTITA</span>
            <span class="memory-mark" aria-hidden="true"><svg viewBox="0 0 48 48" fill="none"><circle cx="24" cy="24" r="18" stroke="currentColor" stroke-width="1.5"/><path d="M6 24h36M24 6v36M12 11q22 13 0 26m24-26q-22 13 0 26" stroke="currentColor" stroke-width="1.5"/></svg></span>
            <strong>{{ memory.title || 'Card della partita' }}</strong><span class="memory-match">{{ matchLabel(memory) }}</span><time v-if="formatDate(memory.event_date)">{{ formatDate(memory.event_date) }}</time>
          </li>
        </ul>
      </div>

      <div v-if="frames.length" class="collection-section">
        <div class="collection-section-head"><h3>Il tuo stile</h3><button v-if="equippedFrame" class="reset-frame" type="button" :disabled="busy" @click="equipFrame('')">Togli cornice</button></div>
        <ul class="frame-list">
          <li v-for="frame in frames" :key="`${frame.key}:${frame.event_id}`" class="owned-frame">
            <span class="frame-preview" aria-hidden="true"><span>{{ initials }}</span></span>
            <span class="collection-copy"><strong>{{ frame.title || 'Cornice della società' }}</strong><span>{{ equippedFrame === frame.key ? 'Sulla tua tessera' : 'Nella tua collezione' }}</span></span>
            <button type="button" class="equip-frame" :aria-pressed="equippedFrame === frame.key" :disabled="busy || equippedFrame === frame.key" @click="equipFrame(frame.key)">{{ equippedFrame === frame.key ? 'Indossata' : 'Indossa' }}</button>
          </li>
        </ul>
      </div>

      <p v-if="!hasCollection && !error" class="collection-empty">{{ registered ? 'La tua collezione comincia qui. Conquista i badge e scegli un ricordo della partita.' : 'La tua collezione comincia qui. Accedi al profilo per conservare i tuoi badge e ricordi.' }}</p>
      <button class="collection-shop" type="button" :disabled="busy" @click="emit('open-shop')"><span><strong>Dai il tuo stile alla tessera</strong><small>Scopri cornici e ricordi nello shop</small></span><span aria-hidden="true">↗</span></button>
    </template>
  </section>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  nickname: { type: String, default: '' },
  clubName: { type: String, default: '' },
  clubLogoUrl: { type: String, default: '' },
  registered: { type: Boolean, default: false },
  collection: { type: Object, default: null },
  loading: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  error: { type: String, default: '' },
});
const emit = defineEmits(['equip', 'open-shop']);
const displayNickname = computed(() => String(props.nickname || '').trim() || 'Tifoso');
const initials = computed(() => displayNickname.value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => Array.from(part)[0]).join('').toUpperCase());
const items = computed(() => Array.isArray(props.collection?.items) ? props.collection.items.filter((item) => item && typeof item === 'object') : []);
const badges = computed(() => Array.isArray(props.collection?.badges) ? props.collection.badges.filter((badge) => badge && typeof badge === 'object') : []);
const frames = computed(() => items.value.filter((item) => item.kind === 'frame' && item.key === 'club_frame'));
const memories = computed(() => items.value.filter((item) => item.kind === 'match_card'));
const equippedFrame = computed(() => props.collection?.equipped_frame === 'club_frame' && frames.value.some((frame) => frame.key === 'club_frame') ? 'club_frame' : '');
const matchesFollowed = computed(() => Number.isInteger(props.collection?.matches_followed) && props.collection.matches_followed >= 0 ? props.collection.matches_followed : 0);
const hasCollection = computed(() => badges.value.length > 0 || items.value.length > 0);

function equipFrame(key) {
  if (props.loading || props.busy || key === equippedFrame.value) return;
  if (key !== '' && !frames.value.some((frame) => frame.key === key)) return;
  if (key !== '' && key !== 'club_frame') return;
  emit('equip', key);
}
function matchLabel(entry) {
  const club = String(entry?.club_name || props.clubName || '').trim();
  const opponent = String(entry?.opponent || '').trim();
  return [club, opponent ? `vs ${opponent}` : ''].filter(Boolean).join(' ') || 'La tua partita';
}
function formatDate(value) {
  if (!value) return '';
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return '';
  return date.toLocaleDateString('it-IT', { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' });
}
</script>

<style scoped>
.fan-collection { display: flex; flex-direction: column; gap: 19px; min-width: 0; color: var(--ink, #23161a); font-family: inherit; }
.fan-collection * { box-sizing: border-box; }
.fan-pass { position: relative; isolation: isolate; display: flex; flex-direction: column; align-items: center; min-width: 0; overflow: hidden; padding: 21px 16px 0; border: 1px solid var(--red-line, #ffd2ce); border-radius: 22px; background: radial-gradient(ellipse at 50% 22%, var(--red-soft, #ffe8e6), transparent 58%), linear-gradient(145deg, var(--card, #fff), var(--bg, #fff7f2)); box-shadow: 0 4px 0 var(--line, #f1e4e0); text-align: center; }
.fan-pass::before { content: ''; position: absolute; z-index: -1; inset: 8px; border: 1px solid var(--red-line, #ffd2ce); border-radius: 15px; opacity: .55; pointer-events: none; }
.fan-pass::after { content: ''; position: absolute; z-index: -1; top: -40px; bottom: -40px; left: 14%; width: 27%; transform: rotate(32deg); background: var(--card, #fff); opacity: .45; pointer-events: none; }
.fan-pass.has-club-frame { border: 3px solid #cfaa58; padding-top: 19px; background: radial-gradient(ellipse at 50% 22%, #ffe5a857, transparent 58%), linear-gradient(145deg, var(--card, #fff), var(--bg, #fff7f2)); box-shadow: 0 4px 0 #b58b3d, inset 0 0 0 2px #fff0bd; }
.has-club-frame::before { border: 1px solid #d3ad5a; opacity: .85; }
.pass-eyebrow { align-self: flex-start; max-width: calc(100% - 44px); color: var(--muted, #8c7a80); font-size: 8px; font-weight: 800; letter-spacing: .15em; text-align: left; }
.pass-club-logo { position: absolute; top: 16px; right: 16px; width: 34px; height: 34px; object-fit: contain; }
.pass-portrait { position: relative; display: grid; place-items: center; width: 100px; height: 100px; margin: 15px 0 8px; color: var(--red-text, #bd1523); }
.pass-portrait > span { display: grid; place-items: center; width: 67px; height: 67px; border: 1px solid var(--red-line, #ffd2ce); border-radius: 50%; background: var(--card, #fff); color: var(--red-text, #bd1523); font-family: 'Fraunces', serif; font-size: 28px; font-weight: 750; }
.pass-portrait svg { position: absolute; inset: 0; width: 100%; height: 100%; opacity: .45; pointer-events: none; }
.pass-nickname { max-width: 100%; color: var(--ink, #23161a); font-family: 'Fraunces', serif; font-size: 25px; font-weight: 750; line-height: 1.1; overflow-wrap: anywhere; }
.pass-club-name { max-width: 100%; margin-top: 6px; color: var(--muted, #8c7a80); font-size: 10px; font-weight: 700; overflow-wrap: anywhere; }
.pass-frame-label { margin-top: 9px; border-radius: 99px; padding: 4px 8px; background: #f7e1aa; color: #66430e; font-size: 8px; font-weight: 800; }
.pass-stats { display: flex; justify-content: center; flex-wrap: wrap; gap: 6px 16px; width: calc(100% + 32px); margin-top: 18px; padding: 12px 10px; background: var(--red-soft, #ffe8e6); color: var(--muted, #8c7a80); font-size: 9px; line-height: 1.5; }
.pass-stats b { margin-right: 3px; color: var(--ink, #23161a); font-size: 13px; font-weight: 900; }
.collection-message, .collection-empty, .collection-error { margin: 0; border-radius: 14px; padding: 13px; background: var(--bg, #fff7f2); color: var(--muted, #8c7a80); font-size: 11px; line-height: 1.6; }
.collection-error { border: 1px solid var(--red-line, #ffd2ce); color: var(--red-text, #bd1523); }
.collection-section { display: flex; flex-direction: column; gap: 10px; min-width: 0; }
.collection-section-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.collection-section-head h3 { margin: 0; color: var(--ink, #23161a); font-size: 12px; font-weight: 850; line-height: 1.4; }
.collection-section-head > span { display: grid; place-items: center; min-width: 23px; height: 23px; padding: 0 5px; border-radius: 50%; background: var(--red-soft, #ffe8e6); color: var(--red-text, #bd1523); font-size: 10px; font-weight: 800; }
.badge-list, .frame-list, .memory-grid { display: grid; gap: 9px; margin: 0; padding: 0; list-style: none; }
.earned-badge, .owned-frame { display: flex; align-items: center; gap: 11px; min-width: 0; border: 1px solid var(--line, #f1e4e0); border-radius: 16px; padding: 12px; background: var(--card, #fff); }
.badge-medal { display: grid; place-items: center; width: 43px; height: 48px; flex: none; border-radius: 13px; background: #fff0c7; color: #a86f10; }
.badge-medal svg { width: 28px; height: 34px; }
.collection-copy { display: flex; flex: 1; flex-direction: column; gap: 4px; min-width: 0; }
.collection-copy strong { color: var(--ink, #23161a); font-size: 11px; font-weight: 850; line-height: 1.35; overflow-wrap: anywhere; }
.collection-copy > span { color: var(--muted, #8c7a80); font-size: 10px; line-height: 1.45; overflow-wrap: anywhere; }
.collection-copy small { color: var(--muted, #8c7a80); font-size: 9px; }
.memory-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: stretch; }
.match-memory { display: flex; flex-direction: column; align-items: center; gap: 6px; min-width: 0; border: 1px solid var(--red-line, #ffd2ce); border-radius: 17px 17px 26px 26px; padding: 15px 10px; background: linear-gradient(145deg, var(--red-soft, #ffe8e6), var(--card, #fff)); text-align: center; }
.memory-eyebrow { color: var(--muted, #8c7a80); font-size: 6px; font-weight: 850; letter-spacing: .08em; }
.memory-mark { display: grid; place-items: center; width: 48px; height: 48px; margin: 5px 0; color: var(--red-text, #bd1523); }
.memory-mark svg { width: 100%; height: 100%; }
.match-memory > strong { color: var(--ink, #23161a); font-family: 'Fraunces', serif; font-size: 14px; font-weight: 750; line-height: 1.2; overflow-wrap: anywhere; }
.memory-match { color: var(--muted, #8c7a80); font-size: 9px; line-height: 1.5; overflow-wrap: anywhere; }
.match-memory time { margin-top: auto; padding-top: 7px; color: var(--red-text, #bd1523); font-size: 8px; font-weight: 800; }
.frame-preview { display: grid; place-items: center; width: 42px; height: 51px; flex: none; border: 2px solid #c49b46; border-radius: 10px 10px 15px 15px; background: linear-gradient(145deg, #fff2c9, #e1bd70); box-shadow: inset 0 0 0 2px #ffefbe; color: #70501d; font-family: 'Fraunces', serif; font-size: 15px; font-weight: 750; }
.equip-frame, .reset-frame { flex: none; border: 0; border-radius: 10px; padding: 9px 10px; background: var(--red-soft, #ffe8e6); color: var(--red-text, #bd1523); font-family: inherit; font-size: 9px; font-weight: 800; cursor: pointer; }
.reset-frame { padding: 5px 0; background: none; font-size: 9px; text-decoration: underline; text-underline-offset: 3px; }
.equip-frame[aria-pressed='true'] { background: var(--bg, #fff7f2); color: var(--muted, #8c7a80); }
.collection-shop { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; border: 1px solid var(--red-line, #ffd2ce); border-radius: 15px; padding: 14px; background: var(--red-soft, #ffe8e6); color: var(--red-text, #bd1523); font-family: inherit; text-align: left; cursor: pointer; }
.collection-shop strong, .collection-shop small { display: block; }
.collection-shop strong { font-size: 11px; font-weight: 850; }
.collection-shop small { margin-top: 4px; font-size: 9px; line-height: 1.5; }
.collection-shop > span:last-child { font-size: 22px; }
.fan-collection button:disabled { cursor: default; opacity: .6; }
.fan-collection button:focus-visible { outline: 2px solid var(--red-text, #bd1523); outline-offset: 3px; }
@media (max-width: 340px) { .owned-frame { flex-wrap: wrap; } .equip-frame { margin-left: auto; } .pass-stats { gap: 6px 10px; } }
</style>

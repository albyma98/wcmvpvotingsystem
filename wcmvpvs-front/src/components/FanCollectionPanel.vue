<template>
  <section class="fan-collection" aria-label="La tua collezione tifoso" :aria-busy="loading || busy">
    <article class="fan-pass" :aria-label="`Tessera tifoso di ${displayNickname}`">
      <FanAvatar class="pass-portrait" :name="displayNickname" :framed="equippedFrame === 'club_frame'" :size="60" />
      <div class="pass-identity">
        <span class="pass-eyebrow">TESSERA TIFOSO</span>
        <strong class="pass-nickname">{{ displayNickname }}</strong>
        <span class="pass-club-name">{{ clubName || 'La tua squadra' }}</span>
      </div>
      <img v-if="clubLogoUrl" class="pass-club-logo" :src="clubLogoUrl" :alt="clubName || 'Logo della società'" loading="lazy" />
      <div class="pass-stats"><span><b>{{ loading || !collection ? '—' : matchesFollowed }}</b> partite seguite</span><span><b>{{ loading || !collection ? '—' : badges.length }}</b> badge</span><slot name="identity-action" /></div>
    </article>

    <slot name="profile-actions" />

    <p v-if="loading" class="collection-message" role="status">Carichiamo la tua collezione…</p>
    <p v-if="error" class="collection-error" role="alert">{{ error }}</p>

    <template v-if="!loading">
      <details v-if="badges.length" class="collection-section">
        <summary class="collection-section-head"><span>I tuoi badge</span><span class="collection-count">{{ badges.length }}</span><span class="collection-chevron" aria-hidden="true">⌄</span></summary>
        <ul class="badge-list">
          <li v-for="badge in badges" :key="`${badge.key}:${badge.event_id}`" class="earned-badge">
            <span class="badge-medal" aria-hidden="true"><svg viewBox="0 0 32 36" fill="none"><path d="m8 3 8 12L24 3M11 3l5 8 5-8" stroke="currentColor" stroke-width="3"/><circle cx="16" cy="23" r="10" stroke="currentColor" stroke-width="2"/><path d="m16 17 1.7 3.5 3.8.6-2.8 2.7.6 3.8-3.3-1.8-3.3 1.8.6-3.8-2.8-2.7 3.8-.6Z" fill="currentColor"/></svg></span>
            <span class="collection-copy"><strong>{{ badge.title || 'Badge della partita' }}</strong><span>{{ matchLabel(badge) }}</span><small v-if="formatDate(badge.event_date)">{{ formatDate(badge.event_date) }}</small></span>
          </li>
        </ul>
      </details>

      <details v-if="memories.length" class="collection-section">
        <summary class="collection-section-head"><span>Ricordi delle partite</span><span class="collection-count">{{ memories.length }}</span><span class="collection-chevron" aria-hidden="true">⌄</span></summary>
        <ul class="memory-grid">
          <li v-for="memory in memories" :key="`${memory.key}:${memory.event_id}`" class="match-memory">
            <button type="button" class="memory-open" :disabled="busy" :aria-label="`Apri il ricordo ${matchLabel(memory)}`" @click="inspectMemory(memory)"><FanMatchCard :memory="memory" :club-name="clubName" :club-logo-url="clubLogoUrl" :sport="sport" compact /></button>
            <span class="memory-hint">Tocca per ingrandire <span aria-hidden="true">↗</span></span>
          </li>
        </ul>
      </details>

      <details v-if="frames.length" class="collection-section">
        <summary class="collection-section-head"><span>Le tue cornici</span><span class="collection-count">{{ frames.length }}</span><span class="collection-chevron" aria-hidden="true">⌄</span></summary>
        <ul class="frame-list">
          <li v-for="frame in frames" :key="`${frame.key}:${frame.event_id}`" class="owned-frame">
            <FanAvatar :name="displayNickname" framed :size="38" />
            <span class="collection-copy"><strong>{{ frame.title || 'Cornice della società' }}</strong><span>{{ equippedFrame === frame.key ? 'Sulla tua tessera' : 'Nella tua collezione' }}</span></span>
            <button type="button" class="equip-frame" :aria-pressed="equippedFrame === frame.key" :disabled="busy" @click="equipFrame(equippedFrame === frame.key ? '' : frame.key)">{{ equippedFrame === frame.key ? 'Togli' : 'Indossa' }}</button>
          </li>
        </ul>
      </details>

      <p v-if="!hasCollection && !error" class="collection-empty">{{ registered ? 'La tua collezione comincia qui. Conquista i badge e scegli un ricordo della partita.' : 'La tua collezione comincia qui. Accedi al profilo per conservare i tuoi badge e ricordi.' }}</p>
      <button class="collection-shop" type="button" :disabled="busy" @click="emit('open-shop')"><span><strong>Esplora lo shop</strong><small>Cornici e ricordi con le tue monete</small></span><span aria-hidden="true">↗</span></button>
    </template>
  </section>
</template>

<script setup>
import { computed } from 'vue';
import FanAvatar from './FanAvatar.vue';
import FanMatchCard from './FanMatchCard.vue';

const props = defineProps({
  nickname: { type: String, default: '' },
  clubName: { type: String, default: '' },
  clubLogoUrl: { type: String, default: '' },
  sport: { type: String, default: '' },
  registered: { type: Boolean, default: false },
  collection: { type: Object, default: null },
  loading: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  error: { type: String, default: '' },
});
const emit = defineEmits(['equip', 'open-shop', 'inspect']);
const displayNickname = computed(() => String(props.nickname || '').trim() || 'Tifoso');
const initials = computed(() => displayNickname.value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => Array.from(part)[0]).join('').toUpperCase());
const items = computed(() => Array.isArray(props.collection?.items) ? props.collection.items.filter((item) => item && typeof item === 'object') : []);
const badges = computed(() => Array.isArray(props.collection?.badges) ? props.collection.badges.filter((badge) => badge && typeof badge === 'object') : []);
const frames = computed(() => items.value.filter((item) => item.kind === 'frame' && item.key === 'club_frame'));
const memories = computed(() => items.value.filter((item) => item.kind === 'match_card'));
const equippedFrame = computed(() => props.collection?.equipped_frame === 'club_frame' && frames.value.some((frame) => frame.key === 'club_frame') ? 'club_frame' : '');
const matchesFollowed = computed(() => Number.isInteger(props.collection?.matches_followed) && props.collection.matches_followed >= 0 ? props.collection.matches_followed : 0);
const hasCollection = computed(() => badges.value.length > 0 || items.value.length > 0);

function inspectMemory(memory) {
  if (props.loading || props.busy || !memories.value.includes(memory)) return;
  emit('inspect', memory);
}

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
.fan-collection { display: flex; flex: none; flex-direction: column; gap: 10px; min-width: 0; color: var(--ink, #23161a); font-family: inherit; }
.fan-collection * { box-sizing: border-box; }
.fan-pass { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: 12px; min-width: 0; overflow: hidden; padding: 14px 14px 0; border: 1px solid var(--red-line, #ffd2ce); border-radius: 17px; background: linear-gradient(120deg, var(--card, #fff), var(--red-soft, #ffe8e6)); }
.pass-identity { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; min-width: 0; padding: 3px 0; }
.pass-eyebrow { color: var(--muted, #8c7a80); font-size: 8px; font-weight: 800; letter-spacing: .12em; }
.pass-club-logo { width: 27px; height: 27px; object-fit: contain; }
.pass-nickname { max-width: 100%; color: var(--ink, #23161a); font-family: 'Fraunces', serif; font-size: 21px; font-weight: 750; line-height: 1.1; overflow-wrap: anywhere; }
.pass-club-name { max-width: 100%; color: var(--muted, #8c7a80); font-size: 10px; font-weight: 600; overflow-wrap: anywhere; }
.pass-stats { grid-column: 1 / -1; display: flex; align-items: center; flex-wrap: wrap; gap: 5px 12px; margin: 0 -14px; padding: 5px 14px; min-height: 36px; border-top: 1px solid var(--red-line, #ffd2ce); color: var(--muted, #8c7a80); font-size: 10px; line-height: 1.5; }
.pass-stats b { margin-right: 2px; color: var(--ink, #23161a); font-size: 11px; font-weight: 800; }
.collection-message, .collection-empty, .collection-error { margin: 0; border-radius: 14px; padding: 13px; background: var(--bg, #fff7f2); color: var(--muted, #8c7a80); font-size: 11px; line-height: 1.6; }
.collection-error { border: 1px solid var(--red-line, #ffd2ce); color: var(--red-text, #bd1523); }
.collection-section { min-width: 0; border: 1px solid var(--line, #f1e4e0); border-radius: 13px; background: var(--card, #fff); }
.collection-section-head { display: flex; align-items: center; gap: 10px; min-height: 48px; padding: 10px 12px; color: var(--ink, #23161a); font-size: 12px; font-weight: 750; line-height: 1.4; cursor: pointer; list-style: none; }
.collection-section-head::-webkit-details-marker { display: none; }
.collection-section-head > span:first-child { flex: 1; min-width: 0; }
.collection-count { display: grid; place-items: center; min-width: 22px; height: 22px; padding: 0 5px; border-radius: 99px; background: var(--red-soft, #ffe8e6); color: var(--red-text, #bd1523); font-size: 10px; font-weight: 800; }
.collection-chevron { color: var(--muted, #8c7a80); font-size: 17px; line-height: 1; }
.collection-section[open] .collection-chevron { transform: rotate(180deg); }
.collection-section[open] > ul { padding: 0 10px 10px; }
.badge-list, .frame-list, .memory-grid { display: grid; gap: 9px; margin: 0; padding: 0; list-style: none; }
.earned-badge, .owned-frame { display: flex; align-items: center; gap: 10px; min-width: 0; border-radius: 10px; padding: 9px; background: var(--bg, #fff7f2); }
.badge-medal { display: grid; place-items: center; width: 32px; height: 36px; flex: none; border-radius: 9px; background: #fff0c7; color: #a86f10; }
.badge-medal svg { width: 22px; height: 27px; }
.collection-copy { display: flex; flex: 1; flex-direction: column; gap: 4px; min-width: 0; }
.collection-copy strong { color: var(--ink, #23161a); font-size: 11px; font-weight: 850; line-height: 1.35; overflow-wrap: anywhere; }
.collection-copy > span { color: var(--muted, #8c7a80); font-size: 10px; line-height: 1.45; overflow-wrap: anywhere; }
.collection-copy small { color: var(--muted, #8c7a80); font-size: 9px; }
.memory-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: stretch; }
.match-memory { min-width: 0; }
.memory-open { display: block; width: 100%; border: 0; padding: 0; border-radius: 10px; background: transparent; color: inherit; cursor: pointer; transition: transform .2s ease; }
.memory-open:active { transform: scale(.98); }
.memory-hint { display: flex; justify-content: space-between; margin-top: 9px; padding: 0 3px; color: var(--muted); font-size: 9px; }
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
.fan-collection button:focus-visible, .collection-section-head:focus-visible { outline: 2px solid var(--red-text, #bd1523); outline-offset: 3px; }
@media (max-width: 340px) { .owned-frame { flex-wrap: wrap; } .equip-frame { margin-left: auto; } .pass-stats { gap: 6px 10px; } }
@media (prefers-reduced-motion: reduce) { .memory-open { transition: none; } }
</style>

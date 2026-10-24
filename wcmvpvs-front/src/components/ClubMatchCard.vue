<script setup>
import { computed } from 'vue'
import { formatClubDate } from '../clubHub'

const props = defineProps({ match: { type: Object, required: true }, teamName: { type: String, default: 'Squadra' }, featured: Boolean })
const homeName = computed(() => props.match.home ? props.teamName : props.match.opponent)
const awayName = computed(() => props.match.home ? props.match.opponent : props.teamName)
const statusLabel = computed(() => props.match.status === 'live' ? 'In diretta' : props.match.status === 'completed' ? 'Risultato finale' : 'In programma')
</script>

<template>
  <article class="club-match-card" :class="{ 'is-featured': featured }">
    <div class="club-match-meta">
      <span v-if="featured" class="club-match-feature">{{ match.status === 'completed' ? 'Ultimo risultato' : match.status === 'live' ? 'Ora in campo' : 'Prossimo appuntamento' }}</span>
      <span v-else class="club-match-competition">{{ match.competition || teamName }}</span>
      <span class="club-match-status" :class="match.status"><i v-if="match.status === 'live'" aria-hidden="true"></i>{{ statusLabel }}</span>
    </div>
    <time class="club-match-date" :datetime="match.scheduledAt">{{ formatClubDate(match.scheduledAt) }}</time>
    <div class="club-match-scoreboard">
      <strong>{{ homeName }}</strong>
      <div v-if="match.status === 'completed'" class="club-match-score" :aria-label="`Risultato: ${match.home ? match.scoreFor : match.scoreAgainst} a ${match.home ? match.scoreAgainst : match.scoreFor}`">
        <b>{{ match.home ? match.scoreFor : match.scoreAgainst }}</b><span>:</span><b>{{ match.home ? match.scoreAgainst : match.scoreFor }}</b>
      </div>
      <span v-else class="club-match-vs">VS</span>
      <strong>{{ awayName }}</strong>
    </div>
    <div v-if="match.location || match.roundLabel || (featured && match.competition)" class="club-match-details">
      <span v-if="match.location">{{ match.location }}</span><span v-if="match.roundLabel">{{ match.roundLabel }}</span><span v-if="featured && match.competition">{{ match.competition }}</span>
    </div>
  </article>
</template>

<style scoped>
.club-match-card{min-width:0;padding:22px;border:1px solid #dbe2ec;border-radius:22px;background:#fff;box-shadow:inset 0 2px 0 #fff,0 4px 0 #dce2eb,0 10px 22px rgba(15,23,42,.06);color:#0f172a}
.club-match-card.is-featured{border-top:4px solid var(--club-accent,#f59e0b)}
.club-match-meta{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px}
.club-match-competition{color:#475569;font-size:12px;font-weight:800;overflow-wrap:anywhere}.club-match-feature{color:#334155;font-size:11px;font-weight:900;text-transform:uppercase;letter-spacing:.08em}
.club-match-status{display:inline-flex;align-items:center;justify-content:center;gap:6px;padding:6px 9px;border-radius:8px;background:#eef2f6;color:#475569;font-size:10px;font-weight:850;line-height:1.2;white-space:nowrap}.club-match-status.live{background:#fee2e2;color:#991b1b}.club-match-status.completed{background:#dcfce7;color:#166534}.club-match-status i{width:6px;height:6px;border-radius:50%;background:currentColor}
.club-match-date{display:block;margin-top:12px;color:#64748b;font-size:12px;line-height:1.5}
.club-match-scoreboard{display:grid;grid-template-columns:minmax(0,1fr) auto minmax(0,1fr);align-items:center;gap:16px;margin:20px 0}
.club-match-scoreboard>strong{font-family:var(--club-font-title,inherit);font-size:clamp(15px,1.8vw,21px);line-height:1.3;overflow-wrap:anywhere}.club-match-scoreboard>strong:last-child{text-align:right}
.club-match-score{display:flex;align-items:center;gap:6px}.club-match-score b{display:grid;place-items:center;min-width:38px;min-height:44px;padding:5px 8px;border:1px solid color-mix(in srgb,var(--club-primary,#172554) 70%,#fff);border-radius:10px;background:var(--club-primary,#172554);color:var(--club-on-primary,#fff);font-family:var(--club-font-title,inherit);font-size:26px;line-height:1;box-shadow:inset 0 2px 0 rgba(255,255,255,.3),0 3px 0 color-mix(in srgb,var(--club-primary,#172554) 65%,#030712)}.club-match-score>span{color:#94a3b8;font-weight:900}
.club-match-vs{display:grid;place-items:center;min-width:42px;min-height:42px;border:1px solid #dbe2ec;border-radius:12px;background:#f1f5f9;color:#64748b;font-size:12px;font-weight:900}
.club-match-details{display:flex;flex-wrap:wrap;gap:8px;padding-top:12px;border-top:1px solid #edf0f5;color:#64748b;font-size:12px;line-height:1.5}.club-match-details>span+span::before{content:'·';margin-right:8px;color:#94a3b8}
@media(max-width:560px){.club-match-card{padding:16px;border-radius:18px}.club-match-scoreboard{gap:9px;margin:18px 0}.club-match-scoreboard>strong{font-size:14px}.club-match-score{gap:3px}.club-match-score b{min-width:28px;min-height:36px;padding:5px;font-size:21px}.club-match-meta{align-items:flex-start}.club-match-status{font-size:9px}.club-match-date,.club-match-details{font-size:11px}}
</style>

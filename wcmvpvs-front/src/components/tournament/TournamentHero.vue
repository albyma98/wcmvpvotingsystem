<script setup>
const props = defineProps({
  tournament: { type: Object, required: true }
})

// "Sunset Beach Cup" → prima parola bianca, resto in gold (fallback senza logo)
const firstWord = () => props.tournament.name.split(' ')[0]
const restWords = () => props.tournament.name.split(' ').slice(1).join(' ')
</script>

<template>
  <header
    class="hero"
    :class="{ 'has-img': !!tournament.heroImage }"
    :style="tournament.heroImage ? { '--hero-img': `url(${tournament.heroImage})` } : {}"
  >
    <div class="hero-brand">
      <img v-if="tournament.logo" :src="tournament.logo" class="hero-logo" :alt="tournament.name" />
      <h1 v-else class="hero-title">
        {{ firstWord() }}<span class="accent">{{ restWords() }}</span>
      </h1>
      <div class="hero-format">{{ tournament.format }}</div>
    </div>

    <div class="hero-meta">
      <span>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M16 2v4M8 2v4M3 10h18"/></svg>
        {{ tournament.dateLabel }}
      </span>
      <span class="sep" aria-hidden="true"></span>
      <span>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/></svg>
        {{ tournament.location }}
      </span>
    </div>

    <div class="hero-pills">
      <span class="pill pill--live">
        <span class="live-dot" aria-hidden="true"></span>{{ tournament.statusLabel }}
      </span>
      <span class="pill pill--phase">{{ tournament.phaseLabel }}</span>
    </div>
  </header>
</template>

<style scoped>
.hero {
  flex: none;
  position: relative;
  padding: calc(6px + env(safe-area-inset-top)) 16px 10px;
  background:
    linear-gradient(180deg, rgba(10,10,14,.25) 0%, rgba(10,10,14,.35) 55%, var(--tm-bg) 100%),
    radial-gradient(120% 110% at 50% 0%, #2E4B6E 0%, #7A4A3A 42%, #D2803C 68%, #1A1A22 100%);
  background-size: cover;
  background-position: center;
}
.hero.has-img {
  background-image:
    linear-gradient(180deg, rgba(10,10,14,.30) 0%, rgba(10,10,14,.45) 60%, var(--tm-bg) 100%),
    var(--hero-img);
}
.hero-brand { text-align: center; padding-top: 2px; }
.hero-logo { max-height: clamp(44px, 8dvh, 72px); max-width: 70%; filter: drop-shadow(0 4px 14px rgba(0,0,0,.5)); }
.hero-title {
  font-size: clamp(20px, 3.6dvh, 30px); font-weight: 900; font-style: italic; letter-spacing: .5px;
  line-height: .95; text-transform: uppercase; text-shadow: 0 3px 12px rgba(0,0,0,.6);
}
.hero-title .accent { display: block; color: var(--tm-gold); font-size: .78em; letter-spacing: 2px; }
.hero-format { margin-top: 4px; font-size: clamp(9px, 1.3dvh, 11px); font-weight: 800; letter-spacing: 3px; opacity: .92; }
.hero-meta {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  margin-top: clamp(5px, 1dvh, 10px); font-size: clamp(10.5px, 1.6dvh, 13px); font-weight: 700; letter-spacing: .4px;
}
.hero-meta .sep { width: 1px; height: 12px; background: rgba(255,255,255,.35); }
.hero-meta svg { width: 1.05em; height: 1.05em; margin-right: 5px; vertical-align: -2px; }
.hero-pills { display: flex; gap: 8px; justify-content: center; margin-top: clamp(6px, 1.1dvh, 12px); }
.pill {
  display: inline-flex; align-items: center; gap: 6px; padding: clamp(4px, .8dvh, 7px) 14px;
  border-radius: 999px; font-size: clamp(9.5px, 1.4dvh, 11.5px); font-weight: 800; letter-spacing: .8px; text-transform: uppercase;
}
.pill--live { border: 1px solid rgba(255,255,255,.55); background: rgba(10,10,14,.35); backdrop-filter: blur(6px); }
.pill--phase { background: rgba(255,255,255,.10); color: var(--tm-text-dim); }
.live-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--tm-live); animation: tm-pulse 1.6s ease-in-out infinite; flex: none; }
@keyframes tm-pulse { 0%,100% { opacity: 1; transform: scale(1); } 50% { opacity: .45; transform: scale(.85); } }
@media (prefers-reduced-motion: reduce) { .live-dot { animation: none; } }
</style>

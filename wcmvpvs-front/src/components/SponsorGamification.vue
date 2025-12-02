<template>
  <section class="sponsor-shell">
    <header class="sponsor-header">
      <div>
        <p class="sponsor-eyebrow">Sponsor Gamification</p>
        <h3>Mini-giochi dedicati allo sponsor</h3>
        <p>Scopri l’offerta, completa il mini-gioco e ottieni coupon immediati.</p>
      </div>
      <button class="btn primary" type="button" @click="$emit('play', 'sponsorReveal')">
        Scopri l'offerta
      </button>
    </header>

    <div class="sponsor-grid">
      <div class="sponsor-card">
        <p class="sponsor-card__eyebrow">Tap reveal → coupon</p>
        <h4>Sponsor Challenge</h4>
        <p>Premi per scoprire il coupon del matchday e guadagna punti extra fedeltà.</p>
        <button class="btn outline" type="button" @click="$emit('play', 'coupon')">Attiva coupon</button>
      </div>

      <div class="sponsor-card">
        <p class="sponsor-card__eyebrow">Mini-game</p>
        <h4>Volley Smash</h4>
        <p>Evita i blocchi dello sponsor e colpisci il bersaglio: combo perfetta per la classifica live.</p>
        <button class="btn outline" type="button" @click="$emit('play', 'minigame')">Gioca ora</button>
      </div>

      <div class="sponsor-card">
        <p class="sponsor-card__eyebrow">Reward</p>
        <h4>Badge partner</h4>
        <p>Completa entrambe le sfide sponsor e ricevi un badge esclusivo e punti bonus.</p>
        <button class="btn outline" type="button" @click="$emit('play', 'partner-badge')">Sblocca</button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.sponsor-shell {
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 20px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.sponsor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.sponsor-eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: #94a3b8;
  margin: 0;
}

.sponsor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem;
}

.sponsor-card {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 14px;
  padding: 0.85rem;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.sponsor-card__eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.14em;
  color: #cbd5e1;
  margin: 0;
}

.btn {
  border: 1px solid transparent;
  border-radius: 12px;
  padding: 0.55rem 1rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn.primary {
  background: linear-gradient(90deg, #f59e0b, #f97316);
  color: #0b172a;
  font-weight: 700;
}

.btn.outline {
  background: transparent;
  border-color: rgba(148, 163, 184, 0.5);
  color: #e2e8f0;
}
</style>

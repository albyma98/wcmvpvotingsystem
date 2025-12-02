<template>
  <section class="gamification-shell">
    <header class="gamification-hero">
      <div>
        <p class="gamification-hero__eyebrow">Missione partita</p>
        <h2 class="gamification-hero__title">Benvenuto! Inizia la Missione della Partita</h2>
        <p class="gamification-hero__subtitle">
          Segui le tappe, accumula punti e sblocca badge esclusivi per la tua fedeltà.
        </p>
        <div class="gamification-hero__actions">
          <button type="button" class="btn primary" @click="$emit('action', 'preMatch')">
            Inizia Missione
          </button>
          <button type="button" class="btn ghost" @click="$emit('open-rewards')">
            Ricompense e badge
          </button>
        </div>
      </div>
      <div class="level-card">
        <p class="level-card__label">Livello attuale</p>
        <p class="level-card__title">{{ level.label }} {{ level.badge }}</p>
        <p class="level-card__points">{{ points }} pt</p>
        <div class="level-progress">
          <div class="level-progress__bar">
            <span class="level-progress__fill" :style="{ width: `${progressToNext}%` }"></span>
          </div>
          <p class="level-progress__hint">{{ progressLabel }}</p>
        </div>
      </div>
    </header>

    <div class="gamification-grid">
      <div class="gamification-card">
        <header class="gamification-card__header">
          <p class="gamification-card__eyebrow">Missioni sequenziali</p>
          <h3>Roadmap della partita</h3>
          <p>Completa le tappe nell'ordine consigliato. Ogni missione mostra il feedback immediato e il pulsante "continua".</p>
        </header>
        <ol class="mission-list">
          <li
            v-for="mission in missions"
            :key="mission.id"
            class="mission-step"
            :class="{
              'mission-step--active': mission.status === 'active',
              'mission-step--done': mission.status === 'done',
            }"
          >
            <div class="mission-step__header">
              <div class="mission-step__title">
                <span class="mission-step__badge">{{ mission.order }}</span>
                <div>
                  <p class="mission-step__eyebrow">{{ mission.label }}</p>
                  <h4>{{ mission.title }}</h4>
                </div>
              </div>
              <p class="mission-step__points">+{{ mission.points }} pt</p>
            </div>
            <p class="mission-step__description">{{ mission.description }}</p>
            <div class="mission-step__footer">
              <span class="mission-step__status">{{ mission.statusLabel }}</span>
              <button
                v-if="mission.status !== 'done'"
                type="button"
                class="btn outline"
                @click="$emit('action', mission.action)"
              >
                Continua
              </button>
              <span v-else class="mission-step__feedback">{{ mission.feedback }}</span>
            </div>
          </li>
        </ol>
      </div>

      <div class="gamification-card">
        <header class="gamification-card__header">
          <p class="gamification-card__eyebrow">Classifiche</p>
          <h3>Live leaderboard</h3>
          <p>Consulta le classifiche giornaliere, mensili e stagionali.</p>
        </header>
        <div class="leaderboard-tabs">
          <button
            v-for="tab in ['daily', 'monthly', 'season']"
            :key="tab"
            class="leaderboard-tab"
            :class="{ 'leaderboard-tab--active': leaderboardPeriod === tab }"
            type="button"
            @click="$emit('change-period', tab)">
            {{ periodLabel(tab) }}
          </button>
        </div>
        <ul class="leaderboard-list">
          <li v-for="player in leaderboard" :key="player.name" class="leaderboard-row" :class="{ 'leaderboard-row--me': player.isMe }">
            <div class="leaderboard-row__meta">
              <span class="leaderboard-row__rank">#{{ player.rank }}</span>
              <div>
                <p class="leaderboard-row__name">{{ player.name }}</p>
                <p class="leaderboard-row__hint">{{ player.hint }}</p>
              </div>
            </div>
            <div class="leaderboard-row__score">
              <p class="leaderboard-row__points">{{ player.points }} pt</p>
              <p class="leaderboard-row__delta">{{ player.delta }}</p>
            </div>
          </li>
        </ul>
      </div>

      <div class="gamification-card">
        <header class="gamification-card__header">
          <p class="gamification-card__eyebrow">Ricompense</p>
          <h3>Badge e premi sbloccabili</h3>
          <p>Ogni livello sblocca vantaggi dedicati: corsie preferenziali, coupon sponsor, premi esclusivi.</p>
        </header>
        <div class="reward-grid">
          <div v-for="reward in rewards" :key="reward.id" class="reward-tile">
            <p class="reward-tile__level">{{ reward.level }}</p>
            <h4 class="reward-tile__title">{{ reward.title }}</h4>
            <p class="reward-tile__description">{{ reward.description }}</p>
            <p class="reward-tile__status" :class="{ 'reward-tile__status--active': reward.unlocked }">
              {{ reward.unlocked ? 'Sbloccato' : `Richiede ${reward.required} pt` }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
const props = defineProps({
  points: { type: Number, default: 0 },
  level: { type: Object, required: true },
  progressToNext: { type: Number, default: 0 },
  progressLabel: { type: String, default: '' },
  missions: { type: Array, default: () => [] },
  leaderboard: { type: Array, default: () => [] },
  rewards: { type: Array, default: () => [] },
  leaderboardPeriod: { type: String, default: 'daily' },
});

const periodLabel = (period) => {
  if (period === 'monthly') return 'Mensile';
  if (period === 'season') return 'Stagionale';
  return 'Giornaliera';
};
</script>

<style scoped>
.gamification-shell {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1.5rem 1rem;
}

.gamification-hero {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.12), rgba(14, 165, 233, 0.1));
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 24px;
  padding: 1.5rem;
  display: flex;
  gap: 1rem;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
}

.gamification-hero__eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.15em;
  font-size: 0.75rem;
  color: #cbd5e1;
  margin-bottom: 0.25rem;
}

.gamification-hero__title {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0;
}

.gamification-hero__subtitle {
  color: #cbd5e1;
  margin-top: 0.35rem;
}

.gamification-hero__actions {
  display: flex;
  gap: 0.75rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}

.level-card {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.3);
  border-radius: 18px;
  padding: 1rem 1.25rem;
  width: 280px;
  box-shadow: 0 18px 42px rgba(8, 15, 28, 0.4);
}

.level-card__label {
  text-transform: uppercase;
  letter-spacing: 0.14em;
  font-size: 0.75rem;
  color: #94a3b8;
}

.level-card__title {
  margin: 0.15rem 0;
  font-size: 1.4rem;
  font-weight: 700;
}

.level-card__points {
  margin: 0;
  color: #cbd5e1;
}

.level-progress {
  margin-top: 0.75rem;
}

.level-progress__bar {
  background: rgba(148, 163, 184, 0.2);
  height: 10px;
  border-radius: 999px;
  overflow: hidden;
}

.level-progress__fill {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, #22d3ee, #60a5fa);
}

.level-progress__hint {
  margin: 0.3rem 0 0;
  color: #cbd5e1;
  font-size: 0.9rem;
}

.gamification-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1rem;
}

.gamification-card {
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 20px;
  padding: 1.25rem;
  box-shadow: 0 18px 42px rgba(8, 15, 28, 0.4);
}

.gamification-card__header h3 {
  margin: 0.25rem 0;
  font-size: 1.2rem;
}

.gamification-card__eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.2em;
  color: #94a3b8;
  font-size: 0.75rem;
}

.mission-list {
  list-style: none;
  padding: 0;
  margin: 1rem 0 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.mission-step {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.mission-step--active {
  border-color: #38bdf8;
  box-shadow: 0 14px 26px rgba(56, 189, 248, 0.18);
}

.mission-step--done {
  border-color: #34d399;
}

.mission-step__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.mission-step__title {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.mission-step__badge {
  background: rgba(148, 163, 184, 0.15);
  color: #e2e8f0;
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-weight: 700;
}

.mission-step__eyebrow {
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #94a3b8;
  font-size: 0.75rem;
}

.mission-step__description {
  margin: 0;
  color: #cbd5e1;
}

.mission-step__points {
  margin: 0;
  color: #22d3ee;
  font-weight: 700;
}

.mission-step__footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}

.mission-step__status {
  color: #cbd5e1;
}

.mission-step__feedback {
  color: #34d399;
  font-weight: 600;
}

.leaderboard-tabs {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.25rem;
  margin: 0.75rem 0;
}

.leaderboard-tab {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.2);
  color: #e2e8f0;
  padding: 0.5rem;
  border-radius: 12px;
}

.leaderboard-tab--active {
  border-color: #60a5fa;
  box-shadow: 0 12px 24px rgba(96, 165, 250, 0.2);
}

.leaderboard-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.leaderboard-row {
  background: rgba(30, 41, 59, 0.75);
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 12px;
  padding: 0.65rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.leaderboard-row--me {
  border-color: #22d3ee;
}

.leaderboard-row__meta {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.leaderboard-row__rank {
  background: rgba(148, 163, 184, 0.15);
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
}

.leaderboard-row__hint {
  margin: 0;
  color: #94a3b8;
}

.leaderboard-row__name {
  margin: 0;
  font-weight: 700;
}

.leaderboard-row__score {
  text-align: right;
}

.leaderboard-row__points {
  margin: 0;
}

.leaderboard-row__delta {
  margin: 0;
  color: #22d3ee;
  font-size: 0.85rem;
}

.reward-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.reward-tile {
  background: rgba(30, 41, 59, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 14px;
  padding: 0.75rem;
}

.reward-tile__level {
  text-transform: uppercase;
  letter-spacing: 0.14em;
  color: #94a3b8;
  font-size: 0.75rem;
}

.reward-tile__title {
  margin: 0.1rem 0;
}

.reward-tile__description {
  margin: 0;
  color: #cbd5e1;
}

.reward-tile__status {
  margin: 0.4rem 0 0;
  color: #94a3b8;
  font-weight: 600;
}

.reward-tile__status--active {
  color: #34d399;
}

.btn {
  border: 1px solid transparent;
  border-radius: 12px;
  padding: 0.55rem 1rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn.primary {
  background: linear-gradient(90deg, #22d3ee, #60a5fa);
  color: #0b172a;
  font-weight: 700;
}

.btn.ghost {
  background: transparent;
  border-color: rgba(148, 163, 184, 0.5);
  color: #e2e8f0;
}

.btn.outline {
  background: transparent;
  border-color: rgba(148, 163, 184, 0.5);
  color: #e2e8f0;
}
</style>

<script setup>
import { onMounted, ref } from 'vue';
import { apiClient } from '../../api';

const games = ref([]);
const loading = ref(false);
const loadError = ref('');
const options = ['Calcio', 'Pallavolo', 'Pallacanestro', 'Tutti'];

async function load() {
  loading.value = true;
  loadError.value = '';
  try {
    const { data } = await apiClient.get('/admin/master/minigames');
    games.value = data.map(game => ({ ...game, selected: [...game.sports], saving: false, error: '', saved: false }));
  } catch {
    loadError.value = 'Impossibile caricare i minigiochi. Riprova.';
  } finally {
    loading.value = false;
  }
}

function toggle(game, sport, checked) {
  game.saved = false;
  game.error = '';
  if (sport === 'Tutti') {
    game.selected = checked ? ['Tutti'] : [];
  } else {
    game.selected = game.selected.filter(value => value !== 'Tutti' && value !== sport);
    if (checked) game.selected.push(sport);
  }
}

function isDirty(game) {
  return [...game.selected].sort().join('|') !== [...game.sports].sort().join('|');
}

async function save(game) {
  if (game.saving || !isDirty(game)) return;
  game.saving = true;
  game.error = '';
  game.saved = false;
  try {
    const { data } = await apiClient.put(`/admin/master/minigames/${game.id}`, { sports: [...game.selected] });
    game.sports = [...data.sports];
    game.selected = [...data.sports];
    game.saved = true;
  } catch {
    game.error = 'Salvataggio non riuscito. Le modifiche sono ancora da salvare.';
  } finally {
    game.saving = false;
  }
}

onMounted(load);
</script>

<template>
  <section class="gamification" aria-labelledby="gamification-title">
    <header class="heading">
      <div>
        <h2 id="gamification-title">Gamification</h2>
        <p>Decidi quali minigiochi rendere disponibili alle società in base al loro sport.</p>
      </div>
      <button type="button" class="refresh" :disabled="loading || games.some(game => game.saving || isDirty(game))" @click="load">Aggiorna</button>
    </header>
    <p class="hint">Seleziona uno o più sport, oppure Tutti. Senza selezioni il gioco è disabilitato. Le impostazioni valgono per tutte le società dello sport scelto; restano necessarie le eventuali attivazioni del singolo evento.</p>
    <p v-if="loading" role="status">Caricamento minigiochi…</p>
    <p v-if="loadError" class="error" role="alert">{{ loadError }}</p>
    <div v-if="!loading && !loadError" class="game-list">
      <article v-for="game in games" :key="game.id" class="game-card">
        <div class="game-info">
          <h3>{{ game.name }}</h3>
          <p>{{ game.description }}</p>
          <small>{{ game.sports.length ? `Disponibile per: ${game.sports.join(', ')}` : 'Disabilitato per tutti gli sport' }}</small>
        </div>
        <fieldset :disabled="game.saving">
          <legend class="sr-only">Sport per {{ game.name }}</legend>
          <label v-for="sport in options" :key="sport" :class="['sport-option', { selected: game.selected.includes(sport) }]">
            <input type="checkbox" :checked="game.selected.includes(sport)" @change="toggle(game, sport, $event.target.checked)" />
            {{ sport }}
          </label>
        </fieldset>
        <div class="save-row">
          <span v-if="game.error" class="error" role="alert">{{ game.error }}</span>
          <span v-else-if="game.saved" class="success" role="status">Impostazioni salvate</span>
          <span v-else-if="isDirty(game)" class="pending" role="status">Modifiche da salvare</span>
          <button type="button" class="save" :disabled="game.saving || !isDirty(game)" :aria-label="`Salva sport per ${game.name}`" @click="save(game)">{{ game.saving ? 'Salvataggio…' : 'Salva' }}</button>
        </div>
      </article>
      <p v-if="!games.length" class="hint">Nessun minigioco disponibile.</p>
    </div>
  </section>
</template>

<style scoped>
.gamification { color: #0f172a; }
.heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
h2, h3 { margin: 0; }
.heading p, .game-info p, .hint { color: #64748b; line-height: 1.5; }
.hint { font-size: 13px; margin: 14px 0 20px; }
.game-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.game-card { border: 1px solid #e2e8f0; border-radius: 16px; background: #f8fafc; padding: 20px; }
.game-info p { font-size: 13px; min-height: 40px; margin: 8px 0; }
.game-info small { color: #475569; }
fieldset { display: flex; flex-wrap: wrap; gap: 8px; margin: 18px 0; padding: 0; border: 0; }
.sport-option { display: flex; align-items: center; gap: 7px; border: 1px solid #cbd5e1; border-radius: 9px; padding: 9px 12px; cursor: pointer; font-size: 13px; background: white; }
.sport-option.selected { background: #e0f2fe; border-color: #0284c7; color: #075985; }
input { accent-color: #0284c7; }
.save-row { display: flex; justify-content: flex-end; align-items: center; gap: 12px; min-height: 38px; font-size: 12px; }
button { border-radius: 9px; padding: 9px 15px; cursor: pointer; font-weight: 700; }
.refresh { background: white; color: #0f172a; border: 1px solid #cbd5e1; }
.save { color: white; background: #0284c7; border: 1px solid #0284c7; }
button:disabled { opacity: .45; cursor: default; }
.error { color: #b91c1c; }
.success { color: #15803d; }
.pending { color: #a16207; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
@media (max-width: 850px) { .game-list { grid-template-columns: 1fr; } }
</style>

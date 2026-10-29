<script setup>
import { ref } from "vue";
defineProps({ state: Object });
const emit = defineEmits(["debug", "focus"]);
const result = ref("3-0");
</script>
<template>
  <details class="debug">
    <summary>Laboratorio demo · solo sviluppo</summary>
    <p>Nessuna operazione modifica profili o monetine reali.</p>
    <div>
      <button @click="emit('debug', 'day')">Giorno successivo</button
      ><button @click="emit('debug', 'location')">Cambia posizione</button
      ><button @click="emit('focus', 'treasure')">Mostra il tesoro</button
      ><button @click="emit('debug', 'collect')">Simula ritrovamento</button
      ><button @click="emit('debug', 'reset-claim')">Ripristina missione</button
      ><button @click="emit('debug', 'coins')">+100 monetine demo</button
      ><button @click="emit('debug', 'unlock', 'celebration')">
        Sblocca esultanza</button
      ><button @click="emit('debug', 'match')">Nuova partita demo</button
      ><select v-model="result" aria-label="Risultato demo">
        <option
          v-for="score in ['3-0', '3-1', '3-2', '2-3', '1-3', '0-3']"
          :key="score"
        >
          {{ score }}
        </option></select
      ><button @click="emit('debug', 'result', result)">
        Conferma risultato e verifica</button
      ><button @click="emit('debug', 'friend')">Simula amico accettato</button
      ><select
        aria-label="Trofeo demo"
        @change="emit('debug', 'trophy', $event.target.value)"
      >
        <option value="">Sblocca un trofeo demo…</option>
        <option v-for="t in state.trophies" :value="t.key" :key="t.key">
          {{ t.title }}
        </option></select
      ><button @click="emit('debug', 'reset')">Ripristina dati demo</button>
    </div>
  </details>
</template>
<style scoped>
.debug {
  margin-top: 24px;
  border-top: 1px dashed #b9c8a0;
  padding-top: 16px;
}
.debug summary {
  cursor: pointer;
  font-size: 11px;
}
.debug p {
  font-size: 10px;
}
.debug > div {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.debug button {
  font: inherit;
  color: inherit;
  font-size: 10px;
  flex: 1 0 44%;
  border: 0;
  cursor: pointer;
  border-radius: 9px;
  background: #e5edda;
  padding: 10px;
  min-height: 42px;
}
.debug select {
  font: inherit;
  color: inherit;
  width: 100%;
  font-size: 16px;
  min-height: 44px;
  border: 1px solid #c9d7b6;
  background: #fffef4;
  border-radius: 8px;
  padding: 9px;
}
</style>

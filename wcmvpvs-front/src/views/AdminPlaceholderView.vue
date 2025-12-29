<template>
  <AppShell>
    <section class="page">
      <div class="card">
        <p class="eyebrow">Sezione in arrivo</p>
        <h1>{{ title }}</h1>
        <p class="muted">{{ description }}</p>
        <div class="callout">
          <p class="muted">Questa sezione sarà presto collegata ai dati reali dell'MVP System.</p>
          <button class="primary" type="button" @click="goEvents">Torna agli eventi</button>
        </div>
      </div>
    </section>
  </AppShell>
</template>

<script setup>
import { computed } from 'vue';
import AppShell from '../components/layout/AppShell.vue';
import { useRoute, useRouter } from '../router';

const route = useRoute();
const router = useRouter();

const title = computed(() => route.value?.meta?.title || 'Modulo admin');
const description = computed(() => route.value?.meta?.description || 'Layout placeholder');

function goEvents() {
  router.push('/admin/events');
}
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-strong);
  border-radius: 1rem;
  padding: 1.5rem;
}

.eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-weight: 800;
  color: var(--text-muted);
  margin: 0;
}

h1 {
  margin: 0.2rem 0;
}

.muted {
  color: var(--text-muted);
  margin: 0;
}

.callout {
  margin-top: 1rem;
  padding: 1rem;
  border-radius: 1rem;
  border: 1px dashed var(--border-strong);
  background: rgba(255, 255, 255, 0.02);
}

button {
  margin-top: 0.75rem;
  border: none;
  border-radius: 0.9rem;
  padding: 0.8rem 1rem;
  font-weight: 800;
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  color: #0b1220;
  cursor: pointer;
}
</style>

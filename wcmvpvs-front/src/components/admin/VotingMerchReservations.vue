<template>
  <section v-if="Array.isArray(reservations)" class="voting-merch-reservations">
    <component :is="heading" class="reservations-heading">
      {{ reservations.length ? 'PRENOTAZIONI DIVISE EFFETTUATE' : 'NESSUNA PRENOTAZIONE DIVISE' }}
    </component>
    <ul v-if="reservations.length" class="reservations-list">
      <li v-for="reservation in reservations" :key="reservation.id">{{ reservation.name }}</li>
    </ul>
  </section>
</template>

<script setup>
defineProps({
  reservations: Array,
  heading: { type: String, default: 'h3' },
});
</script>

<style scoped>
.voting-merch-reservations {
  margin: 1.25rem 0;
  padding: clamp(.9rem, 3vw, 1.25rem);
  border: 1px solid var(--border, #e3e7ef);
  border-radius: .75rem;
  background: var(--v2-soft, var(--bg-base, #f5f6f8));
}
.voting-merch-reservations .reservations-heading {
  margin: 0;
  font-size: clamp(1.15rem, 3vw, 1.5rem);
  font-weight: 800;
  line-height: 1.25;
}
.reservations-list {
  margin: .75rem 0 0;
  padding-inline-start: 1.25rem;
  list-style: disc;
  overflow-wrap: anywhere;
}
.reservations-list li + li { margin-top: .3rem; }
</style>

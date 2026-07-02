<script setup>
defineProps({
  sponsors: { type: Array, default: () => [] } // { id, name, logo?, url? }
})

/**
 * Nota monetizzazione: ogni impression della strip è inventory vendibile.
 * Se serve tracking per il report sponsor post-torneo, agganciare qui un
 * IntersectionObserver → POST /api/v1/analytics/sponsor-impression (batch).
 */
</script>

<template>
  <section class="sponsor-strip" v-if="sponsors.length">
    <div class="label">Main Sponsor</div>
    <div class="sponsor-logos">
      <component
        v-for="s in sponsors"
        :key="s.id"
        :is="s.url ? 'a' : 'span'"
        :href="s.url"
        target="_blank"
        rel="noopener"
        class="sp-item"
      >
        <img v-if="s.logo" :src="s.logo" :alt="s.name" loading="lazy" />
        <span v-else class="sp-text">{{ s.name }}</span>
      </component>
    </div>
  </section>
</template>

<style scoped>
.sponsor-strip {
  flex: none; background: var(--tm-surface);
  border: 1px solid var(--tm-border); border-radius: var(--tm-radius);
  padding: clamp(7px,1.2dvh,11px) 12px clamp(8px,1.3dvh,12px);
}
.label { font-size: clamp(8px,1.15dvh,10px); font-weight: 800; letter-spacing: 1.6px; color: var(--tm-text-faint); text-transform: uppercase; }
.sponsor-logos {
  display: flex; align-items: center; justify-content: space-between;
  gap: 12px; margin-top: clamp(5px,.9dvh,10px); overflow-x: auto; scrollbar-width: none;
}
.sponsor-logos::-webkit-scrollbar { display: none; }
.sp-item { display: inline-flex; align-items: center; text-decoration: none; color: inherit; }
.sponsor-logos img { height: clamp(16px,2.6dvh,24px); object-fit: contain; filter: grayscale(1) brightness(3); }
.sp-text { font-weight: 900; font-size: clamp(10px,1.6dvh,13px); letter-spacing: .5px; white-space: nowrap; opacity: .95; }
</style>

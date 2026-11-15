<template>
  <button v-if="visible && sheet" type="button" aria-haspopup="dialog" @click="$emit('open', $event.currentTarget)"><slot>Shop</slot></button>
  <a v-else-if="visible" :href="href" @click="onClick"><slot>Shop</slot></a>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { fetchMerchSummary, trackMerchEvent } from './merchApi';
import { fanPortalPath } from './shopModel';

const props = defineProps({
  slug: { type: String, required: true },
  source: { type: String, default: '' },
  eventId: { type: Number, default: 0 },
  matchdayReady: { type: Boolean, default: true },
  includeFeatured: Boolean,
  sheet: Boolean,
});
const emit = defineEmits(['open', 'featured']);
const visible = ref(false);
const href = computed(() => `${fanPortalPath(props.slug)}/shop`);

watch(() => [props.slug, props.eventId, props.matchdayReady], async ([slug, eventId, matchdayReady], previous, onCleanup) => {
  visible.value = false;
  if (props.includeFeatured && slug !== previous?.[0]) emit('featured', null);
  if (!slug || typeof window === 'undefined') return;
  const controller = new AbortController();
  onCleanup(() => controller.abort());
  const includeFeatured = props.includeFeatured && matchdayReady;
  try {
    const summary = await fetchMerchSummary(slug, controller.signal, matchdayReady ? eventId : 0, includeFeatured);
    if (controller.signal.aborted) return;
    visible.value = Boolean(summary?.settings?.enabled && summary.settings.visible);
    if (includeFeatured) emit('featured', summary?.settings?.enabled ? summary.featured_product || null : null);
  } catch {
    if (includeFeatured && !controller.signal.aborted) emit('featured', null);
  }
}, { immediate: true });

function onClick() {
  if (props.source === 'arena') trackMerchEvent(props.slug, 'merch_outbound_click');
}
</script>

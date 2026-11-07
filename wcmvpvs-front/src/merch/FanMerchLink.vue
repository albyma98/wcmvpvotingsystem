<template>
  <a v-if="visible" :href="href" @click="onClick"><slot>Shop</slot></a>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { fetchMerchVisibility, trackMerchEvent } from './merchApi';
import { fanPortalPath } from './shopModel';

const props = defineProps({
  slug: { type: String, required: true },
  source: { type: String, default: '' },
});
const visible = ref(false);
const href = computed(() => `${fanPortalPath(props.slug)}/shop`);

watch(() => props.slug, async (slug, previous, onCleanup) => {
  visible.value = false;
  if (!slug || typeof window === 'undefined') return;
  const controller = new AbortController();
  onCleanup(() => controller.abort());
  try {
    const result = await fetchMerchVisibility(slug, controller.signal);
    if (!controller.signal.aborted) visible.value = result;
  } catch { /* An unavailable shop should not block the fan portal. */ }
}, { immediate: true });

function onClick() {
  if (props.source === 'arena') trackMerchEvent(props.slug, 'merch_outbound_click');
}
</script>

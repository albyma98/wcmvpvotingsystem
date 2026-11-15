<template>
  <button class="featured-merch" type="button" aria-haspopup="dialog" @click="$emit('open', $event.currentTarget, product.id)">
    <img v-if="product.media?.[0]?.url" :src="merchMediaUrl(product.media[0].url)" alt="" width="80" height="80" decoding="async" />
    <span class="featured-merch-copy">
      <strong class="featured-merch-title">{{ product.name }}</strong>
      <span class="featured-merch-prices">
        <del v-if="price.originalCents != null">{{ merchMoney(price.originalCents) }}</del>
        <strong>{{ price.label }}</strong>
      </span>
      <span class="featured-merch-action">{{ action }} <span aria-hidden="true">→</span></span>
    </span>
  </button>
</template>

<script setup>
import { computed } from 'vue';
import { merchMediaUrl } from './merchApi';
import { isMerchAvailable, merchMoney, merchPriceSummary } from './shopModel';

const props = defineProps({ product: { type: Object, required: true } });
defineEmits(['open']);

const price = computed(() => merchPriceSummary(props.product));
const action = computed(() => {
  if (!isMerchAvailable(props.product)) return 'Al momento esaurito';
  return props.product.offline_enabled ? 'Prenota ora, non fartela scappare' : 'Acquista ora, non fartela scappare';
});
</script>

<style scoped>
.featured-merch {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--red-line);
  border-radius: var(--home-radius);
  background: linear-gradient(125deg, var(--red-soft), var(--card) 66%);
  box-shadow: var(--home-shadow);
  color: var(--ink);
  font: inherit;
  text-align: start;
  cursor: pointer;
}
.featured-merch:hover { border-color: var(--red-text); }
.featured-merch img {
  flex: none;
  width: clamp(64px, 20vw, 84px);
  height: clamp(64px, 20vw, 84px);
  border-radius: 10px;
  background: var(--card);
  object-fit: cover;
}
.featured-merch-copy { display: grid; flex: 1; min-width: 0; gap: 4px; overflow-wrap: anywhere; }
.featured-merch-title { line-height: 1.2; }
.featured-merch-prices { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px 8px; }
.featured-merch-prices del { color: var(--muted); font-size: .8125rem; }
.featured-merch-action { color: var(--red-text); font-size: .8125rem; font-weight: 750; }
</style>

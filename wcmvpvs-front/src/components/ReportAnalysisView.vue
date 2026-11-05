<script setup>
import { analysisMetricReferences } from '../recipientReport.js';
defineProps({ analysis: { type: Object, required: true }, kpis: { type: Array, default: () => [] }, status: String });
</script>
<template>
  <section class="report-analysis" aria-label="Analisi del report">
    <p class="report-analysis__source">{{ status === 'ready' ? 'Analisi OpenAI validata' : 'Sintesi deterministica' }}</p>
    <h3>{{ analysis.headline }}</h3><p>{{ analysis.summary }}</p>
    <template v-for="(items, kind) in { 'Evidenze positive': analysis.positive_insights, 'Criticità e attenzione': analysis.critical_insights }" :key="kind"><div v-if="items?.length"><h4>{{ kind }}</h4><article v-for="(item, index) in items" :key="index"><strong>{{ item.title }}</strong><p>{{ item.description }}</p><small>{{ analysisMetricReferences(item, kpis) }}</small></article></div></template>
    <div v-if="analysis.recommendations?.length"><h4>Indicazioni operative</h4><article v-for="(item, index) in analysis.recommendations" :key="index"><strong>{{ item.action }}</strong><p>{{ item.reason }}</p><small>Priorità {{ { high: 'alta', medium: 'media', low: 'bassa' }[item.priority] }} · {{ analysisMetricReferences(item, kpis) }}</small></article></div>
    <details v-if="analysis.limitations?.length"><summary>Limiti dell’analisi</summary><ul><li v-for="(limit, index) in analysis.limitations" :key="index">{{ limit }}</li></ul></details>
  </section>
</template>
<style scoped>
.report-analysis{background:#f1f7f6;border:1px solid #cde2dc;border-radius:12px;padding:18px;font-size:12px;line-height:1.6;color:#243e50;overflow-wrap:anywhere}.report-analysis__source{font-size:10px;text-transform:uppercase;letter-spacing:.06em;color:#146451}.report-analysis h3{margin:4px 0 10px;font-size:18px}.report-analysis h4{margin-bottom:8px}.report-analysis article{margin:12px 0}.report-analysis article p{margin:4px 0}.report-analysis small{color:#516378}.report-analysis summary{cursor:pointer;margin-top:12px}
</style>

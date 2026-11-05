<script setup>
import { formatReportValue } from '../engagementReport.js';
import { comparisonLabel, recipientSourceLabel } from '../recipientReport.js';
defineProps({ kpi: { type: Object, required: true }, compare: Boolean, busy: Boolean, eventContext: Boolean });
defineEmits(['complete']);
</script>

<template>
  <article class="report-card" :class="{ 'is-missing': kpi.value == null }">
    <div class="report-card__top"><h4>{{ kpi.label }}</h4><strong>{{ formatReportValue(kpi.value, kpi.unit) }}</strong></div>
    <small v-if="eventContext">Contesto complessivo dell’evento, non copertura del singolo partner</small>
    <p v-if="compare" class="report-card__comparison"><span v-if="kpi.comparison?.previous != null">Precedente: {{ formatReportValue(kpi.comparison.previous, kpi.unit) }} · </span>{{ comparisonLabel(kpi) }}<small v-for="source in kpi.comparison?.previous_provenance || []" :key="source.field_key">Precedente da fonte manuale {{ source.verification_status === 'verified' ? 'verificata' : 'NON verificata' }}</small></p>
    <p v-if="kpi.value == null" class="report-card__missing">{{ kpi.availability?.message || 'Misura non disponibile.' }}</p>
    <button v-if="kpi.value == null || kpi.provenance?.some(m => m.source_type === 'manual_kpi')" type="button" :disabled="busy" @click="$emit('complete', kpi)">{{ kpi.value == null ? 'Inserisci valore' : 'Modifica valore manuale' }}</button>
    <details><summary>Definizione e provenienza</summary><p>{{ kpi.description }}</p><p v-if="kpi.numerator != null && kpi.denominator != null">Base: {{ formatReportValue(kpi.numerator) }} / {{ formatReportValue(kpi.denominator) }}</p><p>{{ recipientSourceLabel(kpi) }} · {{ kpi.note }}</p><p v-for="source in kpi.provenance || []" :key="source.field_key">{{ source.source_description }} · {{ source.verification_status === 'verified' ? 'Verificata' : 'NON verificata' }} · Revisione {{ source.revision }}</p></details>
  </article>
</template>

<style scoped>
.report-card{border:1px solid #dce5ed;border-radius:12px;padding:16px;background:white;min-width:0}.report-card.is-missing{border-color:#dfc88a;background:#fffcf2}.report-card__top{display:flex;justify-content:space-between;gap:16px;align-items:flex-start}.report-card h4{font-size:13px;line-height:1.5;margin:0}.report-card strong{font-size:24px;color:#0e2139;white-space:nowrap}.report-card p,.report-card small,.report-card details{font-size:11px;color:#516378;line-height:1.6;overflow-wrap:anywhere}.report-card small{display:block;margin-top:6px}.report-card__comparison{font-variant-numeric:tabular-nums}.report-card .report-card__missing{color:#805918}.report-card summary{cursor:pointer;margin-top:10px}.report-card button{padding:7px 10px;border:1px solid #bba569;border-radius:6px;background:white;color:#624d1e;cursor:pointer}
</style>

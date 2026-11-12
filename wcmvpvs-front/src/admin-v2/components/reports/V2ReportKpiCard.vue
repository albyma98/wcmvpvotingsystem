<script>
import { useController } from '../../../components/ReportKpiCard.vue?admin-v2-controller';

export default {name:'V2ReportKpiCard',components:{},props:{kpi:{type:Object,required:true},compare:Boolean,eventContext:Boolean},setup(props,{emit}) {return useController(props,emit);}};
</script>
<template>
  <article class="v2-report-card" :class="{ 'is-missing': kpi.value == null }">
    <div class="v2-report-card__top"><h4>{{ kpi.label }}</h4><strong>{{ formatReportValue(kpi.value, kpi.unit) }}</strong></div>
    <small v-if="eventContext">Contesto complessivo dell’evento, non copertura del singolo partner</small>
    <p v-if="compare" class="v2-report-card__comparison"><span v-if="kpi.comparison?.previous != null">Precedente: {{ formatReportValue(kpi.comparison.previous, kpi.unit) }} · </span>{{ comparisonLabel(kpi) }}<small v-for="source in kpi.comparison?.previous_provenance || []" :key="source.field_key">Precedente da fonte manuale {{ source.verification_status === 'verified' ? 'verificata' : 'NON verificata' }}</small></p>
    <p v-if="kpi.value == null" class="v2-report-card__missing">{{ kpi.availability?.message || 'Misura non disponibile.' }}</p>
    <details><summary>Definizione e provenienza</summary><p>{{ kpi.description }}</p><p v-if="kpi.numerator != null && kpi.denominator != null">Base: {{ formatReportValue(kpi.numerator) }} / {{ formatReportValue(kpi.denominator) }}</p><p>{{ recipientSourceLabel(kpi) }}<template v-if="kpi.note && kpi.note !== kpi.description"> · {{ kpi.note }}</template></p><p v-for="source in kpi.provenance || []" :key="source.field_key">{{ source.source_description }} · {{ source.verification_status === 'verified' ? 'Verificata' : 'NON verificata' }} · Revisione {{ source.revision }}</p></details>
  </article>
</template>

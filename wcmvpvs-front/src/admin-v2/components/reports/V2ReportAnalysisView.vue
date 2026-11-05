<script>
import { useController } from '../../../components/ReportAnalysisView.vue?admin-v2-controller';

export default {
  name: 'V2ReportAnalysisView',
  props: {
    analysis: { type: Object, required: true },
    kpis: { type: Array, default: () => [] },
    status: String,
    review: { type: Object, default: () => ({ state: 'draft' }) },
    dirty: Boolean,
    busy: Boolean,
    warnings: { type: Array, default: () => [] },
  },
  emits: ['update:analysis', 'save', 'approve', 'discard'],
  setup(props, { emit }) { return useController(props, emit); },
};
</script>
<template>
  <section class="v2-report-analysis" aria-label="Analisi del report">
    <p class="v2-report-analysis__source">{{ status === 'ready' ? 'Analisi OpenAI' : 'Sintesi locale' }} · {{ dirty || review.state === 'draft' ? 'Da approvare' : review.state === 'approved' ? 'Approvata' : 'Scartata' }}</p>
    <p v-if="review.state === 'approved' && !dirty" role="status">Approvata da {{ review.reviewed_by || 'amministratore' }}{{ review.reviewed_at ? ` · ${new Date(review.reviewed_at).toLocaleString('it-IT')}` : '' }}. Ogni modifica richiede una nuova approvazione.</p>
    <p v-if="review.state === 'discarded' && !dirty" role="status">Analisi esclusa dal PDF. Puoi modificarla e approvarla nuovamente.</p>
    <p>Rivedi titoli e testi, confronta eventuali numeri con i KPI e approva solo la versione che vuoi inserire nel PDF. Modificare, salvare, approvare o scartare non chiama OpenAI.</p>
    <p v-for="warning in warnings" :key="warning" class="v2-report-analysis__warning">{{ warning }}</p>
    <fieldset :disabled="busy">
      <legend class="v2-report-analysis__legend">Testi dell’analisi modificabili</legend>
      <label>Titolo dell’analisi<input type="text" :value="analysis.headline" maxlength="180" @input="setText('headline', $event.target.value)" /></label>
      <label>Sintesi dell’analisi<textarea :value="analysis.summary" maxlength="6000" rows="6" @input="setText('summary', $event.target.value)"></textarea></label>
      <template v-for="[list, title] in [['positive_insights', 'Evidenze positive'], ['critical_insights', 'Criticità e attenzione']]" :key="list">
        <section v-if="analysis[list]?.length"><h4>{{ title }}</h4><article v-for="(item, index) in analysis[list]" :key="index">
          <label>Titolo evidenza {{ index + 1 }}<input type="text" :value="item.title" maxlength="180" @input="setItem(list, index, 'title', $event.target.value)" /></label>
          <label>Testo evidenza {{ index + 1 }}<textarea :value="item.description" maxlength="3000" rows="3" @input="setItem(list, index, 'description', $event.target.value)"></textarea></label>
          <small>{{ analysisMetricReferences(item, kpis) }}</small><button type="button" class="v2-report-analysis__remove" @click="removeItem(list, index)">Rimuovi evidenza {{ index + 1 }}</button>
        </article></section>
      </template>
      <section v-if="analysis.recommendations?.length"><h4>Indicazioni operative</h4><article v-for="(item, index) in analysis.recommendations" :key="index">
        <label>Titolo indicazione {{ index + 1 }}<input type="text" :value="item.action" maxlength="500" @input="setItem('recommendations', index, 'action', $event.target.value)" /></label>
        <label>Motivazione {{ index + 1 }}<textarea :value="item.reason" maxlength="3000" rows="3" @input="setItem('recommendations', index, 'reason', $event.target.value)"></textarea></label>
        <label>Priorità<select :value="item.priority" @change="setItem('recommendations', index, 'priority', $event.target.value)"><option value="high">Alta</option><option value="medium">Media</option><option value="low">Bassa</option></select></label>
        <small>{{ analysisMetricReferences(item, kpis) }}</small><button type="button" class="v2-report-analysis__remove" @click="removeItem('recommendations', index)">Rimuovi indicazione {{ index + 1 }}</button>
      </article></section>
      <details><summary>Limiti e note dell’analisi (modificabili)</summary><div v-for="(limit, index) in analysis.limitations" :key="index"><label>Limite o nota {{ index + 1 }}<textarea :value="limit" maxlength="1000" rows="2" @input="setLimit(index, $event.target.value)"></textarea></label><button type="button" class="v2-report-analysis__remove" @click="removeItem('limitations', index)">Rimuovi nota {{ index + 1 }}</button></div><button type="button" :disabled="analysis.limitations.length >= 32" @click="addLimit">Aggiungi nota</button></details>
      <div class="v2-report-analysis__actions"><button type="button" :disabled="!dirty" @click="emit('save')">Salva bozza</button><button type="button" class="v2-report-analysis__approve" :disabled="review.state === 'approved' && !dirty" @click="emit('approve')">Approva analisi</button><button type="button" class="v2-report-analysis__discard" :disabled="review.state === 'discarded' && !dirty" @click="emit('discard')">Scarta analisi</button></div>
      <p v-if="dirty" role="status">Modifiche da salvare o approvare. La versione modificata non è ancora inclusa nel PDF.</p>
    </fieldset>
  </section>
</template>

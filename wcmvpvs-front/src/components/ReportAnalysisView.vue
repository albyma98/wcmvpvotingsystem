<script setup>
import { analysisMetricReferences } from '../recipientReport.js';

const props = defineProps({ analysis: { type: Object, required: true }, kpis: { type: Array, default: () => [] }, status: String, review: { type: Object, default: () => ({ state: 'draft' }) }, dirty: Boolean, busy: Boolean, warnings: { type: Array, default: () => [] } });
const emit = defineEmits(['update:analysis', 'save', 'approve', 'discard']);
function edit(change) {
  if (props.busy) return;
  const value = JSON.parse(JSON.stringify(props.analysis));
  change(value); emit('update:analysis', value);
}
function setText(field, value) { edit(a => { a[field] = value; }); }
function setItem(list, index, field, value) { edit(a => { a[list][index][field] = value; }); }
function setLimit(index, value) { edit(a => { a.limitations[index] = value; }); }
function removeItem(list, index) { edit(a => { a[list].splice(index, 1); }); }
function addLimit() { edit(a => { if (a.limitations.length < 32) a.limitations.push(''); }); }
</script>

<template>
  <section class="report-analysis" aria-label="Analisi del report">
    <p class="report-analysis__source">{{ status === 'ready' ? 'Analisi OpenAI' : 'Sintesi locale' }} · {{ dirty || review.state === 'draft' ? 'Da approvare' : review.state === 'approved' ? 'Approvata' : 'Scartata' }}</p>
    <p v-if="review.state === 'approved' && !dirty" role="status">Approvata da {{ review.reviewed_by || 'amministratore' }}{{ review.reviewed_at ? ` · ${new Date(review.reviewed_at).toLocaleString('it-IT')}` : '' }}. Ogni modifica richiede una nuova approvazione.</p>
    <p v-if="review.state === 'discarded' && !dirty" role="status">Analisi esclusa dal PDF. Puoi modificarla e approvarla nuovamente.</p>
    <p>Rivedi titoli e testi, confronta eventuali numeri con i KPI e approva solo la versione che vuoi inserire nel PDF. Modificare, salvare, approvare o scartare non chiama OpenAI.</p>
    <p v-for="warning in warnings" :key="warning" class="report-analysis__warning">{{ warning }}</p>
    <fieldset :disabled="busy">
      <legend class="report-analysis__legend">Testi dell’analisi modificabili</legend>
      <label>Titolo dell’analisi<input type="text" :value="analysis.headline" maxlength="180" @input="setText('headline', $event.target.value)" /></label>
      <label>Sintesi dell’analisi<textarea :value="analysis.summary" maxlength="6000" rows="6" @input="setText('summary', $event.target.value)"></textarea></label>
      <template v-for="[list, title] in [['positive_insights', 'Evidenze positive'], ['critical_insights', 'Criticità e attenzione']]" :key="list">
        <section v-if="analysis[list]?.length"><h4>{{ title }}</h4><article v-for="(item, index) in analysis[list]" :key="index">
          <label>Titolo evidenza {{ index + 1 }}<input type="text" :value="item.title" maxlength="180" @input="setItem(list, index, 'title', $event.target.value)" /></label>
          <label>Testo evidenza {{ index + 1 }}<textarea :value="item.description" maxlength="3000" rows="3" @input="setItem(list, index, 'description', $event.target.value)"></textarea></label>
          <small>{{ analysisMetricReferences(item, kpis) }}</small><button type="button" class="report-analysis__remove" @click="removeItem(list, index)">Rimuovi evidenza {{ index + 1 }}</button>
        </article></section>
      </template>
      <section v-if="analysis.recommendations?.length"><h4>Indicazioni operative</h4><article v-for="(item, index) in analysis.recommendations" :key="index">
        <label>Titolo indicazione {{ index + 1 }}<input type="text" :value="item.action" maxlength="500" @input="setItem('recommendations', index, 'action', $event.target.value)" /></label>
        <label>Motivazione {{ index + 1 }}<textarea :value="item.reason" maxlength="3000" rows="3" @input="setItem('recommendations', index, 'reason', $event.target.value)"></textarea></label>
        <label>Priorità<select :value="item.priority" @change="setItem('recommendations', index, 'priority', $event.target.value)"><option value="high">Alta</option><option value="medium">Media</option><option value="low">Bassa</option></select></label>
        <small>{{ analysisMetricReferences(item, kpis) }}</small><button type="button" class="report-analysis__remove" @click="removeItem('recommendations', index)">Rimuovi indicazione {{ index + 1 }}</button>
      </article></section>
      <details><summary>Limiti e note dell’analisi (modificabili)</summary><div v-for="(limit, index) in analysis.limitations" :key="index"><label>Limite o nota {{ index + 1 }}<textarea :value="limit" maxlength="1000" rows="2" @input="setLimit(index, $event.target.value)"></textarea></label><button type="button" class="report-analysis__remove" @click="removeItem('limitations', index)">Rimuovi nota {{ index + 1 }}</button></div><button type="button" :disabled="analysis.limitations.length >= 32" @click="addLimit">Aggiungi nota</button></details>
      <div class="report-analysis__actions"><button type="button" :disabled="!dirty" @click="emit('save')">Salva bozza</button><button type="button" class="report-analysis__approve" :disabled="review.state === 'approved' && !dirty" @click="emit('approve')">Approva analisi</button><button type="button" class="report-analysis__discard" :disabled="review.state === 'discarded' && !dirty" @click="emit('discard')">Scarta analisi</button></div>
      <p v-if="dirty" role="status">Modifiche da salvare o approvare. La versione modificata non è ancora inclusa nel PDF.</p>
    </fieldset>
  </section>
</template>

<style scoped>
.report-analysis{background:#f1f7f6;border:1px solid #cde2dc;border-radius:12px;padding:18px;font-size:12px;line-height:1.6;color:#243e50;overflow-wrap:anywhere}.report-analysis__source{font-size:10px;text-transform:uppercase;letter-spacing:.06em;color:#146451}.report-analysis h4{margin:24px 0 8px}.report-analysis article{margin:12px 0 20px;padding-bottom:12px;border-bottom:1px solid #cde2dc}.report-analysis small{display:block;color:#516378;margin:6px 0}.report-analysis summary{cursor:pointer;margin:16px 0 10px}.report-analysis fieldset{border:0;margin:0;padding:0;min-width:0}.report-analysis__legend{font-size:14px;font-weight:700;margin-bottom:8px}.report-analysis label{display:block;font-weight:600;margin:12px 0}.report-analysis input,.report-analysis textarea,.report-analysis select{display:block;width:100%;box-sizing:border-box;margin-top:5px;background:white;color:#243e50;border:1px solid #aebfbb;border-radius:7px;padding:10px;font:inherit;font-weight:400}.report-analysis textarea{resize:vertical;min-height:70px;line-height:1.6}.report-analysis button{font:inherit;cursor:pointer;border:1px solid #acc3bc;background:white;color:#274154;border-radius:7px;padding:8px 12px}.report-analysis button:disabled{opacity:.5;cursor:default}.report-analysis .report-analysis__approve{background:#0e6659;color:white;border-color:#0e6659}.report-analysis .report-analysis__discard{color:#9a2929;border-color:#cda9a6}.report-analysis .report-analysis__remove{font-size:11px;padding:5px 9px}.report-analysis__actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:20px}.report-analysis__warning{border-left:3px solid #ba8c21;padding:8px 12px;background:#fff9e8}.report-analysis :focus-visible{outline:3px solid #23a98e;outline-offset:2px}
</style>

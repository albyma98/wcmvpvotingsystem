<script>
import { useController } from '../../../components/RecipientReportBuilder.vue?admin-v2-controller';
import ReportKpiCard from './V2ReportKpiCard.vue';
import ReportAnalysisView from './V2ReportAnalysisView.vue';
export default {name:'V2RecipientReportBuilder',components:{ReportKpiCard,ReportAnalysisView},props:{eventId:{type:Number,required:true},authHeaders:{type:Object,default:()=>({})}},emits:["generated","unauthorized","busy"],setup(props,{emit}) {return useController(props,emit);}};
</script>
<template>
  <div class="v2-recipient" :aria-busy="busy">
    <p class="v2-recipient__eyebrow">Report per destinatario</p>
    <nav class="v2-recipient__tabs" aria-label="Destinatario del report"><button v-for="p in snapshot?.profiles || [{ id: 'executive', name: 'Dirigenza' }, { id: 'marketing', name: 'Marketing' }, { id: 'sponsor', name: 'Sponsor' }, { id: 'analytics', name: 'Analitico' }]" :key="p.id" type="button" :aria-pressed="recipient === p.id" :disabled="busy" @click="recipient = p.id">{{ p.name }}</button></nav>
    <div v-if="recipient === 'sponsor'" class="v2-recipient__options"><label>Ambito sponsor<select v-model="sponsorMode" :disabled="busy" @change="changeSponsor"><option value="aggregate">Sponsor aggregati dell’evento</option><option value="dedicated">Singolo sponsor</option></select></label><label v-if="sponsorMode === 'dedicated'">Partner<select v-model="sponsorID" :disabled="busy" @change="changeSponsor"><option :value="0">Seleziona un partner</option><option v-for="s in sponsors" :key="s.id" :value="s.id">{{ s.report_name || s.name }}</option></select></label></div>
    <p v-if="error" role="alert" class="v2-recipient__error">{{ error }} <button type="button" :disabled="busy" @click="load">Ricarica</button></p>
    <p v-if="loading" role="status">Calcolo dei dati e confronto con la precedente partita conclusa…</p>
    <template v-if="snapshot && !loading">
      <h3>{{ profile?.name }}</h3><p>{{ profile?.description }}</p>
      <div class="v2-recipient__options"><label><input v-model="compare" type="checkbox" :disabled="busy" /> Confronta con partita precedente</label><label><input v-model="includeAI" type="checkbox" :disabled="busy" /> Includi analisi AI / sintesi</label><label><input v-model="appendix" type="checkbox" :disabled="busy || !profile?.allow_appendix" /> Includi appendice completa (20 KPI)</label></div>
      <p class="v2-recipient__caption">Sezioni PDF: {{ profile?.id === 'analytics' ? '20 indicatori, valori, denominatori e metodologia' : `${kpis.length} indicatori, confronto e sintesi` }}{{ appendix && profile?.allow_appendix ? ' + appendice dei 20 KPI' : '' }}. {{ profile?.id === 'analytics' ? 'Il corpo comprende già l’appendice.' : '' }}</p>
      <p v-if="compare" class="v2-recipient__caption">{{ snapshot.previous_event ? `Partita precedente: ${snapshot.previous_event.title} · ${snapshot.previous_event.start_datetime}` : 'Nessuna partita precedente conclusa disponibile.' }} La direzione della variazione non indica automaticamente un miglioramento.</p>
      <p v-for="conflict in snapshot.conflicts || []" :key="conflict" role="alert" class="v2-recipient__error">{{ conflict }}</p>
      <p v-if="missing.length" class="v2-recipient__notice">{{ missing.length }} indicatori N/D per dati insufficienti.</p>
      <p v-if="sourceMessage" role="status">{{ sourceMessage }}</p>
      <button v-if="selectedSource && !sourceOpen" type="button" :disabled="busy" @click="openSources">{{ attendance ? 'Modifica presenze ufficiali' : 'Registra presenze ufficiali' }}</button>
      <section v-if="sourceOpen" class="v2-recipient__source" aria-label="Presenze ufficiali">
        <h3>Presenze ufficiali</h3>
        <p>Il numero di spettatori viene salvato per la partita e usato come denominatore del KPI di adozione.</p>
        <form @submit.prevent="saveSources"><div class="v2-recipient__source-fields"><label>Spettatori presenti<input v-model="sourceForm.value" type="number" :min="selectedSource?.min" :max="selectedSource?.max" step="1" required :disabled="busy" /></label><label>Fonte del dato<input v-model="sourceForm.source_description" type="text" minlength="3" maxlength="500" required placeholder="Es. distinta ufficiale / sistema biglietteria, riferimento" :disabled="busy" /></label><label>Nota facoltativa<textarea v-model="sourceForm.note" maxlength="1000" :disabled="busy"></textarea></label></div><label><input v-model="sourceForm.verified" type="checkbox" :disabled="busy" /> Ho verificato la fonte (altrimenti il report indica NON verificata)</label><label><input v-model="sourceForm.confirmed" type="checkbox" :disabled="busy" /> Confermo il conteggio per questa partita</label><ul v-if="sourceErrors.length || sourceForm.confirmed && sourceValidation.length" role="alert"><li v-for="message in sourceValidation" :key="message">{{ message }}</li></ul><div class="v2-recipient__actions"><button type="submit" :disabled="busy">Salva e aggiorna</button><button type="button" :disabled="busy" @click="sourceOpen = false">Annulla</button><button type="button" :disabled="busy" @click="showHistory">Storico modifiche</button></div></form>
        <ol v-if="history"><li v-for="(change, i) in history" :key="i">{{ historyFieldLabel(change.field_key) }}{{ change.sponsor_id ? ` · Sponsor #${change.sponsor_id}` : '' }} · {{ change.changed_at }} · amministratore {{ change.changed_by }} · {{ change.previous?.value ?? 'N/D' }} → {{ change.current.value }} · {{ change.current.source_description }} · {{ change.current.verification_status }}</li></ol>
      </section>
      <section v-for="group in groups" :key="group.name"><h3>{{ group.name }}</h3><div class="v2-recipient__grid"><ReportKpiCard v-for="kpi in group.kpis" :key="kpi.id" :kpi="kpi" :compare="compare" :event-context="!!snapshot.sponsor_id && kpi.id === 'visitors'" /></div></section>
      <section class="v2-recipient__ai">
        <h3>Modifica e approva l’analisi</h3>
        <div class="v2-recipient__actions"><button type="button" :disabled="busy" @click="generateAnalysis(false)">{{ aiLoading ? 'Analisi in corso…' : 'Genera analisi' }}</button><button type="button" :disabled="busy" @click="generateAnalysis(true)">Rigenera analisi</button></div>
        <p class="v2-recipient__caption">Genera e Rigenera sono gli unici pulsanti che possono chiamare OpenAI. La generazione prepara una bozza da rivedere e approvare; rigenerare sostituisce la bozza con un nuovo testo da approvare. Puoi anche modificare e approvare la sintesi locale senza usare l’AI.</p>
        <p v-if="reviewLoading" role="status">Caricamento bozza e revisione…</p>
        <p v-if="reviewMessage" role="status">{{ reviewMessage }}</p>
        <ReportAnalysisView v-if="analysis" :analysis="analysis.analysis" :status="analysis.status" :kpis="kpis" :review="analysis.review" :dirty="analysis.dirty" :busy="busy" :warnings="analysis.warnings" @update:analysis="editAnalysis" @save="reviewAnalysis('draft')" @approve="reviewAnalysis('approve')" @discard="reviewAnalysis('discard')" />
        <p v-if="includeAI && !analysisApproved" class="v2-recipient__notice">Per includere l’analisi nel PDF, approva la bozza corrente. Puoi esportare senza analisi togliendo la spunta a «Includi analisi AI / sintesi».</p>
        <button type="button" :disabled="busy" @click="loadAnalysisReview">Ricarica bozza salvata</button>
      </section>
      <details><summary>Limiti e metodologia</summary><ul><li v-for="(note, i) in snapshot.details?.coverage_notes || []" :key="i">{{ note }}</li></ul><p>I KPI usano i dati salvati per la partita. Le presenze ufficiali riportano lo stato di verifica della fonte.</p></details>
      <label class="v2-recipient__notes">Note del report<textarea v-model="notes" maxlength="4000" rows="2" :disabled="busy"></textarea></label>
      <div class="v2-recipient__actions"><button type="button" :disabled="busy || includeAI && !analysisApproved" @click="renderReport(false)">Anteprima PDF</button><button type="button" class="v2-recipient__primary" :disabled="busy || includeAI && !analysisApproved" @click="renderReport(true)">{{ working ? 'Generazione…' : 'Esporta PDF personalizzato' }}</button><button type="button" :disabled="busy" @click="load">Aggiorna sorgenti e confronto</button></div>
      <p class="v2-recipient__caption">Anteprima ed esportazione salvano uno snapshot versionato. La versione approvata resta riproducibile anche se le sorgenti cambiano successivamente.</p>
      <details @toggle="event => { if (event.target.open && versions === null) loadVersions(); }"><summary>Archivio versioni approvate</summary><p v-if="versions?.length === 0">Nessuna versione approvata per questa partita.</p><ul v-if="versions"><li v-for="version in versions" :key="version.id"><button type="button" :disabled="busy" @click="downloadVersion(version.id)">{{ version.recipient }}{{ version.sponsor ? ` · ${version.sponsor}` : '' }} · {{ version.approved_at }}</button></li></ul><button v-if="versions?.length && versions.length % 50 === 0" type="button" @click="loadVersions(true)">Carica versioni precedenti</button></details>
      <iframe v-if="previewUrl" :src="previewUrl" title="Anteprima del PDF approvato" class="v2-recipient__preview"></iframe>
    </template>
  </div>
</template>

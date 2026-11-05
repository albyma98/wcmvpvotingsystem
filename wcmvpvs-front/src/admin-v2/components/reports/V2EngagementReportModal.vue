<script>
import '../../styles/report.css';
import { useController } from '../../../components/EngagementReportModal.vue?admin-v2-controller';
import RecipientReportBuilder from './V2RecipientReportBuilder.vue';
export default {name:'V2EngagementReportModal',components:{RecipientReportBuilder},props:{modelValue:Boolean,eventId:{type:Number,default:0},eventTitle:{type:String,default:''},authHeaders:{type:Object,default:()=>({})}},emits:["update:modelValue","generated","unauthorized"],setup(props,{emit}) {return useController(props,emit);}};
</script>
<template>
  <Teleport to="body">
    <div v-if="modelValue" class="admin-v2 v2-report-backdrop" @click.self="close">
      <section ref="panel" class="v2-report" role="dialog" aria-modal="true" aria-labelledby="v2-report-title" tabindex="-1" :aria-busy="loading || saving">
        <header class="v2-report__header">
          <div><p class="v2-report__eyebrow">ARCHIVIO EVENTI / REPORT</p><h2 id="v2-report-title">Prepara il report della partita</h2><p class="v2-report__event">{{ snapshot?.event?.title || eventTitle }}</p></div>
          <button class="v2-report__close" type="button" aria-label="Chiudi report" :disabled="saving || recipientBusy" @click="close">×</button>
        </header>

        <div class="v2-report__body">
          <p v-if="loading" role="status" class="v2-report__loading">Preparazione dei dati della partita…</p>
          <div v-if="error" class="v2-report__error" role="alert">{{ error }}<button v-if="!snapshot && !loading" type="button" class="v2-report__link" @click="loadReport">Riprova</button></div>
          <div v-if="snapshot?.profiles?.length && !loading" class="v2-report__previous"><span>{{ legacyMode ? 'Revisione libera' : 'Report per destinatario' }}</span><button type="button" class="v2-report__link" :disabled="saving || recipientBusy" @click="legacyMode = !legacyMode">{{ legacyMode ? 'Report per destinatario' : 'Personalizza tutti i KPI' }}</button></div>
          <RecipientReportBuilder v-if="snapshot?.profiles?.length && !loading && !legacyMode" :event-id="eventId" :auth-headers="authHeaders" @generated="emit('generated', $event)" @unauthorized="emit('unauthorized')" @busy="recipientBusy = $event" />
          <template v-if="snapshot && !loading && (!snapshot.profiles?.length || legacyMode)">
            <div class="v2-report__intro"><div><strong>Una fotografia verificabile dell’evento</strong><p>Il report riconcilia la misurazione dedicata alla home /new con voti, giochi, premi, wallet e feedback salvati per l’evento. Le tracce operative ricostruiscono una partecipazione minima verificabile; non misurano tutte le visite. N/D indica una misura assente o un pubblico di riferimento non disponibile.</p></div><span class="v2-report__badge">{{ kpis.length }} KPI disponibili</span></div>
            <p class="v2-report__meta">Calcolato il {{ dateLabel(snapshot.calculated_at) }}<template v-if="snapshot.measurement_started_at"> · Primo dato disponibile: {{ dateLabel(snapshot.measurement_started_at) }}</template></p>
            <div v-if="snapshot.latest_report" class="v2-report__previous"><span>Ultimo report approvato: {{ dateLabel(snapshot.latest_report.approved_at) }}</span><button type="button" class="v2-report__link" :disabled="saving" @click="downloadPrevious">Scarica versione approvata</button></div>

            <section v-if="coverageNotes.length || operationalItems.length || sourceTables.length" class="v2-report__coverage" aria-labelledby="v2-report-coverage-title">
              <h3 id="v2-report-coverage-title">Copertura e dati operativi</h3>
              <ul v-if="coverageNotes.length" class="v2-report__coverage-notes"><li v-for="(note, index) in coverageNotes" :key="index">{{ note }}</li></ul>
              <dl v-if="operationalItems.length" class="v2-report__operational"><div v-for="item in operationalItems" :key="item.key"><dt>{{ item.label }}</dt><dd>{{ formatReportValue(item.value) }}</dd></div></dl>
              <p v-if="operationalItems.length" class="v2-report__coverage-caption">I totali operativi descrivono azioni registrate: premi, risposte e clic non equivalgono a persone uniche.</p>
              <details v-if="sourceTables.length" class="v2-report__sources"><summary>Tabelle e righe consultate</summary><ul><li v-for="source in sourceTables" :key="source.table"><code>{{ source.table }}</code><span>{{ formatReportValue(source.rows) }} righe</span></li></ul></details>
            </section>

            <section v-if="timeEstimate" class="v2-report__coverage" aria-labelledby="v2-report-time-title">
              <h3 id="v2-report-time-title">Tempo sulla piattaforma: stima e copertura</h3>
              <p class="v2-report__time-method">{{ timeEstimate.method }}</p>
              <p class="v2-report__time-coverage">Stima ricostruibile per <strong>{{ formatReportValue(timeEstimate.covered_visitors) }}</strong> utenti su {{ formatReportValue(timeEstimate.total_visitors) }} osservabili, ospiti e registrati. Esclusi dal calcolo: {{ formatReportValue(timeEstimate.excluded_visitors) }}.</p>
              <p class="v2-report__coverage-caption">Il tempo stimato unisce intervalli misurati e intervalli ricostruiti tra azioni vicine. Una singola azione senza un intervallo valido non genera secondi stimati; nessun tempo viene aggiunto prima della prima azione o dopo l’ultima.</p>
            </section>

            <section class="v2-report__selection" aria-labelledby="v2-report-selection-title">
              <div><h3 id="v2-report-selection-title">KPI da inserire nel PDF</h3><p aria-live="polite">{{ selectedKpiIds.length }} di {{ kpis.length }} selezionati · {{ format === 'extended' ? 'massimo 30 nel report completo' : 'massimo 20 nel one pager' }}</p></div>
              <div class="v2-report__selection-actions"><button type="button" class="v2-report__link" :disabled="saving" @click="selectKpis('all')">Seleziona tutti</button><button type="button" class="v2-report__link" :disabled="saving" @click="selectKpis('available')">Solo KPI con dati</button><button type="button" class="v2-report__link" :disabled="saving" @click="selectKpis('none')">Deseleziona tutti</button></div>
              <p v-for="message in selectionErrors" :key="message" class="v2-report__selection-warning" role="status">{{ message }}</p>
            </section>

            <section v-for="group in groups" :key="group.name" class="v2-report__group">
              <h3>{{ group.name }}</h3>
              <div class="v2-report__table-head" aria-hidden="true"><span>Indicatore e fonte</span><span>Rilevato</span><span>Valore report</span><span>Revisione</span></div>
              <article v-for="kpi in group.kpis" :key="kpi.id" class="v2-report__kpi" :class="{ 'is-corrected': kpi.corrected }">
                <div class="v2-report__kpi-name"><strong>{{ kpi.label }}</strong><label class="v2-report__include"><input v-model="selectedKpiIds" :value="kpi.id" :disabled="saving" type="checkbox" :aria-label="`Includi nel PDF: ${kpi.label}`" /> Includi nel PDF</label><details><summary>Definizione e fonte</summary><p>{{ kpi.description || formulaLabel(kpi) }}</p><p>Fonte: {{ sourceLabel(kpi.source) }}<template v-if="kpi.note"> · {{ kpi.note }}</template></p></details></div>
                <div class="v2-report__measured"><span class="v2-report__mobile-label">Rilevato</span>{{ formatReportValue(kpi.measured_value, kpi.unit) }}</div>
                <div class="v2-report__reviewed"><span class="v2-report__mobile-label">Valore report</span><strong>{{ formatReportValue(kpi.value, kpi.unit) }}</strong><small v-if="kpi.denominator !== undefined && kpi.numerator !== null && kpi.denominator !== null">{{ formatReportValue(kpi.numerator) }} / {{ formatReportValue(kpi.denominator) }}</small><small v-if="kpi.corrected">Corretto in revisione</small></div>
                <div class="v2-report__override"><button type="button" class="v2-report__link" :disabled="saving" @click="toggleOverride(kpi)">{{ overrides[kpi.id] ? 'Ripristina calcolo' : 'Modifica KPI' }}</button></div>
                <div v-if="overrides[kpi.id]" class="v2-report__override-fields"><label>Valore approvato (vuoto = N/D)<input v-model="overrides[kpi.id].value" :disabled="saving" type="number" min="0" :max="kpi.unit === 'percent' ? 100 : undefined" :step="kpi.unit === 'count' ? 1 : 'any'" :aria-label="`Valore approvato: ${kpi.label}`" /></label></div>
                <label class="v2-report__description">Breve descrizione (facoltativa)<input v-model="descriptions[kpi.id]" :disabled="saving" type="text" maxlength="120" :aria-label="`Breve descrizione: ${kpi.label}`" placeholder="Aggiungi una breve descrizione per il report" /><small>{{ descriptions[kpi.id]?.length || 0 }}/120</small></label>
              </article>
            </section>

            <details class="v2-report__facts"><summary>Verifica o modifica i dati di base <span>Le percentuali si ricalcolano automaticamente</span></summary><div class="v2-report__facts-grid"><label v-for="key in factKeys" :key="key">{{ factLabel(key) }}<input v-model="facts[key]" :disabled="saving" type="number" min="0" step="1" placeholder="N/D" /><small>Rilevato: {{ formatReportValue(snapshot.facts?.[key]) }}</small></label></div></details>
            <label class="v2-report__notes">Note per il report (facoltative)<textarea v-model="notes" :disabled="saving" rows="3" maxlength="4000" placeholder="Contesto dell’evento ed eventuali osservazioni…"></textarea><small>Le note complete restano archiviate nella versione approvata. Il PDF può abbreviarle per rispettare il formato scelto.</small></label>

            <fieldset class="v2-report__formats" :disabled="saving"><legend>Formato del PDF</legend><label :class="{ selected: format === 'one_page' }"><input v-model="format" type="radio" value="one_page" /><span><strong>One pager</strong><small>1 pagina · fino a 20 KPI selezionati</small></span></label><label :class="{ selected: format === 'extended' }"><input v-model="format" type="radio" value="extended" /><span><strong>Report completo</strong><small>3 pagine · fino a 30 KPI, giochi e partner</small></span></label></fieldset>
            <div v-if="validationErrors.length" class="v2-report__error" role="alert"><strong>Controlla i dati prima di confermare:</strong><ul><li v-for="item in validationErrors" :key="item">{{ item }}</li></ul></div>
            <p class="v2-report__signature">Il report è firmato con il solo logo dell’emittente. Dati originali, correzioni e note vengono conservati con un identificativo di approvazione.</p>
          </template>
        </div>

        <footer v-if="!snapshot?.profiles?.length || legacyMode" class="v2-report__footer"><button type="button" class="v2-report__button v2-report__button--light" :disabled="saving || loading || !snapshot" @click="resetReview">Ripristina dati rilevati</button><div><button type="button" class="v2-report__button v2-report__button--light" :disabled="saving" @click="close">Chiudi</button><button type="button" class="v2-report__button v2-report__button--primary" :disabled="loading || saving || !snapshot" @click="approveAndDownload">{{ saving ? 'Generazione in corso…' : approved ? 'Scarica PDF approvato' : 'Conferma e genera PDF' }}</button></div></footer>
      </section>
    </div>
  </Teleport>
</template>

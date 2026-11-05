<template>
  <div class="marketing">
    <div class="tabs" role="group" aria-label="Sezioni marketing">
      <button v-for="t in tabs" :key="t.id" type="button" class="btn outline" :class="{ active: tab === t.id }" :aria-pressed="tab === t.id" @click="tab = t.id">
        {{ t.label }}
      </button>
    </div>

    <section v-if="tab === 'audience'" class="card">
      <BaseSearchInput v-model="q" placeholder="Cerca nickname o telefono" />
      <BaseTable>
        <table class="marketing-table">
          <thead>
            <tr><th>Nickname</th><th>Telefono</th><th>Creato</th><th>Last seen</th><th>Coins</th><th>Consenso marketing</th></tr>
          </thead>
          <tbody>
            <tr v-for="a in audience" :key="a.fan_id" @click="selectedFan = a">
              <td><button type="button" class="audience-open" :aria-label="`Apri dettagli di ${a.nickname}`" @click.stop="selectedFan = a">{{ a.nickname }}</button></td>
              <td>{{ maskPhone(a.phone) }}</td>
              <td>{{ a.created_at }}</td>
              <td>{{ a.last_seen_at || '-' }}</td>
              <td>{{ a.coins }}</td>
              <td><span :class="['badge', a.marketing_consent ? 'ok' : 'no']">{{ a.marketing_consent ? 'Presente' : 'Assente' }}</span></td>
            </tr>
          </tbody>
        </table>
      </BaseTable>
      <aside v-if="selectedFan" class="drawer">
        <div class="drawer-heading"><h4>{{ selectedFan.nickname }}</h4><button class="btn outline" type="button" aria-label="Chiudi dettaglio fan" @click="selectedFan = null">Chiudi</button></div>
        <p>{{ selectedFan.phone }}</p>
        <BaseTextarea><textarea v-model="singleMessage" rows="4" aria-label="Messaggio per il fan" placeholder="Scrivi il tuo messaggio…" /></BaseTextarea>
        <BaseButton :disabled="!selectedFan.marketing_consent" @click="sendSingle">Invia SMS</BaseButton>
      </aside>
    </section>

    <section v-else-if="tab === 'campaigns'" class="card">
      <p class="billing-line">Costo per SMS: <strong>€ {{ billing.sms_cost.toFixed(2) }}</strong></p>
      <p class="billing-line">SMS gratuiti disponibili: <strong>{{ billing.free_sms_remaining }}</strong></p>
      <label class="marketing-field">Nome campagna<BaseInput><input v-model="campaign.name" placeholder="Nome campagna" /></BaseInput></label>
      <label class="marketing-field">Messaggio<BaseTextarea><textarea v-model="campaign.message" rows="4" placeholder="Messaggio" /></BaseTextarea></label>
      <div class="campaign-filters">
        <p class="filters-title">Filtri audience (cumulabili)</p>
        <label><input v-model="campaignFilters.male" type="checkbox" /> Uomo</label>
        <label><input v-model="campaignFilters.female" type="checkbox" /> Donna</label>
        <label><input v-model="campaignFilters.newUsers" type="checkbox" /> Nuovi utenti (creazione = ultimo evento)</label>
        <label><input v-model="campaignFilters.nonZeroCoins" type="checkbox" /> Saldo monete ≠ 0</label>
        <label><input v-model="campaignFilters.topCoins" type="checkbox" /> Più monete di tutti</label>
      </div>
      <p>Utenti inclusi automaticamente: <strong>{{ selectedCampaignAudience.length }}</strong></p>
      <div class="selected-audience-list">
        <p v-if="selectedCampaignAudience.length === 0" class="empty-list">Nessun utente selezionato con i filtri correnti.</p>
        <ul v-else>
          <li v-for="fan in selectedCampaignAudience" :key="fan.fan_id">
            {{ fan.nickname }} · {{ maskPhone(fan.phone) }} · {{ fan.coins }} monete
          </li>
        </ul>
      </div>
      <p>Stima costo: € {{ estimatedCost.toFixed(2) }}</p>
      <div class="actions"><BaseButton @click="createCampaign">Programma</BaseButton><BaseButton @click="sendNow">Invia ora</BaseButton><BaseButton @click="sendTest">Invia test</BaseButton></div>
    </section>

    <section v-else-if="tab === 'templates'" class="card">
      <label class="marketing-field">Nome template<BaseInput><input v-model="templateDraft.name" placeholder="Nome" /></BaseInput></label>
      <label class="marketing-field">Categoria<BaseSelect><select v-model="templateDraft.category"><option v-for="c in categories" :key="c" :value="c">{{ c }}</option></select></BaseSelect></label>
      <label class="marketing-field">Testo del messaggio<BaseTextarea><textarea v-model="templateDraft.body" rows="4" placeholder="Scrivi il testo del template…" /></BaseTextarea></label>
      <BaseButton @click="saveTemplate">Salva template</BaseButton>
      <ul><li v-for="t in templates" :key="t.id">{{ t.name }} <button class="btn danger" @click="removeTemplate(t.id)">Elimina</button></li></ul>
    </section>

    <section v-else class="card">
      <div class="billing-box">
        <p>Costo per SMS: <strong>€ {{ billing.sms_cost.toFixed(2) }}</strong></p>
        <p>SMS gratuiti rimasti: <strong>{{ billing.free_sms_remaining }}</strong></p>
        <p>Totale SMS: <strong>{{ billing.total_messages }}</strong></p>
        <p>Totale addebitato: <strong>€ {{ billing.total_cost_charged.toFixed(2) }}</strong></p>
      </div>
      <BaseTable>
        <table class="marketing-table">
          <thead>
            <tr><th>ID</th><th>Telefono</th><th>Status</th><th>Costo</th><th>Error</th><th>Creato</th></tr>
          </thead>
          <tbody>
            <tr v-for="l in logs" :key="l.id">
              <td>{{ l.id }}</td>
              <td>{{ maskPhone(l.phone) }}</td>
              <td>{{ l.status }}</td>
              <td>€ {{ Number(l.sms_cost_charged || 0).toFixed(2) }} <span v-if="l.used_free_sms" class="badge ok">free</span></td>
              <td>{{ l.error }}</td>
              <td>{{ l.created_at }}</td>
            </tr>
          </tbody>
        </table>
      </BaseTable>
    </section>
  </div>
</template>
<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { apiClient } from '../../../api';
import BaseButton from '../ui/BaseButton.vue'; import BaseInput from '../ui/BaseInput.vue'; import BaseSelect from '../ui/BaseSelect.vue'; import BaseSearchInput from '../ui/BaseSearchInput.vue'; import BaseTable from '../ui/BaseTable.vue'; import BaseTextarea from '../ui/BaseTextarea.vue';
const tab = ref('audience'); const tabs = [{ id: 'audience', label: 'Audience' }, { id: 'campaigns', label: 'SMS Campaigns' }, { id: 'templates', label: 'Templates' }, { id: 'logs', label: 'Logs' }];
const q = ref(''); const audience = ref([]); const selectedFan = ref(null); const singleMessage = ref(''); const campaign = ref({ id: 0, name: '', message: '', query: '', fan_ids: [] }); const templates = ref([]); const templateDraft = ref({ name: '', body: '', category: 'promo' }); const logs = ref([]); const categories = ['match', 'promo', 'premi'];
const campaignFilters = ref({ male: false, female: false, newUsers: false, nonZeroCoins: false, topCoins: false });
const billing = ref({ sms_cost: 0.08, free_sms_remaining: 0, total_messages: 0, total_cost_charged: 0 });
const headers = computed(() => ({}));
const maxCoinsInAudience = computed(() => audience.value.reduce((max, fan) => Math.max(max, Number(fan.coins || 0)), 0));
const selectedCampaignAudience = computed(() => {
  const enabledFilters = Object.values(campaignFilters.value).some(Boolean);
  return audience.value.filter((fan) => {
    if (!fan.marketing_consent) return false;
    if (!enabledFilters) return true;
    const gender = String(fan.gender || '').toUpperCase();
    if (campaignFilters.value.male || campaignFilters.value.female) {
      const maleOk = campaignFilters.value.male && gender === 'M';
      const femaleOk = campaignFilters.value.female && gender === 'F';
      if (!maleOk && !femaleOk) return false;
    }
    if (campaignFilters.value.newUsers && fan.created_at !== fan.last_seen_at) return false;
    if (campaignFilters.value.nonZeroCoins && Number(fan.coins || 0) === 0) return false;
    if (campaignFilters.value.topCoins && Number(fan.coins || 0) !== maxCoinsInAudience.value) return false;
    return true;
  });
});
const estimatedCost = computed(() => {
  const recipients = selectedCampaignAudience.value.length;
  const payable = Math.max(0, recipients - Number(billing.value.free_sms_remaining || 0));
  return payable * Number(billing.value.sms_cost || 0);
});
const maskPhone = (v) => (v ? `${v.slice(0, 4)}****${v.slice(-2)}` : '');
const loadAudience = async () => { const { data } = await apiClient.get('/admin/marketing/audience', { params: { q: q.value }, headers: headers.value }); audience.value = data || []; };
const loadTemplates = async () => { const { data } = await apiClient.get('/admin/marketing/templates', { headers: headers.value }); templates.value = data || []; };
const loadLogs = async () => { const { data } = await apiClient.get('/admin/marketing/logs', { headers: headers.value }); logs.value = data?.items || []; billing.value = data?.summary || billing.value; };
const loadBilling = async () => { const { data } = await apiClient.get('/admin/marketing/billing', { headers: headers.value }); billing.value = data || billing.value; };
const sendSingle = async () => { if (!selectedFan.value) return; await apiClient.post('/admin/marketing/sms/single', { fan_id: selectedFan.value.fan_id, message: singleMessage.value }, { headers: headers.value }); await loadLogs(); };
const createCampaign = async () => { const { data } = await apiClient.post('/admin/marketing/campaigns', campaign.value, { headers: headers.value }); campaign.value.id = data?.campaign?.id || 0; await loadLogs(); };
const sendNow = async () => { if (!campaign.value.id) return; await apiClient.post(`/admin/marketing/campaigns/${campaign.value.id}/send-now`, {}, { headers: headers.value }); await loadLogs(); };
const sendTest = async () => { if (!campaign.value.id) return; const phone = prompt('Numero test admin'); if (!phone) return; await apiClient.post(`/admin/marketing/campaigns/${campaign.value.id}/test`, { phone }, { headers: headers.value }); await loadLogs(); };
const saveTemplate = async () => { await apiClient.post('/admin/marketing/templates', templateDraft.value, { headers: headers.value }); templateDraft.value = { name: '', body: '', category: 'promo' }; await loadTemplates(); };
const removeTemplate = async (id) => { await apiClient.delete(`/admin/marketing/templates/${id}`, { headers: headers.value }); await loadTemplates(); };
watch(q, loadAudience);
watch(selectedCampaignAudience, (fans) => {
  campaign.value.fan_ids = fans.map((fan) => fan.fan_id);
}, { immediate: true });
onMounted(async () => { await Promise.all([loadAudience(), loadTemplates(), loadBilling(), loadLogs()]); });
</script>
<style scoped>
.marketing { display: grid; gap: 20px; min-width: 0; }
.marketing > .card { display: grid; gap: 16px; box-shadow: none; padding: 0; border: 0; }
.marketing-table { width: 100%; border-collapse: collapse; font-size: 13px; text-align: left; }
.marketing-table th, .marketing-table td { padding: 14px 16px; border-bottom: 1px solid var(--border); }
.marketing-table th { font-size: 11px; font-weight: 650; color: var(--text-secondary); background: var(--bg-card-hover); white-space: nowrap; }
.marketing-table tbody tr:hover { background: var(--bg-card-hover); }
.audience-open { color: var(--accent); font-weight: 600; text-align: left; min-height: 32px; }
.drawer { display: grid; gap: 14px; padding: 20px; border: 1px solid var(--border); border-radius: 12px; background: var(--bg-card-hover); }
.drawer-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.drawer-heading h4 { font-size: 16px; margin: 0; font-weight: 650; }
.drawer p { margin: 0; font-size: 13px; color: var(--text-secondary); }
.badge.ok { color: #166534; background: #ecfdf5; }
.badge.no { color: #991b1b; background: #fff1f2; }
.tabs { display: flex; gap: 6px; flex-wrap: wrap; padding: 5px; background: var(--bg-card-hover); border: 1px solid var(--border); border-radius: 10px; width: fit-content; }
.tabs .btn { color: var(--text-secondary); border: 1px solid transparent; background: transparent; font-size: 12px; min-height: 36px; box-shadow: none; }
.tabs .btn.active { background: #fff; color: var(--accent); border-color: var(--border); box-shadow: 0 1px 3px #1822380a; }
.billing-box { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; padding: 20px; border: 1px solid var(--border); border-radius: 12px; background: var(--bg-card-hover); font-size: 13px; }
.billing-box p { margin: 0; }
.billing-line { margin: 0; font-size: 13px; color: var(--text-secondary); }
.marketing-field { display: grid; gap: 8px; font-size: 13px; font-weight: 600; }
.campaign-filters { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; padding: 20px; margin: 0; border: 1px solid var(--border); border-radius: 12px; }
.campaign-filters label { display: flex; align-items: center; gap: 9px; font-size: 12px; }
.filters-title { font-weight: 650; font-size: 13px; margin: 0 0 5px; grid-column: 1 / -1; }
.selected-audience-list { max-height: 220px; overflow: auto; border: 1px solid var(--border); border-radius: 10px; padding: 12px 16px; font-size: 12px; line-height: 1.8; }
.selected-audience-list ul { margin: 0; padding-left: 16px; }
.empty-list { margin: 0; color: var(--text-muted); }
.actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
@media (max-width: 600px) { .billing-box, .campaign-filters { grid-template-columns: minmax(0, 1fr); } .tabs { width: 100%; } .tabs .btn { flex: 1 1 auto; } }
</style>

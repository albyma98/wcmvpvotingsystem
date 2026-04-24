<template>
  <div class="marketing">
    <div class="tabs">
      <button v-for="t in tabs" :key="t.id" class="btn outline" :class="{ active: tab === t.id }" @click="tab = t.id">
        {{ t.label }}
      </button>
    </div>

    <section v-if="tab === 'audience'" class="card">
      <BaseSearchInput v-model="q" placeholder="Cerca nickname o telefono" />
      <BaseTable>
        <table class="marketing-table">
          <thead>
            <tr><th>Nickname</th><th>Telefono</th><th>Creato</th><th>Last seen</th><th>Coins</th><th>Terms</th></tr>
          </thead>
          <tbody>
            <tr v-for="a in audience" :key="a.fan_id" @click="selectedFan = a">
              <td>{{ a.nickname }}</td>
              <td>{{ maskPhone(a.phone) }}</td>
              <td>{{ a.created_at }}</td>
              <td>{{ a.last_seen_at || '-' }}</td>
              <td>{{ a.coins }}</td>
              <td><span :class="['badge', a.accepted_terms ? 'ok' : 'no']">{{ a.accepted_terms ? 'accepted' : 'missing' }}</span></td>
            </tr>
          </tbody>
        </table>
      </BaseTable>
      <aside v-if="selectedFan" class="drawer">
        <h4>{{ selectedFan.nickname }}</h4>
        <p>{{ selectedFan.phone }}</p>
        <BaseTextarea><textarea v-model="singleMessage" rows="4" /></BaseTextarea>
        <BaseButton @click="sendSingle">Invia SMS</BaseButton>
      </aside>
    </section>

    <section v-else-if="tab === 'campaigns'" class="card">
      <p class="billing-line">Costo per SMS: <strong>€ {{ billing.sms_cost.toFixed(2) }}</strong></p>
      <p class="billing-line">SMS gratuiti disponibili: <strong>{{ billing.free_sms_remaining }}</strong></p>
      <BaseInput><input v-model="campaign.name" placeholder="Nome campagna" /></BaseInput>
      <BaseTextarea><textarea v-model="campaign.message" rows="4" placeholder="Messaggio" /></BaseTextarea>
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
      <BaseInput><input v-model="templateDraft.name" placeholder="Nome" /></BaseInput>
      <BaseSelect><select v-model="templateDraft.category"><option v-for="c in categories" :key="c" :value="c">{{ c }}</option></select></BaseSelect>
      <BaseTextarea><textarea v-model="templateDraft.body" rows="4" placeholder="Body" /></BaseTextarea>
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
    if (!fan.accepted_terms) return false;
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
<style scoped>.marketing-table{width:100%;border-collapse:collapse}.marketing-table th,.marketing-table td{padding:.5rem;border-bottom:1px solid #e2e8f0}.drawer{margin-top:1rem;padding:1rem;border:1px solid #e2e8f0;border-radius:.8rem}.badge.ok{color:#166534}.badge.no{color:#991b1b}.tabs{display:flex;gap:.5rem;margin-bottom:1rem}.tabs .active{background:#e2e8f0}.billing-box{margin-bottom:1rem;padding:.75rem;border:1px solid #e2e8f0;border-radius:.75rem}.billing-line{margin:.2rem 0}.campaign-filters{display:grid;grid-template-columns:1fr;gap:.35rem;margin:.75rem 0}.filters-title{font-weight:600;margin:0 0 .25rem}.selected-audience-list{max-height:220px;overflow:auto;border:1px solid #e2e8f0;border-radius:.75rem;padding:.5rem .75rem;margin-bottom:.75rem}.selected-audience-list ul{margin:0;padding-left:1rem}.empty-list{margin:0;color:#64748b}</style>

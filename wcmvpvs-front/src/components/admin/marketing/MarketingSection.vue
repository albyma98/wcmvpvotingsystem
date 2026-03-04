<template>
  <div class="marketing">
    <div class="tabs">
      <button v-for="t in tabs" :key="t.id" class="btn outline" :class="{ active: tab === t.id }" @click="tab = t.id">{{ t.label }}</button>
    </div>

    <section v-if="tab === 'audience'" class="card">
      <BaseSearchInput v-model="q" placeholder="Cerca nickname o telefono" />
      <BaseTable>
        <table class="marketing-table"><thead><tr><th>Nickname</th><th>Telefono</th><th>Creato</th><th>Last seen</th><th>Coins</th><th>Terms</th></tr></thead><tbody>
          <tr v-for="a in audience" :key="a.fan_id" @click="selectedFan = a"><td>{{ a.nickname }}</td><td>{{ maskPhone(a.phone) }}</td><td>{{ a.created_at }}</td><td>{{ a.last_seen_at || '-' }}</td><td>{{ a.coins }}</td><td><span :class="['badge', a.accepted_terms ? 'ok' : 'no']">{{ a.accepted_terms ? 'accepted' : 'missing' }}</span></td></tr>
        </tbody></table>
      </BaseTable>
      <aside v-if="selectedFan" class="drawer"><h4>{{ selectedFan.nickname }}</h4><p>{{ selectedFan.phone }}</p><BaseTextarea><textarea v-model="singleMessage" rows="4" /></BaseTextarea><BaseButton @click="sendSingle">Invia SMS</BaseButton></aside>
    </section>

    <section v-else-if="tab === 'campaigns'" class="card">
      <BaseInput><input v-model="campaign.name" placeholder="Nome campagna" /></BaseInput>
      <BaseTextarea><textarea v-model="campaign.message" rows="4" placeholder="Messaggio" /></BaseTextarea>
      <BaseInput><input v-model="campaign.query" placeholder="Filtro audience" /></BaseInput>
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
      <BaseTable><table class="marketing-table"><thead><tr><th>ID</th><th>Telefono</th><th>Status</th><th>Error</th><th>Creato</th></tr></thead><tbody><tr v-for="l in logs" :key="l.id"><td>{{ l.id }}</td><td>{{ maskPhone(l.phone) }}</td><td>{{ l.status }}</td><td>{{ l.error }}</td><td>{{ l.created_at }}</td></tr></tbody></table></BaseTable>
    </section>
  </div>
</template>
<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { apiClient } from '../../../api';
import BaseButton from '../ui/BaseButton.vue'; import BaseInput from '../ui/BaseInput.vue'; import BaseSelect from '../ui/BaseSelect.vue'; import BaseSearchInput from '../ui/BaseSearchInput.vue'; import BaseTable from '../ui/BaseTable.vue'; import BaseTextarea from '../ui/BaseTextarea.vue';
const tab = ref('audience'); const tabs = [{ id: 'audience', label: 'Audience' }, { id: 'campaigns', label: 'SMS Campaigns' }, { id: 'templates', label: 'Templates' }, { id: 'logs', label: 'Logs' }];
const q = ref(''); const audience = ref([]); const selectedFan = ref(null); const singleMessage = ref(''); const campaign = ref({ id: 0, name: '', message: '', query: '', fan_ids: [] }); const templates = ref([]); const templateDraft = ref({ name: '', body: '', category: 'promo' }); const logs = ref([]); const categories = ['match', 'promo', 'premi'];
const headers = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('adminToken') || ''}` }));
const estimatedCost = computed(() => audience.value.filter((a) => a.accepted_terms).length * 0.08);
const maskPhone = (v) => (v ? `${v.slice(0, 4)}****${v.slice(-2)}` : '');
const loadAudience = async () => { const { data } = await apiClient.get('/admin/marketing/audience', { params: { q: q.value }, headers: headers.value }); audience.value = data || []; };
const loadTemplates = async () => { const { data } = await apiClient.get('/admin/marketing/templates', { headers: headers.value }); templates.value = data || []; };
const loadLogs = async () => { const { data } = await apiClient.get('/admin/marketing/logs', { headers: headers.value }); logs.value = data || []; };
const sendSingle = async () => { if (!selectedFan.value) return; await apiClient.post('/admin/marketing/sms/single', { fan_id: selectedFan.value.fan_id, message: singleMessage.value }, { headers: headers.value }); await loadLogs(); };
const createCampaign = async () => { const { data } = await apiClient.post('/admin/marketing/campaigns', campaign.value, { headers: headers.value }); campaign.value.id = data?.campaign?.id || 0; await loadLogs(); };
const sendNow = async () => { if (!campaign.value.id) return; await apiClient.post(`/admin/marketing/campaigns/${campaign.value.id}/send-now`, {}, { headers: headers.value }); await loadLogs(); };
const sendTest = async () => { if (!campaign.value.id) return; const phone = prompt('Numero test admin'); if (!phone) return; await apiClient.post(`/admin/marketing/campaigns/${campaign.value.id}/test`, { phone }, { headers: headers.value }); await loadLogs(); };
const saveTemplate = async () => { await apiClient.post('/admin/marketing/templates', templateDraft.value, { headers: headers.value }); templateDraft.value = { name: '', body: '', category: 'promo' }; await loadTemplates(); };
const removeTemplate = async (id) => { await apiClient.delete(`/admin/marketing/templates/${id}`, { headers: headers.value }); await loadTemplates(); };
watch(q, loadAudience); onMounted(async () => { await Promise.all([loadAudience(), loadTemplates(), loadLogs()]); });
</script>
<style scoped>.marketing-table{width:100%;border-collapse:collapse}.marketing-table th,.marketing-table td{padding:.5rem;border-bottom:1px solid #e2e8f0}.drawer{margin-top:1rem;padding:1rem;border:1px solid #e2e8f0;border-radius:.8rem}.badge.ok{color:#166534}.badge.no{color:#991b1b}.tabs{display:flex;gap:.5rem;margin-bottom:1rem}.tabs .active{background:#e2e8f0}</style>

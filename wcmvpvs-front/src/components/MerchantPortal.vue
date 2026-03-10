<template>
  <div class="merchant-shell">
    <section v-if="!token" class="card auth-card">
      <h1>Portale BAR</h1>
      <p>Accesso merchant con conferma password super admin.</p>
      <form @submit.prevent="login" class="form-grid">
        <input v-model="form.password" type="password" placeholder="Password" required />
        <button class="btn primary" :disabled="loading">{{ loading ? 'Accesso…' : 'Entra' }}</button>
      </form>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <template v-else>
      <header class="merchant-header card">
        <div>
          <h2>{{ profile.displayName || profile.username }}</h2>
          <p>{{ profile.organizationSlug }}</p>
        </div>
        <button class="btn" @click="logout">Esci</button>
      </header>

      <nav class="tabs card">
        <button v-for="s in sections" :key="s.id" class="btn" :class="{primary: section===s.id}" @click="section=s.id">{{ s.label }}</button>
      </nav>

      <section v-if="section==='dashboard'" class="grid cards">
        <article v-for="k in summaryCards" :key="k.label" class="stat card"><h3>{{ k.value }}</h3><p>{{ k.label }}</p></article>
      </section>

      <section v-else-if="section==='live'" class="card">
        <h3>Ordini Live</h3>
        <div v-for="o in orders" :key="o.id" class="order-card">
          <div><strong>#{{ o.id }}</strong> • {{ fmtTime(o.created_at) }} • € {{ (o.total_cents/100).toFixed(2) }}</div>
          <div class="muted">{{ o.sector }} / {{ o.row }} / {{ o.seat }}</div>
          <div class="muted">{{ o.products }}</div>
          <div class="actions">
            <button v-for="st in statuses" :key="st" class="btn" @click="changeStatus(o.id,st)">{{ st }}</button>
          </div>
        </div>
      </section>

      <section v-else-if="section==='history'" class="card">
        <h3>Storico ordini</h3>
        <div class="filters"><input type="date" v-model="filters.from"/><input type="date" v-model="filters.to"/><select v-model="filters.status"><option value="all">Tutti</option><option v-for="st in statuses" :key="st">{{ st }}</option></select><button class="btn" @click="loadOrders">Filtra</button></div>
      </section>

      <section v-else-if="section==='menu'" class="card">
        <h3>Menu</h3>
        <form @submit.prevent="saveProduct" class="form-grid"><input v-model.trim="product.name" placeholder="Nome prodotto" required/><input v-model.number="product.price_cents" type="number" min="0" placeholder="Prezzo centesimi" required/><button class="btn primary">Salva</button></form>
        <div v-for="p in products" :key="p.id" class="order-card"><strong>{{ p.name }}</strong> - € {{ (p.price_cents/100).toFixed(2) }}
          <div class="actions"><button class="btn" @click="setFlags(p,!p.is_active,p.is_available)">{{ p.is_active?'Disattiva':'Attiva' }}</button><button class="btn" @click="setFlags(p,p.is_active,!p.is_available)">{{ p.is_available?'Esaurito':'Disponibile' }}</button></div>
        </div>
      </section>

      <section v-else-if="section==='availability'" class="card"><h3>Disponibilità rapida</h3><div class="actions"><button v-for="p in products" :key="p.id" class="btn" @click="setFlags(p,p.is_active,!p.is_available)">{{ p.name }} → {{ p.is_available?'Esaurito':'Disponibile' }}</button></div></section>
      <section v-else class="card"><h3>Impostazioni</h3><p>Partner: {{ profile.displayName }}</p><p>Società: {{ profile.organizationSlug }}</p></section>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { apiClient } from '../api';
const form = reactive({ password:'' });
const token = ref(localStorage.getItem('merchantToken')||'');
const profile = reactive({ username: localStorage.getItem('merchantUsername')||'', displayName: localStorage.getItem('merchantDisplayName')||'', organizationSlug: window.location.pathname.split('/').filter(Boolean)[0] || '' });
const loading=ref(false), error=ref(''), section=ref('dashboard');
const summary=ref({orders_received:0,orders_pending:0,orders_preparing:0,orders_ready:0,orders_completed:0,total_revenue_cents:0});
const orders=ref([]), products=ref([]);
const product=reactive({name:'',price_cents:0});
const filters=reactive({from:'',to:'',status:'all'});
const statuses=['new','in_preparazione','pronto','completato','annullato'];
const sections=[{id:'dashboard',label:'Dashboard'},{id:'live',label:'Ordini Live'},{id:'history',label:'Storico ordini'},{id:'menu',label:'Menu'},{id:'availability',label:'Disponibilità rapida'},{id:'settings',label:'Impostazioni'}];
const authHeaders=computed(()=>({headers:{Authorization:`Bearer ${token.value}`}}));
const summaryCards=computed(()=>[{label:'Ordini ricevuti',value:summary.value.orders_received},{label:'In attesa',value:summary.value.orders_pending},{label:'In preparazione',value:summary.value.orders_preparing},{label:'Pronti',value:summary.value.orders_ready},{label:'Completati',value:summary.value.orders_completed},{label:'Incasso totale',value:`€ ${(summary.value.total_revenue_cents/100).toFixed(2)}`}]);
const pathPrefix=()=>{const a=window.location.pathname.split('/').filter(Boolean); return a.length>1?`/${a[0]}`:''};
async function login(){loading.value=true;error.value='';try{const{data}=await apiClient.post(`${pathPrefix()}/merchant/login`,{password:form.password});token.value=data.token;profile.username=data.username;profile.displayName=data.display_name;localStorage.setItem('merchantToken',token.value);localStorage.setItem('merchantUsername',profile.username);localStorage.setItem('merchantDisplayName',profile.displayName);form.password='';await bootstrap();}catch(e){error.value='Password super admin non valida o merchant BAR non configurato.'}finally{loading.value=false;}}
function logout(){token.value='';localStorage.removeItem('merchantToken');localStorage.removeItem('merchantUsername');localStorage.removeItem('merchantDisplayName');}
async function loadSummary(){const {data}=await apiClient.get(`${pathPrefix()}/merchant/dashboard/summary`,authHeaders.value);summary.value=data||summary.value;}
async function loadOrders(){const {data}=await apiClient.get(`${pathPrefix()}/merchant/orders`,{...authHeaders.value,params:filters});orders.value=(data||[]).map(o=>({...o,products:o.products||o.products_json}));}
async function changeStatus(id,status){await apiClient.post(`${pathPrefix()}/merchant/orders/${id}/status`,{status},authHeaders.value);await loadOrders();await loadSummary();}
async function loadProducts(){const {data}=await apiClient.get(`${pathPrefix()}/merchant/products`,authHeaders.value);products.value=data||[];}
async function saveProduct(){await apiClient.post(`${pathPrefix()}/merchant/products`,{...product,is_active:true,is_available:true},authHeaders.value);product.name='';product.price_cents=0;await loadProducts();}
async function setFlags(p,is_active,is_available){await apiClient.post(`${pathPrefix()}/merchant/products/${p.id}/flags`,{is_active,is_available},authHeaders.value);await loadProducts();}
async function bootstrap(){await Promise.all([loadSummary(),loadOrders(),loadProducts()]);}
function fmtTime(v){return new Date(v).toLocaleTimeString('it-IT',{hour:'2-digit',minute:'2-digit'});} 
onMounted(()=>{ if(token.value) bootstrap(); setInterval(()=>token.value&&loadOrders(),7000); });
</script>

<style scoped>
.merchant-shell{max-width:1100px;margin:0 auto;padding:1rem;display:grid;gap:1rem}.card{background:#fff;border-radius:14px;padding:1rem}.tabs,.actions{display:flex;gap:.5rem;flex-wrap:wrap}.btn{padding:.8rem 1rem;border-radius:10px;border:1px solid #d1d5db;background:#fff}.primary{background:#111827;color:#fff}.form-grid{display:grid;gap:.75rem}.grid.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:.75rem}.order-card{border:1px solid #e5e7eb;border-radius:12px;padding:.75rem;margin:.5rem 0}.muted{color:#6b7280}.error{color:#b91c1c}
</style>

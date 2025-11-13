<template>
  <div class="master-portal">
    <header class="master-header">
      <div>
        <p class="eyebrow">Portale master</p>
        <h1>Controllo centrale</h1>
        <p class="subtitle">Monitora le società e lo stato dei voti in tutta la piattaforma.</p>
      </div>
      <div class="header-actions" v-if="isSuperAdmin">
        <a class="btn ghost" href="/admin" title="Vai al pannello società">Portale società</a>
        <button class="btn outline" type="button" @click="logout">Esci</button>
      </div>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
      <h2>Accedi come super admin</h2>
      <form class="form-grid" @submit.prevent="login">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button class="btn primary" type="submit" :disabled="isLoggingIn">
          {{ isLoggingIn ? "Accesso in corso…" : "Entra" }}
        </button>
      </form>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </section>

    <section v-else-if="!isSuperAdmin" class="card warning-card">
      <h2>Accesso limitato</h2>
      <p>Solo gli utenti con ruolo <strong>superadmin</strong> possono accedere al portale master.</p>
      <div class="warning-actions">
        <a class="btn ghost" href="/admin">Vai al portale società</a>
        <button class="btn outline" type="button" @click="logout">Esci</button>
      </div>
    </section>

    <section v-else class="master-shell">
      <nav class="master-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['nav-btn', { active: activeSection === tab.id }]"
          type="button"
          @click="switchSection(tab.id)"
        >
          {{ tab.label }}
        </button>
        <button
          v-if="selectedOrganizationId && activeSection === 'organization-detail'"
          class="nav-btn"
          type="button"
          @click="switchSection('organizations')"
        >
          Torna alla lista
        </button>
      </nav>

      <div class="master-content">
        <div v-if="activeSection === 'dashboard'" class="grid-cards">
          <article class="stat-card" aria-live="polite">
            <p class="label">Società registrate</p>
            <p class="value">{{ summary.total_organizations ?? 0 }}</p>
            <button class="btn ghost" type="button" @click="fetchSummary" :disabled="isLoadingSummary">
              {{ isLoadingSummary ? 'Aggiornamento…' : 'Aggiorna' }}
            </button>
          </article>
          <article class="stat-card">
            <p class="label">Totale voti</p>
            <p class="value">{{ summary.total_votes ?? 0 }}</p>
            <small>Storico complessivo</small>
          </article>
          <article class="stat-card">
            <p class="label">Voti ultimi 7 giorni</p>
            <p class="value">{{ summary.votes_last_7_days ?? 0 }}</p>
            <small>Monitoraggio attività recente</small>
          </article>
          <article class="stat-card">
            <p class="label">Totale partite</p>
            <p class="value">{{ summary.total_events ?? 0 }}</p>
            <small>Eventi registrati nel sistema</small>
          </article>
        </div>

        <div v-else-if="activeSection === 'organizations'" class="organizations-view">
          <header class="section-header">
            <div>
              <h2>Società</h2>
              <p>Gestisci anagrafiche e stato delle società.</p>
            </div>
            <div class="section-actions">
              <button class="btn outline" type="button" @click="fetchOrganizations" :disabled="isLoadingOrganizations">
                {{ isLoadingOrganizations ? 'Aggiornamento…' : 'Aggiorna elenco' }}
              </button>
              <button class="btn primary" type="button" @click="openCreateOrganization">
                Nuova società
              </button>
            </div>
          </header>

          <div v-if="organizationFormVisible" class="card form-card">
            <header>
              <h3>{{ organizationFormMode === 'create' ? 'Crea società' : 'Modifica società' }}</h3>
              <button class="btn ghost" type="button" @click="closeOrganizationForm">Chiudi</button>
            </header>
            <form class="form-grid" @submit.prevent="submitOrganizationForm">
              <label>
                Nome
                <input v-model.trim="organizationForm.name" type="text" required />
              </label>
              <label>
                Slug / URL pubblico
                <input
                  v-model.trim="organizationForm.slug"
                  type="text"
                  required
                  placeholder="es. volley-milano o https://mia-societa.it"
                />
                <small class="help-text">Puoi inserire uno slug (verrà normalizzato) oppure un URL completo.</small>
              </label>
              <label>
                Città / Descrizione
                <input v-model.trim="organizationForm.city" type="text" placeholder="Es. Milano" />
              </label>
              <label>
                Logo (URL)
                <input v-model.trim="organizationForm.logo_url" type="url" placeholder="https://…" />
              </label>
              <label class="switch-field">
                <input type="checkbox" v-model="organizationForm.is_active" />
                <span>Società attiva</span>
              </label>
              <div class="form-actions">
                <button class="btn outline" type="button" @click="closeOrganizationForm">Annulla</button>
                <button class="btn primary" type="submit" :disabled="isSavingOrganization">
                  {{ isSavingOrganization ? 'Salvataggio…' : 'Salva' }}
                </button>
              </div>
              <p v-if="organizationFormError" class="error">{{ organizationFormError }}</p>
            </form>
          </div>

          <div class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Società</th>
                  <th>Slug / URL</th>
                  <th>Città</th>
                  <th>Stato</th>
                  <th>Creata il</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="org in organizations" :key="org.id">
                  <td>
                    <div class="org-cell">
                      <img v-if="org.logo_url" :src="org.logo_url" :alt="`Logo ${org.name}`" />
                      <div>
                        <p class="org-name">{{ org.name }}</p>
                        <small>ID {{ org.id }}</small>
                      </div>
                    </div>
                  </td>
                  <td>
                    <div class="slug-cell">
                      <a
                        v-if="org.slug"
                        :href="resolvePublicLink(org.slug)"
                        class="slug-link"
                        target="_blank"
                        rel="noreferrer"
                      >
                        {{ org.slug }}
                      </a>
                      <span v-else class="muted">—</span>
                    </div>
                  </td>
                  <td>{{ org.city || '—' }}</td>
                  <td>
                    <span :class="['status-pill', org.is_active ? 'active' : 'inactive']">
                      {{ org.is_active ? 'Attiva' : 'Disattiva' }}
                    </span>
                  </td>
                  <td>{{ formatDate(org.created_at) }}</td>
                  <td class="actions">
                    <button class="btn ghost" type="button" @click="viewOrganization(org.id)">
                      Dettagli
                    </button>
                    <button class="btn outline" type="button" @click="openEditOrganization(org)">
                      Modifica
                    </button>
                  </td>
                </tr>
                <tr v-if="!organizations.length && !isLoadingOrganizations">
                  <td colspan="4" class="empty">Nessuna società registrata.</td>
                </tr>
                <tr v-if="isLoadingOrganizations">
                  <td colspan="4" class="empty">Caricamento in corso…</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-else-if="activeSection === 'organization-detail'" class="detail-view">
          <div v-if="organizationDetail">
            <header class="section-header">
              <div>
                <h2>{{ organizationDetail.organization.name }}</h2>
                <p>{{ organizationDetail.organization.city || 'Nessuna descrizione disponibile' }}</p>
              </div>
              <div class="section-actions">
                <a
                  v-if="organizationDetail.organization.slug"
                  :href="resolvePublicLink(organizationDetail.organization.slug)"
                  class="btn outline"
                  target="_blank"
                  rel="noreferrer"
                >
                  Pagina pubblica
                </a>
                <a :href="resolveSocietyLink(organizationDetail.organization.id)" class="btn ghost" target="_blank">
                  Apri pannello società
                </a>
                <button class="btn outline" type="button" @click="switchSection('organizations')">Torna alla lista</button>
              </div>
            </header>

            <div class="detail-grid">
              <article class="card info-card">
                <div class="logo-preview" v-if="organizationDetail.organization.logo_url">
                  <img :src="organizationDetail.organization.logo_url" :alt="organizationDetail.organization.name" />
                </div>
                <dl>
                  <div>
                    <dt>ID</dt>
                    <dd>{{ organizationDetail.organization.id }}</dd>
                  </div>
                  <div>
                    <dt>Slug / URL pubblico</dt>
                    <dd>
                      <a
                        v-if="organizationDetail.organization.slug"
                        :href="resolvePublicLink(organizationDetail.organization.slug)"
                        class="slug-link"
                        target="_blank"
                        rel="noreferrer"
                      >
                        {{ organizationDetail.organization.slug }}
                      </a>
                      <span v-else class="muted">—</span>
                    </dd>
                  </div>
                  <div>
                    <dt>Stato</dt>
                    <dd>
                      <span :class="['status-pill', organizationDetail.organization.is_active ? 'active' : 'inactive']">
                        {{ organizationDetail.organization.is_active ? 'Attiva' : 'Disattiva' }}
                      </span>
                    </dd>
                  </div>
                  <div>
                    <dt>Creato il</dt>
                    <dd>{{ formatDate(organizationDetail.organization.created_at) }}</dd>
                  </div>
                </dl>
              </article>

              <article class="card info-card">
                <h3>Riepilogo voti</h3>
                <div class="stat-line">
                  <span>Totale voti</span>
                  <strong>{{ organizationDetail.stats.total_votes }}</strong>
                </div>
                <div class="stat-line">
                  <span>Partite totali</span>
                  <strong>{{ organizationDetail.stats.total_matches }}</strong>
                </div>
                <div class="stat-line">
                  <span>Ultima partita</span>
                  <div>
                    <strong>{{ organizationDetail.stats.last_match_votes }}</strong>
                    <small>{{ formatDate(organizationDetail.stats.last_match_date) }}</small>
                  </div>
                </div>
              </article>
            </div>
          </div>
          <p v-else class="empty">
            {{ isLoadingDetail ? 'Caricamento dettagli…' : 'Seleziona una società per vedere i dettagli.' }}
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';

const token = ref(localStorage.getItem('adminToken') || '');
const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const activeRole = ref(localStorage.getItem('adminRole') || '');
const isAuthenticated = computed(() => Boolean(token.value));
const isSuperAdmin = computed(() => activeRole.value === 'superadmin');

const tabs = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'organizations', label: 'Società' },
];
const activeSection = ref('dashboard');

const loginForm = reactive({ username: '', password: '' });
const isLoggingIn = ref(false);
const loginError = ref('');

const summary = reactive({ total_organizations: 0, total_votes: 0, votes_last_7_days: 0, total_events: 0 });
const isLoadingSummary = ref(false);
const summaryLoaded = ref(false);

const organizations = ref([]);
const isLoadingOrganizations = ref(false);
const organizationsLoaded = ref(false);

const selectedOrganizationId = ref(0);
const organizationDetail = ref(null);
const isLoadingDetail = ref(false);

const organizationForm = reactive({ id: 0, name: '', slug: '', city: '', logo_url: '', is_active: true });
const organizationFormVisible = ref(false);
const organizationFormMode = ref('create');
const isSavingOrganization = ref(false);
const organizationFormError = ref('');

const authHeaders = computed(() => ({
  headers: { Authorization: token.value ? `Bearer ${token.value}` : '' },
}));

function resetOrganizationForm() {
  organizationForm.id = 0;
  organizationForm.name = '';
  organizationForm.slug = '';
  organizationForm.city = '';
  organizationForm.logo_url = '';
  organizationForm.is_active = true;
  organizationFormError.value = '';
}

function openCreateOrganization() {
  organizationFormMode.value = 'create';
  resetOrganizationForm();
  organizationFormVisible.value = true;
}

function openEditOrganization(org) {
  organizationFormMode.value = 'edit';
  organizationForm.id = org.id;
  organizationForm.name = org.name;
  organizationForm.slug = org.slug || '';
  organizationForm.city = org.city || '';
  organizationForm.logo_url = org.logo_url || '';
  organizationForm.is_active = Boolean(org.is_active);
  organizationFormVisible.value = true;
}

function closeOrganizationForm() {
  organizationFormVisible.value = false;
  organizationFormError.value = '';
}

function switchSection(section) {
  activeSection.value = section;
}

function resolveSocietyLink(id) {
  return `/admin?society=${id}`;
}

function resolvePublicLink(slug) {
  if (!slug) return '';
  if (/^https?:\/\//i.test(slug)) {
    return slug;
  }
  if (typeof window === 'undefined' || !window.location?.origin) {
    return `/${slug}`;
  }
  return `${window.location.origin}/${slug}`;
}

function formatDate(value) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('it-IT');
}

async function login() {
  if (!loginForm.username || !loginForm.password) {
    loginError.value = 'Inserisci username e password';
    return;
  }
  loginError.value = '';
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post('/admin/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data?.token || '';
    activeUsername.value = data?.username || '';
    activeRole.value = data?.role || '';
    localStorage.setItem('adminToken', token.value);
    localStorage.setItem('adminUsername', activeUsername.value);
    localStorage.setItem('adminRole', activeRole.value);
    loginForm.username = '';
    loginForm.password = '';
    if (!isSuperAdmin.value) {
      loginError.value = 'Account privo di privilegi master.';
    } else {
      ensureSectionData(activeSection.value);
    }
  } catch (error) {
    if (error?.response?.status === 401) {
      loginError.value = 'Credenziali non valide';
    } else {
      loginError.value = 'Impossibile completare l\'accesso. Riprova.';
    }
  } finally {
    isLoggingIn.value = false;
  }
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  activeRole.value = '';
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  localStorage.removeItem('adminRole');
  organizations.value = [];
  organizationDetail.value = null;
  summaryLoaded.value = false;
  organizationsLoaded.value = false;
  selectedOrganizationId.value = 0;
  activeSection.value = 'dashboard';
}

async function fetchSummary() {
  if (!isSuperAdmin.value || !token.value) return;
  isLoadingSummary.value = true;
  try {
    const { data } = await apiClient.get('/admin/master/summary', authHeaders.value);
    summary.total_organizations = data?.total_organizations ?? 0;
    summary.total_votes = data?.total_votes ?? 0;
    summary.votes_last_7_days = data?.votes_last_7_days ?? 0;
    summary.total_events = data?.total_events ?? 0;
    summaryLoaded.value = true;
  } catch (error) {
    console.error('Impossibile caricare la dashboard master', error);
  } finally {
    isLoadingSummary.value = false;
  }
}

async function fetchOrganizations() {
  if (!isSuperAdmin.value || !token.value) return;
  isLoadingOrganizations.value = true;
  try {
    const { data } = await apiClient.get('/admin/master/organizations', authHeaders.value);
    organizations.value = Array.isArray(data) ? data : [];
    organizationsLoaded.value = true;
  } catch (error) {
    console.error('Impossibile caricare le società', error);
  } finally {
    isLoadingOrganizations.value = false;
  }
}

async function fetchOrganizationDetail(id = selectedOrganizationId.value) {
  if (!isSuperAdmin.value || !token.value || !id) return;
  isLoadingDetail.value = true;
  try {
    const { data } = await apiClient.get(`/admin/master/organizations/${id}`, authHeaders.value);
    organizationDetail.value = data || null;
  } catch (error) {
    console.error('Impossibile caricare il dettaglio società', error);
    organizationDetail.value = null;
  } finally {
    isLoadingDetail.value = false;
  }
}

async function submitOrganizationForm() {
  if (!organizationForm.name) {
    organizationFormError.value = 'Il nome è obbligatorio';
    return;
  }
  organizationFormError.value = '';
  isSavingOrganization.value = true;
  try {
    const payload = {
      name: organizationForm.name,
      slug: organizationForm.slug,
      city: organizationForm.city,
      logo_url: organizationForm.logo_url,
      is_active: organizationForm.is_active,
    };
    if (organizationFormMode.value === 'create') {
      await apiClient.post('/admin/master/organizations', payload, authHeaders.value);
    } else {
      await apiClient.put(`/admin/master/organizations/${organizationForm.id}`, payload, authHeaders.value);
    }
    closeOrganizationForm();
    fetchOrganizations();
    if (organizationFormMode.value === 'edit' && selectedOrganizationId.value === organizationForm.id) {
      fetchOrganizationDetail();
    }
  } catch (error) {
    if (error?.response?.status === 400) {
      organizationFormError.value = 'Dati non validi. Verifica i campi obbligatori.';
    } else if (error?.response?.status === 404) {
      organizationFormError.value = 'Società non trovata.';
    } else {
      organizationFormError.value = 'Errore durante il salvataggio. Riprova.';
    }
  } finally {
    isSavingOrganization.value = false;
  }
}

function viewOrganization(id) {
  selectedOrganizationId.value = id;
  organizationDetail.value = null;
  activeSection.value = 'organization-detail';
}

function ensureSectionData(section) {
  if (!isSuperAdmin.value) return;
  if (section === 'dashboard' && !summaryLoaded.value) {
    fetchSummary();
  }
  if (section === 'organizations' && !organizationsLoaded.value) {
    fetchOrganizations();
  }
  if (section === 'organization-detail' && selectedOrganizationId.value && !organizationDetail.value && !isLoadingDetail.value) {
    fetchOrganizationDetail();
  }
}

watch(activeSection, ensureSectionData);
watch(isSuperAdmin, (value) => {
  if (value) {
    ensureSectionData(activeSection.value);
  }
});

watch(selectedOrganizationId, (value) => {
  if (value && activeSection.value === 'organization-detail') {
    fetchOrganizationDetail(value);
  }
});

onMounted(() => {
  if (isSuperAdmin.value) {
    ensureSectionData(activeSection.value);
  }
});
</script>

<style scoped>
.master-portal {
  padding: clamp(1.5rem, 4vw, 3rem);
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  color: #0f172a;
}

.master-header {
  background: linear-gradient(135deg, #0f172a, #1e293b);
  color: #fff;
  padding: clamp(1.5rem, 3vw, 2.5rem);
  border-radius: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.master-header .subtitle {
  color: rgba(255, 255, 255, 0.8);
  margin: 0.25rem 0 0;
}

.master-header .eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.2em;
  font-size: 0.75rem;
  margin: 0;
  color: rgba(255, 255, 255, 0.7);
}

.master-header h1 {
  margin: 0.2rem 0;
  font-size: clamp(1.5rem, 4vw, 2.5rem);
}

.header-actions {
  display: flex;
  gap: 0.75rem;
}

.master-shell {
  background: #fff;
  border-radius: 1.5rem;
  padding: 1.5rem;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.12);
}

.master-nav {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1.25rem;
}

.nav-btn {
  border: 1px solid rgba(15, 23, 42, 0.15);
  background: #f8fafc;
  color: #0f172a;
  padding: 0.6rem 1.4rem;
  border-radius: 999px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s ease;
}

.nav-btn.active,
.nav-btn:hover {
  background: #0f172a;
  color: #fff;
}

.master-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.grid-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: #0f172a;
  color: #fff;
  padding: 1.5rem;
  border-radius: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.1);
}

.stat-card .label {
  text-transform: uppercase;
  font-size: 0.8rem;
  letter-spacing: 0.1em;
  color: rgba(255, 255, 255, 0.8);
}

.stat-card .value {
  font-size: clamp(2rem, 5vw, 2.8rem);
  margin: 0;
}

.organizations-view .section-header,
.detail-view .section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.section-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.table-wrapper {
  overflow-x: auto;
  background: #fff;
  border-radius: 1rem;
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.table-wrapper table {
  width: 100%;
  border-collapse: collapse;
}

.table-wrapper th,
.table-wrapper td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
}

.table-wrapper th {
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #475569;
}

.org-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.org-cell img {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid rgba(15, 23, 42, 0.1);
}

.org-name {
  margin: 0;
  font-weight: 600;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 600;
}

.status-pill.active {
  background: rgba(16, 185, 129, 0.15);
  color: #047857;
}

.status-pill.inactive {
  background: rgba(248, 113, 113, 0.15);
  color: #b91c1c;
}

.slug-cell {
  display: flex;
  align-items: center;
  min-height: 2rem;
}

.slug-link {
  color: #2563eb;
  font-weight: 600;
  text-decoration: none;
}

.slug-link:hover,
.slug-link:focus {
  text-decoration: underline;
}

.muted {
  color: #94a3b8;
}

.actions {
  display: flex;
  gap: 0.5rem;
}

.empty {
  text-align: center;
  color: #64748b;
}

.form-card header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  font-weight: 600;
}

.form-grid input {
  padding: 0.65rem 0.8rem;
  border-radius: 0.65rem;
  border: 1px solid rgba(15, 23, 42, 0.2);
}

.help-text {
  font-size: 0.75rem;
  color: #64748b;
}

.switch-field {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}

.form-actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.login-card,
.warning-card,
.card {
  background: #fff;
  border-radius: 1.25rem;
  padding: 1.5rem;
  box-shadow: 0 15px 40px rgba(15, 23, 42, 0.1);
}

.warning-card {
  border: 1px solid rgba(248, 113, 113, 0.4);
}

.warning-actions {
  display: flex;
  gap: 1rem;
  margin-top: 1rem;
}

.btn {
  border: none;
  border-radius: 999px;
  padding: 0.5rem 1.25rem;
  font-weight: 600;
  cursor: pointer;
}

.btn.primary {
  background: linear-gradient(135deg, #0ea5e9, #2563eb);
  color: #fff;
}

.btn.outline {
  border: 1px solid rgba(15, 23, 42, 0.2);
  background: transparent;
  color: #0f172a;
}

.btn.ghost {
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid transparent;
  color: inherit;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1.25rem;
}

.info-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.logo-preview {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  border: 2px solid rgba(15, 23, 42, 0.1);
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

dl {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem 1rem;
}

dl dt {
  font-size: 0.8rem;
  text-transform: uppercase;
  color: #94a3b8;
}

dl dd {
  margin: 0;
  font-weight: 600;
}

.stat-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
}

.stat-line strong {
  font-size: 1.5rem;
}

.error {
  color: #b91c1c;
  margin-top: 0.5rem;
}

@media (max-width: 640px) {
  .master-header,
  .section-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-actions,
  .section-actions {
    width: 100%;
    flex-direction: column;
  }
}
</style>

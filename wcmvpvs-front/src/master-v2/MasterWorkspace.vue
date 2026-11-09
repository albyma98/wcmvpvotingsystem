<script setup>
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, provide, reactive, ref, toValue, watch } from 'vue';
import { apiClient } from '../api';
import { advanceMasterSession, observeMasterSession } from './api';
import { masterBase, masterSectionPath, masterSections, readMasterRoute } from './routes';
import { masterWorkspaceKey } from './workspace';
import MasterIcon from './ui/MasterIcon.vue';
import OverviewView from './OverviewView.vue';
import './master.css';

const props = defineProps({ currentPath: { type: String, required: true } });
const emit = defineEmits(['navigate']);
const PortalsView = defineAsyncComponent(() => import('./portals/PortalsView.vue'));
const screens = {
  analytics: defineAsyncComponent(() => import('./operations/AnalyticsView.vue')),
  crm: defineAsyncComponent(() => import('./crm/CrmView.vue')),
  organizations: defineAsyncComponent(() => import('./organizations/OrganizationsView.vue')),
  clubs: PortalsView,
  tournaments: PortalsView,
  arenas: defineAsyncComponent(() => import('./arenas/ArenasView.vue')),
  'qr-redirects': defineAsyncComponent(() => import('./operations/QrRedirectsView.vue')),
  minigames: defineAsyncComponent(() => import('./operations/MinigamesView.vue')),
  'report-branding': defineAsyncComponent(() => import('./operations/BrandingView.vue')),
};
const groups = ['Workspace', 'Gestione', 'Configurazione'].map((label) => ({
  label, entries: masterSections.filter((section) => section.group === label),
}));

const path = ref(props.currentPath);
const route = computed(() => readMasterRoute(path.value));
const pageContext = ref({});
const title = computed(() => {
  if (pageContext.value.title) return pageContext.value.title;
  if (!route.value) return 'Pagina non trovata';
  const { info, mode, id } = route.value;
  if (mode === 'create') return `Creazione ${info.singular}`;
  if (mode === 'edit') return `Modifica ${info.singular}`;
  if (id) return `${info.label} · #${id}`;
  return info.label;
});

const collapsed = ref(false);
const mobileOpen = ref(false);
const menuTrigger = ref(null);
const menuClose = ref(null);
const pageTitle = ref(null);
const dirtySources = reactive(new Set());
const dirty = computed(() => [...dirtySources].some((read) => read()));
const discardMessage = 'Ci sono modifiche non salvate. Vuoi uscire e perderle?';
let approvedPath = '';

function confirmLeave() {
  return !dirty.value || window.confirm(discardMessage);
}

function navigate(href, { discard = false } = {}) {
  if (href === path.value) {
    closeMenu(true);
    return true;
  }
  if (!discard && !confirmLeave()) return false;
  approvedPath = href;
  emit('navigate', href);
  return true;
}

function follow(event) {
  if (event.defaultPrevented || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
  const href = event.currentTarget.getAttribute('href');
  if (!href) return;
  event.preventDefault();
  navigate(href);
}

function beforeUnload(event) {
  if (dirty.value) event.preventDefault();
}

async function toggleMenu() {
  mobileOpen.value = true;
  await nextTick();
  menuClose.value?.focus();
}

async function closeMenu(restoreFocus = false) {
  if (!mobileOpen.value) return;
  mobileOpen.value = false;
  if (restoreFocus) {
    await nextTick();
    menuTrigger.value?.focus();
  }
}

watch(() => props.currentPath, async (next) => {
  const approved = next === approvedPath;
  approvedPath = '';
  if (next === path.value) return;
  if (!approved && !confirmLeave()) {
    emit('navigate', path.value);
    return;
  }
  pageContext.value = {};
  path.value = next;
  closeMenu();
  await nextTick();
  window.scrollTo(0, 0);
  pageTitle.value?.focus({ preventScroll: true });
});

watch(dirty, (value) => {
  if (value) window.addEventListener('beforeunload', beforeUnload);
  else window.removeEventListener('beforeunload', beforeUnload);
});

provide(masterWorkspaceKey, {
  navigate,
  follow,
  setContext: (context) => { pageContext.value = context; },
  registerDirty(source) {
    const read = () => toValue(source);
    dirtySources.add(read);
    return () => dirtySources.delete(read);
  },
});

const session = ref('checking');
const username = ref('');
const summary = ref(null);
const error = ref('');
const busy = ref(false);
const login = reactive({ username: '', password: '' });

function remember(data) {
  username.value = data?.username || '';
  localStorage.setItem('adminUsername', username.value);
  localStorage.setItem('adminRole', data?.role || '');
}

function forget() {
  username.value = '';
  summary.value = null;
  localStorage.removeItem('adminUsername');
  localStorage.removeItem('adminRole');
}

async function restoreSession() {
  session.value = 'checking';
  error.value = '';
  try {
    const { data } = await apiClient.get('/admin/master/summary');
    summary.value = data;
    if (localStorage.getItem('adminRole')?.toLowerCase() === 'superadmin') username.value = localStorage.getItem('adminUsername') || '';
    session.value = 'ready';
  } catch (failure) {
    const status = failure.response?.status;
    if (status === 401) {
      forget();
      session.value = 'login';
    } else if (status === 403) {
      session.value = 'denied';
    } else {
      session.value = 'unavailable';
      error.value = 'Impossibile verificare la sessione. Riprova tra poco.';
    }
  }
}

async function signIn() {
  if (busy.value || !login.username || !login.password) return;
  busy.value = true;
  error.value = '';
  try {
    const { data } = await apiClient.post('/admin/login', { ...login });
    advanceMasterSession();
    remember(data);
    login.username = '';
    login.password = '';
    session.value = data?.role?.toLowerCase() === 'superadmin' ? 'ready' : 'denied';
  } catch (failure) {
    const status = failure.response?.status;
    error.value = status === 401 ? 'Nome utente o password non validi.' : status === 429 ? 'Troppi tentativi. Attendi prima di riprovare.' : 'Impossibile completare l’accesso. Riprova.';
  } finally {
    busy.value = false;
  }
}

async function signOut() {
  if (busy.value || !confirmLeave()) return;
  busy.value = true;
  error.value = '';
  try {
    await apiClient.post('/admin/logout');
    advanceMasterSession();
    forget();
    pageContext.value = {};
    closeMenu();
    session.value = 'login';
  } catch {
    error.value = 'Impossibile terminare la sessione. Riprova.';
  } finally {
    busy.value = false;
  }
}

const stopObserving = observeMasterSession(() => {
  forget();
  pageContext.value = {};
  closeMenu();
  error.value = 'La sessione è scaduta. Accedi di nuovo.';
  session.value = 'login';
});
const consumeSummary = () => { summary.value = null; };
const handleDesktop = (event) => { if (event.matches) closeMenu(); };
let desktop;
onMounted(() => {
  desktop = window.matchMedia('(min-width: 861px)');
  desktop.addEventListener('change', handleDesktop);
  restoreSession();
});
onBeforeUnmount(() => {
  stopObserving();
  desktop?.removeEventListener('change', handleDesktop);
  window.removeEventListener('beforeunload', beforeUnload);
});
</script>

<template>
  <div class="m-root" :class="{ 'm-collapsed': collapsed, 'm-mobile-open': mobileOpen }" @keydown.esc="closeMenu(true)">
    <template v-if="session === 'ready'">
      <a class="m-skip" href="#master-content">Vai al contenuto</a>
      <button v-if="mobileOpen" class="m-scrim" type="button" aria-label="Chiudi navigazione" tabindex="-1" @click="closeMenu(true)"></button>
      <aside class="m-sidebar" id="master-navigation">
        <div class="m-sidebar-top">
          <a class="m-brand" :href="masterBase" aria-label="ArenaBoostX · Panoramica" @click="follow">
            <span class="m-brand-mark"><MasterIcon name="boost" :size="19" /></span>
            <span class="m-brand-text"><strong>ArenaBoost<span>X</span></strong><small>Master Admin</small></span>
          </a>
          <button ref="menuClose" class="m-btn m-icon-btn m-subtle m-mobile-close" type="button" aria-label="Chiudi navigazione" @click="closeMenu(true)"><MasterIcon name="close" /></button>
        </div>
        <nav class="m-sidebar-nav" aria-label="Sezioni Master Admin">
          <div v-for="group in groups" :key="group.label" class="m-nav-group">
            <p class="m-nav-heading">{{ group.label }}</p>
            <a v-for="entry in group.entries" :key="entry.id" class="m-nav-link" :href="masterSectionPath(entry.id)" :class="{ 'm-current': route?.section === entry.id }" :aria-current="route?.section === entry.id ? 'page' : undefined" :aria-label="entry.label" :title="entry.label" @click="follow">
              <MasterIcon :name="entry.icon" /><span>{{ entry.label }}</span>
            </a>
          </div>
        </nav>
        <div class="m-sidebar-footer">
          <a class="m-nav-link" href="/admin" title="Portale società" aria-label="Portale società"><MasterIcon name="arrow-up-right" /><span>Portale società</span></a>
          <button class="m-nav-link m-collapse-toggle" type="button" :aria-label="collapsed ? 'Espandi barra laterale' : 'Comprimi barra laterale'" :aria-expanded="!collapsed" aria-controls="master-navigation" @click="collapsed = !collapsed"><MasterIcon name="panel" /><span>Comprimi menu</span></button>
        </div>
      </aside>

      <div class="m-main-shell" :inert="mobileOpen || undefined">
        <header class="m-topbar">
          <div class="m-top-left">
            <button ref="menuTrigger" type="button" class="m-btn m-subtle m-icon-btn m-mobile-trigger" aria-label="Apri navigazione" :aria-expanded="mobileOpen" aria-controls="master-navigation" @click="toggleMenu"><MasterIcon name="menu" /></button>
            <nav aria-label="Percorso" class="m-breadcrumb">
              <a :href="masterBase" @click="follow">Master Admin</a>
              <MasterIcon name="chevron-right" :size="13" />
              <a v-if="route && route.mode !== 'list'" :href="masterSectionPath(route.section)" @click="follow">{{ route.info.label }}</a>
              <span v-else aria-current="page">{{ route?.info.label ?? 'Pagina non trovata' }}</span>
              <template v-if="route && route.mode !== 'list'"><MasterIcon name="chevron-right" :size="13" /><span aria-current="page">{{ route.mode === 'create' ? 'Creazione' : route.mode === 'edit' ? 'Modifica' : 'Dettaglio' }}</span></template>
            </nav>
          </div>
          <div class="m-account">
            <span class="m-account-mark" aria-hidden="true">{{ (username || 'MA').slice(0, 2).toUpperCase() }}</span>
            <div class="m-account-name"><strong>{{ username || 'Master Admin' }}</strong><span>Superadmin</span></div>
            <button type="button" class="m-btn m-subtle m-icon-btn" aria-label="Esci" title="Esci" :disabled="busy" @click="signOut"><MasterIcon name="logout" /></button>
          </div>
        </header>

        <main id="master-content" class="m-content">
          <div class="m-page-head">
            <div>
              <h1 ref="pageTitle" tabindex="-1">{{ title }}</h1>
              <p v-if="route?.mode === 'list'">{{ route.info.description }}</p>
            </div>
            <a v-if="pageContext.action" class="m-btn m-primary" :href="pageContext.action.href" @click="follow"><MasterIcon :name="pageContext.action.icon || 'plus'" :size="17" />{{ pageContext.action.label }}</a>
          </div>
          <p v-if="error" class="m-error" role="alert">{{ error }}</p>
          <OverviewView v-if="route?.section === 'overview'" :initial-summary="summary" @summary-consumed="consumeSummary" />
          <Suspense v-else-if="route" :timeout="0">
            <component :is="screens[route.section]" :key="route.section" :route="route" />
            <template #fallback><div class="m-panel m-table-state" role="status"><span class="m-spinner" aria-hidden="true"></span><p>Caricamento della sezione…</p></div></template>
          </Suspense>
          <div v-else class="m-panel m-table-state">
            <p>Il percorso richiesto non corrisponde a una sezione del Master Admin.</p>
            <a class="m-btn" :href="masterBase" @click="follow">Torna alla panoramica</a>
          </div>
        </main>
      </div>
    </template>

    <main v-else class="m-auth">
      <div class="m-auth-inner">
        <div class="m-brand">
          <span class="m-brand-mark"><MasterIcon name="boost" :size="23" /></span>
          <span class="m-brand-text"><strong>ArenaBoost<span>X</span></strong><small>Master Admin</small></span>
        </div>
        <section class="m-panel m-auth-card" aria-labelledby="master-auth-title">
          <template v-if="session === 'login'">
            <h1 id="master-auth-title">Accedi al workspace</h1>
            <p class="m-muted-text">Usa le credenziali del tuo account superadmin.</p>
            <form class="m-stack" @submit.prevent="signIn">
              <label class="m-field">Nome utente<input v-model.trim="login.username" class="m-input" name="username" autocomplete="username" placeholder="Nome utente amministratore" required /></label>
              <label class="m-field">Password<input v-model="login.password" class="m-input" name="password" type="password" autocomplete="current-password" placeholder="La tua password" required /></label>
              <p v-if="error" class="m-error" role="alert">{{ error }}</p>
              <button class="m-btn m-primary" type="submit" :disabled="busy">{{ busy ? 'Accesso in corso…' : 'Accedi' }}<MasterIcon name="arrow-right" :size="17" /></button>
            </form>
          </template>
          <template v-else-if="session === 'checking'">
            <h1 id="master-auth-title">Verifica della sessione</h1>
            <p class="m-auth-status m-muted-text" role="status"><span class="m-spinner" aria-hidden="true"></span>Caricamento in corso…</p>
          </template>
          <template v-else-if="session === 'denied'">
            <h1 id="master-auth-title">Accesso riservato</h1>
            <p class="m-muted-text">Il ruolo superadmin è necessario per accedere al Master Admin.</p>
            <p v-if="error" class="m-error" role="alert">{{ error }}</p>
            <div class="m-actions"><a class="m-btn" href="/admin">Portale società</a><button type="button" class="m-btn" :disabled="busy" @click="signOut">Esci</button></div>
          </template>
          <template v-else>
            <h1 id="master-auth-title">Workspace non disponibile</h1>
            <p class="m-error" role="alert">{{ error }}</p>
            <button class="m-btn m-primary" type="button" @click="restoreSession">Riprova</button>
          </template>
        </section>
        <a class="m-link m-auth-legacy" href="/admin/master">Apri il portale Master originale<MasterIcon name="arrow-up-right" :size="14" /></a>
      </div>
    </main>
  </div>
</template>

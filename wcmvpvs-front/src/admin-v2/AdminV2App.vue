<template>
 <div ref="root" class="admin-v2">
  <section v-if="!c.isAuthenticated" class="v2-login-shell">
   <aside class="v2-login-story"><a :href="`/${encodeURIComponent(organizationSlug)}/new`" class="v2-login-brand"><V2Icon name="boost" :size="32" />ArenaBoostX</a><div><span class="v2-eyebrow">IL WORKSPACE DELLA TUA SOCIETÀ</span><h1>Ogni partita.<br>Ogni connessione.<br><em>Tutto sotto controllo.</em></h1><p>Uno spazio per organizzare le esperienze in arena, dare valore ai partner e accompagnare la tua community.</p><div class="v2-login-points"><span><V2Icon name="events" />Eventi ed esperienze</span><span><V2Icon name="results" />Risultati e report</span><span><V2Icon name="teams" />Persone e partner</span></div></div><small>ArenaBoostX · Club workspace</small></aside>
   <div class="v2-login-content"><form class="v2-login-form v2-stack" @submit.prevent="c.login"><div><span class="v2-eyebrow">BENVENUTO</span><h2>Accedi al tuo workspace</h2><p class="v2-muted">Gestisci la tua società, da un unico posto.</p></div><span class="v2-badge">Società · {{ organizationSlug }}</span><label class="v2-field">Nome utente<input v-model.trim="c.loginForm.username" autocomplete="username" required placeholder="Il tuo nome utente" /></label><label class="v2-field">Password<input v-model="c.loginForm.password" type="password" autocomplete="current-password" required placeholder="La tua password" /></label><p v-if="c.loginError" class="v2-error" role="alert">{{ c.loginError }}</p><p v-if="c.accessError" class="v2-error" role="alert">{{ c.accessError }}</p><button class="v2-button v2-primary" :disabled="c.isLoggingIn" :aria-busy="c.isLoggingIn">{{ c.isLoggingIn ? 'Accesso in corso…' : 'Accedi al workspace' }}<V2Icon name="arrow-right" :size="18" /></button><small><V2Icon name="admins" :size="14" /> Area riservata agli amministratori della società</small><a class="v2-login-legacy" :href="`/${encodeURIComponent(organizationSlug)}/admin`">Apri amministrazione precedente</a></form></div>
  </section>
  <div v-else-if="!c.sessionReady" class="v2-loading"><template v-if="c.initialLoading"><span class="v2-spinner" aria-hidden="true"></span><p role="status">Verifica dell’accesso alla società…</p></template><template v-else><V2Icon name="admins" :size="40" /><h1>Workspace non disponibile</h1><p class="v2-muted" role="alert">{{ c.accessError || 'Verifica la sessione per continuare.' }}</p><div class="v2-toolbar"><button class="v2-button v2-primary" @click="refreshWorkspace">Riprova</button><button class="v2-button" @click="c.logout">Esci</button></div></template></div>
  <V2Workspace v-else :section="route.section" :detail="route.detail" :meta="meta">
   <div class="v2-stack">
    <div v-if="c.accessError || c.globalError" class="v2-error" role="alert">{{ c.accessError || c.globalError }}<button class="v2-link" type="button" @click="refreshWorkspace">Riprova caricamento</button></div>
    <div v-if="c.initialLoading" class="v2-grid-3" aria-label="Caricamento dati" role="status"><div v-for="n in 3" :key="n" class="v2-skeleton"></div></div>
    <V2Panel v-else-if="!allowed" :title="unknown ? 'Pagina non trovata' : 'Accesso non disponibile'"><p class="v2-muted">{{ unknown ? 'Questa sezione non esiste nel workspace.' : 'Il tuo ruolo non permette di accedere a questa sezione.' }}</p><button class="v2-button" style="margin-top:20px" @click="navigate(defaultSection)">Torna al workspace</button></V2Panel>
    <DashboardView v-else-if="route.section === 'dashboard'" />
    <EventsView v-else-if="route.section === 'events'" :detail="route.detail" />
    <VotingView v-else-if="route.section === 'closing'" key="closing" />
    <VotingView v-else-if="route.section === 'settings'" key="settings" settings />
    <EntitiesView v-else-if="['teams','players','sponsors','coupons','partners','admins'].includes(route.section)" :section="route.section" :key="route.section" />
    <InsightsView v-else-if="['results','selfies','history'].includes(route.section)" :section="route.section" :key="route.section" />
    <BarView v-else-if="route.section === 'bar'" />
    <MerchView v-else-if="route.section === 'merch'" :detail="route.detail" />
    <MarketingView v-else-if="route.section === 'marketing'" />
    <LotteryView v-else-if="route.section === 'lottery'" :detail="route.detail" />
   </div>
  </V2Workspace>
  <V2Dialog :open="feedback.confirmation.open" :title="feedback.confirmation.title" :busy="feedback.confirmation.pending" @close="feedback.confirmation.open = false"><p>{{ feedback.confirmation.message }}</p><template #footer><button class="v2-button" :disabled="feedback.confirmation.pending" @click="feedback.confirmation.open=false">Annulla</button><button class="v2-button v2-primary" :disabled="feedback.confirmation.pending" @click="feedback.acceptConfirmation">{{ feedback.confirmation.pending ? 'Operazione in corso…' : 'Conferma' }}</button></template></V2Dialog>
  <div class="v2-toast-stack" aria-live="polite" aria-atomic="false"><div v-for="toast in feedback.toasts.value" :key="toast.id" class="v2-toast" :class="toast.tone" :role="toast.tone === 'error' ? 'alert' : 'status'"><span>{{ toast.message }}</span><button type="button" aria-label="Chiudi notifica" @click="feedback.toasts.value=feedback.toasts.value.filter(t=>t.id!==toast.id)"><V2Icon name="close" :size="16" /></button></div></div>
 </div>
</template>
<script setup>
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onErrorCaptured, provide, ref, watch } from 'vue';
import { apiClient } from '../api';
import { useAdminV2Controller } from './composables/useAdminV2Controller';
import { useFeedback } from './composables/useFeedback';
import { readAdminV2Route, navigateAdminV2 } from './router/routes';
import { sectionMeta } from './config/navigation';
import V2Workspace from './layouts/V2Workspace.vue';
import V2Panel from './components/shared/V2Panel.vue';
import V2Dialog from './components/shared/V2Dialog.vue';
import V2Icon from './components/shared/V2Icon.vue';
import DashboardView from './views/DashboardView.vue';
import './styles/admin-v2.css';
const EventsView = defineAsyncComponent(()=>import('./views/EventsView.vue'));
const EntitiesView = defineAsyncComponent(()=>import('./views/EntitiesView.vue'));
const InsightsView = defineAsyncComponent(()=>import('./views/InsightsView.vue'));
const BarView = defineAsyncComponent(()=>import('./views/BarView.vue'));
const MerchView = defineAsyncComponent(()=>import('./views/MerchView.vue'));
const MarketingView = defineAsyncComponent(()=>import('./views/MarketingView.vue'));
const VotingView = defineAsyncComponent(()=>import('./views/VotingView.vue'));
const LotteryView = defineAsyncComponent(()=>import('./views/LotteryView.vue'));
const props = defineProps({organizationSlug:{type:String,required:true},currentPath:{type:String,required:true}});
const c = useAdminV2Controller(props); const feedback=useFeedback();const root=ref(null);
c.organizationSlug=props.organizationSlug;
const route=computed(()=>readAdminV2Route(props.currentPath)||{section:'dashboard',detail:''});
c.route=route;
const defaultSection=computed(()=>c.availableTabs[0]?.id||'closing');
const unknown=computed(()=>!sectionMeta[route.value.section]);
const allowed=computed(()=>['settings','merch'].includes(route.value.section)?c.isSuperAdmin:route.value.section==='lottery'?!c.isStaff:c.availableTabs.some(t=>t.id===route.value.section));
const meta=computed(()=>sectionMeta[route.value.section]||{label:'Pagina non trovata',group:'Workspace',description:''});
let failures=0,mutations=0,unauthorizing=false;const snapshots=new WeakMap(),dirtyForms=new Set();
let submittedForm=null,submissionTimer;const previousTitle=document.title;
function formValues(form){return JSON.stringify([...form.querySelectorAll('input:not([type=search]),select,textarea')].map(el=>[el.id,el.type,el.type==='checkbox'||el.type==='radio'?el.checked:el.value]));}
function editableForm(event){return c.isAuthenticated && event.target.closest?.('.admin-v2') ? event.target.closest('form') : null;}
function rememberForm(event){const form=editableForm(event);if(form&&!snapshots.has(form))snapshots.set(form,formValues(form));}
function trackForm(event){const form=editableForm(event);if(!form)return;if(!snapshots.has(form))snapshots.set(form,'');if(snapshots.get(form)!==formValues(form))dirtyForms.add(form);else dirtyForms.delete(form);}
function hasDirty(){return dirtyForms.size>0;}
function cleanForm(form){if(form){snapshots.set(form,formValues(form));dirtyForms.delete(form);}}
function rememberSubmission(event){submittedForm=editableForm(event);clearTimeout(submissionTimer);submissionTimer=setTimeout(()=>{submittedForm=null;},0);}
function navigate(section,detail=''){if(hasDirty()&&!window.confirm('Ci sono modifiche non salvate. Vuoi lasciare questa pagina?'))return;dirtyForms.clear();navigateAdminV2(props.organizationSlug,section,detail);}
function refreshWorkspace(){if(hasDirty()&&!window.confirm('Ricaricare i dati e sostituire le modifiche non salvate?'))return;dirtyForms.clear();return c.retrySession();}
function beforeUnload(event){if(hasDirty()){event.preventDefault();event.returnValue='';}}
window.addEventListener('beforeunload',beforeUnload);
document.addEventListener('focusin',rememberForm,true);document.addEventListener('input',trackForm,true);document.addEventListener('change',trackForm,true);document.addEventListener('submit',rememberSubmission,true);
c.navigate=navigate;c.refreshWorkspace=refreshWorkspace;c.notify=feedback.notify;c.busy=feedback.busy;c.discardDraft=cleanForm;
c.goToLottery=()=>navigate('lottery');
c.confirmAction=(title,message,action)=>{const form=submittedForm;feedback.confirmAction(title,message,async()=>{const errorsBefore=failures,mutationsBefore=mutations;await action();if(failures===errorsBefore&&mutations>mutationsBefore)cleanForm(form);});};
c.run=(key,fn,success)=>{const errorsBefore=failures,mutationsBefore=mutations;const form=submittedForm;return feedback.run(key,async()=>{c.globalError='';const result=await fn();await nextTick();if(failures===errorsBefore&&mutations>mutationsBefore){if(success)feedback.notify(success);cleanForm(form);}return result;});};
provide('admin-v2',c);
const requests=new AbortController();
const requestInterceptor=apiClient.interceptors.request.use(config=>{if(!config.signal)config.signal=requests.signal;return config;});
const interceptor=apiClient.interceptors.response.use(response=>{if(!['get','head','options'].includes(response.config?.method))mutations++;return response;},error=>{failures++;const status=error?.response?.status,url=error?.config?.url||'';if(status===401&&!url.includes('/admin/login')&&!url.includes('/admin/logout')&&!unauthorizing){unauthorizing=true;c.handleUnauthorized();setTimeout(()=>{unauthorizing=false;},0);}return Promise.reject(error);});
onErrorCaptured(error=>{feedback.notify(error?.response?.data?.message||'Impossibile completare il caricamento. Riprova.','error');return false;});
watch([()=>route.value.section,()=>c.sessionReady,()=>c.availableTabs.map(t=>t.id).join(',')],()=>{
 if(!c.sessionReady)return;
 if(route.value.section==='dashboard'&&!c.isSuperAdmin){navigateAdminV2(props.organizationSlug,defaultSection.value,'',true);return;}
 if(allowed.value && ['lottery','merch'].includes(route.value.section)){c.stopResultsPolling();c.section='';}
 else if(allowed.value)c.selectSection(route.value.section==='settings'?'dashboard':route.value.section);
 else c.stopResultsPolling();
},{immediate:true});
watch(()=>route.value.section,()=>{dirtyForms.clear();});
watch(meta,value=>{document.title=`${value.label} · ArenaBoostX`;},{immediate:true});
onBeforeUnmount(()=>{clearTimeout(submissionTimer);requests.abort();apiClient.interceptors.request.eject(requestInterceptor);apiClient.interceptors.response.eject(interceptor);document.title=previousTitle;window.removeEventListener('beforeunload',beforeUnload);document.removeEventListener('focusin',rememberForm,true);document.removeEventListener('input',trackForm,true);document.removeEventListener('change',trackForm,true);document.removeEventListener('submit',rememberSubmission,true);});
</script>
<style scoped>
.v2-login-shell{display:grid;grid-template-columns:1.05fr 1fr;min-height:100vh}.v2-login-story{display:flex;flex-direction:column;justify-content:space-between;padding:50px 64px;background:#133c31;color:#fff;overflow:hidden;position:relative}.v2-login-story:after{content:'';position:absolute;width:550px;height:550px;right:-220px;bottom:-260px;border:1px solid #ffffff14;border-radius:50%;box-shadow:0 0 0 70px #ffffff04,0 0 0 140px #ffffff03;pointer-events:none}.v2-login-brand{display:flex;align-items:center;gap:10px;color:#fff;font-size:24px;font-weight:700;letter-spacing:-.8px}.v2-login-story h1{font-size:clamp(32px,3.4vw,58px);font-weight:550;line-height:1.13;letter-spacing:-2px;margin:24px 0}.v2-login-story h1 em{color:#a3d4af;font-style:normal}.v2-login-story .v2-eyebrow{color:#adccbb;font-size:10px}.v2-login-story p{max-width:410px;font-size:14px;line-height:1.8;color:#d1e0d7}.v2-login-story>small{color:#a9c1b6;font-size:10px}.v2-login-points{display:flex;flex-wrap:wrap;gap:16px;margin-top:36px}.v2-login-points span{display:flex;align-items:center;gap:6px;font-size:10px;color:#cfdfd5}.v2-login-points svg{width:15px}.v2-login-content{display:grid;place-items:center;padding:40px;background:#fafbfc}.v2-login-form{width:min(360px,100%);gap:22px}.v2-login-form h2{font-size:27px;font-weight:650;letter-spacing:-.9px;margin:12px 0 10px}.v2-login-form>div>p{font-size:13px}.v2-login-form>.v2-badge{justify-self:start}.v2-login-form>.v2-button{margin-top:5px;min-height:48px;justify-content:space-between}.v2-login-form>small{display:flex;align-items:center;gap:7px;font-size:10px;justify-content:center}.v2-login-legacy{text-align:center;font-size:11px}
@media(max-width:1000px){.v2-login-story{padding:36px}.v2-login-points{flex-direction:column;gap:12px}.v2-login-content{padding:28px}}@media(max-width:700px){.v2-login-shell{display:block}.v2-login-story{padding:24px 24px 32px}.v2-login-story>div,.v2-login-story>small{display:none}.v2-login-brand{font-size:20px}.v2-login-content{padding:40px 24px 60px}.v2-login-form h2{font-size:26px}}
</style>

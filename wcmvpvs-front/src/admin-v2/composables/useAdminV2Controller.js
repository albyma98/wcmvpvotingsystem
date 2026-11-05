import { onBeforeUnmount, onMounted, proxyRefs, ref } from 'vue';
import { useController } from '../../components/AdminPortal.vue?admin-v2-controller';

export function sessionErrorMessage(error) {
  if (error?.response?.status === 401) return 'Sessione scaduta. Effettua di nuovo l’accesso.';
  if (error?.response?.status === 403) return 'Non hai accesso a questa società o a questa operazione.';
  return 'Impossibile caricare il workspace. Controlla la connessione e riprova.';
}

/** Thin presentation/session adapter; CRUD, payloads and permissions stay in the existing controller. */
export function useAdminV2Controller(props) {
  const sessionReady = ref(false);
  const initialLoading = ref(false);
  const accessError = ref('');
  let disposed = false;
  const controller = useController(props, () => {}, {
    onSessionVerified() { if (!disposed) { sessionReady.value = true; accessError.value = ''; } },
    onSessionError(error) { if (!disposed) { sessionReady.value = false; accessError.value = sessionErrorMessage(error); } },
    onSessionEnded() { sessionReady.value = false; },
    onLoading(loading) { if (!disposed) initialLoading.value = loading; },
    onLoadError(error) { if (!disposed) accessError.value = sessionErrorMessage(error); },
    onRequestError(error) {
      if (!disposed && error?.response?.status === 403) accessError.value = sessionErrorMessage(error);
    },
  });
  let pending;
  async function retrySession() {
    if (pending || !controller.isAuthenticated.value) return pending;
    accessError.value = '';
    pending = controller.loadAll().catch(() => {}).finally(() => { pending = undefined; });
    return pending;
  }
  onMounted(retrySession);
  onBeforeUnmount(() => { disposed = true; controller.stopResultsPolling(); });
  return proxyRefs({ ...controller, sessionReady, initialLoading, accessError, retrySession });
}

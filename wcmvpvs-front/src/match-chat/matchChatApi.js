import { apiClient, getDeviceHeaders, getFanSessionHeaders, resolveApiUrl } from '../api';

export const chatEmojis = ['❤️', '🔥', '😂', '👏', '💪'];
export const chatReasons = { abuse: 'Contenuto inappropriato', spam: 'Spam', other: 'Altro' };

export const chatRequestId = () => Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('');
export const chatTime = value => new Date(value).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' });
export const chatDateTime = value => new Date(value).toLocaleString('it-IT', { dateStyle: 'short', timeStyle: 'short' });

export function matchChatRequest(slug, eventId, { admin = false, method = 'get', path = '', data, params, signal } = {}) {
  return apiClient({
    method, data, params, signal, timeout: 12000,
    url: `${admin ? '/admin' : ''}/events/${eventId}/match-chat${path}`,
    headers: { 'X-Organization-Slug': slug, ...(admin ? {} : { ...getDeviceHeaders(), ...getFanSessionHeaders(slug) }) },
  }).then(response => response.data);
}

export function chatErrorCode(error) {
  return error?.response?.data?.error;
}

export function chatErrorText(error) {
  return ({
    chat_disabled: 'La chat è stata disattivata.',
    chat_muted: 'Il tuo profilo è temporaneamente sospeso in questa chat.',
    chat_rate_limited: 'Stai interagendo troppo velocemente. Riprova tra poco.',
    chat_duplicate: 'Hai già inviato un messaggio uguale di recente.',
    chat_verification_required: 'Accedi con un profilo tifoso verificato per partecipare.',
    chat_item_not_found: 'Il contenuto non è più disponibile.',
    chat_reply_unavailable: 'Il messaggio a cui stai rispondendo non è più disponibile.',
    invalid_message: 'Inserisci un messaggio da 1 a 180 caratteri.',
  })[chatErrorCode(error)] || 'Operazione non riuscita. Riprova.';
}

// Reset backoff after the server acknowledges the subscription.
export function connectMatchChat(slug, eventId, { statusOnly, preview, onUpdate, onDisconnect }) {
  const url = new URL(resolveApiUrl(`/events/${eventId}/match-chat/${statusOnly ? 'status/' : ''}stream`));
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.searchParams.set('organization_slug', slug);
  if (statusOnly && preview) url.searchParams.set('preview', '1');
  let socket;
  let timer;
  let handshakeTimer;
  let stopped = false;
  let delay = 1000;

  function retry() {
    clearTimeout(handshakeTimer);
    if (stopped) return;
    onDisconnect?.();
    if (stopped) return;
    timer = setTimeout(connect, delay + Math.random() * delay / 4);
    delay = Math.min(delay * 2, 20000);
  }

  function connect() {
    try {
      socket = new WebSocket(url);
    } catch {
      retry();
      return;
    }
    handshakeTimer = setTimeout(() => socket.close(), 12000);
    socket.onmessage = event => {
      let update;
      try { update = JSON.parse(event.data); } catch { return; }
      if (update.type === 'status') {
        clearTimeout(handshakeTimer);
        delay = 1000;
      }
      onUpdate(update);
    };
    socket.onclose = retry;
  }

  connect();
  return () => {
    stopped = true;
    clearTimeout(timer);
    clearTimeout(handshakeTimer);
    if (!socket) return;
    socket.onmessage = null;
    socket.onclose = null;
    socket.close();
  };
}

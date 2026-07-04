import { onMounted, onUnmounted, watch, isRef, unref } from 'vue'

/**
 * Stream live del torneo via SSE (rimpiazza il polling).
 * Apre un EventSource su /api/v1/tournaments/:slug/stream e chiama onUpdate()
 * a ogni notifica di cambiamento emessa dal server dopo una scrittura.
 *
 * - slugSource: string | Ref | getter — lo slug può arrivare dopo (es. console
 *   operatore, che lo riceve dopo il login): in quel caso ci si aggancia appena
 *   diventa disponibile.
 * - Riconnessione: nativa dell'EventSource. Al riaggancio e al rientro in
 *   foreground rifà una onUpdate() per non perdere aggiornamenti persi offline.
 */
export function useTournamentStream (slugSource, onUpdate) {
  let es = null
  let opened = false
  const getSlug = () => (typeof slugSource === 'function' ? slugSource() : unref(slugSource))

  function connect () {
    const slug = getSlug()
    if (!slug || es) return
    try {
      es = new EventSource(`/api/v1/tournaments/${slug}/stream`, { withCredentials: true })
    } catch { es = null; return }
    es.addEventListener('update', () => onUpdate())
    es.addEventListener('open', () => {
      if (opened) onUpdate()   // riconnessione: recupera ciò che è cambiato mentre eravamo giù
      opened = true
    })
  }
  function disconnect () { es?.close(); es = null; opened = false }

  function onVisibility () { if (!document.hidden) onUpdate() }

  onMounted(() => {
    connect()
    if (isRef(slugSource) || typeof slugSource === 'function') {
      watch(getSlug, (s) => { if (s && !es) connect() })
    }
    document.addEventListener('visibilitychange', onVisibility)
  })

  onUnmounted(() => {
    disconnect()
    document.removeEventListener('visibilitychange', onVisibility)
  })
}

import { ref, onMounted, onUnmounted } from 'vue'

const MOCK = {
  tournament: {
    slug: 'sunset-beach-cup', name: 'Sunset Beach Cup', format: 'BEACH VOLLEY 4X4',
    dateLabel: '8 - 11 GIUGNO 2024', location: 'LIDO DI CLASSE, RA',
    statusLabel: 'TORNEO IN CORSO', phaseLabel: 'FASE A GIRONI', logo: null, heroImage: null
  },
  liveMatches: [
    { id: 'm1', court: 'CAMPO 2', teamA: { name: 'Mambo Beach' }, teamB: { name: 'Netbreakers' }, score: { a: 1, b: 1 }, setLabel: '1° SET', sets: ['21-18', '18-21'] },
    { id: 'm2', court: 'CAMPO 3', teamA: { name: 'Sunset Kings' }, teamB: { name: 'Block Party' }, score: { a: 0, b: 1 }, setLabel: '2° SET', sets: ['19-21'] }
  ],
  nextMatch: { court: 'CAMPO 1', time: '18:30', teamA: { name: 'Sand Kings' }, teamB: { name: 'Beach Volley', sub: 'PESARO' } },
  tiles: [
    { id: 'calendar', icon: 'calendar', label: 'Calendario', sub: 'Tutte le partite', color: '#35357F', route: '/calendar' },
    { id: 'standings', icon: 'chart', label: 'Classifiche', sub: 'Gironi e ranking', color: '#5B2333', route: '/standings' },
    { id: 'bracket', icon: 'bracket', label: 'Tabellone', sub: 'Fase finale', color: '#0E5F4C', route: '/bracket' },
    { id: 'mvp', icon: 'star', label: 'Vota MVP', sub: 'Vota il migliore', color: '#A8730F', route: '/mvp' },
    { id: 'prizes', icon: 'trophy', label: 'Premi', sub: 'Cosa si vince', color: '#6B5A12', route: '/prizes' },
    { id: 'gallery', icon: 'gallery', label: 'Gallery', sub: 'Foto del torneo', color: '#8F2B44', route: '/gallery' },
    { id: 'rules', icon: 'doc', label: 'Regolamento', sub: 'Info e regole', color: '#3A4A63', route: '/rules' },
    { id: 'event', icon: 'info', label: 'Info Evento', sub: 'Mappa e servizi', color: '#5B2E86', route: '/event' }
  ],
  sponsors: [
    { id: 1, name: 'WC WearingCash', tier: 'main' },
    { id: 2, name: 'MIKASA', tier: 'main' },
    { id: 3, name: 'BEACH ARENA', tier: 'partner', brandColor: '#0E7C5B' },
    { id: 4, name: 'BPER Banca', tier: 'partner', brandColor: '#00539F' },
    { id: 5, name: 'RADIO BRUNO', tier: 'partner', brandColor: '#FFD400' },
    { id: 6, name: 'GELATERIA ONDA', tier: 'partner', brandColor: '#FF6B9D' },
    { id: 7, name: 'BAGNO 54', tier: 'partner', brandColor: '#F97316' },
    { id: 8, name: 'SPORT CAFÈ', tier: 'partner', brandColor: '#8B5CF6' }
  ],
  shopProducts: []
}


/**
 * Home torneo: snapshot completo al mount + aggiornamenti live via SSE (push,
 * niente polling). Il server notifica sullo stream a ogni scrittura e noi
 * rifacciamo la sola fetch /live (payload ~1KB).
 *
 * GET /api/v1/tournaments/:slug/home    → snapshot completo
 * GET /api/v1/tournaments/:slug/live    → solo { liveMatches, nextMatch }
 * GET /api/v1/tournaments/:slug/stream  → SSE: tick a ogni cambiamento
 */
export function useTournamentHome (slug, { mock = false } = {}) {
  const tournament = ref(null)
  const liveMatches = ref([])
  const nextMatch = ref(null)
  const tiles = ref([])
  const sponsors = ref([])
  const shopProducts = ref([])
  const loading = ref(true)
  const error = ref(null)

  let es = null
  let esOpened = false
  let aborter = null

  async function fetchHome () {
    loading.value = true
    error.value = null
    if (mock) {
      tournament.value = MOCK.tournament
      liveMatches.value = MOCK.liveMatches
      nextMatch.value = MOCK.nextMatch
      tiles.value = MOCK.tiles
      sponsors.value = MOCK.sponsors
      shopProducts.value = MOCK.shopProducts
      loading.value = false
      return
    }
    try {
      const res = await fetch(`/api/v1/tournaments/${slug}/home`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      tournament.value = data.tournament
      liveMatches.value = data.liveMatches ?? []
      nextMatch.value = data.nextMatch ?? null
      tiles.value = data.tiles ?? []
      sponsors.value = data.sponsors ?? []
      shopProducts.value = data.shopProducts ?? []
    } catch (e) {
      error.value = e
    } finally {
      loading.value = false
    }
  }

  async function pollLive () {
    if (mock) return
    // Non pollare in background: risparmia batteria e banda in tribuna
    if (document.hidden) return
    try {
      aborter?.abort()
      aborter = new AbortController()
      // no-store: l'SSE spinge ad ogni punto, il fetch NON deve tornare la
      // risposta cachata dal browser (altrimenti il punteggio "salta" ogni ~3s)
      const res = await fetch(`/api/v1/tournaments/${slug}/live`, { signal: aborter.signal, cache: 'no-store' })
      if (!res.ok) return
      const data = await res.json()
      liveMatches.value = data.liveMatches ?? []
      if (data.nextMatch !== undefined) nextMatch.value = data.nextMatch
      if (data.tournament?.phaseLabel && tournament.value) {
        tournament.value.phaseLabel = data.tournament.phaseLabel
        tournament.value.statusLabel = data.tournament.statusLabel
      }
    } catch { /* polling silenzioso: il prossimo tick riprova */ }
  }

  function onVisibility () {
    if (!document.hidden) pollLive() // refresh immediato al rientro in app
  }

  onMounted(() => {
    fetchHome()
    if (!mock) {
      try {
        es = new EventSource(`/api/v1/tournaments/${slug}/stream`, { withCredentials: true })
        es.addEventListener('update', pollLive)
        es.addEventListener('open', () => { if (esOpened) pollLive(); esOpened = true })
      } catch { es = null }
    }
    document.addEventListener('visibilitychange', onVisibility)
  })

  onUnmounted(() => {
    es?.close()
    aborter?.abort()
    document.removeEventListener('visibilitychange', onVisibility)
  })

  return { tournament, liveMatches, nextMatch, tiles, sponsors, shopProducts, loading, error, refresh: fetchHome }
}

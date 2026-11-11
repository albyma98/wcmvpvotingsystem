export function clubHomeTiles(data = {}) {
  const icons = data.club?.tileIcons ?? {}
  const contacts = data.club?.contacts ?? []
  return [
    ...(data.teams ?? []).slice(0, 4).map((team, index) => ({
      id: `team-${team.id}`, kind: 'team', team,
      label: team.name, sub: team.championship || team.category || 'Scopri la squadra',
      icon: ['🏐', '⚡', '🏆', '⭐'][index % 4],
      iconImage: icons[`team-${team.id}`] || team.logoUrl || '',
    })),
    { id: 'matches', kind: 'panel', panel: 'matches', label: 'Partite', sub: `${data.matches?.length ?? 0} appuntamenti`, icon: '📅', iconImage: icons.matches || '' },
    { id: 'announcements', kind: 'panel', panel: 'announcements', label: 'Annunci', sub: `${(data.posts ?? []).filter(post => post.kind === 'announcement').length} comunicazioni`, icon: '📣', iconImage: icons.announcements || '' },
    { id: 'contacts', kind: 'panel', panel: 'contacts', label: 'Contatti', sub: contacts.length ? `${contacts.length} ${contacts.length === 1 ? 'contatto' : 'contatti'} della società` : 'Scrivici o vieni a trovarci', icon: '☎', iconImage: icons.contacts || '' },
    { id: 'sponsors', kind: 'panel', panel: 'sponsors', label: 'I nostri Sponsor', sub: 'Chi sostiene la nostra società', icon: '🤝', iconImage: icons.sponsors || '' },
  ]
}

export function clubContactPhoneLink(phone = '') {
  const number = phone.replace(/[^\d+]/g, '')
  return /\d/.test(number) ? `tel:${number}` : ''
}

export function sortClubMatches(matches = []) {
  return [...matches].sort((a, b) => {
    const first = Date.parse(a.scheduledAt)
    const second = Date.parse(b.scheduledAt)
    if (!Number.isFinite(first)) return Number.isFinite(second) ? 1 : 0
    if (!Number.isFinite(second)) return -1
    return first - second
  })
}

export function formatClubDate(value) {
  if (!value) return 'Data da definire'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('it-IT', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

export function clubInitials(name) {
  return String(name || '').split(/\s+/).map(part => part[0]).join('').slice(0, 2).toUpperCase()
}

export function clubScoreLabel(match) {
  return match.home ? `${match.scoreFor} – ${match.scoreAgainst}` : `${match.scoreAgainst} – ${match.scoreFor}`
}

export function clubMatchSets(match) {
  try {
    const sets = JSON.parse(match.setsJson || '[]')
    return Array.isArray(sets) ? sets.filter(value => typeof value === 'string') : []
  } catch {
    return []
  }
}

export function clubSetLabel(set, home) {
  const scores = set.split(/[-–:]/).map(score => score.trim())
  return scores.length === 2 ? (home ? scores : scores.reverse()).join('–') : set
}

export function clubMatchHasWinner(match) {
  return Boolean(match.bestOf) && Math.max(match.scoreFor, match.scoreAgainst) > match.bestOf / 2
}

export function clubCurrentSetLabel(match) {
  if (!match.bestOf || match.status !== 'live') return ''
  if (clubMatchHasWinner(match)) return 'In attesa del risultato finale'
  const points = match.home ? [match.pointsFor, match.pointsAgainst] : [match.pointsAgainst, match.pointsFor]
  return `${match.scoreFor + match.scoreAgainst + 1}° set · ${points.join('–')}`
}

export function connectClubMatches(slug, onMessage, onConnection) {
  if (typeof WebSocket !== 'function') return () => {}
  const url = new URL(`/api/v1/clubs/${encodeURIComponent(slug)}/matches/stream`, window.location.href)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  let socket
  let timer
  let retryDelay = 500
  let stopped = false

  function connect() {
    if (stopped) return
    socket = new WebSocket(url)
    socket.onmessage = event => {
      let update
      try { update = JSON.parse(event.data) } catch { return }
      if (Array.isArray(update.matches)) {
        retryDelay = 500
        onConnection(true)
      }
      onMessage(update)
    }
    socket.onclose = () => {
      if (stopped) return
      onConnection(false)
      timer = setTimeout(connect, retryDelay)
      retryDelay = Math.min(retryDelay * 2, 10000)
    }
  }

  connect()
  return () => {
    stopped = true
    clearTimeout(timer)
    socket.onmessage = null
    socket.onclose = null
    socket.close()
  }
}

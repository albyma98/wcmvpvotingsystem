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

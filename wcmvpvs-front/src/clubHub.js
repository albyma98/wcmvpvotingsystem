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
    { id: 'contacts', kind: 'panel', panel: 'contacts', label: 'Contatti', sub: contacts.length ? `${contacts.length} ${contacts.length === 1 ? 'contatto' : 'contatti'} della società` : 'Scrivici o vieni a trovarci', icon: '☎', iconImage: icons.contacts || '', wide: true },
  ]
}

export function clubContactPhoneLink(phone = '') {
  const number = phone.replace(/[^\d+]/g, '')
  return /\d/.test(number) ? `tel:${number}` : ''
}

// Resolve the current Host on the server. No hostname or club slug is supplied by the browser.
export async function resolveClubDomain(fetcher = globalThis.fetch, signal) {
  const response = await fetcher('/api/v1/club-domain', { credentials: 'same-origin', cache: 'no-store', signal })
  if (response.status === 204) return null
  if (!response.ok) throw new Error('domain_unavailable')
  const domain = await response.json()
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(domain.clubSlug || '') || !/^[a-z0-9.-]+$/.test(domain.primaryHostname || '')) {
    throw new Error('invalid_domain_response')
  }
  return domain
}

export function clubDomainRoute(domain, pathname) {
  if (!domain) return null
  const parts = pathname.split('/').filter(Boolean)
  const clubSlug = domain.clubSlug
  if (!parts.length) return { view: 'club', clubSlug, teamSlug: '' }
  if (parts[0] === 'club-admin' && parts[1] === clubSlug && parts.length === 2) return { view: 'club-admin', clubSlug, teamSlug: '' }
  if (parts[0] === 'club' && parts[1] === clubSlug) parts.splice(0, 2)
  if (!parts.length) return { view: 'club', clubSlug, teamSlug: '' }
  const mvp = clubMVPRoute(parts)
  if (mvp) return { view: 'club-mvp', clubSlug, ...mvp }
  if (parts[0] === 'team' && parts.length === 2) {
    try { return { view: 'club', clubSlug, teamSlug: decodeURIComponent(parts[1]) } } catch { /* malformed URL */ }
  }
  return { view: 'club-not-found', clubSlug, teamSlug: '' }
}

export function clubHomePath(slug, customDomain = false) {
  return customDomain ? '/' : `/club/${encodeURIComponent(slug)}`
}

function clubMVPRoute(parts) {
  if (parts[0] !== 'mvp') return null
  if (parts.length === 1) return { matchId: '' }
  if (parts.length === 2 && /^[1-9]\d*$/.test(parts[1])) return { matchId: parts[1] }
  return null
}

export function clubMVPPath(slug, customDomain, matchId) {
  const base = customDomain ? '/mvp' : `${clubHomePath(slug)}/mvp`
  return matchId ? `${base}/${matchId}` : base
}

export function clubMVPEntryRoute(domain, pathname) {
  if (domain) {
    const route = clubDomainRoute(domain, pathname)
    return route.view === 'club-mvp' ? route : null
  }
  const [prefix, clubSlug, ...parts] = pathname.split('/').filter(Boolean)
  const route = prefix === 'club' && clubSlug ? clubMVPRoute(parts) : null
  return route ? { view: 'club-mvp', clubSlug, ...route } : null
}

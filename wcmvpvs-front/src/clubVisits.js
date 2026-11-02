const visitorStorageKey = 'clubhub_visitor_id_v1'
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
let memoryVisitorId

function visitorId() {
  try {
    const stored = globalThis.localStorage?.getItem(visitorStorageKey)
    if (uuidPattern.test(stored)) return stored
  } catch { /* Storage can be disabled by the browser. */ }
  memoryVisitorId ||= globalThis.crypto.randomUUID()
  try { globalThis.localStorage?.setItem(visitorStorageKey, memoryVisitorId) } catch { /* Keep the identifier for this page. */ }
  return memoryVisitorId
}

// Call once after a public page loads, never for modal data requests or CMS pages.
export async function recordClubVisit(slug) {
  try {
    await fetch(`/api/v1/clubs/${encodeURIComponent(slug)}/visits`, {
      method: 'POST',
      credentials: 'omit',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ visitorId: visitorId(), visitId: globalThis.crypto.randomUUID() }),
      keepalive: true,
    })
  } catch { /* Analytics must never prevent the public page from working. */ }
}

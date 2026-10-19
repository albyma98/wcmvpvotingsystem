function sessionKey(organizationSlug) {
  const slug = String(organizationSlug || '').trim().toLowerCase();
  return slug ? `fan:${encodeURIComponent(slug)}:session_token` : '';
}

export function readFanSessionToken(storage, organizationSlug) {
  const key = sessionKey(organizationSlug);
  if (!storage || !key) return '';
  try { return String(storage.getItem(key) || '').trim(); } catch { return ''; }
}

export function saveFanSessionToken(storage, organizationSlug, token) {
  const key = sessionKey(organizationSlug);
  if (!storage || !key || !token) return;
  try { storage.setItem(key, String(token).trim()); } catch { /* Storage unavailable. */ }
}

// Legacy tokens must be validated by /fan/me before entering a club's storage.
export function readLegacyFanSessionToken(storage) {
  try { return String(storage?.getItem('fan:session_token') || '').trim(); } catch { return ''; }
}

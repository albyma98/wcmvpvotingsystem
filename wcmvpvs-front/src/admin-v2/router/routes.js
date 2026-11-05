const reservedSlugs = new Set(['admin', 't', 'ta', 'club', 'club-admin', 'operator', 'op', 'proietta', 'partner', 'lottery', 'welcome']);

export function readAdminV2Route(pathname = '') {
  const parts = pathname.split('/').filter(Boolean);
  if (parts.length < 3 || parts[1] !== 'admin' || parts[2] !== 'new' || reservedSlugs.has(parts[0])) return null;
  try {
    const slug = decodeURIComponent(parts[0]);
    if (!slug || /[/?#\\]/.test(slug) || reservedSlugs.has(slug)) return null;
    return { slug, section: parts[3] ? decodeURIComponent(parts[3]) : 'dashboard', detail: parts.slice(4).map(decodeURIComponent).join('/') };
  } catch { return null; }
}

export function adminV2Path(slug, section = 'dashboard', detail = '') {
  if (!slug || /[/?#\\]/.test(slug) || reservedSlugs.has(slug)) throw new Error('Società non valida');
  const base = `/${encodeURIComponent(slug)}/admin/new`;
  return base + (section && section !== 'dashboard' ? `/${encodeURIComponent(section)}` : '') + (detail ? `/${String(detail).split('/').map(encodeURIComponent).join('/')}` : '');
}

export function navigateAdminV2(slug, section = 'dashboard', detail = '', replace = false) {
  const target = adminV2Path(slug, section, detail);
  window.history[replace ? 'replaceState' : 'pushState']({}, '', target);
  window.dispatchEvent(new Event('popstate'));
  return target;
}

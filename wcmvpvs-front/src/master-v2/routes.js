export const masterBase = '/admin/master/new';

export const masterSections = [
  { id: 'overview', group: 'Workspace', label: 'Panoramica', icon: 'dashboard', description: 'Metriche e strumenti per amministrare ArenaBoostX.' },
  { id: 'analytics', group: 'Workspace', label: 'Analytics', icon: 'results', description: 'Votazioni, visibilità degli sponsor e coinvolgimento del pubblico.' },
  { id: 'organizations', group: 'Gestione', label: 'Società sportive', icon: 'building', singular: 'società', description: 'Anagrafiche, servizi e configurazioni delle società.' },
  { id: 'clubs', group: 'Gestione', label: 'ClubHub', icon: 'globe', singular: 'ClubHub', description: 'Portali dei club, accessi amministrativi e domini.' },
  { id: 'tournaments', group: 'Gestione', label: 'Tornei', icon: 'lottery', singular: 'torneo', description: 'Tornei pubblici e accessi alla gestione operativa.' },
  { id: 'arenas', group: 'Gestione', label: 'Arene', icon: 'box', singular: 'arena', description: 'Catalogo delle arene 3D e configurazioni delle scene.' },
  { id: 'qr-redirects', group: 'Configurazione', label: 'Redirect QR', icon: 'qr', description: 'Destinazioni e stato dei percorsi usati nei codici QR.' },
  { id: 'minigames', group: 'Configurazione', label: 'Minigiochi', icon: 'gamepad', description: 'Disponibilità dei minigiochi per disciplina sportiva.' },
  { id: 'report-branding', group: 'Configurazione', label: 'Report e marchio', icon: 'file', description: 'Emittente e marchio dei report di piattaforma.' },
];

const collections = {
  organizations: ['arena', 'edit'],
  clubs: ['domains'],
  tournaments: [],
  arenas: ['edit'],
};

export function isMasterWorkspacePath(path) {
  return path === masterBase || path.startsWith(`${masterBase}/`);
}

export function masterSectionPath(section) {
  return section === 'overview' ? masterBase : `${masterBase}/${section}`;
}

export function readMasterRoute(path) {
  if (!isMasterWorkspacePath(path)) return null;
  const tail = path.slice(masterBase.length + 1).replace(/([^/])\/$/, '$1');
  const [section, item, tab, ...extra] = tail ? tail.split('/') : ['overview'];
  const info = masterSections.find((entry) => entry.id === section);
  if (!info || extra.length || (section === 'overview' && tail)) return null;
  if (item === undefined) return { section, id: null, mode: 'list', tab: '', info };
  const tabs = collections[section];
  if (!tabs || (tab !== undefined && !tabs.includes(tab))) return null;
  if (item === 'create' && !tab) return { section, id: null, mode: 'create', tab: '', info };
  const id = Number(item);
  if (!/^[1-9]\d*$/.test(item) || !Number.isSafeInteger(id)) return null;
  return { section, id, mode: tab === 'edit' ? 'edit' : 'detail', tab: tab || '', info };
}

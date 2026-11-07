import { normalizeOrganizationColors } from '../../organizationTheme.js';
import { masterSectionPath } from '../routes.js';

export const ORGANIZATIONS_ENDPOINT = '/admin/master/organizations';
export const ORGANIZATIONS_PATH = masterSectionPath('organizations');
export const ORGANIZATION_SPORTS = ['Calcio', 'Pallacanestro', 'Pallavolo'];
export const COLOR_FIELDS = [
  { key: 'primary_color', label: 'Colore primario', hint: 'Pulsanti e azioni principali' },
  { key: 'secondary_color', label: 'Colore secondario', hint: 'Accenti ed elementi in evidenza' },
  { key: 'tertiary_color', label: 'Colore di sfondo', hint: 'Sfondo della home tifosi' },
];

export function organizationDraft(organization) {
  return {
    name: organization?.name ?? '',
    slug: organization?.slug ?? '',
    city: organization?.city ?? '',
    sport: organization?.sport ?? '',
    arena_id: organization?.arena_id ?? null,
    logo_url: organization?.logo_url ?? '',
    is_active: organization ? Boolean(organization.is_active) : true,
    bar_enabled: organization?.bar_enabled !== false,
    sms_cost: Number(organization?.sms_cost ?? 0.08),
    free_sms: Number(organization?.free_sms ?? 0),
    ...normalizeOrganizationColors(organization),
  };
}

export function organizationPayload(draft, original) {
  const payload = {
    ...draft,
    ...normalizeOrganizationColors(draft),
    sms_cost: Number(draft.sms_cost || 0),
    free_sms: Number(draft.free_sms || 0),
  };
  if (original && draft.arena_id === (original.arena_id ?? null)) delete payload.arena_id;
  return payload;
}

export function publicOrganizationUrl(slug) {
  if (!slug) return '';
  return /^https?:\/\//i.test(slug) ? slug : `/${slug}`;
}

export function adminOrganizationUrl(organization) {
  if (!organization.slug) return `/admin?society=${organization.id}`;
  const publicUrl = publicOrganizationUrl(organization.slug);
  try {
    const url = new URL(publicUrl, window.location.origin);
    url.pathname = `${url.pathname.replace(/\/+$/, '')}/admin`;
    return url.toString();
  } catch {
    return `${publicUrl.replace(/\/+$/, '')}/admin`;
  }
}

export const smsPrice = new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' });

export function organizationDate(value) {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('it-IT');
}

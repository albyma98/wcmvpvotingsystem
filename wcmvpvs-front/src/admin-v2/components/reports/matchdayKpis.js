const numberFormat = new Intl.NumberFormat('it-IT', { maximumFractionDigits: 2 });
const euroFormat = new Intl.NumberFormat('it-IT', { style: 'currency', currency: 'EUR' });
const dateFormat = new Intl.DateTimeFormat('it-IT', { dateStyle: 'medium', timeStyle: 'short' });
const shortDateFormat = new Intl.DateTimeFormat('it-IT', { day: 'numeric', month: 'short' });

const units = { percent: '%', seconds: ' s', milliseconds: ' ms' };

export const statusLabels = {
  available: 'Disponibile',
  partial: 'Parziale',
  missing: 'Dato mancante',
  not_measurable: 'Non misurabile',
  not_applicable: 'Non applicabile',
};

export const sourceLabels = { db: 'DB', manual: 'Manuale', mixed: 'Misto' };

export function formatKpi(value, unit) {
  if (!Number.isFinite(value)) return '—';
  if (unit === 'eur') return euroFormat.format(value);
  return `${numberFormat.format(value)}${units[unit] || ''}`;
}

export function formatDelta(value, unit) {
  if (!Number.isFinite(value)) return '—';
  const sign = value > 0 ? '+' : '';
  if (unit === 'percent') return `${sign}${formatKpi(value)} p.p.`;
  return `${sign}${formatKpi(value, unit)}`;
}

export function formatKpiDate(value, short = false) {
  if (!value) return '';
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? (short ? shortDateFormat : dateFormat).format(date) : '';
}

export function manualDrafts(fields, retained = {}) {
  return Object.fromEntries(fields.map(field => [field.code, retained[field.code] || {
    value: field.value == null ? '' : String(field.value),
    note: field.note || '',
  }]));
}

export function manualChanges(fields, drafts) {
  return fields.flatMap(field => {
    const draft = drafts[field.code];
    if (!draft) return [];
    const value = draft.value === '' ? null : Number(draft.value);
    const note = draft.note.trim();
    if (value === (field.value ?? null) && note === (field.note || '')) return [];
    return [{ code: field.code, value, note, revision: field.revision ?? 0 }];
  });
}

export function manualChangeError(change, field) {
  const max = field.code === 'attendees' ? 1e7 : 1e9;
  if (!Number.isFinite(change.value) || change.value < 0 || change.value > max || (field.unit === 'count' && !Number.isInteger(change.value))) {
    const message = `Inserisci un numero ${field.unit === 'count' ? 'intero ' : ''}tra 0 e ${formatKpi(max)}.`;
    return { input: 'value', message };
  }
  if (field.unit === 'eur' && Math.abs(change.value * 100 - Math.round(change.value * 100)) > 0.0001) {
    return { input: 'value', message: 'Inserisci un importo con al massimo due cifre decimali.' };
  }
  if ([...change.note].length < 3 || [...change.note].length > 500) return { input: 'note', message: 'Indica la fonte o aggiungi una nota di 3–500 caratteri.' };
  return null;
}

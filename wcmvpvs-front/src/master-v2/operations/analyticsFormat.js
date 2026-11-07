const numbers = new Intl.NumberFormat('it-IT');
const decimals = new Intl.NumberFormat('it-IT', { maximumFractionDigits: 1 });
const day = new Intl.DateTimeFormat('it-IT', { day: 'numeric', month: 'short', timeZone: 'UTC' });
const month = new Intl.DateTimeFormat('it-IT', { month: 'long', year: 'numeric', timeZone: 'UTC' });

function hasNumber(value) {
  return value !== null && value !== undefined && value !== '' && Number.isFinite(Number(value));
}

export function number(value) {
  return hasNumber(value) ? numbers.format(value) : '—';
}

export function percent(value) {
  return hasNumber(value) ? `${decimals.format(value)}%` : '—';
}

export function duration(value) {
  if (!hasNumber(value)) return '—';
  const total = Math.max(0, Math.floor(Number(value)));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor(total / 60) % 60;
  if (hours) return `${number(hours)} h ${minutes} min`;
  if (minutes) return `${minutes} min`;
  return `${total} s`;
}

export function delta(change) {
  if (!hasNumber(change?.absolute)) return '—';
  const absolute = `${change.absolute > 0 ? '+' : ''}${number(change.absolute)}`;
  return hasNumber(change.percent) ? `${absolute} (${percent(change.percent)})` : absolute;
}

export function deltaTone(value) {
  if (value > 0) return 'm-good';
  if (value < 0) return 'm-warn';
  return 'm-muted';
}

export function dateLabel(value) {
  const date = value ? new Date(value) : null;
  return date && Number.isFinite(date.getTime()) ? date.toLocaleDateString('it-IT') : '—';
}

export function monthLabel(value) {
  if (!value) return '—';
  const date = new Date(`${value}-01T00:00:00Z`);
  return Number.isFinite(date.getTime()) ? month.format(date) : value;
}

export function trendPoints(series) {
  if (!Array.isArray(series)) return [];
  return series.flatMap(point => {
    const date = point?.date ? new Date(point.date) : null;
    if (!date || !Number.isFinite(date.getTime()) || !hasNumber(point?.votes)) return [];
    return [{ time: date.getTime(), label: day.format(date), votes: Number(point.votes) }];
  }).sort((a, b) => a.time - b.time);
}

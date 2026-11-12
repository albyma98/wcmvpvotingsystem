import { formatReportValue, reportSourceLabel } from './engagementReport.js';

export function recipientKpis(snapshot, recipient) {
  const profile = snapshot?.profiles?.find(item => item.id === recipient);
  const byId = new Map((snapshot?.kpis || []).map(item => [item.id, item]));
  return (profile?.kpi_ids || []).map(id => byId.get(id)).filter(Boolean);
}

export function recipientSourceLabel(kpi) {
  const direct = (kpi.provenance || []).some(m => m.source_type === 'manual_kpi');
  if (direct && kpi.source === 'manual_verified') return 'Valore KPI manuale verificato';
  if (direct && kpi.source === 'manual_unverified') return 'Valore KPI manuale NON verificato';
  if (kpi.source === 'manual_verified') return 'Sorgente manuale verificata';
  if (kpi.source === 'manual_unverified') return 'Sorgente manuale NON verificata';
  return reportSourceLabel(kpi.source);
}

export function comparisonLabel(kpi) {
  const c = kpi.comparison;
  if (!c || c.absolute == null) return c?.reason || 'Confronto N/D';
  const signed = value => `${value > 0 ? '+' : value < 0 ? '-' : ''}${formatReportValue(Math.abs(value), 'percent').replace('%', '')}`;
  if (c.percentage_points != null) return `${signed(c.percentage_points)} punti percentuali`;
  const absolute = `${c.absolute > 0 ? '+' : c.absolute < 0 ? '-' : ''}${formatReportValue(Math.abs(c.absolute), kpi.unit)}`;
  return `${absolute}${c.percent != null ? ` (${signed(c.percent)}%)` : c.reason ? ` · ${c.reason}` : ''}`;
}

export function reportAttendanceField(kpis, attendance) {
  const adoption = kpis.find(kpi => kpi.id === 'venue_adoption');
  if (!adoption || !attendance && !adoption.availability?.manual_allowed) return null;
  return { type: 'integer', unit: 'people', min: 0, max: 10000000 };
}

export function manualSourceErrors(form, field = { type: 'integer', min: 0, max: 10000000 }) {
  const errors = [];
  const n = form.value === '' || form.value == null ? NaN : Number(form.value);
  if (!Number.isFinite(n) || n < field.min || n > field.max || field.type === 'integer' && !Number.isInteger(n)) errors.push(`Inserisci ${field.type === 'integer' ? 'un valore intero' : 'un valore numerico'} tra ${field.min} e ${field.max}.`);
  if (String(form.source_description || '').trim().length < 3 || String(form.source_description).length > 500) errors.push('Indica una fonte verificabile da 3 a 500 caratteri.');
  if (String(form.note || '').length > 1000) errors.push('La nota può contenere al massimo 1.000 caratteri.');
  if (!form.confirmed) errors.push('Conferma che il valore si riferisce a questa partita e alla definizione del KPI selezionato.');
  return errors;
}

export function analysisMetricReferences(analysisItem, kpis) {
  return (analysisItem.metric_ids || []).map(id => kpis.find(k => k.id === id && k.value != null)).filter(Boolean).map(k => `${k.label}: ${formatReportValue(k.value, k.unit)}`).join(' · ');
}

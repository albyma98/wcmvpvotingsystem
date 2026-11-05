import { formatReportValue, reportSourceLabel } from './engagementReport.js';

export function recipientKpis(snapshot, recipient) {
  const profile = snapshot?.profiles?.find(item => item.id === recipient);
  const byId = new Map((snapshot?.kpis || []).map(item => [item.id, item]));
  return (profile?.kpi_ids || []).map(id => byId.get(id)).filter(Boolean);
}

export function recipientSourceLabel(kpi) {
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

export function manualSourceErrors(form) {
  const errors = [];
  const n = form.value === '' || form.value == null ? NaN : Number(form.value);
  if (!Number.isInteger(n) || n < 0 || n > 10000000) errors.push('Inserisci presenze intere tra 0 e 10.000.000.');
  if (String(form.source_description || '').trim().length < 3 || String(form.source_description).length > 500) errors.push('Indica una fonte verificabile da 3 a 500 caratteri.');
  if (String(form.note || '').length > 1000) errors.push('La nota può contenere al massimo 1.000 caratteri.');
  if (!form.confirmed) errors.push('Conferma che le presenze si riferiscono a questa partita e al pubblico presente.');
  return errors;
}

export function analysisMetricReferences(analysisItem, kpis) {
  return (analysisItem.metric_ids || []).map(id => kpis.find(k => k.id === id && k.value != null)).filter(Boolean).map(k => `${k.label}: ${formatReportValue(k.value, k.unit)}`).join(' · ');
}

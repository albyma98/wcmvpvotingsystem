import { jsPDF } from 'jspdf';
import { REPORT_KPIS, formatReportValue, reportOperationalItems, reportSourceLabel } from './engagementReport.js';

const C = { navy: [14, 33, 57], teal: [12, 120, 101], ink: [28, 45, 65], muted: [99, 115, 133], pale: [242, 246, 249], line: [218, 227, 235] };
const WIDTH = 210, HEIGHT = 297, M = 16;

// Helvetica provides crisp selectable text and supports Italian WinAnsi glyphs.
// Unsupported code points (including emojis in user-supplied names) are removed.
export function pdfText(value) {
  return String(value ?? '').normalize('NFC').replace(/[\u2010-\u2015]/g, '-').replace(/[\u2018\u2019]/g, "'").replace(/[\u201c\u201d]/g, '"').replace(/\u2026/g, '...').replace(/\u00a0/g, ' ').replace(/[^\x20-\x7e\xa0-\xff\n]/g, '').replace(/[ \t]+/g, ' ').trim();
}

function date(value) { const parsed = new Date(value); return value && Number.isFinite(parsed.getTime()) ? new Intl.DateTimeFormat('it-IT', { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'Europe/Rome' }).format(parsed) : 'N/D'; }

function createPainter(doc) {
  function text(value, x, y, size = 9, color = C.ink, bold = false, options) {
    doc.setFont('helvetica', bold ? 'bold' : 'normal'); doc.setFontSize(size); doc.setTextColor(...color); doc.text(pdfText(value), x, y, options);
  }
  function lines(value, x, y, width, maxLines = 2, size = 9, color = C.ink, bold = false, lineHeight = size * .4) {
    doc.setFont('helvetica', bold ? 'bold' : 'normal'); doc.setFontSize(size);
    const all = doc.splitTextToSize(pdfText(value), width);
    const result = all.slice(0, maxLines);
    if (all.length > maxLines && result.length) {
      let final = result[result.length - 1];
      while (doc.getTextWidth(`${final}...`) > width && final.length) final = final.slice(0, -1);
      result[result.length - 1] = `${final.trimEnd()}...`;
    }
    result.forEach((line, index) => text(line, x, y + index * lineHeight, size, color, bold));
    return all.length > maxLines;
  }
  function box(x, y, w, h, color, radius = 2) { doc.setFillColor(...color); doc.roundedRect(x, y, w, h, radius, radius, 'F'); }
  function fit(value, x, y, width, size = 9, color = C.ink, bold = false, align = 'left') {
    const cleaned = pdfText(value);
    doc.setFont('helvetica', bold ? 'bold' : 'normal'); doc.setFontSize(size);
    const scaled = Math.max(6, Math.min(size, size * width / Math.max(1, doc.getTextWidth(cleaned))));
    doc.setFontSize(scaled);
    let result = cleaned;
    if (doc.getTextWidth(result) > width) {
      while (doc.getTextWidth(`${result}...`) > width && result.length) result = result.slice(0, -1);
      result += '...';
    }
    text(result, x, y, scaled, color, bold, { align });
  }
  function line(y, x = M, end = WIDTH - M) { doc.setDrawColor(...C.line); doc.setLineWidth(.2); doc.line(x, y, end, y); }
  return { text, lines, fit, box, line };
}

function normalizedKpis(report) {
  return REPORT_KPIS.map(definition => {
    const item = report.kpis?.find(kpi => kpi.id === definition.id) || {};
    const override = report.overrides?.[definition.id];
    const corrected = item.corrected || item.source === 'manual' || Boolean(override);
    return { ...definition, ...item, numerator: override ? null : item.numerator, denominator: override ? null : item.denominator, corrected, display_description: item.display_description || '' };
  });
}

function coverageNotes(report) {
  return (report.details?.coverage_notes || []).filter(note => typeof note === 'string' && note.trim());
}

function sourceSummary(report) {
  const hasOperational = reportOperationalItems(report.details).length > 0 || report.kpis.some(kpi => ['operational', 'mixed'].includes(kpi.source));
  return hasOperational ? 'Fonti: telemetria /new e dati operativi dell’evento.' : 'Fonte: misurazione dedicata alla home /new.';
}

function compactOperationalSummary(report) {
  const values = Object.fromEntries(reportOperationalItems(report.details).map(item => [item.key, item.value]));
  const parts = [];
  if (Object.hasOwn(values, 'votes')) parts.push(`${formatReportValue(values.votes)} voti`);
  if (Object.hasOwn(values, 'shop_redemptions')) parts.push(`${formatReportValue(values.shop_redemptions)} riscatti`);
  const coins = [];
  if (Object.hasOwn(values, 'coins_earned')) coins.push(`+${formatReportValue(values.coins_earned)}`);
  if (Object.hasOwn(values, 'coins_spent')) coins.push(`-${formatReportValue(values.coins_spent)}`);
  if (coins.length) parts.push(`coin ${coins.join('/')}`);
  if (Object.hasOwn(values, 'feedback_responses')) parts.push(`${formatReportValue(values.feedback_responses)} feedback`);
  if (Object.hasOwn(values, 'sponsor_clicks')) parts.push(`${formatReportValue(values.sponsor_clicks)} clic sponsor`);
  return parts.length ? `Operativo: ${parts.join(' | ')}` : '';
}

function drawLogo(doc, data, x, y, maxW, maxH) {
  if (!/^data:image\/(png|jpe?g);base64,/i.test(data || '')) return false;
  try {
    const properties = doc.getImageProperties(data);
    const scale = Math.min(maxW / properties.width, maxH / properties.height);
    doc.addImage(data, properties.fileType, x, y, properties.width * scale, properties.height * scale);
    return true;
  } catch { return false; }
}

function header(doc, p, report, title, subtitle) {
  p.box(0, 0, WIDTH, 3, C.teal, 0);
  drawLogo(doc, report.branding?.logo_data, M, 12, 60, 12);
  p.text('EVENT INTELLIGENCE', WIDTH - M, 18, 7.4, C.teal, true, { align: 'right' });
  p.line(29);
  p.text(title, M, 40, 21, C.navy, true);
  p.lines(subtitle, M, 47, WIDTH - M * 2, 1, 8.5, C.muted);
  p.lines(report.event?.title || `Evento ${report.event?.id || ''}`, M, 57, 128, 1, 11.5, C.ink, true);
  p.text(date(report.event?.start_datetime), WIDTH - M, 57, 9, C.muted, false, { align: 'right' });
  p.lines([report.event?.organization_name, report.event?.location].filter(Boolean).join('  |  '), M, 63, WIDTH - M * 2, 1, 8, C.muted);
}

function footer(doc, p, report, page, total) {
  p.line(280);
  drawLogo(doc, report.branding?.logo_data, M, 282, 36, 8);
  p.fit(`Report #${report.id || 'N/D'}`, 84, 286, 48, 7, C.muted);
  p.text(`Approvato ${date(report.approved_at)}  |  ${page}/${total}`, WIDTH - M, 286, 7, C.muted, false, { align: 'right' });
}

function summary(p, kpis, y) {
  const featured = ['visitors', 'active_participation', 'sponsor_reach'];
  featured.forEach((id, index) => {
    const kpi = kpis.find(item => item.id === id);
    const x = M + index * 61;
    p.box(x, y, 56, 24, index === 1 ? C.navy : C.pale);
    p.fit(formatReportValue(kpi.value, kpi.unit), x + 4, y + 11, 48, 17, index === 1 ? [255, 255, 255] : C.navy, true);
    p.fit(kpi.label, x + 4, y + 19, 48, 7, index === 1 ? [195, 226, 219] : C.muted);
  });
}

function overview(doc, p, report, kpis, total) {
  header(doc, p, report, 'Il valore dell’engagement', 'Sintesi direzionale | 20 indicatori strategici dell’esperienza /new');
  summary(p, kpis, 70);
  const operational = compactOperationalSummary(report);
  if (operational) p.fit(operational, M, 99, 178, 7, C.muted);
  p.text('PERFORMANCE DELL’EVENTO', M, 104, 8, C.teal, true);
  p.text('Rilevato e verificato dalla società', WIDTH - M, 104, 7, C.muted, false, { align: 'right' });
  const colW = 86, gap = 6, rowH = 16.2;
  kpis.forEach((kpi, index) => {
    const col = Math.floor(index / 10), row = index % 10;
    const x = M + col * (colW + gap), y = 109 + row * rowH;
    if (row % 2 === 0) p.box(x, y, colW, rowH - .5, C.pale, 1);
    p.text(`${String(index + 1).padStart(2, '0')}`, x + 2, y + 4.5, 6.3, C.muted);
    p.lines(kpi.label, x + 8, y + 4.5, 47, 2, 7.3, C.ink, false, 3.1);
    p.fit(formatReportValue(kpi.value, kpi.unit), x + colW - 2, y + 5.5, 26, 10, C.navy, true, 'right');
    const reference = kpi.denominator != null && kpi.numerator != null ? `${formatReportValue(kpi.numerator)}/${formatReportValue(kpi.denominator)}` : '';
    p.fit(reference, x + colW - 2, y + 8.8, 26, 6.3, C.muted, false, 'right');
    if (kpi.display_description) p.lines(kpi.display_description, x + 8, y + 11.1, 76, 2, 7, C.muted, false, 3.2);
  });
  p.lines(report.notes ? `Note: ${report.notes}` : coverageNotes(report)[0] || 'N/D: misura non disponibile. Testi abbreviati per la leggibilità.', M, 275, 178, 1, 6.5, C.muted);
  footer(doc, p, report, 1, total);
}

function table(p, title, columns, rows, y, emptyMessage, maxRows = 6, rowHeight = 9) {
  p.text(title, M, y, 11, C.navy, true);
  const headY = y + 5;
  p.box(M, headY, 178, 10, C.navy, 1);
  columns.forEach(column => p.text(column.label, M + column.x + 3, headY + 6, 7, [255, 255, 255], true));
  if (!rows.length) { p.text(emptyMessage, M + 3, headY + 20, 8, C.muted); return; }
  rows.slice(0, maxRows).forEach((row, index) => {
    const rowY = headY + 10 + index * rowHeight;
    if (index % 2 === 0) p.box(M, rowY, 178, rowHeight, C.pale, 0);
    columns.forEach(column => p.lines(row[column.key] ?? 'N/D', M + column.x + 3, rowY + rowHeight * .65, column.width - 5, 1, 7.4, C.ink));
  });
  if (rows.length > maxRows) p.text(`Mostrati ${maxRows} di ${rows.length} elementi. Dettaglio completo nella versione approvata archiviata.`, M + 3, headY + 15 + maxRows * rowHeight, 6.7, C.muted);
}

function detailPage(doc, p, report) {
  doc.addPage();
  header(doc, p, report, 'Esperienze e partner', 'Dettaglio operativo | Utilizzo dei giochi e risposta agli sponsor');
  const games = (report.details?.games || []).map(item => ({ name: item.name || item.id, players: formatReportValue(item.players), started: formatReportValue(item.rounds_started), completed: formatReportValue(item.rounds_completed) }));
  table(p, 'Giochi e attività', [{ key: 'name', label: 'ESPERIENZA', x: 0, width: 76 }, { key: 'players', label: 'GIOCATORI', x: 76, width: 34 }, { key: 'started', label: 'AVVIATI', x: 110, width: 32 }, { key: 'completed', label: 'COMPLETATI', x: 142, width: 36 }], games, 77, 'Nessun dettaglio giochi disponibile nella misurazione.', 8, 7.2);
  const sponsors = (report.details?.sponsors || []).map(item => {
    const exposureMeasured = item.exposure_measured !== false;
    return {
      name: item.name || item.id,
      reach: formatReportValue(exposureMeasured ? item.reach : null),
      impressions: formatReportValue(exposureMeasured ? item.impressions : null),
      clickers: formatReportValue(exposureMeasured ? item.clickers : null),
      ctr: formatReportValue(exposureMeasured ? item.ctr : null, 'percent'),
    };
  });
  table(p, 'Risultati per sponsor', [{ key: 'name', label: 'PARTNER', x: 0, width: 62 }, { key: 'reach', label: 'COPERTURA', x: 62, width: 30 }, { key: 'impressions', label: 'ESPOSIZIONI', x: 92, width: 35 }, { key: 'clickers', label: 'UTENTI CLIC', x: 127, width: 31 }, { key: 'ctr', label: 'TASSO', x: 158, width: 20 }], sponsors, 163, 'Nessun dettaglio sponsor disponibile nella misurazione.');
  const operational = reportOperationalItems(report.details);
  if (operational.length) {
    p.text('DATI OPERATIVI DELL’EVENTO', M, 257, 7.2, C.teal, true);
    p.lines(operational.map(item => `${item.label}: ${formatReportValue(item.value)}`).join(' | '), M, 262, 178, 3, 7, C.ink, false, 3.4);
  }
  footer(doc, p, report, 2, 3);
}

function methodPage(doc, p, report, kpis) {
  doc.addPage();
  header(doc, p, report, 'Metodo e prossimi passi', 'Trasparenza della misurazione | Fonti e indicazioni operative');
  p.text('BASE DI CALCOLO E FONTI', M, 76, 9, C.teal, true);
  kpis.forEach((kpi, index) => {
    const y = 82 + index * 7.3;
    if (index % 2 === 0) p.box(M, y, 178, 7.3, C.pale, 0);
    p.text(String(index + 1).padStart(2, '0'), M + 2, y + 4.5, 6.2, C.muted);
    p.lines(kpi.label, M + 8, y + 4.5, 65, 1, 6.7, C.ink, false);
    const reference = kpi.denominator != null && kpi.numerator != null ? `${formatReportValue(kpi.numerator)} / ${formatReportValue(kpi.denominator)}` : formatReportValue(kpi.value, kpi.unit);
    p.lines(reference, M + 77, y + 4.5, 39, 1, 6.5, C.muted);
    const source = kpi.value == null ? 'N/D' : reportSourceLabel(kpi.corrected ? 'manual' : kpi.source, true);
    p.lines(source, M + 121, y + 4.5, 54, 1, 6.5, C.muted);
  });
  p.text('COPERTURA DELLE FONTI', M, 237, 8, C.teal, true);
  const sourceCount = report.details?.sources?.length;
  const coverage = coverageNotes(report).join(' ') || 'Le tracce operative descrivono una partecipazione minima. Le visite senza azioni e i pubblici eleggibili richiedono telemetria. Risposte e clic senza identità non misurano persone uniche.';
  p.lines(`${sourceSummary(report)}${sourceCount ? ` ${sourceCount} tabelle consultate.` : ''} ${coverage}`, M, 243, 178, 3, 7.3, C.ink, false, 3.5);
  p.lines(report.notes ? `Note: ${report.notes}` : 'N/D = misura assente o denominatore nullo. Utenti anonimi deduplicati per browser. Il tempo medio usa solo utenti con intervalli validi, pagina visibile e attività recente. Le registrazioni non attestano consenso marketing.', M, 258, 178, 4, 7, C.muted, false, 3.4);
  p.text('Testi e tabelle abbreviati per la leggibilità. Dati integrali conservati nel report approvato.', M, 275, 6.3, C.muted);
  footer(doc, p, report, 3, 3);
}

/** Returns a deterministic A4 document from an immutable server-approved snapshot. */
export function createEngagementReportPdf(report) {
  if (!report?.id || !report?.approved_at || !Array.isArray(report.kpis)) throw new Error('È necessaria una versione approvata del report.');
  if (!['one_page', 'extended'].includes(report.format)) throw new Error('Formato report non supportato.');
  const doc = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4', compress: true, putOnlyUsedFonts: true });
  const p = createPainter(doc), kpis = normalizedKpis(report), total = report.format === 'extended' ? 3 : 1;
  doc.setProperties({ title: pdfText(`Engagement - ${report.event?.title || 'Evento'}`), subject: pdfText(`20 KPI - Report approvato ${report.id}`), author: pdfText(report.branding?.company_name || 'ArenaBoostX'), creator: 'ArenaBoostX Event Intelligence', keywords: 'engagement, evento, KPI, sponsor, report' });
  overview(doc, p, report, kpis, total);
  if (total === 3) { detailPage(doc, p, report); methodPage(doc, p, report, kpis); }
  return doc;
}

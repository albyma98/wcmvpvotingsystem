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
  const definitions = new Map(REPORT_KPIS.map(kpi => [kpi.id, kpi]));
  const available = new Map();
  for (const item of report.kpis) {
    if (!item || !definitions.has(item.id) || available.has(item.id)) throw new Error('Il report contiene indicatori sconosciuti o duplicati.');
    available.set(item.id, item);
  }
  const selection = report.selected_kpi_ids === undefined ? [...available.keys()] : report.selected_kpi_ids;
  if (!Array.isArray(selection) || !selection.length) throw new Error('Seleziona almeno una KPI da inserire nel PDF.');
  if (new Set(selection).size !== selection.length) throw new Error('La selezione delle KPI contiene duplicati.');
  if (selection.some(id => !available.has(id))) throw new Error('La selezione contiene una KPI non presente nel report approvato.');
  if (selection.length > (report.format === 'one_page' ? 20 : 30)) throw new Error(report.format === 'one_page' ? 'Il formato one pager contiene al massimo 20 KPI. Riduci la selezione o scegli il report di tre pagine.' : 'Il report di tre pagine contiene al massimo 30 KPI.');
  const selected = new Set(selection);
  // Snapshot order is canonical. Older approved reports retain their actual KPI
  // set instead of acquiring newly introduced indicators at download time.
  return report.kpis.filter(item => selected.has(item.id)).map(item => {
    const definition = definitions.get(item.id);
    const override = report.overrides?.[item.id];
    const corrected = item.corrected || item.source === 'manual' || Boolean(override);
    return { ...definition, ...item, numerator: override ? null : item.numerator, denominator: override ? null : item.denominator, corrected, display_description: item.display_description || '' };
  });
}

function isEstimated(kpi) {
  return kpi.source === 'estimated' || ['estimated_total_time', 'estimated_avg_time'].includes(kpi.id);
}

function referenceText(kpi) {
  const fraction = kpi.denominator != null && kpi.numerator != null ? `${formatReportValue(kpi.numerator)}/${formatReportValue(kpi.denominator)}` : '';
  return [isEstimated(kpi) ? 'Stima' : '', fraction].filter(Boolean).join(' | ');
}

function timeEstimateSummary(report, kpis) {
  if (!kpis.some(isEstimated) || !report.details?.time_estimate) return '';
  const estimate = report.details.time_estimate;
  const parts = [];
  const number = value => typeof value === 'number' && Number.isFinite(value) && value >= 0;
  if (number(estimate.covered_visitors) && number(estimate.total_visitors)) parts.push(`Copertura: ${formatReportValue(estimate.covered_visitors)}/${formatReportValue(estimate.total_visitors)} partecipanti osservabili`);
  if (number(estimate.excluded_visitors)) parts.push(`${formatReportValue(estimate.excluded_visitors)} senza intervalli stimabili`);
  if (number(estimate.interval_cap_seconds) && number(estimate.gap_limit_seconds)) parts.push(`Intervalli fino a ${formatReportValue(estimate.interval_cap_seconds)} s; pause ammesse fino a ${formatReportValue(estimate.gap_limit_seconds)} s`);
  if (typeof estimate.method === 'string' && estimate.method.trim()) parts.push(estimate.method.trim());
  return parts.length ? `Stima del tempo: ${parts.join('. ')}.` : '';
}

function selectedOperationalItems(report, kpis) {
  const selected = new Set(kpis.map(kpi => kpi.id));
  const relevant = {
    votes: ['mvp_participation'], identified_participants: ['identified_visitors'],
    game_reward_receipts: ['completed_rounds'], shop_redemptions: ['shop_redemptions'],
    coins_earned: ['completed_rounds', 'redemption_conversion', 'shop_redemptions'],
    coins_spent: ['redemption_conversion', 'shop_redemptions'],
    feedback_responses: ['feedback_collected'], sponsor_clicks: ['sponsor_clicks'], sponsor_clicks_without_identity: ['sponsor_clicks'],
  };
  return reportOperationalItems(report.details)
    .filter(item => relevant[item.key]?.some(id => selected.has(id)))
    .map(item => ['coins_earned', 'coins_spent'].includes(item.key) ? { ...item, label: item.key === 'coins_earned' ? 'Coin virtuali guadagnati' : 'Coin virtuali spesi' } : item);
}

function coverageNotes(report) {
  // Time estimates have their own selection-aware method and coverage block.
  // Do not repeat the generic backend note when those KPIs were not selected.
  return (report.details?.coverage_notes || []).filter(note => typeof note === 'string' && note.trim() && !/^\s*tempo stimato\s*:/i.test(note));
}

function sourceSummary(report, kpis) {
  const hasOperational = reportOperationalItems(report.details).length > 0 || kpis.some(kpi => ['operational', 'mixed'].includes(kpi.source));
  return hasOperational ? 'Fonti: telemetria /new e dati operativi dell’evento.' : 'Fonte: misurazione dedicata alla home /new.';
}

function compactOperationalSummary(report, kpis) {
  const values = Object.fromEntries(selectedOperationalItems(report, kpis).map(item => [item.key, item.value]));
  const parts = [];
  if (Object.hasOwn(values, 'votes')) parts.push(`${formatReportValue(values.votes)} voti`);
  if (Object.hasOwn(values, 'shop_redemptions')) parts.push(`${formatReportValue(values.shop_redemptions)} riscatti`);
  const coins = [];
  if (Object.hasOwn(values, 'coins_earned')) coins.push(`+${formatReportValue(values.coins_earned)}`);
  if (Object.hasOwn(values, 'coins_spent')) coins.push(`-${formatReportValue(values.coins_spent)}`);
  if (coins.length) parts.push(`coin virtuali ${coins.join('/')}`);
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
  const featured = ['visitors', 'active_participation', 'sponsor_reach'].map(id => kpis.find(item => item.id === id)).filter(Boolean);
  const cards = [...featured, ...kpis.filter(kpi => !featured.includes(kpi))].slice(0, 3);
  cards.forEach((kpi, index) => {
    const x = M + index * 61;
    p.box(x, y, 56, 24, index === 1 ? C.navy : C.pale);
    p.fit(formatReportValue(kpi.value, kpi.unit), x + 4, y + 11, 48, 17, index === 1 ? [255, 255, 255] : C.navy, true);
    p.fit(`${isEstimated(kpi) ? 'Stima | ' : ''}${kpi.label}`, x + 4, y + 19, 48, 7, index === 1 ? [195, 226, 219] : C.muted);
  });
}

function overview(doc, p, report, kpis, total) {
  header(doc, p, report, 'Il valore dell’engagement', `Sintesi direzionale | ${kpis.length} indicatori selezionati dell’esperienza /new`);
  summary(p, kpis, 70);
  const operational = compactOperationalSummary(report, kpis);
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
    p.fit(referenceText(kpi), x + colW - 2, y + 8.8, 26, 6.3, C.muted, false, 'right');
    if (kpi.display_description) p.lines(kpi.display_description, x + 8, y + 11.1, 76, 2, 7, C.muted, false, 3.2);
  });
  const note = report.notes ? `Note: ${report.notes}` : coverageNotes(report)[0] || 'N/D: misura non disponibile. Testi abbreviati per la leggibilità.';
  const estimate = timeEstimateSummary(report, kpis);
  p.lines([estimate, note].filter(Boolean).join(' '), M, estimate ? 273 : 275, 178, estimate ? 2 : 1, 6.5, C.muted, false, 2.8);
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

function kpiSource(kpi) {
  if (kpi.value == null) return isEstimated(kpi) ? 'Stima N/D' : 'N/D';
  if (isEstimated(kpi)) return kpi.corrected ? 'Stima verificata' : 'Stima da azioni';
  return reportSourceLabel(kpi.corrected ? 'manual' : kpi.source, true);
}

function indicatorPage(doc, p, report, kpis, offset, page) {
  if (page > 1) doc.addPage();
  const items = kpis.slice(offset, offset + 15);
  header(doc, p, report, 'Performance dell’evento', `Indicatori selezionati | ${offset + 1}-${offset + items.length} di ${kpis.length}`);
  p.text('INDICATORE E DESCRIZIONE', M, 72, 7, C.teal, true);
  p.text('VALORE E FONTE', WIDTH - M, 72, 7, C.teal, true, { align: 'right' });
  items.forEach((kpi, index) => {
    const y = 76 + index * 13;
    if (index % 2 === 0) p.box(M, y, 178, 12.6, C.pale, 1);
    p.text(String(offset + index + 1).padStart(2, '0'), M + 2, y + 4.8, 6.5, C.muted);
    p.lines(kpi.label, M + 9, y + 4.8, 124, 1, 8.2, C.ink, true);
    p.fit(formatReportValue(kpi.value, kpi.unit), WIDTH - M - 2, y + 4.8, 38, 11, C.navy, true, 'right');
    if (kpi.display_description) p.lines(kpi.display_description, M + 9, y + 8.6, 127, 2, 7.2, C.muted, false, 3.3);
    const reference = referenceText(kpi);
    if (reference) p.fit(reference, WIDTH - M - 2, y + 8.3, 36, 6.1, C.muted, false, 'right');
    p.fit(kpiSource(kpi), WIDTH - M - 2, y + 11.5, 36, 6.1, C.muted, false, 'right');
  });
  p.text('N/D = misura non disponibile. Descrizioni abbreviate per la leggibilità.', M, 275, 6.4, C.muted);
  footer(doc, p, report, page, 3);
}

function detailPage(doc, p, report, kpis = [], combinedMethod = false) {
  doc.addPage();
  const selected = new Set(kpis.map(kpi => kpi.id));
  function detailColumns(name, metrics) {
    const active = metrics.filter(metric => metric.kpis.some(id => selected.has(id)));
    return [{ key: 'name', label: name, x: 0, width: 62 }, ...active.map((metric, index) => ({ ...metric, x: 62 + index * (116 / active.length), width: 116 / active.length }))];
  }
  const gameColumns = detailColumns('ESPERIENZA', [
    { key: 'players', label: 'GIOCATORI', kpis: ['game_participation', 'replay_rate'] },
    { key: 'started', label: 'AVVIATI', kpis: ['game_participation', 'replay_rate'] },
    { key: 'completed', label: 'COMPLETATI', kpis: ['completed_rounds'] },
  ]);
  const sponsorColumns = detailColumns('PARTNER', [
    { key: 'reach', label: 'COPERTURA', kpis: ['sponsor_reach'] },
    { key: 'impressions', label: 'ESPOSIZIONI', kpis: ['sponsor_impressions'] },
    { key: 'clickers', label: 'UTENTI CLIC', kpis: ['sponsor_ctr'] },
    { key: 'ctr', label: 'TASSO', kpis: ['sponsor_ctr'] },
  ]);
  const hasGameDetails = gameColumns.length > 1;
  const hasSponsorDetails = sponsorColumns.length > 1;
  const hasExperienceDetails = hasGameDetails || hasSponsorDetails;
  const title = hasExperienceDetails ? combinedMethod ? 'Esperienze, partner e metodo' : 'Esperienze e partner' : combinedMethod ? 'Fonti e metodo' : 'Contesto della misurazione';
  header(doc, p, report, title, `Dettaglio delle KPI selezionate | ${kpis.length} indicatori nel report`);
  const games = (report.details?.games || []).map(item => ({ name: item.name || item.id, players: formatReportValue(item.players), started: formatReportValue(item.rounds_started), completed: formatReportValue(item.rounds_completed) }));
  if (hasGameDetails) table(p, 'Giochi e attività', gameColumns, games, 77, 'Nessun dettaglio giochi disponibile nella misurazione.', 8, combinedMethod ? 6.5 : 7.2);
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
  if (hasSponsorDetails) table(p, 'Risultati per sponsor', sponsorColumns, sponsors, hasGameDetails ? combinedMethod ? 156 : 163 : 77, 'Nessun dettaglio sponsor disponibile nella misurazione.', 6, combinedMethod ? 6.5 : 9);
  const operational = selectedOperationalItems(report, kpis);
  if (operational.length) {
    const y = hasExperienceDetails ? combinedMethod ? 222 : 257 : 77;
    p.text('DATI OPERATIVI DELL’EVENTO', M, y, 7.2, C.teal, true);
    p.lines(operational.map(item => `${item.label}: ${formatReportValue(item.value)}`).join(' | '), M, y + 5, 178, combinedMethod ? 2 : 3, 7, C.ink, false, 3.4);
  }
  if (!hasExperienceDetails && !operational.length && !combinedMethod) {
    p.text('COPERTURA DELLE KPI SELEZIONATE', M, 77, 8, C.teal, true);
    p.lines(coverageNotes(report).join(' ') || sourceSummary(report, kpis), M, 84, 178, 6, 8, C.ink, false, 4);
    const estimate = timeEstimateSummary(report, kpis);
    if (estimate) p.lines(estimate, M, 114, 178, 8, 8, C.ink, false, 4);
    if (report.notes) p.lines(`Note: ${report.notes}`, M, 155, 178, 8, 8, C.muted, false, 4);
  }
  if (combinedMethod) {
    p.text('METODO E COPERTURA', M, 238, 7.5, C.teal, true);
    const sources = new Map();
    kpis.forEach(kpi => sources.set(kpiSource(kpi), (sources.get(kpiSource(kpi)) || 0) + 1));
    p.lines(`Fonti delle ${kpis.length} KPI selezionate: ${[...sources].map(([name, count]) => `${name} (${count})`).join('; ')}.`, M, 243, 178, 2, 7, C.ink, false, 3.3);
    const coverage = coverageNotes(report).join(' ') || sourceSummary(report, kpis);
    p.lines(coverage, M, 252, 178, 2, 6.8, C.muted, false, 3.2);
    const estimate = timeEstimateSummary(report, kpis);
    if (estimate) p.lines(estimate, M, 261, 178, 3, 6.8, C.ink, false, 3.2);
    p.lines(report.notes ? `Note: ${report.notes}` : 'Testi e tabelle abbreviati. Dati integrali nel report approvato.', M, 275, 178, 1, 6.3, C.muted);
  }
  footer(doc, p, report, combinedMethod ? 3 : 2, 3);
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
    p.lines(kpiSource(kpi), M + 121, y + 4.5, 54, 1, 6.5, C.muted);
  });
  p.text('COPERTURA DELLE FONTI', M, 237, 8, C.teal, true);
  const sourceCount = report.details?.sources?.length;
  const coverage = coverageNotes(report).join(' ') || 'Le tracce operative descrivono una partecipazione minima. Le visite senza azioni e i pubblici eleggibili richiedono telemetria. Risposte e clic senza identità non misurano persone uniche.';
  const estimate = timeEstimateSummary(report, kpis);
  p.lines(`${sourceSummary(report, kpis)}${sourceCount ? ` ${sourceCount} tabelle consultate.` : ''} ${coverage}`, M, 243, 178, estimate ? 2 : 3, 7.3, C.ink, false, 3.5);
  if (estimate) p.lines(estimate, M, 253, 178, 3, 7, C.ink, false, 3.3);
  p.lines(report.notes ? `Note: ${report.notes}` : 'N/D = misura assente o denominatore nullo. Utenti anonimi deduplicati per browser. Il tempo medio misurato usa solo intervalli validi, pagina visibile e attività recente. Le registrazioni non attestano consenso marketing.', M, estimate ? 266 : 258, 178, estimate ? 2 : 4, 7, C.muted, false, 3.4);
  p.text('Testi e tabelle abbreviati per la leggibilità. Dati integrali conservati nel report approvato.', M, 275, 6.3, C.muted);
  footer(doc, p, report, 3, 3);
}

/** Returns a deterministic A4 document from an immutable server-approved snapshot. */
export function createEngagementReportPdf(report) {
  if (!report?.id || !report?.approved_at || !Array.isArray(report.kpis)) throw new Error('È necessaria una versione approvata del report.');
  if (!['one_page', 'extended'].includes(report.format)) throw new Error('Formato report non supportato.');
  const doc = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4', compress: true, putOnlyUsedFonts: true });
  const p = createPainter(doc), kpis = normalizedKpis(report), total = report.format === 'extended' ? 3 : 1;
  doc.setProperties({ title: pdfText(`Engagement - ${report.event?.title || 'Evento'}`), subject: pdfText(`${kpis.length} KPI - Report approvato ${report.id}`), author: pdfText(report.branding?.company_name || 'ArenaBoostX'), creator: 'ArenaBoostX Event Intelligence', keywords: 'engagement, evento, KPI, sponsor, report' });
  if (kpis.length > 20) {
    indicatorPage(doc, p, report, kpis, 0, 1);
    indicatorPage(doc, p, report, kpis, 15, 2);
    detailPage(doc, p, report, kpis, true);
  } else {
    overview(doc, p, report, kpis, total);
    if (total === 3) { detailPage(doc, p, report, kpis); methodPage(doc, p, report, kpis); }
  }
  return doc;
}

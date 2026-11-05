import { engagementReportFixture } from './engagementReportFixture.js';
import { readFileSync } from 'node:fs';

// Demo/test data uses the real server-owned labels, definitions and groups.
// It never reads a database, environment file or production export.
const catalogSource = readFileSync(new URL('../../wcmvpvs-back/service/api/engagement_report_metrics.go', import.meta.url), 'utf8');
const catalog = new Map(catalogSource.split('\n').filter(line => line.includes('{ID: ') && line.includes('Description: ')).map(line => {
  const fields = {};
  for (const [, key, value] of line.matchAll(/(ID|Label|Group|Description|FactKey|NumeratorKey|DenominatorKey): ("(?:[^"\\]|\\.)*")/g)) fields[key] = JSON.parse(value);
  return [fields.ID, fields];
}));

export const recipientProfiles = [
  { id: 'executive', name: 'Dirigenza', kpi_ids: ['visitors', 'venue_adoption', 'new_profiles', 'event_retention', 'sponsor_reach'], groups: ['Pubblico', 'Acquisizione', 'Fidelizzazione', 'Sponsor'], objective: 'Mostrare il valore per la società sportiva', layout: 'executive', max_insights: 3, allow_appendix: true },
  { id: 'marketing', name: 'Marketing', kpi_ids: ['visitors', 'active_participation', 'new_profiles', 'registration_conversion', 'event_retention', 'active_time', 'mvp_participation', 'multi_experience', 'feedback_response', 'sponsor_reach'], groups: ['Pubblico', 'Acquisizione', 'Engagement', 'Fidelizzazione', 'Feedback', 'Sponsor'], objective: 'Comprendere comportamento e opportunità', layout: 'marketing', max_insights: 3, allow_appendix: true },
  { id: 'sponsor', name: 'Sponsor', kpi_ids: ['sponsor_reach', 'sponsor_impressions', 'sponsor_ctr', 'sponsored_participation', 'visitors'], groups: ['Sponsor', 'Pubblico'], objective: 'Documentare visibilità e interazioni misurabili', layout: 'sponsor', max_insights: 3, allow_appendix: true },
  { id: 'analytics', name: 'Analitico / completo', kpi_ids: engagementReportFixture().kpis.slice(0, 20).map(k => k.id), groups: ['Pubblico', 'Engagement', 'Gaming', 'Acquisizione', 'Fidelizzazione', 'Sponsor', 'Feedback'], objective: 'Consentire una lettura tecnica verificabile', layout: 'analytics', max_insights: 3, allow_appendix: false },
].map(p => ({ ...p, description: p.objective, tone: 'Professionale', ai_instructions: 'Usare solo dati disponibili.' }));

export function recipientReportFixture(profile = 'executive', appendix = false, stress = false) {
  const r = engagementReportFixture(profile === 'analytics' || profile === 'marketing' ? 'extended' : 'one_page');
  Object.assign(r, { id: 'abcdef1234567890abcdef1234567890abcdef', calculation_id: '1234567890abcdef1234567890abcdef123456', schema_version: 2, profiles: recipientProfiles, recipient: recipientProfiles.find(p => p.id === profile), include_appendix: appendix, compare_previous: true, data_hash: 'synthetic-data-hash', analysis_status: 'fallback', previous_event: { ...r.event, id: 86, start_datetime: '2026-10-16T18:30:00Z' }, overrides: {}, notes: 'ESEMPIO DIMOSTRATIVO. Dati sintetici, nessun dato di produzione.' });
  r.selected_kpi_ids = r.recipient.kpi_ids;
  r.manual_sources = [{ event_id: r.event.id, field_key: 'attendees', value: 1800, unit: 'people', source_type: 'official_attendance', source_description: 'Distinta ufficiale dimostrativa', verification_status: 'verified', revision: 1, entered_by: 77, entered_at: r.calculated_at, updated_at: r.calculated_at, note: '' }];
  r.kpis.forEach(k => {
    const definition = catalog.get(k.id);
    Object.assign(k, { label: definition.Label, group: definition.Group, description: definition.Description, fact_key: definition.FactKey, numerator_key: definition.NumeratorKey, denominator_key: definition.DenominatorKey });
    k.source = 'new_tracking'; k.corrected = false;
    k.availability = { reason: 'available', message: 'Dato disponibile.', manual_allowed: false, fields: [], dependencies: [k.numeratorKey, k.denominatorKey].filter(Boolean) };
    const previous = k.value == null ? null : k.unit === 'count' ? Math.round(k.value * .9) : k.value * .9;
    const absolute = k.value == null ? null : k.value - previous;
    k.comparison = { previous, absolute, percent: k.unit === 'count' && previous > 0 ? absolute / previous * 100 : null, percentage_points: k.unit === 'percent' ? absolute : null, direction: 'up' };
    if (k.id === 'venue_adoption') { k.provenance = r.manual_sources; k.source = 'manual_verified'; }
  });
  r.analysis = { headline: { executive: 'Partecipazione e valore per la società', marketing: 'Acquisizione e comportamento dei tifosi', sponsor: 'Visibilità e interazioni misurabili', analytics: 'Qualità e limiti della misurazione' }[profile], summary: 'Esempio con dati sintetici. I risultati descrivono il pubblico osservabile e le attività documentate. Le differenze tra eventi non provano cause o benefici economici.', positive_insights: [{ title: 'Pubblico osservabile', description: 'La partecipazione documentata costituisce una base per leggere il valore dell’evento.', metric_ids: ['visitors'] }], critical_insights: [], recommendations: [{ action: 'Consolidare la misurazione tra eventi', reason: 'La continuità della raccolta rende più utile il confronto.', metric_ids: ['visitors'], priority: 'medium' }], limitations: ['Esempio dimostrativo, senza chiamate OpenAI. N/D non equivale a zero.', 'Profili e dispositivi osservati non certificano persone fisiche o consenso marketing.'] };
  r.details.coverage_notes = ['Gli indicatori si riferiscono alle identità osservate e non a tutti gli spettatori.', 'La provenienza arena non dimostra da sola la presenza fisica. Le presenze ufficiali provengono dalla fonte manuale dichiarata.'];
  if (stress) {
    r.notes = 'Nota molto lunga da mantenere leggibile e senza sovrapposizioni nel report. '.repeat(55);
    r.kpis.forEach(k => { k.description = `${k.description} ${'Definizione e limiti metodologici di misurazione. '.repeat(20)}`; });
    r.analysis.summary = 'Interpretazione estesa dei risultati disponibili, senza attribuire cause. '.repeat(8);
  }
  return r;
}

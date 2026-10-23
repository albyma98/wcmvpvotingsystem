// This schema is shared conceptually with the independent /new report calculator.
// A missing measurement stays null, including ratios with no eligible audience.
export const REPORT_KPIS = [
  ['visitors', 'Visitatori unici', 'Pubblico e attenzione', 'count', 'visitors'],
  ['venue_adoption', 'Adozione sul pubblico presente', 'Pubblico e attenzione', 'percent', 'onsite_visitors', 'attendees'],
  ['active_participation', 'Partecipazione attiva', 'Pubblico e attenzione', 'percent', 'active_visitors', 'engagement_eligible_visitors'],
  ['active_time', 'Tempo attivo medio misurato', 'Pubblico e attenzione', 'seconds', 'active_seconds', 'timed_visitors'],
  ['mvp_participation', 'Partecipazione al voto MVP', 'Esperienze e giochi', 'percent', 'voters', 'vote_eligible_visitors'],
  ['game_participation', 'Partecipazione ai giochi', 'Esperienze e giochi', 'percent', 'players', 'game_eligible_visitors'],
  ['completed_rounds', 'Round di gioco completati', 'Esperienze e giochi', 'count', 'rounds_completed'],
  ['replay_rate', 'Tasso di rigiocata', 'Esperienze e giochi', 'percent', 'repeat_players', 'players'],
  ['multi_experience', 'Partecipazione a più esperienze', 'Esperienze e giochi', 'percent', 'multi_experience_visitors', 'active_visitors'],
  ['new_profiles', 'Nuovi profili verificati', 'Relazione con i tifosi', 'count', 'new_profiles'],
  ['registration_conversion', 'Conversione alla registrazione', 'Relazione con i tifosi', 'percent', 'new_profiles', 'registration_starters'],
  ['event_retention', 'Ritorno tra eventi', 'Relazione con i tifosi', 'percent', 'returning_profiles', 'previous_event_profiles'],
  ['redemption_conversion', 'Riscatto di collezionabili', 'Relazione con i tifosi', 'percent', 'redeemers', 'shop_eligible_profiles'],
  ['sponsor_reach', 'Copertura unica sponsor', 'Valore per i partner', 'count', 'sponsor_reach'],
  ['sponsor_impressions', 'Esposizioni visibili sponsor', 'Valore per i partner', 'count', 'sponsor_impressions'],
  ['sponsor_ctr', 'Tasso di clic degli esposti', 'Valore per i partner', 'percent', 'sponsor_clickers', 'sponsor_reach'],
  ['sponsored_participation', 'Esperienze sponsorizzate', 'Valore per i partner', 'percent', 'sponsored_players', 'sponsored_exposed_visitors'],
  ['feedback_response', 'Risposta al feedback', 'Ascolto e soddisfazione', 'percent', 'feedback_respondents', 'feedback_eligible_visitors'],
  ['satisfaction', 'Soddisfazione esperienza', 'Ascolto e soddisfazione', 'percent', 'experience_positive', 'experience_responses'],
  ['perks_interest', 'Interesse per premi e vantaggi', 'Ascolto e soddisfazione', 'percent', 'perks_positive', 'perks_responses'],
].map(([id, label, group, unit, numeratorKey, denominatorKey]) => ({ id, label, group, unit, numeratorKey, denominatorKey }));

export const REPORT_FACT_LABELS = {
  visitors: 'Visitatori unici', onsite_visitors: 'Visitatori attribuiti all’arena', attendees: 'Spettatori presenti (dato della società)',
  active_visitors: 'Visitatori attivi', active_seconds: 'Secondi attivi complessivi', voters: 'Votanti unici',
  engagement_eligible_visitors: 'Pubblico osservabile con rilevamento delle visite', timed_visitors: 'Utenti con intervalli di tempo attivo misurati',
  vote_eligible_visitors: 'Visitatori con voto disponibile', players: 'Giocatori unici', game_eligible_visitors: 'Visitatori con giochi disponibili',
  rounds_completed: 'Round completati', repeat_players: 'Giocatori con almeno due partite', multi_experience_visitors: 'Utenti attivi in almeno due aree',
  new_profiles: 'Nuovi profili verificati', registration_starters: 'Ospiti che iniziano la registrazione', returning_profiles: 'Profili tornati dall’evento precedente',
  previous_event_profiles: 'Profili attivi nell’evento precedente', redeemers: 'Profili con riscatto confermato', shop_eligible_profiles: 'Profili attivi con shop disponibile',
  sponsor_reach: 'Utenti esposti agli sponsor', sponsor_impressions: 'Esposizioni sponsor visibili', sponsor_clickers: 'Utenti esposti che cliccano sponsor',
  sponsored_players: 'Utenti delle esperienze sponsorizzate', sponsored_exposed_visitors: 'Utenti esposti alle esperienze sponsorizzate',
  feedback_respondents: 'Partecipanti al feedback', feedback_eligible_visitors: 'Utenti con feedback disponibile',
  experience_positive: 'Risposte positive sull’esperienza', experience_responses: 'Risposte valide sull’esperienza',
  perks_positive: 'Risposte positive sui vantaggi', perks_responses: 'Risposte valide sui vantaggi',
};

export function reportSourceLabel(value, compact = false) {
  const labels = compact ? {
    new_tracking: 'Telemetria /new', operational: 'Dati operativi', mixed: 'Telemetria + operativi',
    manual: 'Verificato / corretto', unavailable: 'N/D',
  } : {
    new_tracking: 'Misurazione dedicata home /new', operational: 'Dati operativi dell’evento', mixed: 'Misurazione /new e dati operativi riconciliati',
    manual: 'Inserimento manuale verificato', unavailable: 'Misura non disponibile',
  };
  return labels[value] || labels.unavailable;
}

export const REPORT_OPERATIONAL_LABELS = {
  votes: 'Voti registrati', identified_participants: 'Profili identificati attivi',
  game_reward_receipts: 'Premi di gioco confermati', shop_redemptions: 'Riscatti shop confermati',
  coins_earned: 'Coin guadagnati', coins_spent: 'Coin spesi', feedback_responses: 'Risposte al feedback',
  sponsor_clicks: 'Clic sponsor registrati', sponsor_clicks_without_identity: 'Clic sponsor senza identità',
};

export function reportOperationalItems(details = {}) {
  return Object.entries(REPORT_OPERATIONAL_LABELS)
    .map(([key, label]) => ({ key, label, value: numericReportValue(details?.operational?.[key]) }))
    .filter(item => item.value !== null);
}

export function numericReportValue(value) {
  if (value === '' || value === null || value === undefined) return null;
  const result = Number(value);
  return Number.isFinite(result) ? result : null;
}

function reportKpiDefinition(schema, original = {}) {
  return {
    ...schema,
    numeratorKey: original.fact_key || original.numerator_key || schema.numeratorKey,
    denominatorKey: original.denominator_key || schema.denominatorKey,
  };
}

export function calculateReportKpis(facts = {}, originalKpis = [], overrides = {}, originalFacts = {}, descriptions = {}) {
  return REPORT_KPIS.map((schema) => {
    const original = originalKpis.find((item) => item.id === schema.id) || {};
    const definition = reportKpiDefinition(schema, original);
    const numerator = numericReportValue(facts[definition.numeratorKey]);
    const denominator = definition.denominatorKey ? numericReportValue(facts[definition.denominatorKey]) : undefined;
    const calculated = definition.denominatorKey
      ? numerator !== null && denominator !== null && denominator > 0 ? numerator / denominator * (definition.unit === 'percent' ? 100 : 1) : null
      : numerator;
    const override = overrides[definition.id];
    const originalValue = numericReportValue(original.value);
    const calculationChanged = originalValue === null || calculated === null
      ? originalValue !== calculated
      : Math.abs(originalValue - calculated) > 0.005;
    const factChanged = (key, originalKpiField) => Object.hasOwn(originalFacts, key)
      ? numericReportValue(facts[key]) !== numericReportValue(originalFacts[key])
      : Object.hasOwn(original, originalKpiField) && numericReportValue(facts[key]) !== numericReportValue(original[originalKpiField]);
    const baseChanged = definition.denominatorKey && (factChanged(definition.numeratorKey, 'numerator') || factChanged(definition.denominatorKey, 'denominator'));
    const corrected = Boolean(override) || calculationChanged || Boolean(baseChanged);
    return {
      ...definition, ...original, numeratorKey: definition.numeratorKey, denominatorKey: definition.denominatorKey,
      numerator: override ? null : numerator, denominator: override ? null : denominator,
      calculated_numerator: numerator, calculated_denominator: denominator,
      measured_value: original.value ?? null,
      display_description: Object.hasOwn(descriptions, definition.id) ? descriptions[definition.id] : original.display_description || '',
      calculated_value: calculated,
      value: override ? numericReportValue(override.value) : calculated,
      correction_note: override?.note || '',
      corrected,
      source: corrected ? 'manual' : original.source,
    };
  });
}

export function validateReportReview(facts, overrides, notes = '', originalKpis = [], descriptions = {}) {
  const errors = [];
  for (const [key, value] of Object.entries(facts || {})) {
    if (value === null || value === '') continue;
    const number = Number(value);
    if (!Number.isFinite(number) || number < 0 || !Number.isInteger(number)) errors.push(`${REPORT_FACT_LABELS[key] || key}: inserisci un intero non negativo o lascia vuoto.`);
  }
  const visitors = numericReportValue(facts?.visitors);
  const activeVisitors = numericReportValue(facts?.active_visitors);
  if (visitors !== null && activeVisitors !== null && activeVisitors > visitors) errors.push('I partecipanti attivi non possono superare i partecipanti digitali osservabili.');
  for (const schema of REPORT_KPIS) {
    const kpi = reportKpiDefinition(schema, originalKpis.find(item => item.id === schema.id));
    const numerator = numericReportValue(facts?.[kpi.numeratorKey]);
    const denominator = numericReportValue(facts?.[kpi.denominatorKey]);
    if (kpi.unit === 'percent' && numerator !== null && denominator !== null && numerator > denominator) {
      errors.push(`${kpi.label}: il numeratore supera il pubblico di riferimento.`);
    }
    const override = overrides?.[kpi.id];
    if (!override) continue;
    const value = numericReportValue(override.value);
    if (override.value !== null && override.value !== '' && (value === null || value < 0 || (kpi.unit === 'percent' && value > 100) || (kpi.unit === 'count' && !Number.isInteger(value)))) errors.push(`${kpi.label}: valore non valido.`);
  }
  for (const schema of REPORT_KPIS) {
    const description = descriptions[schema.id];
    if (description !== undefined && (typeof description !== 'string' || Array.from(description).length > 120)) errors.push(`${schema.label}: la breve descrizione non può superare 120 caratteri.`);
  }
  if (String(notes).length > 4000) errors.push('Le note del report non possono superare 4.000 caratteri.');
  return errors;
}

export function formatReportValue(value, unit = 'count') {
  const number = numericReportValue(value);
  if (number === null) return 'N/D';
  const formatted = new Intl.NumberFormat('it-IT', { maximumFractionDigits: unit === 'percent' ? 1 : 0 }).format(number);
  return `${formatted}${unit === 'percent' ? '%' : unit === 'seconds' ? ' s' : ''}`;
}

export function reportDownloadFilename(snapshot) {
  const title = String(snapshot.event?.title || 'evento').normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 70);
  return `arenaboostx_${title}_${snapshot.format === 'extended' ? 'completo' : 'one-pager'}_${snapshot.id || snapshot.event?.id || 'report'}.pdf`;
}

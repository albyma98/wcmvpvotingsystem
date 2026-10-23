import { calculateReportKpis } from '../src/engagementReport.js';

export function engagementReportFixture(format = 'one_page', stress = false) {
  const facts = {
    visitors: 1240, onsite_visitors: 960, attendees: 1800, active_visitors: 897, active_seconds: 281480,
    voters: 712, vote_eligible_visitors: 1140, players: 534, game_eligible_visitors: 1120, rounds_completed: 1894,
    repeat_players: 322, multi_experience_visitors: 492, new_profiles: 187, registration_starters: 278,
    returning_profiles: 204, previous_event_profiles: 580, redeemers: 96, shop_eligible_profiles: 420,
    sponsor_reach: 1103, sponsor_impressions: 5340, sponsor_clickers: 149, sponsored_players: 340, sponsored_exposed_visitors: 780,
    feedback_respondents: 282, feedback_eligible_visitors: 880, experience_positive: 231, experience_responses: 282, perks_positive: 198, perks_responses: 282,
  };
  const overrides = { visitors: { value: 1240, note: 'Deduplicazione verificata con il responsabile della società.' } };
  const kpis = calculateReportKpis(facts, [], overrides).map(kpi => ({ ...kpi, source: 'new_tracking', corrected: Boolean(overrides[kpi.id]) }));
  if (stress) {
    kpis.forEach(kpi => { kpi.value = kpi.unit === 'percent' ? 100 : 999999999999999; kpi.numerator = 999999999999999; kpi.denominator = 999999999999999; });
    kpis[3].value = null;
  }
  const name = stress ? 'Società sportiva internazionale con un nome estremamente lungo '.repeat(40) : 'Pallacanestro Aurora';
  return {
    id: 42, calculation_id: 'fixture-42', format, approved_at: '2026-10-23T12:00:00Z', calculated_at: '2026-10-23T11:50:00Z', measurement_started_at: '2026-10-23T08:00:00Z',
    event: { id: 87, title: stress ? `${name} - derby città 🏀` : 'Aurora vs Riviera - Una serata da protagonisti', start_datetime: '2026-10-23T18:30:00Z', organization_name: name, location: 'PalaAurora · Bologna' },
    branding: { company_name: stress ? 'ArenaBoostX '.repeat(50) : 'ArenaBoostX', logo_data: '' }, facts, overrides, kpis,
    notes: stress ? 'Verifica dei dati, acquisizione e riconciliazione dei risultati dell’evento. '.repeat(70) : 'Dati verificati dalla società. La campagna in arena ha supportato voto MVP, giochi e attività dei partner. Le presenze provengono dal conteggio della società.',
    details: {
      games: Array.from({ length: stress ? 20 : 8 }, (_, i) => ({ id: `game-${i}`, name: stress ? name : ['Basketball Shootout', 'Quiz della squadra', 'Memory Flash', 'Sponsor Rush', 'Crossover Rush', 'Playmaker', 'Tap Challenge', 'Reaction Test'][i], players: 240 - i * 15, rounds_started: 740 - i * 30, rounds_completed: 620 - i * 25 })),
      sponsors: Array.from({ length: stress ? 14 : 3 }, (_, i) => ({ id: i + 1, name: stress ? name : ['Partner principale', 'Sponsor del match', 'Partner territorio'][i], reach: 1000 - i * 100, impressions: 2400 - i * 240, clickers: 100 - i * 20, ctr: 10 - i })),
    },
  };
}

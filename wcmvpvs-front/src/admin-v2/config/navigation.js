export const navigation = [
 { label: 'Centro operativo', items: [
  { id: 'dashboard', label: 'Panoramica', description: 'La tua società, a colpo d’occhio.' },
  { id: 'events', label: 'Eventi', description: 'Partite, esperienze e contenuti per il tuo pubblico.' },
  { id: 'closing', label: 'Regia votazioni', description: 'Gestisci la chiusura MVP, i premi e le notifiche.' },
  { id: 'bar', label: 'Bar e ordini', description: 'Il servizio in arena, dal catalogo alla consegna.' },
 ]},
 { label: 'Pubblico e partner', items: [
  { id: 'sponsors', label: 'Sponsor', description: 'Visibilità, creatività e ordine di pubblicazione.' },
  { id: 'coupons', label: 'Coupon e vantaggi', description: 'Promozioni per i fan, collegate alle tue partite.' },
  { id: 'selfies', label: 'Selfie MVP', description: 'I momenti del pubblico e la loro moderazione.' },
  { id: 'marketing', label: 'Marketing', description: 'Audience, campagne SMS, template e invii.' },
 ]},
 { label: 'Analisi', items: [
  { id: 'results', label: 'Risultati live', description: 'Classifica MVP e misurazione degli sponsor.' },
  { id: 'history', label: 'Archivio e report', description: 'Eventi conclusi, analisi e report da condividere.' },
 ]},
 { label: 'Società', items: [
  { id: 'teams', label: 'Squadre', description: 'Le squadre che partecipano ai tuoi eventi.' },
  { id: 'players', label: 'Roster giocatori', description: 'Profili, immagini e convocazioni.' },
  { id: 'partners', label: 'Accessi partner', description: 'Account e credenziali dei tuoi partner.' },
  { id: 'admins', label: 'Amministratori', description: 'Account e ruoli della società.' },
 ]},
];
export const sectionMeta = Object.fromEntries(navigation.flatMap(g => g.items.map(i => [i.id, { ...i, group: g.label }])));
sectionMeta.settings = { id: 'settings', label: 'Votazioni e notifiche', group: 'Società', description: 'Organizzatori, premi e comunicazioni WhatsApp.' };
sectionMeta.lottery = { id: 'lottery', label: 'Lotteria e vincitori', group: 'Centro operativo', description: 'Ticket ammessi, estrazione dei premi e assegnazione dei vincitori.' };

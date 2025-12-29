import AdminEventsView from '../views/AdminEventsView.vue';
import AdminPlayersView from '../views/AdminPlayersView.vue';
import AdminHistoryView from '../views/AdminHistoryView.vue';
import AdminPlaceholderView from '../views/AdminPlaceholderView.vue';
import PublicVoteView from '../views/PublicVoteView.vue';
import CashLanding from '../components/CashLanding.vue';
import TicketValidationView from '../components/TicketValidationView.vue';

export const routes = [
  { path: '/', redirect: '/admin/events' },
  { path: '/vote', name: 'public-vote', component: PublicVoteView },
  { path: '/welcome', name: 'welcome', component: CashLanding },
  { path: '/lottery/validate', name: 'ticket-validation', component: TicketValidationView },
  { path: '/admin/events', name: 'admin-events', component: AdminEventsView },
  { path: '/admin/players', name: 'admin-players', component: AdminPlayersView },
  { path: '/admin/history', name: 'admin-history', component: AdminHistoryView },
  {
    path: '/admin/live/event',
    name: 'admin-live-event',
    component: AdminPlaceholderView,
    meta: { title: 'Evento live', description: 'Monitor live dell\'evento attivo.' },
  },
  {
    path: '/admin/live/votes',
    name: 'admin-live-votes',
    component: AdminPlaceholderView,
    meta: { title: 'Votazioni live', description: 'Tracking voti in tempo reale.' },
  },
  {
    path: '/admin/live/lottery',
    name: 'admin-live-lottery',
    component: AdminPlaceholderView,
    meta: { title: 'Lotteria live', description: 'Gestione estrazioni e premi.' },
  },
  {
    path: '/admin/live/selfie',
    name: 'admin-live-selfie',
    component: AdminPlaceholderView,
    meta: { title: 'Selfie MVP', description: 'Moderazione e approvazione selfie.' },
  },
  {
    path: '/admin/teams',
    name: 'admin-teams',
    component: AdminPlaceholderView,
    meta: { title: 'Squadre', description: 'Gestione roster squadre.' },
  },
  {
    path: '/admin/sponsors',
    name: 'admin-sponsors',
    component: AdminPlaceholderView,
    meta: { title: 'Sponsor', description: 'Gestione inventory e click tracking.' },
  },
  {
    path: '/admin/coupons',
    name: 'admin-coupons',
    component: AdminPlaceholderView,
    meta: { title: 'Coupon', description: 'Gestione coupon e conversioni.' },
  },
  {
    path: '/admin/results',
    name: 'admin-results',
    component: AdminPlaceholderView,
    meta: { title: 'Risultati', description: 'Analisi risultati live.' },
  },
  {
    path: '/admin/reports',
    name: 'admin-reports',
    component: AdminPlaceholderView,
    meta: { title: 'Reportistica', description: 'Report esportabili e KPI.' },
  },
  {
    path: '/admin/admins',
    name: 'admin-admins',
    component: AdminPlaceholderView,
    meta: { title: 'Gestione admin', description: 'Ruoli e permessi di sistema.' },
  },
  {
    path: '/admin/logout',
    name: 'admin-logout',
    component: AdminPlaceholderView,
    meta: { title: 'Logout', description: 'Disconnessione sicura dalla console.' },
  },
  { path: '*', redirect: '/admin/events' },
];

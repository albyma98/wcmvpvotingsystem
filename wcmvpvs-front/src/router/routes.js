import PublicVoteView from '../views/PublicVoteView.vue';
import CashLanding from '../components/CashLanding.vue';
import TicketValidationView from '../components/TicketValidationView.vue';
import AdminPortal from '../components/AdminPortal.vue';
import MasterPortal from '../components/MasterPortal.vue';
import PartnerPortal from '../components/PartnerPortal.vue';

export const routes = [
  {
    path: '/',
    redirect: (location) => (location?.organizationSlug ? '/admin' : '/vote'),
  },
  { path: '/vote', name: 'public-vote', component: PublicVoteView },
  { path: '/welcome', name: 'welcome', component: CashLanding },
  { path: '/lottery/validate', name: 'ticket-validation', component: TicketValidationView },
  {
    path: '/partner',
    name: 'partner-portal',
    component: PartnerPortal,
    props: (route) => ({
      currentPath: route?.location?.rawPath || route?.fullPath || '/',
      currentSearch: route?.location?.search || '',
    }),
  },
  {
    path: '/admin',
    name: 'admin-portal',
    component: AdminPortal,
    props: (route) => ({ organizationSlug: route?.slug || '' }),
  },
  { path: '/admin/events', redirect: '/admin' },
  { path: '/admin/players', redirect: '/admin' },
  { path: '/admin/history', redirect: '/admin' },
  {
    path: '/admin/live/event',
    redirect: '/admin',
  },
  {
    path: '/admin/live/votes',
    redirect: '/admin',
  },
  {
    path: '/admin/live/lottery',
    redirect: '/admin',
  },
  {
    path: '/admin/live/selfie',
    redirect: '/admin',
  },
  {
    path: '/admin/teams',
    redirect: '/admin',
  },
  {
    path: '/admin/sponsors',
    redirect: '/admin',
  },
  {
    path: '/admin/coupons',
    redirect: '/admin',
  },
  {
    path: '/admin/results',
    redirect: '/admin',
  },
  {
    path: '/admin/reports',
    redirect: '/admin',
  },
  {
    path: '/admin/admins',
    redirect: '/admin',
  },
  {
    path: '/admin/logout',
    redirect: '/admin',
  },
  {
    path: '/master',
    name: 'master-portal',
    component: MasterPortal,
  },
  { path: '*', redirect: '/admin' },
];

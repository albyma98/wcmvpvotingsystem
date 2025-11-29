import { createRouter, createWebHistory } from 'vue-router';
import AppLayout from '@/layout/AppLayout.vue';
import AdminPortal from '@/components/AdminPortal.vue';
import AdminLottery from '@/components/AdminLottery.vue';
import ShopAdminPortal from '@/components/shop/ShopAdminPortal.vue';
import TicketValidationView from '@/components/TicketValidationView.vue';
import LegacyApp from '@/views/LegacyApp.vue';

const baseAdminChildren = [
  {
    path: 'master',
    name: 'admin-master',
    component: AdminPortal,
    meta: { section: 'dashboard' },
  },
  {
    path: 'events',
    name: 'admin-events',
    component: AdminPortal,
    meta: { section: 'events' },
  },
  {
    path: 'shop',
    name: 'admin-shop',
    component: AdminPortal,
    meta: { section: 'shop' },
  },
  {
    path: 'sponsors',
    name: 'admin-sponsors',
    component: AdminPortal,
    meta: { section: 'sponsors' },
  },
  {
    path: 'lottery',
    name: 'admin-lottery',
    component: AdminLottery,
  },
  {
    path: 'validation',
    name: 'admin-validation',
    component: TicketValidationView,
  },
].map((route) => ({
  ...route,
  props: (r) => ({ organizationSlug: r.params.organization || '' }),
}));

const buildAdminChildren = (withOrganization = false) => [
  {
    path: '',
    redirect: (to) =>
      withOrganization && to.params.organization
        ? `/${to.params.organization}/admin/master`
        : '/admin/master',
  },
  ...baseAdminChildren,
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/admin',
      component: AppLayout,
      children: buildAdminChildren(),
    },
    {
      path: '/:organization/admin',
      component: AppLayout,
      children: buildAdminChildren(true),
      props: { default: true },
    },
    {
      path: '/shop/admin',
      component: AppLayout,
      children: [
        {
          path: '',
          name: 'shop-admin',
          component: ShopAdminPortal,
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'legacy-app',
      component: LegacyApp,
    },
  ],
});

export default router;

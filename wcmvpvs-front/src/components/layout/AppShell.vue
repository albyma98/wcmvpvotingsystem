<template>
  <div class="shell" :class="{ 'shell--collapsed': collapsed && !isMobile }">
    <TopBar
      :company-name="companyName"
      :organization="organization"
      :active-event="activeEvent"
      :user-name="userName"
      :user-role="userRole"
      @toggle-sidebar="toggleSidebar"
    />
    <div class="shell__body">
      <SideNav
        :sections="navSections"
        :collapsed="collapsed && !isMobile"
        :overlay="isMobile"
        :visible="isMobile ? sidebarOpen : true"
        @navigate="handleNavigate"
        @close="closeSidebar"
      />
      <main class="shell__content">
        <slot />
      </main>
    </div>
    <div v-if="isMobile && sidebarOpen" class="shell__overlay" @click="closeSidebar"></div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { useRouter } from '../../router';
import TopBar from './TopBar.vue';
import SideNav from './SideNav.vue';

const router = useRouter();

const sidebarOpen = ref(false);
const collapsed = ref(false);
const isMobile = ref(false);

const companyName = 'WCMVP Control';
const organization = 'Sky Volley Group';
const activeEvent = 'Final Eight - Gara 3';
const userName = 'Alessia Conti';
const userRole = 'Admin di sistema';

const navSections = [
  {
    label: 'LIVE',
    items: [
      { label: 'Evento attivo', to: '/admin/live/event', icon: '⚡' },
      { label: 'Votazioni', to: '/admin/live/votes', icon: '🗳️' },
      { label: 'Lotteria', to: '/admin/live/lottery', icon: '🎟️' },
      { label: 'Selfie', to: '/admin/live/selfie', icon: '🤳' },
    ],
  },
  {
    label: 'SETUP',
    items: [
      { label: 'Eventi', to: '/admin/events', icon: '📅' },
      { label: 'Squadre', to: '/admin/teams', icon: '👥' },
      { label: 'Giocatori', to: '/admin/players', icon: '🏐' },
      { label: 'Sponsor', to: '/admin/sponsors', icon: '💎' },
      { label: 'Coupon', to: '/admin/coupons', icon: '🎫' },
    ],
  },
  {
    label: 'ANALYTICS',
    items: [
      { label: 'Risultati', to: '/admin/results', icon: '📊' },
      { label: 'Storico', to: '/admin/history', icon: '🕑' },
      { label: 'Report', to: '/admin/reports', icon: '📑' },
    ],
  },
  {
    label: 'SYSTEM',
    items: [
      { label: 'Admin', to: '/admin/admins', icon: '🛠️' },
      { label: 'Logout', to: '/admin/logout', icon: '⏻' },
    ],
  },
];

function updateViewport() {
  const mobile = typeof window !== 'undefined' ? window.innerWidth < 992 : false;
  isMobile.value = mobile;
  if (!mobile) {
    sidebarOpen.value = true;
  }
}

function toggleSidebar() {
  if (isMobile.value) {
    sidebarOpen.value = !sidebarOpen.value;
  } else {
    collapsed.value = !collapsed.value;
  }
}

function closeSidebar() {
  if (isMobile.value) {
    sidebarOpen.value = false;
  }
}

function handleNavigate(path) {
  router.push(path);
  if (isMobile.value) {
    sidebarOpen.value = false;
  }
}

onMounted(() => {
  updateViewport();
  sidebarOpen.value = !isMobile.value;
  if (typeof window !== 'undefined') {
    window.addEventListener('resize', updateViewport, { passive: true });
  }
});

onBeforeUnmount(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', updateViewport);
  }
});
</script>

<style scoped>
.shell {
  min-height: 100vh;
  background: radial-gradient(circle at 20% 20%, rgba(37, 99, 235, 0.08), transparent 30%),
    radial-gradient(circle at 80% 0%, rgba(16, 185, 129, 0.08), transparent 28%),
    linear-gradient(180deg, #0b1220, #0b1220);
  color: var(--text-primary);
}

.shell__body {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0;
}

.shell__content {
  position: relative;
  padding: 1.25rem 1.5rem 2.5rem;
}

.shell__overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  z-index: 20;
}

@media (max-width: 1024px) {
  .shell__body {
    grid-template-columns: 1fr;
  }

  .shell__content {
    padding: 1rem;
  }
}
</style>

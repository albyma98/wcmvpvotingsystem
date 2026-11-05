<template>
  <div class="admin-layout" :class="{ 'sidebar-open': mobileOpen, 'sidebar-collapsed': collapsed }">
    <a class="admin-layout__skip" href="#admin-main-content">Vai al contenuto</a>
    <aside
      id="admin-navigation"
      ref="sidebar"
      class="admin-layout__sidebar"
      :role="drawerOpen ? 'dialog' : undefined"
      :aria-modal="drawerOpen ? true : undefined"
      :aria-label="drawerOpen ? 'Menu di navigazione' : undefined"
      :inert="isMobile && !mobileOpen"
      @keydown="handleDrawerKeydown"
    >
      <button class="admin-layout__close" type="button" aria-label="Chiudi menu" @click="emit('close-mobile')">
        <AdminIcon name="close" :size="19" />
      </button>
      <slot name="sidebar" />
    </aside>
    <div class="admin-layout__body" :inert="drawerOpen">
      <slot name="header" />
      <main id="admin-main-content" class="admin-layout__content" tabindex="-1">
        <slot />
      </main>
    </div>
    <button
      v-if="mobileOpen"
      class="admin-layout__overlay"
      type="button"
      aria-label="Chiudi menu"
      tabindex="-1"
      @click="emit('close-mobile')"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import AdminIcon from './ui/AdminIcon.vue';

const props = defineProps({
  mobileOpen: { type: Boolean, default: false },
  collapsed: { type: Boolean, default: false },
});
const emit = defineEmits(['close-mobile']);
const sidebar = ref(null);
const isMobile = ref(false);
const drawerOpen = computed(() => props.mobileOpen && isMobile.value);
let viewport;
let previousFocus;
let previousOverflow;
let disposed = false;

function focusableElements() {
  return [...(sidebar.value?.querySelectorAll('button:not(:disabled), a[href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])') || [])]
    .filter(element => element.getClientRects().length && !element.closest('[inert]'));
}

function handleDrawerKeydown(event) {
  if (!drawerOpen.value) return;
  if (event.key === 'Escape') {
    event.preventDefault();
    emit('close-mobile');
    return;
  }
  if (event.key !== 'Tab') return;
  const elements = focusableElements();
  const first = elements[0];
  const last = elements.at(-1);
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last?.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first?.focus();
  }
}

function releaseDrawer() {
  if (previousOverflow !== undefined) {
    document.body.style.overflow = previousOverflow;
    previousOverflow = undefined;
  }
  const target = previousFocus;
  previousFocus = undefined;
  nextTick(() => { if ((disposed || !drawerOpen.value) && target?.isConnected) target.focus(); });
}

watch(drawerOpen, async open => {
  if (typeof document === 'undefined') return;
  if (!open) { releaseDrawer(); return; }
  previousFocus = document.activeElement;
  previousOverflow = document.body.style.overflow;
  document.body.style.overflow = 'hidden';
  await nextTick();
  if (drawerOpen.value) focusableElements()[0]?.focus();
});

function updateViewport() {
  isMobile.value = viewport.matches;
  if (!isMobile.value && props.mobileOpen) emit('close-mobile');
}
onMounted(() => {
  viewport = window.matchMedia('(max-width: 1023px)');
  updateViewport();
  viewport.addEventListener('change', updateViewport);
});
onBeforeUnmount(() => {
  disposed = true;
  viewport?.removeEventListener('change', updateViewport);
  if (typeof document !== 'undefined') releaseDrawer();
});
</script>

<style scoped>
.admin-layout {
  display: grid;
  grid-template-columns: 252px minmax(0, 1fr);
  min-height: 100vh;
  background: var(--bg-base, #f6f7fb);
  transition: grid-template-columns 180ms ease;
}
.admin-layout.sidebar-collapsed { grid-template-columns: 80px minmax(0, 1fr); }
.admin-layout__sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
  height: 100dvh;
  min-width: 0;
  overflow: hidden;
  background: #111827;
  z-index: 45;
}
.admin-layout__body { min-width: 0; display: flex; flex-direction: column; min-height: 100vh; }
.admin-layout__content { flex: 1; min-width: 0; padding: 30px 36px 44px; }
.admin-layout__content:focus { outline: none; }
.admin-layout__overlay, .admin-layout__close { display: none; }
.admin-layout__skip { position: fixed; top: -100px; left: 16px; z-index: 1000; padding: 12px 18px; background: white; color: #3730a3; border-radius: 8px; font-weight: 600; }
.admin-layout__skip:focus { top: 12px; }
@media (min-width: 1600px) { .admin-layout__content { padding-inline: 44px; } }
@media (max-width: 1199px) { .admin-layout__content { padding: 24px; } }
@media (max-width: 1023px) {
  .admin-layout { display: block; }
  .admin-layout__sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    width: min(88vw, 292px);
    transform: translateX(-101%);
    visibility: hidden;
    transition: transform 200ms ease, visibility 200ms;
    z-index: 80;
    box-shadow: 12px 0 48px #0f172a30;
  }
  .admin-layout.sidebar-open .admin-layout__sidebar { transform: translateX(0); visibility: visible; }
  .admin-layout__close { display: grid; place-items: center; position: absolute; right: 10px; top: 15px; width: 40px; height: 40px; padding: 0; border: 0; border-radius: 8px; color: #d6dce8; background: transparent; cursor: pointer; z-index: 1; }
  .admin-layout__close:hover { background: #ffffff12; color: white; }
  .admin-layout__close:focus-visible { outline: 2px solid #a5b4fc; outline-offset: -2px; }
  .admin-layout__overlay { display: block; position: fixed; inset: 0; z-index: 75; border: 0; background: #0f172a60; backdrop-filter: blur(3px); cursor: pointer; }
  .admin-layout__content { padding: 24px; }
}
@media (max-width: 599px) { .admin-layout__content { padding: 20px 16px 32px; } }
@media (prefers-reduced-motion: reduce) { .admin-layout, .admin-layout__sidebar { transition: none; } }
</style>

<template>
  <header class="topbar">
    <div class="topbar__brand">
      <button class="ghost-button icon-only" type="button" aria-label="Apri menu" @click="$emit('toggle-sidebar')">
        <span class="icon">☰</span>
      </button>
      <div class="brand-mark">MVP</div>
      <div class="brand-meta">
        <p class="brand-title">{{ companyName }}</p>
        <p class="brand-context">Console amministrativa</p>
      </div>
    </div>
    <div class="topbar__context">
      <div class="badge">
        <span class="dot"></span>
        <span class="badge-label">Evento attivo</span>
        <strong class="badge-value">{{ activeEvent }}</strong>
      </div>
      <div class="badge ghost">
        <span class="badge-label">Società</span>
        <strong class="badge-value">{{ organization }}</strong>
      </div>
    </div>
    <div class="topbar__user">
      <div class="user-meta">
        <p class="user-name">{{ userName }}</p>
        <small class="user-role">{{ userRole }}</small>
      </div>
      <div class="avatar">{{ userInitials }}</div>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  companyName: {
    type: String,
    default: 'MVP System',
  },
  organization: {
    type: String,
    default: 'Nessuna società',
  },
  activeEvent: {
    type: String,
    default: 'Nessun evento live',
  },
  userName: {
    type: String,
    default: 'Admin',
  },
  userRole: {
    type: String,
    default: 'Super admin',
  },
});

const userInitials = computed(() => {
  const parts = props.userName.split(' ');
  if (!parts.length) return 'A';
  return parts
    .map((part) => part.charAt(0))
    .join('')
    .slice(0, 2)
    .toUpperCase();
});
</script>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--border-strong);
  background: rgba(10, 16, 28, 0.92);
  backdrop-filter: blur(14px);
}

.topbar__brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 0.85rem;
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  color: #0b1020;
  font-weight: 800;
  letter-spacing: 0.05em;
  box-shadow: 0 10px 30px rgba(37, 99, 235, 0.35);
}

.brand-meta {
  display: flex;
  flex-direction: column;
}

.brand-title {
  margin: 0;
  font-weight: 700;
  color: var(--text-primary);
}

.brand-context {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.9rem;
}

.topbar__context {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 999px;
  background: rgba(96, 165, 250, 0.12);
  color: var(--text-primary);
  border: 1px solid rgba(96, 165, 250, 0.2);
}

.badge.ghost {
  background: rgba(255, 255, 255, 0.04);
  border-color: var(--border-strong);
}

.badge-label {
  color: var(--text-muted);
  font-weight: 600;
}

.badge-value {
  color: var(--text-primary);
  font-weight: 700;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: #22c55e;
  box-shadow: 0 0 0 8px rgba(34, 197, 94, 0.15);
}

.topbar__user {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.35rem 0.5rem;
  border-radius: 999px;
  border: 1px solid var(--border-strong);
  background: rgba(255, 255, 255, 0.03);
}

.user-meta {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.user-name {
  margin: 0;
  color: var(--text-primary);
  font-weight: 700;
}

.user-role {
  margin: 0;
  color: var(--text-muted);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 50%;
  background: linear-gradient(145deg, #22c55e, #16a34a);
  color: #0a101c;
  font-weight: 800;
  box-shadow: 0 10px 24px rgba(34, 197, 94, 0.35);
}

.ghost-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.45rem 0.5rem;
  border-radius: 0.65rem;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
}

.ghost-button:hover {
  background: rgba(255, 255, 255, 0.1);
}

.icon-only {
  width: 2.5rem;
  height: 2.5rem;
}

.icon {
  font-size: 1.05rem;
}

@media (max-width: 1024px) {
  .topbar {
    grid-template-columns: 1fr auto;
    grid-template-rows: auto auto;
  }

  .topbar__context {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}
</style>

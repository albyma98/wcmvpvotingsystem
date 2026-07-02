<script setup>
const props = defineProps({
  team: { type: Object, required: true },   // { name, logo?, sub? }
  size: { type: String, default: 'lg' }     // 'lg' | 'md'
})

const initials = () =>
  props.team.name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
</script>

<template>
  <div class="team">
    <div class="team-logo" :class="`team-logo--${size}`">
      <img v-if="team.logo" :src="team.logo" :alt="team.name" loading="lazy" />
      <template v-else>{{ initials() }}</template>
    </div>
    <div class="team-name">{{ team.name }}</div>
    <div class="team-sub" v-if="team.sub">{{ team.sub }}</div>
  </div>
</template>

<style scoped>
.team { display: flex; flex-direction: column; align-items: center; gap: clamp(3px,.6dvh,6px); min-width: 0; }
.team-logo {
  border-radius: 50%; background: #0D0D12; border: 2px solid rgba(255,255,255,.14);
  display: grid; place-items: center; overflow: hidden; font-weight: 900; font-size: 20px;
}
.team-logo--lg { width: clamp(38px,6.8dvh,58px); height: clamp(38px,6.8dvh,58px); font-size: clamp(12px,2dvh,17px); }
.team-logo--md { width: clamp(36px,6.4dvh,54px); height: clamp(36px,6.4dvh,54px); font-size: clamp(11px,1.9dvh,16px); }
.team-logo img { width: 100%; height: 100%; object-fit: cover; }
.team-name { font-size: clamp(9.5px,1.5dvh,12px); font-weight: 800; letter-spacing: .5px; text-transform: uppercase; text-align: center; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
.team-sub { font-size: clamp(8px,1.15dvh,10px); color: var(--tm-text-faint); letter-spacing: 1px; margin-top: -2px; }
</style>

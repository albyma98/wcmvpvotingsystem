<template>
  <span class="fan-avatar" :class="{ 'is-framed': framed }" :style="avatarStyle" role="img" :aria-label="accessibleLabel">
    <span class="fan-avatar-core" aria-hidden="true"><span>{{ initials }}</span></span>
    <span v-if="framed" class="fan-avatar-club-light" aria-hidden="true"></span>
    <img v-if="framed" class="fan-avatar-frame" :src="avatarFrameSmall" :srcset="`${avatarFrameSmall} 256w, ${avatarFrame} 640w`" :sizes="`${avatarSize}px`" alt="" aria-hidden="true" decoding="async" />
    <span v-if="framed" class="fan-avatar-glint" aria-hidden="true"></span>
  </span>
</template>

<script setup>
import { computed } from 'vue';
import avatarFrame from '../assets/fan-collectibles/avatar-frame.webp';
import avatarFrameSmall from '../assets/fan-collectibles/avatar-frame-small.webp';

const props = defineProps({
  name: { type: String, default: '' },
  framed: { type: Boolean, default: false },
  size: { type: [Number, String], default: 44 },
});

const displayName = computed(() => typeof props.name === 'string' && props.name.trim() ? props.name.trim() : 'Tifoso');
const initials = computed(() => displayName.value.split(/\s+/).slice(0, 2).map((part) => Array.from(part)[0]).join('').toLocaleUpperCase('it-IT'));
const avatarSize = computed(() => {
  const size = typeof props.size === 'number' || typeof props.size === 'string' && props.size.trim() !== '' ? Number(props.size) : NaN;
  return Number.isFinite(size) && size > 0 ? Math.min(240, Math.max(20, size)) : 44;
});
const avatarStyle = computed(() => ({ '--avatar-size': `${avatarSize.value}px` }));
const accessibleLabel = computed(() => `${displayName.value}${props.framed ? ', cornice della società indossata' : ''}`);
</script>

<style scoped>
.fan-avatar { --avatar-size: 44px; position: relative; isolation: isolate; display: inline-grid; place-items: center; width: var(--avatar-size); height: var(--avatar-size); flex: none; vertical-align: middle; color: var(--ink, #23161a); }
.fan-avatar, .fan-avatar * { box-sizing: border-box; }
.fan-avatar-core { position: absolute; inset: 8%; display: grid; place-items: center; overflow: hidden; border: 1px solid var(--red-line, #ffd2ce); border-radius: 50%; background: radial-gradient(circle at 35% 25%, var(--card, #fff), var(--red-soft, #ffe8e6)); box-shadow: inset 0 1px 0 #ffffffc2, 0 2px 5px #0000000b; }
.fan-avatar-core > span { max-width: 100%; font-family: inherit; font-size: calc(var(--avatar-size) * .31); font-weight: 900; line-height: 1; letter-spacing: -.05em; text-align: center; }
.is-framed .fan-avatar-core { inset: 18.5%; border: 0; background: radial-gradient(circle at 35% 20%, #39434d, #141a21 72%); color: #f4f7fc; box-shadow: inset 0 0 0 calc(var(--avatar-size) * .035) var(--red, #e4212e), inset 0 2px 8px #000000ad; }
.is-framed .fan-avatar-core > span { position: relative; z-index: 1; font-size: calc(var(--avatar-size) * .23); text-shadow: 0 2px 3px #0009; }
.fan-avatar-club-light { position: absolute; z-index: 1; inset: 9%; border-radius: 50%; background: conic-gradient(from 15deg, transparent 0 24%, var(--red, #c52b45) 28% 34%, transparent 38% 73%, var(--gold, #5264bc) 78% 84%, transparent 90%); box-shadow: 0 0 calc(var(--avatar-size) * .08) color-mix(in srgb, var(--red, #c52b45) 25%, transparent); opacity: .85; }
.fan-avatar-club-light::after { content: ''; position: absolute; inset: 11%; border-radius: inherit; background: #1b232d; }
.fan-avatar-core { z-index: 2; }
.fan-avatar-frame { position: absolute; z-index: 3; inset: 0; width: 100%; height: 100%; object-fit: contain; filter: drop-shadow(0 2px 2px #0005); pointer-events: none; animation: frame-equip .7s cubic-bezier(.2,.75,.2,1) both; }
.fan-avatar-glint { position: absolute; z-index: 4; inset: 3%; overflow: hidden; border-radius: 50%; pointer-events: none; }
.fan-avatar-glint::after { content: ''; position: absolute; inset: -55%; background: linear-gradient(110deg, transparent 42%, #ffffff00 44%, #ffffffa3 49%, #ffffff00 53%, transparent 56%); transform: translateX(-60%); opacity: 0; }
@media (hover: hover) { .is-framed:hover .fan-avatar-glint::after { animation: avatar-reflection .7s ease-out both; } }
:focus-visible .is-framed .fan-avatar-glint::after { animation: avatar-reflection .7s ease-out both; }
@keyframes avatar-reflection { 0% { transform: translateX(-60%); opacity: 0; } 15% { opacity: .65; } 100% { transform: translateX(60%); opacity: 0; } }
@keyframes frame-equip { 0% { opacity: 0; transform: scale(1.12) rotate(-6deg); } 100% { opacity: 1; transform: none; } }
@media (prefers-reduced-motion: reduce) { .fan-avatar-glint { display: none; } .fan-avatar-frame { animation: none; } }
</style>

<script setup>
import { computed } from "vue";
import {
  avatarLook,
  HAIR_NAMES,
  BODY_NAMES,
  EXPRESSION_NAMES,
  SKIN_COLORS,
  HAIR_COLORS,
  SHIRT_COLORS,
} from "./avatarLook.js";
const props = defineProps({
  equipment: { type: Object, default: () => ({}) },
  seed: { type: Number, default: 1 },
  disabled: Boolean,
});
const emit = defineEmits(["change"]);
const look = computed(() =>
  avatarLook({ seed: props.seed, equipment: props.equipment }),
);
const options = [
  { key: "hair", label: "Acconciatura", values: HAIR_NAMES },
  { key: "body", label: "Silhouette", values: BODY_NAMES },
  { key: "expression", label: "Espressione", values: EXPRESSION_NAMES },
  { key: "skin", label: "Tonalità della pelle", values: SKIN_COLORS },
  { key: "hairColor", label: "Colore capelli", values: HAIR_COLORS },
  { key: "shirt", label: "Colore maglia base", values: SHIRT_COLORS },
];
function selected(key) {
  if (key === "skin") return SKIN_COLORS.indexOf(look.value.skin);
  if (key === "hairColor") return HAIR_COLORS.indexOf(look.value.hairColor);
  if (key === "shirt") return SHIRT_COLORS.indexOf(look.value.shirt);
  return look.value[key];
}
</script>
<template>
  <div class="appearance-controls">
    <label v-for="option in options" :key="option.key"
      ><span>{{ option.label }}</span
      ><select
        :aria-label="option.label"
        :value="selected(option.key)"
        :disabled="
          disabled ||
          (option.key === 'hair' && !!equipment.hair) ||
          (option.key === 'shirt' && !!equipment.shirt)
        "
        @change="emit('change', { [option.key]: Number($event.target.value) })"
      >
        <option v-if="option.key === 'hair' && equipment.hair" :value="4">
          Acconciatura del guardaroba
        </option>
        <option v-if="option.key === 'shirt' && equipment.shirt" :value="-1">
          Maglia del guardaroba
        </option>
        <option
          v-for="(value, index) in option.values"
          :value="index"
          :key="index"
        >
          {{
            value.startsWith("#")
              ? (option.key === "skin" ? "Tonalità " : "Colore ") + (index + 1)
              : value
          }}
        </option></select
      ><i
        v-if="['skin', 'hairColor', 'shirt'].includes(option.key)"
        :style="{ background: look[option.key] }"
        aria-hidden="true"
      ></i
    ></label>
    <p>
      Varianti estetiche libere. I capi equipaggiati mantengono il loro aspetto.
    </p>
  </div>
</template>
<style scoped>
.appearance-controls {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin: 12px 0;
}
.appearance-controls label {
  position: relative;
  margin: 0 !important;
  min-width: 0;
}
.appearance-controls label > span {
  font-size: 11px;
  display: block;
  margin-bottom: 6px;
}
.appearance-controls select {
  display: block !important;
  width: 100%;
  min-height: 42px;
  background: #fffef7;
  border: 1px solid #cbd6bd;
  border-radius: 9px;
  color: #2d4432;
  padding: 8px;
  font-size: 12px;
}
.appearance-controls i {
  display: block;
  position: absolute;
  right: 20px;
  top: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1px solid #819276;
}
.appearance-controls p {
  grid-column: 1/-1;
  font-size: 10px;
  color: #7a8c6b;
  line-height: 1.6;
}
@media (max-width: 700px) {
  .appearance-controls select {
    font-size: 16px;
  }
}
</style>

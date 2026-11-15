<script setup>
import { computed } from 'vue';

const props = defineProps({
  draft: { type: Object, required: true },
  disabled: Boolean,
});

const choices = computed(() => [
  ...props.draft.questions.map(({ id, title }) => ({ id, title: title.trim() || id })),
  { id: 'suggestion', title: 'Domanda aperta per i suggerimenti' },
]);

function atSelectionLimit(id) {
  const { selectedQuestions } = props.draft;
  return selectedQuestions.includes(id) ? selectedQuestions.length === 1 : selectedQuestions.length >= 4;
}
</script>

<template>
  <fieldset class="feedback-question-selection" :disabled="disabled">
    <legend>Domande da mostrare</legend>
    <p>Seleziona da 1 a 4 domande ({{ draft.selectedQuestions.length }}/4).</p>
    <label v-for="choice in choices" :key="choice.id" class="v2-check">
      <input v-model="draft.selectedQuestions" type="checkbox" :value="choice.id" :disabled="atSelectionLimit(choice.id)" />
      {{ choice.title }}
    </label>
  </fieldset>
</template>

<style scoped>
.feedback-question-selection { display: grid; gap: .6rem; min-width: 0; }
.feedback-question-selection label { display: flex; align-items: flex-start; gap: .5rem; cursor: pointer; }
.feedback-question-selection input { margin-top: .2em; flex: none; }
</style>

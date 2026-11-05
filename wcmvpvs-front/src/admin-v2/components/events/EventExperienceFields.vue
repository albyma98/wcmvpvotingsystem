<script setup>
defineProps({ draft: { type: Object, required: true }, disabled: Boolean });
const groups = [
 { title: 'Prima del voto', fields: [['show_pre_vote_sponsors', 'Sponsor a bordo campo'], ['show_pre_vote_bottom_sponsors', 'Sponsor a fondo campo'], ['show_vote_counter', 'Contatore dei voti in tempo reale']] },
 { title: 'Dopo il voto', fields: [['show_vote_trend', 'Andamento dei voti'], ['show_selfie', 'Selfie MVP'], ['show_reaction_test', 'Gioco dei riflessi'], ['show_feedback_survey', 'Sondaggio di feedback']] },
];
</script>
<template>
 <div class="v2-grid-2"><fieldset v-for="group in groups" :key="group.title"><legend>{{ group.title }}</legend><label v-for="[field, label] in group.fields" :key="field" class="v2-check"><input v-model="draft[field]" type="checkbox" :disabled="disabled" />{{ label }}</label></fieldset></div>
</template>

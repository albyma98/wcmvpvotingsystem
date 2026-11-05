<script setup>
import { inject } from 'vue';
const props = defineProps({ event: { type: Object, required: true } });
const c = inject('admin-v2');
const remove = (question) => c.confirmAction('Eliminare la domanda?', 'La domanda verrà rimossa dal quiz di questo evento.', () => c.run(`quiz-question-${props.event.id}`, () => c.deleteQuizQuestion(props.event.id, question), 'Domanda eliminata.'));
</script>
<template>
 <div class="v2-stack">
  <form class="v2-stack" @submit.prevent="c.run(`quiz-config-${event.id}`, () => c.saveQuizConfig(event.id), 'Configurazione quiz salvata.')">
   <div class="v2-toolbar"><label class="v2-check"><input v-model="c.quizDraftFor(event.id).enabled" type="checkbox" />Quiz abilitato per questo evento</label><button class="v2-button" type="button" :disabled="c.busy(`quiz-reload-${event.id}`)" @click="c.confirmAction('Ricaricare il quiz?', 'Le modifiche locali del quiz saranno sostituite dai dati salvati.', () => c.run(`quiz-reload-${event.id}`, () => c.loadQuizForEvent(event.id)))">Ricarica dati salvati</button></div>
   <p class="v2-muted">Aggiungi le domande e scegli la risposta corretta. Sono previste da due a quattro risposte per domanda.</p>
   <button class="v2-button v2-primary v2-quiz-save" :disabled="c.busy(`quiz-config-${event.id}`)">{{ c.busy(`quiz-config-${event.id}`) ? 'Salvataggio…' : 'Salva configurazione quiz' }}</button>
  </form>
  <p v-if="!c.quizQuestionsFor(event.id).length" class="v2-empty">Il quiz non contiene ancora domande.</p>
  <form v-for="(question,index) in c.quizQuestionsFor(event.id)" :key="question.id || `new-${index}`" class="v2-stack v2-question" @submit.prevent="c.run(`quiz-question-${event.id}`, () => c.saveQuizQuestion(event.id, question), 'Domanda salvata.')">
   <div class="v2-toolbar"><span class="v2-badge">Domanda {{ index + 1 }}</span><span class="v2-muted">{{ question.id ? 'Salvata' : 'Nuova domanda da salvare' }}</span></div>
   <label class="v2-field">Testo della domanda<input v-model="question.question_text" placeholder="Scrivi la domanda per i tifosi…" /></label>
   <div class="v2-grid-2"><label v-for="(_,answerIndex) in question.answers" :key="answerIndex" class="v2-field">Risposta {{ answerIndex + 1 }}<input v-model="question.answers[answerIndex]" :placeholder="`Risposta ${answerIndex + 1}`" /></label></div>
   <div class="v2-grid-2"><label class="v2-field">Risposta corretta<select v-model.number="question.correct_index"><option v-for="(_,answerIndex) in question.answers" :key="answerIndex" :value="answerIndex">{{ answerIndex + 1 }} · {{ question.answers[answerIndex] || `Risposta ${answerIndex + 1}` }}</option></select></label><label class="v2-field">Ordine della domanda<input v-model.number="question.order_index" type="number" min="0" /></label></div>
   <div class="v2-toolbar"><button type="button" class="v2-button" :disabled="question.answers.length >= 4" @click="c.addAnswerToQuestion(event.id, index)">Aggiungi risposta</button><button type="button" class="v2-button" :disabled="question.answers.length <= 2" @click="c.removeAnswerFromQuestion(event.id, index)">Rimuovi ultima risposta</button><button class="v2-button v2-primary" :disabled="c.busy(`quiz-question-${event.id}`)">Salva domanda</button><button class="v2-button v2-danger" type="button" :disabled="c.busy(`quiz-question-${event.id}`)" @click="remove(question)">Elimina domanda</button></div>
  </form>
  <div><button class="v2-button" type="button" :disabled="c.busy(`quiz-question-${event.id}`)" @click="c.addQuizQuestionDraft(event.id)">Aggiungi domanda</button></div>
 </div>
</template>
<style scoped>.v2-question{border:1px solid var(--v2-line);padding:20px;border-radius:10px;background:var(--v2-surface)}.v2-quiz-save{justify-self:start}@media(max-width:640px){.v2-question{padding:14px}.v2-question .v2-toolbar>.v2-button{flex:1}}</style>

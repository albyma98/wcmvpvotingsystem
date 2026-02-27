<template>
  <Teleport to="body">
    <div v-if="modelValue" class="fixed inset-0 z-[140] flex items-end bg-slate-950/85 p-3 sm:items-center sm:justify-center">
      <div class="w-full max-w-md rounded-2xl border border-white/20 bg-slate-900 p-4 text-white">
        <template v-if="!showForm">
          <h3 class="text-lg font-black uppercase">{{ content.title }}</h3>
          <p class="mt-2 text-sm text-slate-200">{{ content.message }}</p>
          <ul v-if="content.benefits?.length" class="mt-3 space-y-1 text-sm">
            <li v-for="benefit in content.benefits" :key="benefit">• {{ benefit }}</li>
          </ul>
          <div class="mt-4 flex gap-2">
            <button class="flex-1 rounded bg-amber-500 px-3 py-2 text-xs font-black text-slate-950" @click="showForm = true">{{ content.primaryCta }}</button>
            <button class="rounded border border-white/30 px-3 py-2 text-xs font-bold" @click="closeAsLater">{{ content.secondaryCta || 'ORA NO' }}</button>
          </div>
        </template>

        <template v-else>
          <h3 class="text-lg font-black uppercase">Salva profilo tifoso</h3>
          <form class="mt-3 space-y-2" @submit.prevent="submit">
            <input v-model.trim="form.nickname" required class="w-full rounded bg-slate-800 px-3 py-2" placeholder="Nickname" />
            <input v-model.trim="form.gender" class="w-full rounded bg-slate-800 px-3 py-2" placeholder="Sesso (opzionale)" />
            <input v-model.trim="form.phone" required class="w-full rounded bg-slate-800 px-3 py-2" placeholder="Numero di telefono" />
            <label class="flex items-start gap-2 text-xs">
              <input v-model="form.acceptedTerms" type="checkbox" required class="mt-0.5" />
              <span>Accetto termini e consenso per consegna premi e gestione profilo tifoso.</span>
            </label>
            <p v-if="errorMessage" class="text-xs text-red-300">{{ errorMessage }}</p>
            <div class="flex gap-2 pt-1">
              <button :disabled="loading" class="flex-1 rounded bg-emerald-500 px-3 py-2 text-xs font-black text-slate-950">{{ loading ? 'SALVATAGGIO...' : 'CONFERMA PROFILO' }}</button>
              <button type="button" class="rounded border border-white/30 px-3 py-2 text-xs font-bold" @click="showForm = false">INDIETRO</button>
            </div>
          </form>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';

const props = defineProps({
  modelValue: Boolean,
  trigger: { type: String, default: 'after_vote' },
  earnedCoins: { type: Number, default: 0 },
  onSubmit: { type: Function, default: null },
});
const emit = defineEmits(['update:modelValue', 'dismissed']);

const showForm = ref(false);
const loading = ref(false);
const errorMessage = ref('');
const form = reactive({ nickname: '', gender: '', phone: '', acceptedTerms: false });

const copyMap = {
  after_vote: {
    title: 'Partecipa ai premi, salva il profilo tifoso',
    message: 'Da oggi per partecipare alla lotteria premi è richiesta la registrazione: ci aiuta a contattare i tifosi vincenti e consegnare i premi evitando ritiri mancati.',
    benefits: ['Partecipa all\'estrazione dei premi', 'Entra in classifica tifosi', 'Spendi monete per premi e coupon', 'Salva monete per le prossime partite'],
    primaryCta: 'SALVA PROFILO E PARTECIPA',
    secondaryCta: 'ORA NO',
  },
  leaderboard: {
    title: 'Entra in classifica tifosi',
    message: 'Vuoi comparire con il tuo nome e scalare posizioni nella community?',
    benefits: ['Compari in classifica ufficiale', 'Salvi monete e punti esperienza'],
    primaryCta: 'ENTRA IN CLASSIFICA',
  },
  spend_redeem: {
    title: 'Attiva profilo per riscattare',
    message: 'Per riscattare premi e coupon serve un profilo tifoso, così possiamo consegnarti il premio e validare il ritiro.',
    benefits: [],
    primaryCta: 'ATTIVA PROFILO PER RISCATTARE',
  },
  after_earn: {
    title: 'Salva le tue monete',
    message: `Hai guadagnato ${props.earnedCoins} monete. Se non le salvi, alla prossima partita riparti da zero.`,
    benefits: [],
    primaryCta: 'SALVA MONETE',
    secondaryCta: 'CONTINUA COME OSPITE',
  },
};

const content = computed(() => copyMap[props.trigger] || copyMap.after_vote);

watch(() => props.modelValue, (open) => {
  if (!open) {
    showForm.value = false;
    errorMessage.value = '';
  }
});

function closeAsLater() {
  emit('dismissed', props.trigger);
  emit('update:modelValue', false);
}

async function submit() {
  loading.value = true;
  errorMessage.value = '';
  const result = props.onSubmit ? await props.onSubmit({ ...form, trigger: props.trigger }) : { ok: false };
  loading.value = false;
  if (result?.ok === false) {
    errorMessage.value = result.message || 'Errore salvataggio profilo';
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue" class="fan-modal-overlay fixed inset-0 z-[140] flex items-center justify-center p-3">
      <div class="fan-modal-panel w-full max-w-lg rounded-3xl border border-white/15 p-4 text-white sm:p-6">
        <template v-if="stage === 'prompt'">
          <div class="text-center">
            <p class="text-5xl leading-none">{{ content.hero }}</p>
            <h3 class="mt-3 text-2xl font-black tracking-tight">{{ content.title }}</h3>
            <p class="mt-2 text-sm text-slate-200">{{ content.subtitle }}</p>
          </div>

          <div class="mt-4 grid gap-2 sm:grid-cols-3">
            <article v-for="benefit in content.benefits" :key="benefit.label" class="fan-benefit-card rounded-2xl border border-white/20 p-3 text-center">
              <p class="text-2xl">{{ benefit.icon }}</p>
              <p class="mt-1 text-xs font-extrabold uppercase tracking-wide">{{ benefit.label }}</p>
            </article>
          </div>

          <article v-if="props.trigger === 'spend_redeem'" class="mt-3 rounded-2xl border border-emerald-300/30 bg-emerald-400/10 p-3 text-center">
            <p class="text-xs uppercase tracking-wide text-emerald-100/80">Reward selezionato</p>
            <p class="mt-1 text-sm font-black">{{ rewardLabel }}</p>
          </article>

          <p v-if="content.note" class="mt-3 text-center text-xs text-slate-300">{{ content.note }}</p>

          <div class="mt-4 flex flex-col gap-2 sm:flex-row">
            <button class="fan-primary-cta flex-1 rounded-xl px-4 py-3 text-xs font-black tracking-wide text-slate-950" @click="startForm">{{ content.primaryCta }}</button>
            <button class="rounded-xl border border-white/30 px-4 py-3 text-xs font-bold text-slate-100" @click="closeAsLater">{{ content.secondaryCta }}</button>
          </div>
        </template>

        <template v-else-if="stage === 'step1'">
          <div class="flex items-center justify-center gap-2 text-xl">
            <span>🎟</span><span>🏆</span><span>🪙</span>
          </div>
          <h3 class="mt-3 text-center text-xl font-black tracking-tight">Scegli il tuo nome da tifoso</h3>
          <form class="mt-4 space-y-3" @submit.prevent="goStep2">
            <input
              v-model.trim="form.nickname"
              required
              maxlength="24"
              class="w-full rounded-2xl border border-white/25 bg-slate-900/80 px-4 py-3 text-center text-lg font-black outline-none ring-amber-300/45 transition focus:ring"
              placeholder="NICKNAME"
            />
            <div class="flex flex-wrap justify-center gap-2">
              <button
                v-for="suggestion in nicknameSuggestions"
                :key="suggestion"
                type="button"
                class="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-xs font-bold text-slate-100"
                @click="form.nickname = suggestion"
              >
                {{ suggestion }}
              </button>
            </div>
            <button class="fan-primary-cta w-full rounded-xl px-4 py-3 text-xs font-black tracking-wide text-slate-950">CONTINUA</button>
          </form>
        </template>

        <template v-else-if="stage === 'step2'">
          <p class="text-center text-[11px] font-bold uppercase tracking-[0.2em] text-cyan-200/80">STEP 2 DI 2</p>
          <div class="flex items-center justify-center gap-2 text-xl">
            <span>🎟</span><span>🏆</span><span>🪙</span>
          </div>
          <h3 class="mt-3 text-center text-xl font-black tracking-tight">📱 Dove possiamo avvisarti se vinci</h3>
          <form class="mt-4 space-y-3" @submit.prevent="submit">
            <FormInputCard icon="📱" label="TELEFONO" helper="Ti avvisiamo solo se vinci" :is-valid="isPhoneValid">
              <template #default="fieldControl">
                <div class="flex items-center gap-2 rounded-xl border border-white/10 bg-slate-950/40 px-3 py-3">
                  <span class="rounded-lg bg-white/10 px-2 py-1 text-sm font-bold text-slate-200">+39</span>
                  <input
                    ref="phoneInputRef"
                    v-model.trim="form.phone"
                    required
                    type="tel"
                    inputmode="numeric"
                    autocomplete="tel-national"
                    class="w-full bg-transparent text-base font-semibold text-white outline-none placeholder:text-slate-400"
                    placeholder="3331234567"
                    @focus="fieldControl.onFocus"
                    @blur="fieldControl.onBlur"
                  />
                </div>
              </template>
            </FormInputCard>

            <FormInputCard icon="👤" label="SESSO" optional>
              <template #default="fieldControl">
                <input
                  v-model.trim="form.gender"
                  class="w-full rounded-xl border border-white/10 bg-slate-950/40 px-3 py-3 text-base font-semibold text-white outline-none transition placeholder:text-slate-400 focus:border-cyan-300/40"
                  placeholder="Inserisci sesso"
                  @focus="fieldControl.onFocus"
                  @blur="fieldControl.onBlur"
                />
              </template>
            </FormInputCard>

            <label
              class="consent-card flex cursor-pointer items-start gap-3 rounded-2xl border border-white/15 p-3 text-xs transition-all duration-300"
              :class="form.acceptedTerms ? 'border-cyan-300/60 shadow-[0_0_16px_rgba(34,211,238,0.25)]' : ''"
            >
              <input v-model="form.acceptedTerms" type="checkbox" required class="mt-0.5 h-4 w-4 accent-cyan-300" />
              <span>Accetto il consenso per consegna premi e gestione profilo tifoso.</span>
            </label>
            <p v-if="errorMessage" class="text-xs text-red-300">{{ errorMessage }}</p>
            <div class="flex gap-2">
              <button
                :disabled="loading || !isPhoneValid"
                class="fan-primary-cta fan-primary-cta-step2 flex-1 rounded-xl px-4 py-3 text-xs font-black tracking-wide text-slate-950"
                :class="isPhoneValid ? 'fan-primary-cta-active' : 'opacity-50 saturate-75'"
              >
                {{ loading ? 'ATTIVAZIONE...' : 'ATTIVA PROFILO' }}
              </button>
              <button type="button" class="rounded-xl border border-white/30 px-4 py-3 text-xs font-bold" @click="stage = 'step1'">Indietro</button>
            </div>
          </form>
        </template>

        <template v-else>
          <div class="coin-glow text-center">
            <p class="text-5xl leading-none">✨🪙</p>
            <h3 class="mt-3 text-2xl font-black">Profilo tifoso attivo!</h3>
            <p class="mt-2 text-sm text-slate-200">Monete salvate: {{ savedCoins }}</p>
            <p class="mx-auto mt-3 w-fit rounded-full border border-amber-300/35 bg-amber-300/15 px-3 py-1 text-xs font-black text-amber-100">Ora sei in classifica</p>
            <button class="fan-primary-cta mt-5 w-full rounded-xl px-4 py-3 text-xs font-black tracking-wide text-slate-950" @click="closeSuccess">CONTINUA</button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue';
import FormInputCard from './FormInputCard.vue';

const props = defineProps({
  modelValue: Boolean,
  trigger: { type: String, default: 'after_vote' },
  earnedCoins: { type: Number, default: 0 },
  walletCoins: { type: Number, default: 0 },
  rewardLabel: { type: String, default: 'Coupon Match Day' },
  onSubmit: { type: Function, default: null },
});
const emit = defineEmits(['update:modelValue', 'dismissed']);

const stage = ref('prompt');
const loading = ref(false);
const errorMessage = ref('');
const form = reactive({ nickname: '', gender: '', phone: '', acceptedTerms: false });
const savedCoins = ref(0);
const phoneInputRef = ref(null);
const nicknameSuggestions = ['MuroTotale', 'VolleyKing', 'AceHunter', 'CurvaNord'];
const isPhoneValid = computed(() => /^\d{8,15}$/.test((form.phone || '').replace(/\s+/g, '')));

const copyMap = {
  after_vote: {
    hero: '✅🎟',
    title: 'Voto confermato!',
    subtitle: 'Vuoi partecipare anche all’estrazione premi?',
    note: 'Serve il profilo per contattare i vincitori.',
    benefits: [
      { icon: '🎟', label: 'Lotteria premi' },
      { icon: '🏆', label: 'Classifica tifosi' },
      { icon: '🪙', label: 'Monete salvate' },
    ],
    primaryCta: 'ATTIVA PROFILO TIFOSO',
    secondaryCta: 'Continua come ospite',
  },
  leaderboard: {
    hero: '🏆',
    title: 'Entra in classifica tifosi',
    subtitle: 'Scala la vetta e sfida gli altri tifosi.',
    benefits: [
      { icon: '🥇', label: 'Classifica ufficiale' },
      { icon: '🪙', label: 'Monete salvate' },
      { icon: '🔥', label: 'Sfida i tifosi' },
    ],
    primaryCta: 'ENTRA IN CLASSIFICA',
    secondaryCta: 'Non ora',
  },
  spend_redeem: {
    hero: '🎁',
    title: 'Riscattalo con le tue monete',
    subtitle: 'Attiva ora il profilo per sbloccare il premio.',
    benefits: [
      { icon: '✅', label: 'Riscatti i premi' },
      { icon: '📞', label: 'Ti contattiamo se vinci' },
    ],
    primaryCta: 'ATTIVA PROFILO E RISCATTA',
    secondaryCta: 'Chiudi',
  },
  after_earn: {
    hero: '🪙✨',
    title: 'Salva monete',
    subtitle: `Hai guadagnato ${props.earnedCoins} monete. Attiva il profilo e non le perdi.`,
    benefits: [
      { icon: '🪙', label: 'Monete salvate' },
      { icon: '🏆', label: 'Entra in classifica' },
      { icon: '🎁', label: 'Sblocca premi' },
    ],
    primaryCta: 'SALVA MONETE',
    secondaryCta: 'Continua come ospite',
  },
};

const content = computed(() => copyMap[props.trigger] || copyMap.after_vote);

watch(() => props.modelValue, (open) => {
  if (!open) {
    stage.value = 'prompt';
    errorMessage.value = '';
    loading.value = false;
    savedCoins.value = 0;
  }
});

watch(() => stage.value, async (nextStage) => {
  if (nextStage !== 'step2') {
    return;
  }
  await nextTick();
  phoneInputRef.value?.focus();
});

function startForm() {
  stage.value = 'step1';
}

function goStep2() {
  if (!form.nickname) {
    return;
  }
  stage.value = 'step2';
}

function closeAsLater() {
  emit('dismissed', props.trigger);
  emit('update:modelValue', false);
}

function closeSuccess() {
  emit('update:modelValue', false);
}

async function submit() {
  loading.value = true;
  errorMessage.value = '';
  const result = props.onSubmit ? await props.onSubmit({ ...form, trigger: props.trigger }) : { ok: false };
  loading.value = false;
  if (result?.ok === false) {
    errorMessage.value = result.message || 'Errore salvataggio profilo';
    return;
  }
  savedCoins.value = Math.max(0, Number(result?.wallet ?? props.walletCoins ?? props.earnedCoins) || 0);
  stage.value = 'success';
}
</script>

<style scoped>
.fan-modal-overlay {
  background: rgba(2, 6, 23, 0.74);
  backdrop-filter: blur(14px);
}

.fan-modal-panel {
  background:
    radial-gradient(circle at 10% 0%, rgba(250, 204, 21, 0.2), transparent 40%),
    radial-gradient(circle at 100% 0%, rgba(52, 211, 153, 0.2), transparent 34%),
    linear-gradient(170deg, rgba(15, 23, 42, 0.96), rgba(2, 6, 23, 0.98));
  box-shadow: 0 0 45px rgba(14, 165, 233, 0.2), 0 20px 45px rgba(0, 0, 0, 0.55);
  animation: modal-enter 0.28s ease;
}

.fan-benefit-card {
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.14), rgba(15, 23, 42, 0.15));
}

.fan-primary-cta {
  background: linear-gradient(125deg, #fde047, #f97316);
  box-shadow: 0 0 16px rgba(251, 191, 36, 0.55);
  transition: transform 0.16s ease, box-shadow 0.16s ease;
}

.fan-primary-cta:active {
  transform: scale(0.98);
  box-shadow: 0 0 8px rgba(251, 191, 36, 0.35);
}

.fan-primary-cta-step2 {
  transition: transform 0.18s ease, box-shadow 0.2s ease, opacity 0.2s ease;
}

.fan-primary-cta-step2:disabled {
  cursor: not-allowed;
  box-shadow: none;
}

.fan-primary-cta-active {
  animation: cta-ready 1.5s ease-in-out infinite alternate;
}

.consent-card {
  background: linear-gradient(160deg, rgba(15, 23, 42, 0.88), rgba(30, 41, 59, 0.48));
}

.coin-glow {
  animation: coin-glow 1.4s ease-in-out infinite alternate;
}

@keyframes modal-enter {
  from {
    opacity: 0;
    transform: translateY(10px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes coin-glow {
  from {
    filter: drop-shadow(0 0 0 rgba(253, 224, 71, 0));
  }
  to {
    filter: drop-shadow(0 0 10px rgba(253, 224, 71, 0.5));
  }
}

@keyframes cta-ready {
  from {
    box-shadow: 0 0 10px rgba(251, 191, 36, 0.35);
  }
  to {
    box-shadow: 0 0 18px rgba(56, 189, 248, 0.55), 0 0 24px rgba(251, 191, 36, 0.42);
  }
}
</style>

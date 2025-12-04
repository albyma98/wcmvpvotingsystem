<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import VolleyCourt from "./VolleyCourt.vue";
import PlayerCard from "./PlayerCard.vue";
import SelfieMvpSection from "./SelfieMvpSection.vue";
import ReactionTestSection from "./ReactionTestSection.vue";
import LiveResultsSection from "./LiveResultsSection.vue";
import {
  apiClient,
  vote,
  fetchVoteStatus,
  sendJsonBeacon,
  submitEventFeedback,
  trackPageEngagement,
  fetchEventEngagement,
} from "../api";
import { DEFAULT_ROSTER_SCHEMA, mapPlayersToLayout } from "../roster";
import { getOrCreateDeviceId } from "../deviceId";

const props = defineProps({
  eventId: {
    type: Number,
    default: undefined,
  },
  activeEvent: {
    type: Object,
    default: null,
  },
  activeEventChecked: {
    type: Boolean,
    default: false,
  },
  loadingActiveEvent: {
    type: Boolean,
    default: false,
  },
});

const rawPlayers = ref([]);
const isLoadingPlayers = ref(false);
const playersError = ref("");
const rosterSchema = ref(DEFAULT_ROSTER_SCHEMA);

const calledUpPlayers = computed(() =>
  Array.isArray(rawPlayers.value)
    ? rawPlayers.value.filter((player) => player?.is_called_up === true)
    : [],
);

const formatPlayerName = (player) => {
  if (!player) {
    return "";
  }

  const firstName =
    typeof player.first_name === "string"
      ? player.first_name.trim()
      : typeof player.firstName === "string"
        ? player.firstName.trim()
        : "";
  const lastName =
    typeof player.last_name === "string"
      ? player.last_name.trim()
      : typeof player.lastName === "string"
        ? player.lastName.trim()
        : "";
  const composed = `${firstName} ${lastName}`.trim();
  if (composed) {
    return composed;
  }
  if (typeof player.name === "string") {
    const fallbackName = player.name.trim();
    if (fallbackName) {
      return fallbackName;
    }
  }
  if (typeof player.raw === "object" && player.raw) {
    const rawName = formatPlayerName(player.raw);
    if (rawName) {
      return rawName;
    }
  }
  return "";
};

const effectiveRosterSchema = computed(() => {
  const totalCalledUp = calledUpPlayers.value.length;
  if (totalCalledUp === 12 || totalCalledUp === 13 || totalCalledUp === 14) {
    return totalCalledUp;
  }
  return rosterSchema.value;
});

const fieldPlayers = computed(() =>
  mapPlayersToLayout(calledUpPlayers.value, {
    layoutSchema: effectiveRosterSchema.value,
  }),
);
const selectedPlayer = computed(() => {
  if (!votedPlayerId.value || votedPlayerId.value <= 0) {
    return null;
  }
  return (
    fieldPlayers.value.find((player) => player.id === votedPlayerId.value) ||
    calledUpPlayers.value.find((player) => player.id === votedPlayerId.value) ||
    null
  );
});
const selectedPlayerName = computed(() => formatPlayerName(selectedPlayer.value));
const activeSponsorIds = computed(() =>
  sponsors.value
    .map((item) => {
      const parsed = Number(item?.id);
      return Number.isFinite(parsed) ? parsed : 0;
    })
    .filter((id) => id > 0),
);

const sponsors = ref([]);
const courtSponsors = computed(() => {
  const schema = effectiveRosterSchema.value;
  if (schema === 14) {
    return sponsors.value.slice(0, 1);
  }
  return sponsors.value.slice(0, 2);
});
const sponsorSectionRef = ref(null);
const sponsorObserverThresholds = [0, 0.25, 0.5, 0.75, 1];
let sponsorIntersectionObserver = null;
let sponsorVisibilityInterval = 0;
const sponsorVisibilityState = {
  isVisible: false,
  visibleSince: 0,
  accumulatedMs: 0,
};
const recordedSponsorSessions = new Set();
const recordedSponsorSeen = new Set();
const recordedSponsorWatched = new Set();
const hasVoted = ref(false);
const isCheckingVoteStatus = ref(false);

const totalVotes = ref(0);
const isVoteTotalLoading = ref(false);
const voteTotalError = ref("");
const isRefreshingVoteTotal = ref(false);
let voteTotalTimer = null;
let countdownTimer = null;
const nowTimestamp = ref(Date.now());
const waitingMessages = [
  "Prepartita: preparati a scegliere il tuo MVP 🏆",
  "Chi merita i riflettori oggi?",
  "Il tuo voto può cambiare tutto",
];
const waitingMessageIndex = ref(0);
let waitingMessageTimer = null;
const activeWaitingMessage = computed(
  () => waitingMessages[waitingMessageIndex.value] ?? waitingMessages[0],
);
const hasTriggeredFinalCountdownEffect = ref(false);

const engagementStats = ref(null);
const userEngagementSeconds = ref(0);
const engagementLoading = ref(false);
const engagementState = reactive({
  startedAt: 0,
  accumulatedMs: 0,
});

const updateNowTimestamp = () => {
  nowTimestamp.value = Date.now();
};

const stopCountdownTimer = () => {
  if (typeof window !== "undefined" && countdownTimer) {
    window.clearInterval(countdownTimer);
    countdownTimer = null;
  }
};

const startCountdownTimer = () => {
  if (typeof window === "undefined") {
    return;
  }
  stopCountdownTimer();
  updateNowTimestamp();
  countdownTimer = window.setInterval(updateNowTimestamp, 1000);
};

const stopWaitingMessageTimer = () => {
  if (typeof window !== "undefined" && waitingMessageTimer) {
    window.clearInterval(waitingMessageTimer);
    waitingMessageTimer = null;
  }
};

const startWaitingMessageTimer = () => {
  if (typeof window === "undefined" || waitingMessageTimer || !isPreMatch.value) {
    return;
  }
  waitingMessageTimer = window.setInterval(() => {
    waitingMessageIndex.value = (waitingMessageIndex.value + 1) % waitingMessages.length;
  }, 9000);
};

const isMobileDevice = () => {
  if (typeof navigator === "undefined") {
    return false;
  }
  return /Mobi|Android|iP(ad|hone|od)/i.test(navigator.userAgent || "");
};

const triggerCountdownVibration = () => {
  if (!isMobileDevice()) {
    return;
  }
  if (typeof navigator !== "undefined" && typeof navigator.vibrate === "function") {
    navigator.vibrate(60);
  }
};

const resetEngagementState = () => {
  engagementState.startedAt = 0;
  engagementState.accumulatedMs = 0;
  userEngagementSeconds.value = 0;
};

const nowMs = () => (typeof performance !== "undefined" ? performance.now() : Date.now());

const pauseEngagementTimer = () => {
  if (!engagementState.startedAt) {
    return;
  }
  engagementState.accumulatedMs += nowMs() - engagementState.startedAt;
  engagementState.startedAt = 0;
};

const startEngagementTimer = () => {
  if (typeof document === "undefined") {
    return;
  }
  if (document.visibilityState === "hidden") {
    return;
  }
  if (!currentEventId.value || engagementState.startedAt) {
    return;
  }
  engagementState.startedAt = nowMs();
};

const sendEngagementIfNeeded = async (targetEventId = currentEventId.value) => {
  pauseEngagementTimer();
  if (!targetEventId) {
    return;
  }
  const seconds = Math.floor(engagementState.accumulatedMs / 1000);
  if (seconds <= 0) {
    return;
  }
  engagementLoading.value = true;
  try {
    await trackPageEngagement(targetEventId, seconds);
    userEngagementSeconds.value = seconds;
  } catch (error) {
    console.error("Impossibile registrare il tempo di permanenza", error);
  } finally {
    engagementLoading.value = false;
    engagementState.accumulatedMs = 0;
  }
};

const handleVisibilityChange = () => {
  if (typeof document === "undefined") {
    return;
  }
  if (document.visibilityState === "hidden") {
    sendEngagementIfNeeded();
  } else {
    startEngagementTimer();
  }
};

const handlePageHide = () => {
  sendEngagementIfNeeded();
};

const loadEngagementStats = async (eventId) => {
  if (!eventId) {
    engagementStats.value = null;
    return;
  }
  engagementLoading.value = true;
  try {
    const { ok, data } = await fetchEventEngagement(eventId);
    if (ok) {
      engagementStats.value = data ?? null;
    }
  } catch (error) {
    console.error("Impossibile caricare le statistiche di permanenza", error);
  } finally {
    engagementLoading.value = false;
  }
};

const formattedVoteTotal = computed(() =>
  Number.isFinite(totalVotes.value)
    ? totalVotes.value.toLocaleString("it-IT")
    : "0",
);

const stopVoteTotalPolling = () => {
  if (voteTotalTimer) {
    window.clearInterval(voteTotalTimer);
    voteTotalTimer = null;
  }
};

const startVoteTotalPolling = () => {
  if (!preVoteSettings.value.showVoteCounter) {
    return;
  }
  stopVoteTotalPolling();
  voteTotalTimer = window.setInterval(() => {
    refreshVoteTotal({ silent: true });
  }, 4000);
};

const refreshVoteTotal = async ({ silent = false } = {}) => {
  if (!preVoteSettings.value.showVoteCounter) {
    totalVotes.value = 0;
    voteTotalError.value = "";
    if (!silent) {
      isVoteTotalLoading.value = false;
    }
    return;
  }
  const eventId = currentEventId.value;
  if (!eventId) {
    totalVotes.value = 0;
    voteTotalError.value = "";
    if (!silent) {
      isVoteTotalLoading.value = false;
    }
    return;
  }

  if (isRefreshingVoteTotal.value) {
    return;
  }

  isRefreshingVoteTotal.value = true;
  if (!silent) {
    isVoteTotalLoading.value = true;
  }

  try {
    const { data } = await apiClient.get(`/events/${eventId}/votes/count`);
    const rawTotal = Number(
      typeof data?.total === "number" ? data.total : (data?.count ?? 0),
    );
    totalVotes.value = Number.isFinite(rawTotal) ? rawTotal : 0;
    voteTotalError.value = "";
  } catch (error) {
    console.error("Impossibile aggiornare il totale voti", error);
    voteTotalError.value = "Totale voti non disponibile in questo momento.";
  } finally {
    if (!silent) {
      isVoteTotalLoading.value = false;
    }
    isRefreshingVoteTotal.value = false;
  }
};

async function loadSponsors() {
  try {
    const { data } = await apiClient.get("/sponsors");
    if (Array.isArray(data)) {
      sponsors.value = data
        .map((item, index) => {
          const image =
            typeof item?.logo_data === "string" ? item.logo_data : "";
          if (!image) {
            return null;
          }
          const resolvedName =
            typeof item?.name === "string" && item.name.trim()
              ? item.name.trim()
              : "";
          const resolvedLink =
            typeof item?.link_url === "string" && item.link_url.trim()
              ? item.link_url.trim()
              : "";
          return {
            id: Number(item?.id) || index + 1,
            name: resolvedName,
            image,
            link: resolvedLink,
          };
        })
        .filter(Boolean);
    } else {
      sponsors.value = [];
    }
  } catch (error) {
    console.error("Impossibile caricare gli sponsor", error);
    sponsors.value = [];
  }
}

function recordSponsorClick(sponsor) {
  if (!sponsor || !sponsor.id) {
    return;
  }
  const eventId = currentEventId.value;
  if (!eventId) {
    return;
  }
  const payload = {
    device_id: getOrCreateDeviceId(),
    at: new Date().toISOString(),
  };
  sendJsonBeacon(
    `/events/${eventId}/sponsors/${sponsor.id}/click`,
    payload,
  ).catch(() => {});
}

const handleSponsorClick = (sponsor) => {
  recordSponsorClick(sponsor);
};

const getNow = () =>
  typeof performance !== "undefined" && performance.now
    ? performance.now()
    : Date.now();

function resetSponsorVisibility() {
  sponsorVisibilityState.isVisible = false;
  sponsorVisibilityState.visibleSince = 0;
  sponsorVisibilityState.accumulatedMs = 0;
}

function stopSponsorVisibilityInterval() {
  if (sponsorVisibilityInterval) {
    window.clearInterval(sponsorVisibilityInterval);
    sponsorVisibilityInterval = 0;
  }
}

function currentSponsorViewDuration() {
  const now = getNow();
  let total = sponsorVisibilityState.accumulatedMs;
  if (sponsorVisibilityState.isVisible && sponsorVisibilityState.visibleSince) {
    total += now - sponsorVisibilityState.visibleSince;
  }
  return total;
}

function startSponsorVisibilityInterval() {
  if (typeof window === "undefined") {
    return;
  }
  if (sponsorVisibilityInterval) {
    return;
  }
  sponsorVisibilityInterval = window.setInterval(() => {
    const eventId = currentEventId.value;
    if (!eventId) {
      stopSponsorVisibilityInterval();
      return;
    }
    if (!sponsorVisibilityState.isVisible) {
      return;
    }
    const durationMs = currentSponsorViewDuration();
    if (durationMs >= 2000 && !recordedSponsorWatched.has(eventId)) {
      sendSponsorExposureEvent(eventId, "watched", durationMs);
    }
  }, 250);
}

function ensureSponsorSession(eventId) {
  if (!preVoteSettings.value.showSponsors) {
    return;
  }
  if (!eventId || recordedSponsorSessions.has(eventId)) {
    return;
  }
  recordedSponsorSessions.add(eventId);
  sendJsonBeacon(`/events/${eventId}/sponsors/session`, {
    device_id: getOrCreateDeviceId(),
    at: new Date().toISOString(),
  }).catch(() => {});
}

function sendSponsorExposureEvent(eventId, type, durationMs = 0) {
  if (!preVoteSettings.value.showSponsors) {
    return;
  }
  if (!eventId) {
    return;
  }
  if (type === "seen") {
    if (recordedSponsorSeen.has(eventId)) {
      return;
    }
    recordedSponsorSeen.add(eventId);
  } else if (type === "watched") {
    if (recordedSponsorWatched.has(eventId)) {
      return;
    }
    recordedSponsorWatched.add(eventId);
  }

  const ids = activeSponsorIds.value;
  if (!ids.length) {
    return;
  }

  const payload = {
    device_id: getOrCreateDeviceId(),
    sponsor_ids: ids,
    type,
    duration_ms:
      type === "watched" && durationMs > 0 ? Math.round(durationMs) : undefined,
  };

  sendJsonBeacon(`/events/${eventId}/sponsors/exposures`, payload).catch(
    () => {},
  );
}

function handleSponsorVisibility(isVisible) {
  const eventId = currentEventId.value;
  if (!eventId) {
    return;
  }

  if (isVisible) {
    ensureSponsorSession(eventId);
    if (!sponsorVisibilityState.isVisible) {
      sponsorVisibilityState.isVisible = true;
      sponsorVisibilityState.visibleSince = getNow();
    }
    sendSponsorExposureEvent(eventId, "seen");
    startSponsorVisibilityInterval();
  } else {
    if (
      sponsorVisibilityState.isVisible &&
      sponsorVisibilityState.visibleSince
    ) {
      sponsorVisibilityState.accumulatedMs +=
        getNow() - sponsorVisibilityState.visibleSince;
      sponsorVisibilityState.visibleSince = 0;
      sponsorVisibilityState.isVisible = false;
    }
    const durationMs = currentSponsorViewDuration();
    if (durationMs >= 2000 && !recordedSponsorWatched.has(eventId)) {
      sendSponsorExposureEvent(eventId, "watched", durationMs);
    }
    stopSponsorVisibilityInterval();
  }
}

function setupSponsorObserver() {
  if (typeof window === "undefined" || !("IntersectionObserver" in window)) {
    return;
  }
  const target = sponsorSectionRef.value;
  if (!target) {
    return;
  }
  if (sponsorIntersectionObserver) {
    sponsorIntersectionObserver.disconnect();
  }
  sponsorIntersectionObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.target !== target) {
          return;
        }
        const visible = entry.isIntersecting && entry.intersectionRatio > 0;
        handleSponsorVisibility(visible);
      });
    },
    { threshold: sponsorObserverThresholds },
  );
  sponsorIntersectionObserver.observe(target);
}

function teardownSponsorObserver() {
  if (sponsorIntersectionObserver) {
    sponsorIntersectionObserver.disconnect();
    sponsorIntersectionObserver = null;
  }
}

async function loadPlayers() {
  isLoadingPlayers.value = true;
  playersError.value = "";
  try {
    const { data } = await apiClient.get("/public/players");
    const schemaCandidate = Number(data?.roster_schema);
    rosterSchema.value =
      schemaCandidate === 12 || schemaCandidate === 13 || schemaCandidate === 14
        ? schemaCandidate
        : DEFAULT_ROSTER_SCHEMA;

    const payload = Array.isArray(data?.players) ? data.players : data;

    if (Array.isArray(payload)) {
      rawPlayers.value = payload.map((item) => ({
        id: Number(item?.id) || 0,
        first_name: typeof item?.first_name === "string" ? item.first_name : "",
        last_name: typeof item?.last_name === "string" ? item.last_name : "",
        role: typeof item?.role === "string" ? item.role : "",
        jersey_number:
          typeof item?.jersey_number === "number"
            ? item?.jersey_number
            : Number.isFinite(Number(item?.jersey_number))
              ? Number(item?.jersey_number)
              : null,
        image_url: typeof item?.image_url === "string" ? item.image_url : "",
        is_called_up: item?.is_called_up === true,
      }));
    } else {
      rawPlayers.value = [];
    }
  } catch (error) {
    console.error("Impossibile caricare i giocatori", error);
    playersError.value =
      "Non è stato possibile caricare i giocatori. Riprova più tardi.";
    rawPlayers.value = [];
  } finally {
    isLoadingPlayers.value = false;
  }
}

const votedPlayerId = ref(null);
const isVoting = ref(false);
const cardSize = ref(88);
const errorMessage = ref("");
const pendingPlayer = ref(null);
const isEditingVote = ref(false);
const voteUpdateMessage = ref("");
const showTicketModal = ref(false);
const showAlreadyVotedModal = ref(false);
const ticketCode = ref("");
const ticketQrUrl = ref("");
const ticketLoadError = ref("");
const isTicketLoading = ref(false);
const showVoteSummary = computed(
  () => hasVoted.value && Boolean(ticketCode.value || ticketQrUrl.value),
);

const buildQrUrl = (qrData) =>
  qrData
    ? `https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(
        qrData,
      )}`
    : "";

const TICKET_STORAGE_PREFIX = "wcmvp-ticket:";

const getTicketStorageKey = (eventId) =>
  `${TICKET_STORAGE_PREFIX}${eventId}`;

function persistTicketData(eventId, data) {
  if (typeof window === "undefined" || !eventId) {
    return;
  }
  try {
    window.localStorage.setItem(
      getTicketStorageKey(eventId),
      JSON.stringify(data),
    );
  } catch (error) {
    console.warn("Impossibile salvare il ticket in locale", error);
  }
}

function loadStoredTicketData(eventId) {
  if (typeof window === "undefined" || !eventId) {
    return null;
  }
  try {
    const raw = window.localStorage.getItem(getTicketStorageKey(eventId));
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    const code = typeof parsed.code === "string" ? parsed.code.trim() : "";
    const qrData =
      typeof parsed.qrData === "string" && parsed.qrData.trim()
        ? parsed.qrData.trim()
        : "";
    return {
      code,
      qrData,
    };
  } catch (error) {
    console.warn("Impossibile caricare il ticket salvato", error);
    return null;
  }
}

const DEFAULT_FEEDBACK_SURVEY = Object.freeze({
  questions: [
    {
      id: "experience",
      title: "Com’è stata la tua esperienza di voto oggi?",
      answers: [
        { value: "very_easy", label: "Facilissima", icon: "🤩" },
        { value: "easy", label: "Abbastanza semplice", icon: "🙂" },
        { value: "complex", label: "Un po’ macchinosa", icon: "😐" },
        { value: "hard", label: "Difficile", icon: "😣" },
      ],
    },
    {
      id: "team_spirit",
      title:
        "Ti sei sentito parte della squadra mentre sceglievi l’MVP del pubblico?",
      answers: [
        { value: "high", label: "Sì, tantissimo!", icon: "🔥" },
        { value: "medium", label: "In parte", icon: "🙂" },
        { value: "low", label: "Non proprio", icon: "🙄" },
      ],
    },
    {
      id: "perks_interest",
      title:
        "Immagina che la tua partecipazione ti permetta di vivere esperienze speciali o vantaggi come vero tifoso… ti piacerebbe?",
      answers: [
        { value: "yes", label: "Sì, assolutamente", icon: "💙" },
        { value: "maybe", label: "Forse", icon: "🙂" },
        { value: "no", label: "No", icon: "🙄" },
      ],
    },
    {
      id: "mini_games_interest",
      title:
        "Ti piacerebbe divertirti ancora di più con mini-giochi o sfide tra un set e l’altro per mettere alla prova i tuoi riflessi?",
      answers: [
        { value: "super_excited", label: "Sì, carichissimo!", icon: "🔥" },
        { value: "maybe", label: "Forse più avanti", icon: "🙂" },
        { value: "no", label: "No grazie", icon: "🙄" },
      ],
    },
  ],
  suggestionPrompt:
    "Se potessi migliorare qualcosa, cosa ti piacerebbe aggiungere o cambiare?",
});

function normalizeFeedbackSurvey(raw) {
  const normalized = {
    questions: DEFAULT_FEEDBACK_SURVEY.questions.map((question) => ({
      id: question.id,
      title: question.title,
      answers: question.answers.map((answer) => ({
        value: answer.value,
        label: answer.label,
        icon: answer.icon || "",
      })),
    })),
    suggestionPrompt: DEFAULT_FEEDBACK_SURVEY.suggestionPrompt,
  };

  if (!raw || typeof raw !== "object") {
    return normalized;
  }

  const questionOverrides = new Map();
  const rawQuestions = Array.isArray(raw.questions)
    ? raw.questions
    : raw.Questions;
  if (Array.isArray(rawQuestions)) {
    rawQuestions.forEach((question) => {
      if (!question || typeof question !== "object") {
        return;
      }
      const id = typeof question.id === "string" ? question.id.trim() : "";
      if (!id) {
        return;
      }
      questionOverrides.set(id, question);
    });
  }

  normalized.questions = normalized.questions.map((question) => {
    const override = questionOverrides.get(question.id);
    if (!override || typeof override !== "object") {
      return question;
    }

    const overrideTitle =
      typeof override.title === "string"
        ? override.title.trim()
        : typeof override.Title === "string"
          ? override.Title.trim()
          : "";
    if (overrideTitle) {
      question.title = overrideTitle;
    }

    const answerOverrides = new Map();
    const rawAnswers = Array.isArray(override.answers)
      ? override.answers
      : override.Answers;
    if (Array.isArray(rawAnswers)) {
      rawAnswers.forEach((answer) => {
        if (!answer || typeof answer !== "object") {
          return;
        }
        const value =
          typeof answer.value === "string" ? answer.value.trim() : "";
        if (!value) {
          return;
        }
        answerOverrides.set(value, answer);
      });
    }

    question.answers = question.answers.map((answer) => {
      const overrideAnswer = answerOverrides.get(answer.value);
      if (!overrideAnswer || typeof overrideAnswer !== "object") {
        return { ...answer };
      }
      const label =
        typeof overrideAnswer.label === "string"
          ? overrideAnswer.label.trim()
          : typeof overrideAnswer.Label === "string"
            ? overrideAnswer.Label.trim()
            : "";
      const icon =
        typeof overrideAnswer.icon === "string"
          ? overrideAnswer.icon.trim()
          : typeof overrideAnswer.Icon === "string"
            ? overrideAnswer.Icon.trim()
            : "";
      return {
        value: answer.value,
        label: label || answer.label,
        icon: icon || answer.icon || "",
      };
    });

    return question;
  });

  const rawSuggestion =
    typeof raw.suggestion_prompt === "string"
      ? raw.suggestion_prompt
      : typeof raw.suggestionPrompt === "string"
        ? raw.suggestionPrompt
        : "";
  if (rawSuggestion && rawSuggestion.trim()) {
    normalized.suggestionPrompt = rawSuggestion.trim();
  }

  return normalized;
}

const feedbackSurvey = computed(() =>
  normalizeFeedbackSurvey(
    props.activeEvent?.feedback_survey ?? props.activeEvent?.feedbackSurvey,
  ),
);

const feedbackQuestions = computed(() =>
  feedbackSurvey.value.questions.map((question) => ({
    id: question.id,
    answerKey: question.id,
    title: question.title,
    options: question.answers.map((answer) => ({
      value: answer.value,
      label: answer.label,
      icon: answer.icon || "",
    })),
  })),
);

const hasOptionalFeedbackQuestion = computed(
  () => Boolean(feedbackSurvey.value.suggestionPrompt?.trim()),
);

const optionalFeedbackQuestion = computed(() => ({
  id: "suggestion",
  answerKey: "suggestion",
  title: feedbackSurvey.value.suggestionPrompt,
}));

const feedbackAnswers = reactive({
  experience: "",
  team_spirit: "",
  perks_interest: "",
  mini_games_interest: "",
  suggestion: "",
});

const feedbackStep = ref(0);
const showFeedbackModal = ref(false);
const isFeedbackSubmitting = ref(false);
const feedbackError = ref("");
const showFeedbackThankYou = ref(false);
const hasCompletedFeedback = ref(false);
const optionalFeedbackMaxLength = 80;
const feedbackStoragePrefix = "wcmvpvs-feedback";
const mandatoryFeedbackKeys = [
  "experience",
  "team_spirit",
  "perks_interest",
  "mini_games_interest",
];

const activeFeedbackQuestion = computed(() =>
  feedbackStep.value < feedbackQuestions.value.length
    ? feedbackQuestions.value[feedbackStep.value]
    : null,
);

const isOptionalFeedbackStep = computed(
  () =>
    hasOptionalFeedbackQuestion.value &&
    feedbackStep.value >= feedbackQuestions.value.length,
);

const showFeedbackActions = computed(
  () =>
    isOptionalFeedbackStep.value ||
    feedbackStep.value >= feedbackQuestions.value.length - 1,
);

const feedbackStepLabel = computed(() => {
  if (feedbackStep.value < feedbackQuestions.value.length) {
    return `Step ${feedbackStep.value + 1} di ${feedbackQuestions.value.length}`;
  }
  return "Extra (opzionale)";
});

const feedbackProgress = computed(() => {
  if (!feedbackQuestions.value.length) {
    return 0;
  }
  const effectiveStep = Math.min(
    feedbackStep.value,
    feedbackQuestions.value.length - 1,
  );
  return Math.round(
    ((effectiveStep + 1) / feedbackQuestions.value.length) * 100,
  );
});

const shouldShowFeedbackCta = computed(
  () =>
    hasVoted.value &&
    postVoteSettings.value.showFeedbackSurvey &&
    !hasCompletedFeedback.value,
);

const showFeedbackThankYouMessage = computed(
  () =>
    postVoteSettings.value.showFeedbackSurvey &&
    hasCompletedFeedback.value &&
    showFeedbackThankYou.value,
);

const handleSelfieSubmitted = () => {
  hasVoted.value = true;
};

const feedbackStorageKey = (eventId) => {
  if (!eventId) {
    return "";
  }
  return `${feedbackStoragePrefix}:${eventId}`;
};

function clearFeedbackAnswers() {
  feedbackAnswers.experience = "";
  feedbackAnswers.team_spirit = "";
  feedbackAnswers.perks_interest = "";
  feedbackAnswers.mini_games_interest = "";
  feedbackAnswers.suggestion = "";
}

function resetFeedbackFlow() {
  feedbackStep.value = 0;
  feedbackError.value = "";
  isFeedbackSubmitting.value = false;
  showFeedbackModal.value = false;
  clearFeedbackAnswers();
}

function readFeedbackCompletion(eventId) {
  if (!eventId || typeof window === "undefined") {
    return false;
  }
  try {
    return window.localStorage?.getItem(feedbackStorageKey(eventId)) === "1";
  } catch (error) {
    return false;
  }
}

function persistFeedbackCompletion(eventId) {
  if (!eventId || typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage?.setItem(feedbackStorageKey(eventId), "1");
  } catch (error) {
    // ignore storage errors
  }
}

function openFeedbackModal() {
  if (!shouldShowFeedbackCta.value) {
    return;
  }
  const firstIncompleteIndex = feedbackQuestions.value.findIndex(
    (question) => !feedbackAnswers[question.answerKey],
  );
  feedbackStep.value =
    firstIncompleteIndex >= 0
      ? firstIncompleteIndex
      : hasOptionalFeedbackQuestion.value
        ? feedbackQuestions.value.length
        : Math.max(feedbackQuestions.value.length - 1, 0);
  feedbackError.value = "";
  showFeedbackModal.value = true;
}

function closeFeedbackModal() {
  if (isFeedbackSubmitting.value) {
    return;
  }
  showFeedbackModal.value = false;
  feedbackError.value = "";
}

function isFeedbackOptionSelected(question, option) {
  if (!question || !option) {
    return false;
  }
  return feedbackAnswers[question.answerKey] === option.value;
}

function handleFeedbackOptionSelect(option) {
  if (!option) {
    return;
  }
  const question = activeFeedbackQuestion.value;
  if (!question) {
    return;
  }
  feedbackAnswers[question.answerKey] = option.value;
  feedbackError.value = "";
  const maxStep = hasOptionalFeedbackQuestion.value
    ? feedbackQuestions.value.length
    : Math.max(feedbackQuestions.value.length - 1, 0);
  const nextStep = Math.min(
    feedbackStep.value + 1,
    maxStep,
  );
  feedbackStep.value = nextStep;
}

function goToPreviousFeedbackStep() {
  if (feedbackStep.value <= 0 || isFeedbackSubmitting.value) {
    return;
  }
  feedbackStep.value -= 1;
  feedbackError.value = "";
}

function skipOptionalFeedback() {
  if (isFeedbackSubmitting.value) {
    return;
  }
  feedbackAnswers.suggestion = "";
  submitFeedback();
}

function handleFeedbackContinue() {
  if (isFeedbackSubmitting.value) {
    return;
  }
  const question = activeFeedbackQuestion.value;
  const answer = question ? feedbackAnswers[question.answerKey] : null;
  if (!answer) {
    feedbackError.value = "Rispondi a tutte le domande per continuare.";
    return;
  }
  if (hasOptionalFeedbackQuestion.value) {
    feedbackStep.value = feedbackQuestions.value.length;
    feedbackError.value = "";
    return;
  }
  submitFeedback();
}

async function submitFeedback() {
  if (isFeedbackSubmitting.value) {
    return;
  }
  const eventId = currentEventId.value;
  if (!eventId) {
    feedbackError.value = "Evento non disponibile in questo momento.";
    return;
  }
  for (const key of mandatoryFeedbackKeys) {
    if (!feedbackAnswers[key]) {
      feedbackError.value = "Rispondi a tutte le domande per continuare.";
      return;
    }
  }
  isFeedbackSubmitting.value = true;
  feedbackError.value = "";
  try {
    const suggestion = feedbackAnswers.suggestion.trim();
    const result = await submitEventFeedback(eventId, {
      experience: feedbackAnswers.experience,
      team_spirit: feedbackAnswers.team_spirit,
      perks_interest: feedbackAnswers.perks_interest,
      mini_games_interest: feedbackAnswers.mini_games_interest,
      suggestion,
    });
    if (result?.ok) {
      persistFeedbackCompletion(eventId);
      hasCompletedFeedback.value = true;
      showFeedbackThankYou.value = true;
      resetFeedbackFlow();
      return;
    }
    if (result?.status === 400) {
      feedbackError.value = "Controlla le risposte e riprova.";
    } else {
      feedbackError.value =
        "Non siamo riusciti a salvare il tuo feedback. Riprova tra qualche istante.";
    }
  } catch (error) {
    feedbackError.value =
      "Non siamo riusciti a salvare il tuo feedback. Riprova tra qualche istante.";
  } finally {
    isFeedbackSubmitting.value = false;
  }
}

async function refreshVoteStatus(eventId) {
  if (!eventId) {
    hasVoted.value = false;
    return;
  }
  isCheckingVoteStatus.value = true;
  const storedTicket = loadStoredTicketData(eventId);
  if (storedTicket) {
    ticketCode.value = storedTicket.code;
    ticketQrUrl.value = buildQrUrl(storedTicket.qrData);
    isTicketLoading.value = false;
  }
  try {
    const { ok, hasVoted: status, playerId } = await fetchVoteStatus(eventId);
    if (ok) {
      const resolvedPlayerId = Number.isFinite(playerId) ? Number(playerId) : null;
      votedPlayerId.value = resolvedPlayerId;
      hasVoted.value = Boolean(status) || Boolean(storedTicket?.code);
      if (!hasVoted.value) {
        voteUpdateMessage.value = "";
      }
    }
  } catch (error) {
    console.warn("Impossibile verificare lo stato del voto", error);
    if (storedTicket?.code) {
      hasVoted.value = true;
    }
  } finally {
    isCheckingVoteStatus.value = false;
  }
}

const sanitizeName = (value) => {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
};

const resolveTeamName = (event, keys) => {
  if (!event) {
    return "";
  }

  for (const key of keys) {
    if (key in event) {
      const resolved = sanitizeName(event[key]);
      if (resolved) {
        return resolved;
      }
    }
  }

  return "";
};

const homeTeamName = computed(() =>
  resolveTeamName(props.activeEvent, [
    "team1_name",
    "team1",
    "home_team",
    "homeTeam",
    "team1Name",
  ]),
);

const awayTeamName = computed(() =>
  resolveTeamName(props.activeEvent, [
    "team2_name",
    "team2",
    "away_team",
    "awayTeam",
    "team2Name",
  ]),
);

const eventTitle = computed(() => {
  const home = homeTeamName.value;
  const away = awayTeamName.value;

  if (home || away) {
    const fallbackHome = home || "Squadra di casa";
    const fallbackAway = away || "Squadra ospite";
    return `${fallbackHome} - ${fallbackAway}`;
  }

  return "Vota il tuo MVP";
});

const currentEventId = computed(() => props.eventId ?? props.activeEvent?.id);

const resolveEventFlag = (event, keys, fallback = true) => {
  if (!event || typeof event !== "object") {
    return fallback;
  }
  for (const key of keys) {
    if (Object.prototype.hasOwnProperty.call(event, key)) {
      return Boolean(event[key]);
    }
  }
  return fallback;
};

const preVoteSettings = computed(() => {
  const event = props.activeEvent || null;
  const showCourtSponsors = resolveEventFlag(
    event,
    [
      "show_pre_vote_sponsors",
      "showPreVoteSponsors",
      "show_sponsors",
      "showSponsors",
    ],
    true,
  );
  const showBottomSponsors = resolveEventFlag(
    event,
    [
      "show_pre_vote_bottom_sponsors",
      "showPreVoteBottomSponsors",
      "show_pre_vote_sponsor_wall",
    ],
    showCourtSponsors,
  );
  return {
    showSponsors: showCourtSponsors || showBottomSponsors,
    showCourtSponsors,
    showBottomSponsors,
    showVoteCounter: resolveEventFlag(
      event,
      ["show_vote_counter", "showVoteCounter", "show_pre_vote_vote_counter"],
      true,
    ),
  };
});

const postVoteSettings = computed(() => {
  const event = props.activeEvent || null;
  return {
    showReactionTest: resolveEventFlag(
      event,
      ["show_reaction_test", "showReactionTest"],
      true,
    ),
    showSelfie: resolveEventFlag(event, ["show_selfie", "showSelfie"], true),
    showVoteTrend: resolveEventFlag(
      event,
      ["show_vote_trend", "showVoteTrend", "show_live_results"],
      true,
    ),
    showFeedbackSurvey: resolveEventFlag(
      event,
      ["show_feedback_survey", "showFeedbackSurvey"],
      true,
    ),
  };
});

const visibleCourtSponsors = computed(() =>
  preVoteSettings.value.showCourtSponsors ? courtSponsors.value : [],
);

const showSponsorSection = computed(
  () => preVoteSettings.value.showBottomSponsors && sponsors.value.length > 0,
);

const sponsorGridClass = computed(() => {
  const count = sponsors.value.length;
  if (count <= 1) {
    return ["grid-cols-1"];
  }
  if (count === 2) {
    return ["grid-cols-2", "grid-rows-1"];
  }
  if (count === 3) {
    return ["grid-cols-2", "md:grid-cols-3"];
  }
  return ["grid-cols-2", "md:grid-cols-3"];
});

const showVoteCounterSection = computed(
  () => preVoteSettings.value.showVoteCounter && Boolean(currentEventId.value),
);

const showInactiveNotice = computed(
  () => props.activeEventChecked && !props.activeEvent,
);
const isCheckingActiveEvent = computed(
  () => props.loadingActiveEvent && !props.activeEventChecked,
);
const isVotingClosed = computed(() => {
  if (!props.activeEvent) {
    return false;
  }
  const raw =
    props.activeEvent.votes_closed ??
    props.activeEvent.votesClosed ??
    props.activeEvent.is_voting_closed;
  return Boolean(raw);
});
const isPreMatch = computed(() => !isVotingClosed.value && isEventUpcoming.value);
const votingOpen = computed(
  () => !isVotingClosed.value && !isEventUpcoming.value,
);

const showLiveResultsSection = computed(() => {
  if (!postVoteSettings.value.showVoteTrend) {
    return false;
  }
  if (!currentEventId.value) {
    return false;
  }
  return hasVoted.value || isCheckingVoteStatus.value;
});

const showSelfieSection = computed(() => {
  if (!postVoteSettings.value.showSelfie) {
    return false;
  }
  if (!currentEventId.value) {
    return false;
  }
  return hasVoted.value || isCheckingVoteStatus.value;
});

const showReactionTestSection = computed(() => {
  if (!postVoteSettings.value.showReactionTest) {
    return false;
  }
  if (!currentEventId.value) {
    return false;
  }
  return hasVoted.value;
});

const resolveEventStartValue = (event) => {
  if (!event || typeof event !== "object") {
    return null;
  }

  const candidateKeys = [
    "start_datetime",
    "startDatetime",
    "startDateTime",
    "start_time",
    "startTime",
    "start_at",
    "startAt",
    "start",
  ];

  for (const key of candidateKeys) {
    if (key in event) {
      const value = event[key];
      if (value instanceof Date) {
        return value;
      }
      if (typeof value === "string") {
        const trimmed = value.trim();
        if (trimmed) {
          return trimmed;
        }
      }
      if (typeof value === "number" && Number.isFinite(value)) {
        return value;
      }
    }
  }

  return null;
};

const eventStartTimestamp = computed(() => {
  const raw = resolveEventStartValue(props.activeEvent);
  if (!raw) {
    return null;
  }

  if (raw instanceof Date) {
    const timestamp = raw.getTime();
    return Number.isFinite(timestamp) ? timestamp : null;
  }

  if (typeof raw === "number") {
    return raw > 0 ? raw : null;
  }

  if (typeof raw === "string") {
    const normalized = raw.includes("T") ? raw : raw.replace(" ", "T");
    const parsed = new Date(normalized);
    const timestamp = parsed.getTime();
    return Number.isNaN(timestamp) ? null : timestamp;
  }

  return null;
});

const timeUntilEventStartMs = computed(() => {
  const start = eventStartTimestamp.value;
  if (!start) {
    return 0;
  }
  const diff = start - nowTimestamp.value;
  return diff > 0 ? diff : 0;
});

const countdownSeconds = computed(() =>
  Math.ceil(timeUntilEventStartMs.value / 1000),
);

const countdownParts = computed(() => {
  const total = countdownSeconds.value;
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const totalHours = Math.floor(total / 3600);
  return { days, hours, minutes, seconds, totalHours };
});

const countdownLabel = computed(() => {
  const { totalHours, minutes, seconds } = countdownParts.value;
  return [totalHours, minutes, seconds]
    .map((value) => String(value).padStart(2, "0"))
    .join(":");
});

const countdownDaysLabel = computed(() => {
  const { days, hours } = countdownParts.value;
  if (days <= 0) {
    return "";
  }
  const dayLabel = days === 1 ? "giorno" : "giorni";
  const hourLabel = hours === 1 ? "ora" : "ore";
  return `${days} ${dayLabel} e ${hours} ${hourLabel} rimanenti`;
});

const isCountdownCritical = computed(
  () => isPreMatch.value && countdownSeconds.value > 0 && countdownSeconds.value <= 5,
);

const isEventUpcoming = computed(() => timeUntilEventStartMs.value > 0);

watch(
  currentEventId,
  (eventId, previousEventId) => {
    if (previousEventId) {
      sendEngagementIfNeeded(previousEventId);
    }
    resetEngagementState();
    engagementStats.value = null;
    votedPlayerId.value = null;
    pendingPlayer.value = null;
    errorMessage.value = "";
    showTicketModal.value = false;
    ticketCode.value = "";
    ticketQrUrl.value = "";
    ticketLoadError.value = "";
    isTicketLoading.value = false;
    showAlreadyVotedModal.value = false;
    totalVotes.value = 0;
    voteTotalError.value = "";
    stopVoteTotalPolling();
    if (eventId && preVoteSettings.value.showVoteCounter) {
      refreshVoteTotal();
      startVoteTotalPolling();
    } else {
      isVoteTotalLoading.value = false;
    }
    hasVoted.value = false;
    isEditingVote.value = false;
    voteUpdateMessage.value = "";
    const storedTicket = loadStoredTicketData(eventId);
    if (storedTicket) {
      ticketCode.value = storedTicket.code;
      ticketQrUrl.value = buildQrUrl(storedTicket.qrData);
      hasVoted.value = Boolean(storedTicket.code);
    }
    if (eventId) {
      refreshVoteStatus(eventId);
    }
    if (eventId && isVotingClosed.value) {
      loadEngagementStats(eventId);
    }
    startEngagementTimer();
    resetSponsorVisibility();
    stopSponsorVisibilityInterval();
    teardownSponsorObserver();
    if (eventId && preVoteSettings.value.showSponsors) {
      ensureSponsorSession(eventId);
      nextTick(() => {
        if (showSponsorSection.value) {
          setupSponsorObserver();
        }
      });
    }
    resetFeedbackFlow();
    if (eventId) {
      const completed = readFeedbackCompletion(eventId);
      hasCompletedFeedback.value = completed;
      showFeedbackThankYou.value = completed && hasVoted.value;
    } else {
      hasCompletedFeedback.value = false;
      showFeedbackThankYou.value = false;
    }
  },
  { immediate: true },
);

watch(
  sponsors,
  (list) => {
    if (!list.length || !preVoteSettings.value.showBottomSponsors) {
      resetSponsorVisibility();
      stopSponsorVisibilityInterval();
      teardownSponsorObserver();
      return;
    }
    if (!currentEventId.value) {
      return;
    }
    nextTick(() => {
      if (showSponsorSection.value) {
        setupSponsorObserver();
      }
    });
  },
  { deep: true },
);

watch(hasVoted, (voted) => {
  if (!voted) {
    if (!hasCompletedFeedback.value) {
      showFeedbackThankYou.value = false;
    }
    showFeedbackModal.value = false;
    isEditingVote.value = false;
    voteUpdateMessage.value = "";
    return;
  }
  if (hasCompletedFeedback.value && postVoteSettings.value.showFeedbackSurvey) {
    showFeedbackThankYou.value = true;
  }
});

watch(
  () => postVoteSettings.value.showFeedbackSurvey,
  (enabled) => {
    if (!enabled) {
      showFeedbackModal.value = false;
    }
  },
);

watch(
  () => preVoteSettings.value.showVoteCounter,
  (enabled) => {
    stopVoteTotalPolling();
    if (!enabled) {
      totalVotes.value = 0;
      voteTotalError.value = "";
      isVoteTotalLoading.value = false;
      return;
    }
    if (currentEventId.value) {
      refreshVoteTotal();
      startVoteTotalPolling();
    }
  },
);

watch(
  () => preVoteSettings.value.showSponsors,
  (enabled) => {
    if (!enabled) {
      resetSponsorVisibility();
      stopSponsorVisibilityInterval();
      teardownSponsorObserver();
      return;
    }
    if (currentEventId.value) {
      ensureSponsorSession(currentEventId.value);
      nextTick(() => {
        if (showSponsorSection.value) {
          setupSponsorObserver();
        }
      });
    }
  },
);

watch(
  () => preVoteSettings.value.showBottomSponsors,
  (enabled) => {
    if (!enabled) {
      resetSponsorVisibility();
      stopSponsorVisibilityInterval();
      teardownSponsorObserver();
      return;
    }
    if (currentEventId.value) {
      nextTick(() => {
        if (showSponsorSection.value) {
          setupSponsorObserver();
        }
      });
    }
  },
);

watch(fieldPlayers, (players) => {
  if (!pendingPlayer.value) {
    return;
  }
  const replacement = players.find(
    (player) => player.id === pendingPlayer.value.id,
  );
  if (replacement) {
    pendingPlayer.value = replacement;
  } else {
    pendingPlayer.value = null;
  }
});

watch(isVotingClosed, (closed) => {
  if (closed) {
    pauseEngagementTimer();
    sendEngagementIfNeeded();
    loadEngagementStats(currentEventId.value);
    pendingPlayer.value = null;
    showTicketModal.value = false;
    showAlreadyVotedModal.value = false;
    ticketLoadError.value = "";
    isTicketLoading.value = false;
    isEditingVote.value = false;
    voteUpdateMessage.value = "";
  } else {
    engagementStats.value = null;
    startEngagementTimer();
  }
});

watch(
  isEventUpcoming,
  (upcoming) => {
    if (upcoming) {
      startCountdownTimer();
      pendingPlayer.value = null;
      showTicketModal.value = false;
      showAlreadyVotedModal.value = false;
      ticketLoadError.value = "";
      isTicketLoading.value = false;
    } else {
      stopCountdownTimer();
    }
  },
  { immediate: true },
);

watch(
  isPreMatch,
  (active) => {
    if (active) {
      waitingMessageIndex.value = 0;
      startWaitingMessageTimer();
    } else {
      stopWaitingMessageTimer();
    }
  },
  { immediate: true },
);

watch(votingOpen, (open) => {
  if (open) {
    stopWaitingMessageTimer();
    return;
  }
  isEditingVote.value = false;
});

watch(isCountdownCritical, (critical) => {
  if (critical && !hasTriggeredFinalCountdownEffect.value) {
    triggerCountdownVibration();
    hasTriggeredFinalCountdownEffect.value = true;
  }
  if (!critical) {
    hasTriggeredFinalCountdownEffect.value = false;
  }
});

const clamp = (value, min, max) => Math.min(Math.max(value, min), max);

const formatSeconds = (seconds) => {
  const total = Math.max(0, Math.floor(seconds || 0));
  const minutes = Math.floor(total / 60);
  const remainingSeconds = total % 60;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours > 0) {
    return `${hours}h ${remainingMinutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${remainingSeconds}s`;
  }
  return `${remainingSeconds}s`;
};

const engagementSummary = computed(() => {
  if (!engagementStats.value) {
    return null;
  }
  return {
    total: formatSeconds(engagementStats.value.total_duration_seconds),
    average: formatSeconds(engagementStats.value.average_duration_seconds),
    users: engagementStats.value.total_users ?? 0,
  };
});

const userEngagementLabel = computed(() => formatSeconds(userEngagementSeconds.value));

const updateCardSize = () => {
  const width = window.innerWidth;
  const height = window.innerHeight;
  const sizeFromWidth = width / 5.8;
  const sizeFromHeight = height / 9.8;
  cardSize.value = clamp(Math.min(sizeFromWidth, sizeFromHeight), 58, 112);
};

const afterVoteCardSize = computed(() =>
  clamp(
    Math.round(cardSize.value * 0.82),
    54,
    Math.max(cardSize.value - 12, 72),
  ),
);

onMounted(() => {
  updateCardSize();
  window.addEventListener("resize", updateCardSize, { passive: true });
  if (typeof document !== "undefined") {
    document.addEventListener("visibilitychange", handleVisibilityChange, { passive: true });
  }
  if (typeof window !== "undefined") {
    window.addEventListener("pagehide", handlePageHide);
    window.addEventListener("blur", pauseEngagementTimer);
    window.addEventListener("focus", startEngagementTimer);
  }
  loadSponsors();
  loadPlayers();
  if (currentEventId.value) {
    if (preVoteSettings.value.showVoteCounter) {
      refreshVoteTotal();
      startVoteTotalPolling();
    }
    refreshVoteStatus(currentEventId.value);
    if (preVoteSettings.value.showSponsors) {
      ensureSponsorSession(currentEventId.value);
    }
    startEngagementTimer();
  }
  nextTick(() => {
    if (showSponsorSection.value) {
      setupSponsorObserver();
    }
  });
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", updateCardSize);
  if (typeof document !== "undefined") {
    document.removeEventListener("visibilitychange", handleVisibilityChange);
  }
  if (typeof window !== "undefined") {
    window.removeEventListener("pagehide", handlePageHide);
    window.removeEventListener("blur", pauseEngagementTimer);
    window.removeEventListener("focus", startEngagementTimer);
  }
  sendEngagementIfNeeded();
  stopVoteTotalPolling();
  stopCountdownTimer();
  stopSponsorVisibilityInterval();
  teardownSponsorObserver();
  stopWaitingMessageTimer();
});

const canEditVote = computed(() => hasVoted.value && votingOpen.value);
const canSelectPlayers = computed(
  () =>
    votingOpen.value &&
    !showInactiveNotice.value &&
    !isCheckingActiveEvent.value &&
    !isVotingClosed.value &&
    !isEventUpcoming.value &&
    (!hasVoted.value || isEditingVote.value),
);
const disableVotes = computed(
  () =>
    !canSelectPlayers.value ||
    isVoting.value,
);

const openPlayerModal = (player) => {
  if (!canSelectPlayers.value || isVoting.value) {
    return;
  }

  pendingPlayer.value = player;
  errorMessage.value = "";
  voteUpdateMessage.value = "";
};

const closeModal = () => {
  if (isVoting.value) {
    return;
  }
  pendingPlayer.value = null;
};

const closeTicketModal = () => {
  showTicketModal.value = false;
  isTicketLoading.value = false;
};

const closeAlreadyVotedModal = () => {
  showAlreadyVotedModal.value = false;
};

const startVoteEdit = () => {
  if (!canEditVote.value) {
    return;
  }
  isEditingVote.value = true;
  voteUpdateMessage.value = "";
};

const voteForPlayer = async (player) => {
  if (isVotingClosed.value || isEventUpcoming.value || !votingOpen.value) {
    return;
  }

  if (isVoting.value || !player?.id) {
    return;
  }

  if (votedPlayerId.value === player.id) {
    return;
  }

  errorMessage.value = "";
  const previousPlayerId = votedPlayerId.value;
  isVoting.value = true;

  const eventId = currentEventId.value;
  if (!eventId) {
    errorMessage.value = "Nessun evento attivo al momento.";
    isVoting.value = false;
    return;
  }

  try {
    const response = await vote({ eventId, playerId: player.id });
    if (response?.ok) {
      const voteResult = response.vote || {};
      votedPlayerId.value = player.id;
      pendingPlayer.value = null;
      hasVoted.value = true;
      isEditingVote.value = false;

      const resolvedPlayerName =
        formatPlayerName(player) || formatPlayerName(player?.raw) || "il tuo giocatore";
      voteUpdateMessage.value = previousPlayerId && previousPlayerId !== player.id
        ? `Voto aggiornato! Ora il tuo MVP è ${resolvedPlayerName}.`
        : `Voto registrato! Ora il tuo MVP è ${resolvedPlayerName}.`;

      const codeSource = voteResult.code || "";
      const qrSource = voteResult.qr_data || "";

      if (codeSource) {
        ticketCode.value = codeSource;
        ticketLoadError.value = "";
        isTicketLoading.value = Boolean(qrSource);
        ticketQrUrl.value = buildQrUrl(qrSource);
        if (!qrSource) {
          isTicketLoading.value = false;
        }
        persistTicketData(eventId, {
          code: codeSource,
          qrData: qrSource,
        });
        showTicketModal.value = true;
        refreshVoteTotal({ silent: true });
      } else {
        console.warn("voteForPlayer: missing ticket data", response);
        errorMessage.value =
          "Non siamo riusciti a generare il QR del ticket. Riprova.";
      }
    } else {
      voteUpdateMessage.value = "";
      if (response?.status === 409) {
        pendingPlayer.value = null;
        showAlreadyVotedModal.value = true;
        errorMessage.value = "";
        if (!votedPlayerId.value) {
          votedPlayerId.value = -1;
        }
        hasVoted.value = true;
      } else if (response?.status === 429) {
        errorMessage.value =
          response?.message ||
          "Stai votando troppo rapidamente. Attendi qualche istante e riprova.";
      } else {
        errorMessage.value =
          response?.message ||
          "Non è stato possibile registrare il voto. Riprova.";
      }
    }
  } catch (error) {
    console.error("vote error", error);
    voteUpdateMessage.value = "";
    errorMessage.value =
      "Si è verificato un errore. Riprova tra qualche istante.";
  } finally {
    isVoting.value = false;
  }
};

const isModalOpen = computed(() => Boolean(pendingPlayer.value));

const modalActionLabel = computed(() => {
  if (!pendingPlayer.value) {
    return "Vota MVP";
  }
  if (votedPlayerId.value === pendingPlayer.value.id) {
    return "Voto registrato";
  }
  if (isVoting.value) {
    return "Invio...";
  }
  return "Vota MVP";
});

const confirmVote = () => {
  if (!pendingPlayer.value || votedPlayerId.value === pendingPlayer.value.id) {
    return;
  }
  voteForPlayer(pendingPlayer.value);
};

const handleQrLoaded = () => {
  isTicketLoading.value = false;
};

const handleQrError = () => {
  isTicketLoading.value = false;
  ticketQrUrl.value = "";
  ticketLoadError.value =
    "Non siamo riusciti a caricare il QR del ticket. Riprova tra qualche istante.";
};
</script>

<template>
  <div
    class="min-h-screen bg-gradient-to-b from-slate-950 via-slate-900 to-slate-950 text-slate-100 flex flex-col"
  >
    <main
      v-if="!isCheckingActiveEvent && !showInactiveNotice"
      class="flex-1 overflow-y-auto"
    >
      <div class="flex flex-col" :class="hasVoted ? 'gap-6' : 'gap-10'">
        <section v-if="isVotingClosed" class="px-4">
          <div class="closed-banner" role="status" aria-live="polite">
            <h3>Votazioni chiuse</h3>
            <p>
              Grazie per aver partecipato! Ti aspettiamo alla prossima partita
              al palazzetto.
            </p>
          </div>
        </section>
        <section v-if="isVotingClosed && engagementSummary" class="px-4">
          <div class="engagement-card" role="status" aria-live="polite">
            <div class="engagement-card__header">
              <p class="engagement-card__eyebrow">Tempo di permanenza</p>
              <p v-if="engagementSummary.users" class="engagement-card__meta">
                Basato su {{ engagementSummary.users.toLocaleString("it-IT") }} tifosi
              </p>
            </div>
            <div class="engagement-card__grid">
              <div>
                <p class="engagement-card__label">Totale evento</p>
                <p class="engagement-card__value">{{ engagementSummary.total }}</p>
              </div>
              <div>
                <p class="engagement-card__label">Tempo medio</p>
                <p class="engagement-card__value">{{ engagementSummary.average }}</p>
              </div>
              <div v-if="userEngagementSeconds > 0">
                <p class="engagement-card__label">Il tuo tempo sulla pagina</p>
                <p class="engagement-card__value">{{ userEngagementLabel }}</p>
              </div>
            </div>
          </div>
        </section>
        <section v-if="showVoteSummary" class="px-4">
          <div class="vote-summary" role="status" aria-live="polite">
            <div class="vote-summary__content">
              <p class="vote-summary__eyebrow">Hai votato!</p>
              <h3 class="vote-summary__title">
                Conserva il tuo codice per l'estrazione
              </h3>
              <p class="vote-summary__code" aria-label="Codice di voto">
                Codice: <span>{{ ticketCode }}</span>
              </p>
              <p class="vote-summary__hint">
                Mostra questo codice e il QR allo staff in caso di estrazione
                del premio.
              </p>
              <p v-if="ticketLoadError" class="vote-summary__error">
                {{ ticketLoadError }}
              </p>
            </div>
            <div class="vote-summary__qr" aria-hidden="true">
              <div v-if="isTicketLoading" class="vote-summary__qr-loader">
                <span class="qr-loader"></span>
              </div>
              <img v-else-if="ticketQrUrl" :src="ticketQrUrl" alt="QR code" />
              <div v-else class="vote-summary__qr-placeholder">
                QR non disponibile
              </div>
            </div>
          </div>
        </section>
        <section v-if="!hasVoted" class="px-4">
          <div class="mb-6 text-center prematch-intro">
            <Transition name="waiting-message-fade" mode="out-in">
              <p
                v-if="isPreMatch"
                :key="activeWaitingMessage"
                class="prematch-intro__message"
              >
                {{ activeWaitingMessage }}
              </p>
            </Transition>
            <h2
              class="text-lg font-semibold uppercase tracking-[0.1em] text-slate-200"
            >
              {{ eventTitle }}
            </h2>
            <p v-if="!isEventUpcoming" class="mt-2 text-sm text-slate-300">
              Tocca la card del tuo giocatore preferito per assegnarli il voto
            </p>
            <p v-else class="mt-2 text-sm text-slate-300">
              La votazione sarà disponibile all'inizio della partita.
            </p>
            <div
              v-if="isPreMatch && countdownLabel"
              class="prematch-countdown"
              :class="{ 'is-critical': isCountdownCritical }"
            >
              <span class="prematch-countdown__eyebrow">Countdown pre-match</span>
              <span class="prematch-countdown__value">{{ countdownLabel }}</span>
              <span v-if="countdownDaysLabel" class="prematch-countdown__meta">
                {{ countdownDaysLabel }}
              </span>
            </div>
          </div>
          <div
            v-if="fieldPlayers.length"
            :class="['relative h-[95svh]', isPreMatch ? 'prematch-stage' : '']"
          >
            <div
              v-if="isPreMatch"
              class="prematch-shine"
              aria-hidden="true"
            ></div>
            <VolleyCourt
              class="block h-full w-full"
              :players="fieldPlayers"
              :card-size="cardSize"
              :selected-player-id="votedPlayerId"
              :disable-votes="disableVotes"
              :is-voting="isVoting"
              :court-sponsors="visibleCourtSponsors"
              :is-pre-match="isPreMatch"
              :voting-open="votingOpen"
              @select="openPlayerModal"
              @sponsor-click="handleSponsorClick"
            />
          </div>
          <p v-else-if="isLoadingPlayers" class="players-message">
            Caricamento dei giocatori in corso…
          </p>
          <p v-else-if="playersError" class="players-message error">
            {{ playersError }}
          </p>
          <p v-else class="players-message">
            I giocatori non sono ancora stati configurati. Torna più tardi!
          </p>
        </section>
        <section v-else class="px-4 after-vote-section">
          <div class="after-vote-panel">
            <h3>{{ eventTitle }}</h3>
            <p>
              Hai già espresso il tuo voto per questa partita. Conserva il
              codice mostrato in alto e attendi l'estrazione dei premi.
            </p>
          </div>

          <div class="after-vote-success">
            <p class="after-vote-success__eyebrow">
              Voto registrato <span aria-hidden="true">✅</span>
            </p>
            <h3 class="after-vote-success__title">
              Grazie per aver partecipato!
            </h3>
            <button
              v-if="shouldShowFeedbackCta"
              type="button"
              class="feedback-cta"
              @click="openFeedbackModal"
            >
              <span class="feedback-cta__label">Migliora la tua esperienza!💙</span>
              <span class="feedback-cta__time">(in solo 15 secondi)</span>
            </button>
            <p
              v-else-if="showFeedbackThankYouMessage"
              class="after-vote-success__thanks"
            >
              Grazie 💙 Hai aiutato a migliorare l’esperienza dei tifosi 🙌
            </p>
          </div>

          <div class="after-vote-selection">
            <div class="after-vote-selection__header">
              <p class="after-vote-selection__eyebrow">Hai votato:</p>
              <h4 class="after-vote-selection__player">
                {{ selectedPlayerName || "Il tuo giocatore" }}
              </h4>
              <p v-if="votingOpen" class="after-vote-selection__hint">
                Puoi cambiare il tuo voto finché le votazioni sono aperte.
              </p>
              <p v-else class="after-vote-selection__hint">
                Le votazioni sono chiuse. Il tuo voto finale è salvato.
              </p>
              <button
                v-if="canEditVote && !isEditingVote"
                type="button"
                class="after-vote-selection__action"
                @click="startVoteEdit"
              >
                Modifica il mio voto
              </button>
              <p v-else-if="isEditingVote" class="after-vote-selection__hint emphasize">
                Puoi cambiare il tuo voto finché le votazioni sono aperte.
              </p>
              <p v-if="voteUpdateMessage" class="after-vote-selection__confirmation">
                {{ voteUpdateMessage }}
              </p>
            </div>

            <div
              v-if="fieldPlayers.length"
              class="after-vote-selection__court"
              :class="{ 'is-locked': !canSelectPlayers }"
            >
              <VolleyCourt
                class="block h-full w-full"
                :players="fieldPlayers"
                :card-size="afterVoteCardSize"
                :selected-player-id="votedPlayerId"
                :disable-votes="disableVotes"
                :is-voting="isVoting"
                :court-sponsors="visibleCourtSponsors"
                :is-pre-match="isPreMatch"
                :voting-open="votingOpen"
                @select="openPlayerModal"
                @sponsor-click="handleSponsorClick"
              />
            </div>
            <p v-else-if="isLoadingPlayers" class="players-message">
              Caricamento dei giocatori in corso…
            </p>
            <p v-else-if="playersError" class="players-message error">
              {{ playersError }}
            </p>
            <p v-else class="players-message">
              I giocatori non sono ancora stati configurati. Torna più tardi!
            </p>
          </div>

          <LiveResultsSection
            v-if="showLiveResultsSection"
            class="mt-6"
            :event-id="currentEventId"
            :enabled="hasVoted && postVoteSettings.showVoteTrend"
          />
        </section>

        <SelfieMvpSection
          v-if="showSelfieSection"
          :class="['px-4', hasVoted ? 'pt-0' : '']"
          :event-id="currentEventId"
          :enabled="hasVoted && postVoteSettings.showSelfie"
          :loading-status="isCheckingVoteStatus"
          :compact="hasVoted"
          @selfie-submitted="handleSelfieSubmitted"
        />

        <ReactionTestSection
          v-if="showReactionTestSection"
          class="mt-8"
          :event-id="currentEventId"
          :enabled="hasVoted && postVoteSettings.showReactionTest"
        />

        <section v-if="showSponsorSection" ref="sponsorSectionRef" class="px-4">
          <div
            class="relative overflow-hidden rounded-[2.25rem] border border-slate-700/40 bg-gradient-to-br from-slate-900 via-slate-800 to-slate-950 shadow-[0_26px_52px_rgba(8,15,28,0.55)]"
            aria-labelledby="sponsor-title"
          >
            <div
              class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(148,163,184,0.18),_transparent_55%)]"
            ></div>
            <div
              class="pointer-events-none absolute inset-0 bg-[linear-gradient(145deg,_rgba(30,41,59,0.45),_transparent_60%)] mix-blend-screen"
            ></div>

            <div class="relative flex h-full flex-col">
              <header class="px-6 pt-6 pb-4">
                <p
                  id="sponsor-title"
                  class="text-xs font-semibold uppercase tracking-[0.45em] text-slate-300"
                >
                  Sponsor
                </p>
              </header>

              <div class="flex-1 px-6 pb-6">
                <div class="grid auto-rows-fr gap-4" :class="sponsorGridClass">
                  <template v-for="sponsor in sponsors" :key="sponsor.id">
                    <a
                      v-if="sponsor.link"
                      class="group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/10 bg-slate-900/40 shadow-[0_16px_32px_rgba(8,15,28,0.45)]"
                      :href="sponsor.link"
                      target="_blank"
                      rel="noopener noreferrer"
                      :aria-label="sponsor.name"
                      @click="handleSponsorClick(sponsor)"
                    >
                      <div
                        class="absolute inset-0 bg-gradient-to-br from-white/5 via-transparent to-white/10 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
                      ></div>
                      <img
                        :src="sponsor.image"
                        :alt="sponsor.name"
                        class="relative h-full w-full object-cover"
                      />
                      <div
                        class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/85 via-slate-950/25 to-transparent px-4 pb-4 pt-8"
                      >
                        <p
                          class="text-xs font-medium uppercase tracking-[0.25em] text-slate-200 text-center"
                        >
                          {{ sponsor.name }}
                        </p>
                      </div>
                    </a>
                    <div
                      v-else
                      class="group relative flex items-center justify-center overflow-hidden rounded-3xl border border-white/10 bg-slate-900/40 shadow-[0_16px_32px_rgba(8,15,28,0.45)]"
                      :aria-label="sponsor.name"
                    >
                      <div
                        class="absolute inset-0 bg-gradient-to-br from-white/5 via-transparent to-white/10 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
                      ></div>
                      <img
                        :src="sponsor.image"
                        :alt="sponsor.name"
                        class="relative h-full w-full object-cover"
                      />
                      <div
                        class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/85 via-slate-950/25 to-transparent px-4 pb-4 pt-8"
                      >
                        <p
                          class="text-xs font-medium uppercase tracking-[0.25em] text-slate-200 text-center"
                        >
                          {{ sponsor.name }}
                        </p>
                      </div>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-if="showVoteCounterSection" class="px-4">
          <div class="vote-counter" role="status" aria-live="polite">
            <div class="vote-counter__header">
              <p class="vote-counter__title">Totale voti registrati</p>
              <span
                v-if="isVoteTotalLoading"
                class="vote-counter__spinner"
                aria-hidden="true"
              ></span>
            </div>
            <p class="vote-counter__value">{{ formattedVoteTotal }}</p>
            <p v-if="voteTotalError" class="vote-counter__message error">
              {{ voteTotalError }}
            </p>
            <p v-else class="vote-counter__message">
              Aggiornamento automatico ogni pochi secondi
            </p>
          </div>
        </section>

        <p
          v-if="errorMessage"
          class="px-4 pb-6 text-center text-sm text-rose-400"
        >
          {{ errorMessage }}
        </p>

        <transition name="feedback-modal-fade">
          <div
            v-if="showFeedbackModal"
            class="feedback-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="feedback-modal-title"
            @click.self="closeFeedbackModal"
          >
            <div class="feedback-modal__panel">
              <header class="feedback-modal__header">
                <button
                  class="feedback-modal__close"
                  type="button"
                  @click="closeFeedbackModal"
                  aria-label="Chiudi sondaggio"
                >
                  ×
                </button>
                <p class="feedback-modal__step">{{ feedbackStepLabel }}</p>
                <div class="feedback-modal__progress">
                  <div
                    class="feedback-modal__progress-bar"
                    :style="{ width: `${feedbackProgress}%` }"
                  ></div>
                </div>
              </header>

              <div class="feedback-modal__body">
                <h2 id="feedback-modal-title" class="feedback-modal__title">
                  {{
                    isOptionalFeedbackStep
                      ? optionalFeedbackQuestion.title
                      : activeFeedbackQuestion?.title
                  }}
                </h2>
                <p v-if="isOptionalFeedbackStep" class="feedback-modal__hint">
                  Risposta facoltativa (max
                  {{ optionalFeedbackMaxLength }} caratteri)
                </p>
                <div
                  v-if="!isOptionalFeedbackStep && activeFeedbackQuestion"
                  class="feedback-modal__options"
                >
                  <button
                    v-for="option in activeFeedbackQuestion.options"
                    :key="option.value"
                    type="button"
                    class="feedback-modal__option"
                    :class="{
                      active: isFeedbackOptionSelected(
                        activeFeedbackQuestion,
                        option,
                      ),
                    }"
                    @click="handleFeedbackOptionSelect(option)"
                  >
                    <span
                      class="feedback-modal__option-icon"
                      aria-hidden="true"
                      >{{ option.icon }}</span
                    >
                    <span class="feedback-modal__option-label">{{
                      option.label
                    }}</span>
                  </button>
                </div>
                <div v-else-if="isOptionalFeedbackStep" class="feedback-modal__optional">
                  <input
                    v-model="feedbackAnswers.suggestion"
                    :maxlength="optionalFeedbackMaxLength"
                    type="text"
                    class="feedback-modal__input"
                    placeholder="Scrivi qui (max 80 caratteri)"
                  />
                  <span class="feedback-modal__counter"
                    >{{ feedbackAnswers.suggestion.length }}/{{
                      optionalFeedbackMaxLength
                    }}</span
                  >
                </div>
                <p
                  v-if="feedbackError"
                  class="feedback-modal__error"
                  role="alert"
                >
                  {{ feedbackError }}
                </p>
              </div>

              <footer class="feedback-modal__footer">
                <button
                  v-if="feedbackStep > 0 && !isFeedbackSubmitting"
                  type="button"
                  class="feedback-modal__back"
                  @click="goToPreviousFeedbackStep"
                >
                  Indietro
                </button>
                <div
                  class="feedback-modal__footer-actions"
                  :class="{ 'is-hidden': !showFeedbackActions }"
                >
                  <button
                    v-if="isOptionalFeedbackStep"
                    type="button"
                    class="feedback-modal__skip"
                    @click="skipOptionalFeedback"
                    :disabled="isFeedbackSubmitting"
                  >
                    Salta
                  </button>
                  <button
                    v-if="isOptionalFeedbackStep"
                    type="button"
                    class="feedback-modal__submit"
                    @click="submitFeedback"
                    :disabled="isFeedbackSubmitting"
                  >
                    {{ isFeedbackSubmitting ? "Invio…" : "Invia" }}
                  </button>
                  <button
                    v-else
                    type="button"
                    class="feedback-modal__submit"
                    @click="handleFeedbackContinue"
                    :disabled="isFeedbackSubmitting"
                  >
                    {{
                      hasOptionalFeedbackQuestion
                        ? "Continua"
                        : isFeedbackSubmitting
                          ? "Invio…"
                          : "Invia"
                    }}
                  </button>
                </div>
              </footer>
            </div>
          </div>
        </transition>
      </div>
    </main>

    <div
      v-else
      class="flex flex-1 items-center justify-center px-6 py-12 text-center"
    >
      <div class="inactive-panel">
        <template v-if="isCheckingActiveEvent">
          <h2
            class="text-2xl font-semibold uppercase tracking-[0.2em] text-slate-100"
          >
            Verifica evento in corso…
          </h2>
          <p class="mt-4 text-base text-slate-300">
            Stiamo controllando se è disponibile una partita su cui votare.
          </p>
        </template>
        <template v-else>
          <h2
            class="text-2xl font-semibold uppercase tracking-[0.2em] text-slate-100"
          >
            Nessuna partita in corso
          </h2>
          <p class="mt-4 text-base text-slate-300">
            Attendi la prossima partita per votare il tuo MVP. Ti aspettiamo al
            palazzetto!
          </p>
        </template>
      </div>
    </div>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && isModalOpen"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-6 py-10"
      >
        <button
          class="absolute inset-0"
          type="button"
          @click="closeModal"
          aria-label="Chiudi"
        ></button>
        <div
          class="relative z-10 w-full max-w-xs rounded-[2.25rem] border border-white/10 bg-slate-900/95 px-6 py-7 text-center shadow-[0_30px_60px_rgba(15,23,42,0.6)]"
        >
          <div class="flex justify-center">
            <PlayerCard
              v-if="pendingPlayer"
              :player="pendingPlayer"
              :card-size="cardSize * 1.3"
              :is-selected="votedPlayerId === pendingPlayer.id"
              :disabled="true"
            />
          </div>
          <div class="mt-6 flex flex-col gap-3">
            <button
              class="w-full rounded-full bg-yellow-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.35em] text-slate-900 transition-colors duration-200 hover:bg-yellow-300 disabled:cursor-not-allowed disabled:opacity-70"
              type="button"
              :disabled="
                isVoting || !pendingPlayer || votedPlayerId === pendingPlayer.id
              "
              @click="confirmVote"
            >
              {{ modalActionLabel }}
            </button>
            <button
              class="w-full rounded-full border border-white/15 px-4 py-3 text-sm font-semibold uppercase tracking-[0.3em] text-slate-200 transition-colors duration-200 hover:bg-white/10"
              type="button"
              @click="closeModal"
            >
              Annulla
            </button>
          </div>
        </div>
      </div>
    </transition>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && showTicketModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-6 py-10"
      >
        <button
          class="absolute inset-0"
          type="button"
          @click="closeTicketModal"
          aria-label="Chiudi"
        ></button>
        <div
          class="relative z-10 w-full max-w-sm rounded-[2.25rem] border border-white/10 bg-slate-900/95 px-6 py-7 text-center shadow-[0_30px_60px_rgba(15,23,42,0.6)]"
        >
          <h3
            class="text-lg font-semibold uppercase tracking-[0.35em] text-slate-100"
          >
            Voto registrato
          </h3>
          <p class="mt-3 text-sm text-slate-300">
            Fai subito uno screenshot di questa pagina e conservalo. Attendi la
            fine della partita per l'estrazione dei premi e mostra lo screenshot
            allo staff nel caso in cui venga estratto il tuo codice.
          </p>
          <div class="important-notice" role="alert">
            <p
              class="font-semibold uppercase tracking-[0.25em] text-yellow-300"
            >
              Importante
            </p>
            <p class="mt-2 text-sm leading-relaxed text-slate-100">
              SENZA LO SCREENSHOT IL PREMIO NON POTRA' ESSERE ASSEGNATO.
            </p>
          </div>
          <div
            class="mt-5 flex flex-col items-center gap-2 text-lg text-slate-200"
          >
            <p class="font-bold tracking-[0.2em]">Codice: {{ ticketCode }}</p>
          </div>
          <div
            v-if="isTicketLoading"
            class="mt-6 flex flex-col items-center gap-3 text-slate-200"
            role="status"
            aria-live="polite"
          >
            <span class="qr-loader"></span>
            <p
              class="text-sm font-semibold uppercase tracking-[0.3em] text-slate-300"
            >
              Attendi…
            </p>
          </div>
          <p v-if="ticketLoadError" class="mt-4 text-sm text-rose-300">
            {{ ticketLoadError }}
          </p>
          <img
            v-if="ticketQrUrl"
            :src="ticketQrUrl"
            alt="QR code"
            class="mx-auto mt-6 h-40 w-40 rounded-3xl border border-white/10 bg-white p-3"
            :class="{ hidden: isTicketLoading }"
            @load="handleQrLoaded"
            @error="handleQrError"
          />
          <button
            class="mt-7 w-full rounded-full bg-yellow-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.35em] text-slate-900 transition-colors duration-200 hover:bg-yellow-300"
            type="button"
            @click="closeTicketModal"
          >
            Chiudi
          </button>
        </div>
      </div>
    </transition>

    <transition name="fade">
      <div
        v-if="!showInactiveNotice && showAlreadyVotedModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-6 py-10"
      >
        <button
          class="absolute inset-0"
          type="button"
          @click="closeAlreadyVotedModal"
          aria-label="Chiudi"
        ></button>
        <div
          class="relative z-10 w-full max-w-sm rounded-[2.25rem] border border-white/10 bg-slate-900/95 px-6 py-7 text-center shadow-[0_30px_60px_rgba(15,23,42,0.6)]"
        >
          <h3
            class="text-lg font-semibold uppercase tracking-[0.35em] text-slate-100"
          >
            Hai già votato
          </h3>
          <p class="mt-3 text-sm text-slate-300">
            Puoi esprimere il tuo voto una sola volta per partita. Attendi la
            fine della gara per scoprire l'estrazione dei premi.
          </p>
          <button
            class="mt-7 w-full rounded-full bg-yellow-400 px-4 py-3 text-sm font-semibold uppercase tracking-[0.35em] text-slate-900 transition-colors duration-200 hover:bg-yellow-300"
            type="button"
            @click="closeAlreadyVotedModal"
          >
            Ho capito
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.waiting-message-fade-enter-active,
.waiting-message-fade-leave-active {
  transition: opacity 0.4s ease;
}

.waiting-message-fade-enter-from,
.waiting-message-fade-leave-to {
  opacity: 0;
}

.prematch-intro {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: center;
}

.prematch-intro__message {
  margin: 0;
  font-size: 0.95rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #cbd5e1;
}

.prematch-countdown {
  margin-top: 0.75rem;
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 1.25rem;
  border-radius: 999px;
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.14), rgba(59, 130, 246, 0.1));
  border: 1px solid rgba(148, 163, 184, 0.28);
  box-shadow: 0 18px 38px rgba(8, 15, 28, 0.35);
}

.prematch-countdown__eyebrow {
  font-size: 0.7rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: #67e8f9;
}

.prematch-countdown__value {
  font-size: 1.2rem;
  font-weight: 700;
  color: #e2e8f0;
  letter-spacing: 0.08em;
}

.prematch-countdown__meta {
  font-size: 0.85rem;
  color: rgba(226, 232, 240, 0.78);
}

.prematch-countdown.is-critical {
  animation: countdown-pulse 0.9s ease-in-out infinite;
  box-shadow: 0 20px 48px rgba(59, 130, 246, 0.4);
}

.prematch-stage {
  isolation: isolate;
}

.prematch-shine {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(
    120deg,
    rgba(255, 255, 255, 0) 20%,
    rgba(148, 163, 184, 0.2) 40%,
    rgba(255, 255, 255, 0) 60%
  );
  transform: translateX(-100%);
  animation: prematch-shine 11s ease-in-out infinite;
  mix-blend-mode: screen;
  opacity: 0.6;
  will-change: transform, opacity;
  z-index: 2;
}

@keyframes prematch-shine {
  0% {
    transform: translateX(-120%);
    opacity: 0;
  }
  8% {
    opacity: 0.55;
  }
  18% {
    transform: translateX(120%);
    opacity: 0;
  }
  100% {
    transform: translateX(120%);
    opacity: 0;
  }
}

@keyframes countdown-pulse {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.06);
  }
  100% {
    transform: scale(1);
  }
}

@keyframes countdown-flash {
  0% {
    opacity: 0.95;
  }
  50% {
    opacity: 1;
  }
  100% {
    opacity: 0.95;
  }
}

.closed-banner {
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.75);
  padding: 1.75rem 1.5rem;
  text-align: center;
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.45);
}

.closed-banner h3 {
  margin: 0 0 0.75rem;
  font-size: 1.1rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: #fbbf24;
}

.closed-banner p {
  margin: 0;
  font-size: 0.95rem;
  color: #e2e8f0;
}

.engagement-card {
  margin-top: 1rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.9), rgba(15, 23, 42, 0.75));
  padding: 1.5rem;
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.4);
}

.engagement-card__header {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
  justify-content: space-between;
}

.engagement-card__eyebrow {
  margin: 0;
  font-size: 0.8rem;
  letter-spacing: 0.32em;
  text-transform: uppercase;
  color: #38bdf8;
}

.engagement-card__meta {
  margin: 0;
  font-size: 0.85rem;
  color: rgba(226, 232, 240, 0.75);
}

.engagement-card__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
  margin-top: 1rem;
}

.engagement-card__label {
  margin: 0;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.65);
}

.engagement-card__value {
  margin: 0.25rem 0 0;
  font-size: 1.4rem;
  font-weight: 700;
  color: #f8fafc;
}

.vote-summary {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.75rem 1.5rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: linear-gradient(
    145deg,
    rgba(15, 23, 42, 0.9),
    rgba(30, 41, 59, 0.75)
  );
  box-shadow: 0 28px 52px rgba(15, 23, 42, 0.55);
}

.vote-summary__content {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.vote-summary__eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.45em;
  text-transform: uppercase;
  color: #facc15;
}

.vote-summary__title {
  margin: 0;
  font-size: 1.05rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #f8fafc;
}

.vote-summary__code {
  margin: 0.25rem 0 0;
  font-size: 1.2rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #f8fafc;
}

.vote-summary__code span {
  color: #38bdf8;
}

.vote-summary__hint {
  margin: 0.5rem 0 0;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.85);
  line-height: 1.5;
}

.vote-summary__error {
  margin: 0.5rem 0 0;
  font-size: 0.85rem;
  color: #fecaca;
}

.vote-summary__qr {
  display: flex;
  align-items: center;
  justify-content: center;
}

.vote-summary__qr img {
  width: 112px;
  height: 112px;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: #fff;
  padding: 0.75rem;
}

.vote-summary__qr-loader {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112px;
  height: 112px;
  border-radius: 1.5rem;
  border: 1px dashed rgba(148, 163, 184, 0.35);
}

.vote-summary__qr-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 112px;
  height: 112px;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.2);
  color: rgba(148, 163, 184, 0.75);
  font-size: 0.75rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  text-align: center;
  padding: 0.75rem;
}

.vote-summary__qr-loader .qr-loader {
  width: 2.5rem;
  height: 2.5rem;
}

.after-vote-panel {
  padding: 1.5rem 1.5rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(15, 23, 42, 0.6);
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.45);
  text-align: center;
}

.after-vote-panel h3 {
  margin: 0;
  font-size: 1rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #f8fafc;
}

.after-vote-panel p {
  margin: 0.75rem 0 0;
  font-size: 0.9rem;
  color: rgba(226, 232, 240, 0.85);
}

.after-vote-section {
  display: flex;
  flex-direction: column;
  gap: 1.75rem;
  padding-bottom: 1.5rem;
}

.after-vote-success {
  padding: 1.75rem 1.5rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.75);
  box-shadow: 0 26px 48px rgba(15, 23, 42, 0.5);
  text-align: center;
}

.after-vote-success__eyebrow {
  margin: 0;
  font-size: 0.85rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(148, 163, 184, 0.85);
}

.after-vote-success__title {
  margin: 0.5rem 0 0;
  font-size: 1.35rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: #f8fafc;
}

.after-vote-success__thanks {
  margin: 1.25rem 0 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: rgba(191, 219, 254, 0.95);
}

.after-vote-selection {
  padding: 1.25rem 1.5rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(148, 163, 184, 0.28);
  background: linear-gradient(145deg, rgba(15, 23, 42, 0.78), rgba(30, 41, 59, 0.65));
  box-shadow: 0 22px 42px rgba(8, 15, 28, 0.6);
}

.after-vote-selection__header {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  align-items: flex-start;
}

.after-vote-selection__eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(148, 163, 184, 0.9);
}

.after-vote-selection__player {
  margin: 0;
  font-size: 1.35rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #f8fafc;
}

.after-vote-selection__hint {
  margin: 0.15rem 0 0;
  color: rgba(203, 213, 225, 0.95);
  font-size: 0.95rem;
}

.after-vote-selection__hint.emphasize {
  color: #facc15;
  font-weight: 700;
}

.after-vote-selection__action {
  margin-top: 0.5rem;
  padding: 0.85rem 1.2rem;
  border-radius: 999px;
  background: linear-gradient(135deg, #facc15, #fbbf24);
  color: #0f172a;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  border: none;
  cursor: pointer;
  box-shadow: 0 18px 32px rgba(250, 191, 36, 0.28);
  transition: transform 0.15s ease, box-shadow 0.15s ease, filter 0.15s ease;
}

.after-vote-selection__action:hover {
  transform: translateY(-1px);
  box-shadow: 0 22px 38px rgba(250, 191, 36, 0.32);
}

.after-vote-selection__action:active {
  transform: translateY(1px);
  filter: brightness(0.96);
}

.after-vote-selection__confirmation {
  margin: 0.35rem 0 0;
  color: #facc15;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.after-vote-selection__court {
  margin-top: 1rem;
  padding: 0.5rem;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: radial-gradient(circle at 20% 20%, rgba(148, 163, 184, 0.08), transparent 55%),
    radial-gradient(circle at 80% 0%, rgba(251, 191, 36, 0.08), transparent 45%),
    rgba(15, 23, 42, 0.65);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05), 0 16px 32px rgba(8, 15, 28, 0.4);
  min-height: 300px;
}

.after-vote-selection__court.is-locked {
  opacity: 0.92;
}

.feedback-cta {
  margin: 1.25rem auto 0;
  min-width: 100%;
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.15rem;
  padding: 0.95rem 1.5rem;
  border-radius: 999px;
  border: none;
  background: linear-gradient(135deg, #38bdf8, #6366f1);
  color: #0f172a;
  font-size: 1rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  box-shadow: 0 22px 40px rgba(99, 102, 241, 0.4);
  cursor: pointer;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    filter 0.2s ease;
}

.feedback-cta:hover {
  transform: translateY(-1px);
  box-shadow: 0 26px 48px rgba(99, 102, 241, 0.45);
}

.feedback-cta:active {
  transform: translateY(1px);
  filter: brightness(0.95);
}

.feedback-cta__label {
  font-size: 0.95rem;
  letter-spacing: 0.08em;
}

.feedback-cta__time {
  font-size: 0.75rem;
  letter-spacing: 0.12em;
  font-weight: 600;
}

@media (min-width: 640px) {
  .feedback-cta {
    min-width: auto;
    padding: 0.95rem 2.5rem;
  }
}

.feedback-modal {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: calc(1.5rem + env(safe-area-inset-top, 0px)) 1.25rem
    calc(1.5rem + env(safe-area-inset-bottom, 0px));
  background: rgba(8, 15, 28, 0.78);
  backdrop-filter: blur(10px);
}

.feedback-modal__panel {
  width: min(440px, 100%);
  max-height: 100%;
  padding: 1.5rem 1.5rem 1.75rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.95);
  box-shadow: 0 30px 60px rgba(8, 15, 28, 0.6);
  color: #f8fafc;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  overflow-y: auto;
}

.feedback-modal__header {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  position: relative;
}

.feedback-modal__close {
  position: absolute;
  top: 0;
  right: 0;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.85);
  color: rgba(226, 232, 240, 0.95);
  font-size: 1.5rem;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    filter 0.2s ease;
}

.feedback-modal__close:hover {
  filter: brightness(1.1);
}

.feedback-modal__step {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  color: rgba(148, 163, 184, 0.8);
}

.feedback-modal__progress {
  width: 100%;
  height: 6px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.25);
  overflow: hidden;
}

.feedback-modal__progress-bar {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(135deg, #38bdf8, #6366f1);
  transition: width 0.3s ease;
}

.feedback-modal__body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.feedback-modal__title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 700;
  line-height: 1.35;
}

.feedback-modal__hint {
  margin: 0;
  font-size: 0.85rem;
  color: rgba(191, 219, 254, 0.75);
}

.feedback-modal__options {
  display: grid;
  gap: 0.75rem;
}

.feedback-modal__option {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 1rem 1.1rem;
  border-radius: 1.25rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(30, 41, 59, 0.65);
  color: rgba(226, 232, 240, 0.95);
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  transition:
    transform 0.15s ease,
    border-color 0.15s ease,
    background 0.15s ease;
}

.feedback-modal__option:hover {
  transform: translateY(-1px);
  border-color: rgba(148, 163, 184, 0.5);
}

.feedback-modal__option.active {
  border-color: rgba(96, 165, 250, 0.8);
  background: linear-gradient(
    135deg,
    rgba(56, 189, 248, 0.25),
    rgba(99, 102, 241, 0.3)
  );
  box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.35);
}

.feedback-modal__option-icon {
  font-size: 1.75rem;
}

.feedback-modal__option-label {
  flex: 1;
  text-align: left;
}

.feedback-modal__optional {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.feedback-modal__input {
  width: 100%;
  padding: 0.85rem 1rem;
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.6);
  color: #f8fafc;
  font-size: 0.95rem;
}

.feedback-modal__input:focus {
  outline: none;
  border-color: rgba(94, 234, 212, 0.85);
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.25);
}

.feedback-modal__counter {
  align-self: flex-end;
  font-size: 0.75rem;
  color: rgba(148, 163, 184, 0.75);
}

.feedback-modal__error {
  margin: 0.25rem 0 0;
  font-size: 0.85rem;
  font-weight: 600;
  color: #fca5a5;
}

.feedback-modal__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.feedback-modal__back {
  border: none;
  background: none;
  color: rgba(148, 163, 184, 0.8);
  font-size: 0.85rem;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  cursor: pointer;
  padding: 0.5rem 0.75rem;
  border-radius: 999px;
  transition: color 0.2s ease;
}

.feedback-modal__back:hover {
  color: rgba(226, 232, 240, 0.95);
}

.feedback-modal__footer-actions {
  display: flex;
  gap: 0.75rem;
  margin-left: auto;
}

.feedback-modal__footer-actions.is-hidden {
  display: none;
}

.feedback-modal__skip,
.feedback-modal__submit {
  border-radius: 999px;
  padding: 0.65rem 1.4rem;
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    filter 0.2s ease;
}

.feedback-modal__skip {
  background: rgba(148, 163, 184, 0.2);
  border: 1px solid rgba(148, 163, 184, 0.45);
  color: rgba(226, 232, 240, 0.85);
}

.feedback-modal__skip:hover {
  background: rgba(148, 163, 184, 0.3);
}

.feedback-modal__submit {
  border: none;
  background: linear-gradient(135deg, #38bdf8, #6366f1);
  color: #0f172a;
  box-shadow: 0 18px 36px rgba(99, 102, 241, 0.45);
}

.feedback-modal__submit:hover {
  transform: translateY(-1px);
  box-shadow: 0 22px 44px rgba(99, 102, 241, 0.5);
}

.feedback-modal__submit:active {
  transform: translateY(1px);
  filter: brightness(0.95);
}

.feedback-modal__skip:disabled,
.feedback-modal__submit:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.feedback-modal-fade-enter-active,
.feedback-modal-fade-leave-active {
  transition: opacity 0.25s ease;
}

.feedback-modal-fade-enter-from,
.feedback-modal-fade-leave-to {
  opacity: 0;
}

@media (min-width: 640px) {
  .vote-summary {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 2rem;
  }

  .vote-summary__content {
    flex: 1;
  }

  .vote-summary__qr {
    flex-shrink: 0;
  }
}

.inactive-panel {
  width: 100%;
  max-width: 480px;
  padding: 2.5rem 2rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.65);
  box-shadow: 0 30px 60px rgba(15, 23, 42, 0.55);
}

.players-message {
  margin: 2rem auto;
  max-width: 420px;
  padding: 1.5rem;
  border-radius: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.55);
  text-align: center;
  font-size: 0.95rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #e2e8f0;
}

.players-message.error {
  border-color: rgba(248, 113, 113, 0.35);
  background: rgba(127, 29, 29, 0.45);
  color: #fee2e2;
}

.important-notice {
  margin-top: 1.75rem;
  padding: 1.5rem 1.25rem;
  border-radius: 1.75rem;
  border: 1px solid rgba(250, 204, 21, 0.5);
  background: rgba(30, 64, 175, 0.35);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.45);
  text-align: center;
}

.inactive-panel h2 {
  margin: 0;
}

.inactive-panel p {
  margin: 0;
  line-height: 1.6;
}

.qr-loader {
  width: 3rem;
  height: 3rem;
  border-radius: 9999px;
  border: 4px solid rgba(148, 163, 184, 0.25);
  border-top-color: #fbbf24;
  animation: qr-spin 0.9s linear infinite;
}

.vote-counter {
  margin-top: -0.5rem;
  margin-bottom: 1rem;
  padding: 1.75rem 1.5rem;
  border-radius: 2rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: linear-gradient(
    145deg,
    rgba(15, 23, 42, 0.9),
    rgba(30, 41, 59, 0.65)
  );
  box-shadow: 0 28px 48px rgba(15, 23, 42, 0.5);
}

.vote-counter__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.vote-counter__title {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.35em;
  text-transform: uppercase;
  color: #e2e8f0;
}

.vote-counter__spinner {
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 9999px;
  border: 3px solid rgba(148, 163, 184, 0.25);
  border-top-color: #38bdf8;
  animation: counter-spin 0.8s linear infinite;
}

.vote-counter__value {
  margin: 1rem 0 0;
  font-size: clamp(2.5rem, 6vw, 3.25rem);
  font-weight: 700;
  letter-spacing: 0.08em;
  color: #fbbf24;
  text-shadow: 0 12px 24px rgba(251, 191, 36, 0.35);
}

.vote-counter__message {
  margin: 0.75rem 0 0;
  font-size: 0.85rem;
  color: #cbd5f5;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.vote-counter__message.error {
  color: #fecaca;
}

@keyframes qr-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes counter-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>

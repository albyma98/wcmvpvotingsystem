<template>
  <div class="events-section">

    <!-- ───── CREATE FORM ───── -->
    <div class="create-panel">
      <div class="create-panel__header" @click="formOpen = !formOpen">
        <div class="create-panel__title">
          <span class="create-panel__icon">＋</span>
          <span>Crea nuovo evento</span>
        </div>
        <div class="create-panel__meta">
          <span v-if="!hasEnoughTeams" class="badge-warning">Aggiungi squadre prima</span>
          <span class="create-panel__chevron" :class="{ open: formOpen }">›</span>
        </div>
      </div>

      <transition name="slide-down">
        <div v-if="formOpen" class="create-panel__body">
          <p v-if="!hasEnoughTeams" class="alert alert--info">
            Aggiungi almeno due squadre dalla sezione "Squadre" per creare un evento.
          </p>

          <form id="event-create-form" @submit.prevent="$emit('create-event')" class="create-form">

            <!-- Row 1: Teams + Datetime/Location -->
            <div class="create-form__grid">
              <div class="field-group">
                <label class="field-label">Squadra di casa</label>
                <input
                  :value="teamInputs.home"
                  type="text"
                  list="admin-team-options"
                  :disabled="!hasEnoughTeams"
                  placeholder="Seleziona squadra…"
                  class="field-input"
                  @change="$emit('team-input', 'home', $event.target.value)"
                  @blur="$emit('team-input', 'home', $event.target.value)"
                />
              </div>

              <div class="vs-divider">VS</div>

              <div class="field-group">
                <label class="field-label">Squadra ospite</label>
                <input
                  :value="teamInputs.away"
                  type="text"
                  list="admin-team-options"
                  :disabled="!hasEnoughTeams"
                  placeholder="Seleziona squadra…"
                  class="field-input"
                  @change="$emit('team-input', 'away', $event.target.value)"
                  @blur="$emit('team-input', 'away', $event.target.value)"
                />
              </div>

              <datalist id="admin-team-options">
                <option v-for="t in teams" :key="t.id" :value="teamOptionValue(t)" />
              </datalist>
            </div>

            <div class="create-form__row2">
              <div class="field-group">
                <label class="field-label">Data e ora</label>
                <input
                  :value="newEvent.start_datetime"
                  type="datetime-local"
                  :disabled="!hasEnoughTeams"
                  required
                  class="field-input"
                  @input="$emit('update:new-event', 'start_datetime', $event.target.value)"
                />
              </div>
              <div class="field-group">
                <label class="field-label">Location</label>
                <input
                  :value="newEvent.location"
                  type="text"
                  placeholder="Es. Palazzetto dello Sport"
                  :disabled="!hasEnoughTeams"
                  class="field-input"
                  @input="$emit('update:new-event', 'location', $event.target.value)"
                />
              </div>
            </div>

            <!-- Toggles: Pre-vote -->
            <div class="options-block">
              <div class="options-block__header" @click="preVoteOpen = !preVoteOpen">
                <span class="options-block__dot options-block__dot--cyan"></span>
                <span class="options-block__label">Esperienze pre-voto</span>
                <span class="options-block__count">3 opzioni</span>
                <span class="options-block__chevron" :class="{ open: preVoteOpen }">›</span>
              </div>
              <transition name="slide-down">
                <div v-if="preVoteOpen" class="options-block__body">
                  <label class="toggle-row">
                    <span class="toggle-row__text">Sponsor a bordo campo</span>
                    <input type="checkbox" :checked="newEvent.show_pre_vote_sponsors" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_pre_vote_sponsors', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Sponsor a fondo campo</span>
                    <input type="checkbox" :checked="newEvent.show_pre_vote_bottom_sponsors" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_pre_vote_bottom_sponsors', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Totale voti in tempo reale</span>
                    <input type="checkbox" :checked="newEvent.show_vote_counter" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_vote_counter', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                </div>
              </transition>
            </div>

            <!-- Toggles: Post-vote -->
            <div class="options-block">
              <div class="options-block__header" @click="postVoteOpen = !postVoteOpen">
                <span class="options-block__dot options-block__dot--purple"></span>
                <span class="options-block__label">Esperienze post-voto</span>
                <span class="options-block__count">4 opzioni</span>
                <span class="options-block__chevron" :class="{ open: postVoteOpen }">›</span>
              </div>
              <transition name="slide-down">
                <div v-if="postVoteOpen" class="options-block__body">
                  <label class="toggle-row">
                    <span class="toggle-row__text">Andamento dei voti</span>
                    <input type="checkbox" :checked="newEvent.show_vote_trend" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_vote_trend', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Selfie MVP</span>
                    <input type="checkbox" :checked="newEvent.show_selfie" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_selfie', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Mini-gioco riflessi</span>
                    <input type="checkbox" :checked="newEvent.show_reaction_test" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_reaction_test', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Sondaggio feedback</span>
                    <input type="checkbox" :checked="newEvent.show_feedback_survey" :disabled="!hasEnoughTeams"
                      @change="$emit('update:new-event', 'show_feedback_survey', $event.target.checked)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                </div>
              </transition>
            </div>

            <!-- Premi -->
            <div class="options-block">
              <div class="options-block__header" @click="newPrizesOpen = !newPrizesOpen">
                <span class="options-block__dot options-block__dot--gold"></span>
                <span class="options-block__label">Premi in palio</span>
                <span class="options-block__count">{{ newEventPrizes.length }} {{ newEventPrizes.length === 1 ? 'premio' : 'premi' }}</span>
                <span class="options-block__chevron" :class="{ open: newPrizesOpen }">›</span>
              </div>
              <transition name="slide-down">
                <div v-if="newPrizesOpen" class="options-block__body">
                  <div v-for="(prize, i) in newEventPrizes" :key="i" class="prize-row">
                    <div class="prize-row__num">#{{ i + 1 }}</div>
                    <input v-model.trim="prize.name" type="text" :placeholder="`Premio ${i + 1}`" class="field-input" :disabled="!hasEnoughTeams" />
                    <input v-model.trim="prize.winSmsText" type="text" placeholder="Testo SMS vittoria" class="field-input" :disabled="!hasEnoughTeams" />
                    <button class="btn-icon btn-icon--danger" type="button" :disabled="newEventPrizes.length <= 1" @click="$emit('remove-new-prize', i)" title="Rimuovi">✕</button>
                  </div>
                  <button class="btn-ghost" type="button" @click="$emit('add-new-prize')" :disabled="!hasEnoughTeams">＋ Aggiungi premio</button>
                </div>
              </transition>
            </div>

            <!-- Footer -->
            <div class="create-form__footer">
              <div v-if="lastCreatedEventLink" class="event-link-banner">
                <span>✓ Evento creato!</span>
                <a :href="lastCreatedEventLink" target="_blank" rel="noopener">{{ lastCreatedEventLink }}</a>
                <button class="btn-ghost btn-ghost--sm" type="button" @click="$emit('copy-link', lastCreatedEventLink)">Copia link</button>
              </div>
              <div class="create-form__actions">
                <button class="btn-deactivate" type="button" @click="$emit('deactivate-events')" :disabled="!activeEventId || isDisablingEvents">
                  {{ isDisablingEvents ? 'Disattivazione…' : 'Disattiva tutti' }}
                </button>
                <button class="btn-create" type="submit" :disabled="!hasEnoughTeams">
                  <span>Crea evento</span>
                  <span class="btn-create__arrow">→</span>
                </button>
              </div>
            </div>

          </form>
        </div>
      </transition>
    </div>

    <!-- ───── EVENT LIST ───── -->
    <div class="events-list">
      <p v-if="!visibleEvents.length" class="empty-state">
        Nessun evento attivo. Creane uno qui sopra.
      </p>

      <div
        v-for="event in visibleEvents"
        :key="event.id"
        class="event-card"
        :class="{
          'event-card--active': event.is_active,
          'event-card--closed': event.is_active && event.votes_closed,
        }"
      >
        <!-- ── Card Header ── -->
        <div class="event-card__header">
          <div class="event-card__identity">
            <h3 class="event-card__title">{{ eventLabel(event) }}</h3>
            <div class="event-card__meta">
              <span class="event-card__date">{{ formatEventDate(event.start_datetime) }}</span>
              <span v-if="event.location" class="event-card__sep">·</span>
              <span v-if="event.location" class="event-card__location">{{ event.location }}</span>
            </div>
          </div>
          <div class="event-card__badges">
            <span v-if="event.is_active && !event.votes_closed" class="badge badge--active">
              <span class="badge__dot"></span>Live
            </span>
            <span v-if="event.is_active && event.votes_closed" class="badge badge--closed">Chiuse</span>
            <span v-if="!event.is_active" class="badge badge--idle">Inattivo</span>
          </div>
        </div>

        <!-- ── Quick Actions ── -->
        <div class="event-card__actions">
          <button
            class="action-btn action-btn--success"
            type="button"
            :disabled="event.is_active || updatingEventId === event.id"
            @click="$emit('activate-event', event.id)"
          >
            <span v-if="event.is_active">✓ Attivo</span>
            <span v-else-if="updatingEventId === event.id">Attivazione…</span>
            <span v-else>Attiva</span>
          </button>

          <button class="action-btn action-btn--secondary" type="button" @click="$emit('open-vote', event.id)">
            ↗ Pagina voto
          </button>

          <button
            class="action-btn action-btn--warning"
            type="button"
            :disabled="concludingEventId === event.id"
            @click="$emit('conclude-event', event.id)"
          >
            <span v-if="concludingEventId === event.id">Conclusione…</span>
            <span v-else>Concludi</span>
          </button>

          <button class="action-btn action-btn--danger" type="button" @click="$emit('delete-event', event.id)">
            Elimina
          </button>

          <div class="action-btn__link-copy">
            <code class="event-link">{{ buildEventLink(event.id) }}</code>
            <button class="btn-icon" type="button" @click="$emit('copy-link', buildEventLink(event.id))" title="Copia link">⎘</button>
          </div>
        </div>

        <!-- ── Tabs ── -->
        <div class="event-tabs">
          <div class="event-tabs__bar">
            <button
              v-for="tab in eventTabList"
              :key="tab.id"
              type="button"
              class="event-tabs__btn"
              :class="{ active: (activeTab[event.id] || 'settings') === tab.id }"
              @click="setTab(event.id, tab.id)"
            >
              <span class="event-tabs__icon">{{ tab.icon }}</span>
              {{ tab.label }}
            </button>
          </div>

          <div class="event-tabs__panel">

            <!-- TAB: Impostazioni -->
            <div v-if="(activeTab[event.id] || 'settings') === 'settings'" class="tab-settings">
              <div class="toggles-grid">
                <div class="toggles-col">
                  <div class="toggles-col__label">Pre-voto</div>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Sponsor bordo campo</span>
                    <input type="checkbox" v-model="event.show_pre_vote_sponsors" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Sponsor fondo campo</span>
                    <input type="checkbox" v-model="event.show_pre_vote_bottom_sponsors" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Contatore voti live</span>
                    <input type="checkbox" v-model="event.show_vote_counter" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                </div>
                <div class="toggles-col">
                  <div class="toggles-col__label">Post-voto</div>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Andamento voti</span>
                    <input type="checkbox" v-model="event.show_vote_trend" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Selfie MVP</span>
                    <input type="checkbox" v-model="event.show_selfie" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Mini-gioco riflessi</span>
                    <input type="checkbox" v-model="event.show_reaction_test" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                  <label class="toggle-row">
                    <span class="toggle-row__text">Sondaggio feedback</span>
                    <input type="checkbox" v-model="event.show_feedback_survey" :disabled="isSavingPrizesFor(event.id)" />
                    <span class="toggle-track"><span class="toggle-thumb"></span></span>
                  </label>
                </div>
              </div>
              <div class="tab-save-row">
                <button class="btn-save" type="button" @click="$emit('save-prizes', event)" :disabled="isSavingPrizesFor(event.id)">
                  {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva impostazioni' }}
                </button>
              </div>
            </div>

            <!-- TAB: Premi -->
            <div v-else-if="(activeTab[event.id] || 'settings') === 'prizes'" class="tab-prizes">
              <div class="prizes-list">
                <div
                  v-for="(prize, i) in prizeDraftsFor(event.id)"
                  :key="prize.id || i"
                  class="prize-row"
                >
                  <div class="prize-row__num">#{{ i + 1 }}</div>
                  <input v-model="prize.name" type="text" :placeholder="`Premio ${i + 1}`" class="field-input" :disabled="isSavingPrizesFor(event.id)" />
                  <input v-model="prize.winSmsText" type="text" placeholder="Testo SMS vincitore" class="field-input" :disabled="isSavingPrizesFor(event.id)" />
                  <span v-if="prize.winner" class="prize-winner-badge">
                    🏆 {{ prizeWinnerLabel(prize) }}
                  </span>
                  <button
                    class="btn-icon btn-icon--danger"
                    type="button"
                    :disabled="prize.winner || prizeDraftsFor(event.id).length <= 1 || isSavingPrizesFor(event.id)"
                    @click="$emit('remove-prize', event.id, i)"
                    title="Rimuovi"
                  >✕</button>
                </div>
              </div>
              <div class="tab-save-row">
                <button class="btn-ghost" type="button" @click="$emit('add-prize', event.id)" :disabled="isSavingPrizesFor(event.id)">＋ Aggiungi premio</button>
                <button class="btn-save" type="button" @click="$emit('save-prizes', event)" :disabled="isSavingPrizesFor(event.id)">
                  {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva premi' }}
                </button>
              </div>
              <p v-if="eventPrizeErrors[event.id]" class="tab-error">{{ eventPrizeErrors[event.id] }}</p>
            </div>

            <!-- TAB: Sondaggio -->
            <div v-else-if="(activeTab[event.id] || 'settings') === 'feedback'" class="tab-feedback">
              <div class="feedback-questions">
                <div
                  v-for="question in feedbackDraftFor(event.id).questions"
                  :key="`q-${event.id}-${question.id}`"
                  class="feedback-question"
                >
                  <div class="feedback-question__header">
                    <span class="feedback-question__id">{{ question.id }}</span>
                    <input v-model="question.title" type="text" :placeholder="`Domanda: ${question.id}`" class="field-input" :disabled="isSavingPrizesFor(event.id)" />
                  </div>
                  <div class="feedback-answers">
                    <label
                      v-for="answer in question.answers"
                      :key="`a-${event.id}-${question.id}-${answer.value}`"
                      class="feedback-answer"
                    >
                      <span class="feedback-answer__icon" v-if="answer.icon">{{ answer.icon }}</span>
                      <code class="feedback-answer__code">{{ answer.value }}</code>
                      <input v-model="answer.label" type="text" :placeholder="`Risposta: ${answer.value}`" class="field-input" :disabled="isSavingPrizesFor(event.id)" />
                    </label>
                  </div>
                </div>
              </div>
              <div class="field-group" style="margin-top:1rem;">
                <label class="field-label">Domanda suggerimenti (opzionale)</label>
                <textarea v-model="feedbackDraftFor(event.id).suggestionPrompt" rows="2" maxlength="120" class="field-input" :disabled="isSavingPrizesFor(event.id)" placeholder="Testo domanda aperta…"></textarea>
              </div>
              <div class="tab-save-row">
                <button class="btn-save" type="button" @click="$emit('save-prizes', event)" :disabled="isSavingPrizesFor(event.id)">
                  {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva sondaggio' }}
                </button>
              </div>
              <p v-if="eventFeedbackErrors[event.id]" class="tab-error">{{ eventFeedbackErrors[event.id] }}</p>
            </div>

            <!-- TAB: Quiz -->
            <div v-else-if="(activeTab[event.id] || 'settings') === 'quiz'" class="tab-quiz">
              <div class="quiz-status-bar">
                <span class="quiz-status-label">Stato quiz:</span>
                <button
                  class="action-btn"
                  :class="quizDraftFor(event.id).enabled ? 'action-btn--success' : 'action-btn--secondary'"
                  type="button"
                  @click="quizDraftFor(event.id).enabled = true"
                >Abilitato</button>
                <button
                  class="action-btn"
                  :class="!quizDraftFor(event.id).enabled ? 'action-btn--danger' : 'action-btn--secondary'"
                  type="button"
                  @click="quizDraftFor(event.id).enabled = false"
                >Disabilitato</button>
                <div class="quiz-status-bar__spacer"></div>
                <button class="btn-ghost btn-ghost--sm" type="button" @click="$emit('reload-quiz', event.id)">↻ Ricarica</button>
                <button class="btn-save" type="button" @click="$emit('save-quiz', event.id)">Salva config</button>
              </div>

              <div class="quiz-questions">
                <div
                  v-for="(q, qi) in quizQuestionsFor(event.id)"
                  :key="q.id || qi"
                  class="quiz-question"
                >
                  <div class="quiz-question__header">
                    <span class="quiz-question__num">Q{{ qi + 1 }}</span>
                    <input v-model="q.question_text" type="text" placeholder="Testo della domanda" class="field-input quiz-question__text" />
                    <div class="quiz-question__meta">
                      <label class="quiz-meta-field">
                        <span>Corretta</span>
                        <input type="number" min="0" :max="q.answers.length - 1" v-model.number="q.correct_index" class="field-input field-input--sm" />
                      </label>
                      <label class="quiz-meta-field">
                        <span>Ordine</span>
                        <input type="number" min="0" v-model.number="q.order_index" class="field-input field-input--sm" />
                      </label>
                    </div>
                  </div>
                  <div class="quiz-answers">
                    <div v-for="(_, ai) in q.answers" :key="ai" class="quiz-answer">
                      <span
                        class="quiz-answer__badge"
                        :class="{ correct: q.correct_index === ai }"
                      >{{ ai }}</span>
                      <input v-model="q.answers[ai]" type="text" :placeholder="`Risposta ${ai + 1}`" class="field-input" />
                    </div>
                  </div>
                  <div class="quiz-question__actions">
                    <button class="btn-ghost btn-ghost--sm" type="button" @click="$emit('add-answer', event.id, qi)" :disabled="q.answers.length >= 4">＋ Risposta</button>
                    <button class="btn-ghost btn-ghost--sm btn-ghost--danger" type="button" @click="$emit('remove-answer', event.id, qi)" :disabled="q.answers.length <= 2">－ Risposta</button>
                    <button class="btn-save btn-save--sm" type="button" @click="$emit('save-question', event.id, q)">Salva</button>
                    <button class="action-btn action-btn--danger" type="button" @click="$emit('delete-question', event.id, q)">Elimina</button>
                  </div>
                </div>

                <button class="btn-ghost" type="button" @click="$emit('add-question', event.id)">＋ Aggiungi domanda</button>
              </div>
            </div>

            <!-- TAB: Stories -->
            <div v-else-if="(activeTab[event.id] || 'settings') === 'stories'" class="tab-stories">
              <div class="stories-toolbar">
                <span v-if="isStoriesLoading(event.id)" class="loading-indicator">
                  <span class="spinner"></span> Caricamento…
                </span>
                <button class="btn-ghost btn-ghost--sm" type="button" @click="$emit('reload-stories', event.id)" :disabled="isStoriesLoading(event.id)">↻ Ricarica</button>
                <button class="btn-ghost" type="button" @click="$emit('add-story', event.id)">＋ Aggiungi story</button>
              </div>

              <div class="stories-list">
                <div
                  v-for="(story, si) in storiesForEvent(event.id)"
                  :key="story.id || si"
                  class="story-row"
                >
                  <div class="story-row__thumb">
                    <img v-if="story.thumbnail_url" :src="story.thumbnail_url" :alt="story.player_name || 'Story'" />
                    <div v-else class="story-thumb-placeholder">▶</div>
                  </div>
                  <div class="story-row__fields">
                    <div class="story-row__grid">
                      <input v-model="story.player_name" type="text" placeholder="Giocatore (opz.)" class="field-input" />
                      <input v-model="story.title" type="text" placeholder="Titolo (opz.)" class="field-input" />
                      <input v-model="story.thumbnail_url" type="url" placeholder="Thumbnail URL" class="field-input" />
                      <div class="story-row__video">
                        <input v-model="story.video_url" type="url" placeholder="Video URL" class="field-input" />
                        <input
                          :id="`story-video-${event.id}-${si}`"
                          type="file"
                          accept="video/mp4,video/webm,video/quicktime"
                          style="display:none"
                          @change="$emit('upload-story-video', event.id, story, si, $event)"
                        />
                        <button
                          class="btn-ghost btn-ghost--sm"
                          type="button"
                          @click="triggerVideoPicker(event.id, si)"
                          :disabled="isStoryVideoUploading(event.id, si) || isStoriesSaving(event.id)"
                        >
                          {{ isStoryVideoUploading(event.id, si) ? 'Upload…' : '⬆ Carica video' }}
                        </button>
                      </div>
                    </div>
                    <div class="story-row__footer">
                      <label class="toggle-row toggle-row--inline">
                        <input type="checkbox" v-model="story.is_active" />
                        <span class="toggle-track"><span class="toggle-thumb"></span></span>
                        <span class="toggle-row__text">Attiva</span>
                      </label>
                      <div class="story-row__order">
                        <button class="btn-icon" type="button" @click="$emit('move-story', event.id, si, -1)" :disabled="si === 0">↑</button>
                        <button class="btn-icon" type="button" @click="$emit('move-story', event.id, si, 1)" :disabled="si === storiesForEvent(event.id).length - 1">↓</button>
                      </div>
                      <button class="btn-save btn-save--sm" type="button" @click="$emit('save-story', event.id, story, si)" :disabled="isStoriesSaving(event.id) || isStoryVideoUploading(event.id, si)">Salva</button>
                      <button class="action-btn action-btn--danger" type="button" @click="$emit('delete-story', event.id, story, si)" :disabled="isStoriesSaving(event.id)">Elimina</button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue';

const props = defineProps({
  visibleEvents: Array,
  teams: Array,
  hasEnoughTeams: Boolean,
  teamInputs: Object,
  newEvent: Object,
  newEventPrizes: Array,
  activeEventId: Number,
  updatingEventId: Number,
  concludingEventId: Number,
  isDisablingEvents: Boolean,
  lastCreatedEventLink: String,
  savingEventPrizes: Number,
  eventPrizeDrafts: Object,
  eventPrizeErrors: Object,
  eventFeedbackDrafts: Object,
  eventFeedbackErrors: Object,
  quizConfigsByEvent: Object,
  quizQuestionsByEvent: Object,
  eventStoriesById: Object,
  eventStoriesLoading: Object,
  eventStoriesSaving: Object,
  eventStoriesUploading: Object,
});

defineEmits([
  'create-event', 'delete-event', 'activate-event', 'conclude-event', 'deactivate-events',
  'open-vote', 'copy-link',
  'team-input', 'update:new-event',
  'add-new-prize', 'remove-new-prize',
  'add-prize', 'remove-prize', 'save-prizes',
  'add-question', 'delete-question', 'save-question', 'reload-quiz', 'save-quiz',
  'add-answer', 'remove-answer',
  'add-story', 'delete-story', 'save-story', 'reload-stories',
  'move-story', 'upload-story-video',
]);

// UI state
const formOpen = ref(true);
const preVoteOpen = ref(false);
const postVoteOpen = ref(false);
const newPrizesOpen = ref(false);
const activeTab = reactive({});

const eventTabList = [
  { id: 'settings', icon: '⚙', label: 'Impostazioni' },
  { id: 'prizes',   icon: '🎁', label: 'Premi' },
  { id: 'feedback', icon: '📋', label: 'Sondaggio' },
  { id: 'quiz',     icon: '🧩', label: 'Quiz' },
  { id: 'stories',  icon: '▶', label: 'Stories' },
];

function setTab(eventId, tabId) {
  activeTab[eventId] = tabId;
}

function isSavingPrizesFor(eventId) {
  return props.savingEventPrizes === eventId;
}

function prizeDraftsFor(eventId) {
  return props.eventPrizeDrafts?.[eventId] || [];
}

function feedbackDraftFor(eventId) {
  return props.eventFeedbackDrafts?.[eventId] || { questions: [], suggestionPrompt: '' };
}

function quizDraftFor(eventId) {
  return props.quizConfigsByEvent?.[eventId] || { enabled: false };
}

function quizQuestionsFor(eventId) {
  return props.quizQuestionsByEvent?.[eventId] || [];
}

function storiesForEvent(eventId) {
  return props.eventStoriesById?.[eventId] || [];
}

function isStoriesLoading(eventId) {
  return props.eventStoriesLoading?.[eventId] === true;
}

function isStoriesSaving(eventId) {
  return props.eventStoriesSaving?.[eventId] === true;
}

function isStoryVideoUploading(eventId, index) {
  return props.eventStoriesUploading?.[`${eventId}-${index}`] === true;
}

function prizeWinnerLabel(prize) {
  return prize?.winner?.ticketCode || '';
}

function teamOptionValue(team) {
  return `${team.name} (#${team.id})`;
}

function eventLabel(event) {
  const t1 = props.teams?.find(t => t.id === event.team1_id);
  const t2 = props.teams?.find(t => t.id === event.team2_id);
  const n1 = t1?.name || event.team1_name || '—';
  const n2 = t2?.name || event.team2_name || '—';
  return `${n1} vs ${n2}`;
}

function formatEventDate(value) {
  if (!value) return 'Data da definire';
  const d = new Date(value);
  if (isNaN(d)) return value.replace('T', ' ');
  return d.toLocaleString('it-IT', { dateStyle: 'medium', timeStyle: 'short' });
}

function buildEventLink(eventId) {
  const url = new URL(window.location.origin);
  if (eventId) url.searchParams.set('eventId', String(eventId));
  return url.toString();
}

function triggerVideoPicker(eventId, index) {
  const el = document.getElementById(`story-video-${eventId}-${index}`);
  if (el) el.click();
}
</script>

<style scoped>
/* ── Fonts (same as admin theme) ── */
@import url('https://fonts.googleapis.com/css2?family=Barlow+Condensed:wght@600;700;800&family=IBM+Plex+Sans:wght@400;500;600&display=swap');

.events-section {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  font-family: 'IBM Plex Sans', system-ui, sans-serif;
  color: #e2eaf6;
}

/* ─── Create Panel ─────────────────────────────────────────────── */
.create-panel {
  background: #111827;
  border: 1px solid rgba(56,189,248,0.14);
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.35);
}

.create-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.1rem 1.5rem;
  cursor: pointer;
  user-select: none;
  transition: background 0.18s;
}

.create-panel__header:hover { background: rgba(56,189,248,0.04); }

.create-panel__title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-family: 'Barlow Condensed', sans-serif;
  font-size: 1.1rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: #e2eaf6;
}

.create-panel__icon {
  width: 28px;
  height: 28px;
  background: linear-gradient(135deg, #0284c7, #4f46e5);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
  color: #fff;
  flex-shrink: 0;
}

.create-panel__meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.badge-warning {
  padding: 0.2rem 0.7rem;
  background: rgba(245,158,11,0.12);
  border: 1px solid rgba(251,191,36,0.2);
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 600;
  color: #fbbf24;
  letter-spacing: 0.04em;
}

.create-panel__chevron {
  font-size: 1.3rem;
  color: rgba(148,163,184,0.5);
  transition: transform 0.25s ease;
  display: inline-block;
}

.create-panel__chevron.open { transform: rotate(90deg); }

.create-panel__body {
  padding: 0 1.5rem 1.5rem;
  border-top: 1px solid rgba(56,189,248,0.08);
}

/* ─── Create Form ──────────────────────────────────────────────── */
.create-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding-top: 1.25rem;
}

.create-form__grid {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 0.75rem;
  align-items: end;
}

.vs-divider {
  font-family: 'Barlow Condensed', sans-serif;
  font-size: 1.1rem;
  font-weight: 800;
  color: rgba(56,189,248,0.5);
  letter-spacing: 0.1em;
  padding-bottom: 0.55rem;
  text-align: center;
  align-self: end;
}

.create-form__row2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

@media (max-width: 680px) {
  .create-form__grid { grid-template-columns: 1fr; }
  .vs-divider { display: none; }
  .create-form__row2 { grid-template-columns: 1fr; }
}

.create-form__footer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding-top: 0.5rem;
}

.create-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
}

/* ─── Options Block (collapsible) ─────────────────────────────── */
.options-block {
  border: 1px solid rgba(56,189,248,0.1);
  border-radius: 10px;
  overflow: hidden;
  background: rgba(15,23,42,0.5);
}

.options-block__header {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.75rem 1rem;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.options-block__header:hover { background: rgba(56,189,248,0.04); }

.options-block__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.options-block__dot--cyan   { background: #38bdf8; box-shadow: 0 0 6px rgba(56,189,248,0.5); }
.options-block__dot--purple { background: #818cf8; box-shadow: 0 0 6px rgba(129,140,248,0.5); }
.options-block__dot--gold   { background: #fbbf24; box-shadow: 0 0 6px rgba(251,191,36,0.5); }

.options-block__label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #e2eaf6;
  flex: 1;
}

.options-block__count {
  font-size: 0.72rem;
  color: rgba(148,163,184,0.5);
}

.options-block__chevron {
  font-size: 1.1rem;
  color: rgba(148,163,184,0.4);
  transition: transform 0.22s ease;
  display: inline-block;
}

.options-block__chevron.open { transform: rotate(90deg); }

.options-block__body {
  padding: 0.25rem 1rem 0.85rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  border-top: 1px solid rgba(56,189,248,0.07);
}

/* ─── Toggle Row ───────────────────────────────────────────────── */
.toggle-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.6rem;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
}

.toggle-row:hover { background: rgba(56,189,248,0.05); }

.toggle-row input[type="checkbox"] {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
  pointer-events: none;
}

.toggle-track {
  position: relative;
  width: 36px;
  height: 20px;
  background: rgba(148,163,184,0.2);
  border-radius: 999px;
  flex-shrink: 0;
  transition: background 0.2s;
  cursor: pointer;
}

.toggle-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 1px 4px rgba(0,0,0,0.3);
  transition: transform 0.2s ease, background 0.2s;
}

.toggle-row input[type="checkbox"]:checked ~ .toggle-track { background: linear-gradient(135deg, #0284c7, #4f46e5); }
.toggle-row input[type="checkbox"]:checked ~ .toggle-track .toggle-thumb { transform: translateX(16px); }
.toggle-row input[type="checkbox"]:disabled ~ .toggle-track { opacity: 0.4; cursor: not-allowed; }

.toggle-row--inline { flex-direction: row; gap: 0.5rem; }

.toggle-row__text {
  font-size: 0.855rem;
  color: rgba(226,234,246,0.85);
  flex: 1;
}

/* ─── Fields ───────────────────────────────────────────────────── */
.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.field-label {
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  color: rgba(148,163,184,0.65);
}

.field-input {
  width: 100%;
  padding: 0.55rem 0.85rem;
  background: rgba(15,23,42,0.8);
  border: 1px solid rgba(56,189,248,0.14);
  border-radius: 8px;
  color: #e2eaf6;
  font-family: inherit;
  font-size: 0.875rem;
  transition: border-color 0.18s, box-shadow 0.18s;
  min-height: 38px;
  box-sizing: border-box;
}

.field-input::placeholder { color: rgba(148,163,184,0.35); }
.field-input:focus { outline: none; border-color: #38bdf8; box-shadow: 0 0 0 3px rgba(56,189,248,0.12); }
.field-input:disabled { opacity: 0.45; cursor: not-allowed; }
.field-input--sm { max-width: 72px; padding: 0.4rem 0.55rem; font-size: 0.82rem; min-height: 32px; }

textarea.field-input { resize: vertical; min-height: 60px; }

/* ─── Buttons ─────────────────────────────────────────────────── */
.btn-create {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 1.6rem;
  background: linear-gradient(135deg, #0284c7, #4f46e5);
  border: none;
  border-radius: 10px;
  color: #fff;
  font-family: inherit;
  font-size: 0.9rem;
  font-weight: 700;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(2,132,199,0.35);
  transition: all 0.18s;
  letter-spacing: 0.02em;
}
.btn-create:not(:disabled):hover { box-shadow: 0 6px 28px rgba(2,132,199,0.55); transform: translateY(-1px); }
.btn-create:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-create__arrow { font-size: 1rem; }

.btn-deactivate {
  display: inline-flex;
  align-items: center;
  padding: 0.6rem 1.1rem;
  background: transparent;
  border: 1px solid rgba(148,163,184,0.2);
  border-radius: 10px;
  color: rgba(148,163,184,0.65);
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s;
}
.btn-deactivate:not(:disabled):hover { border-color: rgba(248,113,113,0.3); color: #f87171; }
.btn-deactivate:disabled { opacity: 0.3; cursor: not-allowed; }

.btn-ghost {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.5rem 0.85rem;
  background: rgba(56,189,248,0.07);
  border: 1px solid rgba(56,189,248,0.15);
  border-radius: 8px;
  color: #38bdf8;
  font-family: inherit;
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-ghost:hover { background: rgba(56,189,248,0.15); }
.btn-ghost:disabled { opacity: 0.35; cursor: not-allowed; }
.btn-ghost--sm { padding: 0.35rem 0.65rem; font-size: 0.78rem; }
.btn-ghost--danger { color: #f87171; background: rgba(248,113,113,0.07); border-color: rgba(248,113,113,0.15); }
.btn-ghost--danger:hover { background: rgba(248,113,113,0.14); }

.btn-save {
  padding: 0.55rem 1.15rem;
  background: linear-gradient(135deg, #059669, #0d9488);
  border: none;
  border-radius: 8px;
  color: #fff;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 700;
  cursor: pointer;
  box-shadow: 0 3px 14px rgba(5,150,105,0.3);
  transition: all 0.18s;
}
.btn-save:not(:disabled):hover { box-shadow: 0 5px 20px rgba(5,150,105,0.45); transform: translateY(-1px); }
.btn-save:disabled { opacity: 0.4; cursor: not-allowed; box-shadow: none; }
.btn-save--sm { padding: 0.38rem 0.8rem; font-size: 0.78rem; }

.btn-icon {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(148,163,184,0.1);
  border: 1px solid rgba(148,163,184,0.15);
  border-radius: 6px;
  color: rgba(148,163,184,0.7);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}
.btn-icon:hover { background: rgba(148,163,184,0.18); color: #e2eaf6; }
.btn-icon:disabled { opacity: 0.3; cursor: not-allowed; }
.btn-icon--danger { color: rgba(248,113,113,0.7); border-color: rgba(248,113,113,0.15); }
.btn-icon--danger:hover { background: rgba(248,113,113,0.12); color: #f87171; }
.btn-icon--danger:disabled { opacity: 0.25; }

/* ─── Prize Rows ────────────────────────────────────────────────── */
.prize-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.35rem 0;
}
.prize-row__num {
  font-family: 'Barlow Condensed', sans-serif;
  font-size: 1rem;
  font-weight: 700;
  color: rgba(251,191,36,0.65);
  width: 24px;
  flex-shrink: 0;
  text-align: center;
}
.prize-winner-badge {
  font-size: 0.75rem;
  color: #34d399;
  font-weight: 600;
  white-space: nowrap;
}

/* ─── Event Link Banner ─────────────────────────────────────────── */
.event-link-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
  padding: 0.7rem 1rem;
  background: rgba(5,150,105,0.1);
  border: 1px solid rgba(52,211,153,0.2);
  border-radius: 8px;
  font-size: 0.85rem;
  color: #34d399;
}
.event-link-banner a { color: #5eead4; word-break: break-all; }

/* ─── Alert ─────────────────────────────────────────────────────── */
.alert {
  padding: 0.75rem 1rem;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 500;
  margin-top: 0.75rem;
}
.alert--info {
  background: rgba(56,189,248,0.08);
  border: 1px solid rgba(56,189,248,0.18);
  color: #7dd3fc;
}

/* ─── Empty State ───────────────────────────────────────────────── */
.empty-state {
  text-align: center;
  color: rgba(148,163,184,0.45);
  font-size: 0.9rem;
  padding: 2.5rem;
  border: 1px dashed rgba(148,163,184,0.15);
  border-radius: 12px;
}

/* ─── Events List ───────────────────────────────────────────────── */
.events-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* ─── Event Card ────────────────────────────────────────────────── */
.event-card {
  background: #111827;
  border: 1px solid rgba(56,189,248,0.1);
  border-radius: 16px;
  overflow: hidden;
  transition: border-color 0.2s, box-shadow 0.2s;
  box-shadow: 0 4px 24px rgba(0,0,0,0.3);
}

.event-card--active {
  border-color: rgba(56,189,248,0.3);
  box-shadow: 0 6px 32px rgba(0,0,0,0.4), 0 0 0 1px rgba(56,189,248,0.1);
}

.event-card--closed {
  border-color: rgba(251,191,36,0.25);
}

/* ─── Card Header ────────────────────────────────────────────────── */
.event-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.1rem 1.5rem 0.85rem;
}

.event-card__title {
  font-family: 'Barlow Condensed', sans-serif;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #f0f6ff;
  margin: 0 0 0.3rem;
}

.event-card__meta {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.event-card__date {
  font-size: 0.82rem;
  color: rgba(148,163,184,0.65);
}

.event-card__sep {
  color: rgba(148,163,184,0.3);
}

.event-card__location {
  font-size: 0.82rem;
  color: rgba(148,163,184,0.55);
}

.event-card__badges {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-shrink: 0;
}

/* ─── Badges ─────────────────────────────────────────────────────── */
.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.22rem 0.7rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}
.badge--active {
  background: rgba(56,189,248,0.12);
  border: 1px solid rgba(56,189,248,0.25);
  color: #38bdf8;
}
.badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #38bdf8;
  animation: pulse-dot 1.5s ease infinite;
}
@keyframes pulse-dot {
  0%,100% { opacity: 1; transform: scale(1); }
  50%      { opacity: 0.5; transform: scale(0.75); }
}
.badge--closed {
  background: rgba(251,191,36,0.12);
  border: 1px solid rgba(251,191,36,0.22);
  color: #fbbf24;
}
.badge--idle {
  background: rgba(148,163,184,0.1);
  border: 1px solid rgba(148,163,184,0.15);
  color: rgba(148,163,184,0.65);
}

/* ─── Card Actions ────────────────────────────────────────────────── */
.event-card__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  padding: 0 1.5rem 0.85rem;
  border-bottom: 1px solid rgba(56,189,248,0.07);
}

.action-btn {
  padding: 0.42rem 0.9rem;
  border-radius: 7px;
  border: 1px solid transparent;
  font-family: inherit;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.action-btn:disabled { opacity: 0.35; cursor: not-allowed; pointer-events: none; }

.action-btn--success {
  background: rgba(5,150,105,0.15);
  border-color: rgba(52,211,153,0.2);
  color: #34d399;
}
.action-btn--success:hover { background: rgba(5,150,105,0.25); }

.action-btn--secondary {
  background: rgba(56,189,248,0.08);
  border-color: rgba(56,189,248,0.18);
  color: #38bdf8;
}
.action-btn--secondary:hover { background: rgba(56,189,248,0.16); }

.action-btn--warning {
  background: rgba(217,119,6,0.15);
  border-color: rgba(251,191,36,0.2);
  color: #fbbf24;
}
.action-btn--warning:hover { background: rgba(217,119,6,0.25); }

.action-btn--danger {
  background: rgba(239,68,68,0.1);
  border-color: rgba(248,113,113,0.18);
  color: #f87171;
}
.action-btn--danger:hover { background: rgba(239,68,68,0.2); }

.action-btn__link-copy {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  margin-left: auto;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.event-link {
  font-family: 'IBM Plex Mono', monospace;
  font-size: 0.73rem;
  color: rgba(148,163,184,0.5);
  background: rgba(15,23,42,0.6);
  padding: 0.25rem 0.55rem;
  border-radius: 5px;
  word-break: break-all;
}

/* ─── Tabs ───────────────────────────────────────────────────────── */
.event-tabs {
  display: flex;
  flex-direction: column;
}

.event-tabs__bar {
  display: flex;
  gap: 0;
  border-bottom: 1px solid rgba(56,189,248,0.08);
  overflow-x: auto;
  scrollbar-width: none;
}
.event-tabs__bar::-webkit-scrollbar { display: none; }

.event-tabs__btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.7rem 1.1rem;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: rgba(148,163,184,0.55);
  font-family: inherit;
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
}
.event-tabs__btn:hover { color: #e2eaf6; }
.event-tabs__btn.active {
  color: #38bdf8;
  border-bottom-color: #38bdf8;
}

.event-tabs__icon { font-size: 0.9rem; }

.event-tabs__panel {
  padding: 1.25rem 1.5rem 1.5rem;
}

/* ─── Tab: Settings ──────────────────────────────────────────────── */
.toggles-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem 2rem;
  margin-bottom: 1rem;
}

@media (max-width: 640px) {
  .toggles-grid { grid-template-columns: 1fr; }
}

.toggles-col__label {
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: rgba(148,163,184,0.4);
  margin-bottom: 0.35rem;
}

.tab-save-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.65rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(56,189,248,0.07);
  flex-wrap: wrap;
}

.tab-error {
  color: #f87171;
  font-size: 0.82rem;
  margin-top: 0.5rem;
}

/* ─── Tab: Prizes ─────────────────────────────────────────────────── */
.prizes-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

/* ─── Tab: Feedback ──────────────────────────────────────────────── */
.feedback-questions {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.feedback-question {
  background: rgba(15,23,42,0.5);
  border: 1px solid rgba(56,189,248,0.08);
  border-radius: 10px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.feedback-question__header {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.feedback-question__id {
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(56,189,248,0.6);
  flex-shrink: 0;
  min-width: 60px;
}

.feedback-answers {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.5rem;
}

.feedback-answer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.feedback-answer__icon { font-size: 1rem; flex-shrink: 0; }
.feedback-answer__code {
  font-family: monospace;
  font-size: 0.7rem;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  background: rgba(56,189,248,0.1);
  color: #38bdf8;
  flex-shrink: 0;
  white-space: nowrap;
}

/* ─── Tab: Quiz ──────────────────────────────────────────────────── */
.quiz-status-bar {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
  margin-bottom: 1.25rem;
  padding: 0.75rem 1rem;
  background: rgba(15,23,42,0.5);
  border: 1px solid rgba(56,189,248,0.08);
  border-radius: 10px;
}

.quiz-status-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: rgba(148,163,184,0.55);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.quiz-status-bar__spacer { flex: 1; }

.quiz-questions {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.quiz-question {
  background: rgba(15,23,42,0.5);
  border: 1px solid rgba(56,189,248,0.09);
  border-radius: 10px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}

.quiz-question__header {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.quiz-question__num {
  font-family: 'Barlow Condensed', sans-serif;
  font-size: 1rem;
  font-weight: 800;
  color: rgba(56,189,248,0.5);
  flex-shrink: 0;
  width: 28px;
}

.quiz-question__text { flex: 1; min-width: 180px; }

.quiz-question__meta {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.quiz-meta-field {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.72rem;
  color: rgba(148,163,184,0.55);
}

.quiz-answers {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.45rem;
}

.quiz-answer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.quiz-answer__badge {
  width: 22px;
  height: 22px;
  border-radius: 5px;
  background: rgba(148,163,184,0.12);
  color: rgba(148,163,184,0.55);
  font-size: 0.72rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
}

.quiz-answer__badge.correct {
  background: rgba(5,150,105,0.2);
  color: #34d399;
  border: 1px solid rgba(52,211,153,0.25);
}

.quiz-question__actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  padding-top: 0.25rem;
  border-top: 1px solid rgba(56,189,248,0.06);
}

/* ─── Tab: Stories ──────────────────────────────────────────────── */
.stories-toolbar {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.loading-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.82rem;
  color: rgba(148,163,184,0.55);
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(56,189,248,0.2);
  border-top-color: #38bdf8;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.stories-list {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.story-row {
  display: flex;
  gap: 0.85rem;
  background: rgba(15,23,42,0.5);
  border: 1px solid rgba(56,189,248,0.09);
  border-radius: 10px;
  padding: 0.85rem;
  align-items: flex-start;
}

.story-row__thumb {
  width: 52px;
  height: 52px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  background: rgba(15,23,42,0.8);
  border: 1px solid rgba(56,189,248,0.12);
  display: flex;
  align-items: center;
  justify-content: center;
}

.story-row__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.story-thumb-placeholder {
  font-size: 1.25rem;
  color: rgba(56,189,248,0.3);
}

.story-row__fields {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.story-row__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}

@media (max-width: 600px) {
  .story-row { flex-direction: column; }
  .story-row__grid { grid-template-columns: 1fr; }
}

.story-row__video {
  display: flex;
  gap: 0.4rem;
  align-items: center;
  grid-column: 1 / -1;
}

.story-row__video .field-input { flex: 1; }

.story-row__footer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  padding-top: 0.4rem;
  border-top: 1px solid rgba(56,189,248,0.06);
}

.story-row__order {
  display: flex;
  gap: 0.3rem;
  margin-left: 0.5rem;
}

/* ─── Transitions ───────────────────────────────────────────────── */
.slide-down-enter-active,
.slide-down-leave-active {
  transition: max-height 0.28s ease, opacity 0.22s ease;
  max-height: 1200px;
  overflow: hidden;
}
.slide-down-enter-from,
.slide-down-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>

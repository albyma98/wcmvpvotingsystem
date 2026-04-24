<template>
  <div class="admin-portal">
    <section v-if="!isAuthenticated" class="card login-card">
      <header class="admin-header login-header">
        <h1>Area amministratore</h1>
        <p class="subtitle">Gestisci eventi, squadre e votazioni MVP</p>
        <p v-if="organizationSlug" class="context-badge">Società: {{ organizationSlug }}</p>
      </header>
      <h2>Accedi</h2>
      <form @submit.prevent="login" class="form-grid">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button class="btn primary" type="submit" :disabled="isLoggingIn">{{ isLoggingIn ? "Accesso in corso…" : "Entra" }}</button>
      </form>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </section>

    <AdminLayout v-else :mobile-open="isSidebarOpen" @close-mobile="isSidebarOpen = false">
      <template #sidebar>
        <AdminSidebar
          :groups="navigationGroups"
          :active-section="section"
          :organization-slug="organizationSlug"
          @select="selectSection"
          @lottery="goToLottery"
          @logout="logout"
        />
      </template>
      <template #header>
        <AdminHeader :title="currentSectionTitle" :section-group="currentSectionGroup" @toggle-menu="isSidebarOpen = !isSidebarOpen">
          <template #actions>
            <span class="header-user">{{ activeUsername }}</span>
          </template>
        </AdminHeader>
      </template>

      <div class="portal-content">
        <p v-if="globalError" class="error">{{ globalError }}</p>

        <section v-if="section === 'dashboard'" class="card dashboard-grid">
          <article class="dashboard-card" v-for="action in dashboardActions" :key="action.id">
            <h3>{{ action.label }}</h3>
            <p>{{ action.description }}</p>
            <button class="btn primary" type="button" @click="selectSection(action.id)">Apri</button>
          </article>
        </section>

        <section v-if="section === 'events'" class="ev-section">

          <!-- ══ CREATE PANEL ══════════════════════════════════════════ -->
          <div class="ev-create-panel">
            <div class="ev-create-panel__hd" @click="createFormOpen = !createFormOpen">
              <div class="ev-create-panel__title">
                <span class="ev-create-panel__icon">＋</span>
                <span>Crea nuovo evento</span>
              </div>
              <div class="ev-create-panel__meta">
                <span v-if="!hasEnoughTeams" class="ev-badge-warn">Aggiungi squadre prima</span>
                <span class="ev-chevron" :class="{ open: createFormOpen }">›</span>
              </div>
            </div>

            <transition name="ev-slide">
              <div v-if="createFormOpen" class="ev-create-panel__body">
                <p v-if="!hasEnoughTeams" class="ev-alert-info">
                  Aggiungi almeno due squadre dalla sezione "Squadre" per abilitare la creazione di un evento.
                </p>
                <form id="event-create-form" @submit.prevent="createEvent" class="ev-cform">

                  <!-- Teams + match info -->
                  <div class="ev-cform__teams">
                    <div class="ev-field">
                      <label class="ev-label">Squadra di casa</label>
                      <input v-model="teamInputs.home" type="text" list="admin-team-options" :disabled="!hasEnoughTeams" placeholder="Digita o seleziona…" required class="ev-input" @change="handleTeamInput('home')" @blur="handleTeamInput('home')" />
                    </div>
                    <div class="ev-vs">VS</div>
                    <div class="ev-field">
                      <label class="ev-label">Squadra ospite</label>
                      <input v-model="teamInputs.away" type="text" list="admin-team-options" :disabled="!hasEnoughTeams" placeholder="Digita o seleziona…" required class="ev-input" @change="handleTeamInput('away')" @blur="handleTeamInput('away')" />
                    </div>
                    <datalist id="admin-team-options">
                      <option v-for="team in teams" :key="team.id" :value="teamOptionValue(team)" />
                    </datalist>
                  </div>

                  <div class="ev-cform__row2">
                    <div class="ev-field">
                      <label class="ev-label">Data e ora</label>
                      <input v-model="newEvent.start_datetime" type="datetime-local" :disabled="!hasEnoughTeams" required class="ev-input" />
                    </div>
                    <div class="ev-field">
                      <label class="ev-label">Location</label>
                      <input v-model.trim="newEvent.location" type="text" placeholder="Es. Palazzetto dello Sport" :disabled="!hasEnoughTeams" class="ev-input" />
                    </div>
                  </div>

                  <!-- Pre-vote options -->
                  <div class="ev-opts-block">
                    <div class="ev-opts-block__hd">
                      <span class="ev-opts-dot ev-opts-dot--cyan"></span>
                      <span class="ev-opts-label">Esperienze pre-voto</span>
                    </div>
                    <div class="ev-opts-grid">
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_pre_vote_sponsors" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Sponsor bordo campo</span>
                      </label>
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_pre_vote_bottom_sponsors" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Sponsor fondo campo</span>
                      </label>
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_vote_counter" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Contatore voti live</span>
                      </label>
                    </div>
                  </div>

                  <!-- Post-vote options -->
                  <div class="ev-opts-block">
                    <div class="ev-opts-block__hd">
                      <span class="ev-opts-dot ev-opts-dot--purple"></span>
                      <span class="ev-opts-label">Esperienze post-voto</span>
                    </div>
                    <div class="ev-opts-grid">
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_vote_trend" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Andamento voti</span>
                      </label>
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_selfie" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Selfie MVP</span>
                      </label>
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_reaction_test" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Mini-gioco riflessi</span>
                      </label>
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_feedback_survey" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Sondaggio feedback</span>
                      </label>
                    </div>
                  </div>

                  <!-- Branded Mini-Game -->
                  <div class="ev-opts-block">
                    <div class="ev-opts-block__hd">
                      <span class="ev-opts-dot ev-opts-dot--orange"></span>
                      <span class="ev-opts-label">Branded Mini-Game</span>
                    </div>
                    <div class="ev-opts-grid">
                      <label class="ev-toggle">
                        <input type="checkbox" v-model="newEvent.show_branded_game" :disabled="!hasEnoughTeams" />
                        <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                        <span class="ev-toggle__text">Attiva mini-game sponsor</span>
                      </label>
                    </div>
                    <div v-if="newEvent.show_branded_game" class="bg-config-panel">
                      <div class="bg-config-grid">
                        <div class="bg-config-fields">
                          <div class="bg-field-row">
                            <label class="bg-label">ID Sponsor <span class="bg-req">*</span></label>
                            <input class="ev-input" type="text" v-model.trim="newEvent.brandedGameConfigDraft.sponsor_id" placeholder="es. acme-sport" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Nome Sponsor <span class="bg-req">*</span></label>
                            <input class="ev-input" type="text" v-model.trim="newEvent.brandedGameConfigDraft.sponsor_name" placeholder="es. ACME Sport" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">URL Logo Sponsor</label>
                            <input class="ev-input" type="url" v-model.trim="newEvent.brandedGameConfigDraft.sponsor_logo_url" placeholder="https://…/logo.png" />
                          </div>
                          <div class="bg-field-row bg-field-row--colors">
                            <div>
                              <label class="bg-label">Colore primario</label>
                              <div class="bg-color-row">
                                <input type="color" v-model="newEvent.brandedGameConfigDraft.primary_color" class="bg-color-swatch" />
                                <input class="ev-input bg-color-hex" type="text" v-model="newEvent.brandedGameConfigDraft.primary_color" maxlength="7" />
                              </div>
                            </div>
                            <div>
                              <label class="bg-label">Colore secondario</label>
                              <div class="bg-color-row">
                                <input type="color" v-model="newEvent.brandedGameConfigDraft.secondary_color" class="bg-color-swatch" />
                                <input class="ev-input bg-color-hex" type="text" v-model="newEvent.brandedGameConfigDraft.secondary_color" maxlength="7" />
                              </div>
                            </div>
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Tipo gioco <span class="bg-req">*</span></label>
                            <select class="ev-input" v-model="newEvent.brandedGameConfigDraft.game_type">
                              <option value="tap_challenge">Tap Battle ⚡</option>
                              <option value="memory_flash">Memory Flash 🃏</option>
                              <option value="sponsor_rush">Sponsor Rush 🏃</option>
                            </select>
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">CTA Label <span class="bg-opt">(opzionale)</span></label>
                            <input class="ev-input" type="text" v-model.trim="newEvent.brandedGameConfigDraft.cta_label" placeholder="es. Scopri di più" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">CTA URL <span class="bg-opt">(opzionale)</span></label>
                            <input class="ev-input" type="url" v-model.trim="newEvent.brandedGameConfigDraft.cta_url" placeholder="https://…" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Tipo reward</label>
                            <div class="bg-reward-row">
                              <label class="bg-radio">
                                <input type="radio" v-model="newEvent.brandedGameConfigDraft.reward_type" value="coins" /> Coin reward
                              </label>
                              <label class="bg-radio">
                                <input type="radio" v-model="newEvent.brandedGameConfigDraft.reward_type" value="none" /> Nessun premio
                              </label>
                              <label class="bg-radio bg-radio--disabled" title="Coming soon">
                                <input type="radio" disabled value="coupon" /> Coupon <span class="bg-coming-soon">coming soon</span>
                              </label>
                            </div>
                          </div>
                          <div class="bg-field-row" v-if="newEvent.brandedGameConfigDraft.reward_type === 'coins'">
                            <label class="bg-label">Coin reward</label>
                            <input class="ev-input bg-input-sm" type="number" min="0" v-model.number="newEvent.brandedGameConfigDraft.reward_coins" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Partite max per utente</label>
                            <input class="ev-input bg-input-sm" type="number" min="1" v-model.number="newEvent.brandedGameConfigDraft.max_plays_per_user" />
                          </div>
                          <p v-if="validateBrandedGameConfigClient(newEvent.brandedGameConfigDraft)" class="bg-error">
                            {{ validateBrandedGameConfigClient(newEvent.brandedGameConfigDraft) }}
                          </p>
                        </div>

                        <!-- Preview -->
                        <div class="bg-preview">
                          <p class="bg-preview__label">Anteprima entry</p>
                          <div class="bg-preview-entry"
                            :style="{ background: newEvent.brandedGameConfigDraft.primary_color, color: newEvent.brandedGameConfigDraft.secondary_color }">
                            <img v-if="newEvent.brandedGameConfigDraft.sponsor_logo_url"
                              :src="newEvent.brandedGameConfigDraft.sponsor_logo_url"
                              class="bg-preview-logo" alt="logo sponsor" />
                            <span class="bg-preview-icon" v-else>🏆</span>
                            <span class="bg-preview-text">
                              Gioca con {{ newEvent.brandedGameConfigDraft.sponsor_name || 'Sponsor' }}
                            </span>
                          </div>
                          <p class="bg-preview__label" style="margin-top:12px">Tipo gioco</p>
                          <div class="bg-preview-badge">
                            <span v-if="newEvent.brandedGameConfigDraft.game_type === 'tap_challenge'">Tap Battle ⚡</span>
                            <span v-else-if="newEvent.brandedGameConfigDraft.game_type === 'memory_flash'">Memory Flash 🃏</span>
                            <span v-else-if="newEvent.brandedGameConfigDraft.game_type === 'sponsor_rush'">Sponsor Rush 🏃</span>
                          </div>
                          <p class="bg-preview__label" style="margin-top:12px">Reward</p>
                          <div class="bg-preview-badge" v-if="newEvent.brandedGameConfigDraft.reward_type === 'coins'">
                            🪙 {{ newEvent.brandedGameConfigDraft.reward_coins }} coin per partita
                          </div>
                          <div class="bg-preview-badge" v-else>Nessun premio</div>
                          <p class="bg-preview__label" style="margin-top:12px">Limite</p>
                          <div class="bg-preview-badge">{{ newEvent.brandedGameConfigDraft.max_plays_per_user }} partita/e per utente</div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- Prizes -->
                  <div class="ev-opts-block">
                    <div class="ev-opts-block__hd">
                      <span class="ev-opts-dot ev-opts-dot--gold"></span>
                      <span class="ev-opts-label">Premi in palio</span>
                      <span class="ev-opts-count">{{ newEventPrizes.length }} {{ newEventPrizes.length === 1 ? 'premio' : 'premi' }}</span>
                    </div>
                    <div class="ev-prizes-list">
                      <div v-for="(prize, index) in newEventPrizes" :key="`new-ev-prize-${index}`" class="ev-prize-row">
                        <span class="ev-prize-num">#{{ index + 1 }}</span>
                        <input v-model.trim="prize.name" type="text" :placeholder="`Premio ${index + 1}`" :disabled="!hasEnoughTeams" class="ev-input" />
                        <input v-model.trim="prize.winSmsText" type="text" placeholder="SMS vittoria" :disabled="!hasEnoughTeams" class="ev-input" />
                        <button class="ev-btn-icon ev-btn-icon--danger" type="button" @click="removeNewEventPrize(index)" :disabled="newEventPrizes.length <= 1" title="Rimuovi">✕</button>
                      </div>
                    </div>
                    <button class="ev-btn-ghost" type="button" @click="addNewEventPrize" :disabled="!hasEnoughTeams">＋ Aggiungi premio</button>
                  </div>

                  <!-- Footer actions -->
                  <div class="ev-cform__footer">
                    <div v-if="lastCreatedEventLink" class="ev-success-banner">
                      <span>✓ Evento creato!</span>
                      <a :href="lastCreatedEventLink" target="_blank" rel="noopener">{{ lastCreatedEventLink }}</a>
                      <button class="ev-btn-ghost ev-btn-ghost--sm" type="button" @click="copyLink(lastCreatedEventLink)">Copia link</button>
                    </div>
                    <div class="ev-cform__actions">
                      <button class="ev-btn-deactivate" type="button" @click="deactivateEvents" :disabled="!activeEventId || isDisablingEvents">
                        {{ isDisablingEvents ? 'Disattivazione…' : 'Disattiva tutti' }}
                      </button>
                      <button class="ev-btn-create" type="submit" :disabled="!hasEnoughTeams">
                        Crea evento <span>→</span>
                      </button>
                    </div>
                  </div>

                </form>
              </div>
            </transition>
          </div>

          <!-- ══ EVENTS LIST ══════════════════════════════════════════ -->
          <div class="ev-list">
            <p v-if="!visibleEvents.length" class="ev-empty">
              Nessun evento. Creane uno qui sopra.
            </p>

            <div v-for="event in visibleEvents" :key="event.id" class="ev-card"
              :class="{ 'ev-card--active': event.is_active, 'ev-card--closed': event.is_active && event.votes_closed }">

              <!-- Card header -->
              <div class="ev-card__hd">
                <div class="ev-card__identity">
                  <h3 class="ev-card__title">{{ eventLabel(event) }}</h3>
                  <div class="ev-card__meta">
                    <span>{{ formatEventDate(event.start_datetime) }}</span>
                    <span v-if="event.location" class="ev-sep">·</span>
                    <span v-if="event.location">{{ event.location }}</span>
                  </div>
                </div>
                <div class="ev-card__badges">
                  <span v-if="event.is_active && !event.votes_closed" class="ev-badge ev-badge--live">
                    <span class="ev-badge__pulse"></span>Live
                  </span>
                  <span v-else-if="event.is_active && event.votes_closed" class="ev-badge ev-badge--closed">Chiuse</span>
                  <span v-else class="ev-badge ev-badge--idle">Inattivo</span>
                </div>
              </div>

              <!-- Quick actions bar -->
              <div class="ev-card__actions">
                <button class="ev-act ev-act--success" type="button" :disabled="event.is_active || updatingEventId === event.id" @click="activateEvent(event.id)">
                  <span v-if="event.is_active">✓ Attivo</span>
                  <span v-else-if="updatingEventId === event.id">Attivazione…</span>
                  <span v-else>Attiva</span>
                </button>
                <button class="ev-act ev-act--secondary" type="button" @click="openVote(event.id)">↗ Pagina voto</button>
                <button class="ev-act ev-act--warning" type="button" :disabled="concludingEventId === event.id" @click="concludeEvent(event.id)">
                  <span v-if="concludingEventId === event.id">Conclusione…</span>
                  <span v-else>Concludi</span>
                </button>
                <button class="ev-act ev-act--danger" type="button" @click="deleteEvent(event.id)">Elimina</button>
                <div class="ev-card__link">
                  <code class="ev-link-code">{{ buildEventLink(event.id) }}</code>
                  <button class="ev-btn-icon" type="button" @click="copyLink(buildEventLink(event.id))" title="Copia link">⎘</button>
                </div>
              </div>

              <!-- ── Tab bar ── -->
              <div class="ev-tabs">
                <div class="ev-tabs__bar">
                  <button v-for="tab in [{id:'settings',icon:'⚙',label:'Impostazioni'},{id:'prizes',icon:'🎁',label:'Premi'},{id:'feedback',icon:'📋',label:'Sondaggio'},{id:'quiz',icon:'🧩',label:'Quiz'},{id:'stories',icon:'▶',label:'Stories'}]"
                    :key="tab.id" type="button" class="ev-tabs__btn" :class="{ active: getEventTab(event.id) === tab.id }"
                    @click="setEventTab(event.id, tab.id)">
                    <span>{{ tab.icon }}</span>{{ tab.label }}
                  </button>
                </div>

                <div class="ev-tabs__panel">

                  <!-- ── TAB: Impostazioni ── -->
                  <div v-if="getEventTab(event.id) === 'settings'" class="ev-tab-settings">
                    <div class="ev-toggles-grid">
                      <div class="ev-toggles-col">
                        <div class="ev-toggles-col__label">Pre-voto</div>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_pre_vote_sponsors" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Sponsor bordo campo</span>
                        </label>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_pre_vote_bottom_sponsors" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Sponsor fondo campo</span>
                        </label>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_vote_counter" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Contatore voti live</span>
                        </label>
                      </div>
                      <div class="ev-toggles-col">
                        <div class="ev-toggles-col__label">Post-voto</div>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_vote_trend" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Andamento voti</span>
                        </label>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_selfie" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Selfie MVP</span>
                        </label>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_reaction_test" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Mini-gioco riflessi</span>
                        </label>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_feedback_survey" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Sondaggio feedback</span>
                        </label>
                        <label class="ev-toggle">
                          <input type="checkbox" v-model="event.show_branded_game" :disabled="isSavingPrizesFor(event.id)" />
                          <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                          <span class="ev-toggle__text">Branded Mini-Game</span>
                        </label>
                      </div>
                    </div>

                    <!-- Branded Game inline config (edit) -->
                    <div v-if="event.show_branded_game" class="bg-config-panel bg-config-panel--edit">
                      <div class="bg-config-grid">
                        <div class="bg-config-fields">
                          <div class="bg-field-row">
                            <label class="bg-label">ID Sponsor <span class="bg-req">*</span></label>
                            <input class="ev-input" type="text" v-model.trim="event.brandedGameConfigDraft.sponsor_id" :disabled="isSavingPrizesFor(event.id)" placeholder="es. acme-sport" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Nome Sponsor <span class="bg-req">*</span></label>
                            <input class="ev-input" type="text" v-model.trim="event.brandedGameConfigDraft.sponsor_name" :disabled="isSavingPrizesFor(event.id)" placeholder="es. ACME Sport" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">URL Logo Sponsor</label>
                            <input class="ev-input" type="url" v-model.trim="event.brandedGameConfigDraft.sponsor_logo_url" :disabled="isSavingPrizesFor(event.id)" placeholder="https://…/logo.png" />
                          </div>
                          <div class="bg-field-row bg-field-row--colors">
                            <div>
                              <label class="bg-label">Colore primario</label>
                              <div class="bg-color-row">
                                <input type="color" v-model="event.brandedGameConfigDraft.primary_color" :disabled="isSavingPrizesFor(event.id)" class="bg-color-swatch" />
                                <input class="ev-input bg-color-hex" type="text" v-model="event.brandedGameConfigDraft.primary_color" maxlength="7" :disabled="isSavingPrizesFor(event.id)" />
                              </div>
                            </div>
                            <div>
                              <label class="bg-label">Colore secondario</label>
                              <div class="bg-color-row">
                                <input type="color" v-model="event.brandedGameConfigDraft.secondary_color" :disabled="isSavingPrizesFor(event.id)" class="bg-color-swatch" />
                                <input class="ev-input bg-color-hex" type="text" v-model="event.brandedGameConfigDraft.secondary_color" maxlength="7" :disabled="isSavingPrizesFor(event.id)" />
                              </div>
                            </div>
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Tipo gioco <span class="bg-req">*</span></label>
                            <select class="ev-input" v-model="event.brandedGameConfigDraft.game_type" :disabled="isSavingPrizesFor(event.id)">
                              <option value="tap_challenge">Tap Battle ⚡</option>
                              <option value="memory_flash">Memory Flash 🃏</option>
                              <option value="sponsor_rush">Sponsor Rush 🏃</option>
                            </select>
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">CTA Label <span class="bg-opt">(opzionale)</span></label>
                            <input class="ev-input" type="text" v-model.trim="event.brandedGameConfigDraft.cta_label" :disabled="isSavingPrizesFor(event.id)" placeholder="es. Scopri di più" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">CTA URL <span class="bg-opt">(opzionale)</span></label>
                            <input class="ev-input" type="url" v-model.trim="event.brandedGameConfigDraft.cta_url" :disabled="isSavingPrizesFor(event.id)" placeholder="https://…" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Tipo reward</label>
                            <div class="bg-reward-row">
                              <label class="bg-radio">
                                <input type="radio" v-model="event.brandedGameConfigDraft.reward_type" value="coins" :disabled="isSavingPrizesFor(event.id)" /> Coin reward
                              </label>
                              <label class="bg-radio">
                                <input type="radio" v-model="event.brandedGameConfigDraft.reward_type" value="none" :disabled="isSavingPrizesFor(event.id)" /> Nessun premio
                              </label>
                              <label class="bg-radio bg-radio--disabled" title="Coming soon">
                                <input type="radio" disabled value="coupon" /> Coupon <span class="bg-coming-soon">coming soon</span>
                              </label>
                            </div>
                          </div>
                          <div class="bg-field-row" v-if="event.brandedGameConfigDraft.reward_type === 'coins'">
                            <label class="bg-label">Coin reward</label>
                            <input class="ev-input bg-input-sm" type="number" min="0" v-model.number="event.brandedGameConfigDraft.reward_coins" :disabled="isSavingPrizesFor(event.id)" />
                          </div>
                          <div class="bg-field-row">
                            <label class="bg-label">Partite max per utente</label>
                            <input class="ev-input bg-input-sm" type="number" min="1" v-model.number="event.brandedGameConfigDraft.max_plays_per_user" :disabled="isSavingPrizesFor(event.id)" />
                          </div>
                          <p v-if="validateBrandedGameConfigClient(event.brandedGameConfigDraft)" class="bg-error">
                            {{ validateBrandedGameConfigClient(event.brandedGameConfigDraft) }}
                          </p>
                        </div>
                        <!-- Preview -->
                        <div class="bg-preview">
                          <p class="bg-preview__label">Anteprima entry</p>
                          <div class="bg-preview-entry"
                            :style="{ background: event.brandedGameConfigDraft.primary_color, color: event.brandedGameConfigDraft.secondary_color }">
                            <img v-if="event.brandedGameConfigDraft.sponsor_logo_url"
                              :src="event.brandedGameConfigDraft.sponsor_logo_url"
                              class="bg-preview-logo" alt="logo sponsor" />
                            <span class="bg-preview-icon" v-else>🏆</span>
                            <span class="bg-preview-text">
                              Gioca con {{ event.brandedGameConfigDraft.sponsor_name || 'Sponsor' }}
                            </span>
                          </div>
                          <p class="bg-preview__label" style="margin-top:12px">Tipo gioco</p>
                          <div class="bg-preview-badge">
                            <span v-if="event.brandedGameConfigDraft.game_type === 'tap_challenge'">Tap Battle ⚡</span>
                            <span v-else-if="event.brandedGameConfigDraft.game_type === 'memory_flash'">Memory Flash 🃏</span>
                            <span v-else-if="event.brandedGameConfigDraft.game_type === 'sponsor_rush'">Sponsor Rush 🏃</span>
                          </div>
                          <p class="bg-preview__label" style="margin-top:12px">Reward</p>
                          <div class="bg-preview-badge" v-if="event.brandedGameConfigDraft.reward_type === 'coins'">
                            🪙 {{ event.brandedGameConfigDraft.reward_coins }} coin per partita
                          </div>
                          <div class="bg-preview-badge" v-else>Nessun premio</div>
                          <p class="bg-preview__label" style="margin-top:12px">Limite</p>
                          <div class="bg-preview-badge">{{ event.brandedGameConfigDraft.max_plays_per_user }} partita/e per utente</div>
                        </div>
                      </div>
                    </div>

                    <div class="ev-tab-save-row">
                      <button class="ev-btn-save" type="button" @click="saveEventPrizes(event)" :disabled="isSavingPrizesFor(event.id)">
                        {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva impostazioni' }}
                      </button>
                    </div>
                  </div>

                  <!-- ── TAB: Premi ── -->
                  <div v-else-if="getEventTab(event.id) === 'prizes'" class="ev-tab-prizes">
                    <div class="ev-prizes-list">
                      <div v-for="(prize, i) in eventPrizeDrafts[event.id]" :key="`ev-${event.id}-prize-${i}`" class="ev-prize-row">
                        <span class="ev-prize-num">#{{ i + 1 }}</span>
                        <input v-model="prize.name" type="text" :placeholder="`Premio ${i + 1}`" class="ev-input" :disabled="isSavingPrizesFor(event.id)" />
                        <input v-model="prize.winSmsText" type="text" placeholder="SMS vincitore" class="ev-input" :disabled="isSavingPrizesFor(event.id)" />
                        <span v-if="prize.winner" class="ev-prize-winner">🏆 {{ prize.winner.ticketCode }}</span>
                        <button class="ev-btn-icon ev-btn-icon--danger" type="button"
                          :disabled="!!prize.winner || eventPrizeDrafts[event.id].length <= 1 || isSavingPrizesFor(event.id)"
                          @click="removeEventPrize(event.id, i)" title="Rimuovi">✕</button>
                      </div>
                    </div>
                    <div class="ev-tab-save-row">
                      <button class="ev-btn-ghost" type="button" @click="addEventPrize(event.id)" :disabled="isSavingPrizesFor(event.id)">＋ Aggiungi premio</button>
                      <button class="ev-btn-save" type="button" @click="saveEventPrizes(event)" :disabled="isSavingPrizesFor(event.id)">
                        {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva premi' }}
                      </button>
                    </div>
                    <p v-if="eventPrizeErrors[event.id]" class="ev-tab-error">{{ eventPrizeErrors[event.id] }}</p>
                  </div>

                  <!-- ── TAB: Sondaggio ── -->
                  <div v-else-if="getEventTab(event.id) === 'feedback'" class="ev-tab-feedback">
                    <div class="ev-feedback-questions">
                      <div v-for="question in feedbackDraftFor(event.id).questions" :key="`ev-${event.id}-fbq-${question.id}`" class="ev-feedback-q">
                        <div class="ev-feedback-q__hd">
                          <span class="ev-feedback-q__id">{{ question.id }}</span>
                          <input v-model="question.title" type="text" :placeholder="`Domanda: ${question.id}`" class="ev-input" :disabled="isSavingPrizesFor(event.id)" />
                        </div>
                        <div class="ev-feedback-answers">
                          <label v-for="answer in question.answers" :key="`ev-${event.id}-fba-${question.id}-${answer.value}`" class="ev-feedback-ans">
                            <span v-if="answer.icon" class="ev-feedback-ans__icon">{{ answer.icon }}</span>
                            <code class="ev-feedback-ans__code">{{ answer.value }}</code>
                            <input v-model="answer.label" type="text" :placeholder="`Risposta: ${answer.value}`" class="ev-input" :disabled="isSavingPrizesFor(event.id)" />
                          </label>
                        </div>
                      </div>
                    </div>
                    <div class="ev-field" style="margin-top:0.85rem">
                      <label class="ev-label">Domanda suggerimenti (opzionale)</label>
                      <textarea v-model="feedbackDraftFor(event.id).suggestionPrompt" rows="2" maxlength="120" class="ev-input" :disabled="isSavingPrizesFor(event.id)" placeholder="Testo domanda aperta…"></textarea>
                    </div>
                    <div class="ev-tab-save-row">
                      <button class="ev-btn-save" type="button" @click="saveEventFeedbackSurvey(event)" :disabled="isSavingPrizesFor(event.id)">
                        {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva sondaggio' }}
                      </button>
                    </div>
                    <p v-if="eventFeedbackErrors[event.id]" class="ev-tab-error">{{ eventFeedbackErrors[event.id] }}</p>
                  </div>

                  <!-- ── TAB: Quiz ── -->
                  <div v-else-if="getEventTab(event.id) === 'quiz'" class="ev-tab-quiz">
                    <div class="ev-quiz-statusbar">
                      <span class="ev-quiz-statusbar__label">Stato quiz:</span>
                      <button class="ev-act" :class="quizDraftFor(event.id).enabled ? 'ev-act--success' : 'ev-act--secondary'" type="button" @click="quizDraftFor(event.id).enabled = true">Abilitato</button>
                      <button class="ev-act" :class="!quizDraftFor(event.id).enabled ? 'ev-act--danger' : 'ev-act--secondary'" type="button" @click="quizDraftFor(event.id).enabled = false">Disabilitato</button>
                      <span style="flex:1"></span>
                      <button class="ev-btn-ghost ev-btn-ghost--sm" type="button" @click="loadQuizForEvent(event.id)">↻ Ricarica</button>
                      <button class="ev-btn-save" type="button" @click="saveQuizConfig(event.id)">Salva config</button>
                    </div>
                    <div class="ev-quiz-questions">
                      <div v-for="(q, qi) in quizQuestionsFor(event.id)" :key="`quiz-${event.id}-${q.id || qi}`" class="ev-quiz-q">
                        <div class="ev-quiz-q__hd">
                          <span class="ev-quiz-q__num">Q{{ qi + 1 }}</span>
                          <input v-model="q.question_text" type="text" placeholder="Testo domanda" class="ev-input ev-quiz-q__text" />
                          <label class="ev-quiz-meta">
                            <span>Corretta</span>
                            <input type="number" min="0" :max="q.answers.length - 1" v-model.number="q.correct_index" class="ev-input ev-input--sm" />
                          </label>
                          <label class="ev-quiz-meta">
                            <span>Ordine</span>
                            <input type="number" min="0" v-model.number="q.order_index" class="ev-input ev-input--sm" />
                          </label>
                        </div>
                        <div class="ev-quiz-answers">
                          <div v-for="(_, ai) in q.answers" :key="ai" class="ev-quiz-ans">
                            <span class="ev-quiz-ans__badge" :class="{ correct: q.correct_index === ai }">{{ ai }}</span>
                            <input v-model="q.answers[ai]" type="text" :placeholder="`Risposta ${ai + 1}`" class="ev-input" />
                          </div>
                        </div>
                        <div class="ev-quiz-q__actions">
                          <button class="ev-btn-ghost ev-btn-ghost--sm" type="button" @click="addAnswerToQuestion(event.id, qi)" :disabled="q.answers.length >= 4">＋ Risposta</button>
                          <button class="ev-btn-ghost ev-btn-ghost--sm" type="button" @click="removeAnswerFromQuestion(event.id, qi)" :disabled="q.answers.length <= 2">－ Risposta</button>
                          <button class="ev-btn-save ev-btn-save--sm" type="button" @click="saveQuizQuestion(event.id, q)">Salva</button>
                          <button class="ev-act ev-act--danger" type="button" @click="deleteQuizQuestion(event.id, q)">Elimina</button>
                        </div>
                      </div>
                      <button class="ev-btn-ghost" type="button" @click="addQuizQuestionDraft(event.id)">＋ Aggiungi domanda</button>
                    </div>
                  </div>

                  <!-- ── TAB: Stories ── -->
                  <div v-else-if="getEventTab(event.id) === 'stories'" class="ev-tab-stories">
                    <div class="ev-stories-toolbar">
                      <span v-if="isStoriesLoading(event.id)" class="ev-loading">
                        <span class="ev-spinner"></span> Caricamento…
                      </span>
                      <button class="ev-btn-ghost ev-btn-ghost--sm" type="button" @click="loadStoriesForEvent(event.id)" :disabled="isStoriesLoading(event.id)">↻ Ricarica</button>
                      <button class="ev-btn-ghost" type="button" @click="addStoryDraft(event.id)">＋ Aggiungi story</button>
                    </div>
                    <div class="ev-stories-list">
                      <div v-for="(story, si) in storiesForEvent(event.id)" :key="`story-${event.id}-${story.id || si}`" class="ev-story-row">
                        <div class="ev-story-thumb">
                          <img v-if="story.thumbnail_url" :src="story.thumbnail_url" alt="thumb" />
                          <span v-else>▶</span>
                        </div>
                        <div class="ev-story-fields">
                          <div class="ev-story-grid">
                            <input v-model="story.player_name" type="text" placeholder="Nome giocatore (opz.)" class="ev-input" />
                            <input v-model="story.title" type="text" placeholder="Titolo (opz.)" class="ev-input" />
                            <input v-model="story.thumbnail_url" type="url" placeholder="Thumbnail URL" class="ev-input" />
                            <div class="ev-story-video">
                              <input v-model="story.video_url" type="url" placeholder="Video URL" class="ev-input" />
                              <input :id="`story-video-${event.id}-${si}`" type="file" accept="video/mp4,video/webm,video/quicktime" style="display:none" @change="uploadStoryVideo(event.id, story, si, $event)" />
                              <button class="ev-btn-ghost ev-btn-ghost--sm" type="button" @click="triggerStoryVideoPicker(event.id, si)" :disabled="isStoryVideoUploading(event.id, si) || isStoriesSaving(event.id)">
                                {{ isStoryVideoUploading(event.id, si) ? 'Upload…' : '⬆ Carica' }}
                              </button>
                            </div>
                          </div>
                          <div class="ev-story-footer">
                            <label class="ev-toggle ev-toggle--inline">
                              <input type="checkbox" v-model="story.is_active" />
                              <span class="ev-toggle__track"><span class="ev-toggle__thumb"></span></span>
                              <span class="ev-toggle__text">Attiva</span>
                            </label>
                            <div class="ev-story-order">
                              <button class="ev-btn-icon" type="button" @click="moveStory(event.id, si, -1)" :disabled="si === 0">↑</button>
                              <button class="ev-btn-icon" type="button" @click="moveStory(event.id, si, 1)" :disabled="si === storiesForEvent(event.id).length - 1">↓</button>
                            </div>
                            <button class="ev-btn-save ev-btn-save--sm" type="button" @click="saveStory(event.id, story, si)" :disabled="isStoriesSaving(event.id) || isStoryVideoUploading(event.id, si)">Salva</button>
                            <button class="ev-act ev-act--danger" type="button" @click="deleteStory(event.id, story, si)" :disabled="isStoriesSaving(event.id)">Elimina</button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                </div>
              </div>

            </div>
          </div>

        </section>

        <section v-else-if="section === 'closing'" class="card closing-card">
          <header class="section-header">
            <h2>Chiusura votazioni</h2>
            <p>
              Gestisci lo stato delle votazioni per la partita attualmente
              attiva.
            </p>
          </header>

          <div v-if="activeEventEntry" class="active-event-summary">
            <div class="summary-header">
              <h3>{{ activeEventLabel }}</h3>
              <span
                :class="[
                  'badge',
                  activeEventVotesClosed ? 'badge-closed' : 'badge-open',
                ]"
              >
                {{
                  activeEventVotesClosed
                    ? "Votazioni chiuse"
                    : "Votazioni aperte"
                }}
              </span>
            </div>
            <p class="muted">
              {{ activeEventDateLabel }} • {{ activeEventLocation }}
            </p>

            <div class="actions-row">
              <button
                class="btn warning"
                type="button"
                @click="closeActiveEventVoting"
                :disabled="isClosingVotes || activeEventVotesClosed"
              >
                {{ isClosingVotes ? "Chiusura…" : "Chiudi votazioni" }}
              </button>
              <button
                class="btn success"
                type="button"
                @click="activateEvent(activeEventEntry.id)"
                :disabled="
                  !activeEventEntry ||
                  updatingEventId === activeEventEntry.id ||
                  !activeEventVotesClosed
                "
              >
                <span v-if="updatingEventId === activeEventEntry.id"
                  >Riattivazione…</span
                >
                <span v-else>Attiva</span>
              </button>
              <button
                class="btn outline"
                type="button"
                @click="deactivateEvents"
                :disabled="isDisablingEvents"
              >
                {{ isDisablingEvents ? "Disattivazione…" : "Disattiva" }}
              </button>
            </div>
          </div>
          <div v-else class="info-banner">
            Nessun evento attivo al momento. Attiva una partita dalla sezione
            "Eventi" per gestire le votazioni.
          </div>

          <p v-if="closeVotesMessage" class="success-message">
            {{ closeVotesMessage }}
          </p>
        </section>

        <section v-else-if="section === 'results'" class="card results-card">
          <header class="section-header">
            <h2>Risultati votazioni</h2>
            <p>
              Seleziona un evento per vedere la classifica MVP aggiornata in
              tempo reale.
            </p>
          </header>

          <div class="results-controls">
            <label>
              Evento
              <select
                v-model.number="selectedResultsEventId"
                :disabled="!availableEvents.length"
              >
                <option disabled value="0">Seleziona un evento</option>
                <option
                  v-for="event in availableEvents"
                  :key="event.id"
                  :value="event.id"
                >
                  {{ eventLabel(event) }}
                </option>
              </select>
            </label>
            <button
              class="btn secondary"
              type="button"
              @click="fetchEventResults({ showLoader: true })"
              :disabled="isLoadingResults || !selectedResultsEventId"
            >
              {{ isLoadingResults ? "Aggiornamento…" : "Aggiorna ora" }}
            </button>
          </div>

          <div v-if="selectedResultsEvent" class="results-summary">
            <h3>{{ selectedResultsEventLabel }}</h3>
            <p class="muted">
              {{ selectedResultsEventDate || "Data da definire" }}
            </p>
          </div>

          <p v-if="resultsError" class="error">{{ resultsError }}</p>
          <div v-else-if="!availableEvents.length" class="info-banner">
            Crea un evento per visualizzare i risultati delle votazioni MVP.
          </div>
          <div v-else class="results-leaderboard">
            <div class="results-meta">
              <span><strong>Voti totali:</strong> {{ totalVotes }}</span>
              <span v-if="lastResultsUpdateLabel"
                ><strong>Ultimo aggiornamento:</strong>
                {{ lastResultsUpdateLabel }}</span
              >
              <span class="auto-refresh"
                >Aggiornamento automatico ogni 5 secondi</span
              >
            </div>
            <p v-if="isLoadingResults" class="muted">Caricamento risultati…</p>
            <p v-else-if="!hasResultsVotes" class="muted">
              Non ci sono ancora voti per questo evento.
            </p>
            <ul class="leaderboard-list" aria-live="polite">
              <li
                v-for="(entry, index) in resultsLeaderboard"
                :key="entry.id"
                class="leaderboard-item"
              >
                <div class="rank">#{{ index + 1 }}</div>
                <div class="player-name">
                  <span class="lastname">{{ entry.lastNameUpper }}</span>
                  <span class="firstname">{{ entry.firstName }}</span>
                </div>
                <div class="votes">
                  <strong>{{ entry.votes }}</strong>
                  <span class="muted">{{
                    entry.votes === 1 ? "voto" : "voti"
                  }}</span>
                </div>
                <div class="progress" role="presentation">
                  <div
                    class="progress-bar"
                    :style="{ width: `${entry.percentage}%` }"
                  ></div>
                </div>
              </li>
            </ul>
            <div
              v-if="selectedResultsEventId && !isStaff"
              class="sponsor-analytics"
            >
              <h3>Analisi sponsor</h3>
              <p v-if="sponsorAnalyticsError" class="error">
                {{ sponsorAnalyticsError }}
              </p>
              <p v-else-if="isLoadingSponsorAnalytics" class="muted">
                Caricamento dati sponsor…
              </p>
              <div v-else-if="!hasSponsorAnalyticsData" class="muted">
                Nessun dato sponsor disponibile al momento.
              </div>
              <div v-else class="sponsor-analytics__content">
                <div
                  v-if="sponsorAnalyticsDisplay"
                  class="sponsor-analytics__grid"
                >
                  <div class="sponsor-analytics__card">
                    <span class="sponsor-analytics__label">Utenti totali</span>
                    <strong class="sponsor-analytics__value">{{
                      sponsorAnalyticsDisplay.totalUsersLabel
                    }}</strong>
                  </div>
                  <div class="sponsor-analytics__card">
                    <span class="sponsor-analytics__label">Sezione vista</span>
                    <strong class="sponsor-analytics__value">{{
                      sponsorAnalyticsDisplay.seenRateLabel
                    }}</strong>
                    <span class="sponsor-analytics__hint"
                      >{{ sponsorAnalyticsDisplay.seenUsersLabel }} utenti</span
                    >
                  </div>
                  <div class="sponsor-analytics__card">
                    <span class="sponsor-analytics__label"
                      >Tempo medio visione</span
                    >
                    <strong class="sponsor-analytics__value">{{
                      sponsorAnalyticsDisplay.averageWatchTimeLabel
                    }}</strong>
                    <span class="sponsor-analytics__hint"
                      >{{
                        sponsorAnalyticsDisplay.watchedUsersLabel
                      }}
                      utenti</span
                    >
                  </div>
                  <div class="sponsor-analytics__card">
                    <span class="sponsor-analytics__label">Click totali</span>
                    <strong class="sponsor-analytics__value">{{
                      sponsorAnalyticsDisplay.totalClicksLabel
                    }}</strong>
                    <span class="sponsor-analytics__hint"
                      >{{ sponsorAnalyticsDisplay.clickRateLabel }} •
                      {{
                        sponsorAnalyticsDisplay.uniqueClickersLabel
                      }}
                      utenti</span
                    >
                  </div>
                  <div
                    class="sponsor-analytics__card sponsor-analytics__card--wide"
                  >
                    <span class="sponsor-analytics__label"
                      >Sponsor più visualizzato</span
                    >
                    <strong class="sponsor-analytics__value">{{
                      sponsorAnalyticsDisplay.topSponsorName
                    }}</strong>
                    <span class="sponsor-analytics__hint"
                      >{{
                        sponsorAnalyticsDisplay.topSponsorViewsLabel
                      }}
                      visualizzazioni</span
                    >
                  </div>
                </div>
                <div
                  v-if="sponsorChartRows.length"
                  class="sponsor-analytics__chart"
                >
                  <h4>Andamento visualizzazioni e click</h4>
                  <ul class="sponsor-chart">
                    <li
                      v-for="point in sponsorChartRows"
                      :key="point.timestamp || point.label"
                      class="sponsor-chart__row"
                    >
                      <div class="sponsor-chart__label">{{ point.label }}</div>
                      <div class="sponsor-chart__bars" aria-hidden="true">
                        <div
                          class="sponsor-chart__bar sponsor-chart__bar--seen"
                          :style="{ width: `${point.seenPercent}%` }"
                        ></div>
                        <div
                          class="sponsor-chart__bar sponsor-chart__bar--clicks"
                          :style="{ width: `${point.clicksPercent}%` }"
                        ></div>
                      </div>
                      <div class="sponsor-chart__values">
                        <span
                          >{{ point.seen.toLocaleString("it-IT") }} viste</span
                        >
                        <span
                          >{{
                            point.clicks.toLocaleString("it-IT")
                          }}
                          click</span
                        >
                      </div>
                    </li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-else-if="section === 'selfies'" class="card">
          <header class="section-header">
            <h2>Selfie MVP</h2>
            <p>
              Gestisci i selfie inviati dai tifosi per l'evento selezionato.
            </p>
          </header>

          <div class="form-grid">
            <label>
              Evento
              <select v-model.number="selectedSelfieEventId">
                <option v-if="!availableEvents.length" value="0" disabled>
                  Nessun evento disponibile
                </option>
                <option
                  v-for="event in availableEvents"
                  :key="event.id"
                  :value="event.id"
                >
                  {{ eventLabel(event) }} •
                  {{ formatEventDate(event.start_datetime) }}
                </option>
              </select>
            </label>
          </div>

          <p v-if="selfieModerationMessage" class="success">
            {{ selfieModerationMessage }}
          </p>
          <p v-if="selfieLoadError" class="error">{{ selfieLoadError }}</p>

          <div
            v-if="isLoadingSelfies"
            class="selfie-admin-loader"
            role="status"
            aria-live="polite"
          >
            <span class="spinner" aria-hidden="true"></span>
            <p>Caricamento selfie…</p>
          </div>
          <p v-else-if="!availableEvents.length" class="muted">
            Crea un evento per raccogliere selfie dal pubblico.
          </p>
          <p v-else-if="!eventSelfies.length" class="muted">
            Nessun selfie ricevuto per questo evento al momento.
          </p>
          <div v-else class="selfie-admin-grid">
            <article
              v-for="selfie in eventSelfies"
              :key="selfie.id"
              class="selfie-admin-card"
            >
              <a
                v-if="selfie.image_src"
                :href="selfie.image_src"
                target="_blank"
                rel="noopener noreferrer"
                class="selfie-admin-thumb"
              >
                <img :src="selfie.image_src" :alt="`Selfie ${selfie.id}`" />
              </a>
              <div v-else class="selfie-admin-thumb selfie-admin-thumb--empty">
                <span>Immagine non disponibile</span>
              </div>
              <div class="selfie-admin-body">
                <h3 class="selfie-admin-caption">
                  {{ selfie.caption || "Senza didascalia" }}
                </h3>
                <p class="selfie-admin-meta">
                  Inviato: {{ formatSelfieDate(selfie.submitted_at) || "N/D" }}
                </p>
                <p class="selfie-admin-meta">
                  Device: {{ selfie.device_token || "Non disponibile" }}
                </p>
                <p class="selfie-admin-meta">
                  Dimensione:
                  {{ formatSelfieFileSize(selfie.file_size_bytes) || "N/D" }}
                </p>
                <p class="selfie-admin-meta">
                  Consenso uso immagine:
                  <strong>{{ selfie.accepted_image_terms ? "Accettato" : "Non indicato" }}</strong>
                </p>
                <p class="selfie-admin-status">
                  Stato: <strong>{{ selfieStatusLabel(selfie) }}</strong>
                </p>
                <div class="selfie-admin-actions">
                  <button
                    class="btn danger"
                    type="button"
                    :disabled="isSelfieBusy(selfie.id)"
                    @click="deleteSelfie(selfie)"
                  >
                    Elimina foto
                  </button>
                </div>
              </div>
            </article>
          </div>
        </section>

        <section v-else-if="section === 'history'" class="card history-card">
          <header class="section-header">
            <h2>Storico eventi</h2>
            <p>
              Consulta i dati degli eventi passati con riepilogo voti, MVP e
              interazioni sponsor.
            </p>
          </header>

          <div class="history-toolbar">
            <button
              class="btn secondary"
              type="button"
              @click="refreshEventHistory"
              :disabled="isLoadingEventHistory"
            >
              {{ isLoadingEventHistory ? "Aggiornamento…" : "Aggiorna" }}
            </button>
          </div>

          <p v-if="eventHistorySuccess" class="success-message">
            {{ eventHistorySuccess }}
          </p>
          <p v-if="eventHistoryError" class="error">{{ eventHistoryError }}</p>
          <p v-else-if="isLoadingEventHistory" class="muted text-center">
            Caricamento storico in corso…
          </p>
          <p v-else-if="!eventHistory.length" class="muted text-center">
            Non sono presenti eventi conclusi al momento.
          </p>

          <ul v-else class="history-list">
            <li
              v-for="entry in eventHistory"
              :key="entry.id"
              class="history-item"
            >
              <div class="history-item__header">
                <div>
                  <h3>{{ entry.title }}</h3>
                  <p class="muted">
                    {{ formatHistoryDate(entry.startDatetime) }}
                    <span v-if="entry.location">• {{ entry.location }}</span>
                  </p>
                </div>
                <div class="history-item__meta">
                  <div class="history-item__totals">
                    <span class="history-item__total">
                      <strong>{{ entry.totalVotesLabel }}</strong> voti totali
                    </span>
                    <span class="history-item__total">
                      <strong>{{ entry.totalVisitorsLabel }}</strong> visitatori
                      totali
                    </span>
                    <span class="history-item__unique-visitors">
                      <strong>{{ entry.uniqueVisitorsLabel }}</strong>
                      visitatori unici
                    </span>
                    <span class="history-item__sponsor-total">
                      <strong>{{ entry.sponsorClicksTotalLabel }}</strong> click
                      sponsor
                    </span>
                  </div>
                  <div class="history-item__actions">
                    <button
                      class="btn secondary"
                      type="button"
                      :disabled="isGeneratingHistoryAiReport(entry.id)"
                      @click="generateEventAiReport(entry)"
                    >
                      {{
                        isGeneratingHistoryAiReport(entry.id)
                          ? "Analisi AI…"
                          : entry.aiReport
                            ? "Rigenera report AI"
                            : "Genera report AI"
                      }}
                    </button>
                    <button
                      class="btn outline"
                      type="button"
                      :disabled="isDownloadingHistoryReport(entry.id)"
                      @click="downloadEventHistoryReport(entry)"
                    >
                      {{
                        isDownloadingHistoryReport(entry.id)
                          ? "Generazione report…"
                          : "Scarica report"
                      }}
                    </button>
                    <button
                      v-if="isSuperAdmin"
                      class="btn danger"
                      type="button"
                      @click="openPurgeDialog(entry)"
                    >
                      Elimina evento
                    </button>
                  </div>
                </div>
              </div>

              <div class="history-details">
                <div
                  v-if="entry.aiReport"
                  class="history-details__column history-details__column--ai"
                >
                  <h4>Report AI post-partita</h4>
                  <p class="history-ai-summary">
                    {{ entry.aiReport.executiveSummary }}
                  </p>
                  <p v-if="entry.aiReport.fullReport" class="history-ai-report">
                    {{ entry.aiReport.fullReport }}
                  </p>
                  <div
                    v-if="
                      entry.aiReport.strengths.length ||
                      entry.aiReport.criticalities.length ||
                      entry.aiReport.insights.length ||
                      entry.aiReport.suggestions.length
                    "
                    class="history-ai-grid"
                  >
                    <div v-if="entry.aiReport.strengths.length" class="history-ai-card">
                      <h5>Punti forti</h5>
                      <ul><li v-for="(item, index) in entry.aiReport.strengths" :key="`strength-${entry.id}-${index}`">{{ item }}</li></ul>
                    </div>
                    <div v-if="entry.aiReport.criticalities.length" class="history-ai-card">
                      <h5>Criticità</h5>
                      <ul><li v-for="(item, index) in entry.aiReport.criticalities" :key="`critical-${entry.id}-${index}`">{{ item }}</li></ul>
                    </div>
                    <div v-if="entry.aiReport.insights.length" class="history-ai-card">
                      <h5>Insight automatici</h5>
                      <ul><li v-for="(item, index) in entry.aiReport.insights" :key="`insight-${entry.id}-${index}`">{{ item }}</li></ul>
                    </div>
                    <div v-if="entry.aiReport.suggestions.length" class="history-ai-card">
                      <h5>Suggerimenti operativi</h5>
                      <ul><li v-for="(item, index) in entry.aiReport.suggestions" :key="`suggestion-${entry.id}-${index}`">{{ item }}</li></ul>
                    </div>
                  </div>
                </div>
                <div class="history-details__column">
                  <h4>MVP</h4>
                  <p v-if="entry.mvp">
                    {{ entry.mvp.name }} •
                    {{ entry.mvp.votes.toLocaleString("it-IT") }} voti
                  </p>
                  <p v-else class="muted">Nessun MVP assegnato.</p>
                </div>
                <div class="history-details__column">
                  <h4>Interazioni sponsor</h4>
                  <div
                    v-if="entry.sponsorAnalyticsHasData"
                    class="history-sponsor-summary"
                  >
                    <div class="history-sponsor-summary__grid">
                      <div class="history-sponsor-summary__card">
                        <span class="history-sponsor-summary__label"
                          >Utenti totali</span
                        >
                        <strong class="history-sponsor-summary__value">
                          {{ entry.sponsorAnalyticsDisplay.totalUsersLabel }}
                        </strong>
                      </div>
                      <div class="history-sponsor-summary__card">
                        <span class="history-sponsor-summary__label"
                          >Sezione vista</span
                        >
                        <strong class="history-sponsor-summary__value">
                          {{ entry.sponsorAnalyticsDisplay.seenRateLabel }}
                        </strong>
                        <span class="history-sponsor-summary__hint">
                          {{
                            entry.sponsorAnalyticsDisplay.seenUsersLabel
                          }}
                          utenti
                        </span>
                      </div>
                      <div class="history-sponsor-summary__card">
                        <span class="history-sponsor-summary__label"
                          >Tempo medio visione</span
                        >
                        <strong class="history-sponsor-summary__value">
                          {{
                            entry.sponsorAnalyticsDisplay.averageWatchTimeLabel
                          }}
                        </strong>
                        <span class="history-sponsor-summary__hint">
                          {{
                            entry.sponsorAnalyticsDisplay.watchedUsersLabel
                          }}
                          utenti •
                          {{
                            entry.sponsorAnalyticsDisplay.totalWatchTimeLabel
                          }}
                          totali
                        </span>
                      </div>
                      <div class="history-sponsor-summary__card">
                        <span class="history-sponsor-summary__label"
                          >Click totali</span
                        >
                        <strong class="history-sponsor-summary__value">
                          {{ entry.sponsorAnalyticsDisplay.totalClicksLabel }}
                        </strong>
                        <span class="history-sponsor-summary__hint">
                          {{ entry.sponsorAnalyticsDisplay.clickRateLabel }} •
                          {{
                            entry.sponsorAnalyticsDisplay.uniqueClickersLabel
                          }}
                          utenti
                        </span>
                      </div>
                      <div
                        class="history-sponsor-summary__card history-sponsor-summary__card--wide"
                      >
                        <span class="history-sponsor-summary__label"
                          >Sponsor più visualizzato</span
                        >
                        <strong class="history-sponsor-summary__value">
                          {{ entry.sponsorAnalyticsDisplay.topSponsorName }}
                        </strong>
                        <span class="history-sponsor-summary__hint">
                          {{
                            entry.sponsorAnalyticsDisplay.topSponsorViewsLabel
                          }}
                          visualizzazioni
                        </span>
                      </div>
                    </div>
                  </div>
                  <ul
                    v-if="entry.sponsorClicks.length"
                    class="history-sponsor-list"
                  >
                    <li
                      v-for="sponsor in entry.sponsorClicks"
                      :key="`${entry.id}-sponsor-${sponsor.id}`"
                    >
                      <span class="history-sponsor-name">{{
                        sponsor.reportName || sponsor.name
                      }}</span>
                      <span class="history-sponsor-clicks"
                        >{{
                          sponsor.clicks.toLocaleString("it-IT")
                        }}
                        click</span
                      >
                    </li>
                  </ul>
                  <p v-else class="muted">Nessun click registrato.</p>
                  <div
                    v-if="entry.sponsorAnalyticsTimeline.length"
                    class="history-sponsor-timeline"
                  >
                    <h5>Andamento interazioni</h5>
                    <ul class="history-sponsor-timeline__list">
                      <li
                        v-for="point in entry.sponsorAnalyticsTimeline"
                        :key="`${entry.id}-analytics-${point.timestamp || point.label}`"
                        class="history-sponsor-timeline__item"
                      >
                        <span class="history-sponsor-timeline__time">{{
                          point.label
                        }}</span>
                        <div class="history-sponsor-timeline__values">
                          <span
                            class="history-sponsor-timeline__value history-sponsor-timeline__value--seen"
                          >
                            {{ point.seen.toLocaleString("it-IT") }} viste
                          </span>
                          <span
                            class="history-sponsor-timeline__value history-sponsor-timeline__value--watched"
                          >
                            {{ point.watched.toLocaleString("it-IT") }} guardate
                          </span>
                          <span
                            class="history-sponsor-timeline__value history-sponsor-timeline__value--clicks"
                          >
                            {{ point.clicks.toLocaleString("it-IT") }} click
                          </span>
                        </div>
                      </li>
                    </ul>
                  </div>
                </div>
                <div
                  class="history-details__column history-details__column--feedback"
                >
                  <h4>Sondaggio feedback</h4>
                  <div
                    v-if="entry.feedbackSummary"
                    class="history-feedback-summary"
                  >
                    <p class="history-feedback-summary__total">
                      {{ entry.feedbackSummary.totalResponsesLabel }}
                    </p>
                    <div
                      v-for="question in entry.feedbackSummary.questions"
                      :key="`${entry.id}-feedback-${question.id}`"
                      class="history-feedback-summary__question"
                    >
                      <h5>{{ question.title }}</h5>
                      <ul class="history-feedback-summary__answers">
                        <li
                          v-for="answer in question.answers"
                          :key="`${entry.id}-feedback-${question.id}-${answer.value}`"
                          class="history-feedback-summary__answer"
                        >
                          <div class="history-feedback-summary__answer-header">
                            <span
                              class="history-feedback-summary__answer-label"
                              >{{ answer.label }}</span
                            >
                            <span
                              class="history-feedback-summary__answer-count"
                            >
                              {{ answer.countLabel }}
                              <span v-if="entry.feedbackSummary.hasResponses"
                                >({{ answer.percentLabel }})</span
                              >
                            </span>
                          </div>
                          <div
                            class="history-feedback-summary__answer-bar"
                            role="presentation"
                          >
                            <span
                              class="history-feedback-summary__answer-bar-fill"
                              :style="{ width: answer.barWidth }"
                              aria-hidden="true"
                            ></span>
                          </div>
                        </li>
                      </ul>
                      <p v-if="!question.hasAnswers" class="muted small">
                        Nessuna risposta registrata.
                      </p>
                    </div>
                    <div class="history-feedback-summary__question">
                      <h5>
                        {{ entry.feedbackSummary.suggestionQuestion.title }}
                      </h5>
                      <ul
                        v-if="
                          entry.feedbackSummary.suggestionQuestion
                            .hasSuggestions
                        "
                        class="history-feedback-summary__suggestions"
                      >
                        <li
                          v-for="(suggestion, suggestionIndex) in entry
                            .feedbackSummary.suggestionQuestion.suggestions"
                          :key="`${entry.id}-feedback-suggestion-${suggestionIndex}`"
                        >
                          {{ suggestion }}
                        </li>
                      </ul>
                      <p v-else class="muted small">
                        Nessun suggerimento inviato.
                      </p>
                    </div>
                  </div>
                  <p v-else class="muted">Nessun feedback raccolto.</p>
                </div>
                <div
                  class="history-details__column history-details__column--engagement"
                >
                  <h4>Tracking post voto</h4>
                  <div v-if="entry.engagement" class="history-engagement">
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Tempo totale</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.totalLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Tempo medio</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.averageLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Utenti tracciati</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.usersLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Aperture andamento voti</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.voteTrendOpensLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Aperture Selfie MVP</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.selfieOpensLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Selfie MVP chiuso senza invio</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.selfieAbandonsLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Aperture Reaction test</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.reactionOpensLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Reaction test chiuso senza giocare</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.reactionAbandonsLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Aperture "Migliora la tua esperienza"</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.experienceOpensLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Feedback avviato ma non inviato</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.experienceAbandonsLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Aperture modifica foto</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.photoEditOpensLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Aperture modifica voto</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.voteEditOpensLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Modifica voto chiusa senza aggiornare</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.voteEditAbandonsLabel }}</strong
                      >
                    </div>
                    <div class="history-engagement__row">
                      <span class="history-engagement__label"
                        >Voti modificati dopo l'apertura</span
                      >
                      <strong class="history-engagement__value"
                        >{{ entry.engagement.voteEditCompletionsLabel }}</strong
                      >
                    </div>
                    <p
                      v-if="!entry.engagement.hasData"
                      class="muted small history-engagement__empty"
                    >
                      Nessun dato di permanenza registrato.
                    </p>
                  </div>
                  <p v-else class="muted">Tracking non disponibile.</p>
                </div>
                <div class="history-details__column history-details__column--tracking-events">
                  <h4>Tracking events</h4>
                  <p v-if="!entry.trackingEvents.length" class="muted">
                    Nessun tracking event registrato.
                  </p>
                  <ul v-else class="history-tracking-events">
                    <li
                      v-for="track in entry.trackingEvents"
                      :key="`${entry.id}-tracking-${track.name}`"
                      class="history-tracking-events__item"
                    >
                      <div class="history-tracking-events__head">
                        <strong>{{ track.nameLabel }}</strong>
                        <span>{{ track.countLabel }}</span>
                      </div>
                      <p class="history-tracking-events__meta">
                        Sessioni: {{ track.uniqueSessionsLabel }} · Device: {{ track.uniqueDevicesLabel }} · Fan: {{ track.uniqueFansLabel }}
                      </p>
                      <p
                        v-if="track.rangeLabel"
                        class="history-tracking-events__meta"
                      >
                        Ultima attività: {{ track.rangeLabel }}
                      </p>
                      <div
                        v-if="track.details.length"
                        class="history-tracking-events__chips"
                      >
                        <span
                          v-for="detail in track.details"
                          :key="`${entry.id}-tracking-${track.name}-${detail.label}`"
                          class="history-tracking-events__chip"
                        >
                          <strong>{{ detail.label }}:</strong> {{ detail.value }}
                        </span>
                      </div>
                      <ul
                        v-if="track.metadataSamples.length"
                        class="history-tracking-events__samples"
                      >
                        <li
                          v-for="(sample, sampleIndex) in track.metadataSamples"
                          :key="`${entry.id}-tracking-${track.name}-sample-${sampleIndex}`"
                        >
                          <code>{{ sample }}</code>
                        </li>
                      </ul>
                    </li>
                  </ul>
                </div>
                <div class="history-details__column history-details__column--tracking-analytics">
                  <h4>Analytics evento</h4>
                  <div class="history-analytics-kpi">
                    <div class="history-analytics-kpi__item">
                      <span>Sessioni uniche</span>
                      <strong>{{ entry.trackingAnalytics.labels.uniqueSessions }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Eventi totali</span>
                      <strong>{{ entry.trackingAnalytics.labels.totalEvents }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Voti inviati</span>
                      <strong>{{ entry.trackingAnalytics.labels.votesSubmitted }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Conv. voto</span>
                      <strong>{{ entry.trackingAnalytics.labels.voteConversionRate }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Feedback inviati</span>
                      <strong>{{ entry.trackingAnalytics.labels.feedbackSubmitted }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Click sponsor</span>
                      <strong>{{ entry.trackingAnalytics.labels.sponsorClicks }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Ordini bar</span>
                      <strong>{{ entry.trackingAnalytics.labels.barOrdersCompleted }}</strong>
                    </div>
                    <div class="history-analytics-kpi__item">
                      <span>Eventi/sessione</span>
                      <strong>{{ entry.trackingAnalytics.labels.avgEventsPerSession }}</strong>
                    </div>
                  </div>
                  <p class="history-analytics-meta">
                    First/Last event: {{ formatHistoryDate(entry.trackingAnalytics.firstEventAt) || "N/D" }} → {{ formatHistoryDate(entry.trackingAnalytics.lastEventAt) || "N/D" }}
                  </p>
                  <p class="history-analytics-meta">
                    Peak minuto: {{ formatHistoryDate(entry.trackingAnalytics.peakActivityMinute) || "N/D" }}
                  </p>
                  <div v-if="entry.trackingAnalytics.funnels.length" class="history-analytics-funnels">
                    <h5>Funnel principali</h5>
                    <div
                      v-for="funnel in entry.trackingAnalytics.funnels"
                      :key="`${entry.id}-funnel-${funnel.name}`"
                      class="history-analytics-funnels__item"
                    >
                      <strong>{{ funnel.name.toUpperCase() }}</strong>
                      <ul>
                        <li
                          v-for="step in funnel.steps"
                          :key="`${entry.id}-${funnel.name}-${step.key}`"
                        >
                          <span>{{ step.label }}</span>
                          <span>{{ step.countLabel }}</span>
                        </li>
                      </ul>
                    </div>
                  </div>
                  <div class="history-analytics-segment">
                    <h5>Guest vs Registrati</h5>
                    <p>
                      Voti: G {{ entry.trackingAnalytics.segmentBreakdown.guest.submittedVotes.toLocaleString("it-IT") }}
                      · R {{ entry.trackingAnalytics.segmentBreakdown.registered.submittedVotes.toLocaleString("it-IT") }}
                    </p>
                    <p>
                      Feedback: G {{ entry.trackingAnalytics.segmentBreakdown.guest.feedbackSubmitted.toLocaleString("it-IT") }}
                      · R {{ entry.trackingAnalytics.segmentBreakdown.registered.feedbackSubmitted.toLocaleString("it-IT") }}
                    </p>
                    <p>
                      Sponsor click: G {{ entry.trackingAnalytics.segmentBreakdown.guest.sponsorClicks.toLocaleString("it-IT") }}
                      · R {{ entry.trackingAnalytics.segmentBreakdown.registered.sponsorClicks.toLocaleString("it-IT") }}
                    </p>
                  </div>
                  <div v-if="entry.trackingAnalytics.topEventNames.length" class="history-analytics-top">
                    <h5>Top event_name</h5>
                    <ul>
                      <li
                        v-for="top in entry.trackingAnalytics.topEventNames.slice(0, 5)"
                        :key="`${entry.id}-top-${top.name}`"
                      >
                        <span>{{ top.name }}</span>
                        <span>{{ top.count.toLocaleString("it-IT") }}</span>
                      </li>
                    </ul>
                  </div>
                </div>
                <div
                  class="history-details__column history-details__column--prizes"
                >
                  <h4>Estrazione premi</h4>
                  <p
                    class="history-prize-status"
                    :class="
                      entry.hasPrizeDraw
                        ? 'history-prize-status--success'
                        : 'history-prize-status--pending'
                    "
                  >
                    {{
                      entry.hasPrizeDraw
                        ? "Estrazione eseguita"
                        : "Estrazione non eseguita"
                    }}
                  </p>
                  <p v-if="!entry.prizes.length" class="muted">
                    Nessun premio configurato per l'evento.
                  </p>
                  <ul v-else class="history-prize-list">
                    <li
                      v-for="prize in entry.prizes"
                      :key="`${entry.id}-prize-${prize.id}`"
                      class="history-prize-item"
                    >
                      <span class="history-prize-name">{{ prize.name }}</span>
                      <span v-if="prize.hasWinner" class="history-prize-code">
                        Codice vincente:
                        <strong>{{ prize.winnerTicketCode }}</strong>
                      </span>
                      <span v-else class="history-prize-code muted"
                        >Nessun codice vincente assegnato.</span
                      >
                    </li>
                  </ul>
                </div>
              </div>

              <div class="history-votes" v-if="entry.timeline.length">
                <div class="history-votes__header">
                  <h4>Votazioni</h4>
                  <p v-if="entry.timelineRange" class="history-votes__range">
                    Dal {{ entry.timelineRange.start }} al
                    {{ entry.timelineRange.end }}
                  </p>
                </div>
                <VoteTrendChart
                  v-if="entry.timelineChart.points.length"
                  class="history-votes__chart"
                  :points="entry.timelineChart.points"
                  :start-label="entry.timelineChart.startLabel"
                  :end-label="entry.timelineChart.endLabel"
                  accessible-label="Andamento dei voti ogni 15 minuti"
                />
                <div
                  class="history-votes__actions"
                  v-if="entry.timeline.length"
                >
                  <button
                    class="btn link"
                    type="button"
                    @click="toggleHistoryTimeline(entry)"
                    :aria-expanded="entry.isTimelineExpanded ? 'true' : 'false'"
                    :aria-controls="`history-votes-list-${entry.id}`"
                  >
                    {{
                      entry.isTimelineExpanded
                        ? "Nascondi dettagli"
                        : "Visualizza altro"
                    }}
                  </button>
                </div>
                <ul
                  v-if="entry.isTimelineExpanded"
                  class="history-votes-list"
                  :id="`history-votes-list-${entry.id}`"
                >
                  <li
                    v-for="bucket in entry.timeline"
                    :key="`${entry.id}-bucket-${bucket.start || bucket.rangeLabel}`"
                    class="history-votes-list__item"
                  >
                    <span class="history-votes-list__range">{{
                      bucket.rangeLabel
                    }}</span>
                    <span class="history-votes-list__votes">{{
                      bucket.votesLabel
                    }}</span>
                  </li>
                </ul>
              </div>
            </li>
          </ul>
        </section>

        <AdminTeamsSection
          v-else-if="section === 'teams'"
          :auth-headers="authHeaders"
          :is-super-admin="isSuperAdmin"
          @updated="teams = $event"
        />

        <AdminPlayersSection
          v-else-if="section === 'players'"
          :auth-headers="authHeaders"
          :is-super-admin="isSuperAdmin"
          :teams="teams"
        />

        <AdminSponsorsSection
          v-else-if="section === 'sponsors'"
          :auth-headers="authHeaders"
          :is-super-admin="isSuperAdmin"
        />

        <AdminCouponsSection
          v-else-if="section === 'coupons'"
          :auth-headers="authHeaders"
          :is-super-admin="isSuperAdmin"
          :events="events"
          :partners="partners"
        />

        <AdminBarSection v-else-if="section === 'bar'" :auth-headers="authHeaders" :is-super-admin="isSuperAdmin" />

        <AdminPartnersSection
          v-else-if="section === 'partners'"
          :auth-headers="authHeaders"
          :is-super-admin="isSuperAdmin"
          @updated="partners = $event"
        />

        <AdminAdminsSection
          v-else-if="section === 'admins'"
          :auth-headers="authHeaders"
          :is-super-admin="isSuperAdmin"
        />
        <section v-else-if="section === 'marketing'" class="card">
          <MarketingSection />
        </section>
        <div
          v-if="purgeDialog.visible"
          class="modal-backdrop"
          role="dialog"
          aria-modal="true"
          :aria-label="
            purgeDialog.event
              ? `Conferma eliminazione per ${purgeDialog.event.title}`
              : 'Conferma eliminazione evento'
          "
        >
          <div class="modal-card">
            <h3>Elimina evento</h3>
            <p>
              Questa operazione è permanente e rimuoverà tutti i dati collegati
              all'evento.
            </p>
            <p class="muted">Conferma inserendo la password del super admin.</p>
            <p v-if="purgeDialog.error" class="error">
              {{ purgeDialog.error }}
            </p>
            <label>
              Password super admin
              <input
                v-model="purgeDialog.password"
                type="password"
                autocomplete="current-password"
                required
              />
            </label>
            <div class="modal-actions">
              <button
                class="btn outline"
                type="button"
                @click="closePurgeDialog"
                :disabled="purgeDialog.isSubmitting"
              >
                Annulla
              </button>
              <button
                class="btn danger"
                type="button"
                @click="confirmPurge"
                :disabled="purgeDialog.isSubmitting || !purgeDialog.password"
              >
                {{
                  purgeDialog.isSubmitting
                    ? "Eliminazione…"
                    : "Elimina definitivamente"
                }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
  </div>
</template>

<script setup>
import {
  computed,
  defineProps,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import { apiClient, resolveApiUrl, getOrganizationSlug } from "../api";
import { DEFAULT_ROSTER_SCHEMA, MAX_PLAYER_SLOTS } from "../roster";
import VoteTrendChart from "./VoteTrendChart.vue";
import AdminLayout from "./admin/AdminLayout.vue";
import AdminSidebar from "./admin/AdminSidebar.vue";
import AdminHeader from "./admin/AdminHeader.vue";
import AdminBarSection from "./AdminBarSection.vue";
import SectionHeader from "./admin/ui/SectionHeader.vue";
import BaseSearchInput from "./admin/ui/BaseSearchInput.vue";
import MarketingSection from "./admin/marketing/MarketingSection.vue";
import AdminTeamsSection from "./admin/AdminTeamsSection.vue";
import AdminAdminsSection from "./admin/AdminAdminsSection.vue";
import AdminPartnersSection from "./admin/AdminPartnersSection.vue";
import AdminSponsorsSection from "./admin/AdminSponsorsSection.vue";
import AdminPlayersSection from "./admin/AdminPlayersSection.vue";
import AdminCouponsSection from "./admin/AdminCouponsSection.vue";

const props = defineProps({
  organizationSlug: {
    type: String,
    default: "",
  },
});

const basePath = computed(() => {
  if (props.organizationSlug) {
    const normalized = props.organizationSlug.startsWith("/")
      ? props.organizationSlug
      : `/${props.organizationSlug}`;
    return normalized.replace(/\/+$/, "");
  }
  return (import.meta.env.BASE_URL ?? "/").replace(/\/+$/, "");
});

const resolvedOrganizationSlug = computed(() =>
  (props.organizationSlug || "").replace(/^\/+|\/+$/g, ""),
);

const baseVoteUrl = computed(() => new URL(basePath.value || "/", window.location.origin));
const historyDateFormatter = new Intl.DateTimeFormat("it-IT", {
  dateStyle: "full",
  timeStyle: "short",
});
const historyTimeFormatter = new Intl.DateTimeFormat("it-IT", {
  hour: "2-digit",
  minute: "2-digit",
});
const analyticsTimeFormatter = new Intl.DateTimeFormat("it-IT", {
  dateStyle: "short",
  timeStyle: "short",
});
const selfieDateFormatter = new Intl.DateTimeFormat("it-IT", {
  dateStyle: "medium",
  timeStyle: "short",
});

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

function normalizeFeedbackSurveyInput(raw) {
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

function assignSurveyDraft(target, source) {
  const normalized = normalizeFeedbackSurveyInput(source);
  if (!Array.isArray(target.questions)) {
    target.questions = [];
  }
  target.questions.splice(
    0,
    target.questions.length,
    ...normalized.questions.map((question) => ({
      id: question.id,
      title: question.title,
      answers: question.answers.map((answer) => ({ ...answer })),
    })),
  );
  target.suggestionPrompt = normalized.suggestionPrompt;
}

function toApiSurveyPayload(survey) {
  if (!survey || typeof survey !== "object") {
    return {
      questions: DEFAULT_FEEDBACK_SURVEY.questions.map((question) => ({
        id: question.id,
        title: question.title,
        answers: question.answers.map((answer) => ({
          value: answer.value,
          label: answer.label,
          icon: answer.icon || "",
        })),
      })),
      suggestion_prompt: DEFAULT_FEEDBACK_SURVEY.suggestionPrompt,
    };
  }

  const normalized = normalizeFeedbackSurveyInput(survey);
  return {
    questions: normalized.questions.map((question) => ({
      id: question.id,
      title: question.title,
      answers: question.answers.map((answer) => ({
        value: answer.value,
        label: answer.label,
        icon: answer.icon || "",
      })),
    })),
    suggestion_prompt: normalized.suggestionPrompt,
  };
}

let resultsSseSource = null;

const isSidebarOpen = ref(false);
const section = ref("dashboard");
const tabs = [
  { id: "dashboard", label: "Dashboard" },
  { id: "events", label: "Eventi" },
  { id: "closing", label: "Chiusura votazioni" },
  { id: "results", label: "Risultati" },
  { id: "selfies", label: "Selfie MVP" },
  { id: "history", label: "Storico eventi" },
  { id: "teams", label: "Squadre" },
  { id: "players", label: "Giocatori" },
  { id: "sponsors", label: "Sponsor" },
  { id: "coupons", label: "Coupon" },
  { id: "bar", label: "BAR" },
  { id: "partners", label: "Partners" },
  { id: "admins", label: "Admin" },
  { id: "marketing", label: "Marketing" },
];
const STAFF_TAB_IDS = new Set(["dashboard", "closing", "results"]);
const BAR_TAB_IDS = new Set(["bar"]);

const teams = ref([]);
const players = ref([]);
const events = ref([]);
const admins = ref([]);
const partners = ref([]);
const sponsors = ref([]);
const coupons = ref([]);
const eventHistory = ref([]);
const couponEventSearch = ref("");
const eventSelfies = ref([]);
const isLoadingEventHistory = ref(false);
const eventHistoryError = ref("");
const eventHistorySuccess = ref("");
const hasLoadedEventHistory = ref(false);
const isLoadingSelfies = ref(false);
const selfieLoadError = ref("");
const selfieModerationMessage = ref("");
const selectedSelfieEventId = ref(0);
const selfieBusyState = reactive({});
const historyReportDownloadState = reactive({});
const historyAiGenerateState = reactive({});
const purgeDialog = reactive({
  visible: false,
  event: null,
  password: "",
  error: "",
  isSubmitting: false,
});
const updatingEventId = ref(0);
const concludingEventId = ref(0);
const isDisablingEvents = ref(false);
const selectedResultsEventId = ref(0);
const eventResults = ref([]);
const isLoadingResults = ref(false);
const resultsError = ref("");
const lastResultsUpdate = ref(null);
const sponsorAnalytics = ref(null);
const sponsorAnalyticsError = ref("");
const isLoadingSponsorAnalytics = ref(false);
const newTeamName = ref("");
const playerSlotCount = MAX_PLAYER_SLOTS;
const rosterSchema = ref(DEFAULT_ROSTER_SCHEMA);
const validateRosterSchema = (value) =>
  value === 12 || value === 13 || value === 14 ? value : DEFAULT_ROSTER_SCHEMA;

const PLAYER_IMAGE_MAX_WIDTH = 600;
const PLAYER_IMAGE_MAX_HEIGHT = 600;
const PLAYER_IMAGE_QUALITY = 0.75;

const createEmptyPlayerSlot = (teamId = 0) => ({
  id: 0,
  first_name: "",
  last_name: "",
  role: "",
  jersey_number: "",
  team_id: teamId,
  is_called_up: true,
  image_url: "",
  image_preview: "",
  _imageChangeToken: null,
});

const playerSlots = reactive(
  Array.from({ length: playerSlotCount }, () => createEmptyPlayerSlot()),
);
const playerOverflow = ref([]);
const isSavingPlayers = ref(false);
const playerSaveError = ref("");
const playerSaveMessage = ref("");
function defaultBrandedGameConfig() {
  return {
    sponsor_id: "",
    sponsor_name: "",
    sponsor_logo_url: "",
    primary_color: "#1a73e8",
    secondary_color: "#ffffff",
    game_type: "tap_challenge",
    cta_label: "",
    cta_url: "",
    reward_type: "coins",
    reward_coins: 50,
    max_plays_per_user: 1,
  };
}

function validateBrandedGameConfigClient(cfg) {
  if (!cfg.sponsor_id?.trim()) return "ID sponsor obbligatorio.";
  if (!cfg.sponsor_name?.trim()) return "Nome sponsor obbligatorio.";
  if (!cfg.game_type) return "Seleziona il tipo di gioco.";
  if (cfg.reward_type === "coins" && (cfg.reward_coins ?? 0) < 0)
    return "I coin reward devono essere ≥ 0.";
  if ((cfg.max_plays_per_user ?? 1) < 1)
    return "Partite massime per utente devono essere ≥ 1.";
  return null;
}

function createDefaultNewEventState() {
  return {
    team1_id: 0,
    team2_id: 0,
    start_datetime: "",
    location: "",
    show_reaction_test: true,
    show_selfie: true,
    show_vote_trend: true,
    show_feedback_survey: true,
    show_pre_vote_sponsors: true,
    show_pre_vote_bottom_sponsors: true,
    show_vote_counter: true,
    show_branded_game: false,
    brandedGameConfigDraft: defaultBrandedGameConfig(),
  };
}

function createEmptyCouponDraft() {
  return {
    title: "",
    shortDesc: "",
    sponsorId: partners.value?.[0]?.id ?? 0,
    merchantId: partners.value?.[0]?.id ?? 0,
    matchIds: [],
    startDateInput: "",
    endDateInput: "",
    maxUses: 0,
    status: "draft",
    imageUrl: "",
    imagePreview: "",
  };
}

const newEvent = reactive(createDefaultNewEventState());
const newEventSurvey = reactive(normalizeFeedbackSurveyInput());
const newEventPrizes = ref([{ name: "", winSmsText: "" }]);
const teamInputs = reactive({
  home: "",
  away: "",
});
const newAdmin = reactive({
  username: "",
  password: "",
  role: "",
});
const newPartner = reactive({
  name: "",
  username: "",
  password: "",
});
const maxSponsors = 4;
const couponStatusOptions = ["draft", "active", "paused", "archived"];
const couponStatusLabels = {
  draft: "Bozza",
  active: "Attivo",
  paused: "In pausa",
  archived: "Archiviato",
};
const getCouponStatusLabel = (status) => couponStatusLabels[status] || status;
const newSponsor = reactive({
  name: "",
  reportName: "",
  linkUrl: "",
  logoData: "",
  isActive: true,
});
const newCoupon = reactive(createEmptyCouponDraft());
const desiredActiveSponsorCount = ref(0);
const isCreatingSponsor = ref(false);
const sponsorBeingUpdated = ref(0);
const sponsorBeingDeleted = ref(0);
const isApplyingSponsorCount = ref(false);
const isCreatingCoupon = ref(false);
const couponBeingSaved = ref(0);
const couponBeingDeleted = ref(0);
const partnerBeingUpdated = ref(0);
const partnerBeingDeleted = ref(0);
const couponError = ref("");
const couponSuccess = ref("");
const lastCreatedEventLink = ref("");
const isClosingVotes = ref(false);
const closeVotesMessage = ref("");
const quizConfigsByEvent = reactive({});
const quizQuestionsByEvent = reactive({});
const eventStoriesById = reactive({});
const eventStoriesLoading = reactive({});
const eventStoriesSaving = reactive({});
const eventStoriesUploading = reactive({});
const eventPrizeDrafts = reactive({});
const eventPrizeErrors = reactive({});
const eventFeedbackDrafts = reactive({});
const eventFeedbackErrors = reactive({});
const eventActiveTab = reactive({});
const createFormOpen = ref(true);
function setEventTab(eventId, tab) { eventActiveTab[eventId] = tab; }
function getEventTab(eventId) { return eventActiveTab[eventId] || 'settings'; }
const savingEventPrizes = ref(0);
const portalRef = ref(null);
const toolbarRef = ref(null);

const fallbackTeamId = () => (teams.value.length ? teams.value[0].id : 0);

const resetPlayerSlot = (slot) => {
  Object.assign(slot, createEmptyPlayerSlot(fallbackTeamId()));
};

const resetAllPlayerSlots = () => {
  playerSlots.forEach((slot) => resetPlayerSlot(slot));
};

const ensurePlayerSlotTeams = () => {
  const fallback = fallbackTeamId();
  if (!fallback) {
    return;
  }
  playerSlots.forEach((slot) => {
    if (!slot.team_id) {
      slot.team_id = fallback;
    }
  });
};

const slotHasContent = (slot) => {
  if (!slot) {
    return false;
  }
  const jersey =
    typeof slot.jersey_number === "number"
      ? slot.jersey_number.toString()
      : `${slot.jersey_number || ""}`;
  return (
    slot.first_name.trim() ||
    slot.last_name.trim() ||
    slot.role.trim() ||
    jersey.trim() ||
    slot.image_url.trim()
  );
};

const normalizePlayerPayload = (slot, fallbackTeam) => {
  const sanitizedJersey = Number(slot.jersey_number);
  const jerseyNumber = Number.isFinite(sanitizedJersey) ? sanitizedJersey : 0;
  return {
    first_name: slot.first_name.trim(),
    last_name: slot.last_name.trim(),
    role: slot.role.trim(),
    jersey_number: jerseyNumber,
    image_url: slot.image_url.trim(),
    team_id: slot.team_id || fallbackTeam || 0,
    is_called_up: Boolean(slot.is_called_up),
  };
};

const loadImageFromDataUrl = (dataUrl) =>
  new Promise((resolve, reject) => {
    const image = new Image();
    image.decoding = "async";
    image.onload = () => resolve(image);
    image.onerror = () =>
      reject(new Error("Impossibile caricare l'immagine selezionata."));
    image.src = dataUrl;
  });

const toDataUrlSafely = (canvas, type, quality) => {
  try {
    if (typeof quality === "number") {
      return canvas.toDataURL(type, quality);
    }
    return canvas.toDataURL(type);
  } catch (error) {
    console.warn(
      "Impossibile convertire l'immagine nel formato richiesto:",
      error,
    );
    return "";
  }
};

const extractMimeType = (dataUrl) => {
  if (typeof dataUrl !== "string") {
    return "";
  }
  const match = /^data:([^;]+);/i.exec(dataUrl);
  return match ? match[1] : "";
};

const optimizePlayerImage = async (file) => {
  const originalDataUrl = await readFileAsDataUrl(file);
  if (!originalDataUrl) {
    return "";
  }

  try {
    const image = await loadImageFromDataUrl(originalDataUrl);
    const { naturalWidth: width, naturalHeight: height } = image;
    if (!width || !height) {
      return originalDataUrl;
    }

    const scale = Math.min(
      1,
      PLAYER_IMAGE_MAX_WIDTH / width,
      PLAYER_IMAGE_MAX_HEIGHT / height,
    );
    const targetWidth = Math.max(1, Math.round(width * scale));
    const targetHeight = Math.max(1, Math.round(height * scale));

    const canvas = document.createElement("canvas");
    canvas.width = targetWidth;
    canvas.height = targetHeight;

    const context = canvas.getContext("2d");
    if (!context) {
      return originalDataUrl;
    }

    context.drawImage(image, 0, 0, targetWidth, targetHeight);

    const originalType = extractMimeType(originalDataUrl);
    const candidateTypes = Array.from(
      new Set(["image/webp", "image/jpeg", originalType].filter(Boolean)),
    );

    let bestDataUrl = originalDataUrl;
    let bestSize = originalDataUrl.length;

    candidateTypes.forEach((type) => {
      const quality = type === "image/png" ? undefined : PLAYER_IMAGE_QUALITY;
      const candidate = toDataUrlSafely(canvas, type, quality);
      if (candidate && candidate.length < bestSize) {
        bestDataUrl = candidate;
        bestSize = candidate.length;
      }
    });

    return bestDataUrl;
  } catch (error) {
    console.warn("Impossibile ottimizzare l'immagine del giocatore:", error);
    return originalDataUrl;
  }
};

const handlePlayerImageChange = async (index, event) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = "";
  playerSaveError.value = "";
  const input = event?.target;
  const file = input?.files?.[0];
  if (!file) {
    slot.image_preview = slot.image_url || "";
    return;
  }
  const changeToken = Symbol("player-image-change");
  slot._imageChangeToken = changeToken;

  try {
    const optimizedDataUrl = await optimizePlayerImage(file);
    if (slot._imageChangeToken === changeToken && optimizedDataUrl) {
      slot.image_url = optimizedDataUrl;
      slot.image_preview = optimizedDataUrl;
    }
  } catch (error) {
    console.warn("Caricamento immagine giocatore non riuscito:", error);
  } finally {
    if (slot._imageChangeToken === changeToken) {
      slot._imageChangeToken = null;
    }
    if (input) {
      input.value = "";
    }
  }
};

const handlePlayerUrlChange = (index) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = "";
  playerSaveError.value = "";
  slot.image_preview = slot.image_url || "";
};

const removePlayerImage = (index) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = "";
  playerSaveError.value = "";
  slot.image_url = "";
  slot.image_preview = "";
};

const normalizePlayerResponse = (item) => {
  const firstName =
    typeof item?.first_name === "string" ? item.first_name.trim() : "";
  const lastName =
    typeof item?.last_name === "string" ? item.last_name.trim() : "";
  const role = typeof item?.role === "string" ? item.role.trim() : "";
  const jerseyRaw =
    typeof item?.jersey_number === "number"
      ? item.jersey_number
      : Number(item?.jersey_number);
  const jerseyNumber = Number.isFinite(jerseyRaw) ? jerseyRaw : 0;
  const image =
    typeof item?.image_url === "string" ? item.image_url.trim() : "";
  const team = Number(item?.team_id) || 0;
  const isCalledUp = Boolean(
    item && Object.prototype.hasOwnProperty.call(item, "is_called_up")
      ? item.is_called_up
      : true,
  );
  return {
    id: Number(item?.id) || 0,
    first_name: firstName,
    last_name: lastName,
    role,
    jersey_number: jerseyNumber,
    image_url: image,
    team_id: team,
    is_called_up: isCalledUp,
  };
};

const normalizeSelfieResponse = (item) => {
  if (!item) {
    return null;
  }
  const id = Number(item?.id) || 0;
  if (!id) {
    return null;
  }
  const eventId = Number(item?.event_id) || 0;
  const caption = typeof item?.caption === "string" ? item.caption.trim() : "";
  const imageUrl =
    typeof item?.image_url === "string" ? item.image_url.trim() : "";
  const approved = Boolean(item?.approved);
  const showOnScreen = Boolean(item?.show_on_screen);
  const acceptedImageTerms = Boolean(
    Object.prototype.hasOwnProperty.call(item || {}, "accepted_image_terms")
      ? item.accepted_image_terms
      : item?.acceptedImageTerms,
  );
  const deviceToken =
    typeof item?.device_token === "string" ? item.device_token : "";
  const fileSize = Number(item?.file_size_bytes);
  const fileSizeBytes =
    Number.isFinite(fileSize) && fileSize >= 0 ? fileSize : 0;
  const submittedAt =
    typeof item?.submitted_at === "string"
      ? item.submitted_at
      : typeof item?.created_at === "string"
        ? item.created_at
        : "";
  return {
    id,
    event_id: eventId,
    caption,
    image_url: imageUrl,
    image_src: imageUrl ? resolveApiUrl(imageUrl) : "",
    content_type:
      typeof item?.content_type === "string" ? item.content_type : "",
    approved,
    show_on_screen: showOnScreen,
    accepted_image_terms: acceptedImageTerms,
    device_token: deviceToken,
    file_size_bytes: fileSizeBytes,
    submitted_at: submittedAt,
  };
};

const sortPlayersForDisplay = (a, b) => {
  if (a.jersey_number !== b.jersey_number) {
    const jerseyA = a.jersey_number || Number.MAX_SAFE_INTEGER;
    const jerseyB = b.jersey_number || Number.MAX_SAFE_INTEGER;
    if (jerseyA !== jerseyB) {
      return jerseyA - jerseyB;
    }
  }
  const lastComparison = a.last_name.localeCompare(b.last_name);
  if (lastComparison !== 0) {
    return lastComparison;
  }
  const firstComparison = a.first_name.localeCompare(b.first_name);
  if (firstComparison !== 0) {
    return firstComparison;
  }
  return a.id - b.id;
};

const applyPlayersToSlots = () => {
  const sorted = [...players.value];
  sorted.sort(sortPlayersForDisplay);
  players.value = sorted;
  playerOverflow.value =
    sorted.length > playerSlotCount ? sorted.slice(playerSlotCount) : [];
  const fallback = fallbackTeamId();
  for (let index = 0; index < playerSlotCount; index += 1) {
    const slot = playerSlots[index];
    const player = sorted[index];
    if (slot && player) {
      Object.assign(slot, {
        id: player.id,
        first_name: player.first_name,
        last_name: player.last_name,
        role: player.role,
        jersey_number: player.jersey_number
          ? player.jersey_number.toString()
          : "",
        team_id: player.team_id || fallback,
        is_called_up: player.is_called_up,
        image_url: player.image_url,
        image_preview: player.image_url || "",
      });
    } else if (slot) {
      resetPlayerSlot(slot);
    }
  }
  ensurePlayerSlotTeams();
};

const restorePlayerSlots = () => {
  applyPlayersToSlots();
  playerSaveError.value = "";
  playerSaveMessage.value = "";
};

const savePlayers = async () => {
  if (isSavingPlayers.value) {
    return;
  }
  playerSaveError.value = "";
  playerSaveMessage.value = "";

  const fallback = fallbackTeamId();
  const hasAnyContent = playerSlots.some((slot) => slotHasContent(slot));
  if (!fallback && hasAnyContent) {
    playerSaveError.value =
      "Crea almeno una squadra e assegnala ai giocatori prima di salvare.";
    return;
  }

  isSavingPlayers.value = true;
  const handledIds = new Set();

  try {
    const schemaToSave = validateRosterSchema(rosterSchema.value);
    await secureRequest(() =>
      apiClient.put(
        "/players/settings",
        { roster_schema: schemaToSave },
        authHeaders.value,
      ),
    );

    for (const slot of playerSlots) {
      const hasContent = slotHasContent(slot);
      if (hasContent) {
        const payload = normalizePlayerPayload(slot, fallback);
        if (!payload.first_name || !payload.last_name || !payload.role) {
          playerSaveError.value =
            "Nome, cognome e ruolo sono obbligatori per ogni giocatore salvato.";
          isSavingPlayers.value = false;
          return;
        }
        if (!payload.team_id) {
          playerSaveError.value =
            "Seleziona una squadra per ogni giocatore salvato.";
          isSavingPlayers.value = false;
          return;
        }

        if (slot.id) {
          await secureRequest(() =>
            apiClient.put(`/players/${slot.id}`, payload, authHeaders.value),
          );
          handledIds.add(slot.id);
        } else {
          const { data } = await secureRequest(() =>
            apiClient.post("/players", payload, authHeaders.value),
          );
          const createdId = Number(data?.id) || 0;
          if (createdId) {
            slot.id = createdId;
            handledIds.add(createdId);
          }
        }
      } else if (slot.id) {
        await secureRequest(() =>
          apiClient.delete(`/players/${slot.id}`, authHeaders.value),
        );
        handledIds.add(slot.id);
        resetPlayerSlot(slot);
      } else {
        resetPlayerSlot(slot);
      }
    }

    for (const player of players.value) {
      if (!handledIds.has(player.id)) {
        await secureRequest(() =>
          apiClient.delete(`/players/${player.id}`, authHeaders.value),
        );
        handledIds.add(player.id);
      }
    }

    await loadPlayers();
    playerSaveMessage.value = "Giocatori salvati con successo.";
  } catch (error) {
    if (!playerSaveError.value) {
      playerSaveError.value =
        "Si è verificato un errore durante il salvataggio dei giocatori. Riprova.";
    }
  } finally {
    isSavingPlayers.value = false;
  }
};

const hasEnoughTeams = computed(() => teams.value.length >= 2);
const availableEvents = computed(() =>
  events.value.filter((event) => !event.is_concluded),
);
const visibleEvents = computed(() => availableEvents.value);

const activeEventId = computed(() => {
  const activeEvent = events.value.find((event) => event.is_active);
  return activeEvent ? activeEvent.id : 0;
});
const activeSponsorCount = computed(
  () => sponsors.value.filter((item) => item.isActive).length,
);
const sponsorSliderMax = computed(() =>
  sponsors.value.length
    ? Math.min(maxSponsors, sponsors.value.length)
    : maxSponsors,
);
const selectedResultsEvent = computed(
  () =>
    availableEvents.value.find(
      (event) => event.id === selectedResultsEventId.value,
    ) || null,
);
const activeEventEntry = computed(
  () => events.value.find((event) => event.id === activeEventId.value) || null,
);
const selectedSelfieEvent = computed(
  () =>
    availableEvents.value.find(
      (event) => event.id === selectedSelfieEventId.value,
    ) || null,
);
const selectedSelfieEventLabel = computed(() =>
  selectedSelfieEvent.value ? eventLabel(selectedSelfieEvent.value) : "",
);
const activeEventVotesClosed = computed(() =>
  Boolean(activeEventEntry.value?.votes_closed),
);
const activeEventLabel = computed(() =>
  activeEventEntry.value
    ? eventLabel(activeEventEntry.value)
    : "Nessun evento attivo",
);
const activeEventDateLabel = computed(() =>
  activeEventEntry.value
    ? formatEventDate(activeEventEntry.value.start_datetime)
    : "",
);
const activeEventLocation = computed(() =>
  activeEventEntry.value?.location?.trim()
    ? activeEventEntry.value.location.trim()
    : "Location da definire",
);
const selectedResultsEventLabel = computed(() =>
  selectedResultsEvent.value ? eventLabel(selectedResultsEvent.value) : "",
);
const selectedResultsEventDate = computed(() =>
  selectedResultsEvent.value
    ? formatEventDate(selectedResultsEvent.value.start_datetime)
    : "",
);
const resultsLeaderboard = computed(() => {
  const aggregated = new Map(
    eventResults.value.map((item) => [
      Number(item.player_id) || 0,
      {
        votes: Number(item.votes) || 0,
        lastVoteAt:
          typeof item.last_vote_at === "string" ? item.last_vote_at : "",
      },
    ]),
  );

  const entries = players.value.map((player) => {
    const stats = aggregated.get(player.id) || { votes: 0, lastVoteAt: "" };
    const firstName = player.first_name || "";
    const lastName = player.last_name || "";
    const fullName =
      `${firstName} ${lastName}`.trim() || `Giocatore ${player.id}`;
    const lastNameUpper = (lastName || firstName || fullName).toUpperCase();
    return {
      id: player.id,
      firstName: firstName || fullName,
      lastName,
      lastNameUpper,
      fullName,
      votes: stats.votes,
      lastVoteAt: stats.lastVoteAt,
    };
  });

  aggregated.forEach((stats, playerId) => {
    if (!entries.some((entry) => entry.id === playerId)) {
      const fallbackName = `Giocatore ${playerId}`;
      entries.push({
        id: playerId,
        firstName: fallbackName,
        lastName: "",
        lastNameUpper: fallbackName.toUpperCase(),
        fullName: fallbackName,
        votes: stats.votes,
        lastVoteAt: stats.lastVoteAt,
      });
    }
  });

  entries.sort((a, b) => {
    if (b.votes !== a.votes) {
      return b.votes - a.votes;
    }
    if (a.lastVoteAt && b.lastVoteAt && a.lastVoteAt !== b.lastVoteAt) {
      return a.lastVoteAt.localeCompare(b.lastVoteAt);
    }
    if (a.lastVoteAt && !b.lastVoteAt) {
      return -1;
    }
    if (!a.lastVoteAt && b.lastVoteAt) {
      return 1;
    }
    const lastNameComparison = a.lastName.localeCompare(b.lastName);
    if (lastNameComparison !== 0) {
      return lastNameComparison;
    }
    const firstNameComparison = a.firstName.localeCompare(b.firstName);
    if (firstNameComparison !== 0) {
      return firstNameComparison;
    }
    return a.id - b.id;
  });

  let highestVotes = 0;
  entries.forEach((entry) => {
    if (entry.votes > highestVotes) {
      highestVotes = entry.votes;
    }
  });

  return entries.map((entry) => ({
    ...entry,
    percentage:
      highestVotes > 0 ? Math.round((entry.votes / highestVotes) * 100) : 0,
  }));
});

const sponsorAnalyticsDisplay = computed(() => {
  const data = sponsorAnalytics.value;
  if (!data) {
    return null;
  }

  return {
    totalUsers: data.totalUsers,
    totalUsersLabel: data.totalUsers.toLocaleString("it-IT"),
    seenUsers: data.seenUsers,
    seenUsersLabel: data.seenUsers.toLocaleString("it-IT"),
    seenRateLabel: `${formatPercent(data.seenRate)}%`,
    watchedUsers: data.watchedUsers,
    watchedUsersLabel: data.watchedUsers.toLocaleString("it-IT"),
    averageWatchTimeLabel: formatWatchDuration(data.averageWatchTimeMs),
    totalClicks: data.totalClicks,
    totalClicksLabel: data.totalClicks.toLocaleString("it-IT"),
    uniqueClickersLabel: data.uniqueClickers.toLocaleString("it-IT"),
    clickRateLabel: `${formatPercent(data.clickRate)}%`,
    topSponsorName:
      data.topSponsor?.reportName?.trim() ||
      data.topSponsor?.name?.trim() ||
      "Nessuno",
    topSponsorViewsLabel: data.topSponsor
      ? data.topSponsor.views.toLocaleString("it-IT")
      : "0",
  };
});

const sponsorTimelinePoints = computed(() => {
  if (
    !sponsorAnalytics.value ||
    !Array.isArray(sponsorAnalytics.value.timeline)
  ) {
    return [];
  }

  return sponsorAnalytics.value.timeline.map((item) => {
    const timestamp = typeof item.timestamp === "string" ? item.timestamp : "";
    let label = timestamp;
    if (timestamp) {
      const date = new Date(timestamp);
      if (!Number.isNaN(date.getTime())) {
        label = analyticsTimeFormatter.format(date);
      }
    }
    const seen = Number(item.seen) || 0;
    const watched = Number(item.watched) || 0;
    const clicks = Number(item.clicks) || 0;
    return { timestamp, label, seen, watched, clicks };
  });
});

const sponsorTimelineMaxValue = computed(() => {
  const points = sponsorTimelinePoints.value;
  if (!points.length) {
    return 1;
  }
  return points.reduce(
    (max, point) => Math.max(max, point.seen, point.clicks),
    1,
  );
});

const sponsorChartRows = computed(() => {
  const maxValue = sponsorTimelineMaxValue.value || 1;
  return sponsorTimelinePoints.value.map((point) => ({
    ...point,
    seenPercent: maxValue ? Math.round((point.seen / maxValue) * 100) : 0,
    clicksPercent: maxValue ? Math.round((point.clicks / maxValue) * 100) : 0,
  }));
});

const hasSponsorAnalyticsData = computed(() => {
  const data = sponsorAnalytics.value;
  if (!data) {
    return false;
  }
  const timelineLength = Array.isArray(data.timeline)
    ? data.timeline.length
    : 0;
  return Boolean(data.totalUsers || data.totalClicks || timelineLength);
});
const totalVotes = computed(() =>
  eventResults.value.reduce((sum, item) => sum + (Number(item.votes) || 0), 0),
);
const hasResultsVotes = computed(() => totalVotes.value > 0);
const lastResultsUpdateLabel = computed(() =>
  lastResultsUpdate.value
    ? lastResultsUpdate.value.toLocaleString("it-IT")
    : "",
);
const filteredCouponEvents = computed(() => {
  const term = couponEventSearch.value.trim().toLowerCase();
  if (!term) {
    return events.value;
  }
  return events.value.filter((event) =>
    couponMatchLabel(event).toLowerCase().includes(term) ||
    String(event.start_datetime || "").toLowerCase().includes(term),
  );
});

const activeUsername = ref(localStorage.getItem("adminUsername") || "");
const activeRole = ref(localStorage.getItem("adminRole") || "");
const isAuthenticated = computed(() => Boolean(activeUsername.value));
const isSuperAdmin = computed(() => activeRole.value === "superadmin");
const isStaff = computed(() => activeRole.value === "staff");
const isBarAdmin = computed(() => activeRole.value === "bar");
const isBarFeatureEnabled = ref(true);
const availableTabs = computed(() => {
  const filteredTabs = isBarFeatureEnabled.value
    ? tabs
    : tabs.filter((tab) => tab.id !== "bar");
  if (isSuperAdmin.value) {
    return filteredTabs;
  }
  if (isBarAdmin.value) {
    return filteredTabs.filter((tab) => BAR_TAB_IDS.has(tab.id));
  }
  return filteredTabs.filter((tab) => STAFF_TAB_IDS.has(tab.id));
});

const navigationGroups = computed(() => {
  const allowed = new Map(availableTabs.value.map((tab) => [tab.id, tab]));
  const groups = [
    { label: "Panoramica", ids: ["dashboard", "events"] },
    { label: "Contenuti", ids: ["sponsors", "coupons", "selfies"] },
    { label: "Squadra", ids: ["teams", "players"] },
    { label: "Risultati", ids: ["closing", "results", "history"] },
    { label: "BAR", ids: ["bar"] },
    { label: "Impostazioni", ids: ["partners", "admins", "marketing"] },
  ];
  return groups
    .map((group) => ({
      label: group.label,
      items: group.ids.map((id) => allowed.get(id)).filter(Boolean),
    }))
    .filter((group) => group.items.length);
});

const sectionMetaMap = {
  dashboard: { title: "Dashboard", group: "Panoramica" },
  events: { title: "Eventi", group: "Panoramica" },
  sponsors: { title: "Sponsor", group: "Contenuti" },
  coupons: { title: "Coupon", group: "Contenuti" },
  selfies: { title: "Selfie MVP", group: "Contenuti" },
  teams: { title: "Squadre", group: "Squadra" },
  players: { title: "Giocatori", group: "Squadra" },
  closing: { title: "Chiusura votazioni", group: "Risultati" },
  results: { title: "Risultati", group: "Risultati" },
  history: { title: "Storico eventi", group: "Risultati" },
  bar: { title: "BAR", group: "BAR" },
  partners: { title: "Partners", group: "Impostazioni" },
  admins: { title: "Admin", group: "Impostazioni" },
  marketing: { title: "Marketing", group: "Impostazioni" },
};

const currentSectionMeta = computed(() => sectionMetaMap[section.value] || sectionMetaMap.dashboard);
const currentSectionTitle = computed(() => currentSectionMeta.value.title);
const currentSectionGroup = computed(() => currentSectionMeta.value.group);

const dashboardActions = computed(() =>
  availableTabs.value
    .filter((tab) => ["events", "sponsors", "coupons", "players", "marketing"].includes(tab.id))
    .map((tab) => ({
      ...tab,
      description:
        tab.id === "events"
          ? "Crea o aggiorna partite e flussi di voto."
          : tab.id === "sponsors"
            ? "Gestisci creatività e ordine sponsor."
            : tab.id === "coupons"
              ? "Configura coupon e associazioni alle partite."
              : tab.id === "marketing"
              ? "Gestisci audience, template e invii SMS marketing."
              : "Aggiorna roster e immagini giocatori.",
    })),
);

const loginForm = reactive({
  username: "",
  password: "",
});
const isLoggingIn = ref(false);
const loginError = ref("");
const globalError = ref("");

const authHeaders = computed(() => {
  const headers = {};
  if (resolvedOrganizationSlug.value) {
    headers["X-Organization-Slug"] = resolvedOrganizationSlug.value;
  }
  return { headers };
});

function resetNewEventPrizes() {
  newEventPrizes.value = [{ name: "", winSmsText: "" }];
}

function resetForms() {
  newTeamName.value = "";
  Object.assign(newEvent, createDefaultNewEventState());
  assignSurveyDraft(newEventSurvey, null);
  resetNewEventPrizes();
  teamInputs.home = "";
  teamInputs.away = "";
  Object.assign(newAdmin, { username: "", password: "", role: "" });
  Object.assign(newPartner, { name: "", username: "", password: "" });
  resetNewSponsorForm();
  resetNewCouponForm();
  desiredActiveSponsorCount.value = Math.min(
    sponsorSliderMax.value,
    activeSponsorCount.value,
  );
  restorePlayerSlots();
  playerSaveError.value = "";
  playerSaveMessage.value = "";
}

function selectDefaultSection() {
  section.value = isSuperAdmin.value ? "dashboard" : "closing";
}

function selectSection(nextSection) {
  section.value = nextSection;
  isSidebarOpen.value = false;
}

function ensureValidTeamSelection() {
  if (!hasEnoughTeams.value) {
    newEvent.team1_id = 0;
    newEvent.team2_id = 0;
    teamInputs.home = "";
    teamInputs.away = "";
    return;
  }

  const availableIds = new Set(teams.value.map((team) => team.id));

  if (!availableIds.has(newEvent.team1_id)) {
    newEvent.team1_id = 0;
    teamInputs.home = "";
  }

  if (
    !availableIds.has(newEvent.team2_id) ||
    (newEvent.team1_id !== 0 && newEvent.team1_id === newEvent.team2_id)
  ) {
    newEvent.team2_id = 0;
    teamInputs.away = "";
  }

  syncTeamInputsFromIds();
}

watch(teams, () => {
  ensureValidTeamSelection();
  ensurePlayerSlotTeams();
});
watch(hasEnoughTeams, (enough) => {
  if (!enough) {
    newEvent.team1_id = 0;
    newEvent.team2_id = 0;
    teamInputs.home = "";
    teamInputs.away = "";
  }
});

watch(events, (value) => {
  ensureResultsSelection();
  ensureSelfieSelection();
  const editableEvents = Array.isArray(value)
    ? value.filter((event) => !event.is_concluded)
    : [];
  syncEventPrizeDrafts(editableEvents);
  syncEventFeedbackDrafts(editableEvents);
  if (section.value === "results" && selectedResultsEventId.value) {
    fetchEventResults();
  }
});

watch(activeEventId, () => {
  closeVotesMessage.value = "";
});

watch(activeEventVotesClosed, (closed) => {
  if (!closed) {
    closeVotesMessage.value = "";
  }
});

watch(
  partners,
  (list) => {
    if (!newCoupon.sponsorId && Array.isArray(list) && list.length) {
      newCoupon.sponsorId = list[0].id;
    }
    if (!newCoupon.merchantId && Array.isArray(list) && list.length) {
      newCoupon.merchantId = list[0].id;
    }
  },
  { immediate: true },
);

watch(
  () => newCoupon.sponsorId,
  (sponsorId) => {
    if (Number.isFinite(sponsorId) && sponsorId > 0) {
      newCoupon.merchantId = sponsorId;
    }
  },
);

function clearCollections() {
  teams.value = [];
  players.value = [];
  events.value = [];
  admins.value = [];
  partners.value = [];
  sponsors.value = [];
  coupons.value = [];
  eventHistory.value = [];
  eventSelfies.value = [];
  hasLoadedEventHistory.value = false;
  eventHistoryError.value = "";
  eventHistorySuccess.value = "";
  isLoadingSelfies.value = false;
  selfieLoadError.value = "";
  selfieModerationMessage.value = "";
  selectedSelfieEventId.value = 0;
  resetAllPlayerSlots();
  playerOverflow.value = [];
  playerSaveError.value = "";
  playerSaveMessage.value = "";
  Object.keys(eventPrizeDrafts).forEach((key) => {
    delete eventPrizeDrafts[key];
  });
  Object.keys(eventPrizeErrors).forEach((key) => {
    delete eventPrizeErrors[key];
  });
  Object.keys(eventStoriesById).forEach((key) => {
    delete eventStoriesById[key];
  });
  Object.keys(eventStoriesLoading).forEach((key) => {
    delete eventStoriesLoading[key];
  });
  Object.keys(eventStoriesSaving).forEach((key) => {
    delete eventStoriesSaving[key];
  });
  Object.keys(eventStoriesUploading).forEach((key) => {
    delete eventStoriesUploading[key];
  });
  Object.keys(selfieBusyState).forEach((key) => {
    delete selfieBusyState[key];
  });
  Object.keys(historyReportDownloadState).forEach((key) => {
    delete historyReportDownloadState[key];
  });
  lastCreatedEventLink.value = "";
  resetNewCouponForm();
  couponBeingSaved.value = 0;
  couponBeingDeleted.value = 0;
  isCreatingCoupon.value = false;
  resetNewEventPrizes();
  resetResultsState();
  sponsorAnalytics.value = null;
  sponsorAnalyticsError.value = "";
  isLoadingSponsorAnalytics.value = false;
}

function stopResultsPolling() {
  if (resultsSseSource) {
    resultsSseSource.close();
    resultsSseSource = null;
  }
}

function startResultsPolling() {
  stopResultsPolling();
  if (!selectedResultsEventId.value || typeof EventSource === 'undefined') {
    return;
  }
  const base = resolveApiUrl(`/events/${selectedResultsEventId.value}/votes/stream`);
  const slug = getOrganizationSlug();
  const url = slug ? base + (base.includes('?') ? '&' : '?') + 'organization_slug=' + encodeURIComponent(slug) : base;
  resultsSseSource = new EventSource(url);
  resultsSseSource.addEventListener('message', () => {
    fetchEventResults().catch(() => { /* silent */ });
  });
}

function resetResultsState() {
  stopResultsPolling();
  selectedResultsEventId.value = 0;
  eventResults.value = [];
  resultsError.value = "";
  lastResultsUpdate.value = null;
  isLoadingResults.value = false;
  sponsorAnalytics.value = null;
  sponsorAnalyticsError.value = "";
  isLoadingSponsorAnalytics.value = false;
}

function ensureResultsSelection() {
  const available = availableEvents.value;
  if (!available.length) {
    selectedResultsEventId.value = 0;
    return;
  }
  const exists = available.some(
    (event) => event.id === selectedResultsEventId.value,
  );
  if (!exists) {
    const active = available.find((event) => event.is_active);
    selectedResultsEventId.value = active ? active.id : available[0].id;
  }
}

async function fetchEventResults({ showLoader = false } = {}) {
  if (!selectedResultsEventId.value) {
    eventResults.value = [];
    resultsError.value = "";
    lastResultsUpdate.value = null;
    return;
  }
  if (showLoader) {
    isLoadingResults.value = true;
  }
  resultsError.value = "";
  try {
    const { data } = await secureRequest(() =>
      apiClient.get(
        `/events/${selectedResultsEventId.value}/results`,
        authHeaders.value,
      ),
    );
    if (Array.isArray(data)) {
      eventResults.value = data.map((item) => ({
        player_id: Number(item.player_id) || 0,
        votes: Number(item.votes) || 0,
        last_vote_at:
          typeof item.last_vote_at === "string" ? item.last_vote_at : "",
      }));
    } else {
      eventResults.value = [];
    }
    lastResultsUpdate.value = new Date();
  } catch (error) {
    if (error?.response?.status === 404) {
      resultsError.value = "Evento non trovato.";
    } else if (error?.response?.status === 400) {
      resultsError.value = "Richiesta non valida per i risultati.";
    } else if (error?.response?.status !== 401) {
      resultsError.value =
        "Impossibile caricare i risultati. Riprova più tardi.";
    }
  } finally {
    if (showLoader) {
      isLoadingResults.value = false;
    }
  }

  if (isStaff.value) {
    sponsorAnalytics.value = null;
    sponsorAnalyticsError.value = "";
    isLoadingSponsorAnalytics.value = false;
    return;
  }

  fetchSponsorAnalytics({ showLoader }).catch(() => {});
}

function normalizeSponsorAnalyticsResponse(raw) {
  if (!raw || typeof raw !== "object") {
    return {
      totalUsers: 0,
      seenUsers: 0,
      watchedUsers: 0,
      averageWatchTimeMs: 0,
      totalWatchTimeMs: 0,
      totalClicks: 0,
      uniqueClickers: 0,
      seenRate: 0,
      clickRate: 0,
      topSponsor: null,
      timeline: [],
    };
  }

  const resolveNumber = (value) => {
    const parsed = Number(value ?? 0);
    return Number.isFinite(parsed) ? parsed : 0;
  };

  const topSponsorRaw = raw.top_sponsor ?? raw.topSponsor ?? null;
  let topSponsor = null;
  if (topSponsorRaw && typeof topSponsorRaw === "object") {
    const id = resolveNumber(
      topSponsorRaw.sponsor_id ?? topSponsorRaw.sponsorId,
    );
    const name =
      typeof topSponsorRaw.name === "string" ? topSponsorRaw.name : "";
    const reportName =
      typeof topSponsorRaw.report_name === "string"
        ? topSponsorRaw.report_name
        : typeof topSponsorRaw.reportName === "string"
          ? topSponsorRaw.reportName
          : "";
    const views = resolveNumber(topSponsorRaw.views);
    topSponsor = {
      id,
      name: typeof name === "string" ? name.trim() : "",
      reportName:
        typeof reportName === "string" ? reportName.trim() : "",
      views,
    };
  }

  const timeline = Array.isArray(raw.timeline)
    ? raw.timeline.map((item) => ({
        timestamp: typeof item?.timestamp === "string" ? item.timestamp : "",
        seen: resolveNumber(item?.seen),
        watched: resolveNumber(item?.watched),
        clicks: resolveNumber(item?.clicks),
      }))
    : [];

  return {
    totalUsers: resolveNumber(raw.total_users ?? raw.totalUsers),
    seenUsers: resolveNumber(raw.seen_users ?? raw.seenUsers),
    watchedUsers: resolveNumber(raw.watched_users ?? raw.watchedUsers),
    averageWatchTimeMs: resolveNumber(
      raw.average_watch_time_ms ?? raw.averageWatchTimeMs,
    ),
    totalWatchTimeMs: resolveNumber(
      raw.total_watch_time_ms ?? raw.totalWatchTimeMs,
    ),
    totalClicks: resolveNumber(raw.total_clicks ?? raw.totalClicks),
    uniqueClickers: resolveNumber(raw.unique_clickers ?? raw.uniqueClickers),
    seenRate: resolveNumber(raw.seen_rate ?? raw.seenRate),
    clickRate: resolveNumber(raw.click_rate ?? raw.clickRate),
    topSponsor,
    timeline,
  };
}

async function fetchSponsorAnalytics({ showLoader = false } = {}) {
  if (!selectedResultsEventId.value) {
    sponsorAnalytics.value = null;
    sponsorAnalyticsError.value = "";
    return;
  }

  if (showLoader) {
    isLoadingSponsorAnalytics.value = true;
  }
  sponsorAnalyticsError.value = "";

  try {
    const { data } = await secureRequest(() =>
      apiClient.get(
        `/admin/events/${selectedResultsEventId.value}/sponsors/analytics`,
        authHeaders.value,
      ),
    );
    sponsorAnalytics.value = normalizeSponsorAnalyticsResponse(data);
  } catch (error) {
    if (error?.response?.status === 404) {
      sponsorAnalytics.value = null;
      sponsorAnalyticsError.value =
        "Nessun dato sponsor disponibile per questo evento.";
    } else if (error?.response?.status !== 401) {
      sponsorAnalyticsError.value =
        "Impossibile caricare le statistiche sponsor.";
    }
    throw error;
  } finally {
    if (showLoader) {
      isLoadingSponsorAnalytics.value = false;
    }
  }
}

function normalizePrizeResponse(prize, index = 0) {
  if (!prize || typeof prize !== "object") {
    return null;
  }
  const winner =
    prize.winner && typeof prize.winner === "object" ? prize.winner : null;
  const normalizedWinner = winner
    ? {
        voteId: Number(winner.vote_id ?? winner.voteId) || 0,
        ticketCode:
          typeof (winner.ticket_code ?? winner.ticketCode) === "string"
            ? (winner.ticket_code ?? winner.ticketCode)
            : "",
        playerId: Number(winner.player_id ?? winner.playerId) || 0,
        playerFirstName:
          typeof (winner.player_first_name ?? winner.playerFirstName) ===
          "string"
            ? (winner.player_first_name ?? winner.playerFirstName)
            : "",
        playerLastName:
          typeof (winner.player_last_name ?? winner.playerLastName) === "string"
            ? (winner.player_last_name ?? winner.playerLastName)
            : "",
        assignedAt:
          typeof (winner.assigned_at ?? winner.assignedAt) === "string"
            ? (winner.assigned_at ?? winner.assignedAt)
            : "",
      }
    : null;

  const position = Number(prize.position) || index + 1;
  return {
    id: Number(prize.id) || 0,
    eventId: Number(prize.event_id ?? prize.eventId) || 0,
    name: typeof prize.name === "string" ? prize.name : "",
    winSmsText: typeof (prize.win_sms_text ?? prize.winSmsText) === 'string' ? (prize.win_sms_text ?? prize.winSmsText) : '',
    position,
    winner: normalizedWinner,
  };
}

function normalizeEventResponse(event) {
  const normalized = { ...event };
  normalized.is_active = Boolean(event?.is_active);
  normalized.votes_closed = Boolean(event?.votes_closed);
  normalized.is_concluded = Boolean(event?.is_concluded);
  const resolveFlag = (keys, fallback = true) => {
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
  normalized.show_reaction_test = resolveFlag(
    ["show_reaction_test", "showReactionTest"],
    true,
  );
  normalized.show_selfie = resolveFlag(["show_selfie", "showSelfie"], true);
  normalized.show_vote_trend = resolveFlag(
    ["show_vote_trend", "showVoteTrend", "show_live_results"],
    true,
  );
  normalized.show_feedback_survey = resolveFlag(
    ["show_feedback_survey", "showFeedbackSurvey"],
    true,
  );
  const preVoteSponsorBase = resolveFlag(
    [
      "show_pre_vote_sponsors",
      "showPreVoteSponsors",
      "show_sponsors",
      "showSponsors",
    ],
    true,
  );
  normalized.show_pre_vote_sponsors = preVoteSponsorBase;
  normalized.show_pre_vote_bottom_sponsors = resolveFlag(
    [
      "show_pre_vote_bottom_sponsors",
      "showPreVoteBottomSponsors",
      "show_pre_vote_sponsor_wall",
    ],
    preVoteSponsorBase,
  );
  normalized.show_vote_counter = resolveFlag(
    ["show_vote_counter", "showVoteCounter", "show_pre_vote_vote_counter"],
    true,
  );
  if (Array.isArray(event.prizes)) {
    const mapped = event.prizes
      .map((prize, index) => normalizePrizeResponse(prize, index))
      .filter(Boolean)
      .sort((a, b) => {
        if (a.position === b.position) {
          return a.id - b.id;
        }
        return a.position - b.position;
      });
    normalized.prizes = mapped;
  } else {
    normalized.prizes = [];
  }
  const survey = normalizeFeedbackSurveyInput(
    event?.feedback_survey ?? event?.feedbackSurvey,
  );
  normalized.feedback_survey = survey;
  normalized.feedbackSurvey = survey;

  normalized.show_branded_game = Boolean(event?.show_branded_game);
  try {
    const raw = event?.branded_game_config;
    normalized.brandedGameConfigDraft =
      raw && typeof raw === "string" && raw.trim()
        ? { ...defaultBrandedGameConfig(), ...JSON.parse(raw) }
        : defaultBrandedGameConfig();
  } catch {
    normalized.brandedGameConfigDraft = defaultBrandedGameConfig();
  }

  return normalized;
}

function normalizeSponsorResponse(item) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const normalizedName = typeof item.name === "string" ? item.name.trim() : "";
  const normalizedReportName =
    typeof item.report_name === "string"
      ? item.report_name.trim()
      : typeof item.reportName === "string"
        ? item.reportName.trim()
        : "";
  const normalizedLink =
    typeof item.link_url === "string" ? item.link_url.trim() : "";
  return {
    id: Number(item.id) || 0,
    name: normalizedName,
    reportName: normalizedReportName,
    linkUrl: normalizedLink,
    position: Number(item.position) || 0,
    logoData: typeof item.logo_data === "string" ? item.logo_data : "",
    isActive: Boolean(item.is_active),
  };
}

function toDateTimeLocalInput(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const pad = (num) => `${num}`.padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours(),
  )}:${pad(date.getMinutes())}`;
}

function fromDateTimeLocalInput(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return date.toISOString();
}

function normalizeCouponResponse(item) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const normalizedMatchIds = Array.isArray(item.match_ids ?? item.matchIds)
    ? (item.match_ids ?? item.matchIds)
        .map((value) => Number(value))
        .filter((value) => Number.isFinite(value) && value > 0)
    : [];
  return {
    id: Number(item.id) || 0,
    title: typeof item.title === "string" ? item.title : "",
    shortDesc:
      typeof item.short_desc === "string"
        ? item.short_desc
        : typeof item.shortDesc === "string"
          ? item.shortDesc
          : "",
    sponsorId: Number(item.sponsor_id ?? item.sponsorId) || 0,
    merchantId: Number(item.merchant_id ?? item.merchantId) || 0,
    matchIds: normalizedMatchIds,
    startDate:
      typeof item.start_date === "string"
        ? item.start_date
        : typeof item.startDate === "string"
          ? item.startDate
          : "",
    endDate:
      typeof item.end_date === "string"
        ? item.end_date
        : typeof item.endDate === "string"
          ? item.endDate
          : "",
    maxUses: Number(item.max_uses ?? item.maxUses) || 0,
    status: typeof item.status === "string" ? item.status : "",
    imageUrl:
      typeof item.image_url === "string"
        ? item.image_url
        : typeof item.imageUrl === "string"
          ? item.imageUrl
          : "",
    highlight: Boolean(item.highlight),
    totalViews: Number(item.total_views ?? item.totalViews) || 0,
    totalClaims: Number(item.total_claims ?? item.totalClaims) || 0,
    totalRedemptions:
      Number(item.total_redemptions ?? item.totalRedemptions) || 0,
    createdAt:
      typeof item.created_at === "string"
        ? item.created_at
        : typeof item.createdAt === "string"
          ? item.createdAt
          : "",
    updatedAt:
      typeof item.updated_at === "string"
        ? item.updated_at
        : typeof item.updatedAt === "string"
          ? item.updatedAt
          : "",
  };
}

function toEditableCoupon(coupon) {
  const normalized = normalizeCouponResponse(coupon);
  if (!normalized || !normalized.id) {
    return null;
  }
  return {
    ...normalized,
    startDateInput: toDateTimeLocalInput(normalized.startDate),
    endDateInput: toDateTimeLocalInput(normalized.endDate),
    imagePreview: resolveCouponImageSource(normalized.imageUrl),
  };
}

function serializeCouponPayload(coupon) {
  const normalized = normalizeCouponResponse(coupon);
  return {
    title: normalized?.title?.trim() || "",
    short_desc: normalized?.shortDesc?.trim() || "",
    sponsor_id: normalized?.sponsorId || 0,
    merchant_id: normalized?.merchantId || normalized?.sponsorId || 0,
    match_ids: Array.isArray(coupon?.matchIds)
      ? coupon.matchIds
          .map((value) => Number(value))
          .filter((value) => Number.isFinite(value) && value > 0)
      : [],
    start_date: fromDateTimeLocalInput(coupon?.startDateInput) || normalized?.startDate,
    end_date: fromDateTimeLocalInput(coupon?.endDateInput) || normalized?.endDate,
    max_uses: Number.isFinite(normalized?.maxUses) ? normalized.maxUses : 0,
    status: normalized?.status?.trim() || "draft",
    image_url: normalized?.imageUrl?.trim() || "",
    highlight: false,
  };
}

function resolveCouponImageSource(value) {
  const trimmed =
    typeof value === "string" || value instanceof String
      ? value.toString().trim()
      : "";
  if (!trimmed) {
    return "";
  }
  if (/^data:/i.test(trimmed)) {
    return trimmed;
  }
  return resolveApiUrl(trimmed);
}

const toCamelCaseKey = (key) => {
  if (typeof key !== "string" || !key.includes("_")) {
    return key;
  }
  return key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
};

function normalizeFeedbackSummary(raw, surveyConfig) {
  if (!raw || typeof raw !== "object") {
    return null;
  }

  const config = normalizeFeedbackSurveyInput(surveyConfig);
  const totalRaw = Number(raw.total_responses ?? raw.totalResponses ?? 0);
  const totalResponses = Number.isFinite(totalRaw) ? totalRaw : 0;
  const hasResponses = totalResponses > 0;
  const totalResponsesLabel =
    totalResponses === 1
      ? "1 risposta"
      : `${totalResponses.toLocaleString("it-IT")} risposte`;

  const questions = config.questions.map((question) => {
    const camelKey = toCamelCaseKey(question.id);
    const countsSource =
      (raw[question.id] && typeof raw[question.id] === "object"
        ? raw[question.id]
        : null) ??
      (raw[camelKey] && typeof raw[camelKey] === "object"
        ? raw[camelKey]
        : null);
    const counts =
      countsSource && typeof countsSource === "object" ? countsSource : {};

    const answers = question.answers.map((option) => {
      const resolved = Number(counts?.[option.value] ?? 0);
      const count = Number.isFinite(resolved) ? resolved : 0;
      const percent =
        hasResponses && totalResponses > 0
          ? Math.round((count / totalResponses) * 100)
          : 0;
      const clampedPercent = Math.min(100, Math.max(0, percent));
      const barPercent = hasResponses
        ? Math.max(clampedPercent, count > 0 ? 6 : 0)
        : 0;
      return {
        value: option.value,
        label: option.label,
        icon: option.icon || "",
        count,
        countLabel: count.toLocaleString("it-IT"),
        percent: clampedPercent,
        percentLabel: `${clampedPercent}%`,
        barWidth: `${barPercent}%`,
        hasCount: count > 0,
      };
    });

    const questionTotal = answers.reduce(
      (sum, answer) => sum + answer.count,
      0,
    );
    return {
      id: question.id,
      title: question.title,
      answers,
      totalCount: questionTotal,
      totalCountLabel:
        questionTotal === 1
          ? "1 risposta"
          : `${questionTotal.toLocaleString("it-IT")} risposte`,
      hasAnswers: answers.some((answer) => answer.count > 0),
    };
  });

  const suggestionsSource = Array.isArray(raw.suggestions)
    ? raw.suggestions
    : Array.isArray(raw.suggestion)
      ? raw.suggestion
      : [];
  const suggestions = suggestionsSource
    .map((value) => (typeof value === "string" ? value.trim() : ""))
    .filter(Boolean);

  const suggestionQuestion = {
    id: "suggestions",
    title: config.suggestionPrompt || DEFAULT_FEEDBACK_SURVEY.suggestionPrompt,
    suggestions,
    hasSuggestions: suggestions.length > 0,
  };

  const hasAnyData =
    hasResponses ||
    questions.some((question) => question.hasAnswers) ||
    suggestionQuestion.hasSuggestions;

  return {
    totalResponses,
    totalResponsesLabel,
    hasResponses,
    questions,
    suggestionQuestion,
    hasAnyData,
  };
}

function serializeSponsorPayload(sponsor) {
  return {
    name: sponsor.name.trim(),
    report_name: (sponsor.reportName || "").trim(),
    link_url: sponsor.linkUrl.trim(),
    position: sponsor.position,
    logo_data: sponsor.logoData,
    is_active: sponsor.isActive,
  };
}

function nextSponsorPosition() {
  const used = new Set(sponsors.value.map((item) => item.position));
  for (let index = 1; index <= maxSponsors; index += 1) {
    if (!used.has(index)) {
      return index;
    }
  }
  return Math.min(maxSponsors, sponsors.value.length + 1);
}

function sortedSponsors() {
  return [...sponsors.value].sort((a, b) => a.position - b.position);
}

function recomputeActiveSponsorSlider() {
  desiredActiveSponsorCount.value = Math.min(
    sponsorSliderMax.value,
    activeSponsorCount.value,
  );
}

function resetNewSponsorForm() {
  Object.assign(newSponsor, {
    name: "",
    reportName: "",
    linkUrl: "",
    logoData: "",
    isActive: true,
  });
}

function resetNewCouponForm() {
  Object.assign(newCoupon, createEmptyCouponDraft());
  couponError.value = "";
  couponSuccess.value = "";
}

async function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      resolve(typeof reader.result === "string" ? reader.result : "");
    };
    reader.onerror = () => {
      reject(reader.error || new Error("Impossibile leggere il file"));
    };
    reader.readAsDataURL(file);
  });
}

function syncCouponImageSource(targetCoupon) {
  if (!targetCoupon) {
    return;
  }
  targetCoupon.imagePreview = resolveCouponImageSource(targetCoupon.imageUrl);
}

async function handleCouponImageFileChange(event, targetCoupon) {
  const [file] = event?.target?.files || [];
  if (!file || !targetCoupon) {
    return;
  }
  couponError.value = "";
  couponSuccess.value = "";
  try {
    const dataUrl = await readFileAsDataUrl(file);
    if (dataUrl) {
      targetCoupon.imageUrl = dataUrl;
      targetCoupon.imagePreview = dataUrl;
    }
  } catch (error) {
    console.error("Errore caricamento immagine coupon", error);
    couponError.value = "Impossibile caricare l'immagine del coupon.";
  } finally {
    if (event?.target) {
      event.target.value = "";
    }
  }
}

function clearCouponImage(targetCoupon) {
  if (!targetCoupon) {
    return;
  }
  targetCoupon.imageUrl = "";
  targetCoupon.imagePreview = "";
}

async function handleSponsorLogoChange(event, targetSponsor) {
  const [file] = event?.target?.files || [];
  if (!file) {
    return;
  }
  globalError.value = "";
  try {
    const dataUrl = await readFileAsDataUrl(file);
    if (dataUrl) {
      targetSponsor.logoData = dataUrl;
    }
  } catch (error) {
    console.error("Errore caricamento logo sponsor", error);
    globalError.value = "Impossibile caricare il logo selezionato.";
  } finally {
    if (event?.target) {
      event.target.value = "";
    }
  }
}

async function handleNewSponsorLogoChange(event) {
  await handleSponsorLogoChange(event, newSponsor);
}

function buildEventLink(eventId) {
  const url = new URL(baseVoteUrl.value.toString());
  if (eventId) {
    url.searchParams.set("eventId", String(eventId));
  } else {
    url.searchParams.delete("eventId");
  }
  return url.toString();
}

function goToLottery() {
  const target = new URL(basePath.value || "/", window.location.origin);
  if (!target.pathname.endsWith("/")) {
    target.pathname = `${target.pathname}/`;
  }
  target.pathname = `${target.pathname.replace(/\/+$/, "")}/admin/lottery`;
  window.location.href = target.toString();
}

function teamOptionValue(team) {
  return `${team.name} (#${team.id})`;
}

function syncTeamInputsFromIds() {
  const homeTeam = teams.value.find((team) => team.id === newEvent.team1_id);
  const awayTeam = teams.value.find((team) => team.id === newEvent.team2_id);
  teamInputs.home = homeTeam ? teamOptionValue(homeTeam) : "";
  teamInputs.away = awayTeam ? teamOptionValue(awayTeam) : "";
}

function findTeamFromInput(value) {
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return undefined;
  }
  return (
    teams.value.find(
      (team) => teamOptionValue(team).toLowerCase() === normalized,
    ) ||
    teams.value.find((team) => team.name.trim().toLowerCase() === normalized)
  );
}

function handleTeamInput(position) {
  const key = position === "home" ? "team1_id" : "team2_id";
  const otherKey = position === "home" ? "team2_id" : "team1_id";
  const otherInputKey = position === "home" ? "away" : "home";
  const rawValue = teamInputs[position] || "";
  const matchedTeam = findTeamFromInput(rawValue);

  if (matchedTeam) {
    if (newEvent[otherKey] === matchedTeam.id) {
      newEvent[otherKey] = 0;
      teamInputs[otherInputKey] = "";
    }
    newEvent[key] = matchedTeam.id;
    teamInputs[position] = teamOptionValue(matchedTeam);
  } else {
    newEvent[key] = 0;
    teamInputs[position] = "";
  }
}

function addNewEventPrize() {
  newEventPrizes.value = [...newEventPrizes.value, { name: "", winSmsText: "" }];
}

function removeNewEventPrize(index) {
  if (newEventPrizes.value.length <= 1) {
    return;
  }
  const updated = newEventPrizes.value.filter((_, idx) => idx !== index);
  newEventPrizes.value = updated.length ? updated : [{ name: "", winSmsText: "" }];
}

function prizeDraftsFor(eventId) {
  const drafts = eventPrizeDrafts[eventId];
  if (!Array.isArray(drafts) || drafts.length === 0) {
    eventPrizeDrafts[eventId] = [
      { id: 0, name: "", position: 1, winSmsText: "", winner: null },
    ];
  }
  return eventPrizeDrafts[eventId];
}

function feedbackDraftFor(eventId) {
  if (!eventId) {
    return normalizeFeedbackSurveyInput();
  }
  const existing = eventFeedbackDrafts[eventId];
  if (existing && typeof existing === "object") {
    return existing;
  }
  const event = events.value.find((item) => item.id === eventId);
  const draftSource = event?.feedbackSurvey ?? event?.feedback_survey;
  const draft = normalizeFeedbackSurveyInput(draftSource);
  eventFeedbackDrafts[eventId] = draft;
  return draft;
}

function addPrizeDraft(eventId) {
  const drafts = prizeDraftsFor(eventId);
  const updated = drafts.slice();
  updated.push({ id: 0, name: "", position: updated.length + 1, winSmsText: "", winner: null });
  eventPrizeDrafts[eventId] = updated;
  eventPrizeErrors[eventId] = "";
}

function removePrizeDraft(eventId, index) {
  const drafts = prizeDraftsFor(eventId);
  if (drafts.length <= 1) {
    return;
  }
  const target = drafts[index];
  if (target && target.winner) {
    eventPrizeErrors[eventId] =
      "Impossibile rimuovere un premio già assegnato. Annulla il vincitore dalla lotteria prima di eliminarlo.";
    return;
  }
  const updated = drafts.filter((_, idx) => idx !== index);
  eventPrizeDrafts[eventId] = updated.length
    ? updated.map((item, positionIndex) => ({
        ...item,
        position: positionIndex + 1,
      }))
    : [{ id: 0, name: "", position: 1, winSmsText: "", winner: null }];
}

function isSavingPrizesFor(eventId) {
  return savingEventPrizes.value === eventId;
}

function prizeWinnerLabel(prize) {
  if (!prize || !prize.winner) {
    return "";
  }
  return prize.winner.ticketCode || "";
}

async function saveEventPrizes(event) {
  if (!event || !event.id || isSavingPrizesFor(event.id)) {
    return;
  }

  const drafts = prizeDraftsFor(event.id);
  const sanitized = drafts
    .map((prize, index) => ({
      id: Number(prize.id) || 0,
      name: (prize.name || "").trim(),
      position: index + 1,
      win_sms_text: (prize.winSmsText || '').trim(),
    }))
    .filter((prize) => prize.name);

  eventPrizeErrors[event.id] = "";
  eventFeedbackErrors[event.id] = "";
  const surveyDraft = feedbackDraftFor(event.id);

  const payload = {
    team1_id: event.team1_id,
    team2_id: event.team2_id,
    start_datetime: event.start_datetime,
    location: event.location,
    show_pre_vote_sponsors: Boolean(event.show_pre_vote_sponsors),
    show_pre_vote_bottom_sponsors: Boolean(
      event.show_pre_vote_bottom_sponsors,
    ),
    show_vote_counter: Boolean(event.show_vote_counter),
    show_reaction_test: Boolean(event.show_reaction_test),
    show_selfie: Boolean(event.show_selfie),
    show_vote_trend: Boolean(event.show_vote_trend),
    show_feedback_survey: Boolean(event.show_feedback_survey),
    show_branded_game: Boolean(event.show_branded_game),
    branded_game_config: event.show_branded_game
      ? JSON.stringify(event.brandedGameConfigDraft ?? defaultBrandedGameConfig())
      : "",
    feedback_survey: toApiSurveyPayload(surveyDraft),
    prizes: sanitized,
  };

  savingEventPrizes.value = event.id;
  try {
    await secureRequest(() =>
      apiClient.put(`/events/${event.id}`, payload, authHeaders.value),
    );
    await loadEvents();
  } catch (error) {
    if (error?.response?.status === 409) {
      eventPrizeErrors[event.id] =
        "Non puoi rimuovere un premio già assegnato. Annulla l'assegnazione dalla lotteria prima di modificarlo.";
    } else if (error?.response?.status === 400) {
      const message =
        "Controlla i nomi dei premi e le domande del sondaggio e riprova.";
      eventPrizeErrors[event.id] = message;
      eventFeedbackErrors[event.id] = message;
    } else if (error?.response?.status !== 401) {
      const message = "Impossibile salvare le impostazioni. Riprova più tardi.";
      eventPrizeErrors[event.id] = message;
      eventFeedbackErrors[event.id] = message;
    }
  } finally {
    savingEventPrizes.value = 0;
  }
}

async function saveEventFeedbackSurvey(event) {
  if (!event || !event.id || isSavingPrizesFor(event.id)) {
    return;
  }

  eventFeedbackErrors[event.id] = "";
  const surveyDraft = feedbackDraftFor(event.id);

  const payload = {
    team1_id: event.team1_id,
    team2_id: event.team2_id,
    start_datetime: event.start_datetime,
    location: event.location,
    show_pre_vote_sponsors: Boolean(event.show_pre_vote_sponsors),
    show_pre_vote_bottom_sponsors: Boolean(event.show_pre_vote_bottom_sponsors),
    show_vote_counter: Boolean(event.show_vote_counter),
    show_reaction_test: Boolean(event.show_reaction_test),
    show_selfie: Boolean(event.show_selfie),
    show_vote_trend: Boolean(event.show_vote_trend),
    show_feedback_survey: Boolean(event.show_feedback_survey),
    show_branded_game: Boolean(event.show_branded_game),
    branded_game_config: event.show_branded_game
      ? JSON.stringify(event.brandedGameConfigDraft ?? defaultBrandedGameConfig())
      : "",
    feedback_survey: toApiSurveyPayload(surveyDraft),
  };

  savingEventPrizes.value = event.id;
  try {
    await secureRequest(() =>
      apiClient.put(`/events/${event.id}`, payload, authHeaders.value),
    );
    await loadEvents();
  } catch (error) {
    if (error?.response?.status === 400) {
      eventFeedbackErrors[event.id] =
        "Controlla le domande del sondaggio e riprova.";
    } else if (error?.response?.status !== 401) {
      eventFeedbackErrors[event.id] =
        "Impossibile salvare il sondaggio. Riprova più tardi.";
    }
  } finally {
    savingEventPrizes.value = 0;
  }
}

function syncEventPrizeDrafts(eventList) {
  const ids = new Set(eventList.map((event) => event.id));
  Object.keys(eventPrizeDrafts).forEach((key) => {
    if (!ids.has(Number(key))) {
      delete eventPrizeDrafts[key];
    }
  });
  Object.keys(eventPrizeErrors).forEach((key) => {
    if (!ids.has(Number(key))) {
      delete eventPrizeErrors[key];
    }
  });
  eventList.forEach((event) => {
    const drafts =
      Array.isArray(event.prizes) && event.prizes.length
        ? event.prizes.map((prize, index) => ({
            id: prize.id,
            name: prize.name || "",
            position: prize.position || index + 1,
            winSmsText: prize.winSmsText || '',
            winner: prize.winner
              ? {
                  voteId: prize.winner.voteId,
                  ticketCode: prize.winner.ticketCode,
                  playerFirstName: prize.winner.playerFirstName,
                  playerLastName: prize.winner.playerLastName,
                }
              : null,
          }))
        : [{ id: 0, name: "", position: 1, winSmsText: "", winner: null }];
    eventPrizeDrafts[event.id] = drafts;
  });
}

function syncEventFeedbackDrafts(eventList) {
  const ids = new Set(eventList.map((event) => event.id));
  Object.keys(eventFeedbackDrafts).forEach((key) => {
    if (!ids.has(Number(key))) {
      delete eventFeedbackDrafts[key];
      delete eventFeedbackErrors[key];
    }
  });
  eventList.forEach((event) => {
    if (!event || !event.id) {
      return;
    }
    if (!eventFeedbackDrafts[event.id]) {
      eventFeedbackDrafts[event.id] = normalizeFeedbackSurveyInput(
        event.feedbackSurvey ?? event.feedback_survey,
      );
    }
  });
}

function eventLabel(event) {
  return `${resolveEventTeamName(event, "team1")} vs ${resolveEventTeamName(event, "team2")}`;
}

function formatMatchDateLabel(event) {
  const rawDate = event?.start_datetime ?? event?.startDatetime ?? event?.startDate;
  if (!rawDate) {
    return "";
  }
  const parsed = new Date(rawDate);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  return parsed.toLocaleDateString("it-IT", { day: "2-digit", month: "2-digit" });
}

function couponMatchLabel(event) {
  const baseLabel = eventLabel(event);
  const dateLabel = formatMatchDateLabel(event);
  return dateLabel ? `${baseLabel} ${dateLabel}` : baseLabel;
}

function resolveEventTeamName(event, teamKey) {
  const idKey = `${teamKey}_id`;
  const nameFromTeams = teamName(event?.[idKey]);
  if (nameFromTeams && nameFromTeams !== "—") {
    return nameFromTeams;
  }

  const fallbackKeys = [`${teamKey}_name`, `${teamKey}Name`];
  for (const key of fallbackKeys) {
    const value = event?.[key];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  return "—";
}

function teamName(id) {
  const team = teams.value.find((teamItem) => teamItem.id === id);
  return team ? team.name : "—";
}

function formatEventDate(value) {
  if (!value) {
    return "Data da definire";
  }
  const date = new Date(value);
  if (!Number.isNaN(date.valueOf())) {
    return date.toLocaleString("it-IT");
  }
  return value.replace("T", " ");
}

function formatWatchDuration(ms) {
  const value = Number(ms);
  if (!Number.isFinite(value) || value <= 0) {
    return "0 s";
  }
  if (value >= 60000) {
    const minutes = Math.floor(value / 60000);
    const seconds = Math.round((value % 60000) / 1000);
    if (seconds === 0) {
      return `${minutes}m`;
    }
    return `${minutes}m ${seconds}s`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)} s`;
  }
  return `${Math.round(value)} ms`;
}

function formatSecondsDuration(seconds) {
  const total = Math.max(0, Math.floor(Number(seconds) || 0));
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
}

function formatPercent(
  value,
  minimumFractionDigits = 1,
  maximumFractionDigits = 1,
) {
  if (!Number.isFinite(value)) {
    return "0,0";
  }
  return value.toLocaleString("it-IT", {
    minimumFractionDigits,
    maximumFractionDigits,
  });
}

async function login() {
  if (isLoggingIn.value) {
    return;
  }
  loginError.value = "";
  globalError.value = "";
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post("/admin/login", {
      username: loginForm.username,
      password: loginForm.password,
    });
    activeUsername.value = data.username;
    activeRole.value = data.role || "";
    localStorage.setItem("adminUsername", activeUsername.value);
    localStorage.setItem("adminRole", activeRole.value);
    selectDefaultSection();
    loginForm.username = "";
    loginForm.password = "";
    await loadAll();
  } catch (error) {
    if (error?.response?.status === 401) {
      loginError.value = "Credenziali non valide.";
    } else {
      loginError.value = "Impossibile completare l'accesso. Riprova.";
    }
  } finally {
    isLoggingIn.value = false;
  }
}

async function logout() {
  try { await apiClient.post("/admin/logout"); } catch (_) { /* ignora errori di rete */ }
  activeUsername.value = "";
  activeRole.value = "";
  isBarFeatureEnabled.value = true;
  localStorage.removeItem("adminUsername");
  localStorage.removeItem("adminRole");
  section.value = "events";
  clearCollections();
}

function handleUnauthorized() {
  logout();
  loginError.value = "Sessione scaduta. Effettua di nuovo il login.";
}

async function secureRequest(executor) {
  try {
    return await executor();
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else {
      globalError.value =
        "Si è verificato un errore imprevisto. Riprova più tardi.";
    }
    throw error;
  }
}

async function loadTeams() {
  const { data } = await secureRequest(() =>
    apiClient.get("/teams", authHeaders.value),
  );
  teams.value = data;
  ensureValidTeamSelection();
}

async function loadPlayers() {
  const { data } = await secureRequest(() =>
    apiClient.get("/players", authHeaders.value),
  );

  isBarFeatureEnabled.value = data?.bar_enabled !== false;
  const schemaCandidate = Number(data?.roster_schema);
  if (Number.isFinite(schemaCandidate)) {
    rosterSchema.value = validateRosterSchema(schemaCandidate);
  } else {
    rosterSchema.value = validateRosterSchema(rosterSchema.value);
  }

  const payload = Array.isArray(data?.players) ? data.players : data;

  const normalized = Array.isArray(payload)
    ? payload.map((item) => normalizePlayerResponse(item))
    : [];
  players.value = normalized;
  applyPlayersToSlots();
}

async function loadEvents() {
  const { data } = await secureRequest(() =>
    apiClient.get("/events", authHeaders.value),
  );
  const normalized = Array.isArray(data)
    ? data.map((event) => normalizeEventResponse(event)).filter(Boolean)
    : [];
  events.value = normalized;
  for (const event of normalized) {
    quizDraftFor(event.id);
    if (!quizQuestionsByEvent[event.id]) {
      loadQuizForEvent(event.id).catch(() => {});
    }
    if (!eventStoriesById[event.id]) {
      loadStoriesForEvent(event.id).catch(() => {});
    }
  }
  hasLoadedEventHistory.value = false;
}


function createDefaultQuizDraft(eventId = 0) {
  return {
    event_id: eventId,
    enabled: false,
    questions_per_session: 5,
    seconds_per_question: 8,
    base_reward: 3,
    completion_bonus: 5,
    streak_bonus: 1,
    active_from: "",
    active_to: "",
  };
}

function quizDraftFor(eventId) {
  if (!quizConfigsByEvent[eventId]) {
    quizConfigsByEvent[eventId] = createDefaultQuizDraft(eventId);
  }
  return quizConfigsByEvent[eventId];
}

function quizQuestionsFor(eventId) {
  if (!quizQuestionsByEvent[eventId]) {
    quizQuestionsByEvent[eventId] = [];
  }
  return quizQuestionsByEvent[eventId];
}

function addQuizQuestionDraft(eventId) {
  quizQuestionsFor(eventId).push({ id: 0, question_text: "", answers: ["", ""], correct_index: 0, order_index: quizQuestionsFor(eventId).length });
}

function addAnswerToQuestion(eventId, index) {
  const q = quizQuestionsFor(eventId)[index];
  if (q && q.answers.length < 4) q.answers.push("");
}

function removeAnswerFromQuestion(eventId, index) {
  const q = quizQuestionsFor(eventId)[index];
  if (q && q.answers.length > 2) q.answers.pop();
}

function storiesForEvent(eventId) {
  if (!eventStoriesById[eventId]) {
    eventStoriesById[eventId] = [];
  }
  return eventStoriesById[eventId];
}

function isStoriesLoading(eventId) {
  return eventStoriesLoading[eventId] === true;
}

function isStoriesSaving(eventId) {
  return eventStoriesSaving[eventId] === true;
}

function storyUploadKey(eventId, index) {
  return `${eventId}-${index}`;
}

function isStoryVideoUploading(eventId, index) {
  return eventStoriesUploading[storyUploadKey(eventId, index)] === true;
}

function triggerStoryVideoPicker(eventId, index) {
  const inputId = `story-video-${eventId}-${index}`;
  const input = typeof document !== 'undefined' ? document.getElementById(inputId) : null;
  if (input) {
    input.click();
  }
}

function extractErrorMessage(error) {
  const payload = error?.response?.data;
  if (typeof payload?.message === 'string' && payload.message.trim()) {
    return payload.message.trim();
  }
  if (typeof payload === 'string' && payload.trim()) {
    return payload.trim();
  }
  return '';
}

async function uploadStoryVideo(eventId, story, index, event) {
  const file = event?.target?.files?.[0];
  if (!file) {
    return;
  }
  const uploadKey = storyUploadKey(eventId, index);
  eventStoriesUploading[uploadKey] = true;
  globalError.value = '';
  try {
    const formData = new FormData();
    formData.append('video', file);
    const { data } = await secureRequest(() =>
      apiClient.post(`/admin/events/${eventId}/stories/upload-video`, formData, authHeaders.value),
    );
    story.video_url = String(data?.video_url || '').trim();
  } catch (error) {
    const reason = extractErrorMessage(error);
    globalError.value = reason
      ? `Upload video fallito: ${reason}`
      : 'Upload video fallito. Riprova.';
  } finally {
    eventStoriesUploading[uploadKey] = false;
    if (event?.target) {
      event.target.value = '';
    }
  }
}

async function loadStoriesForEvent(eventId) {
  eventStoriesLoading[eventId] = true;
  try {
    const { data } = await secureRequest(() =>
      apiClient.get(`/admin/events/${eventId}/stories`, authHeaders.value),
    );
    eventStoriesById[eventId] = Array.isArray(data)
      ? data.map((story) => ({
          id: Number(story.id) || 0,
          player_name: String(story.player_name || ''),
          thumbnail_url: String(story.thumbnail_url || ''),
          video_url: String(story.video_url || ''),
          title: String(story.title || ''),
          is_active: story.is_active !== false,
          order_index: Number(story.order_index) || 0,
        }))
      : [];
  } finally {
    eventStoriesLoading[eventId] = false;
  }
}

function addStoryDraft(eventId) {
  const rows = storiesForEvent(eventId);
  rows.push({
    id: 0,
    player_name: '',
    thumbnail_url: '',
    video_url: '',
    title: '',
    is_active: true,
    order_index: rows.length,
  });
}

function moveStory(eventId, index, direction) {
  const rows = [...storiesForEvent(eventId)];
  const target = index + direction;
  if (target < 0 || target >= rows.length) {
    return;
  }
  const [item] = rows.splice(index, 1);
  rows.splice(target, 0, item);
  rows.forEach((story, order) => {
    story.order_index = order;
  });
  eventStoriesById[eventId] = rows;
}

async function saveStory(eventId, story, index) {
  if (!story.thumbnail_url.trim() || !story.video_url.trim()) {
    globalError.value = 'Compila thumbnail e video URL per salvare la story.';
    return;
  }
  eventStoriesSaving[eventId] = true;
  try {
    const payload = {
      player_name: story.player_name.trim(),
      thumbnail_url: story.thumbnail_url.trim(),
      video_url: story.video_url.trim(),
      title: story.title.trim(),
      is_active: Boolean(story.is_active),
      order_index: Number.isFinite(Number(story.order_index)) ? Number(story.order_index) : index,
    };
    if (story.id) {
      await secureRequest(() =>
        apiClient.put(`/admin/events/${eventId}/stories/${story.id}`, payload, authHeaders.value),
      );
    } else {
      await secureRequest(() =>
        apiClient.post(`/admin/events/${eventId}/stories`, payload, authHeaders.value),
      );
    }
    await loadStoriesForEvent(eventId);
  } finally {
    eventStoriesSaving[eventId] = false;
  }
}

async function deleteStory(eventId, story, index) {
  if (!story?.id) {
    eventStoriesById[eventId] = storiesForEvent(eventId).filter((_, idx) => idx !== index);
    return;
  }
  eventStoriesSaving[eventId] = true;
  try {
    await secureRequest(() =>
      apiClient.delete(`/admin/events/${eventId}/stories/${story.id}`, authHeaders.value),
    );
    await loadStoriesForEvent(eventId);
  } finally {
    eventStoriesSaving[eventId] = false;
  }
}

async function loadQuizForEvent(eventId) {
  const { data } = await secureRequest(() => apiClient.get(`/admin/events/${eventId}/quiz`, authHeaders.value));
  quizConfigsByEvent[eventId] = { ...createDefaultQuizDraft(eventId), ...(data?.config || {}) };
  const questions = await secureRequest(() => apiClient.get(`/admin/events/${eventId}/quiz/questions`, authHeaders.value));
  quizQuestionsByEvent[eventId] = Array.isArray(questions?.data) ? questions.data.map((q) => ({ ...q, answers: Array.isArray(q.answers) ? q.answers : ["", ""] })) : [];
}

async function saveQuizConfig(eventId) {
  await secureRequest(() => apiClient.put(`/admin/events/${eventId}/quiz`, quizDraftFor(eventId), authHeaders.value));
}

async function saveQuizQuestion(eventId, q) {
  const payload = {
    question_text: q.question_text,
    answers: q.answers,
    correct_index: Number(q.correct_index) || 0,
    order_index: Number(q.order_index) || 0,
  };
  if (q.id) {
    await secureRequest(() => apiClient.put(`/admin/events/${eventId}/quiz/questions/${q.id}`, payload, authHeaders.value));
  } else {
    await secureRequest(() => apiClient.post(`/admin/events/${eventId}/quiz/questions`, payload, authHeaders.value));
  }
  await loadQuizForEvent(eventId);
}

async function deleteQuizQuestion(eventId, q) {
  if (!q?.id) {
    quizQuestionsByEvent[eventId] = quizQuestionsFor(eventId).filter((item) => item !== q);
    return;
  }
  await secureRequest(() => apiClient.delete(`/admin/events/${eventId}/quiz/questions/${q.id}`, authHeaders.value));
  await loadQuizForEvent(eventId);
}
async function loadAdmins() {
  const { data } = await secureRequest(() =>
    apiClient.get("/admins", authHeaders.value),
  );
  admins.value = data;
}

function normalizePartnerResponse(item) {
  if (!item || typeof item !== "object") {
    return null;
  }

  const createdAt = item.created_at ? new Date(item.created_at) : null;

  return {
    id: Number(item.id) || 0,
    username: item.username || "",
    displayName: item.username || "",
    createdAtLabel: createdAt ? createdAt.toLocaleString("it-IT") : "",
    newPassword: "",
    isUpdating: false,
    isDeleting: false,
  };
}

async function loadPartners() {
  const { data } = await secureRequest(() =>
    apiClient.get("/admin/partners", authHeaders.value),
  );
  const normalized = Array.isArray(data)
    ? data
        .map((item) => normalizePartnerResponse(item))
        .filter((item) => item && item.id)
    : [];
  partners.value = normalized;
}

async function loadSponsors() {
  const { data } = await secureRequest(() =>
    apiClient.get("/admin/sponsors", authHeaders.value),
  );
  const normalized = Array.isArray(data)
    ? data
        .map((item) => normalizeSponsorResponse(item))
        .filter((item) => item && item.id)
        .sort((a, b) => a.position - b.position)
    : [];
  sponsors.value = normalized;
  recomputeActiveSponsorSlider();
}

async function loadCoupons() {
  const { data } = await secureRequest(() =>
    apiClient.get("/admin/coupons", authHeaders.value),
  );
  const normalized = Array.isArray(data)
    ? data
        .map((item) => toEditableCoupon(item))
        .filter((item) => item && item.id)
    : [];
  coupons.value = normalized;
}

async function loadEventSelfies(eventId) {
  if (!eventId) {
    eventSelfies.value = [];
    return;
  }
  isLoadingSelfies.value = true;
  selfieLoadError.value = "";
  selfieModerationMessage.value = "";
  try {
    const { data } = await secureRequest(() =>
      apiClient.get(`/admin/events/${eventId}/selfies`, authHeaders.value),
    );
    const normalized = Array.isArray(data)
      ? data
          .map((item) => normalizeSelfieResponse(item))
          .filter((item) => item && item.id)
      : [];
    eventSelfies.value = normalized;
  } catch (error) {
    if (error?.response?.status !== 401) {
      selfieLoadError.value =
        "Impossibile caricare i selfie per questo evento.";
    }
    eventSelfies.value = [];
  } finally {
    isLoadingSelfies.value = false;
  }
}

function setSelfieBusy(id, busy) {
  if (!id) {
    return;
  }
  if (busy) {
    selfieBusyState[id] = true;
  } else {
    delete selfieBusyState[id];
  }
}

function isSelfieBusy(id) {
  return Boolean(selfieBusyState[id]);
}

function setHistoryReportBusy(id, busy) {
  if (!id) {
    return;
  }
  if (busy) {
    historyReportDownloadState[id] = true;
  } else {
    delete historyReportDownloadState[id];
  }
}

function isDownloadingHistoryReport(id) {
  return Boolean(id && historyReportDownloadState[id]);
}

function setHistoryAiBusy(id, busy) {
  if (!id) return;
  if (busy) historyAiGenerateState[id] = true;
  else delete historyAiGenerateState[id];
}

function isGeneratingHistoryAiReport(id) {
  return Boolean(id && historyAiGenerateState[id]);
}

function selfieStatusLabel(selfie) {
  if (!selfie) {
    return "";
  }
  if (!selfie.approved) {
    return "In attesa di approvazione";
  }
  if (selfie.show_on_screen) {
    return "Approvato per il maxischermo";
  }
  return "Approvato";
}

function formatSelfieDate(value) {
  if (!value) {
    return "";
  }
  try {
    return selfieDateFormatter.format(new Date(value));
  } catch (error) {
    return value;
  }
}

function formatSelfieFileSize(value) {
  const size = Number(value);
  if (!Number.isFinite(size) || size <= 0) {
    return "";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let unitIndex = 0;
  let display = size;
  while (display >= 1024 && unitIndex < units.length - 1) {
    display /= 1024;
    unitIndex += 1;
  }
  const formatter = new Intl.NumberFormat("it-IT", {
    minimumFractionDigits: display < 10 && unitIndex > 0 ? 1 : 0,
    maximumFractionDigits: 1,
  });
  return `${formatter.format(display)} ${units[unitIndex]}`;
}

function ensureSelfieSelection() {
  const available = availableEvents.value;
  if (!available.length) {
    selectedSelfieEventId.value = 0;
    return;
  }
  const current = selectedSelfieEventId.value;
  const stillValid = available.some((event) => event.id === current);
  if (stillValid) {
    return;
  }
  const active = available.find((event) => event.is_active);
  selectedSelfieEventId.value = active ? active.id : available[0].id;
}

async function deleteSelfie(selfie) {
  if (!selfie?.id) {
    return;
  }
  selfieLoadError.value = "";
  selfieModerationMessage.value = "";
  setSelfieBusy(selfie.id, true);
  try {
    await secureRequest(() =>
      apiClient.delete(`/admin/selfies/${selfie.id}`, authHeaders.value),
    );
    eventSelfies.value = eventSelfies.value.filter(
      (item) => item.id !== selfie.id,
    );
    selfieModerationMessage.value = "Selfie eliminato.";
  } catch (error) {
    if (error?.response?.status === 404) {
      eventSelfies.value = eventSelfies.value.filter(
        (item) => item.id !== selfie.id,
      );
      selfieLoadError.value = "Il selfie selezionato non è più disponibile.";
    } else if (error?.response?.status !== 401) {
      selfieLoadError.value =
        "Impossibile eliminare il selfie. Riprova più tardi.";
    }
  } finally {
    setSelfieBusy(selfie.id, false);
  }
}

function parseHistoryDate(value) {
  if (typeof value !== "string" || !value.trim()) {
    return null;
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

function formatHistoryDate(value) {
  const parsed = parseHistoryDate(value);
  if (!parsed) {
    return "Data non disponibile";
  }
  try {
    return historyDateFormatter.format(parsed);
  } catch (error) {
    try {
      return parsed.toLocaleString("it-IT");
    } catch (innerError) {
      return parsed.toString();
    }
  }
}

function formatHistoryTime(date) {
  if (!(date instanceof Date) || Number.isNaN(date.valueOf())) {
    return "";
  }
  try {
    return historyTimeFormatter.format(date);
  } catch (error) {
    try {
      return date.toLocaleTimeString("it-IT", {
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch (innerError) {
      return "";
    }
  }
}

function buildHistoryTimelineChart(buckets, windowLabels = null) {
  if (!Array.isArray(buckets) || !buckets.length) {
    return {
      points: [],
      startLabel: windowLabels?.start || "",
      endLabel: windowLabels?.end || "",
    };
  }

  let cumulative = 0;
  const points = [];
  let computedStart = "";
  let computedEnd = "";

  buckets.forEach((bucket) => {
    const votes = Number(bucket?.votes ?? 0) || 0;
    cumulative += votes;

    const reference = bucket?.end || bucket?.start || "";
    const date = reference ? parseHistoryDate(reference) : null;
    if (!date) {
      return;
    }

    const label =
      bucket?.rangeLabel || bucket?.endLabel || bucket?.startLabel || "";
    if (!computedStart) {
      computedStart = bucket?.startLabel || label || "";
    }
    if (bucket?.endLabel || label) {
      computedEnd = bucket?.endLabel || label || computedEnd;
    }

    const votesLabel = votes.toLocaleString("it-IT");
    const cumulativeLabel = cumulative.toLocaleString("it-IT");
    const tooltipParts = [];
    if (label) {
      tooltipParts.push(label);
    }
    tooltipParts.push(`${votesLabel} voti nel periodo`);
    tooltipParts.push(`${cumulativeLabel} voti totali`);

    points.push({
      date,
      value: cumulative,
      label: label || formatHistoryTime(date),
      tooltip: tooltipParts.join(" · "),
    });
  });

  const startLabel =
    windowLabels?.start || computedStart || buckets[0]?.rangeLabel || "";
  const endLabel =
    windowLabels?.end ||
    computedEnd ||
    buckets[buckets.length - 1]?.rangeLabel ||
    "";

  return {
    points,
    startLabel,
    endLabel,
  };
}

function normalizeHistoryEngagement(raw) {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const totalSeconds = Number(
    raw?.total_duration_seconds ?? raw?.totalDurationSeconds ?? 0,
  );
  const averageSeconds = Number(
    raw?.average_duration_seconds ?? raw?.averageDurationSeconds ?? 0,
  );
  const totalUsers = Number(raw?.total_users ?? raw?.totalUsers ?? 0);
  const voteTrendOpens = Number(
    raw?.vote_trend_opens ?? raw?.voteTrendOpens ?? 0,
  );
  const selfieOpens = Number(raw?.selfie_opens ?? raw?.selfieOpens ?? 0);
  const selfieAbandons = Number(
    raw?.selfie_abandons ?? raw?.selfieAbandons ?? 0,
  );
  const reactionOpens = Number(
    raw?.reaction_opens ?? raw?.reactionOpens ?? 0,
  );
  const reactionAbandons = Number(
    raw?.reaction_abandons ?? raw?.reactionAbandons ?? 0,
  );
  const experienceOpens = Number(
    raw?.experience_opens ?? raw?.experienceOpens ?? 0,
  );
  const experienceAbandons = Number(
    raw?.experience_abandons ?? raw?.experienceAbandons ?? 0,
  );
  const photoEditOpens = Number(
    raw?.photo_edit_opens ?? raw?.photoEditOpens ?? 0,
  );
  const voteEditOpens = Number(
    raw?.vote_edit_opens ?? raw?.voteEditOpens ?? 0,
  );
  const voteEditAbandons = Number(
    raw?.vote_edit_abandons ?? raw?.voteEditAbandons ?? 0,
  );
  const voteEditCompletions = Number(
    raw?.vote_edit_completions ?? raw?.voteEditCompletions ?? 0,
  );

  const normalized = {
    totalSeconds: Number.isFinite(totalSeconds) ? totalSeconds : 0,
    averageSeconds: Number.isFinite(averageSeconds) ? averageSeconds : 0,
    users: Number.isFinite(totalUsers) ? totalUsers : 0,
    voteTrendOpens: Number.isFinite(voteTrendOpens) ? voteTrendOpens : 0,
    selfieOpens: Number.isFinite(selfieOpens) ? selfieOpens : 0,
    selfieAbandons: Number.isFinite(selfieAbandons)
      ? selfieAbandons
      : 0,
    reactionOpens: Number.isFinite(reactionOpens) ? reactionOpens : 0,
    reactionAbandons: Number.isFinite(reactionAbandons)
      ? reactionAbandons
      : 0,
    experienceOpens: Number.isFinite(experienceOpens)
      ? experienceOpens
      : 0,
    experienceAbandons: Number.isFinite(experienceAbandons)
      ? experienceAbandons
      : 0,
    photoEditOpens: Number.isFinite(photoEditOpens) ? photoEditOpens : 0,
    voteEditOpens: Number.isFinite(voteEditOpens) ? voteEditOpens : 0,
    voteEditAbandons: Number.isFinite(voteEditAbandons)
      ? voteEditAbandons
      : 0,
    voteEditCompletions: Number.isFinite(voteEditCompletions)
      ? voteEditCompletions
      : 0,
  };

  return {
    ...normalized,
    hasData:
      normalized.totalSeconds > 0 ||
      normalized.averageSeconds > 0 ||
      normalized.users > 0,
    totalLabel: formatSecondsDuration(normalized.totalSeconds),
    averageLabel: formatSecondsDuration(normalized.averageSeconds),
    usersLabel: normalized.users.toLocaleString("it-IT"),
    voteTrendOpensLabel: normalized.voteTrendOpens.toLocaleString("it-IT"),
    selfieOpensLabel: normalized.selfieOpens.toLocaleString("it-IT"),
    selfieAbandonsLabel: normalized.selfieAbandons.toLocaleString("it-IT"),
    reactionOpensLabel: normalized.reactionOpens.toLocaleString("it-IT"),
    reactionAbandonsLabel:
      normalized.reactionAbandons.toLocaleString("it-IT"),
    experienceOpensLabel:
      normalized.experienceOpens.toLocaleString("it-IT"),
    experienceAbandonsLabel:
      normalized.experienceAbandons.toLocaleString("it-IT"),
    photoEditOpensLabel: normalized.photoEditOpens.toLocaleString("it-IT"),
    voteEditOpensLabel: normalized.voteEditOpens.toLocaleString("it-IT"),
    voteEditAbandonsLabel:
      normalized.voteEditAbandons.toLocaleString("it-IT"),
    voteEditCompletionsLabel:
      normalized.voteEditCompletions.toLocaleString("it-IT"),
  };
}

function formatTrackingValueList(values, limit = 4) {
  if (!Array.isArray(values)) {
    return "";
  }
  const normalized = values
    .map((value) => (typeof value === "string" ? value.trim() : ""))
    .filter(Boolean);
  if (!normalized.length) {
    return "";
  }
  const visible = normalized.slice(0, limit);
  const hiddenCount = normalized.length - visible.length;
  return hiddenCount > 0
    ? `${visible.join(", ")} (+${hiddenCount})`
    : visible.join(", ");
}

function normalizeHistoryTrackingEvents(raw) {
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .map((eventItem) => {
      if (!eventItem || typeof eventItem !== "object") {
        return null;
      }
      const name =
        typeof eventItem?.name === "string" ? eventItem.name.trim() : "";
      const count = Number(eventItem?.count ?? 0) || 0;
      if (!name && count <= 0) {
        return null;
      }

      const uniqueSessions =
        Number(eventItem?.unique_sessions ?? eventItem?.uniqueSessions ?? 0) ||
        0;
      const uniqueDevices =
        Number(eventItem?.unique_devices ?? eventItem?.uniqueDevices ?? 0) || 0;
      const uniqueFans =
        Number(eventItem?.unique_fans ?? eventItem?.uniqueFans ?? 0) || 0;

      const firstOccurredAt =
        typeof (eventItem?.first_occurred_at ?? eventItem?.firstOccurredAt) ===
        "string"
          ? (eventItem?.first_occurred_at ?? eventItem?.firstOccurredAt)
          : "";
      const lastOccurredAt =
        typeof (eventItem?.last_occurred_at ?? eventItem?.lastOccurredAt) ===
        "string"
          ? (eventItem?.last_occurred_at ?? eventItem?.lastOccurredAt)
          : "";

      const formatDateTime = (value) => {
        const parsed = parseHistoryDate(value);
        if (!parsed) {
          return "";
        }
        try {
          return analyticsTimeFormatter.format(parsed);
        } catch (error) {
          return value;
        }
      };

      const fromLabel = formatDateTime(firstOccurredAt);
      const toLabel = formatDateTime(lastOccurredAt);
      const rangeLabel =
        fromLabel && toLabel
          ? `${fromLabel} → ${toLabel}`
          : toLabel || fromLabel || "";

      const details = [
        {
          label: "Pagine",
          value: formatTrackingValueList(eventItem?.pages),
        },
        {
          label: "Sezioni",
          value: formatTrackingValueList(eventItem?.sections),
        },
        {
          label: "Sorgenti",
          value: formatTrackingValueList(eventItem?.sources),
        },
        {
          label: "Domini",
          value: formatTrackingValueList(eventItem?.domains),
        },
        {
          label: "Login",
          value: formatTrackingValueList(eventItem?.login_states ?? eventItem?.loginStates),
        },
        {
          label: "Profilo",
          value: formatTrackingValueList(eventItem?.profile_states ?? eventItem?.profileStates),
        },
      ].filter((detail) => detail.value);

      const metadataSamples = Array.isArray(
        eventItem?.metadata_samples ?? eventItem?.metadataSamples,
      )
        ? (eventItem?.metadata_samples ?? eventItem?.metadataSamples)
            .map((sample) =>
              typeof sample === "string" ? sample.trim() : "",
            )
            .filter(Boolean)
        : [];

      return {
        name: name || "tracking_event",
        nameLabel: name || "Evento tracking",
        count,
        countLabel: `${count.toLocaleString("it-IT")} occorrenze`,
        uniqueSessions,
        uniqueSessionsLabel: uniqueSessions.toLocaleString("it-IT"),
        uniqueDevices,
        uniqueDevicesLabel: uniqueDevices.toLocaleString("it-IT"),
        uniqueFans,
        uniqueFansLabel: uniqueFans.toLocaleString("it-IT"),
        rangeLabel,
        details,
        metadataSamples,
      };
    })
    .filter(Boolean)
    .sort((a, b) => {
      if (b.count !== a.count) {
        return b.count - a.count;
      }
      return a.nameLabel.localeCompare(b.nameLabel, "it");
    });
}

function normalizeHistoryTrackingAnalytics(raw) {
  const data = raw && typeof raw === "object" ? raw : {};
  const asNumber = (value) => Number(value ?? 0) || 0;
  const asRate = (value) => {
    const parsed = Number(value ?? 0);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return 0;
    }
    return parsed;
  };
  const formatRateLabel = (value) => `${formatPercent(asRate(value) * 100)}%`;
  const funnelRaw = Array.isArray(data?.funnels) ? data.funnels : [];
  const funnels = funnelRaw
    .map((funnel) => ({
      name: typeof funnel?.name === "string" ? funnel.name.trim() : "",
      steps: Array.isArray(funnel?.steps)
        ? funnel.steps.map((step) => ({
            key: typeof step?.key === "string" ? step.key.trim() : "",
            label:
              typeof step?.label === "string" && step.label.trim()
                ? step.label.trim()
                : "Step",
            count: asNumber(step?.count),
            countLabel: asNumber(step?.count).toLocaleString("it-IT"),
          }))
        : [],
    }))
    .filter((funnel) => funnel.name && funnel.steps.length);

  const kpi = data?.kpi && typeof data.kpi === "object" ? data.kpi : {};
  const topEventNames = Array.isArray(data?.top_event_names ?? data?.topEventNames)
    ? (data?.top_event_names ?? data?.topEventNames)
        .map((item) => ({
          name: typeof item?.name === "string" ? item.name.trim() : "",
          count: asNumber(item?.count),
        }))
        .filter((item) => item.name)
        .sort((a, b) => b.count - a.count)
    : [];
  const byDomain = data?.by_event_domain ?? data?.byEventDomain ?? {};
  const domainBreakdown = Object.entries(byDomain)
    .map(([domain, value]) => ({
      domain: String(domain || "unknown"),
      count: asNumber(value),
    }))
    .sort((a, b) => b.count - a.count);

  return {
    uniqueSessions: asNumber(data?.unique_sessions ?? data?.uniqueSessions),
    totalEvents: asNumber(data?.total_events ?? data?.totalEvents),
    voteSubmittedCount: asNumber(
      data?.vote_submitted_count ?? data?.voteSubmittedCount,
    ),
    feedbackSubmittedCount: asNumber(
      data?.feedback_submitted_count ?? data?.feedbackSubmittedCount,
    ),
    sponsorClickedCount: asNumber(
      data?.sponsor_clicked_count ?? data?.sponsorClickedCount,
    ),
    barOrderCompletedCount: asNumber(
      data?.bar_order_completed_count ?? data?.barOrderCompletedCount,
    ),
    avgEventsPerSession: asRate(
      data?.avg_events_per_session ?? data?.avgEventsPerSession,
    ),
    voteConversionRate: asRate(
      data?.vote_conversion_rate ?? data?.voteConversionRate,
    ),
    voteCompletionRate: asRate(
      data?.vote_completion_rate ?? data?.voteCompletionRate,
    ),
    feedbackCompletionRate: asRate(
      data?.feedback_completion_rate ?? data?.feedbackCompletionRate,
    ),
    barOrderCompletionRate: asRate(
      data?.bar_order_completion_rate ?? data?.barOrderCompletionRate,
    ),
    firstEventAt:
      typeof (data?.first_event_at ?? data?.firstEventAt) === "string"
        ? (data?.first_event_at ?? data?.firstEventAt)
        : "",
    lastEventAt:
      typeof (data?.last_event_at ?? data?.lastEventAt) === "string"
        ? (data?.last_event_at ?? data?.lastEventAt)
        : "",
    peakActivityMinute:
      typeof (data?.peak_activity_minute ?? data?.peakActivityMinute) ===
      "string"
        ? (data?.peak_activity_minute ?? data?.peakActivityMinute)
        : "",
    segmentBreakdown: {
      guest: {
        submittedVotes: asNumber(
          data?.segment_breakdown?.guest?.submitted_votes ??
            data?.segmentBreakdown?.guest?.submittedVotes,
        ),
        feedbackSubmitted: asNumber(
          data?.segment_breakdown?.guest?.feedback_submitted ??
            data?.segmentBreakdown?.guest?.feedbackSubmitted,
        ),
        sponsorClicks: asNumber(
          data?.segment_breakdown?.guest?.sponsor_clicks ??
            data?.segmentBreakdown?.guest?.sponsorClicks,
        ),
      },
      registered: {
        submittedVotes: asNumber(
          data?.segment_breakdown?.registered?.submitted_votes ??
            data?.segmentBreakdown?.registered?.submittedVotes,
        ),
        feedbackSubmitted: asNumber(
          data?.segment_breakdown?.registered?.feedback_submitted ??
            data?.segmentBreakdown?.registered?.feedbackSubmitted,
        ),
        sponsorClicks: asNumber(
          data?.segment_breakdown?.registered?.sponsor_clicks ??
            data?.segmentBreakdown?.registered?.sponsorClicks,
        ),
      },
    },
    funnels,
    topEventNames,
    domainBreakdown,
    labels: {
      uniqueSessions: asNumber(
        kpi?.unique_sessions ?? kpi?.uniqueSessions ?? data?.unique_sessions,
      ).toLocaleString("it-IT"),
      totalEvents: asNumber(
        kpi?.total_events ?? kpi?.totalEvents ?? data?.total_events,
      ).toLocaleString("it-IT"),
      votesSubmitted: asNumber(
        kpi?.votes_submitted ?? kpi?.votesSubmitted ?? data?.vote_submitted_count,
      ).toLocaleString("it-IT"),
      voteConversionRate: formatRateLabel(
        kpi?.vote_conversion_rate ?? kpi?.voteConversionRate ?? data?.vote_conversion_rate,
      ),
      feedbackSubmitted: asNumber(
        kpi?.feedback_submitted ?? kpi?.feedbackSubmitted ?? data?.feedback_submitted_count,
      ).toLocaleString("it-IT"),
      sponsorClicks: asNumber(
        kpi?.sponsor_clicks ?? kpi?.sponsorClicks ?? data?.sponsor_clicked_count,
      ).toLocaleString("it-IT"),
      barOrdersCompleted: asNumber(
        kpi?.bar_orders_completed ??
          kpi?.barOrdersCompleted ??
          data?.bar_order_completed_count,
      ).toLocaleString("it-IT"),
      avgEventsPerSession: asRate(
        kpi?.avg_events_per_session ??
          kpi?.avgEventsPerSession ??
          data?.avg_events_per_session,
      ).toFixed(2),
      voteCompletionRate: formatRateLabel(data?.vote_completion_rate),
      feedbackCompletionRate: formatRateLabel(data?.feedback_completion_rate),
      barOrderCompletionRate: formatRateLabel(data?.bar_order_completion_rate),
    },
  };
}

function normalizeHistoryEntry(item) {
  const id = Number(item?.id) || 0;
  const homeTeam =
    typeof item?.home_team === "string" ? item.home_team.trim() : "";
  const awayTeam =
    typeof item?.away_team === "string" ? item.away_team.trim() : "";
  const rawTitle = typeof item?.title === "string" ? item.title.trim() : "";
  const fallbackTitle =
    [homeTeam, awayTeam].filter(Boolean).join(" - ") ||
    (id ? `Evento #${id}` : "Evento");
  const startDatetime =
    typeof item?.start_datetime === "string" ? item.start_datetime : "";
  const location =
    typeof item?.location === "string" ? item.location.trim() : "";
  const totalVotes = Number(item?.total_votes ?? item?.totalVotes ?? 0) || 0;

  const sponsorClicks = Array.isArray(item?.sponsor_clicks)
    ? item.sponsor_clicks
        .map((entry) => ({
          id: Number(entry?.sponsor_id) || 0,
          name:
            typeof entry?.name === "string" && entry.name.trim()
              ? entry.name.trim()
              : "Sponsor",
          reportName:
            typeof entry?.report_name === "string" && entry.report_name.trim()
              ? entry.report_name.trim()
              : typeof entry?.reportName === "string" && entry.reportName.trim()
                ? entry.reportName.trim()
                : "",
          link:
            typeof entry?.link_url === "string" ? entry.link_url.trim() : "",
          clicks: Number(entry?.clicks ?? 0) || 0,
        }))
        .sort((a, b) => {
          if (b.clicks !== a.clicks) {
            return b.clicks - a.clicks;
          }
          const labelA = a.reportName || a.name;
          const labelB = b.reportName || b.name;
          return labelA.localeCompare(labelB, "it");
        })
    : [];

  const sponsorClicksTotalRaw = Number(
    item?.sponsor_clicks_total ?? item?.sponsorClicksTotal ?? 0,
  );
  const sponsorClicksTotal = Number.isFinite(sponsorClicksTotalRaw)
    ? sponsorClicksTotalRaw
    : sponsorClicks.reduce(
        (sum, sponsor) => sum + (Number(sponsor.clicks) || 0),
        0,
      );
  const sponsorClicksTotalLabel = Number.isFinite(sponsorClicksTotal)
    ? sponsorClicksTotal.toLocaleString("it-IT")
    : "0";

  const sponsorAnalyticsRaw =
    item?.sponsor_analytics ?? item?.sponsorAnalytics ?? null;
  const sponsorAnalyticsData =
    normalizeSponsorAnalyticsResponse(sponsorAnalyticsRaw);
  const sponsorAnalyticsDisplay = {
    totalUsers: sponsorAnalyticsData.totalUsers,
    totalUsersLabel: sponsorAnalyticsData.totalUsers.toLocaleString("it-IT"),
    seenUsers: sponsorAnalyticsData.seenUsers,
    seenUsersLabel: sponsorAnalyticsData.seenUsers.toLocaleString("it-IT"),
    seenRateLabel: `${formatPercent(sponsorAnalyticsData.seenRate)}%`,
    watchedUsers: sponsorAnalyticsData.watchedUsers,
    watchedUsersLabel:
      sponsorAnalyticsData.watchedUsers.toLocaleString("it-IT"),
    averageWatchTimeLabel: formatWatchDuration(
      sponsorAnalyticsData.averageWatchTimeMs,
    ),
    totalWatchTimeLabel: formatWatchDuration(
      sponsorAnalyticsData.totalWatchTimeMs,
    ),
    totalClicks: sponsorAnalyticsData.totalClicks,
    totalClicksLabel: sponsorAnalyticsData.totalClicks.toLocaleString("it-IT"),
    clickRateLabel: `${formatPercent(sponsorAnalyticsData.clickRate)}%`,
    uniqueClickersLabel:
      sponsorAnalyticsData.uniqueClickers.toLocaleString("it-IT"),
    topSponsorName:
      sponsorAnalyticsData.topSponsor?.reportName?.trim() ||
      sponsorAnalyticsData.topSponsor?.name?.trim() ||
      "Nessuno",
    topSponsorViewsLabel: sponsorAnalyticsData.topSponsor
      ? sponsorAnalyticsData.topSponsor.views.toLocaleString("it-IT")
      : "0",
  };

  const totalVisitors = Number(sponsorAnalyticsData.totalUsers) || 0;
  const uniqueVisitors = Number(sponsorAnalyticsData.seenUsers) || 0;
  const totalVisitorsLabel = totalVisitors.toLocaleString("it-IT");
  const uniqueVisitorsLabel = uniqueVisitors.toLocaleString("it-IT");

  const engagement = normalizeHistoryEngagement(
    item?.engagement ?? item?.engagementStats,
  );

  const sponsorAnalyticsTimelineRaw = Array.isArray(
    sponsorAnalyticsData.timeline,
  )
    ? sponsorAnalyticsData.timeline
    : [];
  const sponsorAnalyticsTimeline = sponsorAnalyticsTimelineRaw.map((point) => {
    const timestamp =
      typeof point?.timestamp === "string" ? point.timestamp : "";
    const seen = Number(point?.seen ?? 0) || 0;
    const watched = Number(point?.watched ?? 0) || 0;
    const clicks = Number(point?.clicks ?? 0) || 0;
    let label = timestamp || "";
    if (timestamp) {
      const parsed = new Date(timestamp);
      if (!Number.isNaN(parsed.valueOf())) {
        label = historyTimeFormatter.format(parsed);
      }
    }
    return {
      timestamp,
      label: label || "—",
      seen,
      watched,
      clicks,
    };
  });

  const sponsorAnalyticsHasData = Boolean(
    sponsorAnalyticsData.totalUsers ||
      sponsorAnalyticsData.totalClicks ||
      sponsorAnalyticsTimeline.length ||
      (sponsorAnalyticsData.topSponsor &&
        sponsorAnalyticsData.topSponsor.views),
  );

  const prizesRaw = Array.isArray(item?.prizes) ? item.prizes : [];
  const normalizedPrizes = prizesRaw
    .map((prize, index) => {
      if (!prize || typeof prize !== "object") {
        return null;
      }
      const id = Number(prize?.id ?? prize?.ID) || 0;
      const position = Number(prize?.position ?? prize?.Position) || index + 1;
      const rawName =
        typeof (prize?.name ?? prize?.Name) === "string"
          ? (prize?.name ?? prize?.Name).trim()
          : "";
      const name = rawName || `Premio ${position || index + 1}`;
      const winnerCodeRaw =
        typeof (prize?.winner_ticket_code ?? prize?.winnerTicketCode) ===
        "string"
          ? (prize?.winner_ticket_code ?? prize?.winnerTicketCode)
          : "";
      const winnerTicketCode = winnerCodeRaw.trim().toUpperCase();
      return {
        id,
        position,
        name,
        winnerTicketCode,
        hasWinner: Boolean(winnerTicketCode),
      };
    })
    .filter(Boolean)
    .sort((a, b) => {
      if (a.position === b.position) {
        return a.id - b.id;
      }
      return a.position - b.position;
    });

  let hasPrizeDraw = Boolean(item?.has_prize_draw ?? item?.hasPrizeDraw);
  if (!hasPrizeDraw) {
    hasPrizeDraw = normalizedPrizes.some((prize) => prize.hasWinner);
  }

  const timelineRaw = Array.isArray(item?.timeline) ? item.timeline : [];
  const timelineBuckets = timelineRaw
    .map((bucket) => {
      const start = typeof bucket?.start === "string" ? bucket.start : "";
      const end = typeof bucket?.end === "string" ? bucket.end : "";
      const votes = Number(bucket?.votes ?? 0) || 0;
      const explicitLabel =
        typeof bucket?.label === "string" ? bucket.label.trim() : "";
      const startDate = start ? parseHistoryDate(start) : null;
      const endDate = end ? parseHistoryDate(end) : null;
      const startTimestamp = startDate ? startDate.getTime() : Number.NaN;
      const endTimestamp = endDate ? endDate.getTime() : Number.NaN;
      const startLabel = startDate
        ? historyTimeFormatter.format(startDate)
        : "";
      const endLabel = endDate ? historyTimeFormatter.format(endDate) : "";
      const rangeLabel = explicitLabel
        ? explicitLabel
        : startLabel && endLabel
          ? `${startLabel} - ${endLabel}`
          : startLabel || endLabel || "";
      return {
        start,
        end,
        votes,
        startLabel,
        endLabel,
        rangeLabel,
        startTimestamp,
        endTimestamp,
      };
    })
    .filter((bucket) => bucket.rangeLabel || bucket.votes || bucket.start)
    .sort((a, b) => {
      const aTime = Number.isFinite(a.startTimestamp)
        ? a.startTimestamp
        : Number.isFinite(a.endTimestamp)
          ? a.endTimestamp
          : Number.POSITIVE_INFINITY;
      const bTime = Number.isFinite(b.startTimestamp)
        ? b.startTimestamp
        : Number.isFinite(b.endTimestamp)
          ? b.endTimestamp
          : Number.POSITIVE_INFINITY;
      if (aTime !== bTime) {
        return aTime - bTime;
      }
      return a.rangeLabel.localeCompare(b.rangeLabel, "it");
    })
    .map((bucket) => ({
      start: bucket.start,
      end: bucket.end,
      rangeLabel: bucket.rangeLabel || "Intervallo",
      votes: bucket.votes,
      votesLabel: Number.isFinite(bucket.votes)
        ? `${bucket.votes.toLocaleString("it-IT")} voti`
        : "0 voti",
      startLabel: bucket.startLabel,
      endLabel: bucket.endLabel,
    }));

  const firstBucketWithStart = timelineBuckets.find(
    (bucket) => bucket.startLabel,
  );
  const lastBucketWithEnd = [...timelineBuckets]
    .reverse()
    .find((bucket) => bucket.endLabel);
  const timelineRangeStart =
    firstBucketWithStart?.startLabel || timelineBuckets[0]?.rangeLabel || "";
  const timelineRangeEnd =
    lastBucketWithEnd?.endLabel ||
    timelineBuckets[timelineBuckets.length - 1]?.rangeLabel ||
    "";
  const timelineRange =
    timelineRangeStart || timelineRangeEnd
      ? {
          start: timelineRangeStart || timelineRangeEnd,
          end: timelineRangeEnd || timelineRangeStart,
        }
      : null;

  const timelineChart = buildHistoryTimelineChart(
    timelineBuckets,
    timelineRange,
  );

  const mvpRaw = item?.mvp;
  let mvp = null;
  if (mvpRaw && Number(mvpRaw?.votes ?? 0) > 0) {
    const firstName =
      typeof mvpRaw?.first_name === "string" ? mvpRaw.first_name.trim() : "";
    const lastName =
      typeof mvpRaw?.last_name === "string" ? mvpRaw.last_name.trim() : "";
    const fallbackName = mvpRaw?.player_id
      ? `Giocatore ${mvpRaw.player_id}`
      : "Giocatore";
    const name =
      [firstName, lastName].filter(Boolean).join(" ") || fallbackName;
    mvp = {
      id: Number(mvpRaw?.player_id) || 0,
      votes: Number(mvpRaw?.votes) || 0,
      name,
    };
  }

  const feedbackSurvey = normalizeFeedbackSurveyInput(
    item?.feedback_survey ?? item?.feedbackSurvey,
  );
  const feedbackSummary = normalizeFeedbackSummary(
    item?.feedback_summary ?? item?.feedbackSummary,
    feedbackSurvey,
  );
  const aiReport = normalizeAiHistoryReport(
    item?.ai_report ?? item?.aiReport ?? null,
  );
  const trackingEvents = normalizeHistoryTrackingEvents(
    item?.tracking_events ?? item?.trackingEvents,
  );
  const trackingAnalytics = normalizeHistoryTrackingAnalytics(
    item?.tracking_analytics ?? item?.trackingAnalytics,
  );

  return {
    id,
    title: rawTitle || fallbackTitle,
    startDatetime,
    location,
    totalVotes,
    totalVisitors,
    totalVisitorsLabel,
    uniqueVisitors,
    uniqueVisitorsLabel,
    totalVotesLabel: Number.isFinite(totalVotes)
      ? totalVotes.toLocaleString("it-IT")
      : "0",
    engagement,
    sponsorClicks,
    sponsorClicksTotal,
    sponsorClicksTotalLabel,
    sponsorAnalytics: sponsorAnalyticsData,
    sponsorAnalyticsDisplay,
    sponsorAnalyticsHasData,
    sponsorAnalyticsTimeline,
    timeline: timelineBuckets,
    timelineChart,
    timelineRange,
    mvp,
    homeTeam,
    awayTeam,
    prizes: normalizedPrizes,
    hasPrizeDraw,
    isTimelineExpanded: false,
    feedbackSummary,
    feedbackSurvey,
    aiReport,
    trackingEvents,
    trackingAnalytics,
  };
}

function normalizeAiHistoryReport(item) {
  if (!item || typeof item !== "object") {
    return null;
  }
  const metrics = item?.metrics && typeof item.metrics === "object" ? item.metrics : {};
  const asList = (value) =>
    Array.isArray(value)
      ? value.map((entry) => (typeof entry === "string" ? entry.trim() : "")).filter(Boolean)
      : [];
  return {
    source: typeof item?.source === "string" ? item.source.trim() : "fallback",
    executiveSummary:
      typeof item?.executive_summary === "string"
        ? item.executive_summary.trim()
        : typeof item?.executiveSummary === "string"
          ? item.executiveSummary.trim()
          : "",
    fullReport:
      typeof item?.full_report === "string"
        ? item.full_report.trim()
        : typeof item?.fullReport === "string"
          ? item.fullReport.trim()
          : "",
    insights: asList(item?.insights),
    suggestions: asList(item?.suggestions),
    strengths: asList(item?.strengths),
    criticalities: asList(item?.criticalities),
    generatedAt:
      typeof item?.generated_at === "string"
        ? item.generated_at
        : typeof item?.generatedAt === "string"
          ? item.generatedAt
          : "",
    metrics: {
      uniqueVoters: Number(metrics?.unique_voters ?? metrics?.uniqueVoters ?? 0) || 0,
      totalSessions: Number(metrics?.total_sessions ?? metrics?.totalSessions ?? 0) || 0,
      newFansRegistered: Number(metrics?.new_fans_registered ?? metrics?.newFansRegistered ?? 0) || 0,
      returningFans: Number(metrics?.returning_fans ?? metrics?.returningFans ?? 0) || 0,
      totalInteractions: Number(metrics?.total_interactions ?? metrics?.totalInteractions ?? 0) || 0,
      rewardRedemptions: Number(metrics?.reward_redemptions ?? metrics?.rewardRedemptions ?? 0) || 0,
      coinsSpentOnRewards: Number(metrics?.coins_spent_on_rewards ?? metrics?.coinsSpentOnRewards ?? 0) || 0,
      sponsorTotalClicks: Number(metrics?.sponsor_total_clicks ?? metrics?.sponsorTotalClicks ?? 0) || 0,
      sponsorSeenSessions: Number(metrics?.sponsor_seen_sessions ?? metrics?.sponsorSeenSessions ?? 0) || 0,
      barOrdersCount: Number(metrics?.bar_orders_count ?? metrics?.barOrdersCount ?? 0) || 0,
      barRevenueCents: Number(metrics?.bar_revenue_cents ?? metrics?.barRevenueCents ?? 0) || 0,
      peakActivityLabel:
        typeof metrics?.peak_activity_label === "string"
          ? metrics.peak_activity_label
          : typeof metrics?.peakActivityLabel === "string"
            ? metrics.peakActivityLabel
            : "",
    },
  };
}

function buildHistoryReportFilename(entry) {
  const eventId = Number(entry?.id) || 0;
  const parsedDate = parseHistoryDate(entry?.startDatetime);
  const datePart = parsedDate
    ? `${parsedDate.getFullYear()}${String(parsedDate.getMonth() + 1).padStart(2, "0")}${String(
        parsedDate.getDate(),
      ).padStart(2, "0")}`
    : "";
  const rawTitle = typeof entry?.title === "string" ? entry.title : "";
  const normalizedTitle = rawTitle
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  const parts = [];
  if (datePart) {
    parts.push(datePart);
  }
  if (normalizedTitle) {
    parts.push(normalizedTitle);
  }
  if (eventId) {
    parts.push(`evento-${eventId}`);
  } else {
    parts.push("evento-storico");
  }
  return `${parts.join("_")}.pdf`;
}

async function downloadEventHistoryReport(entry) {
  if (!entry || typeof entry !== "object" || !entry.id) {
    return;
  }

  const eventId = entry.id;
  setHistoryReportBusy(eventId, true);
  eventHistoryError.value = "";

  try {
    await nextTick();
    const config = {
      ...authHeaders.value,
      responseType: "blob",
    };
    const response = await apiClient.get(
      `/admin/events/history/${eventId}/report`,
      config,
    );
    const blob = new Blob([response?.data], { type: "application/pdf" });
    const filename =
      buildHistoryReportFilename(entry) || `evento-${eventId}.pdf`;
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    const safeTitle =
      typeof entry.title === "string" ? entry.title : `Evento #${entry.id}`;
    eventHistorySuccess.value = `Report per "${safeTitle}" scaricato correttamente.`;
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else if (error?.response?.status === 404) {
      eventHistoryError.value = "Il report richiesto non è disponibile.";
    } else {
      console.error("history report download error", error);
      eventHistoryError.value =
        "Impossibile generare il report PDF. Riprova più tardi.";
    }
    eventHistorySuccess.value = "";
  } finally {
    setHistoryReportBusy(eventId, false);
  }
}

async function generateEventAiReport(entry) {
  if (!entry?.id) return;
  setHistoryAiBusy(entry.id, true);
  eventHistoryError.value = "";
  eventHistorySuccess.value = "";
  try {
    const { data } = await apiClient.post(
      `/admin/events/${entry.id}/ai-report`,
      {},
      authHeaders.value,
    );
    const normalized = normalizeAiHistoryReport(data);
    entry.aiReport = normalized;
    eventHistorySuccess.value = `Report AI aggiornato per "${entry.title}".`;
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else {
      console.error("event ai report generation error", error);
      eventHistoryError.value =
        "Impossibile generare il report AI per questo evento.";
    }
  } finally {
    setHistoryAiBusy(entry.id, false);
  }
}

async function loadEventHistory({ force = false } = {}) {
  if (isLoadingEventHistory.value) {
    return;
  }
  if (!force && hasLoadedEventHistory.value) {
    return;
  }

  if (force) {
    hasLoadedEventHistory.value = false;
  }

  isLoadingEventHistory.value = true;
  eventHistoryError.value = "";
  if (force) {
    eventHistorySuccess.value = "";
  }

  try {
    const { data } = await apiClient.get(
      "/admin/events/history",
      authHeaders.value,
    );
    const normalized = Array.isArray(data)
      ? data.map((entry) => normalizeHistoryEntry(entry))
      : [];
    eventHistory.value = normalized;
    hasLoadedEventHistory.value = true;
  } catch (error) {
    const status = error?.response?.status;
    if (status === 401) {
      handleUnauthorized();
    } else {
      eventHistorySuccess.value = "";
      eventHistoryError.value =
        "Impossibile caricare lo storico eventi. Riprova più tardi.";
    }
  } finally {
    isLoadingEventHistory.value = false;
  }
}

function toggleHistoryTimeline(entry) {
  if (!entry || typeof entry !== "object") {
    return;
  }
  entry.isTimelineExpanded = !entry.isTimelineExpanded;
}

async function refreshEventHistory() {
  await loadEventHistory({ force: true });
}

function openPurgeDialog(entry) {
  purgeDialog.visible = true;
  purgeDialog.event = entry;
  purgeDialog.password = "";
  purgeDialog.error = "";
  purgeDialog.isSubmitting = false;
}

function closePurgeDialog() {
  purgeDialog.visible = false;
  purgeDialog.event = null;
  purgeDialog.password = "";
  purgeDialog.error = "";
  purgeDialog.isSubmitting = false;
}

async function confirmPurge() {
  if (!purgeDialog.event || purgeDialog.isSubmitting || !purgeDialog.password) {
    return;
  }
  purgeDialog.isSubmitting = true;
  purgeDialog.error = "";

  try {
    await apiClient.post(
      `/admin/events/${purgeDialog.event.id}/purge`,
      { password: purgeDialog.password },
      authHeaders.value,
    );
    const removedTitle = purgeDialog.event.title;
    closePurgeDialog();
    await loadEvents();
    await loadEventHistory({ force: true });
    eventHistorySuccess.value = `Evento "${removedTitle}" eliminato.`;
  } catch (error) {
    const status = error?.response?.status;
    if (status === 401) {
      handleUnauthorized();
      return;
    }
    if (status === 403) {
      purgeDialog.error = "Password non valida o privilegi insufficienti.";
    } else if (status === 404) {
      purgeDialog.error = "Evento già rimosso.";
      eventHistory.value = eventHistory.value.filter(
        (entry) => entry.id !== purgeDialog.event.id,
      );
    } else {
      purgeDialog.error = "Impossibile eliminare l'evento. Riprova.";
    }
  } finally {
    purgeDialog.isSubmitting = false;
  }
}

async function loadAll() {
  if (!isAuthenticated.value) {
    return;
  }
  await Promise.all([loadEvents(), loadTeams()]);
  await loadPlayers();
  if (isSuperAdmin.value) {
    await Promise.all([loadAdmins(), loadSponsors(), loadPartners()]);
    await loadCoupons();
  }
  ensureSelfieSelection();
  if (section.value === "selfies" && selectedSelfieEventId.value) {
    await loadEventSelfies(selectedSelfieEventId.value);
  }
  hasLoadedEventHistory.value = false;
  if (section.value === "history") {
    await loadEventHistory({ force: true });
  }
  resetForms();
}

async function createTeam() {
  if (!newTeamName.value) {
    return;
  }
  globalError.value = "";
  await secureRequest(() =>
    apiClient.post("/teams", { name: newTeamName.value }, authHeaders.value),
  );
  newTeamName.value = "";
  await loadTeams();
}

async function deleteTeam(id) {
  globalError.value = "";
  await secureRequest(() =>
    apiClient.delete(`/teams/${id}`, authHeaders.value),
  );
  await loadTeams();
}

async function createEvent() {
  globalError.value = "";
  if (!hasEnoughTeams.value) {
    globalError.value = "Aggiungi almeno due squadre per creare un evento.";
    return;
  }
  if (!newEvent.team1_id || !newEvent.team2_id) {
    globalError.value = "Seleziona entrambe le squadre.";
    return;
  }
  if (newEvent.team1_id === newEvent.team2_id) {
    globalError.value = "Le due squadre devono essere diverse.";
    return;
  }
  if (!newEvent.start_datetime) {
    globalError.value = "Imposta data e ora della partita.";
    return;
  }

  const prizesPayload = newEventPrizes.value
    .map((prize, index) => ({
      id: Number(prize.id) || 0,
      name: (prize.name || "").trim(),
      position: index + 1,
      win_sms_text: (prize.winSmsText || '').trim(),
    }))
    .filter((prize) => prize.name);

  const payload = {
    team1_id: newEvent.team1_id,
    team2_id: newEvent.team2_id,
    start_datetime: newEvent.start_datetime,
    location: newEvent.location,
    show_pre_vote_sponsors: Boolean(newEvent.show_pre_vote_sponsors),
    show_pre_vote_bottom_sponsors: Boolean(
      newEvent.show_pre_vote_bottom_sponsors,
    ),
    show_vote_counter: Boolean(newEvent.show_vote_counter),
    show_reaction_test: Boolean(newEvent.show_reaction_test),
    show_selfie: Boolean(newEvent.show_selfie),
    show_vote_trend: Boolean(newEvent.show_vote_trend),
    show_feedback_survey: Boolean(newEvent.show_feedback_survey),
    show_branded_game: Boolean(newEvent.show_branded_game),
    branded_game_config: newEvent.show_branded_game
      ? JSON.stringify(newEvent.brandedGameConfigDraft ?? defaultBrandedGameConfig())
      : "",
    feedback_survey: toApiSurveyPayload(newEventSurvey),
    prizes: prizesPayload,
  };

  if (newEvent.show_branded_game) {
    const bgErr = validateBrandedGameConfigClient(newEvent.brandedGameConfigDraft ?? {});
    if (bgErr) {
      globalError.value = `Branded Game: ${bgErr}`;
      return;
    }
  }

  const { data } = await secureRequest(() =>
    apiClient.post("/events", payload, authHeaders.value),
  );
  await loadEvents();
  if (data?.id) {
    lastCreatedEventLink.value = buildEventLink(data.id);
  }
  Object.assign(newEvent, createDefaultNewEventState());
  assignSurveyDraft(newEventSurvey, null);
  teamInputs.home = "";
  teamInputs.away = "";
  resetNewEventPrizes();
}

async function deleteEvent(id) {
  globalError.value = "";
  await secureRequest(() =>
    apiClient.delete(`/events/${id}`, authHeaders.value),
  );
  await loadEvents();
}

async function activateEvent(id) {
  if (updatingEventId.value === id) {
    return;
  }
  globalError.value = "";
  closeVotesMessage.value = "";
  updatingEventId.value = id;
  try {
    await secureRequest(() =>
      apiClient.post(`/events/${id}/activate`, {}, authHeaders.value),
    );
    await loadEvents();
  } finally {
    updatingEventId.value = 0;
  }
}

async function deactivateEvents() {
  if (isDisablingEvents.value) {
    return;
  }
  globalError.value = "";
  closeVotesMessage.value = "";
  isDisablingEvents.value = true;
  try {
    await secureRequest(() =>
      apiClient.post("/events/deactivate", {}, authHeaders.value),
    );
    await loadEvents();
  } finally {
    isDisablingEvents.value = false;
  }
}

async function concludeEvent(id) {
  if (concludingEventId.value === id) {
    return;
  }
  globalError.value = "";
  closeVotesMessage.value = "";
  const eventInfo = events.value.find((event) => event.id === id);
  const concludedLabel = eventInfo ? eventLabel(eventInfo) : "";
  concludingEventId.value = id;
  try {
    await secureRequest(() =>
      apiClient.post(`/events/${id}/conclude`, {}, authHeaders.value),
    );
    await loadEvents();
    await loadEventHistory({ force: true });
    if (!eventHistoryError.value) {
      eventHistorySuccess.value = concludedLabel
        ? `Evento "${concludedLabel}" spostato nello storico.`
        : "Evento spostato nello storico.";
    }
  } catch (error) {
    const status = error?.response?.status;
    if (status === 401) {
      return;
    }
    if (status === 404) {
      globalError.value = "Evento non trovato o già rimosso.";
    } else if (status === 409) {
      globalError.value = "L'evento è già stato segnato come concluso.";
    }
    await loadEvents();
  } finally {
    concludingEventId.value = 0;
  }
}

async function closeActiveEventVoting() {
  if (
    !activeEventId.value ||
    isClosingVotes.value ||
    activeEventVotesClosed.value
  ) {
    return;
  }
  closeVotesMessage.value = "";
  globalError.value = "";
  isClosingVotes.value = true;
  try {
    await secureRequest(() =>
      apiClient.post(
        `/events/${activeEventId.value}/close-votes`,
        {},
        authHeaders.value,
      ),
    );
    await loadEvents();
    closeVotesMessage.value =
      "Le votazioni per l'evento attivo sono state chiuse.";
  } catch (error) {
    closeVotesMessage.value = "";
    if (error?.response?.status === 404) {
      globalError.value =
        "Impossibile chiudere le votazioni: nessun evento attivo trovato.";
    }
  } finally {
    isClosingVotes.value = false;
  }
}

async function createAdmin() {
  globalError.value = "";
  await secureRequest(() =>
    apiClient.post("/admins", newAdmin, authHeaders.value),
  );
  Object.assign(newAdmin, { username: "", password: "", role: "" });
  await loadAdmins();
}

async function deleteAdmin(id) {
  globalError.value = "";
  await secureRequest(() =>
    apiClient.delete(`/admins/${id}`, authHeaders.value),
  );
  await loadAdmins();
}

async function createPartner() {
  const username = (newPartner.username || newPartner.name).trim();
  const password = newPartner.password;
  globalError.value = "";

  if (!username || !password) {
    globalError.value = "Inserisci almeno nome utente e password.";
    return;
  }

  await secureRequest(() =>
    apiClient.post(
      "/admin/partners",
      { username, password },
      authHeaders.value,
    ),
  );
  Object.assign(newPartner, { name: "", username: "", password: "" });
  await loadPartners();
}

async function updatePartnerPassword(partner) {
  if (!partner?.id || partnerBeingUpdated.value === partner.id) {
    return;
  }

  const trimmed = (partner.newPassword || "").trim();
  if (!trimmed) {
    return;
  }

  globalError.value = "";
  partnerBeingUpdated.value = partner.id;
  try {
    await secureRequest(() =>
      apiClient.put(
        `/admin/partners/${partner.id}`,
        { password: trimmed },
        authHeaders.value,
      ),
    );
    partner.newPassword = "";
    await loadPartners();
  } finally {
    partnerBeingUpdated.value = 0;
  }
}

async function deletePartner(id) {
  if (!id || partnerBeingDeleted.value === id) {
    return;
  }
  globalError.value = "";
  partnerBeingDeleted.value = id;
  try {
    await secureRequest(() =>
      apiClient.delete(`/admin/partners/${id}`, authHeaders.value),
    );
    await loadPartners();
  } finally {
    partnerBeingDeleted.value = 0;
  }
}

async function createSponsor() {
  if (isCreatingSponsor.value) {
    return;
  }
  globalError.value = "";
  if (sponsors.value.length >= maxSponsors) {
    globalError.value = `Puoi configurare al massimo ${maxSponsors} sponsor.`;
    return;
  }
  const trimmedName = newSponsor.name.trim();
  if (!newSponsor.logoData) {
    globalError.value = "Carica un logo per lo sponsor.";
    return;
  }
  const payload = serializeSponsorPayload({
    name: trimmedName,
    reportName: newSponsor.reportName,
    linkUrl: newSponsor.linkUrl,
    logoData: newSponsor.logoData,
    position: nextSponsorPosition(),
    isActive: false,
  });
  isCreatingSponsor.value = true;
  try {
    await secureRequest(() =>
      apiClient.post("/admin/sponsors", payload, authHeaders.value),
    );
    resetNewSponsorForm();
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value =
        "Controlla i dati inseriti: sono disponibili massimo 4 sponsor.";
    }
  } finally {
    isCreatingSponsor.value = false;
  }
}

async function updateSponsorEntry(sponsor) {
  if (sponsorBeingUpdated.value === sponsor.id) {
    return;
  }
  globalError.value = "";
  const trimmedName = sponsor.name.trim();
  if (!sponsor.logoData) {
    globalError.value = "Carica un logo per lo sponsor.";
    return;
  }
  sponsorBeingUpdated.value = sponsor.id;
  try {
    const payload = serializeSponsorPayload({
      name: trimmedName,
      reportName: sponsor.reportName,
      linkUrl: sponsor.linkUrl,
      logoData: sponsor.logoData,
      position: sponsor.position,
      isActive: sponsor.isActive,
    });
    await secureRequest(() =>
      apiClient.put(
        `/admin/sponsors/${sponsor.id}`,
        payload,
        authHeaders.value,
      ),
    );
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value = "Controlla i dati dello sponsor e riprova.";
    } else if (error?.response?.status === 404) {
      globalError.value = "Sponsor non trovato. Aggiorna la pagina.";
    }
  } finally {
    sponsorBeingUpdated.value = 0;
  }
}

async function deleteSponsorEntry(id) {
  if (sponsorBeingDeleted.value === id) {
    return;
  }
  globalError.value = "";
  sponsorBeingDeleted.value = id;
  try {
    await secureRequest(() =>
      apiClient.delete(`/admin/sponsors/${id}`, authHeaders.value),
    );
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 404) {
      globalError.value = "Sponsor già rimosso.";
    }
  } finally {
    sponsorBeingDeleted.value = 0;
  }
}

async function applyActiveSponsorCount() {
  if (isApplyingSponsorCount.value) {
    return;
  }
  if (!sponsors.value.length) {
    desiredActiveSponsorCount.value = 0;
    return;
  }
  globalError.value = "";
  const target = Math.max(
    0,
    Math.min(maxSponsors, desiredActiveSponsorCount.value),
  );
  isApplyingSponsorCount.value = true;
  try {
    const updates = [];
    sortedSponsors().forEach((sponsor, index) => {
      const shouldBeActive = index < target;
      if (sponsor.isActive !== shouldBeActive) {
        const payload = serializeSponsorPayload({
          name: sponsor.name.trim(),
          reportName: sponsor.reportName,
          linkUrl: sponsor.linkUrl,
          logoData: sponsor.logoData,
          position: sponsor.position,
          isActive: shouldBeActive,
        });
        updates.push(
          secureRequest(() =>
            apiClient.put(
              `/admin/sponsors/${sponsor.id}`,
              payload,
              authHeaders.value,
            ),
          ),
        );
      }
    });
    if (updates.length) {
      await Promise.all(updates);
    }
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value =
        "Impossibile aggiornare il numero di sponsor visibili. Verifica i dati e riprova.";
    }
  } finally {
    isApplyingSponsorCount.value = false;
  }
}

async function createCoupon() {
  if (isCreatingCoupon.value) {
    return;
  }
  couponError.value = "";
  couponSuccess.value = "";
  if (!newCoupon.title.trim()) {
    couponError.value = "Inserisci un titolo per il coupon.";
    return;
  }
  if (!newCoupon.sponsorId) {
    couponError.value = "Seleziona il partner associato.";
    return;
  }
  const payload = serializeCouponPayload(newCoupon);
  isCreatingCoupon.value = true;
  try {
    await secureRequest(() =>
      apiClient.post("/admin/coupons", payload, authHeaders.value),
    );
    couponSuccess.value = "Coupon creato correttamente.";
    resetNewCouponForm();
    await loadCoupons();
  } catch (error) {
    if (error?.response?.status === 400) {
      couponError.value = "Controlla i dati inseriti per il coupon.";
    }
  } finally {
    isCreatingCoupon.value = false;
  }
}

async function updateCouponEntry(coupon) {
  if (!coupon?.id || couponBeingSaved.value === coupon.id) {
    return;
  }
  couponError.value = "";
  couponSuccess.value = "";
  const payload = serializeCouponPayload(coupon);
  couponBeingSaved.value = coupon.id;
  try {
    await secureRequest(() =>
      apiClient.put(
        `/admin/coupons/${coupon.id}`,
        payload,
        authHeaders.value,
      ),
    );
    couponSuccess.value = "Coupon aggiornato.";
    await loadCoupons();
  } catch (error) {
    if (error?.response?.status === 400) {
      couponError.value = "Impossibile salvare il coupon. Verifica i campi.";
    }
  } finally {
    couponBeingSaved.value = 0;
  }
}

async function deleteCouponEntry(id) {
  if (!id || couponBeingDeleted.value === id) {
    return;
  }
  couponError.value = "";
  couponSuccess.value = "";
  couponBeingDeleted.value = id;
  try {
    await secureRequest(() =>
      apiClient.delete(`/admin/coupons/${id}`, authHeaders.value),
    );
    await loadCoupons();
  } catch (error) {
    if (error?.response?.status === 404) {
      couponError.value = "Coupon già rimosso.";
    }
  } finally {
    couponBeingDeleted.value = 0;
  }
}

function openVote(eventId) {
  const url = buildEventLink(eventId);
  window.open(url, "_blank", "noopener");
}

async function copyLink(link) {
  try {
    await navigator.clipboard.writeText(link);
    globalError.value = "";
  } catch (error) {
    globalError.value = "Impossibile copiare il link automaticamente.";
  }
}

function updateToolbarOffset() {
  if (!portalRef.value) {
    return;
  }
  const height = toolbarRef.value?.offsetHeight ?? 0;
  portalRef.value.style.setProperty("--toolbar-height", `${height}px`);
}

function ensureSectionIsAllowed(tabList) {
  if (!isAuthenticated.value) {
    return;
  }
  if (!tabList.some((tab) => tab.id === section.value)) {
    section.value = tabList.length ? tabList[0].id : "";
  }
}

onMounted(() => {
  window.addEventListener("resize", updateToolbarOffset, { passive: true });
  nextTick(updateToolbarOffset);
});

watch(isAuthenticated, () => {
  nextTick(updateToolbarOffset);
});

watch(
  availableTabs,
  (currentTabs) => {
    ensureSectionIsAllowed(currentTabs);
    nextTick(updateToolbarOffset);
  },
  { immediate: true },
);

watch(section, (value, oldValue) => {
  if (value === "results") {
    ensureResultsSelection();
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (oldValue === "results") {
    stopResultsPolling();
  }
  if (value === "selfies") {
    ensureSelfieSelection();
    if (selectedSelfieEventId.value) {
      loadEventSelfies(selectedSelfieEventId.value);
    }
  } else if (oldValue === "selfies") {
    selfieModerationMessage.value = "";
    selfieLoadError.value = "";
  }
  if (value === "history") {
    loadEventHistory();
  }
  nextTick(updateToolbarOffset);
});

watch(selectedResultsEventId, (eventId) => {
  if (section.value === "results" && eventId) {
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (!eventId) {
    stopResultsPolling();
  }
});

watch(selectedSelfieEventId, (eventId) => {
  if (section.value !== "selfies") {
    return;
  }
  selfieModerationMessage.value = "";
  selfieLoadError.value = "";
  if (eventId) {
    loadEventSelfies(eventId);
  } else {
    eventSelfies.value = [];
  }
});

if (isAuthenticated.value) {
  selectDefaultSection();
  loadAll();
}

onBeforeUnmount(() => {
  window.removeEventListener("resize", updateToolbarOffset);
  stopResultsPolling();
});
</script>

<style scoped>
.admin-portal {
  --radius: 12px;
  --gap-md: 16px;
  --gap-lg: 24px;
  --control-height: 44px;
  --ui-border: rgba(148, 163, 184, 0.35);
  min-height: 100vh;
  color: #0f172a;
  background: #f1f5f9;
}

.admin-header {
  margin-bottom: 1.5rem;
  color: #0f172a;
}

.admin-header h1 {
  font-size: 1.75rem;
  margin: 0;
  color: #0f172a;
}

.subtitle {
  margin: 0.5rem 0 0;
  color: #475569;
}

.context-badge {
  display: inline-block;
  margin-top: 0.35rem;
  padding: 0.35rem 0.75rem;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.15);
  color: #bfdbfe;
  font-weight: 600;
  font-size: 0.95rem;
}

.portal { display: block; }

.portal-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 1280px;
  margin: 0 auto;
  width: 100%;
}

.header-user {
  font-weight: 600;
  color: #475569;
}

.dashboard-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.dashboard-card {
  border: 1px solid rgba(148, 163, 184, 0.3);
  border-radius: 0.9rem;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.section-nav {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0;
  position: relative;
}

.card {
  background: #ffffff;
  border-radius: var(--radius);
  padding: 1.5rem;
  box-shadow: 0 15px 35px rgba(15, 23, 42, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.18);
}

.login-card {
  max-width: 520px;
  margin: 3rem auto;
}

.section-header h2 {
  margin: 0 0 0.5rem;
}

.section-header p {
  margin: 0;
  color: #64748b;
}

.info-banner {
  margin: 0 0 1rem;
  padding: 0.85rem 1rem;
  border-radius: 0.75rem;
  background: rgba(59, 130, 246, 0.12);
  color: #1d4ed8;
  font-weight: 500;
}

.info-banner.warning {
  background: rgba(251, 191, 36, 0.18);
  color: #92400e;
}

.actions-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.actions-row .btn {
  padding-left: 1.25rem;
  padding-right: 1.25rem;
}

.form-grid {
  display: grid;
  gap: var(--gap-md);
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  margin-bottom: 1.5rem;
}

.modern-form-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.form-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

.postvote-options {
  margin: 1.25rem 0;
  padding: 1.25rem 1.5rem;
  border-radius: 1.25rem;
  border: 1px solid rgba(148, 163, 184, 0.24);
  background: rgba(248, 250, 252, 0.9);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.08);
  grid-column: 1 / -1;
}

.postvote-options.new-event-prevote,
.postvote-options.new-event-postvote {
  margin-top: 0.5rem;
}

.postvote-options__header {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
  color: #1e293b;
}

.postvote-options__header span,
.postvote-options__header strong {
  font-size: 1rem;
  font-weight: 700;
}

.postvote-options__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 0.75rem;
}

.quiz-toggle-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.quiz-toggle-actions .btn {
  flex: 1;
  min-width: 160px;
}

.postvote-toggle {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.26);
  background: rgba(255, 255, 255, 0.9);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4);
  font-weight: 600;
  color: #0f172a;
}

.postvote-toggle input[type="checkbox"] {
  width: 2.5rem;
  height: 1.5rem;
  appearance: none;
  border-radius: 999px;
  position: relative;
  background: #cbd5e1;
  transition: background 0.2s ease;
  cursor: pointer;
}

.postvote-toggle input[type="checkbox"]::before {
  content: "";
  position: absolute;
  top: 0.13rem;
  left: 0.16rem;
  width: 1.2rem;
  height: 1.2rem;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
  transition: transform 0.2s ease;
}

.postvote-toggle input[type="checkbox"]:checked {
  background: #2563eb;
}

.postvote-toggle input[type="checkbox"]:checked::before {
  transform: translateX(0.95rem);
}

.postvote-toggle__label {
  flex: 1;
  cursor: pointer;
}

.postvote-toggle input[type="checkbox"]:disabled {
  cursor: not-allowed;
}

.postvote-toggle input[type="checkbox"]:disabled + .postvote-toggle__label {
  opacity: 0.55;
}

.player-slots {
  display: grid;
  gap: 1.5rem;
  margin-top: 1.5rem;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.player-slot {
  padding: 1.25rem;
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.28);
  background: rgba(248, 250, 252, 0.95);
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
}

.player-slot legend {
  font-weight: 700;
  font-size: 0.95rem;
  color: #0f172a;
  margin-bottom: 1rem;
}

.player-slot__grid {
  display: grid;
  gap: 1rem;
}

.player-slot__grid label {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

.player-slot__grid .checkbox-inline {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}

.player-slot__grid .checkbox-inline input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
}

.player-slot__grid input,
.player-slot__grid select {
  border-radius: 0.65rem;
  border: 1px solid rgba(148, 163, 184, 0.45);
  padding: 0.55rem 0.75rem;
  font-size: 0.95rem;
  background: #fff;
}

.player-slot__preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.player-slot__preview img {
  width: 100%;
  max-width: 200px;
  aspect-ratio: 3 / 4;
  object-fit: cover;
  border-radius: 0.85rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.18);
}

.coupon-file-input {
  align-self: flex-start;
}

.coupon-image-preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.coupon-image-preview img {
  width: 100%;
  max-width: 220px;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  border-radius: 0.85rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.18);
}

.player-schema {
  margin: 0.75rem 0 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.player-schema label {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 700;
  color: #0f172a;
}

.player-schema select {
  max-width: 240px;
  border-radius: 0.65rem;
  border: 1px solid rgba(148, 163, 184, 0.45);
  padding: 0.55rem 0.75rem;
  font-size: 0.95rem;
  background: #fff;
}

.prize-editor {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.prize-editor__header {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.prize-editor__list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.prize-editor__row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  align-items: center;
}

.prize-editor__row input {
  flex: 1 1 220px;
}

.prize-editor__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.prize-editor__winner {
  font-size: 0.85rem;
  color: #0f766e;
  font-weight: 600;
}

.prize-editor.existing-prizes {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px dashed rgba(148, 163, 184, 0.5);
}

.prize-editor__row .btn {
  flex: 0 0 auto;
}

.feedback-editor {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-top: 0.75rem;
}

.feedback-editor__header {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.feedback-editor__content {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.feedback-editor__question {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 0.85rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(148, 163, 184, 0.08);
}

.feedback-editor__question-title {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-weight: 600;
  color: #0f172a;
}

.feedback-editor__answers {
  display: grid;
  gap: 0.65rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.feedback-editor__answer {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.feedback-editor__answer-meta {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  color: #475569;
}

.feedback-editor__answer-icon {
  font-size: 1.1rem;
  line-height: 1;
}

.feedback-editor__answer-code {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.45rem;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.08);
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 0.75rem;
  color: #1e293b;
}

.feedback-editor__suggestion {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-weight: 600;
  color: #0f172a;
}

.feedback-editor__suggestion textarea {
  resize: vertical;
  min-height: 72px;
}

.feedback-editor.existing-feedback {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px dashed rgba(148, 163, 184, 0.5);
}

input,
select,
textarea {
  border-radius: var(--radius);
  border: 1px solid var(--ui-border);
  padding: 0.65rem 0.85rem;
  font-size: 0.95rem;
  min-height: var(--control-height);
  background: #fff;
  color: #0f172a;
}

input::placeholder,
textarea::placeholder {
  color: #94a3b8;
}

.field-hint {
  font-size: 0.75rem;
  color: #64748b;
}

input:focus,
select:focus,
textarea:focus {
  outline: none;
  border-color: #6366f1;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  border-radius: 999px;
  border: none;
  padding: 0.6rem 1.4rem;
  font-weight: 600;
  cursor: pointer;
  transition:
    transform 0.15s ease,
    box-shadow 0.15s ease;
}

.btn.primary {
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  box-shadow: 0 12px 25px rgba(59, 130, 246, 0.35);
}

.btn.warning {
  background: #f59e0b;
  color: #0f172a;
}

.btn.warning:disabled {
  opacity: 0.85;
}

.btn.secondary {
  background: #e2e8f0;
  color: #0f172a;
}

.btn.success {
  background: #22c55e;
  color: #fff;
}

.btn.success:disabled {
  opacity: 0.8;
  cursor: default;
}

.btn.outline {
  background: transparent;
  color: #2563eb;
  border: 1px solid rgba(37, 99, 235, 0.4);
}

.btn.danger {
  background: #f87171;
  color: #fff;
}

.btn.link {
  background: transparent;
  color: #2563eb;
  padding: 0.2rem 0.4rem;
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
  box-shadow: none;
}

.checkbox-inline input[type="checkbox"],
.checkbox input[type="checkbox"] {
  width: 1.25rem;
  height: 1.25rem;
  accent-color: #2563eb;
}

.btn:not(:disabled):hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 20px rgba(15, 23, 42, 0.15);
}

.section-nav__button {
  font-size: 0.75rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 0.55rem 1.35rem;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.45);
  background: rgba(15, 23, 42, 0.75);
  color: #f8fafc;
  transition:
    background 0.2s ease,
    color 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;
}

.section-nav__button:not(.active):hover,
.section-nav__button:not(.active):focus-visible {
  background: rgba(148, 163, 184, 0.3);
  border-color: rgba(226, 232, 240, 0.65);
  color: #ffffff;
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.35);
}

.section-nav__button:focus-visible {
  outline: 2px solid #fbbf24;
  outline-offset: 3px;
}

.section-nav__button.active {
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  border-color: transparent;
  color: #ffffff;
  box-shadow: 0 18px 36px rgba(59, 130, 246, 0.45);
  transform: translateY(-1px);
}

.item-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.item-list.compact {
  gap: 0.5rem;
}

.item {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 0.85rem;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(248, 250, 252, 0.8);
}

.item.active {
  border-color: rgba(99, 102, 241, 0.55);
  box-shadow: 0 10px 20px rgba(99, 102, 241, 0.2);
}

@media (min-width: 768px) {
  .item {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

@media (max-width: 900px) {
  .modern-form-grid {
    grid-template-columns: 1fr;
  }
}

.item-body h3 {
  margin: 0 0 0.35rem;
}

.partner-actions {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.partner-actions .inline-input {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 12rem;
}

.partner-actions .inline-input input {
  min-width: 0;
}

@media (min-width: 768px) {
  .partner-actions {
    flex-direction: row;
    align-items: center;
    gap: 0.75rem;
  }
}

.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.55rem;
  margin-left: 0.5rem;
  border-radius: 999px;
  background: rgba(79, 70, 229, 0.18);
  color: #4338ca;
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.badge-open {
  background: rgba(34, 197, 94, 0.18);
  color: #15803d;
}

.badge-closed {
  background: rgba(249, 115, 22, 0.2);
  color: #9a3412;
}

.closing-card .active-event-summary {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.closing-card .summary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.success-message {
  margin-top: 1rem;
  color: #15803d;
  font-weight: 600;
}

.item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.muted {
  color: #64748b;
  margin: 0;
}

.muted.small {
  font-size: 0.8rem;
}

.text-center {
  text-align: center;
}

.sponsor-controls {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.history-card {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.history-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.history-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.history-item {
  background: #f8fafc;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 1.25rem;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  box-shadow: 0 18px 28px rgba(15, 23, 42, 0.08);
}

.history-item__header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
}

.history-item__header h3 {
  margin: 0;
  font-size: 1.25rem;
}

.history-item__meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
  font-size: 0.95rem;
  text-align: right;
}

.history-item__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
}

.history-item__actions .btn {
  min-width: 0;
}

.history-item__totals {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.75rem;
}

.history-item__total {
  color: #1e293b;
}

.history-item__unique-visitors {
  color: #334155;
  font-size: 0.9rem;
}

.history-item__unique-visitors strong {
  color: #1e293b;
}

.history-item__sponsor-total {
  color: #334155;
  font-size: 0.9rem;
}

.history-item__sponsor-total strong {
  color: #1e293b;
}

.history-details {
  display: flex;
  flex-wrap: wrap;
  gap: 1.5rem;
}

.history-details__column {
  flex: 1 1 220px;
  background: #edf2f7;
  border-radius: 1rem;
  padding: 1rem 1.25rem;
}

.history-details__column h4 {
  margin: 0 0 0.5rem;
  font-size: 1rem;
  color: #0f172a;
}

.history-details__column--ai {
  flex-basis: 100%;
  background: linear-gradient(180deg, #eef2ff 0%, #f8fafc 100%);
  border: 1px solid rgba(99, 102, 241, 0.16);
}

.history-ai-summary,
.history-ai-report {
  margin: 0 0 0.8rem;
  line-height: 1.6;
  color: #1e293b;
}

.history-ai-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.85rem;
}

.history-ai-card {
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 0.85rem;
  padding: 0.85rem 1rem;
}

.history-ai-card h5 {
  margin: 0 0 0.45rem;
  font-size: 0.95rem;
  color: #0f172a;
}

.history-ai-card ul {
  margin: 0;
  padding-left: 1.1rem;
  color: #334155;
}

.history-engagement {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.history-engagement__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}

.history-engagement__label {
  color: #334155;
  font-weight: 600;
}

.history-engagement__value {
  color: #0f172a;
  font-size: 1rem;
}

.history-engagement__empty {
  margin: 0.25rem 0 0;
}

.history-sponsor-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.history-sponsor-summary {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.history-sponsor-summary__grid {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
}

.history-sponsor-summary__card {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  background: #ffffff;
  border-radius: 0.75rem;
  padding: 0.75rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.history-sponsor-summary__card--wide {
  grid-column: 1 / -1;
}

.history-sponsor-summary__label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #475569;
}

.history-sponsor-summary__value {
  font-size: 1.1rem;
  color: #0f172a;
}

.history-sponsor-summary__hint {
  font-size: 0.8rem;
  color: #64748b;
}

.history-sponsor-name {
  font-weight: 600;
  color: #1e293b;
}

.history-sponsor-clicks {
  margin-left: 0.5rem;
  color: #475569;
  font-size: 0.9rem;
}

.history-sponsor-timeline {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.history-sponsor-timeline h5 {
  margin: 0;
  font-size: 0.95rem;
  color: #0f172a;
}

.history-sponsor-timeline__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.history-sponsor-timeline__item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: #ffffff;
  border-radius: 0.75rem;
  padding: 0.75rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.history-sponsor-timeline__time {
  font-weight: 600;
  color: #1e293b;
}

.history-sponsor-timeline__values {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  font-size: 0.85rem;
}

.history-sponsor-timeline__value {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: #475569;
}

.history-sponsor-timeline__value--seen {
  color: #0284c7;
}

.history-sponsor-timeline__value--watched {
  color: #14b8a6;
}

.history-sponsor-timeline__value--clicks {
  color: #f97316;
}

.history-feedback-summary {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.history-feedback-summary__total {
  margin: 0;
  font-weight: 600;
  color: #0f172a;
}

.history-feedback-summary__question {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.history-feedback-summary__question h5 {
  margin: 0;
  font-size: 0.95rem;
  color: #1e293b;
}

.history-feedback-summary__answers {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.history-feedback-summary__answer {
  background: #ffffff;
  border-radius: 0.75rem;
  padding: 0.6rem 0.75rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.06);
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.history-feedback-summary__answer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  font-size: 0.9rem;
}

.history-feedback-summary__answer-label {
  font-weight: 600;
  color: #0f172a;
}

.history-feedback-summary__answer-count {
  font-variant-numeric: tabular-nums;
  color: #334155;
}

.history-feedback-summary__answer-bar {
  height: 0.4rem;
  border-radius: 999px;
  background: #dbeafe;
  overflow: hidden;
}

.history-feedback-summary__answer-bar-fill {
  display: block;
  height: 100%;
  background: #0ea5e9;
  transition: width 0.3s ease;
}

.history-feedback-summary__suggestions {
  list-style: disc;
  margin: 0;
  padding-left: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  color: #334155;
  font-size: 0.9rem;
}

.history-analytics-kpi {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 0.55rem;
}

.history-analytics-kpi__item {
  background: #ffffff;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 0.65rem;
  padding: 0.55rem 0.65rem;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.history-analytics-kpi__item span {
  font-size: 0.75rem;
  color: #64748b;
}

.history-analytics-kpi__item strong {
  color: #0f172a;
  font-size: 0.95rem;
}

.history-analytics-meta {
  margin: 0.4rem 0 0;
  color: #475569;
  font-size: 0.85rem;
}

.history-analytics-funnels,
.history-analytics-segment,
.history-analytics-top {
  margin-top: 0.7rem;
}

.history-analytics-funnels__item ul,
.history-analytics-top ul {
  list-style: none;
  margin: 0.35rem 0 0;
  padding: 0;
}

.history-analytics-funnels__item li,
.history-analytics-top li {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
  color: #334155;
  padding: 0.15rem 0;
}

.history-prize-status {
  margin: 0 0 0.5rem;
  font-weight: 600;
  font-size: 0.95rem;
}

.history-prize-status--success {
  color: #166534;
}

.history-prize-status--pending {
  color: #b45309;
}

.history-prize-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.history-prize-item {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.history-prize-name {
  font-weight: 600;
  color: #1e293b;
}

.history-prize-code {
  color: #475569;
  font-size: 0.9rem;
}

.history-prize-code strong {
  color: #0f172a;
}

.history-votes {
  border-top: 1px solid rgba(15, 23, 42, 0.08);
  padding-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.history-votes__header {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.history-votes__actions {
  display: flex;
}

.history-votes__actions .btn.link {
  padding-left: 0;
}

.history-votes__header h4 {
  margin: 0;
  font-size: 1rem;
  color: #0f172a;
}

.history-votes__range {
  margin: 0;
  font-size: 0.9rem;
  color: #475569;
}

:deep(.history-votes__chart) {
  margin-top: 0.5rem;
}

.history-votes-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.history-votes-list__item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  border-radius: 0.75rem;
  background: rgba(59, 130, 246, 0.08);
  color: #0f172a;
}

.history-votes-list__range {
  font-weight: 600;
  font-size: 0.95rem;
}

.history-votes-list__votes {
  font-size: 0.9rem;
  color: #1d4ed8;
  font-weight: 600;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  z-index: 20;
}

.modal-card {
  background: #f8fafc;
  border-radius: 1rem;
  padding: 2rem;
  max-width: 420px;
  width: 100%;
  box-shadow: 0 32px 48px rgba(15, 23, 42, 0.35);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.modal-card h3 {
  margin: 0;
  font-size: 1.25rem;
  color: #b91c1c;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

@media (max-width: 720px) {
  .history-item {
    padding: 1.25rem;
  }

  .history-votes-list__item {
    flex-direction: column;
    align-items: flex-start;
  }

  .modal-card {
    padding: 1.5rem;
  }
}

.sponsor-range {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

.sponsor-range input[type="range"] {
  accent-color: #2563eb;
}

.sponsor-form {
  align-items: flex-end;
}

.sponsor-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.85rem;
  border: 1px dashed rgba(148, 163, 184, 0.6);
  background: rgba(241, 245, 249, 0.6);
  overflow: hidden;
  min-height: 120px;
}

.sponsor-preview.new {
  min-height: 100px;
}

.sponsor-preview img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.empty-logo {
  font-size: 0.85rem;
  color: #94a3b8;
}

.sponsors-list {
  margin-top: 1.5rem;
}

.sponsor-item {
  gap: 1.25rem;
}

.sponsor-body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

@media (min-width: 768px) {
  .sponsor-body {
    flex-direction: row;
    align-items: center;
  }

  .sponsor-preview {
    flex: 0 0 220px;
    min-height: 140px;
  }
}

.sponsor-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.form-grid.compact {
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  margin-bottom: 0.75rem;
}

.sponsor-meta {
  font-size: 0.85rem;
}

.coupon-form {
  margin-bottom: 1.25rem;
}

.coupons-list .coupon-item {
  align-items: flex-start;
  gap: 1rem;
}

.choice-group {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.status-options {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.status-option {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  background: #fff;
}

.status-option input {
  margin: 0;
}

.status-option span {
  text-transform: capitalize;
}

.coupon-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.match-selector {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin: 0.5rem 0;
}

.match-selector__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.5rem 1rem;
}

.match-selector__grid--scroll {
  max-height: 260px;
  overflow: auto;
  padding-right: 0.25rem;
}

.match-search {
  width: 100%;
}

.match-selector.inline {
  margin-top: 0;
}

.match-selector.inline .match-selector__grid {
  margin-top: 0.25rem;
}

.coupon-meta {
  font-size: 0.9rem;
}

.inline-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.item-actions.vertical {
  flex-direction: column;
  align-items: stretch;
}

.error {
  color: #dc2626;
  margin-top: 0.75rem;
}

.hint {
  margin: 1rem 0 0;
  padding: 1rem;
  border-radius: 0.75rem;
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
}

.results-card {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.results-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: flex-end;
}

.results-controls label {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-weight: 600;
  color: #1e293b;
}

.results-summary h3 {
  margin: 0;
  font-size: 1.25rem;
}

.results-summary .muted {
  margin: 0.25rem 0 0;
}

.results-leaderboard {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.results-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  font-size: 0.95rem;
  color: #475569;
}

.results-meta .auto-refresh {
  font-size: 0.85rem;
  color: #64748b;
}

.leaderboard-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.leaderboard-item {
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr) 120px;
  gap: 1rem;
  align-items: center;
  padding: 0.85rem 1rem;
  border-radius: 1rem;
  background: linear-gradient(
    135deg,
    rgba(15, 23, 42, 0.85),
    rgba(30, 64, 175, 0.9)
  );
  color: #f8fafc;
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.3);
}

.leaderboard-item .rank {
  font-size: 1.5rem;
  font-weight: 700;
  text-align: center;
}

.leaderboard-item .player-name {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.leaderboard-item .player-name .lastname {
  font-size: 1.2rem;
  letter-spacing: 0.08em;
}

.leaderboard-item .player-name .firstname {
  font-size: 0.95rem;
  text-transform: capitalize;
  opacity: 0.9;
}

.leaderboard-item .votes {
  display: flex;
  align-items: baseline;
  gap: 0.35rem;
  font-size: 1rem;
  justify-content: flex-end;
}

.leaderboard-item .votes strong {
  font-size: 1.4rem;
}

.leaderboard-item .progress {
  grid-column: 1 / -1;
  height: 6px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.35);
  overflow: hidden;
}

.leaderboard-item .progress-bar {
  height: 100%;
  background: linear-gradient(135deg, #facc15, #f97316);
  border-radius: inherit;
  transition: width 0.4s ease;
}

.sponsor-analytics {
  margin-top: 1.5rem;
  padding: 1.5rem;
  border-radius: 1.5rem;
  background: linear-gradient(
    145deg,
    rgba(15, 23, 42, 0.9),
    rgba(30, 64, 175, 0.75)
  );
  border: 1px solid rgba(148, 163, 184, 0.25);
  color: #e2e8f0;
  box-shadow: 0 18px 36px rgba(15, 23, 42, 0.35);
}

.sponsor-analytics h3 {
  margin: 0 0 1rem;
  font-size: 1.25rem;
  font-weight: 600;
  color: #f8fafc;
}

.sponsor-analytics__content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.sponsor-analytics__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 1rem;
}

.sponsor-analytics__card {
  padding: 1rem 1.25rem;
  border-radius: 1rem;
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(148, 163, 184, 0.25);
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.sponsor-analytics__card--wide {
  grid-column: span 2;
}

.sponsor-analytics__label {
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(226, 232, 240, 0.75);
}

.sponsor-analytics__value {
  font-size: 1.4rem;
  font-weight: 700;
  color: #facc15;
}

.sponsor-analytics__hint {
  font-size: 0.8rem;
  color: rgba(226, 232, 240, 0.8);
}

.sponsor-analytics__chart h4 {
  margin: 0 0 0.75rem;
  font-size: 1rem;
  font-weight: 600;
}

.sponsor-chart {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.sponsor-chart__row {
  display: grid;
  grid-template-columns: minmax(140px, 200px) 1fr minmax(160px, 200px);
  gap: 1rem;
  align-items: center;
}

.sponsor-chart__label {
  font-size: 0.85rem;
  color: rgba(226, 232, 240, 0.8);
}

.sponsor-chart__bars {
  display: flex;
  gap: 0.5rem;
  height: 10px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.2);
  overflow: hidden;
}

.sponsor-chart__bar {
  height: 100%;
  border-radius: 999px;
  transition: width 0.3s ease;
}

.sponsor-chart__bar--seen {
  background: linear-gradient(135deg, #38bdf8, #22d3ee);
}

.sponsor-chart__bar--clicks {
  background: linear-gradient(135deg, #f97316, #fb7185);
}

.sponsor-chart__values {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  font-size: 0.85rem;
  color: rgba(226, 232, 240, 0.85);
}

@media (max-width: 768px) {
  .sponsor-analytics__grid {
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  }

  .sponsor-analytics__card--wide {
    grid-column: span 1;
  }

  .sponsor-chart__row {
    grid-template-columns: 1fr;
    gap: 0.5rem;
  }

  .sponsor-chart__values {
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .leaderboard-item {
    grid-template-columns: 56px minmax(0, 1fr);
  }

  .leaderboard-item .votes {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}

.selfie-admin-loader {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem 0;
  color: #475569;
}

.selfie-admin-loader .spinner {
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 999px;
  border: 2px solid rgba(148, 163, 184, 0.35);
  border-top-color: rgba(59, 130, 246, 0.9);
  animation: admin-selfie-spin 0.8s linear infinite;
}

@keyframes admin-selfie-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.selfie-admin-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1.5rem;
  margin-top: 1.5rem;
}

.selfie-admin-card {
  display: flex;
  flex-direction: column;
  border-radius: 1.5rem;
  border: 1px solid rgba(15, 23, 42, 0.12);
  overflow: hidden;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 18px 30px rgba(15, 23, 42, 0.12);
}

.selfie-admin-thumb {
  position: relative;
  width: 100%;
  padding-top: 65%;
  background: #0f172a;
  overflow: hidden;
}

.selfie-admin-thumb img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.selfie-admin-thumb--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.9rem;
  color: #64748b;
  background: rgba(226, 232, 240, 0.4);
}

.selfie-admin-body {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  padding: 1rem 1.25rem 1.25rem;
}

.selfie-admin-caption {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #0f172a;
}

.selfie-admin-meta {
  margin: 0;
  font-size: 0.85rem;
  color: #64748b;
}

.selfie-admin-status {
  margin: 0;
  font-size: 0.9rem;
  color: #1f2937;
}

.selfie-admin-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.selfie-admin-actions .btn {
  flex: 1 1 auto;
  min-width: 160px;
}

.history-tracking-events {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 0.75rem;
}

.history-tracking-events__item {
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 0.75rem;
  padding: 0.75rem;
  background: rgba(15, 23, 42, 0.02);
}

.history-tracking-events__head {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  color: #0f172a;
}

.history-tracking-events__meta {
  margin: 0.25rem 0 0;
  color: #475569;
  font-size: 0.82rem;
}

.history-tracking-events__chips {
  margin-top: 0.55rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.history-tracking-events__chip {
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: #fff;
  border-radius: 999px;
  padding: 0.2rem 0.6rem;
  font-size: 0.74rem;
  color: #334155;
}

.history-tracking-events__samples {
  margin: 0.6rem 0 0;
  padding-left: 1.1rem;
  color: #1e293b;
}

.history-tracking-events__samples code {
  font-size: 0.74rem;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>

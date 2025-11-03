<template>
  <div class="admin-portal-view">
    <Toast />

    <div v-if="!isAuthenticated" class="admin-portal-view__login">
      <Card class="admin-portal-view__login-card">
        <template #title>Area amministratore</template>
        <template #subtitle>Gestisci eventi, squadre e votazioni MVP</template>
        <form class="admin-portal-view__login-form" @submit.prevent="login">
          <div class="p-fluid p-formgrid p-grid">
            <div class="p-col-12">
              <label class="admin-field-label" for="admin-login-username">Username</label>
              <InputText
                id="admin-login-username"
                v-model.trim="loginForm.username"
                autocomplete="username"
                required
              />
            </div>
            <div class="p-col-12">
              <label class="admin-field-label" for="admin-login-password">Password</label>
              <Password
                id="admin-login-password"
                v-model="loginForm.password"
                :feedback="false"
                toggle-mask
                autocomplete="current-password"
                required
              />
            </div>
          </div>
          <div class="admin-portal-view__login-actions">
            <Button
              type="submit"
              label="Entra"
              icon="pi pi-sign-in"
              :loading="isLoggingIn"
              class="admin-portal-view__primary-button"
            />
          </div>
          <Message
            v-if="loginError"
            severity="error"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ loginError }}
          </Message>
        </form>
      </Card>
    </div>

    <AdminLayout
      v-else
      :tabs="availableTabs"
      :active-tab="section"
      :username="activeUsername"
      @tab-change="handleSectionChange"
      @logout="logout"
      @navigate-lottery="goToLottery"
    >
      <template #brand>
        <span class="admin-brand">
          <i class="pi pi-chart-bar admin-brand__icon" />
          WC MVP Control
        </span>
      </template>

      <template #alerts>
        <Message
          v-if="globalError"
          severity="error"
          :closable="false"
          class="admin-portal-view__inline-message"
        >
          {{ globalError }}
        </Message>
      </template>

      <template #toolbar-actions>
        <Tag
          v-if="activeEventEntry"
          value="Evento attivo"
          severity="info"
          icon="pi pi-bolt"
          class="admin-active-event-tag"
        />
      </template>

      <template #default>
        <!-- Events Section -->
        <Card v-if="section === 'events'" class="admin-section-card">
          <template #title>Eventi</template>
          <template #subtitle>Crea partite e gestisci gli eventi attivi.</template>

          <div class="admin-section__toolbar">
            <Button
              label="Disattiva eventi"
              icon="pi pi-power-off"
              severity="secondary"
              outlined
              :disabled="!activeEventId || isDisablingEvents"
              @click="deactivateEvents"
            />
          </div>

          <Message
            v-if="!hasEnoughTeams"
            severity="warn"
            icon="pi pi-users"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Aggiungi almeno due squadre nella sezione "Squadre" per creare un evento.
          </Message>

          <form class="admin-form admin-form--grid" @submit.prevent="createEvent">
            <div class="admin-form__field">
              <label class="admin-field-label" for="event-home-team">Squadra di casa</label>
              <AutoComplete
                id="event-home-team"
                v-model="teamInputs.home"
                :suggestions="teamSearchResults"
                :disabled="!hasEnoughTeams"
                placeholder="Seleziona o digita il nome della squadra"
                @complete="searchTeams"
              />
            </div>
            <div class="admin-form__field">
              <label class="admin-field-label" for="event-away-team">Squadra ospite</label>
              <AutoComplete
                id="event-away-team"
                v-model="teamInputs.away"
                :suggestions="teamSearchResults"
                :disabled="!hasEnoughTeams"
                placeholder="Seleziona o digita il nome della squadra"
                @complete="searchTeams"
              />
            </div>
            <div class="admin-form__field">
              <label class="admin-field-label" for="event-date">Data e ora</label>
              <InputText
                id="event-date"
                v-model="newEvent.start_datetime"
                type="datetime-local"
                :disabled="!hasEnoughTeams"
                required
              />
            </div>
            <div class="admin-form__field">
              <label class="admin-field-label" for="event-location">Location</label>
              <InputText
                id="event-location"
                v-model.trim="newEvent.location"
                placeholder="Es. Palazzetto dello Sport"
                :disabled="!hasEnoughTeams"
              />
            </div>
            <div class="admin-form__field admin-form__field--full">
              <Panel header="Esperienze post voto" toggleable collapsed-icon="pi pi-chevron-down" expanded-icon="pi pi-chevron-up">
                <div class="admin-toggle-grid">
                  <div class="admin-toggle">
                    <span class="admin-toggle__label">Andamento dei voti</span>
                    <InputSwitch v-model="newEvent.show_vote_trend" :disabled="!hasEnoughTeams" />
                  </div>
                  <div class="admin-toggle">
                    <span class="admin-toggle__label">Selfie MVP</span>
                    <InputSwitch v-model="newEvent.show_selfie" :disabled="!hasEnoughTeams" />
                  </div>
                  <div class="admin-toggle">
                    <span class="admin-toggle__label">Mini-gioco riflessi</span>
                    <InputSwitch v-model="newEvent.show_reaction_test" :disabled="!hasEnoughTeams" />
                  </div>
                  <div class="admin-toggle">
                    <span class="admin-toggle__label">Sondaggio feedback</span>
                    <InputSwitch v-model="newEvent.show_feedback_survey" :disabled="!hasEnoughTeams" />
                  </div>
                </div>
              </Panel>
            </div>
            <div class="admin-form__field admin-form__field--full">
              <Panel header="Premi in palio" toggleable collapsed-icon="pi pi-gift">
                <div class="admin-prize-grid">
                  <div
                    v-for="(prize, index) in newEventPrizes"
                    :key="`new-event-prize-${index}`"
                    class="admin-prize-grid__row"
                  >
                    <InputText
                      :placeholder="`Premio ${index + 1}`"
                      v-model.trim="prize.name"
                      :disabled="!hasEnoughTeams"
                    />
                    <Button
                      icon="pi pi-times"
                      severity="secondary"
                      text
                      rounded
                      :disabled="newEventPrizes.length <= 1"
                      @click="removeNewEventPrize(index)"
                    />
                  </div>
                </div>
                <div class="admin-prize-grid__actions">
                  <Button
                    type="button"
                    icon="pi pi-plus"
                    label="Aggiungi premio"
                    severity="secondary"
                    outlined
                    :disabled="!hasEnoughTeams"
                    @click="addNewEventPrize"
                  />
                </div>
              </Panel>
            </div>
            <div class="admin-form__actions">
              <Button
                type="submit"
                label="Crea evento"
                icon="pi pi-calendar-plus"
                :disabled="!hasEnoughTeams"
              />
            </div>
          </form>

          <Message
            v-if="lastCreatedEventLink"
            severity="success"
            class="admin-portal-view__inline-message"
            icon="pi pi-link"
            :closable="false"
          >
            <span class="admin-inline-link">
              <a :href="lastCreatedEventLink" target="_blank" rel="noopener">
                {{ lastCreatedEventLink }}
              </a>
              <Button
                type="button"
                icon="pi pi-copy"
                label="Copia"
                text
                class="admin-inline-link__copy"
                @click="copyLink(lastCreatedEventLink)"
              />
            </span>
          </Message>

          <Divider />

          <DataView
            :value="visibleEvents"
            data-key="id"
            layout="list"
            class="admin-events__list"
            empty-message="Nessun evento disponibile"
          >
            <template #list="{ items }">
              <div v-for="event in items" :key="event.id" class="admin-event-card">
                <Card>
                  <template #title>
                    <span class="admin-event-card__title">
                      {{ eventLabel(event) }}
                      <Tag
                        v-if="event.is_active"
                        value="Attivo"
                        severity="success"
                        icon="pi pi-play"
                        class="admin-event-card__tag"
                      />
                      <Tag
                        v-if="event.is_active && event.votes_closed"
                        value="Votazioni chiuse"
                        severity="warning"
                        icon="pi pi-lock"
                        class="admin-event-card__tag"
                      />
                    </span>
                  </template>
                  <template #subtitle>
                    <span class="admin-event-card__subtitle">
                      {{ formatEventDate(event.start_datetime) }} • {{ event.location || 'Location da definire' }}
                    </span>
                  </template>
                  <div class="admin-event-card__body">
                    <div class="admin-event-card__info">
                      <span class="admin-event-card__link">
                        Link voto:
                        <a :href="buildEventLink(event.id)" target="_blank" rel="noopener">
                          {{ buildEventLink(event.id) }}
                        </a>
                      </span>
                    </div>
                    <div class="admin-event-card__actions">
                      <Button
                        label="Attiva"
                        icon="pi pi-check"
                        severity="success"
                        :disabled="event.is_active || updatingEventId === event.id"
                        @click="activateEvent(event.id)"
                      />
                      <Button
                        label="Apri voto"
                        icon="pi pi-external-link"
                        severity="secondary"
                        outlined
                        @click="openVote(event.id)"
                      />
                      <Button
                        label="Evento terminato"
                        icon="pi pi-flag"
                        severity="warning"
                        outlined
                        :loading="concludingEventId === event.id"
                        @click="concludeEvent(event.id)"
                      />
                      <Button
                        label="Elimina"
                        icon="pi pi-trash"
                        severity="danger"
                        outlined
                        @click="deleteEvent(event.id)"
                      />
                    </div>

                    <Divider />

                    <div class="admin-event-card__postvote">
                      <h4>Esperienze post voto</h4>
                      <div class="admin-toggle-grid">
                        <div class="admin-toggle">
                          <span class="admin-toggle__label">Andamento dei voti</span>
                          <InputSwitch
                            v-model="event.show_vote_trend"
                            :disabled="isSavingPrizesFor(event.id)"
                          />
                        </div>
                        <div class="admin-toggle">
                          <span class="admin-toggle__label">Selfie MVP</span>
                          <InputSwitch
                            v-model="event.show_selfie"
                            :disabled="isSavingPrizesFor(event.id)"
                          />
                        </div>
                        <div class="admin-toggle">
                          <span class="admin-toggle__label">Mini-gioco riflessi</span>
                          <InputSwitch
                            v-model="event.show_reaction_test"
                            :disabled="isSavingPrizesFor(event.id)"
                          />
                        </div>
                        <div class="admin-toggle">
                          <span class="admin-toggle__label">Sondaggio feedback</span>
                          <InputSwitch
                            v-model="event.show_feedback_survey"
                            :disabled="isSavingPrizesFor(event.id)"
                          />
                        </div>
                      </div>
                    </div>

                    <div class="admin-event-card__prizes">
                      <h4>Premi in palio</h4>
                      <div class="admin-prize-grid">
                        <div
                          v-for="(prize, index) in prizeDraftsFor(event.id)"
                          :key="`event-${event.id}-prize-${prize.id || index}`"
                          class="admin-prize-grid__row"
                        >
                          <InputText
                            :placeholder="`Premio ${index + 1}`"
                            v-model="prize.name"
                            :disabled="isSavingPrizesFor(event.id)"
                          />
                          <Tag
                            v-if="prize.winner"
                            severity="success"
                            value="Assegnato"
                            icon="pi pi-ticket"
                          />
                          <Button
                            icon="pi pi-times"
                            severity="secondary"
                            text
                            rounded
                            :disabled="
                              prize.winner || prizeDraftsFor(event.id).length <= 1 || isSavingPrizesFor(event.id)
                            "
                            @click="removePrizeDraft(event.id, index)"
                          />
                        </div>
                      </div>
                      <div class="admin-prize-grid__actions">
                        <Button
                          type="button"
                          icon="pi pi-plus"
                          label="Aggiungi premio"
                          severity="secondary"
                          outlined
                          :disabled="isSavingPrizesFor(event.id)"
                          @click="addPrizeDraft(event.id)"
                        />
                        <Button
                          type="button"
                          icon="pi pi-save"
                          label="Salva impostazioni"
                          :loading="isSavingPrizesFor(event.id)"
                          @click="savePrizesForEvent(event)"
                        />
                      </div>
                      <Message
                        v-if="eventPrizeErrors[event.id]"
                        severity="error"
                        :closable="false"
                        class="admin-portal-view__inline-message"
                      >
                        {{ eventPrizeErrors[event.id] }}
                      </Message>
                    </div>
                  </div>
                </Card>
              </div>
            </template>
          </DataView>
        </Card>

        <!-- Closing Section -->
        <Card v-else-if="section === 'closing'" class="admin-section-card">
          <template #title>Chiusura votazioni</template>
          <template #subtitle>Controlla lo stato delle votazioni per l'evento attivo.</template>

          <div v-if="activeEventEntry" class="admin-status-card">
            <div class="admin-status-card__header">
              <div>
                <h3 class="admin-status-card__title">{{ activeEventLabel }}</h3>
                <span class="admin-status-card__subtitle">
                  {{ activeEventDateLabel }} • {{ activeEventLocation }}
                </span>
              </div>
              <Tag
                :value="activeEventVotesClosed ? 'Votazioni chiuse' : 'Votazioni aperte'"
                :severity="activeEventVotesClosed ? 'warning' : 'success'"
                icon="pi pi-clock"
              />
            </div>
            <div class="admin-status-card__actions">
              <Button
                label="Chiudi votazioni"
                icon="pi pi-lock"
                severity="warning"
                :loading="isClosingVotes"
                :disabled="isClosingVotes || activeEventVotesClosed"
                @click="closeActiveEventVoting"
              />
              <Button
                label="Riattiva"
                icon="pi pi-refresh"
                severity="success"
                :disabled="!activeEventEntry || updatingEventId === activeEventEntry.id || !activeEventVotesClosed"
                :loading="updatingEventId === activeEventEntry.id"
                @click="activateEvent(activeEventEntry.id)"
              />
              <Button
                label="Disattiva"
                icon="pi pi-power-off"
                severity="secondary"
                outlined
                :loading="isDisablingEvents"
                @click="deactivateEvents"
              />
            </div>
            <Message
              v-if="closeVotesMessage"
              severity="success"
              :closable="false"
              class="admin-portal-view__inline-message"
            >
              {{ closeVotesMessage }}
            </Message>
          </div>
          <Message
            v-else
            severity="info"
            icon="pi pi-info-circle"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Nessun evento attivo al momento. Attiva una partita dalla sezione "Eventi" per gestire le votazioni.
          </Message>
        </Card>

        <!-- Results Section -->
        <Card v-else-if="section === 'results'" class="admin-section-card">
          <template #title>Risultati votazioni</template>
          <template #subtitle>Monitora la classifica MVP e i dati sponsor in tempo reale.</template>

          <div class="admin-results__controls">
            <Dropdown
              v-model="selectedResultsEventId"
              :options="resultsEventOptions"
              option-label="label"
              option-value="value"
              placeholder="Seleziona un evento"
              :disabled="!resultsEventOptions.length"
              class="admin-results__dropdown"
            />
            <Button
              label="Aggiorna"
              icon="pi pi-refresh"
              severity="secondary"
              :loading="isLoadingResults"
              :disabled="isLoadingResults || !selectedResultsEventId"
              @click="fetchEventResults({ showLoader: true })"
            />
          </div>

          <div v-if="selectedResultsEvent" class="admin-results__summary">
            <h3>{{ selectedResultsEventLabel }}</h3>
            <span>{{ selectedResultsEventDate || 'Data da definire' }}</span>
          </div>

          <Message
            v-if="resultsError"
            severity="error"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ resultsError }}
          </Message>

          <Message
            v-else-if="!availableEvents.length"
            severity="info"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Crea un evento per visualizzare i risultati delle votazioni MVP.
          </Message>

          <div v-else class="admin-results__content">
            <div class="admin-results__meta">
              <span><strong>Voti totali:</strong> {{ totalVotes }}</span>
              <span v-if="lastResultsUpdateLabel">
                <strong>Ultimo aggiornamento:</strong> {{ lastResultsUpdateLabel }}
              </span>
              <span class="admin-results__auto">Aggiornamento automatico ogni 5 secondi</span>
            </div>
            <DataTable
              :value="resultsLeaderboard"
              data-key="id"
              scrollable
              scroll-height="400px"
              class="admin-results__table"
              striped-rows
            >
              <Column header="#" body-class="admin-results__rank" :body="rankTemplate" />
              <Column field="lastNameUpper" header="Cognome" />
              <Column field="firstName" header="Nome" />
              <Column field="votes" header="Voti">
                <template #body="{ data }">
                  <div class="admin-results__votes">
                    <strong>{{ data.votes }}</strong>
                    <span>{{ data.votes === 1 ? 'voto' : 'voti' }}</span>
                  </div>
                </template>
              </Column>
              <Column header="%">
                <template #body="{ data }">
                  <ProgressBar :value="data.percentage" show-value />
                </template>
              </Column>
            </DataTable>

            <Panel header="Analisi sponsor" toggleable>
              <Message
                v-if="sponsorAnalyticsError"
                severity="error"
                :closable="false"
                class="admin-portal-view__inline-message"
              >
                {{ sponsorAnalyticsError }}
              </Message>
              <Message
                v-else-if="isLoadingSponsorAnalytics"
                severity="info"
                icon="pi pi-spin pi-spinner"
                :closable="false"
                class="admin-portal-view__inline-message"
              >
                Caricamento dati sponsor…
              </Message>
              <Message
                v-else-if="!hasSponsorAnalyticsData"
                severity="warn"
                :closable="false"
                class="admin-portal-view__inline-message"
              >
                Nessun dato sponsor disponibile per questo evento.
              </Message>
              <div v-else class="admin-sponsor-analytics">
                <div class="admin-sponsor-analytics__grid">
                  <Card class="admin-sponsor-analytics__stat" v-for="card in sponsorAnalyticsCards" :key="card.label">
                    <template #title>{{ card.label }}</template>
                    <template #content>
                      <div class="admin-sponsor-analytics__value">{{ card.value }}</div>
                      <small v-if="card.hint" class="admin-sponsor-analytics__hint">{{ card.hint }}</small>
                    </template>
                  </Card>
                </div>
                <div v-if="sponsorChartRows.length" class="admin-sponsor-analytics__chart">
                  <h4>Andamento visualizzazioni e click</h4>
                  <DataTable :value="sponsorChartRows" scrollable scroll-height="260px">
                    <Column field="label" header="Intervallo" />
                    <Column header="Viste">
                      <template #body="{ data }">
                        <ProgressBar :value="data.seenPercent" show-value>
                          {{ data.seen.toLocaleString('it-IT') }} viste
                        </ProgressBar>
                      </template>
                    </Column>
                    <Column header="Click">
                      <template #body="{ data }">
                        <ProgressBar :value="data.clicksPercent" show-value severity="info">
                          {{ data.clicks.toLocaleString('it-IT') }} click
                        </ProgressBar>
                      </template>
                    </Column>
                  </DataTable>
                </div>
              </div>
            </Panel>
          </div>
        </Card>

        <!-- Selfies Section -->
        <Card v-else-if="section === 'selfies'" class="admin-section-card">
          <template #title>Selfie MVP</template>
          <template #subtitle>Modera i selfie inviati dai tifosi.</template>

          <div class="admin-form admin-form--compact">
            <div class="admin-form__field admin-form__field--medium">
              <label class="admin-field-label" for="selfie-event-select">Evento</label>
              <Dropdown
                id="selfie-event-select"
                v-model="selectedSelfieEventId"
                :options="eventDropdownOptions"
                option-label="label"
                option-value="value"
                placeholder="Seleziona evento"
                :disabled="!eventDropdownOptions.length"
              />
            </div>
          </div>

          <Message
            v-if="selfieModerationMessage"
            severity="success"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ selfieModerationMessage }}
          </Message>
          <Message
            v-if="selfieLoadError"
            severity="error"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ selfieLoadError }}
          </Message>

          <div v-if="isLoadingSelfies" class="admin-loader" role="status" aria-live="polite">
            <i class="pi pi-spin pi-spinner" aria-hidden="true" />
            <span>Caricamento selfie…</span>
          </div>
          <Message
            v-else-if="!availableEvents.length"
            severity="info"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Crea un evento per raccogliere selfie dal pubblico.
          </Message>
          <Message
            v-else-if="!eventSelfies.length"
            severity="warn"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Nessun selfie ricevuto per questo evento al momento.
          </Message>
          <div v-else class="admin-selfie-grid">
            <Card v-for="selfie in eventSelfies" :key="selfie.id" class="admin-selfie-card">
              <template #header>
                <div class="admin-selfie-card__thumb" :class="{ 'admin-selfie-card__thumb--empty': !selfie.image_src }">
                  <img v-if="selfie.image_src" :src="selfie.image_src" :alt="`Selfie ${selfie.id}`" />
                  <span v-else>Immagine non disponibile</span>
                </div>
              </template>
              <template #title>{{ selfie.caption || 'Senza didascalia' }}</template>
              <template #subtitle>
                Inviato: {{ formatSelfieDate(selfie.submitted_at) || 'N/D' }} • Device: {{ selfie.device_token || 'N/D' }}
              </template>
              <template #content>
                <p>Dimensione: {{ formatSelfieFileSize(selfie.file_size_bytes) || 'N/D' }}</p>
                <Tag :value="selfieStatusLabel(selfie)" severity="secondary" icon="pi pi-user" />
              </template>
              <template #footer>
                <Button
                  label="Elimina"
                  icon="pi pi-trash"
                  severity="danger"
                  outlined
                  :loading="isSelfieBusy(selfie.id)"
                  @click="deleteSelfie(selfie)"
                />
              </template>
            </Card>
          </div>
        </Card>

        <!-- History Section -->
        <Card v-else-if="section === 'history'" class="admin-section-card">
          <template #title>Storico eventi</template>
          <template #subtitle>Analizza prestazioni, voti e interazioni passate.</template>

          <div class="admin-section__toolbar">
            <Button
              label="Aggiorna"
              icon="pi pi-refresh"
              severity="secondary"
              :loading="isLoadingEventHistory"
              :disabled="isLoadingEventHistory"
              @click="refreshEventHistory"
            />
          </div>

          <Message
            v-if="eventHistorySuccess"
            severity="success"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ eventHistorySuccess }}
          </Message>
          <Message
            v-if="eventHistoryError"
            severity="error"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ eventHistoryError }}
          </Message>
          <Message
            v-else-if="isLoadingEventHistory"
            severity="info"
            icon="pi pi-spin pi-spinner"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Caricamento storico in corso…
          </Message>
          <Message
            v-else-if="!eventHistory.length"
            severity="warn"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Non sono presenti eventi conclusi al momento.
          </Message>

          <Accordion v-else multiple class="admin-history">
            <AccordionPanel v-for="entry in eventHistory" :key="entry.id">
              <template #header>
                <div class="admin-history__header">
                  <div>
                    <h3>{{ entry.title }}</h3>
                    <span>{{ formatHistoryDate(entry.startDatetime) }}<span v-if="entry.location"> • {{ entry.location }}</span></span>
                  </div>
                  <div class="admin-history__chips">
                    <Tag
                      :value="`${entry.totalVotesLabel} voti`"
                      icon="pi pi-users"
                      severity="info"
                    />
                    <Tag
                      :value="`${entry.totalVisitorsLabel} visitatori`"
                      icon="pi pi-compass"
                      severity="secondary"
                    />
                    <Tag
                      :value="`${entry.uniqueVisitorsLabel} unici`"
                      icon="pi pi-star"
                      severity="contrast"
                    />
                  </div>
                </div>
              </template>
              <div class="admin-history__content">
                <div class="admin-history__actions">
                  <Button
                    label="Scarica report"
                    icon="pi pi-download"
                    severity="secondary"
                    :loading="isDownloadingHistoryReport(entry.id)"
                    @click="downloadEventHistoryReport(entry)"
                  />
                  <Button
                    v-if="isSuperAdmin"
                    label="Elimina evento"
                    icon="pi pi-trash"
                    severity="danger"
                    outlined
                    @click="openPurgeDialog(entry)"
                  />
                </div>
                <div class="admin-history__grid">
                  <Card class="admin-history__panel">
                    <template #title>MVP</template>
                    <template #content>
                      <p v-if="entry.mvp">
                        {{ entry.mvp.name }} • {{ entry.mvp.votes.toLocaleString('it-IT') }} voti
                      </p>
                      <Message v-else severity="info" :closable="false">Nessun MVP assegnato.</Message>
                    </template>
                  </Card>
                  <Card class="admin-history__panel">
                    <template #title>Interazioni sponsor</template>
                    <template #content>
                      <div v-if="entry.sponsorAnalyticsHasData" class="admin-history__stats">
                        <div class="admin-history__stat" v-for="stat in sponsorHistoryCards(entry)" :key="stat.label">
                          <span class="admin-history__stat-label">{{ stat.label }}</span>
                          <strong class="admin-history__stat-value">{{ stat.value }}</strong>
                          <small v-if="stat.hint" class="admin-history__stat-hint">{{ stat.hint }}</small>
                        </div>
                      </div>
                      <Message v-else severity="warn" :closable="false">Nessun dato registrato.</Message>
                      <DataTable v-if="entry.sponsorClicks.length" :value="entry.sponsorClicks" size="small">
                        <Column field="name" header="Sponsor" />
                        <Column field="clicks" header="Click">
                          <template #body="{ data }">{{ data.clicks.toLocaleString('it-IT') }}</template>
                        </Column>
                      </DataTable>
                      <DataTable
                        v-if="entry.sponsorAnalyticsTimeline.length"
                        :value="entry.sponsorAnalyticsTimeline"
                        size="small"
                        class="admin-history__timeline"
                      >
                        <Column field="label" header="Intervallo" />
                        <Column header="Viste">
                          <template #body="{ data }">{{ data.seen.toLocaleString('it-IT') }}</template>
                        </Column>
                        <Column header="Guardate">
                          <template #body="{ data }">{{ data.watched.toLocaleString('it-IT') }}</template>
                        </Column>
                        <Column header="Click">
                          <template #body="{ data }">{{ data.clicks.toLocaleString('it-IT') }}</template>
                        </Column>
                      </DataTable>
                    </template>
                  </Card>
                  <Card class="admin-history__panel">
                    <template #title>Sondaggio feedback</template>
                    <template #content>
                      <div v-if="entry.feedbackSummary">
                        <p class="admin-history__stat">{{ entry.feedbackSummary.totalResponsesLabel }}</p>
                        <div
                          v-for="question in entry.feedbackSummary.questions"
                          :key="question.id"
                          class="admin-history__question"
                        >
                          <h4>{{ question.title }}</h4>
                          <DataTable :value="question.answers" size="small">
                            <Column field="label" header="Risposta" />
                            <Column field="countLabel" header="Totale" />
                            <Column field="percentLabel" header="Percentuale" />
                          </DataTable>
                        </div>
                        <div class="admin-history__question">
                          <h4>{{ entry.feedbackSummary.suggestionQuestion.title }}</h4>
                          <Message
                            v-if="!entry.feedbackSummary.suggestionQuestion.hasSuggestions"
                            severity="info"
                            :closable="false"
                          >
                            Nessun suggerimento inviato.
                          </Message>
                          <DataTable
                            v-else
                            :value="entry.feedbackSummary.suggestionQuestion.suggestions"
                            size="small"
                          >
                            <Column field="" header="Suggerimenti">
                              <template #body="{ data }">{{ data }}</template>
                            </Column>
                          </DataTable>
                        </div>
                      </div>
                      <Message v-else severity="info" :closable="false">Nessun feedback raccolto.</Message>
                    </template>
                  </Card>
                  <Card class="admin-history__panel">
                    <template #title>Estrazione premi</template>
                    <template #content>
                      <Tag
                        :value="entry.hasPrizeDraw ? 'Estrazione eseguita' : 'Estrazione non eseguita'"
                        :severity="entry.hasPrizeDraw ? 'success' : 'warn'"
                        icon="pi pi-gift"
                      />
                      <Message
                        v-if="!entry.prizes.length"
                        severity="info"
                        :closable="false"
                        class="admin-portal-view__inline-message"
                      >
                        Nessun premio configurato per l'evento.
                      </Message>
                      <DataTable v-else :value="entry.prizes" size="small">
                        <Column field="name" header="Premio" />
                        <Column header="Codice vincente">
                          <template #body="{ data }">
                            <span v-if="data.hasWinner">{{ data.winnerTicketCode }}</span>
                            <span v-else class="admin-text-muted">Nessun vincitore</span>
                          </template>
                        </Column>
                      </DataTable>
                    </template>
                  </Card>
                </div>
                <div class="admin-history__votes" v-if="entry.timeline.length">
                  <h4>Andamento voti</h4>
                  <VoteTrendChart
                    v-if="entry.timelineChart.points.length"
                    :points="entry.timelineChart.points"
                    :start-label="entry.timelineChart.startLabel"
                    :end-label="entry.timelineChart.endLabel"
                    accessible-label="Andamento dei voti ogni 15 minuti"
                  />
                  <Button
                    text
                    :label="entry.isTimelineExpanded ? 'Nascondi dettagli' : 'Visualizza altro'"
                    :icon="entry.isTimelineExpanded ? 'pi pi-chevron-up' : 'pi pi-chevron-down'"
                    @click="toggleHistoryTimeline(entry)"
                  />
                  <DataTable v-if="entry.isTimelineExpanded" :value="entry.timeline" size="small">
                    <Column field="rangeLabel" header="Intervallo" />
                    <Column field="votesLabel" header="Voti" />
                  </DataTable>
                </div>
              </div>
            </AccordionPanel>
          </Accordion>
        </Card>

        <!-- Teams Section -->
        <Card v-else-if="section === 'teams'" class="admin-section-card">
          <template #title>Squadre</template>
          <template #subtitle>Gestisci le squadre disponibili per gli eventi.</template>

          <form class="admin-form admin-form--inline" @submit.prevent="createTeam">
            <div class="admin-form__field admin-form__field--grow">
              <InputText v-model.trim="newTeamName" placeholder="Nome squadra" required />
            </div>
            <Button type="submit" label="Aggiungi" icon="pi pi-plus" />
          </form>

          <DataTable
            :value="teams"
            data-key="id"
            striped-rows
            class="admin-simple-table"
            empty-message="Nessuna squadra configurata"
          >
            <Column field="name" header="Nome" />
            <Column header="Azioni" body-class="admin-table-actions">
              <template #body="{ data }">
                <Button
                  label="Elimina"
                  icon="pi pi-trash"
                  severity="danger"
                  text
                  @click="deleteTeam(data.id)"
                />
              </template>
            </Column>
          </DataTable>
        </Card>

        <!-- Players Section -->
        <Card v-else-if="section === 'players'" class="admin-section-card">
          <template #title>Giocatori</template>
          <template #subtitle>Gestisci fino a {{ playerSlotCount }} giocatori mostrati nella pagina voto.</template>

          <Message
            v-if="!teams.length"
            severity="warn"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Aggiungi almeno una squadra per assegnare correttamente i giocatori salvati.
          </Message>
          <Message
            v-if="playerOverflow.length"
            severity="info"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            Sono presenti {{ playerOverflow.length }} giocatori aggiuntivi nel database. Verranno rimossi al prossimo salvataggio.
          </Message>

          <div class="admin-player-grid">
            <Card v-for="(slot, index) in playerSlots" :key="`player-slot-${index}`" class="admin-player-card">
              <template #title>Giocatore {{ index + 1 }}</template>
              <template #content>
                <div class="admin-form admin-form--compact">
                  <div class="admin-form__field">
                    <label class="admin-field-label" :for="`player-first-${index}`">Nome</label>
                    <InputText :id="`player-first-${index}`" v-model.trim="slot.first_name" placeholder="Es. Mario" />
                  </div>
                  <div class="admin-form__field">
                    <label class="admin-field-label" :for="`player-last-${index}`">Cognome</label>
                    <InputText :id="`player-last-${index}`" v-model.trim="slot.last_name" placeholder="Es. Rossi" />
                  </div>
                  <div class="admin-form__field">
                    <label class="admin-field-label" :for="`player-role-${index}`">Ruolo</label>
                    <InputText :id="`player-role-${index}`" v-model.trim="slot.role" placeholder="Es. Schiacciatore" />
                  </div>
                  <div class="admin-form__field">
                    <label class="admin-field-label" :for="`player-jersey-${index}`">Numero</label>
                    <InputNumber
                      :id="`player-jersey-${index}`"
                      v-model="slot.jersey_number"
                      :min="0"
                      show-buttons
                    />
                  </div>
                  <div class="admin-form__field">
                    <label class="admin-field-label" :for="`player-team-${index}`">Squadra</label>
                    <Dropdown
                      :id="`player-team-${index}`"
                      v-model="slot.team_id"
                      :options="teamDropdownOptions"
                      option-label="label"
                      option-value="value"
                      placeholder="Seleziona squadra"
                    />
                  </div>
                  <div class="admin-form__field admin-form__field--full">
                    <label class="admin-field-label" :for="`player-url-${index}`">URL immagine</label>
                    <InputText
                      :id="`player-url-${index}`"
                      v-model.trim="slot.image_url"
                      type="url"
                      placeholder="https://..."
                      @input="handlePlayerUrlChange(index)"
                    />
                  </div>
                  <div class="admin-form__field admin-form__field--full">
                    <FileUpload
                      mode="basic"
                      auto
                      accept="image/*"
                      choose-label="Carica immagine"
                      :custom-upload="true"
                      @uploader="(event) => handlePlayerFileUpload(index, event)"
                    />
                  </div>
                  <div v-if="slot.image_preview" class="admin-player-card__preview">
                    <img :src="slot.image_preview" alt="Anteprima giocatore" />
                    <Button
                      label="Rimuovi"
                      icon="pi pi-times"
                      text
                      severity="secondary"
                      @click="removePlayerImage(index)"
                    />
                  </div>
                </div>
              </template>
            </Card>
          </div>

          <div class="admin-player-actions">
            <Button
              label="Ripristina dati salvati"
              icon="pi pi-undo"
              severity="secondary"
              outlined
              :disabled="isSavingPlayers"
              @click="restorePlayerSlots"
            />
            <Button
              label="Salva giocatori"
              icon="pi pi-save"
              :loading="isSavingPlayers"
              @click="savePlayers"
            />
          </div>
          <Message
            v-if="playerSaveError"
            severity="error"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ playerSaveError }}
          </Message>
          <Message
            v-if="playerSaveMessage"
            severity="success"
            :closable="false"
            class="admin-portal-view__inline-message"
          >
            {{ playerSaveMessage }}
          </Message>
        </Card>

        <!-- Sponsors Section -->
        <Card v-else-if="section === 'sponsors'" class="admin-section-card">
          <template #title>Sponsor</template>
          <template #subtitle>Configura gli sponsor e il numero di loghi visibili.</template>

          <Panel header="Nuovo sponsor" toggleable>
            <div class="admin-form admin-form--grid">
              <div class="admin-form__field">
                <label class="admin-field-label" for="sponsor-name">Nome</label>
                <InputText id="sponsor-name" v-model.trim="newSponsor.name" placeholder="Nome sponsor" />
              </div>
              <div class="admin-form__field">
                <label class="admin-field-label" for="sponsor-link">Link</label>
                <InputText id="sponsor-link" v-model.trim="newSponsor.linkUrl" type="url" placeholder="https://..." />
              </div>
              <div class="admin-form__field admin-form__field--full">
                <label class="admin-field-label">Logo</label>
                <FileUpload
                  mode="basic"
                  auto
                  accept="image/*"
                  choose-label="Carica logo"
                  :custom-upload="true"
                  @uploader="handleNewSponsorUpload"
                />
              </div>
            </div>
            <div class="admin-form__actions">
              <Button
                label="Crea sponsor"
                icon="pi pi-plus"
                :loading="isCreatingSponsor"
                @click="createSponsor"
              />
            </div>
          </Panel>

          <Panel header="Sponsor visibili" toggleable>
            <div class="admin-form admin-form--inline">
              <div class="admin-form__field admin-form__field--grow">
                <InputNumber
                  v-model="desiredActiveSponsorCount"
                  :min="0"
                  :max="sponsorSliderMax"
                  show-buttons
                />
              </div>
              <Button
                label="Aggiorna visibilità"
                icon="pi pi-eye"
                :loading="isApplyingSponsorCount"
                @click="applyActiveSponsorCount"
              />
            </div>
          </Panel>

          <DataView
            :value="sortedSponsors()"
            data-key="id"
            layout="grid"
            :rows="2"
            paginator
            class="admin-sponsor-grid"
            empty-message="Nessuno sponsor configurato"
          >
            <template #grid="{ items }">
              <div class="p-grid">
                <div v-for="sponsor in items" :key="sponsor.id" class="p-col-12 p-md-6">
                  <Card class="admin-sponsor-card">
                    <template #title>
                      <div class="admin-sponsor-card__title">
                        {{ sponsor.name }}
                        <Tag
                          :value="sponsor.isActive ? 'Visibile' : 'Nascosto'"
                          :severity="sponsor.isActive ? 'success' : 'danger'"
                        />
                      </div>
                    </template>
                    <template #content>
                      <div class="admin-form admin-form--compact">
                        <div class="admin-form__field">
                          <label class="admin-field-label">Nome</label>
                          <InputText v-model.trim="sponsor.name" />
                        </div>
                        <div class="admin-form__field">
                          <label class="admin-field-label">Link</label>
                          <InputText v-model.trim="sponsor.linkUrl" type="url" />
                        </div>
                        <div class="admin-form__field admin-form__field--full">
                          <FileUpload
                            mode="basic"
                            auto
                            accept="image/*"
                            choose-label="Aggiorna logo"
                            :custom-upload="true"
                            @uploader="(event) => handleSponsorFileUpload(event, sponsor)"
                          />
                        </div>
                        <div class="admin-form__field admin-form__field--full">
                          <ToggleButton
                            v-model="sponsor.isActive"
                            on-label="Visibile"
                            off-label="Nascosto"
                            on-icon="pi pi-eye"
                            off-icon="pi pi-eye-slash"
                          />
                        </div>
                      </div>
                    </template>
                    <template #footer>
                      <div class="admin-sponsor-card__actions">
                        <Button
                          label="Salva"
                          icon="pi pi-save"
                          :loading="sponsorBeingUpdated === sponsor.id"
                          @click="updateSponsorEntry(sponsor)"
                        />
                        <Button
                          label="Elimina"
                          icon="pi pi-trash"
                          severity="danger"
                          outlined
                          :loading="sponsorBeingDeleted === sponsor.id"
                          @click="deleteSponsorEntry(sponsor.id)"
                        />
                      </div>
                    </template>
                  </Card>
                </div>
              </div>
            </template>
          </DataView>
        </Card>

        <!-- Admins Section -->
        <Card v-else-if="section === 'admins'" class="admin-section-card">
          <template #title>Admin</template>
          <template #subtitle>Gestisci gli account con accesso al portale.</template>

          <form class="admin-form admin-form--grid" @submit.prevent="createAdmin">
            <div class="admin-form__field">
              <label class="admin-field-label" for="admin-username">Username</label>
              <InputText id="admin-username" v-model.trim="newAdmin.username" required />
            </div>
            <div class="admin-form__field">
              <label class="admin-field-label" for="admin-password">Password</label>
              <Password id="admin-password" v-model="newAdmin.password" :feedback="false" required />
            </div>
            <div class="admin-form__field">
              <label class="admin-field-label" for="admin-role">Ruolo</label>
              <Dropdown
                id="admin-role"
                v-model="newAdmin.role"
                :options="adminRoleOptions"
                option-label="label"
                option-value="value"
                placeholder="Seleziona ruolo"
                required
              />
            </div>
            <div class="admin-form__actions">
              <Button type="submit" label="Crea admin" icon="pi pi-user-plus" />
            </div>
          </form>

          <DataTable
            :value="admins"
            data-key="id"
            striped-rows
            class="admin-simple-table"
            empty-message="Nessun admin configurato"
          >
            <Column field="username" header="Username" />
            <Column field="role" header="Ruolo" />
            <Column header="Azioni" body-class="admin-table-actions">
              <template #body="{ data }">
                <Button
                  label="Rimuovi"
                  icon="pi pi-trash"
                  severity="danger"
                  text
                  @click="deleteAdmin(data.id)"
                />
              </template>
            </Column>
          </DataTable>
        </Card>
      </template>
    </AdminLayout>

    <Dialog
      v-model:visible="purgeDialog.visible"
      modal
      header="Elimina evento"
      :style="{ width: '30rem' }"
      :draggable="false"
    >
      <p class="admin-dialog__lead">
        Stai per eliminare definitivamente l'evento <strong>{{ purgeDialog.event?.title }}</strong>.
      </p>
      <p>Questa operazione non può essere annullata.</p>
      <div class="admin-form">
        <div class="admin-form__field admin-form__field--full">
          <label class="admin-field-label" for="purge-password">Conferma con password super admin</label>
          <Password
            id="purge-password"
            v-model="purgeDialog.password"
            :feedback="false"
            toggle-mask
            placeholder="Password"
          />
        </div>
      </div>
      <Message
        v-if="purgeDialog.error"
        severity="error"
        :closable="false"
        class="admin-portal-view__inline-message"
      >
        {{ purgeDialog.error }}
      </Message>
      <template #footer>
        <Button
          label="Annulla"
          icon="pi pi-times"
          severity="secondary"
          outlined
          :disabled="purgeDialog.isSubmitting"
          @click="closePurgeDialog"
        />
        <Button
          label="Elimina definitivamente"
          icon="pi pi-trash"
          severity="danger"
          :loading="purgeDialog.isSubmitting"
          :disabled="!purgeDialog.password"
          @click="confirmPurge"
        />
      </template>
    </Dialog>
  </div>
</template>
<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import Toast from 'primevue/toast';
import Card from 'primevue/card';
import InputText from 'primevue/inputtext';
import Password from 'primevue/password';
import Message from 'primevue/message';
import Button from 'primevue/button';
import Tag from 'primevue/tag';
import AutoComplete from 'primevue/autocomplete';
import Panel from 'primevue/panel';
import InputSwitch from 'primevue/inputswitch';
import Divider from 'primevue/divider';
import DataView from 'primevue/dataview';
import Dropdown from 'primevue/dropdown';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import ProgressBar from 'primevue/progressbar';
import FileUpload from 'primevue/fileupload';
import InputNumber from 'primevue/inputnumber';
import ToggleButton from 'primevue/togglebutton';
import Accordion from 'primevue/accordion';
import AccordionPanel from 'primevue/accordionpanel';
import Dialog from 'primevue/dialog';
import { apiClient, resolveApiUrl } from '../../api';
import { PLAYER_LAYOUT } from '../../roster';
import VoteTrendChart from '../../components/VoteTrendChart.vue';
import AdminLayout from '../../layouts/AdminLayout.vue';

const basePath = import.meta.env.BASE_URL ?? '/';
const baseVoteUrl = new URL(basePath, window.location.origin);
const RESULTS_POLL_INTERVAL = 5000;
const historyDateFormatter = new Intl.DateTimeFormat('it-IT', {
  dateStyle: 'full',
  timeStyle: 'short',
});
const historyTimeFormatter = new Intl.DateTimeFormat('it-IT', {
  hour: '2-digit',
  minute: '2-digit',
});
const analyticsTimeFormatter = new Intl.DateTimeFormat('it-IT', {
  dateStyle: 'short',
  timeStyle: 'short',
});
const selfieDateFormatter = new Intl.DateTimeFormat('it-IT', {
  dateStyle: 'medium',
  timeStyle: 'short',
});

const feedbackSummaryQuestions = [
  {
    id: 'experience',
    title: "Com’è stata la tua esperienza di voto oggi?",
    answers: [
      { value: 'very_easy', label: 'Facilissima' },
      { value: 'easy', label: 'Abbastanza semplice' },
      { value: 'complex', label: 'Un po’ macchinosa' },
      { value: 'hard', label: 'Difficile' },
    ],
  },
  {
    id: 'team_spirit',
    title: 'Ti sei sentito parte della squadra mentre sceglievi l’MVP del pubblico?',
    answers: [
      { value: 'high', label: 'Sì, tantissimo!' },
      { value: 'medium', label: 'In parte' },
      { value: 'low', label: 'Non proprio' },
    ],
  },
  {
    id: 'perks_interest',
    title:
      'Immagina che la tua partecipazione ti permetta di vivere esperienze speciali o vantaggi come vero tifoso… ti piacerebbe?',
    answers: [
      { value: 'yes', label: 'Sì, assolutamente' },
      { value: 'maybe', label: 'Forse' },
      { value: 'no', label: 'No' },
    ],
  },
  {
    id: 'mini_games_interest',
    title:
      'Ti piacerebbe divertirti ancora di più con mini-giochi o sfide tra un set e l’altro per mettere alla prova i tuoi riflessi?',
    answers: [
      { value: 'super_excited', label: 'Sì, carichissimo!' },
      { value: 'maybe', label: 'Forse più avanti' },
      { value: 'no', label: 'No grazie' },
    ],
  },
];

const feedbackSummarySuggestion = {
  id: 'suggestions',
  title: 'Se potessi migliorare qualcosa, cosa ti piacerebbe aggiungere o cambiare?',
};

let resultsPollHandle = 0;

const toast = useToast();

const section = ref('events');
const tabs = [
  { id: 'events', label: 'Eventi', icon: 'pi pi-calendar' },
  { id: 'closing', label: 'Chiusura votazioni', icon: 'pi pi-lock' },
  { id: 'results', label: 'Risultati', icon: 'pi pi-chart-line' },
  { id: 'selfies', label: 'Selfie MVP', icon: 'pi pi-camera' },
  { id: 'history', label: 'Storico eventi', icon: 'pi pi-clock' },
  { id: 'teams', label: 'Squadre', icon: 'pi pi-users' },
  { id: 'players', label: 'Giocatori', icon: 'pi pi-user' },
  { id: 'sponsors', label: 'Sponsor', icon: 'pi pi-megaphone' },
  { id: 'admins', label: 'Admin', icon: 'pi pi-shield' },
];
const STAFF_TAB_IDS = new Set(['closing', 'results', 'history', 'selfies']);

const teams = ref([]);
const players = ref([]);
const events = ref([]);
const admins = ref([]);
const sponsors = ref([]);
const teamSearchResults = ref([]);
const eventHistory = ref([]);
const eventSelfies = ref([]);
const isLoadingEventHistory = ref(false);
const eventHistoryError = ref('');
const eventHistorySuccess = ref('');
const hasLoadedEventHistory = ref(false);
const isLoadingSelfies = ref(false);
const selfieLoadError = ref('');
const selfieModerationMessage = ref('');
const selectedSelfieEventId = ref(0);
const selfieBusyState = reactive({});
const historyReportDownloadState = reactive({});
const purgeDialog = reactive({
  visible: false,
  event: null,
  password: '',
  error: '',
  isSubmitting: false,
});
const updatingEventId = ref(0);
const concludingEventId = ref(0);
const isDisablingEvents = ref(false);
const selectedResultsEventId = ref(0);
const eventResults = ref([]);
const isLoadingResults = ref(false);
const resultsError = ref('');
const lastResultsUpdate = ref(null);
const sponsorAnalytics = ref(null);
const sponsorAnalyticsError = ref('');
const isLoadingSponsorAnalytics = ref(false);
const newTeamName = ref('');
const playerSlotCount = PLAYER_LAYOUT.length;

const PLAYER_IMAGE_MAX_WIDTH = 600;
const PLAYER_IMAGE_MAX_HEIGHT = 600;
const PLAYER_IMAGE_QUALITY = 0.75;

const createEmptyPlayerSlot = (teamId = 0) => ({
  id: 0,
  first_name: '',
  last_name: '',
  role: '',
  jersey_number: '',
  team_id: teamId,
  image_url: '',
  image_preview: '',
  _imageChangeToken: null,
});

const playerSlots = reactive(
  Array.from({ length: playerSlotCount }, () => createEmptyPlayerSlot()),
);
const playerOverflow = ref([]);
const isSavingPlayers = ref(false);
const playerSaveError = ref('');
const playerSaveMessage = ref('');
function createDefaultNewEventState() {
  return {
    team1_id: 0,
    team2_id: 0,
    start_datetime: '',
    location: '',
    show_reaction_test: true,
    show_selfie: true,
    show_vote_trend: true,
    show_feedback_survey: true,
  };
}

const newEvent = reactive(createDefaultNewEventState());
const newEventPrizes = ref([{ name: '' }]);
const teamInputs = reactive({
  home: '',
  away: '',
});
const newAdmin = reactive({
  username: '',
  password: '',
  role: '',
});
const adminRoleOptions = [
  { value: 'superadmin', label: 'Super admin' },
  { value: 'staff', label: 'Staff' },
];
const maxSponsors = 4;
const newSponsor = reactive({
  name: '',
  linkUrl: '',
  logoData: '',
  isActive: true,
});
const desiredActiveSponsorCount = ref(0);
const isCreatingSponsor = ref(false);
const sponsorBeingUpdated = ref(0);
const sponsorBeingDeleted = ref(0);
const isApplyingSponsorCount = ref(false);
const lastCreatedEventLink = ref('');
const isClosingVotes = ref(false);
const closeVotesMessage = ref('');
const eventPrizeDrafts = reactive({});
const eventPrizeErrors = reactive({});
const savingEventPrizes = ref(0);

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
  const jersey = typeof slot.jersey_number === 'number' ? slot.jersey_number.toString() : `${slot.jersey_number || ''}`;
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
  };
};

const loadImageFromDataUrl = (dataUrl) =>
  new Promise((resolve, reject) => {
    const image = new Image();
    image.decoding = 'async';
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error('Impossibile caricare l\'immagine selezionata.'));
    image.src = dataUrl;
  });

const toDataUrlSafely = (canvas, type, quality) => {
  try {
    if (typeof quality === 'number') {
      return canvas.toDataURL(type, quality);
    }
    return canvas.toDataURL(type);
  } catch (error) {
    console.warn('Impossibile convertire l\'immagine nel formato richiesto:', error);
    return '';
  }
};

const extractMimeType = (dataUrl) => {
  if (typeof dataUrl !== 'string') {
    return '';
  }
  const match = /^data:([^;]+);/i.exec(dataUrl);
  return match ? match[1] : '';
};

const optimizePlayerImage = async (file) => {
  const originalDataUrl = await readFileAsDataUrl(file);
  if (!originalDataUrl) {
    return '';
  }

  try {
    const image = await loadImageFromDataUrl(originalDataUrl);
    const { naturalWidth: width, naturalHeight: height } = image;
    if (!width || !height) {
      return originalDataUrl;
    }

    const scale = Math.min(1, PLAYER_IMAGE_MAX_WIDTH / width, PLAYER_IMAGE_MAX_HEIGHT / height);
    const targetWidth = Math.max(1, Math.round(width * scale));
    const targetHeight = Math.max(1, Math.round(height * scale));

    const canvas = document.createElement('canvas');
    canvas.width = targetWidth;
    canvas.height = targetHeight;

    const context = canvas.getContext('2d');
    if (!context) {
      return originalDataUrl;
    }

    context.drawImage(image, 0, 0, targetWidth, targetHeight);

    const originalType = extractMimeType(originalDataUrl);
    const candidateTypes = Array.from(
      new Set(['image/webp', 'image/jpeg', originalType].filter(Boolean)),
    );

    let bestDataUrl = originalDataUrl;
    let bestSize = originalDataUrl.length;

    candidateTypes.forEach((type) => {
      const quality = type === 'image/png' ? undefined : PLAYER_IMAGE_QUALITY;
      const candidate = toDataUrlSafely(canvas, type, quality);
      if (candidate && candidate.length < bestSize) {
        bestDataUrl = candidate;
        bestSize = candidate.length;
      }
    });

    return bestDataUrl;
  } catch (error) {
    console.warn('Impossibile ottimizzare l\'immagine del giocatore:', error);
    return originalDataUrl;
  }
};

const applyPlayerImageFile = async (index, file) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  if (!file) {
    slot.image_preview = slot.image_url || '';
    return;
  }
  const changeToken = Symbol('player-image-change');
  slot._imageChangeToken = changeToken;

  try {
    const optimizedDataUrl = await optimizePlayerImage(file);
    if (slot._imageChangeToken === changeToken && optimizedDataUrl) {
      slot.image_url = optimizedDataUrl;
      slot.image_preview = optimizedDataUrl;
    }
  } catch (error) {
    console.warn('Caricamento immagine giocatore non riuscito:', error);
  } finally {
    if (slot._imageChangeToken === changeToken) {
      slot._imageChangeToken = null;
    }
  }
};

const handlePlayerFileUpload = async (index, payload) => {
  const files = Array.isArray(payload?.files) ? payload.files : [];
  await applyPlayerImageFile(index, files[0] ?? null);
  payload?.options?.clear?.();
};

const handlePlayerUrlChange = (index) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  slot.image_preview = slot.image_url || '';
};

const removePlayerImage = (index) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  slot.image_url = '';
  slot.image_preview = '';
};

const normalizePlayerResponse = (item) => {
  const firstName = typeof item?.first_name === 'string' ? item.first_name.trim() : '';
  const lastName = typeof item?.last_name === 'string' ? item.last_name.trim() : '';
  const role = typeof item?.role === 'string' ? item.role.trim() : '';
  const jerseyRaw =
    typeof item?.jersey_number === 'number' ? item.jersey_number : Number(item?.jersey_number);
  const jerseyNumber = Number.isFinite(jerseyRaw) ? jerseyRaw : 0;
  const image = typeof item?.image_url === 'string' ? item.image_url.trim() : '';
  const team = Number(item?.team_id) || 0;
  return {
    id: Number(item?.id) || 0,
    first_name: firstName,
    last_name: lastName,
    role,
    jersey_number: jerseyNumber,
    image_url: image,
    team_id: team,
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
  const caption = typeof item?.caption === 'string' ? item.caption.trim() : '';
  const imageUrl = typeof item?.image_url === 'string' ? item.image_url.trim() : '';
  const approved = Boolean(item?.approved);
  const showOnScreen = Boolean(item?.show_on_screen);
  const deviceToken = typeof item?.device_token === 'string' ? item.device_token : '';
  const fileSize = Number(item?.file_size_bytes);
  const fileSizeBytes = Number.isFinite(fileSize) && fileSize >= 0 ? fileSize : 0;
  const submittedAt =
    typeof item?.submitted_at === 'string'
      ? item.submitted_at
      : typeof item?.created_at === 'string'
      ? item.created_at
      : '';
  return {
    id,
    event_id: eventId,
    caption,
    image_url: imageUrl,
    image_src: imageUrl ? resolveApiUrl(imageUrl) : '',
    content_type: typeof item?.content_type === 'string' ? item.content_type : '',
    approved,
    show_on_screen: showOnScreen,
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
  playerOverflow.value = sorted.length > playerSlotCount ? sorted.slice(playerSlotCount) : [];
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
        jersey_number: player.jersey_number ? player.jersey_number.toString() : '',
        team_id: player.team_id || fallback,
        image_url: player.image_url,
        image_preview: player.image_url || '',
      });
    } else if (slot) {
      resetPlayerSlot(slot);
    }
  }
  ensurePlayerSlotTeams();
};

const restorePlayerSlots = () => {
  applyPlayersToSlots();
  playerSaveError.value = '';
  playerSaveMessage.value = '';
};

const savePlayers = async () => {
  if (isSavingPlayers.value) {
    return;
  }
  playerSaveError.value = '';
  playerSaveMessage.value = '';

  const fallback = fallbackTeamId();
  const hasAnyContent = playerSlots.some((slot) => slotHasContent(slot));
  if (!fallback && hasAnyContent) {
    playerSaveError.value = 'Crea almeno una squadra e assegnala ai giocatori prima di salvare.';
    return;
  }

  isSavingPlayers.value = true;
  const handledIds = new Set();

  try {
    for (const slot of playerSlots) {
      const hasContent = slotHasContent(slot);
      if (hasContent) {
        const payload = normalizePlayerPayload(slot, fallback);
        if (!payload.first_name || !payload.last_name || !payload.role) {
          playerSaveError.value = 'Nome, cognome e ruolo sono obbligatori per ogni giocatore salvato.';
          isSavingPlayers.value = false;
          return;
        }
        if (!payload.team_id) {
          playerSaveError.value = 'Seleziona una squadra per ogni giocatore salvato.';
          isSavingPlayers.value = false;
          return;
        }

        if (slot.id) {
          await secureRequest(() => apiClient.put(`/players/${slot.id}`, payload, authHeaders.value));
          handledIds.add(slot.id);
        } else {
          const { data } = await secureRequest(() => apiClient.post('/players', payload, authHeaders.value));
          const createdId = Number(data?.id) || 0;
          if (createdId) {
            slot.id = createdId;
            handledIds.add(createdId);
          }
        }
      } else if (slot.id) {
        await secureRequest(() => apiClient.delete(`/players/${slot.id}`, authHeaders.value));
        handledIds.add(slot.id);
        resetPlayerSlot(slot);
      } else {
        resetPlayerSlot(slot);
      }
    }

    for (const player of players.value) {
      if (!handledIds.has(player.id)) {
        await secureRequest(() => apiClient.delete(`/players/${player.id}`, authHeaders.value));
        handledIds.add(player.id);
      }
    }

    await loadPlayers();
    playerSaveMessage.value = 'Giocatori salvati con successo.';
  } catch (error) {
    if (!playerSaveError.value) {
      playerSaveError.value = 'Si è verificato un errore durante il salvataggio dei giocatori. Riprova.';
    }
  } finally {
    isSavingPlayers.value = false;
  }
};

const hasEnoughTeams = computed(() => teams.value.length >= 2);
const availableEvents = computed(() => events.value.filter((event) => !event.is_concluded));
const visibleEvents = computed(() => availableEvents.value);
const resultsEventOptions = computed(() =>
  availableEvents.value.map((event) => ({ value: event.id, label: eventLabel(event) })),
);
const eventDropdownOptions = computed(() =>
  availableEvents.value.map((event) => ({
    value: event.id,
    label: `${eventLabel(event)} • ${formatEventDate(event.start_datetime)}`,
  })),
);
const teamDropdownOptions = computed(() => teams.value.map((team) => ({ value: team.id, label: team.name })));

const activeEventId = computed(() => {
  const activeEvent = events.value.find((event) => event.is_active);
  return activeEvent ? activeEvent.id : 0;
});
const activeSponsorCount = computed(() => sponsors.value.filter((item) => item.isActive).length);
const sponsorSliderMax = computed(() =>
  sponsors.value.length ? Math.min(maxSponsors, sponsors.value.length) : maxSponsors,
);
const selectedResultsEvent = computed(() =>
  availableEvents.value.find((event) => event.id === selectedResultsEventId.value) || null,
);
const activeEventEntry = computed(() =>
  events.value.find((event) => event.id === activeEventId.value) || null,
);
const selectedSelfieEvent = computed(() =>
  availableEvents.value.find((event) => event.id === selectedSelfieEventId.value) || null,
);
const selectedSelfieEventLabel = computed(() =>
  selectedSelfieEvent.value ? eventLabel(selectedSelfieEvent.value) : '',
);
const activeEventVotesClosed = computed(() => Boolean(activeEventEntry.value?.votes_closed));
const activeEventLabel = computed(() =>
  activeEventEntry.value ? eventLabel(activeEventEntry.value) : 'Nessun evento attivo',
);
const activeEventDateLabel = computed(() =>
  activeEventEntry.value ? formatEventDate(activeEventEntry.value.start_datetime) : '',
);
const activeEventLocation = computed(() =>
  activeEventEntry.value?.location?.trim() ? activeEventEntry.value.location.trim() : 'Location da definire',
);
const selectedResultsEventLabel = computed(() =>
  selectedResultsEvent.value ? eventLabel(selectedResultsEvent.value) : '',
);
const selectedResultsEventDate = computed(() =>
  selectedResultsEvent.value ? formatEventDate(selectedResultsEvent.value.start_datetime) : '',
);
const resultsLeaderboard = computed(() => {
  const aggregated = new Map(
    eventResults.value.map((item) => [
      Number(item.player_id) || 0,
      {
        votes: Number(item.votes) || 0,
        lastVoteAt: typeof item.last_vote_at === 'string' ? item.last_vote_at : '',
      },
    ]),
  );

  const entries = players.value.map((player) => {
    const stats = aggregated.get(player.id) || { votes: 0, lastVoteAt: '' };
    const firstName = player.first_name || '';
    const lastName = player.last_name || '';
    const fullName = `${firstName} ${lastName}`.trim() || `Giocatore ${player.id}`;
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
        lastName: '',
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
    percentage: highestVotes > 0 ? Math.round((entry.votes / highestVotes) * 100) : 0,
  }));
});

const rankTemplate = (slotProps) => slotProps.index + 1;

const sponsorAnalyticsDisplay = computed(() => {
  const data = sponsorAnalytics.value;
  if (!data) {
    return null;
  }

  return {
    totalUsers: data.totalUsers,
    totalUsersLabel: data.totalUsers.toLocaleString('it-IT'),
    seenUsers: data.seenUsers,
    seenUsersLabel: data.seenUsers.toLocaleString('it-IT'),
    seenRateLabel: `${formatPercent(data.seenRate)}%`,
    watchedUsers: data.watchedUsers,
    watchedUsersLabel: data.watchedUsers.toLocaleString('it-IT'),
    averageWatchTimeLabel: formatWatchDuration(data.averageWatchTimeMs),
    totalClicks: data.totalClicks,
    totalClicksLabel: data.totalClicks.toLocaleString('it-IT'),
    uniqueClickersLabel: data.uniqueClickers.toLocaleString('it-IT'),
    clickRateLabel: `${formatPercent(data.clickRate)}%`,
    topSponsorName: data.topSponsor?.name || 'Nessuno',
    topSponsorViewsLabel: data.topSponsor ? data.topSponsor.views.toLocaleString('it-IT') : '0',
  };
});

const sponsorAnalyticsCards = computed(() => {
  const display = sponsorAnalyticsDisplay.value;
  if (!display) {
    return [];
  }
  return [
    { label: 'Utenti totali', value: display.totalUsersLabel },
    { label: 'Sezione vista', value: display.seenRateLabel, hint: `${display.seenUsersLabel} utenti` },
    {
      label: 'Tempo medio visione',
      value: display.averageWatchTimeLabel,
      hint: `${display.watchedUsersLabel} utenti`,
    },
    {
      label: 'Click totali',
      value: display.totalClicksLabel,
      hint: `${display.clickRateLabel} • ${display.uniqueClickersLabel} utenti`,
    },
    { label: 'Top sponsor', value: display.topSponsorName, hint: display.topSponsorViewsLabel },
  ];
});

const sponsorHistoryCards = (entry) => {
  const display = entry?.sponsorAnalyticsDisplay;
  if (!display) {
    return [];
  }
  return [
    { label: 'Utenti totali', value: display.totalUsersLabel },
    { label: 'Sezione vista', value: display.seenRateLabel, hint: `${display.seenUsersLabel} utenti` },
    {
      label: 'Tempo medio visione',
      value: display.averageWatchTimeLabel,
      hint: `${display.watchedUsersLabel} utenti • ${display.totalWatchTimeLabel}`,
    },
    {
      label: 'Click totali',
      value: display.totalClicksLabel,
      hint: `${display.clickRateLabel} • ${display.uniqueClickersLabel} utenti`,
    },
    { label: 'Sponsor più visualizzato', value: display.topSponsorName, hint: display.topSponsorViewsLabel },
  ];
};

const sponsorTimelinePoints = computed(() => {
  if (!sponsorAnalytics.value || !Array.isArray(sponsorAnalytics.value.timeline)) {
    return [];
  }

  return sponsorAnalytics.value.timeline.map((item) => {
    const timestamp = typeof item.timestamp === 'string' ? item.timestamp : '';
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
  return points.reduce((max, point) => Math.max(max, point.seen, point.clicks), 1);
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
  const timelineLength = Array.isArray(data.timeline) ? data.timeline.length : 0;
  return Boolean(data.totalUsers || data.totalClicks || timelineLength);
});
const totalVotes = computed(() =>
  eventResults.value.reduce((sum, item) => sum + (Number(item.votes) || 0), 0),
);
const hasResultsVotes = computed(() => totalVotes.value > 0);
const lastResultsUpdateLabel = computed(() =>
  lastResultsUpdate.value ? lastResultsUpdate.value.toLocaleString('it-IT') : '',
);

const token = ref(localStorage.getItem('adminToken') || '');
const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const activeRole = ref(localStorage.getItem('adminRole') || '');
const isAuthenticated = computed(() => Boolean(token.value));
const isSuperAdmin = computed(() => activeRole.value === 'superadmin');
const availableTabs = computed(() => {
  if (isSuperAdmin.value) {
    return tabs;
  }
  return tabs.filter((tab) => STAFF_TAB_IDS.has(tab.id));
});

const loginForm = reactive({
  username: '',
  password: '',
});
const isLoggingIn = ref(false);
const loginError = ref('');
const globalError = ref('');

const authHeaders = computed(() => ({
  headers: {
    Authorization: token.value ? `Bearer ${token.value}` : '',
  },
}));

function resetNewEventPrizes() {
  newEventPrizes.value = [{ name: '' }];
}

function resetForms() {
  newTeamName.value = '';
  Object.assign(newEvent, createDefaultNewEventState());
  resetNewEventPrizes();
  teamInputs.home = '';
  teamInputs.away = '';
  Object.assign(newAdmin, { username: '', password: '', role: '' });
  resetNewSponsorForm();
  desiredActiveSponsorCount.value = Math.min(sponsorSliderMax.value, activeSponsorCount.value);
  restorePlayerSlots();
  playerSaveError.value = '';
  playerSaveMessage.value = '';
}

function ensureValidTeamSelection() {
  if (!hasEnoughTeams.value) {
    newEvent.team1_id = 0;
    newEvent.team2_id = 0;
    teamInputs.home = '';
    teamInputs.away = '';
    return;
  }

  const availableIds = new Set(teams.value.map((team) => team.id));

  if (!availableIds.has(newEvent.team1_id)) {
    newEvent.team1_id = 0;
    teamInputs.home = '';
  }

  if (
    !availableIds.has(newEvent.team2_id) ||
    (newEvent.team1_id !== 0 && newEvent.team1_id === newEvent.team2_id)
  ) {
    newEvent.team2_id = 0;
    teamInputs.away = '';
  }

  syncTeamInputsFromIds();
}

watch(teams, () => {
  ensureValidTeamSelection();
  ensurePlayerSlotTeams();
  teamSearchResults.value = teams.value.map((team) => teamOptionValue(team)).slice(0, 10);
});
watch(
  () => teamInputs.home,
  (value, oldValue) => {
    if (value !== oldValue) {
      handleTeamInput('home');
    }
  },
);
watch(
  () => teamInputs.away,
  (value, oldValue) => {
    if (value !== oldValue) {
      handleTeamInput('away');
    }
  },
);
watch(hasEnoughTeams, (enough) => {
  if (!enough) {
    newEvent.team1_id = 0;
    newEvent.team2_id = 0;
    teamInputs.home = '';
    teamInputs.away = '';
  }
});

watch(events, (value) => {
  ensureResultsSelection();
  ensureSelfieSelection();
  const editableEvents = Array.isArray(value)
    ? value.filter((event) => !event.is_concluded)
    : [];
  syncEventPrizeDrafts(editableEvents);
  if (section.value === 'results' && selectedResultsEventId.value) {
    fetchEventResults();
  }
});

watch(activeEventId, () => {
  closeVotesMessage.value = '';
});

watch(activeEventVotesClosed, (closed) => {
  if (!closed) {
    closeVotesMessage.value = '';
  }
});

function clearCollections() {
  teams.value = [];
  players.value = [];
  events.value = [];
  admins.value = [];
  sponsors.value = [];
  eventHistory.value = [];
  eventSelfies.value = [];
  hasLoadedEventHistory.value = false;
  eventHistoryError.value = '';
  eventHistorySuccess.value = '';
  isLoadingSelfies.value = false;
  selfieLoadError.value = '';
  selfieModerationMessage.value = '';
  selectedSelfieEventId.value = 0;
  resetAllPlayerSlots();
  playerOverflow.value = [];
  playerSaveError.value = '';
  playerSaveMessage.value = '';
  Object.keys(eventPrizeDrafts).forEach((key) => {
    delete eventPrizeDrafts[key];
  });
  Object.keys(eventPrizeErrors).forEach((key) => {
    delete eventPrizeErrors[key];
  });
  Object.keys(selfieBusyState).forEach((key) => {
    delete selfieBusyState[key];
  });
  Object.keys(historyReportDownloadState).forEach((key) => {
    delete historyReportDownloadState[key];
  });
  lastCreatedEventLink.value = '';
  resetNewEventPrizes();
  resetResultsState();
  sponsorAnalytics.value = null;
  sponsorAnalyticsError.value = '';
  isLoadingSponsorAnalytics.value = false;
}

function stopResultsPolling() {
  if (resultsPollHandle) {
    window.clearInterval(resultsPollHandle);
    resultsPollHandle = 0;
  }
}

function startResultsPolling() {
  stopResultsPolling();
  if (!selectedResultsEventId.value) {
    return;
  }
  resultsPollHandle = window.setInterval(() => {
    fetchEventResults().catch(() => {
      /* silent */
    });
  }, RESULTS_POLL_INTERVAL);
}

function resetResultsState() {
  stopResultsPolling();
  selectedResultsEventId.value = 0;
  eventResults.value = [];
  resultsError.value = '';
  lastResultsUpdate.value = null;
  isLoadingResults.value = false;
  sponsorAnalytics.value = null;
  sponsorAnalyticsError.value = '';
  isLoadingSponsorAnalytics.value = false;
}

function ensureResultsSelection() {
  const available = availableEvents.value;
  if (!available.length) {
    selectedResultsEventId.value = 0;
    return;
  }
  const exists = available.some((event) => event.id === selectedResultsEventId.value);
  if (!exists) {
    const active = available.find((event) => event.is_active);
    selectedResultsEventId.value = active ? active.id : available[0].id;
  }
}

async function fetchEventResults({ showLoader = false } = {}) {
  if (!selectedResultsEventId.value) {
    eventResults.value = [];
    resultsError.value = '';
    lastResultsUpdate.value = null;
    return;
  }
  if (showLoader) {
    isLoadingResults.value = true;
  }
  resultsError.value = '';
  try {
    const { data } = await secureRequest(() =>
      apiClient.get(`/events/${selectedResultsEventId.value}/results`, authHeaders.value),
    );
    if (Array.isArray(data)) {
      eventResults.value = data.map((item) => ({
        player_id: Number(item.player_id) || 0,
        votes: Number(item.votes) || 0,
        last_vote_at: typeof item.last_vote_at === 'string' ? item.last_vote_at : '',
      }));
    } else {
      eventResults.value = [];
    }
    lastResultsUpdate.value = new Date();
  } catch (error) {
    if (error?.response?.status === 404) {
      resultsError.value = 'Evento non trovato.';
    } else if (error?.response?.status === 400) {
      resultsError.value = 'Richiesta non valida per i risultati.';
    } else if (error?.response?.status !== 401) {
      resultsError.value = 'Impossibile caricare i risultati. Riprova più tardi.';
    }
  } finally {
    if (showLoader) {
      isLoadingResults.value = false;
    }
  }

  fetchSponsorAnalytics({ showLoader }).catch(() => {});
}

function normalizeSponsorAnalyticsResponse(raw) {
  if (!raw || typeof raw !== 'object') {
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
  if (topSponsorRaw && typeof topSponsorRaw === 'object') {
    const id = resolveNumber(topSponsorRaw.sponsor_id ?? topSponsorRaw.sponsorId);
    const name = typeof topSponsorRaw.name === 'string' ? topSponsorRaw.name : '';
    const views = resolveNumber(topSponsorRaw.views);
    topSponsor = { id, name, views };
  }

  const timeline = Array.isArray(raw.timeline)
    ? raw.timeline.map((item) => ({
        timestamp: typeof item?.timestamp === 'string' ? item.timestamp : '',
        seen: resolveNumber(item?.seen),
        watched: resolveNumber(item?.watched),
        clicks: resolveNumber(item?.clicks),
      }))
    : [];

  return {
    totalUsers: resolveNumber(raw.total_users ?? raw.totalUsers),
    seenUsers: resolveNumber(raw.seen_users ?? raw.seenUsers),
    watchedUsers: resolveNumber(raw.watched_users ?? raw.watchedUsers),
    averageWatchTimeMs: resolveNumber(raw.average_watch_time_ms ?? raw.averageWatchTimeMs),
    totalWatchTimeMs: resolveNumber(raw.total_watch_time_ms ?? raw.totalWatchTimeMs),
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
    sponsorAnalyticsError.value = '';
    return;
  }

  if (showLoader) {
    isLoadingSponsorAnalytics.value = true;
  }
  sponsorAnalyticsError.value = '';

  try {
    const { data } = await secureRequest(() =>
      apiClient.get(`/admin/events/${selectedResultsEventId.value}/sponsors/analytics`, authHeaders.value),
    );
    sponsorAnalytics.value = normalizeSponsorAnalyticsResponse(data);
  } catch (error) {
    if (error?.response?.status === 404) {
      sponsorAnalytics.value = null;
      sponsorAnalyticsError.value = 'Nessun dato sponsor disponibile per questo evento.';
    } else if (error?.response?.status !== 401) {
      sponsorAnalyticsError.value = 'Impossibile caricare le statistiche sponsor.';
    }
    throw error;
  } finally {
    if (showLoader) {
      isLoadingSponsorAnalytics.value = false;
    }
  }
}

function normalizePrizeResponse(prize, index = 0) {
  if (!prize || typeof prize !== 'object') {
    return null;
  }
  const winner = prize.winner && typeof prize.winner === 'object' ? prize.winner : null;
  const normalizedWinner = winner
    ? {
        voteId: Number(winner.vote_id ?? winner.voteId) || 0,
        ticketCode: typeof (winner.ticket_code ?? winner.ticketCode) === 'string'
          ? (winner.ticket_code ?? winner.ticketCode)
          : '',
        playerId: Number(winner.player_id ?? winner.playerId) || 0,
        playerFirstName:
          typeof (winner.player_first_name ?? winner.playerFirstName) === 'string'
            ? (winner.player_first_name ?? winner.playerFirstName)
            : '',
        playerLastName:
          typeof (winner.player_last_name ?? winner.playerLastName) === 'string'
            ? (winner.player_last_name ?? winner.playerLastName)
            : '',
        assignedAt:
          typeof (winner.assigned_at ?? winner.assignedAt) === 'string'
            ? (winner.assigned_at ?? winner.assignedAt)
            : '',
      }
    : null;

  const position = Number(prize.position) || index + 1;
  return {
    id: Number(prize.id) || 0,
    eventId: Number(prize.event_id ?? prize.eventId) || 0,
    name: typeof prize.name === 'string' ? prize.name : '',
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
    if (!event || typeof event !== 'object') {
      return fallback;
    }
    for (const key of keys) {
      if (Object.prototype.hasOwnProperty.call(event, key)) {
        return Boolean(event[key]);
      }
    }
    return fallback;
  };
  normalized.show_reaction_test = resolveFlag(['show_reaction_test', 'showReactionTest'], true);
  normalized.show_selfie = resolveFlag(['show_selfie', 'showSelfie'], true);
  normalized.show_vote_trend = resolveFlag(
    ['show_vote_trend', 'showVoteTrend', 'show_live_results'],
    true,
  );
  normalized.show_feedback_survey = resolveFlag(
    ['show_feedback_survey', 'showFeedbackSurvey'],
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
  return normalized;
}

function normalizeSponsorResponse(item) {
  if (!item || typeof item !== 'object') {
    return null;
  }
  const normalizedName = typeof item.name === 'string' ? item.name.trim() : '';
  const normalizedLink = typeof item.link_url === 'string' ? item.link_url.trim() : '';
  return {
    id: Number(item.id) || 0,
    name: normalizedName,
    linkUrl: normalizedLink,
    position: Number(item.position) || 0,
    logoData: typeof item.logo_data === 'string' ? item.logo_data : '',
    isActive: Boolean(item.is_active),
  };
}

const toCamelCaseKey = (key) => {
  if (typeof key !== 'string' || !key.includes('_')) {
    return key;
  }
  return key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
};

function normalizeFeedbackSummary(raw) {
  if (!raw || typeof raw !== 'object') {
    return null;
  }

  const totalRaw = Number(raw.total_responses ?? raw.totalResponses ?? 0);
  const totalResponses = Number.isFinite(totalRaw) ? totalRaw : 0;
  const hasResponses = totalResponses > 0;
  const totalResponsesLabel =
    totalResponses === 1
      ? '1 risposta'
      : `${totalResponses.toLocaleString('it-IT')} risposte`;

  const questions = feedbackSummaryQuestions.map((question) => {
    const camelKey = toCamelCaseKey(question.id);
    const countsSource =
      (raw[question.id] && typeof raw[question.id] === 'object' ? raw[question.id] : null) ??
      (raw[camelKey] && typeof raw[camelKey] === 'object' ? raw[camelKey] : null);
    const counts = countsSource && typeof countsSource === 'object' ? countsSource : {};

    const answers = question.answers.map((option) => {
      const resolved = Number(counts?.[option.value] ?? 0);
      const count = Number.isFinite(resolved) ? resolved : 0;
      const percent = hasResponses && totalResponses > 0 ? Math.round((count / totalResponses) * 100) : 0;
      const clampedPercent = Math.min(100, Math.max(0, percent));
      const barPercent = hasResponses ? Math.max(clampedPercent, count > 0 ? 6 : 0) : 0;
      return {
        value: option.value,
        label: option.label,
        count,
        countLabel: count.toLocaleString('it-IT'),
        percent: clampedPercent,
        percentLabel: `${clampedPercent}%`,
        barWidth: `${barPercent}%`,
        hasCount: count > 0,
      };
    });

    const questionTotal = answers.reduce((sum, answer) => sum + answer.count, 0);
    return {
      id: question.id,
      title: question.title,
      answers,
      totalCount: questionTotal,
      totalCountLabel:
        questionTotal === 1
          ? '1 risposta'
          : `${questionTotal.toLocaleString('it-IT')} risposte`,
      hasAnswers: answers.some((answer) => answer.count > 0),
    };
  });

  const suggestionsSource = Array.isArray(raw.suggestions)
    ? raw.suggestions
    : Array.isArray(raw.suggestion)
    ? raw.suggestion
    : [];
  const suggestions = suggestionsSource
    .map((value) => (typeof value === 'string' ? value.trim() : ''))
    .filter(Boolean);

  const suggestionQuestion = {
    id: feedbackSummarySuggestion.id,
    title: feedbackSummarySuggestion.title,
    suggestions,
    hasSuggestions: suggestions.length > 0,
  };

  const hasAnyData = hasResponses || questions.some((question) => question.hasAnswers) || suggestionQuestion.hasSuggestions;

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
  desiredActiveSponsorCount.value = Math.min(sponsorSliderMax.value, activeSponsorCount.value);
}

function resetNewSponsorForm() {
  Object.assign(newSponsor, { name: '', linkUrl: '', logoData: '', isActive: true });
}

async function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      resolve(typeof reader.result === 'string' ? reader.result : '');
    };
    reader.onerror = () => {
      reject(reader.error || new Error('Impossibile leggere il file'));
    };
    reader.readAsDataURL(file);
  });
}

async function applySponsorLogoFile(file, targetSponsor) {
  if (!file) {
    return;
  }
  globalError.value = '';
  try {
    const dataUrl = await readFileAsDataUrl(file);
    if (dataUrl) {
      targetSponsor.logoData = dataUrl;
    }
  } catch (error) {
    console.error('Errore caricamento logo sponsor', error);
    globalError.value = 'Impossibile caricare il logo selezionato.';
  }
}

async function handleSponsorFileUpload(event, targetSponsor) {
  const files = Array.isArray(event?.files) ? event.files : [];
  if (!files.length) {
    event?.options?.clear?.();
    return;
  }
  try {
    await applySponsorLogoFile(files[0], targetSponsor);
  } finally {
    event?.options?.clear?.();
  }
}

async function handleNewSponsorUpload(event) {
  await handleSponsorFileUpload(event, newSponsor);
}

function buildEventLink(eventId) {
  const url = new URL(baseVoteUrl.toString());
  if (eventId) {
    url.searchParams.set('eventId', String(eventId));
  } else {
    url.searchParams.delete('eventId');
  }
  return url.toString();
}

function goToLottery() {
  const target = new URL(basePath || '/', window.location.origin);
  if (!target.pathname.endsWith('/')) {
    target.pathname = `${target.pathname}/`;
  }
  target.pathname = `${target.pathname.replace(/\/+$/, '')}/admin/lottery`;
  window.location.href = target.toString();
}

function teamOptionValue(team) {
  return `${team.name} (#${team.id})`;
}

function searchTeams(event) {
  const query = typeof event?.query === 'string' ? event.query.toLowerCase().trim() : '';
  const options = teams.value.map((team) => teamOptionValue(team));
  if (!query) {
    teamSearchResults.value = options.slice(0, 10);
    return;
  }
  teamSearchResults.value = options
    .filter((option) => option.toLowerCase().includes(query))
    .slice(0, 10);
}

function syncTeamInputsFromIds() {
  const homeTeam = teams.value.find((team) => team.id === newEvent.team1_id);
  const awayTeam = teams.value.find((team) => team.id === newEvent.team2_id);
  teamInputs.home = homeTeam ? teamOptionValue(homeTeam) : '';
  teamInputs.away = awayTeam ? teamOptionValue(awayTeam) : '';
}

function findTeamFromInput(value) {
  const normalized = value.trim().toLowerCase();
  if (!normalized) {
    return undefined;
  }
  return (
    teams.value.find((team) => teamOptionValue(team).toLowerCase() === normalized) ||
    teams.value.find((team) => team.name.trim().toLowerCase() === normalized)
  );
}

function handleTeamInput(position) {
  const key = position === 'home' ? 'team1_id' : 'team2_id';
  const otherKey = position === 'home' ? 'team2_id' : 'team1_id';
  const otherInputKey = position === 'home' ? 'away' : 'home';
  const rawValue = teamInputs[position] || '';
  const matchedTeam = findTeamFromInput(rawValue);

  if (matchedTeam) {
    if (newEvent[otherKey] === matchedTeam.id) {
      newEvent[otherKey] = 0;
      teamInputs[otherInputKey] = '';
    }
    newEvent[key] = matchedTeam.id;
    teamInputs[position] = teamOptionValue(matchedTeam);
  } else {
    newEvent[key] = 0;
    teamInputs[position] = '';
  }
}

function addNewEventPrize() {
  newEventPrizes.value = [...newEventPrizes.value, { name: '' }];
}

function removeNewEventPrize(index) {
  if (newEventPrizes.value.length <= 1) {
    return;
  }
  const updated = newEventPrizes.value.filter((_, idx) => idx !== index);
  newEventPrizes.value = updated.length ? updated : [{ name: '' }];
}

function prizeDraftsFor(eventId) {
  const drafts = eventPrizeDrafts[eventId];
  if (!Array.isArray(drafts) || drafts.length === 0) {
    eventPrizeDrafts[eventId] = [{ id: 0, name: '', position: 1, winner: null }];
  }
  return eventPrizeDrafts[eventId];
}

function addPrizeDraft(eventId) {
  const drafts = prizeDraftsFor(eventId);
  const updated = drafts.slice();
  updated.push({ id: 0, name: '', position: updated.length + 1, winner: null });
  eventPrizeDrafts[eventId] = updated;
  eventPrizeErrors[eventId] = '';
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
    ? updated.map((item, positionIndex) => ({ ...item, position: positionIndex + 1 }))
    : [{ id: 0, name: '', position: 1, winner: null }];
}

function isSavingPrizesFor(eventId) {
  return savingEventPrizes.value === eventId;
}

function prizeWinnerLabel(prize) {
  if (!prize || !prize.winner) {
    return '';
  }
  return prize.winner.ticketCode || '';
}

async function savePrizesForEvent(event) {
  if (!event || !event.id || isSavingPrizesFor(event.id)) {
    return;
  }

  const drafts = prizeDraftsFor(event.id);
  const sanitized = drafts
    .map((prize, index) => ({
      id: Number(prize.id) || 0,
      name: (prize.name || '').trim(),
      position: index + 1,
    }))
    .filter((prize) => prize.name);

  eventPrizeErrors[event.id] = '';

  const payload = {
    team1_id: event.team1_id,
    team2_id: event.team2_id,
    start_datetime: event.start_datetime,
    location: event.location,
    show_reaction_test: Boolean(event.show_reaction_test),
    show_selfie: Boolean(event.show_selfie),
    show_vote_trend: Boolean(event.show_vote_trend),
    show_feedback_survey: Boolean(event.show_feedback_survey),
    prizes: sanitized,
  };

  savingEventPrizes.value = event.id;
  try {
    await secureRequest(() => apiClient.put(`/events/${event.id}`, payload, authHeaders.value));
    await loadEvents();
  } catch (error) {
    if (error?.response?.status === 409) {
      eventPrizeErrors[event.id] =
        "Non puoi rimuovere un premio già assegnato. Annulla l'assegnazione dalla lotteria prima di modificarlo.";
    } else if (error?.response?.status === 400) {
      eventPrizeErrors[event.id] = 'Controlla i nomi dei premi e riprova.';
    } else if (error?.response?.status !== 401) {
      eventPrizeErrors[event.id] = 'Impossibile salvare i premi. Riprova più tardi.';
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
    const drafts = Array.isArray(event.prizes) && event.prizes.length
      ? event.prizes.map((prize, index) => ({
          id: prize.id,
          name: prize.name || '',
          position: prize.position || index + 1,
          winner: prize.winner
            ? {
                voteId: prize.winner.voteId,
                ticketCode: prize.winner.ticketCode,
                playerFirstName: prize.winner.playerFirstName,
                playerLastName: prize.winner.playerLastName,
              }
            : null,
        }))
      : [{ id: 0, name: '', position: 1, winner: null }];
    eventPrizeDrafts[event.id] = drafts;
  });
}

function eventLabel(event) {
  return `${resolveEventTeamName(event, 'team1')} vs ${resolveEventTeamName(event, 'team2')}`;
}

function resolveEventTeamName(event, teamKey) {
  const idKey = `${teamKey}_id`;
  const nameFromTeams = teamName(event?.[idKey]);
  if (nameFromTeams && nameFromTeams !== '—') {
    return nameFromTeams;
  }

  const fallbackKeys = [`${teamKey}_name`, `${teamKey}Name`];
  for (const key of fallbackKeys) {
    const value = event?.[key];
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
  }

  return '—';
}

function teamName(id) {
  const team = teams.value.find((teamItem) => teamItem.id === id);
  return team ? team.name : '—';
}

function formatEventDate(value) {
  if (!value) {
    return 'Data da definire';
  }
  const date = new Date(value);
  if (!Number.isNaN(date.valueOf())) {
    return date.toLocaleString('it-IT');
  }
  return value.replace('T', ' ');
}

function formatWatchDuration(ms) {
  const value = Number(ms);
  if (!Number.isFinite(value) || value <= 0) {
    return '0 s';
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

function formatPercent(value, minimumFractionDigits = 1, maximumFractionDigits = 1) {
  if (!Number.isFinite(value)) {
    return '0,0';
  }
  return value.toLocaleString('it-IT', {
    minimumFractionDigits,
    maximumFractionDigits,
  });
}

async function login() {
  if (isLoggingIn.value) {
    return;
  }
  loginError.value = '';
  globalError.value = '';
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post('/admin/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data.token;
    activeUsername.value = data.username;
    activeRole.value = data.role || '';
    localStorage.setItem('adminToken', token.value);
    localStorage.setItem('adminUsername', activeUsername.value);
    localStorage.setItem('adminRole', activeRole.value);
    loginForm.username = '';
    loginForm.password = '';
    await loadAll();
  } catch (error) {
    if (error?.response?.status === 401) {
      loginError.value = 'Credenziali non valide.';
    } else {
      loginError.value = 'Impossibile completare l\'accesso. Riprova.';
    }
  } finally {
    isLoggingIn.value = false;
  }
}

function logout() {
  token.value = '';
  activeUsername.value = '';
  activeRole.value = '';
  localStorage.removeItem('adminToken');
  localStorage.removeItem('adminUsername');
  localStorage.removeItem('adminRole');
  section.value = 'events';
  clearCollections();
}

function handleUnauthorized() {
  logout();
  loginError.value = 'Sessione scaduta. Effettua di nuovo il login.';
}

async function secureRequest(executor) {
  try {
    return await executor();
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else {
      globalError.value = 'Si è verificato un errore imprevisto. Riprova più tardi.';
    }
    throw error;
  }
}

async function loadTeams() {
  const { data } = await secureRequest(() => apiClient.get('/teams', authHeaders.value));
  teams.value = data;
  ensureValidTeamSelection();
}

async function loadPlayers() {
  const { data } = await secureRequest(() => apiClient.get('/players', authHeaders.value));
  const normalized = Array.isArray(data)
    ? data.map((item) => normalizePlayerResponse(item))
    : [];
  players.value = normalized;
  applyPlayersToSlots();
}

async function loadEvents() {
  const { data } = await secureRequest(() => apiClient.get('/events', authHeaders.value));
  const normalized = Array.isArray(data)
    ? data.map((event) => normalizeEventResponse(event)).filter(Boolean)
    : [];
  events.value = normalized;
  hasLoadedEventHistory.value = false;
}

async function loadAdmins() {
  const { data } = await secureRequest(() => apiClient.get('/admins', authHeaders.value));
  admins.value = data;
}

async function loadSponsors() {
  const { data } = await secureRequest(() => apiClient.get('/admin/sponsors', authHeaders.value));
  const normalized = Array.isArray(data)
    ? data
        .map((item) => normalizeSponsorResponse(item))
        .filter((item) => item && item.id)
        .sort((a, b) => a.position - b.position)
    : [];
  sponsors.value = normalized;
  recomputeActiveSponsorSlider();
}

async function loadEventSelfies(eventId) {
  if (!eventId) {
    eventSelfies.value = [];
    return;
  }
  isLoadingSelfies.value = true;
  selfieLoadError.value = '';
  selfieModerationMessage.value = '';
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
      selfieLoadError.value = 'Impossibile caricare i selfie per questo evento.';
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

function selfieStatusLabel(selfie) {
  if (!selfie) {
    return '';
  }
  if (!selfie.approved) {
    return 'In attesa di approvazione';
  }
  if (selfie.show_on_screen) {
    return 'Approvato per il maxischermo';
  }
  return 'Approvato';
}

function formatSelfieDate(value) {
  if (!value) {
    return '';
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
    return '';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let unitIndex = 0;
  let display = size;
  while (display >= 1024 && unitIndex < units.length - 1) {
    display /= 1024;
    unitIndex += 1;
  }
  const formatter = new Intl.NumberFormat('it-IT', {
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
  selfieLoadError.value = '';
  selfieModerationMessage.value = '';
  setSelfieBusy(selfie.id, true);
  try {
    await secureRequest(() =>
      apiClient.delete(`/admin/selfies/${selfie.id}`, authHeaders.value),
    );
    eventSelfies.value = eventSelfies.value.filter((item) => item.id !== selfie.id);
    selfieModerationMessage.value = 'Selfie eliminato.';
  } catch (error) {
    if (error?.response?.status === 404) {
      eventSelfies.value = eventSelfies.value.filter((item) => item.id !== selfie.id);
      selfieLoadError.value = 'Il selfie selezionato non è più disponibile.';
    } else if (error?.response?.status !== 401) {
      selfieLoadError.value = 'Impossibile eliminare il selfie. Riprova più tardi.';
    }
  } finally {
    setSelfieBusy(selfie.id, false);
  }
}

function parseHistoryDate(value) {
  if (typeof value !== 'string' || !value.trim()) {
    return null;
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

function formatHistoryDate(value) {
  const parsed = parseHistoryDate(value);
  if (!parsed) {
    return 'Data non disponibile';
  }
  try {
    return historyDateFormatter.format(parsed);
  } catch (error) {
    try {
      return parsed.toLocaleString('it-IT');
    } catch (innerError) {
      return parsed.toString();
    }
  }
}

function formatHistoryTime(date) {
  if (!(date instanceof Date) || Number.isNaN(date.valueOf())) {
    return '';
  }
  try {
    return historyTimeFormatter.format(date);
  } catch (error) {
    try {
      return date.toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' });
    } catch (innerError) {
      return '';
    }
  }
}

function buildHistoryTimelineChart(buckets, windowLabels = null) {
  if (!Array.isArray(buckets) || !buckets.length) {
    return {
      points: [],
      startLabel: windowLabels?.start || '',
      endLabel: windowLabels?.end || '',
    };
  }

  let cumulative = 0;
  const points = [];
  let computedStart = '';
  let computedEnd = '';

  buckets.forEach((bucket) => {
    const votes = Number(bucket?.votes ?? 0) || 0;
    cumulative += votes;

    const reference = bucket?.end || bucket?.start || '';
    const date = reference ? parseHistoryDate(reference) : null;
    if (!date) {
      return;
    }

    const label = bucket?.rangeLabel || bucket?.endLabel || bucket?.startLabel || '';
    if (!computedStart) {
      computedStart = bucket?.startLabel || label || '';
    }
    if (bucket?.endLabel || label) {
      computedEnd = bucket?.endLabel || label || computedEnd;
    }

    const votesLabel = votes.toLocaleString('it-IT');
    const cumulativeLabel = cumulative.toLocaleString('it-IT');
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
      tooltip: tooltipParts.join(' · '),
    });
  });

  const startLabel = windowLabels?.start || computedStart || buckets[0]?.rangeLabel || '';
  const endLabel = windowLabels?.end || computedEnd || buckets[buckets.length - 1]?.rangeLabel || '';

  return {
    points,
    startLabel,
    endLabel,
  };
}

function normalizeHistoryEntry(item) {
  const id = Number(item?.id) || 0;
  const homeTeam = typeof item?.home_team === 'string' ? item.home_team.trim() : '';
  const awayTeam = typeof item?.away_team === 'string' ? item.away_team.trim() : '';
  const rawTitle = typeof item?.title === 'string' ? item.title.trim() : '';
  const fallbackTitle = [homeTeam, awayTeam].filter(Boolean).join(' - ') || (id ? `Evento #${id}` : 'Evento');
  const startDatetime = typeof item?.start_datetime === 'string' ? item.start_datetime : '';
  const location = typeof item?.location === 'string' ? item.location.trim() : '';
  const totalVotes = Number(item?.total_votes ?? item?.totalVotes ?? 0) || 0;

  const sponsorClicks = Array.isArray(item?.sponsor_clicks)
    ? item.sponsor_clicks
        .map((entry) => ({
          id: Number(entry?.sponsor_id) || 0,
          name:
            typeof entry?.name === 'string' && entry.name.trim() ? entry.name.trim() : 'Sponsor',
          link: typeof entry?.link_url === 'string' ? entry.link_url.trim() : '',
          clicks: Number(entry?.clicks ?? 0) || 0,
        }))
        .sort((a, b) => {
          if (b.clicks !== a.clicks) {
            return b.clicks - a.clicks;
          }
          return a.name.localeCompare(b.name, 'it');
        })
    : [];

  const sponsorClicksTotalRaw = Number(item?.sponsor_clicks_total ?? item?.sponsorClicksTotal ?? 0);
  const sponsorClicksTotal = Number.isFinite(sponsorClicksTotalRaw)
    ? sponsorClicksTotalRaw
    : sponsorClicks.reduce((sum, sponsor) => sum + (Number(sponsor.clicks) || 0), 0);
  const sponsorClicksTotalLabel = Number.isFinite(sponsorClicksTotal)
    ? sponsorClicksTotal.toLocaleString('it-IT')
    : '0';

  const sponsorAnalyticsRaw = item?.sponsor_analytics ?? item?.sponsorAnalytics ?? null;
  const sponsorAnalyticsData = normalizeSponsorAnalyticsResponse(sponsorAnalyticsRaw);
  const sponsorAnalyticsDisplay = {
    totalUsers: sponsorAnalyticsData.totalUsers,
    totalUsersLabel: sponsorAnalyticsData.totalUsers.toLocaleString('it-IT'),
    seenUsers: sponsorAnalyticsData.seenUsers,
    seenUsersLabel: sponsorAnalyticsData.seenUsers.toLocaleString('it-IT'),
    seenRateLabel: `${formatPercent(sponsorAnalyticsData.seenRate)}%`,
    watchedUsers: sponsorAnalyticsData.watchedUsers,
    watchedUsersLabel: sponsorAnalyticsData.watchedUsers.toLocaleString('it-IT'),
    averageWatchTimeLabel: formatWatchDuration(sponsorAnalyticsData.averageWatchTimeMs),
    totalWatchTimeLabel: formatWatchDuration(sponsorAnalyticsData.totalWatchTimeMs),
    totalClicks: sponsorAnalyticsData.totalClicks,
    totalClicksLabel: sponsorAnalyticsData.totalClicks.toLocaleString('it-IT'),
    clickRateLabel: `${formatPercent(sponsorAnalyticsData.clickRate)}%`,
    uniqueClickersLabel: sponsorAnalyticsData.uniqueClickers.toLocaleString('it-IT'),
    topSponsorName: sponsorAnalyticsData.topSponsor?.name || 'Nessuno',
    topSponsorViewsLabel: sponsorAnalyticsData.topSponsor
      ? sponsorAnalyticsData.topSponsor.views.toLocaleString('it-IT')
      : '0',
  };

  const totalVisitors = Number(sponsorAnalyticsData.totalUsers) || 0;
  const uniqueVisitors = Number(sponsorAnalyticsData.seenUsers) || 0;
  const totalVisitorsLabel = totalVisitors.toLocaleString('it-IT');
  const uniqueVisitorsLabel = uniqueVisitors.toLocaleString('it-IT');

  const sponsorAnalyticsTimelineRaw = Array.isArray(sponsorAnalyticsData.timeline)
    ? sponsorAnalyticsData.timeline
    : [];
  const sponsorAnalyticsTimeline = sponsorAnalyticsTimelineRaw.map((point) => {
    const timestamp = typeof point?.timestamp === 'string' ? point.timestamp : '';
    const seen = Number(point?.seen ?? 0) || 0;
    const watched = Number(point?.watched ?? 0) || 0;
    const clicks = Number(point?.clicks ?? 0) || 0;
    let label = timestamp || '';
    if (timestamp) {
      const parsed = new Date(timestamp);
      if (!Number.isNaN(parsed.valueOf())) {
        label = historyTimeFormatter.format(parsed);
      }
    }
    return {
      timestamp,
      label: label || '—',
      seen,
      watched,
      clicks,
    };
  });

  const sponsorAnalyticsHasData = Boolean(
    sponsorAnalyticsData.totalUsers ||
      sponsorAnalyticsData.totalClicks ||
      sponsorAnalyticsTimeline.length ||
      (sponsorAnalyticsData.topSponsor && sponsorAnalyticsData.topSponsor.views),
  );

  const prizesRaw = Array.isArray(item?.prizes) ? item.prizes : [];
  const normalizedPrizes = prizesRaw
    .map((prize, index) => {
      if (!prize || typeof prize !== 'object') {
        return null;
      }
      const id = Number(prize?.id ?? prize?.ID) || 0;
      const position = Number(prize?.position ?? prize?.Position) || index + 1;
      const rawName =
        typeof (prize?.name ?? prize?.Name) === 'string' ? (prize?.name ?? prize?.Name).trim() : '';
      const name = rawName || `Premio ${position || index + 1}`;
      const winnerCodeRaw =
        typeof (prize?.winner_ticket_code ?? prize?.winnerTicketCode) === 'string'
          ? (prize?.winner_ticket_code ?? prize?.winnerTicketCode)
          : '';
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
      const start = typeof bucket?.start === 'string' ? bucket.start : '';
      const end = typeof bucket?.end === 'string' ? bucket.end : '';
      const votes = Number(bucket?.votes ?? 0) || 0;
      const explicitLabel = typeof bucket?.label === 'string' ? bucket.label.trim() : '';
      const startDate = start ? parseHistoryDate(start) : null;
      const endDate = end ? parseHistoryDate(end) : null;
      const startTimestamp = startDate ? startDate.getTime() : Number.NaN;
      const endTimestamp = endDate ? endDate.getTime() : Number.NaN;
      const startLabel = startDate ? historyTimeFormatter.format(startDate) : '';
      const endLabel = endDate ? historyTimeFormatter.format(endDate) : '';
      const rangeLabel = explicitLabel
        ? explicitLabel
        : startLabel && endLabel
        ? `${startLabel} - ${endLabel}`
        : startLabel || endLabel || '';
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
      return a.rangeLabel.localeCompare(b.rangeLabel, 'it');
    })
    .map((bucket) => ({
      start: bucket.start,
      end: bucket.end,
      rangeLabel: bucket.rangeLabel || 'Intervallo',
      votes: bucket.votes,
      votesLabel: Number.isFinite(bucket.votes) ? `${bucket.votes.toLocaleString('it-IT')} voti` : '0 voti',
      startLabel: bucket.startLabel,
      endLabel: bucket.endLabel,
    }));

  const firstBucketWithStart = timelineBuckets.find((bucket) => bucket.startLabel);
  const lastBucketWithEnd = [...timelineBuckets].reverse().find((bucket) => bucket.endLabel);
  const timelineRangeStart = firstBucketWithStart?.startLabel || timelineBuckets[0]?.rangeLabel || '';
  const timelineRangeEnd = lastBucketWithEnd?.endLabel || timelineBuckets[timelineBuckets.length - 1]?.rangeLabel || '';
  const timelineRange = timelineRangeStart || timelineRangeEnd
    ? {
        start: timelineRangeStart || timelineRangeEnd,
        end: timelineRangeEnd || timelineRangeStart,
      }
    : null;

  const timelineChart = buildHistoryTimelineChart(timelineBuckets, timelineRange);

  const mvpRaw = item?.mvp;
  let mvp = null;
  if (mvpRaw && Number(mvpRaw?.votes ?? 0) > 0) {
    const firstName = typeof mvpRaw?.first_name === 'string' ? mvpRaw.first_name.trim() : '';
    const lastName = typeof mvpRaw?.last_name === 'string' ? mvpRaw.last_name.trim() : '';
    const fallbackName = mvpRaw?.player_id ? `Giocatore ${mvpRaw.player_id}` : 'Giocatore';
    const name = [firstName, lastName].filter(Boolean).join(' ') || fallbackName;
    mvp = {
      id: Number(mvpRaw?.player_id) || 0,
      votes: Number(mvpRaw?.votes) || 0,
      name,
    };
  }

  const feedbackSummary = normalizeFeedbackSummary(item?.feedback_summary ?? item?.feedbackSummary);

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
    totalVotesLabel: Number.isFinite(totalVotes) ? totalVotes.toLocaleString('it-IT') : '0',
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
  };
}

function buildHistoryReportFilename(entry) {
  const eventId = Number(entry?.id) || 0;
  const parsedDate = parseHistoryDate(entry?.startDatetime);
  const datePart = parsedDate
    ? `${parsedDate.getFullYear()}${String(parsedDate.getMonth() + 1).padStart(2, '0')}${String(
        parsedDate.getDate(),
      ).padStart(2, '0')}`
    : '';
  const rawTitle = typeof entry?.title === 'string' ? entry.title : '';
  const normalizedTitle = rawTitle
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
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
    parts.push('evento-storico');
  }
  return `${parts.join('_')}.pdf`;
}

async function downloadEventHistoryReport(entry) {
  if (!entry || typeof entry !== 'object' || !entry.id) {
    return;
  }

  const eventId = entry.id;
  setHistoryReportBusy(eventId, true);
  eventHistoryError.value = '';

  try {
    await nextTick();
    const config = {
      ...authHeaders.value,
      responseType: 'blob',
    };
    const response = await apiClient.get(`/admin/events/history/${eventId}/report`, config);
    const blob = new Blob([response?.data], { type: 'application/pdf' });
    const filename = buildHistoryReportFilename(entry) || `evento-${eventId}.pdf`;
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    const safeTitle = typeof entry.title === 'string' ? entry.title : `Evento #${entry.id}`;
    eventHistorySuccess.value = `Report per "${safeTitle}" scaricato correttamente.`;
  } catch (error) {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    } else if (error?.response?.status === 404) {
      eventHistoryError.value = 'Il report richiesto non è disponibile.';
    } else {
      console.error('history report download error', error);
      eventHistoryError.value = 'Impossibile generare il report PDF. Riprova più tardi.';
    }
    eventHistorySuccess.value = '';
  } finally {
    setHistoryReportBusy(eventId, false);
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
  eventHistoryError.value = '';
  if (force) {
    eventHistorySuccess.value = '';
  }

  try {
    const { data } = await apiClient.get('/admin/events/history', authHeaders.value);
    const normalized = Array.isArray(data) ? data.map((entry) => normalizeHistoryEntry(entry)) : [];
    eventHistory.value = normalized;
    hasLoadedEventHistory.value = true;
  } catch (error) {
    const status = error?.response?.status;
    if (status === 401) {
      handleUnauthorized();
    } else {
      eventHistorySuccess.value = '';
      eventHistoryError.value = 'Impossibile caricare lo storico eventi. Riprova più tardi.';
    }
  } finally {
    isLoadingEventHistory.value = false;
  }
}

function toggleHistoryTimeline(entry) {
  if (!entry || typeof entry !== 'object') {
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
  purgeDialog.password = '';
  purgeDialog.error = '';
  purgeDialog.isSubmitting = false;
}

function closePurgeDialog() {
  purgeDialog.visible = false;
  purgeDialog.event = null;
  purgeDialog.password = '';
  purgeDialog.error = '';
  purgeDialog.isSubmitting = false;
}

async function confirmPurge() {
  if (!purgeDialog.event || purgeDialog.isSubmitting || !purgeDialog.password) {
    return;
  }
  purgeDialog.isSubmitting = true;
  purgeDialog.error = '';

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
      purgeDialog.error = 'Password non valida o privilegi insufficienti.';
    } else if (status === 404) {
      purgeDialog.error = 'Evento già rimosso.';
      eventHistory.value = eventHistory.value.filter((entry) => entry.id !== purgeDialog.event.id);
    } else {
      purgeDialog.error = 'Impossibile eliminare l\'evento. Riprova.';
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
    await Promise.all([loadAdmins(), loadSponsors()]);
  }
  ensureSelfieSelection();
  if (section.value === 'selfies' && selectedSelfieEventId.value) {
    await loadEventSelfies(selectedSelfieEventId.value);
  }
  hasLoadedEventHistory.value = false;
  if (section.value === 'history') {
    await loadEventHistory({ force: true });
  }
  resetForms();
}

async function createTeam() {
  if (!newTeamName.value) {
    return;
  }
  globalError.value = '';
  await secureRequest(() => apiClient.post('/teams', { name: newTeamName.value }, authHeaders.value));
  newTeamName.value = '';
  await loadTeams();
}

async function deleteTeam(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/teams/${id}`, authHeaders.value));
  await loadTeams();
}

async function createEvent() {
  globalError.value = '';
  if (!hasEnoughTeams.value) {
    globalError.value = 'Aggiungi almeno due squadre per creare un evento.';
    return;
  }
  if (!newEvent.team1_id || !newEvent.team2_id) {
    globalError.value = 'Seleziona entrambe le squadre.';
    return;
  }
  if (newEvent.team1_id === newEvent.team2_id) {
    globalError.value = 'Le due squadre devono essere diverse.';
    return;
  }
  if (!newEvent.start_datetime) {
    globalError.value = 'Imposta data e ora della partita.';
    return;
  }

  const prizesPayload = newEventPrizes.value
    .map((prize, index) => ({
      id: Number(prize.id) || 0,
      name: (prize.name || '').trim(),
      position: index + 1,
    }))
    .filter((prize) => prize.name);

  const payload = {
    team1_id: newEvent.team1_id,
    team2_id: newEvent.team2_id,
    start_datetime: newEvent.start_datetime,
    location: newEvent.location,
    show_reaction_test: Boolean(newEvent.show_reaction_test),
    show_selfie: Boolean(newEvent.show_selfie),
    show_vote_trend: Boolean(newEvent.show_vote_trend),
    show_feedback_survey: Boolean(newEvent.show_feedback_survey),
    prizes: prizesPayload,
  };

  const { data } = await secureRequest(() => apiClient.post('/events', payload, authHeaders.value));
  await loadEvents();
  if (data?.id) {
    lastCreatedEventLink.value = buildEventLink(data.id);
  }
  Object.assign(newEvent, createDefaultNewEventState());
  teamInputs.home = '';
  teamInputs.away = '';
  resetNewEventPrizes();
}

async function deleteEvent(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/events/${id}`, authHeaders.value));
  await loadEvents();
}

async function activateEvent(id) {
  if (updatingEventId.value === id) {
    return;
  }
  globalError.value = '';
  closeVotesMessage.value = '';
  updatingEventId.value = id;
  try {
    await secureRequest(() => apiClient.post(`/events/${id}/activate`, {}, authHeaders.value));
    await loadEvents();
  } finally {
    updatingEventId.value = 0;
  }
}

async function deactivateEvents() {
  if (isDisablingEvents.value) {
    return;
  }
  globalError.value = '';
  closeVotesMessage.value = '';
  isDisablingEvents.value = true;
  try {
    await secureRequest(() => apiClient.post('/events/deactivate', {}, authHeaders.value));
    await loadEvents();
  } finally {
    isDisablingEvents.value = false;
  }
}

async function concludeEvent(id) {
  if (concludingEventId.value === id) {
    return;
  }
  globalError.value = '';
  closeVotesMessage.value = '';
  const eventInfo = events.value.find((event) => event.id === id);
  const concludedLabel = eventInfo ? eventLabel(eventInfo) : '';
  concludingEventId.value = id;
  try {
    await secureRequest(() => apiClient.post(`/events/${id}/conclude`, {}, authHeaders.value));
    await loadEvents();
    await loadEventHistory({ force: true });
    if (!eventHistoryError.value) {
      eventHistorySuccess.value = concludedLabel
        ? `Evento "${concludedLabel}" spostato nello storico.`
        : 'Evento spostato nello storico.';
    }
  } catch (error) {
    const status = error?.response?.status;
    if (status === 401) {
      return;
    }
    if (status === 404) {
      globalError.value = 'Evento non trovato o già rimosso.';
    } else if (status === 409) {
      globalError.value = "L'evento è già stato segnato come concluso.";
    }
    await loadEvents();
  } finally {
    concludingEventId.value = 0;
  }
}

async function closeActiveEventVoting() {
  if (!activeEventId.value || isClosingVotes.value || activeEventVotesClosed.value) {
    return;
  }
  closeVotesMessage.value = '';
  globalError.value = '';
  isClosingVotes.value = true;
  try {
    await secureRequest(() =>
      apiClient.post(`/events/${activeEventId.value}/close-votes`, {}, authHeaders.value),
    );
    await loadEvents();
    closeVotesMessage.value = 'Le votazioni per l\'evento attivo sono state chiuse.';
  } catch (error) {
    closeVotesMessage.value = '';
    if (error?.response?.status === 404) {
      globalError.value = 'Impossibile chiudere le votazioni: nessun evento attivo trovato.';
    }
  } finally {
    isClosingVotes.value = false;
  }
}

async function createAdmin() {
  globalError.value = '';
  await secureRequest(() => apiClient.post('/admins', newAdmin, authHeaders.value));
  Object.assign(newAdmin, { username: '', password: '', role: '' });
  await loadAdmins();
}

async function deleteAdmin(id) {
  globalError.value = '';
  await secureRequest(() => apiClient.delete(`/admins/${id}`, authHeaders.value));
  await loadAdmins();
}

async function createSponsor() {
  if (isCreatingSponsor.value) {
    return;
  }
  globalError.value = '';
  if (sponsors.value.length >= maxSponsors) {
    globalError.value = `Puoi configurare al massimo ${maxSponsors} sponsor.`;
    return;
  }
  const trimmedName = newSponsor.name.trim();
  if (!newSponsor.logoData) {
    globalError.value = 'Carica un logo per lo sponsor.';
    return;
  }
  const payload = serializeSponsorPayload({
    name: trimmedName,
    linkUrl: newSponsor.linkUrl,
    logoData: newSponsor.logoData,
    position: nextSponsorPosition(),
    isActive: false,
  });
  isCreatingSponsor.value = true;
  try {
    await secureRequest(() => apiClient.post('/admin/sponsors', payload, authHeaders.value));
    resetNewSponsorForm();
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value = 'Controlla i dati inseriti: sono disponibili massimo 4 sponsor.';
    }
  } finally {
    isCreatingSponsor.value = false;
  }
}

async function updateSponsorEntry(sponsor) {
  if (sponsorBeingUpdated.value === sponsor.id) {
    return;
  }
  globalError.value = '';
  const trimmedName = sponsor.name.trim();
  if (!sponsor.logoData) {
    globalError.value = 'Carica un logo per lo sponsor.';
    return;
  }
  sponsorBeingUpdated.value = sponsor.id;
  try {
    const payload = serializeSponsorPayload({
      name: trimmedName,
      linkUrl: sponsor.linkUrl,
      logoData: sponsor.logoData,
      position: sponsor.position,
      isActive: sponsor.isActive,
    });
    await secureRequest(() => apiClient.put(`/admin/sponsors/${sponsor.id}`, payload, authHeaders.value));
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 400) {
      globalError.value = 'Controlla i dati dello sponsor e riprova.';
    } else if (error?.response?.status === 404) {
      globalError.value = 'Sponsor non trovato. Aggiorna la pagina.';
    }
  } finally {
    sponsorBeingUpdated.value = 0;
  }
}

async function deleteSponsorEntry(id) {
  if (sponsorBeingDeleted.value === id) {
    return;
  }
  globalError.value = '';
  sponsorBeingDeleted.value = id;
  try {
    await secureRequest(() => apiClient.delete(`/admin/sponsors/${id}`, authHeaders.value));
    await loadSponsors();
  } catch (error) {
    if (error?.response?.status === 404) {
      globalError.value = 'Sponsor già rimosso.';
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
  globalError.value = '';
  const target = Math.max(0, Math.min(maxSponsors, desiredActiveSponsorCount.value));
  isApplyingSponsorCount.value = true;
  try {
    const updates = [];
    sortedSponsors().forEach((sponsor, index) => {
      const shouldBeActive = index < target;
      if (sponsor.isActive !== shouldBeActive) {
        const payload = serializeSponsorPayload({
          name: sponsor.name.trim(),
          linkUrl: sponsor.linkUrl,
          logoData: sponsor.logoData,
          position: sponsor.position,
          isActive: shouldBeActive,
        });
        updates.push(
          secureRequest(() =>
            apiClient.put(`/admin/sponsors/${sponsor.id}`, payload, authHeaders.value),
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
      globalError.value = 'Impossibile aggiornare il numero di sponsor visibili. Verifica i dati e riprova.';
    }
  } finally {
    isApplyingSponsorCount.value = false;
  }
}

function openVote(eventId) {
  const url = buildEventLink(eventId);
  window.open(url, '_blank', 'noopener');
}

async function copyLink(link) {
  try {
    await navigator.clipboard.writeText(link);
    globalError.value = '';
    toast.add({
      severity: 'success',
      summary: 'Link copiato',
      detail: 'Il link è stato copiato negli appunti.',
      life: 3000,
    });
  } catch (error) {
    globalError.value = 'Impossibile copiare il link automaticamente.';
    toast.add({
      severity: 'warn',
      summary: 'Copia non riuscita',
      detail: 'Copia il link manualmente dalla scheda evento.',
      life: 4000,
    });
  }
}

function ensureSectionIsAllowed(tabList) {
  if (!isAuthenticated.value) {
    return;
  }
  if (!tabList.some((tab) => tab.id === section.value)) {
    section.value = tabList.length ? tabList[0].id : '';
  }
}

function handleSectionChange(tabId) {
  if (availableTabs.value.some((tab) => tab.id === tabId)) {
    section.value = tabId;
  }
}

watch(
  availableTabs,
  (currentTabs) => {
    ensureSectionIsAllowed(currentTabs);
  },
  { immediate: true },
);

watch(section, (value, oldValue) => {
  if (value === 'results') {
    ensureResultsSelection();
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (oldValue === 'results') {
    stopResultsPolling();
  }
  if (value === 'selfies') {
    ensureSelfieSelection();
    if (selectedSelfieEventId.value) {
      loadEventSelfies(selectedSelfieEventId.value);
    }
  } else if (oldValue === 'selfies') {
    selfieModerationMessage.value = '';
    selfieLoadError.value = '';
  }
  if (value === 'history') {
    loadEventHistory();
  }
});

watch(selectedResultsEventId, (eventId) => {
  if (section.value === 'results' && eventId) {
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (!eventId) {
    stopResultsPolling();
  }
});

watch(selectedSelfieEventId, (eventId) => {
  if (section.value !== 'selfies') {
    return;
  }
  selfieModerationMessage.value = '';
  selfieLoadError.value = '';
  if (eventId) {
    loadEventSelfies(eventId);
  } else {
    eventSelfies.value = [];
  }
});

if (isAuthenticated.value) {
  loadAll();
}

onBeforeUnmount(() => {
  stopResultsPolling();
});
</script>


<style scoped>
.admin-portal-view {
  min-height: 100vh;
  background: radial-gradient(circle at top left, rgba(59, 130, 246, 0.25), transparent 45%),
    radial-gradient(circle at bottom right, rgba(168, 85, 247, 0.2), transparent 40%),
    linear-gradient(180deg, var(--p-surface-100), var(--p-surface-0));
  padding: 2rem 1.5rem 3rem;
  color: var(--p-surface-800);
}

.admin-portal-view__login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 0;
}

.admin-portal-view__login-card {
  width: 100%;
  max-width: 420px;
  border-radius: 1.25rem;
  box-shadow: 0 20px 60px -40px rgba(15, 23, 42, 0.8);
  overflow: hidden;
}

.admin-portal-view__login-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.admin-field-label {
  display: block;
  font-weight: 600;
  margin-bottom: 0.35rem;
  color: var(--p-surface-700);
}

.admin-portal-view__login-actions {
  display: flex;
  justify-content: flex-end;
}

.admin-portal-view__inline-message {
  margin-top: 1rem;
}

.admin-brand {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 700;
  font-size: 1.25rem;
  color: var(--p-primary-700);
  letter-spacing: 0.02em;
}

.admin-brand__icon {
  font-size: 1.25rem;
  color: var(--p-primary-500);
}

.admin-active-event-tag {
  margin-right: 0.5rem;
}

.admin-section-card {
  position: relative;
}

.admin-section-card :deep(.p-card) {
  background: rgba(255, 255, 255, 0.88);
  border-radius: 1.25rem;
  border: 1px solid var(--p-surface-200);
  box-shadow: 0 24px 60px -40px rgba(15, 23, 42, 0.35);
  overflow: hidden;
}

.admin-section-card :deep(.p-card-title) {
  font-size: 1.35rem;
  font-weight: 700;
  color: var(--p-surface-900);
}

.admin-section-card :deep(.p-card-subtitle) {
  color: var(--p-text-muted-color);
  font-weight: 500;
}

.admin-section__toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.admin-form {
  display: grid;
  gap: 1rem;
}

.admin-form--grid {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.admin-form--inline {
  grid-template-columns: auto auto;
  align-items: end;
  gap: 1rem;
}

.admin-form--compact {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem;
}

.admin-form__field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.admin-form__field--full {
  grid-column: 1 / -1;
}

.admin-form__field--grow {
  flex: 1 1 auto;
}

.admin-form__field--medium {
  max-width: 340px;
}

.admin-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  grid-column: 1 / -1;
}

.admin-toggle-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.admin-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  background: var(--p-surface-100);
  border: 1px solid var(--p-surface-200);
}

.admin-toggle__label {
  font-weight: 600;
  color: var(--p-surface-700);
}

.admin-prize-grid {
  display: grid;
  gap: 0.75rem;
}

.admin-prize-grid__row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.75rem;
  align-items: center;
}

.admin-prize-grid__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.admin-inline-link {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 600;
}

.admin-inline-link a {
  color: var(--p-primary-500);
  text-decoration: none;
}

.admin-events__list {
  margin-top: 2rem;
}

.admin-event-card :deep(.p-card-content) {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.admin-event-card__title {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.admin-event-card__tag {
  font-size: 0.75rem;
}

.admin-event-card__subtitle {
  color: var(--p-text-muted-color);
  font-weight: 500;
}

.admin-event-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.admin-event-card__info {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  color: var(--p-text-muted-color);
}

.admin-event-card__postvote,
.admin-event-card__prizes {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.admin-status-card {
  padding: 1.5rem;
  border-radius: 1.25rem;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.15), rgba(30, 64, 175, 0.05));
  border: 1px solid rgba(59, 130, 246, 0.25);
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.admin-status-card__header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
}

.admin-status-card__title {
  margin: 0;
  font-size: 1.2rem;
}

.admin-status-card__subtitle {
  color: var(--p-text-muted-color);
}

.admin-status-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.admin-results__controls {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1.5rem;
}

.admin-results__dropdown {
  min-width: 260px;
}

.admin-results__summary {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 1.5rem;
}

.admin-results__content {
  display: grid;
  gap: 1.5rem;
}

.admin-results__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  color: var(--p-text-muted-color);
  font-weight: 600;
}

.admin-results__auto {
  font-style: italic;
  color: var(--p-primary-500);
}

.admin-results__votes {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.admin-sponsor-analytics {
  display: grid;
  gap: 1.5rem;
}

.admin-sponsor-analytics__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.admin-sponsor-analytics__stat :deep(.p-card-content) {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  text-align: center;
}

.admin-sponsor-analytics__value {
  font-size: 1.35rem;
  font-weight: 700;
}

.admin-sponsor-analytics__hint {
  color: var(--p-text-muted-color);
}

.admin-sponsor-analytics__chart h4 {
  margin-bottom: 1rem;
}

.admin-loader {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 600;
  color: var(--p-text-muted-color);
}

.admin-selfie-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1.25rem;
}

.admin-selfie-card :deep(.p-card)
{
  border-radius: 1.25rem;
  border: 1px solid var(--p-surface-200);
  overflow: hidden;
}

.admin-selfie-card__thumb {
  position: relative;
  padding-top: 60%;
  background: var(--p-surface-200);
  display: flex;
  align-items: center;
  justify-content: center;
}

.admin-selfie-card__thumb img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.admin-selfie-card__thumb--empty {
  color: var(--p-text-muted-color);
  font-weight: 600;
}

.admin-history {
  margin-top: 1.5rem;
  border-radius: 1.25rem;
  overflow: hidden;
}

.admin-history__header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
  width: 100%;
}

.admin-history__header h3 {
  margin: 0;
}

.admin-history__header span {
  color: var(--p-text-muted-color);
}

.admin-history__chips {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.admin-history__content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding-top: 1.25rem;
}

.admin-history__actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.admin-history__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1rem;
}

.admin-history__panel :deep(.p-card) {
  border-radius: 1.25rem;
  border: 1px solid var(--p-surface-200);
}

.admin-history__stats {
  display: grid;
  gap: 0.75rem;
}

.admin-history__stat {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.admin-history__stat-label {
  font-weight: 600;
  color: var(--p-text-muted-color);
}

.admin-history__stat-value {
  font-size: 1.1rem;
}

.admin-history__stat-hint {
  color: var(--p-text-muted-color);
}

.admin-history__question {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-top: 1rem;
}

.admin-history__votes {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.admin-simple-table :deep(.p-datatable) {
  border-radius: 1.25rem;
  overflow: hidden;
}

.admin-table-actions {
  display: flex;
  justify-content: flex-end;
}

.admin-player-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}

.admin-player-card :deep(.p-card) {
  border-radius: 1.25rem;
  border: 1px solid var(--p-surface-200);
}

.admin-player-card__preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: flex-start;
}

.admin-player-card__preview img {
  width: 100%;
  max-height: 160px;
  object-fit: cover;
  border-radius: 0.75rem;
  border: 1px solid var(--p-surface-200);
}

.admin-player-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.admin-sponsor-grid :deep(.p-dataview-grid .p-dataview-content) {
  padding: 0;
}

.admin-sponsor-card :deep(.p-card) {
  border-radius: 1.25rem;
  border: 1px solid var(--p-surface-200);
}

.admin-sponsor-card__title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
}

.admin-sponsor-card__actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.admin-dialog__lead {
  font-size: 1rem;
  font-weight: 600;
  color: var(--p-surface-800);
}

.admin-text-muted {
  color: var(--p-text-muted-color);
}

@media (max-width: 768px) {
  .admin-portal-view {
    padding: 1.5rem 1rem 2.5rem;
  }
  .admin-section__toolbar,
  .admin-player-actions {
    justify-content: center;
  }
  .admin-results__controls {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>

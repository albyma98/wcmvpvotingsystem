<template>
  <div class="admin-portal">
    <header class="admin-header">
      <h1>Area amministratore</h1>
      <p class="subtitle">Gestisci eventi, squadre e votazioni MVP</p>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
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
        <button class="btn primary" type="submit" :disabled="isLoggingIn">
          {{ isLoggingIn ? 'Accesso in corso…' : 'Entra' }}
        </button>
      </form>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </section>

    <section v-else class="portal" ref="portalRef">
      <div class="toolbar" ref="toolbarRef">
        <div class="user-info">
          <span>Connesso come <strong>{{ activeUsername }}</strong></span>
          <button class="btn outline" type="button" @click="goToLottery">Lotteria</button>
          <button
            v-for="tab in availableTabs"
            :key="tab.id"
            :class="['btn outline', { active: section === tab.id }]"
            type="button"
            :aria-current="section === tab.id ? 'page' : undefined"
            @click="section = tab.id"
          >
            {{ tab.label }}
          </button>
          <button class="btn secondary" type="button" @click="logout">Esci</button>
        </div>



      </div>
      <div class="portal-content">
        <p v-if="globalError" class="error">{{ globalError }}</p>

        <section v-if="section === 'events'" class="card">
        <header class="section-header">
          <h2>Eventi</h2>
          <p>Crea una nuova partita per abilitare il voto pubblico.</p>
        </header>
        <div class="actions-row">
          <button
            class="btn outline"
            type="button"
            @click="deactivateEvents"
            :disabled="!activeEventId || isDisablingEvents"
          >
            {{ isDisablingEvents ? 'Disattivazione…' : 'Disattiva eventi' }}
          </button>
        </div>
        <p v-if="!hasEnoughTeams" class="info-banner">
          Aggiungi almeno due squadre dalla sezione "Squadre" per abilitare la creazione di un evento.
        </p>
        <form @submit.prevent="createEvent" class="form-grid">
          <label>
            Squadra di casa
            <input
              v-model="teamInputs.home"
              type="text"
              list="admin-team-options"
              :disabled="!hasEnoughTeams"
              placeholder="Digita il nome della squadra"
              required
              @change="handleTeamInput('home')"
              @blur="handleTeamInput('home')"
            />
            <small class="field-hint" v-if="hasEnoughTeams">
              Scegli dalla lista oppure digita per filtrare le squadre disponibili.
            </small>
          </label>
          <label>
            Squadra ospite
            <input
              v-model="teamInputs.away"
              type="text"
              list="admin-team-options"
              :disabled="!hasEnoughTeams"
              placeholder="Digita il nome della squadra"
              required
              @change="handleTeamInput('away')"
              @blur="handleTeamInput('away')"
            />
            <small class="field-hint" v-if="hasEnoughTeams">
              Seleziona una squadra diversa da quella di casa.
            </small>
          </label>
          <datalist id="admin-team-options">
            <option v-for="team in teams" :key="team.id" :value="teamOptionValue(team)"></option>
          </datalist>
          <label>
            Data e ora
            <input
              v-model="newEvent.start_datetime"
              type="datetime-local"
              :disabled="!hasEnoughTeams"
              required
            />
          </label>
          <label>
            Location
            <input
              v-model.trim="newEvent.location"
              type="text"
              placeholder="Es. Palazzetto dello Sport"
              :disabled="!hasEnoughTeams"
            />
          </label>
          <div class="prize-editor new-event-prizes">
            <div class="prize-editor__header">
              <span>Premi in palio</span>
              <p class="field-hint">Aggiungi i premi disponibili per la lotteria dell'evento.</p>
            </div>
            <div class="prize-editor__list">
              <div
                v-for="(prize, index) in newEventPrizes"
                :key="`new-event-prize-${index}`"
                class="prize-editor__row"
              >
                <input
                  v-model.trim="prize.name"
                  type="text"
                  :placeholder="`Premio ${index + 1}`"
                  :disabled="!hasEnoughTeams"
                />
                <button
                  class="btn outline"
                  type="button"
                  @click="removeNewEventPrize(index)"
                  :disabled="newEventPrizes.length <= 1"
                >
                  Rimuovi
                </button>
              </div>
            </div>
            <div class="prize-editor__actions">
              <button class="btn secondary" type="button" @click="addNewEventPrize" :disabled="!hasEnoughTeams">
                Aggiungi premio
              </button>
            </div>
          </div>
          <button class="btn primary" type="submit" :disabled="!hasEnoughTeams">Crea evento</button>
        </form>

        <div v-if="lastCreatedEventLink" class="hint">
          Nuovo evento creato! Link pubblico:
          <a :href="lastCreatedEventLink" target="_blank" rel="noopener">{{ lastCreatedEventLink }}</a>
          <button class="btn link" type="button" @click="copyLink(lastCreatedEventLink)">Copia</button>
        </div>

        <ul class="item-list">
          <li v-for="event in visibleEvents" :key="event.id" :class="['item', { active: event.is_active }]">
            <div class="item-body">
              <h3>
                {{ eventLabel(event) }}
                <span v-if="event.is_active" class="badge">Attivo</span>
                <span
                  v-if="event.is_active && event.votes_closed"
                  class="badge badge-closed"
                >
                  Votazioni chiuse
                </span>
              </h3>
              <p class="muted">{{ formatEventDate(event.start_datetime) }} • {{ event.location || 'Location da definire' }}</p>
              <p class="muted">
                Link voto:
                <a :href="buildEventLink(event.id)" target="_blank" rel="noopener">{{ buildEventLink(event.id) }}</a>
              </p>
            </div>
            <div class="item-actions">
              <button
                class="btn success"
                type="button"
                @click="activateEvent(event.id)"
                :disabled="event.is_active || updatingEventId === event.id"
              >
                <span v-if="event.is_active">Evento attivo</span>
                <span v-else-if="updatingEventId === event.id">Attivazione…</span>
                <span v-else>Attiva</span>
              </button>
              <button class="btn secondary" type="button" @click="openVote(event.id)">Apri pagina voto</button>
              <button
                class="btn warning"
                type="button"
                @click="concludeEvent(event.id)"
                :disabled="concludingEventId === event.id"
              >
                <span v-if="concludingEventId === event.id">Conclusione…</span>
                <span v-else>Evento terminato</span>
              </button>
              <button class="btn danger" type="button" @click="deleteEvent(event.id)">Elimina</button>
            </div>
            <div class="prize-editor existing-prizes">
              <div class="prize-editor__header">
                <strong>Premi in palio</strong>
                <p class="field-hint">Modifica l'elenco dei premi. I premi già assegnati non possono essere rimossi.</p>
              </div>
              <div class="prize-editor__list">
                <div
                  v-for="(prize, index) in prizeDraftsFor(event.id)"
                  :key="`event-${event.id}-prize-${prize.id || index}`"
                  class="prize-editor__row"
                >
                  <input
                    v-model="prize.name"
                    type="text"
                    :placeholder="`Premio ${index + 1}`"
                    :disabled="isSavingPrizesFor(event.id)"
                  />
                  <span v-if="prize.winner" class="prize-editor__winner">Assegnato a {{ prizeWinnerLabel(prize) }}</span>
                  <button
                    class="btn outline"
                    type="button"
                    @click="removePrizeDraft(event.id, index)"
                    :disabled="prize.winner || prizeDraftsFor(event.id).length <= 1 || isSavingPrizesFor(event.id)"
                  >
                    Rimuovi
                  </button>
                </div>
              </div>
              <div class="prize-editor__actions">
                <button
                  class="btn secondary"
                  type="button"
                  @click="addPrizeDraft(event.id)"
                  :disabled="isSavingPrizesFor(event.id)"
                >
                  Aggiungi premio
                </button>
                <button
                  class="btn primary"
                  type="button"
                  @click="savePrizesForEvent(event)"
                  :disabled="isSavingPrizesFor(event.id)"
                >
                  {{ isSavingPrizesFor(event.id) ? 'Salvataggio…' : 'Salva premi' }}
                </button>
              </div>
              <p v-if="eventPrizeErrors[event.id]" class="error">{{ eventPrizeErrors[event.id] }}</p>
            </div>
          </li>
        </ul>
      </section>

      <section v-else-if="section === 'closing'" class="card closing-card">
        <header class="section-header">
          <h2>Chiusura votazioni</h2>
          <p>Gestisci lo stato delle votazioni per la partita attualmente attiva.</p>
        </header>

        <div v-if="activeEventEntry" class="active-event-summary">
          <div class="summary-header">
            <h3>{{ activeEventLabel }}</h3>
            <span :class="['badge', activeEventVotesClosed ? 'badge-closed' : 'badge-open']">
              {{ activeEventVotesClosed ? 'Votazioni chiuse' : 'Votazioni aperte' }}
            </span>
          </div>
          <p class="muted">{{ activeEventDateLabel }} • {{ activeEventLocation }}</p>

          <div class="actions-row">
            <button
              class="btn warning"
              type="button"
              @click="closeActiveEventVoting"
              :disabled="isClosingVotes || activeEventVotesClosed"
            >
              {{ isClosingVotes ? 'Chiusura…' : 'Chiudi votazioni' }}
            </button>
            <button
              class="btn success"
              type="button"
              @click="activateEvent(activeEventEntry.id)"
              :disabled="
                !activeEventEntry || updatingEventId === activeEventEntry.id || !activeEventVotesClosed
              "
            >
              <span v-if="updatingEventId === activeEventEntry.id">Riattivazione…</span>
              <span v-else>Attiva</span>
            </button>
            <button
              class="btn outline"
              type="button"
              @click="deactivateEvents"
              :disabled="isDisablingEvents"
            >
              {{ isDisablingEvents ? 'Disattivazione…' : 'Disattiva' }}
            </button>
          </div>
        </div>
        <div v-else class="info-banner">
          Nessun evento attivo al momento. Attiva una partita dalla sezione "Eventi" per gestire le votazioni.
        </div>

        <p v-if="closeVotesMessage" class="success-message">{{ closeVotesMessage }}</p>
      </section>

      <section v-else-if="section === 'results'" class="card results-card">
        <header class="section-header">
          <h2>Risultati votazioni</h2>
          <p>Seleziona un evento per vedere la classifica MVP aggiornata in tempo reale.</p>
        </header>

        <div class="results-controls">
          <label>
            Evento
            <select v-model.number="selectedResultsEventId" :disabled="!events.length">
              <option disabled value="0">Seleziona un evento</option>
              <option v-for="event in events" :key="event.id" :value="event.id">
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
            {{ isLoadingResults ? 'Aggiornamento…' : 'Aggiorna ora' }}
          </button>
        </div>

        <div v-if="selectedResultsEvent" class="results-summary">
          <h3>{{ selectedResultsEventLabel }}</h3>
          <p class="muted">{{ selectedResultsEventDate || 'Data da definire' }}</p>
        </div>

        <p v-if="resultsError" class="error">{{ resultsError }}</p>
        <div v-else-if="!events.length" class="info-banner">
          Crea un evento per visualizzare i risultati delle votazioni MVP.
        </div>
        <div v-else class="results-leaderboard">
          <div class="results-meta">
            <span><strong>Voti totali:</strong> {{ totalVotes }}</span>
            <span v-if="lastResultsUpdateLabel"><strong>Ultimo aggiornamento:</strong> {{ lastResultsUpdateLabel }}</span>
            <span class="auto-refresh">Aggiornamento automatico ogni 5 secondi</span>
          </div>
          <p v-if="isLoadingResults" class="muted">Caricamento risultati…</p>
          <p v-else-if="!hasResultsVotes" class="muted">Non ci sono ancora voti per questo evento.</p>
          <ul class="leaderboard-list" aria-live="polite">
            <li v-for="(entry, index) in resultsLeaderboard" :key="entry.id" class="leaderboard-item">
              <div class="rank">#{{ index + 1 }}</div>
              <div class="player-name">
                <span class="lastname">{{ entry.lastNameUpper }}</span>
                <span class="firstname">{{ entry.firstName }}</span>
              </div>
              <div class="votes">
                <strong>{{ entry.votes }}</strong>
                <span class="muted">{{ entry.votes === 1 ? 'voto' : 'voti' }}</span>
              </div>
              <div class="progress" role="presentation">
                <div class="progress-bar" :style="{ width: `${entry.percentage}%` }"></div>
              </div>
            </li>
          </ul>
        </div>
      </section>

      <section v-else-if="section === 'history'" class="card history-card">
        <header class="section-header">
          <h2>Storico eventi</h2>
          <p>Consulta i dati degli eventi passati con riepilogo voti, MVP e interazioni sponsor.</p>
        </header>

        <div class="history-toolbar">
          <button
            class="btn secondary"
            type="button"
            @click="refreshEventHistory"
            :disabled="isLoadingEventHistory"
          >
            {{ isLoadingEventHistory ? 'Aggiornamento…' : 'Aggiorna' }}
          </button>
        </div>

        <p v-if="eventHistorySuccess" class="success-message">{{ eventHistorySuccess }}</p>
        <p v-if="eventHistoryError" class="error">{{ eventHistoryError }}</p>
        <p v-else-if="isLoadingEventHistory" class="muted text-center">Caricamento storico in corso…</p>
        <p v-else-if="!eventHistory.length" class="muted text-center">Non sono presenti eventi conclusi al momento.</p>

        <ul v-else class="history-list">
          <li v-for="entry in eventHistory" :key="entry.id" class="history-item">
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
                  <span class="history-item__sponsor-total">
                    <strong>{{ entry.sponsorClicksTotalLabel }}</strong> click sponsor
                  </span>
                </div>
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

            <div class="history-details">
              <div class="history-details__column">
                <h4>MVP</h4>
                <p v-if="entry.mvp">
                  {{ entry.mvp.name }} • {{ entry.mvp.votes.toLocaleString('it-IT') }} voti
                </p>
                <p v-else class="muted">Nessun MVP assegnato.</p>
              </div>
              <div class="history-details__column">
                <h4>Interazioni sponsor</h4>
                <ul v-if="entry.sponsorClicks.length" class="history-sponsor-list">
                  <li
                    v-for="sponsor in entry.sponsorClicks"
                    :key="`${entry.id}-sponsor-${sponsor.id}`"
                  >
                    <span class="history-sponsor-name">{{ sponsor.name }}</span>
                    <span class="history-sponsor-clicks">{{ sponsor.clicks.toLocaleString('it-IT') }} click</span>
                  </li>
                </ul>
                <p v-else class="muted">Nessun click registrato.</p>
              </div>
              <div class="history-details__column history-details__column--prizes">
                <h4>Estrazione premi</h4>
                <p
                  class="history-prize-status"
                  :class="entry.hasPrizeDraw ? 'history-prize-status--success' : 'history-prize-status--pending'"
                >
                  {{ entry.hasPrizeDraw ? 'Estrazione eseguita' : 'Estrazione non eseguita' }}
                </p>
                <p v-if="!entry.prizes.length" class="muted">Nessun premio configurato per l'evento.</p>
                <ul v-else class="history-prize-list">
                  <li
                    v-for="prize in entry.prizes"
                    :key="`${entry.id}-prize-${prize.id}`"
                    class="history-prize-item"
                  >
                    <span class="history-prize-name">{{ prize.name }}</span>
                    <span v-if="prize.hasWinner" class="history-prize-code">
                      Codice vincente: <strong>{{ prize.winnerTicketCode }}</strong>
                    </span>
                    <span v-else class="history-prize-code muted">Nessun codice vincente assegnato.</span>
                  </li>
                </ul>
              </div>
            </div>

            <div class="history-votes" v-if="entry.timeline.length">
              <div class="history-votes__header">
                <h4>Votazioni</h4>
                <p v-if="entry.timelineRange" class="history-votes__range">
                  Dal {{ entry.timelineRange.start }} al {{ entry.timelineRange.end }}
                </p>
              </div>
              <ul class="history-votes-list">
                <li
                  v-for="bucket in entry.timeline"
                  :key="`${entry.id}-bucket-${bucket.start || bucket.rangeLabel}`"
                  class="history-votes-list__item"
                >
                  <span class="history-votes-list__range">{{ bucket.rangeLabel }}</span>
                  <span class="history-votes-list__votes">{{ bucket.votesLabel }}</span>
                </li>
              </ul>
            </div>
          </li>
        </ul>
      </section>

      <section v-else-if="section === 'teams'" class="card">
        <header class="section-header">
          <h2>Squadre</h2>
        </header>
        <form @submit.prevent="createTeam" class="form-inline">
          <input v-model.trim="newTeamName" type="text" placeholder="Nome squadra" required />
          <button class="btn primary" type="submit">Aggiungi</button>
        </form>
        <ul class="item-list compact">
          <li v-for="team in teams" :key="team.id" class="item">
            <span>{{ team.name }}</span>
            <button class="btn danger" type="button" @click="deleteTeam(team.id)">Elimina</button>
          </li>
        </ul>
      </section>

      <section v-else-if="section === 'players'" class="card">
        <header class="section-header">
          <h2>Giocatori</h2>
          <p>Gestisci fino a {{ playerSlotCount }} giocatori da mostrare nella pagina di voto.</p>
        </header>

        <p v-if="!teams.length" class="info-banner">
          Aggiungi almeno una squadra per assegnare correttamente i giocatori salvati nel database.
        </p>

        <p v-if="playerOverflow.length" class="info-banner warning">
          Sono presenti {{ playerOverflow.length }} giocatori aggiuntivi nel database. Verranno rimossi al prossimo
          salvataggio.
        </p>

        <div class="player-slots">
          <fieldset
            v-for="(slot, index) in playerSlots"
            :key="`player-slot-${index}`"
            class="player-slot"
          >
            <legend>Giocatore {{ index + 1 }}</legend>
            <div class="player-slot__grid">
              <label>
                Nome
                <input v-model.trim="slot.first_name" type="text" placeholder="Es. Mario" />
              </label>
              <label>
                Cognome
                <input v-model.trim="slot.last_name" type="text" placeholder="Es. Rossi" />
              </label>
              <label>
                Ruolo
                <input v-model.trim="slot.role" type="text" placeholder="Es. Schiacciatore" />
              </label>
              <label>
                Numero di maglia
                <input
                  v-model="slot.jersey_number"
                  type="number"
                  min="0"
                  inputmode="numeric"
                  placeholder="Es. 7"
                />
              </label>
              <label>
                Squadra
                <select v-model.number="slot.team_id">
                  <option :value="0">Seleziona squadra</option>
                  <option v-for="team in teams" :key="team.id" :value="team.id">{{ team.name }}</option>
                </select>
              </label>
              <label>
                URL immagine (opzionale)
                <input
                  v-model.trim="slot.image_url"
                  type="url"
                  placeholder="https://..."
                  @input="handlePlayerUrlChange(index)"
                />
              </label>
              <label class="file-input">
                Oppure carica immagine
                <input type="file" accept="image/*" @change="handlePlayerImageChange(index, $event)" />
              </label>
              <div v-if="slot.image_preview" class="player-slot__preview" aria-label="Anteprima immagine giocatore">
                <img :src="slot.image_preview" alt="Anteprima giocatore" />
                <button class="btn link" type="button" @click="removePlayerImage(index)">Rimuovi</button>
              </div>
            </div>
          </fieldset>
        </div>

        <div class="actions-row">
          <button class="btn outline" type="button" @click="restorePlayerSlots" :disabled="isSavingPlayers">
            Ripristina dati salvati
          </button>
          <button class="btn primary" type="button" @click="savePlayers" :disabled="isSavingPlayers">
            {{ isSavingPlayers ? 'Salvataggio…' : 'Salva giocatori' }}
          </button>
        </div>

        <p v-if="playerSaveError" class="error">{{ playerSaveError }}</p>
        <p v-if="playerSaveMessage" class="success-message">{{ playerSaveMessage }}</p>
      </section>

      <section v-else-if="section === 'sponsors'" class="card">
        <header class="section-header">
          <h2>Sponsor</h2>
          <p>Gestisci fino a {{ maxSponsors }} sponsor da mostrare nella schermata pubblica.</p>
        </header>

        <div class="sponsor-controls" role="group" aria-label="Visibilità sponsor">
          <label class="sponsor-range">
            <span>Numero di sponsor visibili: {{ desiredActiveSponsorCount }} / {{ maxSponsors }}</span>
            <input
              type="range"
              min="0"
              :max="sponsorSliderMax"
              v-model.number="desiredActiveSponsorCount"
              @change="applyActiveSponsorCount"
              :disabled="!sponsors.length || isApplyingSponsorCount"
            />
          </label>
          <p class="muted small">Gli sponsor attivi vengono mostrati nell'ordine indicato qui sotto.</p>
        </div>

        <form @submit.prevent="createSponsor" class="form-grid sponsor-form">
          <label>
            Nome sponsor
            <input v-model.trim="newSponsor.name" type="text" placeholder="Es. Partner ufficiale" />
          </label>
          <label>
            Link (opzionale)
            <input v-model.trim="newSponsor.linkUrl" type="url" placeholder="https://example.com" />
          </label>
          <label class="file-input">
            Logo sponsor
            <input type="file" accept="image/*" @change="handleNewSponsorLogoChange" />
          </label>
          <div v-if="newSponsor.logoData" class="sponsor-preview new" aria-label="Anteprima logo nuovo sponsor">
            <img :src="newSponsor.logoData" alt="Anteprima logo sponsor" />
          </div>
          <button class="btn primary" type="submit" :disabled="isCreatingSponsor">
            {{ isCreatingSponsor ? 'Salvataggio…' : 'Aggiungi sponsor' }}
          </button>
        </form>

        <ul v-if="sponsors.length" class="item-list sponsors-list">
          <li v-for="sponsor in sponsors" :key="sponsor.id" class="item sponsor-item">
            <div class="item-body sponsor-body">
              <div class="sponsor-preview" :aria-label="`Logo sponsor ${sponsor.name || sponsor.position}`">
                <img
                  v-if="sponsor.logoData"
                  :src="sponsor.logoData"
                  :alt="`Logo ${sponsor.name || 'sponsor'}`"
                />
                <span v-else class="empty-logo">Logo non disponibile</span>
              </div>
              <div class="sponsor-fields">
                <div class="form-grid compact">
                  <label>
                    Nome sponsor
                    <input v-model.trim="sponsor.name" type="text" />
                  </label>
                  <label>
                    Link (opzionale)
                    <input v-model.trim="sponsor.linkUrl" type="url" placeholder="https://example.com" />
                  </label>
                  <label class="file-input">
                    Aggiorna logo
                    <input type="file" accept="image/*" @change="(event) => handleSponsorLogoChange(event, sponsor)" />
                  </label>
                </div>
                <p class="muted sponsor-meta">
                  Posizione {{ sponsor.position }} • {{ sponsor.isActive ? 'Visibile' : 'Nascosto' }}
                </p>
              </div>
            </div>
            <div class="item-actions vertical">
              <button
                class="btn secondary"
                type="button"
                @click="updateSponsorEntry(sponsor)"
                :disabled="sponsorBeingUpdated === sponsor.id"
              >
                <span v-if="sponsorBeingUpdated === sponsor.id">Salvataggio…</span>
                <span v-else>Salva</span>
              </button>
              <button
                class="btn danger"
                type="button"
                @click="deleteSponsorEntry(sponsor.id)"
                :disabled="sponsorBeingDeleted === sponsor.id"
              >
                <span v-if="sponsorBeingDeleted === sponsor.id">Eliminazione…</span>
                <span v-else>Elimina</span>
              </button>
            </div>
          </li>
        </ul>
        <p v-else class="muted text-center">Nessuno sponsor configurato al momento.</p>
        </section>

        <section v-else-if="section === 'admins'" class="card">
          <header class="section-header">
            <h2>Utenti amministratori</h2>
          </header>
          <form @submit.prevent="createAdmin" class="form-grid">
            <input v-model.trim="newAdmin.username" type="text" placeholder="Username" required />
            <input v-model="newAdmin.password" type="password" placeholder="Password" required />
            <input v-model.trim="newAdmin.role" type="text" placeholder="Ruolo (es. staff)" />
            <button class="btn primary" type="submit">Aggiungi</button>
          </form>
          <ul class="item-list compact">
            <li v-for="admin in admins" :key="admin.id" class="item">
              <div>
                <strong>{{ admin.username }}</strong>
                <span class="muted"> • {{ admin.role || 'staff' }}</span>
              </div>
              <button class="btn danger" type="button" @click="deleteAdmin(admin.id)">Elimina</button>
            </li>
          </ul>
        </section>
        <div
          v-if="purgeDialog.visible"
          class="modal-backdrop"
          role="dialog"
          aria-modal="true"
          :aria-label="purgeDialog.event ? `Conferma eliminazione per ${purgeDialog.event.title}` : 'Conferma eliminazione evento'"
        >
          <div class="modal-card">
            <h3>Elimina evento</h3>
            <p>Questa operazione è permanente e rimuoverà tutti i dati collegati all'evento.</p>
            <p class="muted">Conferma inserendo la password del super admin.</p>
            <p v-if="purgeDialog.error" class="error">{{ purgeDialog.error }}</p>
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
              <button class="btn outline" type="button" @click="closePurgeDialog" :disabled="purgeDialog.isSubmitting">
                Annulla
              </button>
              <button
                class="btn danger"
                type="button"
                @click="confirmPurge"
                :disabled="purgeDialog.isSubmitting || !purgeDialog.password"
              >
                {{ purgeDialog.isSubmitting ? 'Eliminazione…' : 'Elimina definitivamente' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';
import { PLAYER_LAYOUT } from '../roster';

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

let resultsPollHandle = 0;

const section = ref('events');
const tabs = [
  { id: 'events', label: 'Eventi' },
  { id: 'closing', label: 'Chiusura votazioni' },
  { id: 'results', label: 'Risultati' },
  { id: 'history', label: 'Storico eventi' },
  { id: 'teams', label: 'Squadre' },
  { id: 'players', label: 'Giocatori' },
  { id: 'sponsors', label: 'Sponsor' },
  { id: 'admins', label: 'Admin' },
];
const STAFF_TAB_IDS = new Set(['closing', 'results', 'history']);

const teams = ref([]);
const players = ref([]);
const events = ref([]);
const admins = ref([]);
const sponsors = ref([]);
const eventHistory = ref([]);
const isLoadingEventHistory = ref(false);
const eventHistoryError = ref('');
const eventHistorySuccess = ref('');
const hasLoadedEventHistory = ref(false);
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
const newEvent = reactive({
  team1_id: 0,
  team2_id: 0,
  start_datetime: '',
  location: '',
});
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

const handlePlayerImageChange = async (index, event) => {
  const slot = playerSlots[index];
  if (!slot) {
    return;
  }
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  const input = event?.target;
  const file = input?.files?.[0];
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
    if (input) {
      input.value = '';
    }
  }
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
const visibleEvents = computed(() => events.value.filter((event) => !event.is_concluded));

const activeEventId = computed(() => {
  const activeEvent = events.value.find((event) => event.is_active);
  return activeEvent ? activeEvent.id : 0;
});
const activeSponsorCount = computed(() => sponsors.value.filter((item) => item.isActive).length);
const sponsorSliderMax = computed(() =>
  sponsors.value.length ? Math.min(maxSponsors, sponsors.value.length) : maxSponsors,
);
const selectedResultsEvent = computed(() =>
  events.value.find((event) => event.id === selectedResultsEventId.value) || null,
);
const activeEventEntry = computed(() =>
  events.value.find((event) => event.id === activeEventId.value) || null,
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
  Object.assign(newEvent, {
    team1_id: 0,
    team2_id: 0,
    start_datetime: '',
    location: '',
  });
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
});
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
  hasLoadedEventHistory.value = false;
  eventHistoryError.value = '';
  eventHistorySuccess.value = '';
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
  lastCreatedEventLink.value = '';
  resetNewEventPrizes();
  resetResultsState();
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
}

function ensureResultsSelection() {
  if (!events.value.length) {
    selectedResultsEventId.value = 0;
    return;
  }
  const exists = events.value.some((event) => event.id === selectedResultsEventId.value);
  if (!exists) {
    const active = events.value.find((event) => event.is_active);
    selectedResultsEventId.value = active ? active.id : events.value[0].id;
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

async function handleSponsorLogoChange(event, targetSponsor) {
  const [file] = event?.target?.files || [];
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
  } finally {
    if (event?.target) {
      event.target.value = '';
    }
  }
}

async function handleNewSponsorLogoChange(event) {
  await handleSponsorLogoChange(event, newSponsor);
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

  return {
    id,
    title: rawTitle || fallbackTitle,
    startDatetime,
    location,
    totalVotes,
    totalVotesLabel: Number.isFinite(totalVotes) ? totalVotes.toLocaleString('it-IT') : '0',
    sponsorClicks,
    sponsorClicksTotal,
    sponsorClicksTotalLabel,
    timeline: timelineBuckets,
    timelineRange,
    mvp,
    homeTeam,
    awayTeam,
    prizes: normalizedPrizes,
    hasPrizeDraw,
  };
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
    prizes: prizesPayload,
  };

  const { data } = await secureRequest(() => apiClient.post('/events', payload, authHeaders.value));
  await loadEvents();
  if (data?.id) {
    lastCreatedEventLink.value = buildEventLink(data.id);
  }
  Object.assign(newEvent, {
    team1_id: 0,
    team2_id: 0,
    start_datetime: '',
    location: '',
  });
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
  } catch (error) {
    globalError.value = 'Impossibile copiare il link automaticamente.';
  }
}

function updateToolbarOffset() {
  if (!portalRef.value) {
    return;
  }
  const height = toolbarRef.value?.offsetHeight ?? 0;
  portalRef.value.style.setProperty('--toolbar-height', `${height}px`);
}

function ensureSectionIsAllowed(tabList) {
  if (!isAuthenticated.value) {
    return;
  }
  if (!tabList.some((tab) => tab.id === section.value)) {
    section.value = tabList.length ? tabList[0].id : '';
  }
}

onMounted(() => {
  window.addEventListener('resize', updateToolbarOffset, { passive: true });
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
  if (value === 'results') {
    ensureResultsSelection();
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (oldValue === 'results') {
    stopResultsPolling();
  }
  if (value === 'history') {
    loadEventHistory();
  }
  nextTick(updateToolbarOffset);
});

watch(selectedResultsEventId, (eventId) => {
  if (section.value === 'results' && eventId) {
    fetchEventResults({ showLoader: true });
    startResultsPolling();
  } else if (!eventId) {
    stopResultsPolling();
  }
});

if (isAuthenticated.value) {
  loadAll();
}

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateToolbarOffset);
  stopResultsPolling();
});
</script>

<style scoped>
.admin-portal {
  margin: 0 auto;
  max-width: 960px;
  padding: 2rem 1.5rem 3rem;
  color: #0f172a;
}

.admin-header {
  text-align: center;
  margin-bottom: 2rem;
  color: #f8fafc;
}

.admin-header h1 {
  font-size: 2rem;
  margin: 0;
  color: #f8fafc;
}

.subtitle {
  margin: 0.5rem 0 0;
  color: #cbd5f5;
}

.portal {
  display: flex;
  flex-direction: column;
  gap: 0;
  position: relative;
  --toolbar-height: 0px;
}

.toolbar {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  color: #f1f5f9;
  top: 0;
  z-index: 10;
  background: rgba(15, 23, 42, 0.92);
  border-radius: 1rem;
  padding: 0.75rem 1rem;
  box-shadow: 0 18px 45px rgba(15, 23, 42, 0.45);
  border: 1px solid rgba(148, 163, 184, 0.3);
  backdrop-filter: blur(12px);
}

@media (min-width: 768px) {
  .toolbar {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1.25rem;
  }
}

.portal-content {
  display: flex;
  flex-direction: column;
  padding-top: 1px;
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
  border-radius: 1rem;
  padding: 1.5rem;
  box-shadow: 0 15px 35px rgba(15, 23, 42, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.18);
}

.login-card {
  max-width: 480px;
  margin: 0 auto;
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
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  margin-bottom: 1.5rem;
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

input,
select {
  border-radius: 0.75rem;
  border: 1px solid #cbd5f5;
  padding: 0.65rem 0.85rem;
  font-size: 0.95rem;
  background: #f8fafc;
  color: #0f172a;
}

.field-hint {
  font-size: 0.75rem;
  color: #64748b;
}

input:focus,
select:focus {
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
  transition: transform 0.15s ease, box-shadow 0.15s ease;
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
  transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease,
    box-shadow 0.2s ease, transform 0.2s ease;
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

.item-body h3 {
  margin: 0 0 0.35rem;
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

.history-item__totals {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.75rem;
}

.history-item__total {
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

.history-sponsor-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
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

.sponsor-range input[type='range'] {
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
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.85), rgba(30, 64, 175, 0.9));
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

@media (max-width: 640px) {
  .leaderboard-item {
    grid-template-columns: 56px minmax(0, 1fr);
  }

  .leaderboard-item .votes {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}
</style>

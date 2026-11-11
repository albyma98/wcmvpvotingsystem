<template>
  <div class="cento-root" :style="themeStyle">
    <div class="phone" :inert="isHomeLoading || Boolean(homeVoteFlight)" :aria-hidden="isHomeLoading || Boolean(homeVoteFlight) || undefined" :aria-busy="isHomeLoading">
      <div class="home-scroll" :class="{ 'has-sponsors': sponsors.length > 0 }">
        <header class="topbar">
          <div class="club">
            <div class="club-logo" :class="{ 'has-logo': homeLogoUrl }">
              <img v-if="homeLogoUrl" :src="homeLogoUrl" :alt="homeName" />
              <template v-else>{{ homeBadge }}</template>
            </div>
            <div class="club-name">
              <b>{{ homeName }}</b>
              <span v-if="opponentLabel">{{ opponentLabel }}</span>
            </div>
          </div>
          <div class="top-right">
            <button ref="platformInfoTriggerRef" class="platform-info-button" type="button" aria-label="Informazioni e istruzioni" title="Come funziona ArenaBoostX" aria-haspopup="dialog" aria-controls="fan-platform-info" :aria-expanded="activeSheet === 'info'" @click="openPlatformInfo">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7h.01"/></svg>
            </button>
            <button ref="walletEl" class="wallet" :class="{ 'wallet-pulse': walletPulse }" type="button" :aria-label="`Apri wallet e movimenti, ${totalCoins} monete`" @click="openHomeAction('wallet')">
              <span class="coin" aria-hidden="true"></span> <span>{{ totalCoins }}</span>
            </button>
            <button class="profile-btn" type="button" :aria-label="isRegistered ? 'Profilo tifoso' : 'Accedi o crea il profilo tifoso'" @click="openHomeAction('profile')">
              <FanAvatar v-if="isRegistered" :name="profileNickname" :framed="equippedFrame === 'club_frame'" :size="40" />
              <span v-else class="guest-avatar" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><circle cx="12" cy="8" r="3.5"/><path d="M5 21v-2a7 7 0 0 1 14 0v2"/></svg></span>
            </button>
          </div>
        </header>

        <main class="home-dashboard">
          <section class="mvp-card" :class="{ 'is-confirmed': showHomeVoteReceipt }" aria-labelledby="home-mvp-title">
            <div v-if="showHomeVoteReceipt" class="mvp-receipt" :class="{ 'is-new': highlightPlay }">
              <div ref="homeVoteLanding" class="mvp-thumbnail" :class="{ 'is-in-flight': homeVoteFlight }" aria-hidden="true">
                <FanPlayerCard v-if="votedPlayerCard.fullName !== 'Il tuo MVP' || displayVotedImage" :player="votedPlayerCard" :club-logo-url="homeLogoUrl" loading="eager" />
                <span v-else class="mvp-check">✓</span>
              </div>
              <div class="mvp-receipt-copy">
                <h1 id="home-mvp-title" ref="homeMvpTitle" tabindex="-1" :aria-describedby="votedPlayerCard.fullName !== 'Il tuo MVP' ? 'home-mvp-player' : undefined">Hai votato il tuo MVP!</h1>
                <p v-if="votedPlayerCard.fullName !== 'Il tuo MVP'" id="home-mvp-player">{{ votedPlayerCard.fullName }}</p>
                <p v-if="homeVoteReward" class="mvp-receipt-reward" role="status">+{{ homeVoteReward }} monete <span class="coin" aria-hidden="true"></span></p>
              </div>
              <button v-if="showMatchSummary" class="mvp-secondary" type="button" @click="openHomeAction('match-summary')">Riepilogo</button>
              <button v-else-if="canVote" class="mvp-secondary" type="button" :disabled="isVoting || shopBusy" @click="editHomeVote">Modifica voto</button>
            </div>
            <div v-else class="mvp-heading">
              <h1 id="home-mvp-title" ref="homeMvpTitle" tabindex="-1">Chi è il tuo MVP?</h1>
              <span class="mvp-status" :class="{ 'is-open': canVote }">{{ mvpStatusLabel }}</span>
            </div>
            <Transition name="mvp-collapse" @before-leave="measureMvpCollapse">
              <div v-if="!showHomeVoteReceipt" class="mvp-content">
                <template v-if="canVote">
                  <p v-if="isLoadingPlayers || isLoadingVoteStatus" role="status">Caricamento votazione…</p>
                  <template v-else-if="playersError || voteStatusError">
                    <p role="alert">{{ playersError || voteStatusError }}</p>
                    <button class="mvp-action" type="button" @click="playersError ? loadPlayers() : loadVoteStatus()">Riprova</button>
                  </template>
                  <p v-else-if="!players.length">Nessun giocatore disponibile.</p>
                  <template v-else>
                    <p>{{ homeMvpPlayers.length > 1 ? 'Scorri e conferma dalla card.' : 'Seleziona la card, poi premi «Conferma».' }}</p>
                    <MvpPlayerCarousel
                      :key="walletScope()"
                      ref="homeMvpCarousel"
                      :players="homeMvpPlayers"
                      :voted-player-id="votedPlayerId"
                      :selected-player-id="homeMvpSelectedId"
                      :club-logo-url="homeLogoUrl"
                      :disabled="isVoting || shopBusy"
                      :can-confirm="canConfirmHomeVote"
                      :submitting="isVoting"
                      @select="selectHomeMvp"
                      @confirm="confirmHomeVote"
                    />
                    <p v-if="voteError" class="mvp-error" role="alert">{{ voteError }}</p>
                  </template>
                  <button v-if="mvpEditing" class="mvp-secondary" type="button" :disabled="isVoting" @click="cancelHomeVoteEdit">Annulla modifica</button>
                </template>
                <template v-else>
                  <p :role="mvpConnectionError ? 'status' : undefined">{{ mvpUnavailableMessage }}</p>
                  <button v-if="showMatchSummary" class="mvp-action" type="button" @click="openHomeAction('match-summary')">Vedi il riepilogo</button>
                  <button v-else-if="mvpConnectionError" class="mvp-action" type="button" @click="eventId ? refreshLiveStatus() : $emit('retry-active-event')">Riprova</button>
                </template>
              </div>
            </Transition>
          </section>

          <div class="home-shortcuts">
            <button ref="homePlayButton" class="home-shortcut" :class="{ 'is-vote-next': highlightPlay && !matchConcluded && eventState?.is_active !== false }" type="button" @click="openHomeAction('earn')">
              <span class="home-icon" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m13 2-9 12h6l-1 8 9-12h-6l1-8Z"/></svg></span>
              <span class="shortcut-copy"><b>Gioca</b><span>{{ highlightPlay && featuredMission ? 'Continua a guadagnare monete' : 'Sfide e quiz' }}</span></span>
            </button>
            <FanMerchLink class="home-shortcut" :slug="organizationSlug" sheet @open="openMerchShop">
              <span class="home-icon" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 8h14l2 13H3L5 8Z"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg></span>
              <span class="shortcut-copy"><b>Shop</b><span>Merch ufficiale</span></span>
            </FanMerchLink>
          </div>

          <Transition name="home-slot" mode="out-in">
            <section v-if="hasFeedbackSurvey && !hasSubmittedFeedback" key="inline-feedback" :ref="observeFeedbackInvitation" class="feedback-cta feedback-inline" aria-labelledby="inline-feedback-title" :aria-busy="feedbackSaving">
              <h2 id="inline-feedback-title">La tua voce conta</h2>
              <h3 id="inline-feedback-question" ref="feedbackQuestionTitle" tabindex="-1">{{ activeFeedbackQuestion.title }}</h3>
              <div v-if="activeFeedbackQuestion.id === 'suggestion'" class="feedback-suggestion">
                <textarea v-model="feedbackSuggestion" maxlength="80" rows="2" aria-labelledby="inline-feedback-question" :disabled="feedbackSaving" />
                <small>{{ feedbackSuggestion.length }}/80</small>
              </div>
              <div v-else :key="activeFeedbackQuestion.id" :class="feedbackOptions.length ? 'feedback-emoji-options' : 'feedback-answer-options'" role="group" aria-labelledby="inline-feedback-question">
                <button
                  v-for="option in feedbackOptions.length ? feedbackOptions : activeFeedbackQuestion.answers"
                  :key="option.value"
                  type="button"
                  :aria-label="option.emoji ? option.label : undefined"
                  :aria-pressed="(feedbackPendingAnswer ?? feedbackDraft.answers[activeFeedbackQuestion.id]) === option.value"
                  :disabled="feedbackSaving"
                  @click="answerFeedbackInline(option.value)"
                ><span aria-hidden="true">{{ option.emoji || option.icon }}</span><span v-if="!option.emoji">{{ option.label }}</span></button>
              </div>
              <p v-if="feedbackError" class="feedback-error" role="alert">{{ feedbackError }}</p>
              <div v-if="feedbackStep > 0 || activeFeedbackQuestion.id === 'suggestion'" class="feedback-inline-actions">
                <button v-if="feedbackStep > 0" type="button" :disabled="feedbackSaving" @click="previousFeedbackQuestion">Indietro</button>
                <template v-if="activeFeedbackQuestion.id === 'suggestion'">
                  <button type="button" :disabled="feedbackSaving" @click="answerFeedbackInline('')">Salta</button>
                  <button type="button" :disabled="feedbackSaving" @click="answerFeedbackInline(feedbackSuggestion)">{{ feedbackSaving ? 'Salvataggio…' : 'Invia' }}</button>
                </template>
              </div>
            </section>
            <button v-else-if="featuredMission" key="challenge" ref="homeChallengeButton" class="moment-card" type="button" @click="onMission(featuredMission, 'home_featured')">
              <span class="home-icon" aria-hidden="true">{{ featuredMission.emoji }}</span>
              <span class="moment-copy"><span>Sfida del momento</span><b>{{ featuredMission.title }}</b><span>{{ feedbackThanks ? 'Grazie per il feedback!' : featuredMission.repeatable ? featuredMission.subtitle : nextQuizLabel || 'Una breve sfida da giocare durante le pause' }}</span></span>
              <span class="moment-action">
                <span v-if="featuredMission.pts && (featuredMission.repeatable || !doneKeys.includes(featuredMission.key))" class="row-pts">
                  +{{ featuredMission.pts }}<span class="sr-only"> monete</span><span class="coin" aria-hidden="true"></span>
                </span>
                <b>Gioca <span aria-hidden="true">→</span></b>
              </span>
            </button>
            <p v-else-if="feedbackThanks" key="feedback-thanks" class="feedback-thanks">Grazie per il feedback!</p>
          </Transition>

          <section v-if="eventId" class="journey-card" aria-label="Progressione del badge">
            <div class="journey-head">
              <b>{{ FAN_BADGE_NAME }}<small v-if="progress.unlocked">Sbloccato</small></b>
              <strong>{{ progress.current }}/{{ FIRST_FAN_GOAL }} <span class="coin" aria-hidden="true"></span></strong>
            </div>
            <div class="journey-bar" role="progressbar" :aria-label="`Monete guadagnate per il badge ${FAN_BADGE_NAME}`" :aria-valuenow="progress.current" :aria-valuemin="0" :aria-valuemax="FIRST_FAN_GOAL">
              <span :style="{ width: `${progress.percent}%` }"></span>
            </div>
            <button v-if="journey.pendingVoteBonus" class="journey-next" type="button" :disabled="isSavingCoins" @click="retryVoteBonus">{{ isSavingCoins ? 'Salvataggio…' : 'Riprova a salvare il bonus voto' }}</button>
          </section>

          <MatchChatEntry v-if="eventId && organizationSlug" :key="`${organizationSlug}:${eventId}`"
            :slug="organizationSlug" :event-id="eventId" :event-label="matchChatEventLabel"
            @verify="openRegistration()" @track="recordMatchChat" />

          <button class="rank-link" type="button" @click="openHomeAction('leaderboard')">
            <svg class="rank-link-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 3h8v6a4 4 0 0 1-8 0V3Zm0 2H4v2a4 4 0 0 0 4 4m8-6h4v2a4 4 0 0 1-4 4m-4 2v4m-4 4h8m-7 0v-2a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2"/></svg>
            <b>Classifica tifosi</b>
            <svg class="rank-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 6 6 6-6 6"/></svg>
          </button>

          <button v-if="showLotteryShortcut" :ref="observeLotteryShortcut" class="lottery-shortcut" type="button" :aria-label="`Apri il QR della lotteria, codice ${lotteryCode}`" @click="openLotteryTicket">
            <span>Il tuo codice: <strong>{{ lotteryCode }}</strong></span>
            <span class="lottery-shortcut-action">Apri QR <span aria-hidden="true">→</span></span>
          </button>
        </main>
      </div>

      <section v-if="sponsors.length" class="sponsors" aria-labelledby="home-partners-title" :inert="Boolean(activeSheet || activeGameId)">
        <h2 id="home-partners-title" class="sponsors-head">PARTNER UFFICIALI</h2>
        <div class="sponsor-grid">
          <SponsorBanner
            v-for="sponsor in sponsors.slice(0, 6)"
            :key="sponsor.id"
            tag="button"
            :sponsor="sponsor"
            placement="home_footer"
            :enabled="!isRegistrationOpen"
            :clickable="/^https?:\/\//i.test(sponsor.linkUrl)"
            class="sponsor-chip"
            type="button"
            @click="onSponsorClick(sponsor)"
          >
            <img v-if="sponsor.imageUrl" :src="sponsor.imageUrl" :alt="sponsor.name || 'Sponsor'" />
            <span v-else>{{ sponsor.name }}</span>
          </SponsorBanner>
        </div>
      </section>

      <!-- Overlay condiviso dai pannelli della home -->
      <div class="overlay" :class="{ open: activeSheet }" @click="closeSheets"></div>

      <section id="fan-platform-info" ref="platformInfoRef" class="sheet platform-info-sheet" :class="{ open: activeSheet === 'info' }" role="dialog" :aria-modal="activeSheet === 'info' ? 'true' : undefined" aria-labelledby="fan-platform-info-title" aria-describedby="fan-platform-info-intro" :aria-hidden="activeSheet !== 'info'" :inert="activeSheet !== 'info'" tabindex="-1" @keydown.esc.stop.prevent="closeSheets" @keydown.tab="trapPlatformInfoFocus">
        <div class="grabber" aria-hidden="true"></div>
        <div class="sheet-head">
          <div><span class="platform-info-kicker">COME FUNZIONA ARENABOOSTX</span><h2 id="fan-platform-info-title" class="sheet-title">La tua passione entra in gioco.</h2></div>
          <button class="close" type="button" aria-label="Chiudi informazioni e istruzioni" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body platform-info-body">
          <p id="fan-platform-info-intro" class="platform-info-intro">Vota, gioca e vivi la partita insieme al tuo club.</p>
          <ul class="platform-info-features">
            <li><span class="platform-info-icon" aria-hidden="true">🏆</span><div><h3>Scegli il tuo MVP</h3><p>Vota il protagonista della partita quando le votazioni sono aperte.</p></div></li>
            <li><span class="platform-info-icon" aria-hidden="true">🎮</span><div><h3>Gioca e guadagna monete</h3><p>Puoi rigiocare ai minigiochi quante volte vuoi. Ogni nuova giocata può darti altre monete, secondo le regole del gioco.</p></div></li>
            <li><span class="platform-info-icon" aria-hidden="true">🪙</span><div><h3>Crea la tua collezione</h3><p>Usa le monete per ottenere cornici e card ricordo disponibili nella piattaforma.</p></div></li>
            <li><span class="platform-info-icon" aria-hidden="true">📈</span><div><h3>Sali in classifica</h3><p>Guadagnando monete accumuli anche punti partecipazione. Spendere le monete non riduce i punti in classifica.</p></div></li>
            <li><span class="platform-info-icon" aria-hidden="true">👤</span><div><h3>Porta con te i tuoi progressi</h3><p>Crea il profilo tifoso per ritrovare saldo, collezione e progressi quando accedi. Il profilo include la partecipazione alle iniziative gadget del club.</p></div></li>
          </ul>
        </div>
        <div class="platform-info-actions"><button class="btn-vote" type="button" @click="closeSheets">Ho capito</button></div>
      </section>

      <!-- MVP: lista giocatori reale -->
      <div class="sheet" :class="{ open: activeSheet === 'mvp' }" role="dialog" aria-label="Vota MVP" :aria-hidden="activeSheet !== 'mvp'" :inert="activeSheet !== 'mvp'">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Vota l'MVP del pubblico</div>
            <div class="sheet-sub">{{ hasVoted ? 'Scegli un’altra card e conferma la modifica.' : 'Scegli il tuo campione e conferma il voto.' }}</div>
          </div>
          <button class="close" type="button" aria-label="Chiudi scelta MVP" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div v-if="canVote && !hasVoted && !doneKeys.includes('vote')" class="mvp-vote-invite" :class="{ 'is-active': activeSheet === 'mvp' }">
            <span class="mvp-invite-star" aria-hidden="true">★</span>
            <span><b>Il tuo primo voto vale</b><small>Scegli una card e porta il tuo tifo in classifica</small></span>
            <strong>+{{ VOTE_REWARD }} <span class="coin" aria-hidden="true"></span><small>monete</small></strong>
          </div>
          <div v-if="isLoadingPlayers" class="sheet-empty">Caricamento giocatori…</div>
          <div v-else-if="playersError" class="sheet-empty error">{{ playersError }}</div>
          <div v-else-if="!players.length" class="sheet-empty">Nessun giocatore disponibile.</div>
          <p v-if="voteError" class="sheet-empty error" role="alert">{{ voteError }}</p>
          <FanMvpRoster
            v-if="players.length && !isLoadingPlayers && !playersError"
            :players="players"
            :active="activeSheet === 'mvp'"
            :selected-player-id="votedPlayerId"
            :pending-player-id="pendingVotePlayerId"
            :busy="isVoting"
            :can-vote="canVote && !shopBusy"
            :club-logo-url="homeLogoUrl"
            @select="missionMvpCandidate = $event"
          />
          <button
            v-if="missionMvpCandidate && canVote"
            class="btn-vote"
            type="button"
            :disabled="isVoting || shopBusy || missionMvpCandidate.id === votedPlayerId"
            @click="castVote(missionMvpCandidate)"
          >{{ isVoting ? 'Invio del voto…' : `Conferma voto per ${missionMvpCandidate.fullName}` }}</button>
        </div>
      </div>

      <div class="sheet vote-result-sheet" :class="{ open: activeSheet === 'vote-result' }" role="dialog" :aria-label="celebrationKind === 'vote' ? 'Conferma del voto' : 'Monete guadagnate'" :aria-hidden="activeSheet !== 'vote-result'" :inert="activeSheet !== 'vote-result'">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div class="sheet-title">{{ celebrationKind === 'vote' ? 'Voto registrato' : 'Sfida completata!' }}</div>
          <button class="close" type="button" aria-label="Chiudi conferma voto" @click="closeSheets">✕</button>
        </div>
        <div ref="voteResultEl" class="sheet-body vote-result" :class="{ 'is-mvp-result': celebrationKind === 'vote' && activeSheet === 'vote-result' }">
          <div class="vote-result-summary">
            <FanMvpCelebration v-if="celebrationKind === 'vote' && activeSheet === 'vote-result'" :key="voteRevealRevision" :player="votedPlayerCard" :club-logo-url="homeLogoUrl" compact />
            <span v-else class="vote-result-avatar" aria-hidden="true">{{ progress.unlocked ? '🏅' : '🎉' }}</span>
            <div class="vote-result-copy">
              <h2>{{ celebrationKind === 'vote' ? `Hai votato ${displayVotedName || 'il tuo MVP'}!` : `${celebrationName} completato!` }}</h2>
              <div class="vote-result-status" role="status" aria-live="polite" aria-atomic="true">
                <div v-if="voteBonusCoins" :key="`${celebrationKind}-${voteRevealRevision}-${voteBonusCoins}`" class="vote-result-bonus">
                  <span class="vote-reward-amount">+{{ voteBonusCoins }} <span ref="voteBonusEl" class="coin" aria-hidden="true"></span></span>
                  <span class="vote-reward-copy"><b>{{ celebrationKind === 'vote' ? 'Il tuo tifo vale!' : 'Bonus conquistato!' }}</b><small>Monete salvate · +{{ voteBonusCoins }} punti partecipazione</small></span>
                </div>
                <p v-else-if="journey.pendingVoteBonus">{{ isSavingCoins ? 'Il tuo voto è al sicuro. Salvataggio delle monete…' : `Voto registrato. Il bonus di ${VOTE_REWARD} monete è ancora da salvare.` }}</p>
                <p v-else>Voto aggiornato. Il bonus del primo voto è già stato assegnato.</p>
              </div>
            </div>
          </div>
          <div v-if="showVoteProfilePrompt" class="vote-profile-prompt">
            <span class="vote-profile-eyebrow">IL TUO TIFO MERITA UN PROFILO</span>
            <h3>Crea il tuo profilo tifoso</h3>
            <ul><li><span aria-hidden="true">🪙</span>Salva le tue monete</li><li><span aria-hidden="true">🏆</span>Entra nella classifica tifosi</li><li><span aria-hidden="true">🎁</span><strong>Potrai essere selezionato per i gadget</strong></li></ul>
          </div>
          <div v-else class="vote-result-goal" :class="{ 'is-unlocked': progress.unlocked }">
            <div class="vote-goal-heading"><span class="vote-goal-medal" aria-hidden="true">{{ progress.unlocked ? '🏅' : '★' }}</span><span><small>{{ progress.unlocked ? 'TRAGUARDO RAGGIUNTO' : 'IL TUO PROSSIMO TRAGUARDO' }}</small><b>{{ progress.unlocked ? `Badge ${FAN_BADGE_NAME} sbloccato!` : `Mancano ${progress.remaining} monete al primo badge` }}</b></span><strong>{{ progress.current }}<small>/{{ FIRST_FAN_GOAL }}</small></strong></div>
            <div class="journey-bar"><span :key="`${celebrationKind}-${voteRevealRevision}-${progress.current}`" :class="{ 'is-rewarded': voteBonusCoins > 0 }" :style="{ width: `${progress.percent}%`, '--goal-from': `${Math.min(100, Math.max(0, progress.earned - voteBonusCoins) / FIRST_FAN_GOAL * 100)}%` }"></span></div>
          </div>
          <p v-if="coinError" class="sheet-empty error" role="alert">{{ coinError }}</p>
        </div>
        <div class="vote-result-actions">
          <button v-if="showVoteProfilePrompt" class="btn-vote" type="button" :disabled="shopBusy" @click="openRegistration('after_vote')">Crea il profilo tifoso <span aria-hidden="true">→</span></button>
          <button v-else-if="journey.pendingVoteBonus" class="btn-vote" type="button" :disabled="isSavingCoins" @click="retryVoteBonus">{{ isSavingCoins ? 'Salvataggio…' : 'Riprova a salvare le monete' }}</button>
          <button v-else class="btn-vote" type="button" @click="openJourneyNext">{{ journeyNextLabel }}</button>
          <div class="vote-result-secondary">
            <button v-if="showVoteProfilePrompt && journey.pendingVoteBonus" class="journey-next" type="button" :disabled="isSavingCoins" @click="retryVoteBonus">{{ isSavingCoins ? 'Salvataggio…' : 'Riprova le monete' }}</button>
            <button class="journey-next" type="button" @click="closeSheets">Torna alla partita</button>
          </div>
        </div>
      </div>

      <!-- Guadagna: missioni del match -->
      <div class="sheet earn-sheet" :class="{ open: activeSheet === 'earn' }" role="dialog" aria-label="Guadagna monete" :aria-hidden="activeSheet !== 'earn'" :inert="activeSheet !== 'earn'">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Guadagna monete</div>
            <div class="sheet-sub">Scegli una sfida e guadagna monete</div>
          </div>
          <button class="close" type="button" aria-label="Chiudi minigiochi" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <button class="earn-collection-link" type="button" @click="openHomeAction('spend', 'game_catalog')">Shop Tifoso <span aria-hidden="true">→</span></button>
          <div class="mission-grid" :class="{ 'is-active': activeSheet === 'earn' }">
            <button
              v-for="(mission, index) in earnMissions"
              :key="mission.key"
              :ref="(element) => observeReportMission(element, mission)"
              class="mission-card"
              :class="{ 'is-completed': !mission.repeatable && (doneKeys.includes(mission.key) || (mission.key === 'vote' && hasVoted)) }"
              :style="{ '--mission-delay': `${index * 25}ms` }"
              type="button"
              :title="mission.title"
              :disabled="matchConcluded || (mission.opensMvp && !canVote) || (mission.key === 'quiz' && !quizAvailable)"
              @click="onMission(mission)"
            >
              <span class="mission-card-cover" aria-hidden="true">
                <img class="mission-card-artwork" :src="FAN_GAME_ARTWORK[mission.key]" alt="" width="320" height="320" decoding="async" />
              </span>
              <b v-sponsor-placement="mission.key === 'sponsor-rush' && {
                sponsor: sponsors[0], placement: 'sponsor_rush_entry', game: 'sponsor-rush',
                creativeVariant: 'text', enabled: activeSheet === 'earn' && !activeGameId && !isRegistrationOpen,
                endWhenDisabled: activeSheet !== 'earn',
              }" class="mission-card-title">{{ mission.title }}</b>
              <span class="mission-card-description">{{ !mission.repeatable && (doneKeys.includes(mission.key) || (mission.key === 'vote' && hasVoted)) ? 'Bonus già ricevuto' : mission.subtitle }}</span>
              <span class="mission-card-reward">
                <template v-if="mission.key === 'quiz' && !quizAvailable">Non disponibile</template>
                <template v-else-if="mission.repeatable">+{{ mission.pts }} <span class="coin" aria-hidden="true"></span><small>/ {{ mission.rewardUnit || 'partita' }}</small></template>
                <template v-else-if="doneKeys.includes(mission.key) || (mission.key === 'vote' && hasVoted)"><span aria-hidden="true">✓</span> Ricevuto</template>
                <template v-else>+{{ mission.pts }} <span class="coin" aria-hidden="true"></span></template>
              </span>
            </button>
          </div>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'wallet' }" role="dialog" aria-label="Il tuo wallet" :aria-hidden="activeSheet !== 'wallet'" :inert="activeSheet !== 'wallet'">
        <div class="grabber"></div>
        <div class="sheet-head"><div><div class="sheet-title">Il tuo wallet</div><div class="sheet-sub">Le monete che guadagni votando e giocando</div></div><button class="close" type="button" aria-label="Chiudi wallet" @click="closeSheets">✕</button></div>
        <div class="sheet-body">
          <div class="wallet-total"><span class="coin" aria-hidden="true"></span><strong>{{ totalCoins }}</strong><span>monete disponibili</span></div>
          <div class="wallet-actions"><button type="button" class="btn-vote" @click="openSheet('earn')">Guadagna</button><button type="button" class="btn-vote" @click="openSheet('spend')">Usa le monete</button></div>
          <p class="sheet-sub">{{ hasLoadedCollection ? participationPoints : '—' }} punti partecipazione. Ogni moneta guadagnata vale 1 punto: i punti restano anche quando spendi.</p>
          <h3 class="section-label">Ultimi movimenti</h3>
          <p v-if="isLoadingWallet" class="sheet-empty" role="status">Caricamento wallet…</p>
          <p v-else-if="walletError" class="sheet-empty error" role="alert">{{ walletError }} <button type="button" class="journey-next" @click="loadWallet">Riprova</button></p>
          <p v-else-if="!walletMovements.length" class="sheet-empty">I prossimi guadagni e acquisti compariranno qui.</p>
          <div v-for="movement in walletMovements" :key="movement.id" class="wallet-movement">
            <span><b>{{ movement.label }}</b><small>{{ formatWalletDate(movement.created_at) }}</small></span>
            <strong :class="{ 'is-credit': movement.delta > 0 }">{{ movement.delta > 0 ? '+' : '' }}{{ movement.delta }} <span class="coin" aria-hidden="true"></span></strong>
          </div>
          <p v-if="walletMovements.length" class="sheet-sub">Lo storico mostra i movimenti registrati dall’attivazione del wallet.</p>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'spend' }" role="dialog" aria-label="Shop Tifoso" :aria-hidden="activeSheet !== 'spend'" :inert="activeSheet !== 'spend'">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Shop Tifoso</div>
            <div class="sheet-sub">Oggetti digitali per vivere e ricordare la partita</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div class="shop-balance"><span>Saldo disponibile</span><b>{{ totalCoins }} <span class="coin" aria-hidden="true"></span></b></div>
          <p v-if="isLoadingWallet" class="sheet-empty" role="status">Caricamento catalogo…</p>
          <p v-if="walletError" class="sheet-empty error" role="alert">{{ walletError }} <button type="button" class="journey-next" @click="loadWallet">Riprova</button></p>
          <p v-if="shopMessage" class="shop-message" role="status">{{ shopMessage }}</p>
          <div v-if="activeSheet === 'spend'" class="shop-gallery">
            <article v-for="item in catalog" :key="item.key" class="shop-item">
              <button class="shop-artwork" :class="item.kind" type="button" :aria-label="`Guarda ${item.title}`" :disabled="shopBusy" @click="previewCollectible(item)">
                <template v-if="item.kind === 'frame'"><FanAvatar :name="profileNickname" framed :size="146" /><span class="shop-wear-preview"><FanAvatar :name="profileNickname" framed :size="32" /><b>{{ profileNickname }}</b></span></template>
                <FanMatchCard v-else :memory="shopMatchMemory" :club-name="homeName" :club-logo-url="homeLogoUrl" :sport="organizationSport" compact />
                <span class="shop-artwork-hint">{{ item.kind === 'frame' ? 'Provala sul tuo profilo' : 'Guarda da vicino' }} <span aria-hidden="true">↗</span></span>
              </button>
              <div class="shop-item-copy"><b>{{ item.title }}</b><p>{{ item.description }}</p><button type="button" class="shop-buy" :disabled="shopBusy || isLoadingWallet || Boolean(walletError) || (isRegistered && !item.owned && totalCoins < item.cost)" @click="item.owned ? previewCollectible(item) : redeem(item)">{{ item.owned ? 'Apri il tuo oggetto' : redeeming === item.key ? 'Salvataggio…' : `Ottieni · ${item.cost}` }} <span v-if="!item.owned" class="coin" aria-hidden="true"></span></button><small v-if="!item.owned && totalCoins < item.cost">Ti mancano {{ item.cost - totalCoins }} monete</small></div>
            </article>
          </div>
          <p v-if="!catalog.length && !isLoadingWallet && !walletError" class="sheet-empty">Il catalogo sarà disponibile con una partita del club.</p>
          <button type="button" class="journey-next" @click="openProfile">Apri la tua collezione →</button>
        </div>
      </div>

      <div class="sheet collectible-sheet" :class="{ open: activeSheet === 'collectible' }" role="dialog" :aria-modal="activeSheet === 'collectible' ? 'true' : undefined" :aria-label="selectedCollectible?.title || 'Il tuo oggetto'" :aria-hidden="activeSheet !== 'collectible'" :inert="activeSheet !== 'collectible'">
        <div class="grabber"></div>
        <div class="sheet-head"><div><div class="sheet-title">{{ selectedCollectible?.title }}</div><div class="sheet-sub" role="status">{{ isCollectibleNew ? 'Aggiunta alla tua collezione!' : isCollectiblePreview ? 'Anteprima' : 'Nella tua collezione' }}</div></div><button type="button" class="close" aria-label="Chiudi oggetto" @click="closeCollectible">✕</button></div>
        <div v-if="selectedCollectible && activeSheet === 'collectible'" class="sheet-body collectible-body">
          <div :key="collectibleRevealRevision" class="collectible-stage" :class="{ 'is-new': isCollectibleNew, 'is-frame': selectedCollectible.kind === 'frame' }">
            <template v-if="selectedCollectible.kind === 'frame'"><FanAvatar :name="profileNickname" framed :size="210" /><b class="collectible-fan-name">{{ profileNickname }}</b><span class="collectible-club-name">{{ homeName }}</span></template>
            <FanMatchCard v-else :memory="selectedCollectible" :club-name="homeName" :club-logo-url="homeLogoUrl" :sport="organizationSport" />
          </div>
          <p v-if="selectedCollectible.kind === 'frame'" class="collectible-description">La tua cornice sull’avatar, nel profilo e in classifica.</p>
          <p v-else class="collectible-description">Il ricordo della tua partita, da ritrovare nella collezione.</p>
          <p v-if="walletError" class="sheet-empty error" role="alert">{{ walletError }}</p>
          <button v-if="isCollectiblePreview" type="button" class="btn-vote" :disabled="shopBusy || isLoadingWallet || Boolean(walletError) || (isRegistered && totalCoins < Number(selectedCollectible.cost))" @click="redeem(selectedCollectible)">{{ redeeming ? 'Salvataggio…' : `Ottieni · ${selectedCollectible.cost} monete` }}</button>
          <button v-else-if="selectedCollectible.kind === 'frame'" type="button" class="btn-vote" :disabled="shopBusy" @click="equipCollectionItem(equippedFrame === 'club_frame' ? '' : 'club_frame')">{{ equipping ? 'Salvataggio…' : equippedFrame === 'club_frame' ? 'Togli la cornice' : 'Indossa la cornice' }}</button>
          <p v-if="isCollectiblePreview && totalCoins < Number(selectedCollectible.cost)" class="sheet-sub">Ti mancano {{ Number(selectedCollectible.cost) - totalCoins }} monete.</p>
          <button type="button" class="journey-next" @click="isCollectiblePreview ? closeCollectible() : openProfile()">{{ isCollectiblePreview ? 'Torna allo shop' : 'Apri il tuo profilo →' }}</button>
        </div>
      </div>

      <!-- Classifica tifosi: stesso pannello dal basso delle altre funzioni -->
      <div
        class="sheet"
        :class="{ open: activeSheet === 'leaderboard' }"
        role="dialog"
        aria-label="Classifica tifosi"
        :aria-modal="activeSheet === 'leaderboard' ? 'true' : undefined"
        :aria-hidden="activeSheet !== 'leaderboard'"
        :inert="activeSheet !== 'leaderboard'"
      >
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Classifica tifosi</div>
            <div class="sheet-sub">Punti guadagnati partecipando · spendere monete non cambia la posizione</div>
          </div>
          <button class="close" type="button" aria-label="Chiudi classifica tifosi" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div class="leaderboard-periods" role="group" aria-label="Periodo classifica">
            <button type="button" :class="{ active: leaderboardPeriod === 'general' }" :aria-pressed="leaderboardPeriod === 'general'" @click="selectLeaderboardPeriod('general')">Generale</button>
            <button type="button" :class="{ active: leaderboardPeriod === 'event' }" :aria-pressed="leaderboardPeriod === 'event'" @click="selectLeaderboardPeriod('event')">Questa giornata</button>
          </div>
          <p class="sheet-sub leaderboard-period-description">{{ leaderboardPeriod === 'event' ? 'Punti guadagnati in questa partita' : 'Punti totali guadagnati nella società' }}</p>
          <div v-if="isLoadingLeaderboard && !leaderboardRows.length" class="sheet-empty">Caricamento classifica…</div>
          <div v-else-if="leaderboardError" class="sheet-empty error" role="status">{{ leaderboardError }}</div>
          <div v-else-if="!leaderboardRows.length" class="sheet-empty">
            {{ eventId ? 'Gioca e vota per popolare la classifica.' : 'La classifica sarà disponibile con un evento attivo.' }}
          </div>
          <ol v-if="leaderboardRows.length && !leaderboardError" class="leaderboard-list" :aria-label="leaderboardPeriod === 'event' ? 'Classifica tifosi di questa giornata' : 'Classifica generale tifosi della società'" :aria-busy="isLoadingLeaderboard">
            <li v-for="row in leaderboardRows" :key="row.rank" class="row-item leaderboard-row" :class="{ 'leaderboard-self': row.is_me }" :aria-current="row.is_me ? 'true' : undefined">
              <span class="rank leaderboard-rank" :class="row.rank <= 3 ? `r${row.rank}` : ''">{{ row.rank }}</span>
              <FanAvatar :name="row.name" :framed="row.equipped_frame === 'club_frame'" :size="36" />
              <span class="row-copy"><b>{{ row.name }}</b><span v-if="row.is_me">Tu</span></span>
              <span class="row-pts">{{ row.points }} <small>pt</small></span>
            </li>
          </ol>
          <button v-if="!isRegistered" class="btn-vote" type="button" @click="openRegistration('leaderboard')">Entra in classifica con il tuo nickname</button>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'match-summary' }" role="dialog" aria-label="La tua partita" :aria-hidden="activeSheet !== 'match-summary'" :inert="activeSheet !== 'match-summary'">
        <div class="grabber"></div>
        <div class="sheet-head"><div><div class="sheet-title">La tua partita</div><div class="sheet-sub">{{ homeName }} · {{ opponentLabel || 'Esperienza ufficiale' }}</div></div><button class="close" type="button" aria-label="Chiudi riepilogo" @click="closeSheets">✕</button></div>
        <div class="sheet-body">
          <div class="match-stats"><div><b>{{ hasVoted ? '✓' : '—' }}</b><span>Voto MVP</span></div><div><b>{{ completedGames }}</b><span>Sfide completate</span></div><div><b>{{ progress.earned }}</b><span>Monete guadagnate</span></div></div>
          <div class="row-item leaderboard-row"><span class="avatar" aria-hidden="true">🏆</span><span class="row-copy"><b>{{ finalMvpLabel }}</b><span>{{ matchConcluded ? finalMvpSubtitle : 'Il risultato finale sarà disponibile quando la partita sarà conclusa.' }}</span></span></div>
          <div v-if="finalMvpPlayers.length" class="final-mvp-cards">
            <FanPlayerCard v-for="player in finalMvpPlayers" :key="player.id" :player="player" :club-logo-url="homeLogoUrl" />
          </div>
          <div v-if="progress.unlocked" class="row-item leaderboard-row journey-badge"><span class="avatar" aria-hidden="true">🏅</span><span class="row-copy"><b>{{ FAN_BADGE_NAME }}</b><span>Il tuo badge digitale della partita</span></span></div>
          <p class="sheet-sub">{{ leaderboardUser ? `La tua posizione nella società: #${leaderboardUser.rank}` : 'Grazie per aver partecipato alla serata.' }}</p>
          <button v-if="hasFeedbackSurvey && !hasSubmittedFeedback" class="btn-vote" type="button" @click="openInlineFeedback">Com’è andata? Lascia il tuo feedback</button>
          <p v-else-if="hasSubmittedFeedback" class="sheet-sub">Feedback ricevuto, grazie!</p>
        </div>
      </div>

      <!-- Saldo e punti restano visibili mentre il contenuto del profilo scorre. -->
      <div class="sheet profile-sheet" :class="{ open: activeSheet === 'profile' }" role="dialog" aria-label="Profilo" :aria-hidden="activeSheet !== 'profile'" :inert="activeSheet !== 'profile'">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Il tuo profilo</div>
            <div class="sheet-sub">{{ isRegistered ? 'Account tifoso' : 'Ospite · non registrato' }}</div>
          </div>
          <button class="close" type="button" aria-label="Chiudi profilo" @click="closeSheets">✕</button>
        </div>
        <div class="profile-economy">
          <div class="profile-balances" aria-label="Monete e punti partecipazione">
            <button type="button" class="profile-balance profile-balance-coins" :aria-label="`${totalCoins} monete disponibili: apri saldo e movimenti`" @click="openSheet('wallet')">
              <span class="profile-balance-label"><span class="coin" aria-hidden="true"></span>Monete<span class="balance-arrow" aria-hidden="true">↗</span></span>
              <strong>{{ totalCoins }}</strong>
              <small>Da spendere</small>
            </button>
            <button type="button" class="profile-balance profile-balance-points" :aria-label="`${hasLoadedCollection ? participationPoints : 'Caricamento'} punti partecipazione: apri la classifica`" @click="openLeaderboard">
              <span class="profile-balance-label"><span class="points-mark" aria-hidden="true">★</span>Punti<span class="balance-arrow" aria-hidden="true">↗</span></span>
              <strong>{{ hasLoadedCollection ? participationPoints : '—' }}</strong>
              <small>Partecipazione totale</small>
            </button>
          </div>
          <p class="profile-economy-hint">1 moneta guadagnata = 1 punto. I punti restano anche se spendi.</p>
        </div>
        <div ref="profileBodyEl" class="sheet-body profile-body">
          <a v-if="organizationSlug" class="merch-orders-link" :href="`/${encodeURIComponent(organizationSlug)}/new/orders`">I miei ordini e prenotazioni <span aria-hidden="true">→</span></a>
          <FanCollectionPanel v-if="activeSheet === 'profile'" :nickname="profileNickname" :club-name="homeName" :club-logo-url="homeLogoUrl" :sport="organizationSport" :registered="isRegistered" :collection="hasLoadedCollection ? collection : null" :loading="isLoadingWallet" :busy="shopBusy" :error="walletError" @equip="equipCollectionItem" @open-shop="openSheet('spend')" @inspect="inspectCollectible">
            <template #identity-action>
              <button class="profile-edit" type="button" :aria-expanded="isNicknameEditorOpen" aria-controls="profile-nickname-editor" @click="toggleNicknameEditor">{{ isNicknameEditorOpen ? 'Annulla' : 'Modifica nickname' }}</button>
            </template>
            <template #profile-actions>
              <div v-if="isNicknameEditorOpen" id="profile-nickname-editor" class="nick-editor">
                <label class="nick-field">Nickname
                  <input
                    v-model.trim="nicknameDraft"
                    type="text"
                    minlength="3"
                    maxlength="24"
                    placeholder="Inserisci nickname"
                    class="nick-input"
                    @keyup.enter="submitNickname"
                  />
                </label>
                <button
                  class="chip-mint nick-save"
                  type="button"
                  :disabled="isSavingNickname || !canSubmitNickname"
                  @click="submitNickname"
                >
                  {{ isSavingNickname ? 'Salvataggio…' : 'Aggiorna' }}
                </button>
              </div>
              <p v-if="nicknameError" class="nick-msg error" role="alert">{{ nicknameError }}</p>
              <p v-else-if="nicknameSuccess" class="nick-msg ok" role="status">{{ nicknameSuccess }}</p>
              <button v-if="!isRegistered" class="profile-signin" type="button" @click="openRegistration()">
                <span><strong>Accedi o crea il tuo profilo</strong><small>Conserva monete, punti e collezione</small></span>
                <span aria-hidden="true">→</span>
              </button>
            </template>
          </FanCollectionPanel>
          <button v-if="walletError" type="button" class="journey-next" :disabled="shopBusy" @click="loadWallet">Riprova a caricare la collezione</button>

          <section v-if="showLotteryTicket" :ref="bindLotteryTicket" class="lottery-box" aria-label="Biglietto lotteria MVP" tabindex="-1">
            <h3 class="lottery-title">QR Lotteria MVP</h3>
            <div class="lottery-code">{{ lotteryCode }}</div>
            <img v-if="lotteryQrUrl" :src="lotteryQrUrl" alt="QR lotteria" class="lottery-qr" />
            <p class="lottery-hint">Conserva questo biglietto per il ritiro di un eventuale premio.</p>
          </section>
        </div>
      </div>

      <!-- Overlay gioco a schermo pieno -->
      <div v-if="activeGameId" class="game-overlay">
        <header class="game-head">
          <button class="close" type="button" aria-label="Indietro" :disabled="isBasketballSaving || isPlaymakerSaving || isCrossoverSaving || isVolleyballSaving" @click="closeGame">←</button>
          <div class="game-title">{{ games[activeGameId]?.title }}</div>
          <button class="close" type="button" aria-label="Chiudi" :disabled="isBasketballSaving || isPlaymakerSaving || isCrossoverSaving || isVolleyballSaving" @click="closeGame">✕</button>
        </header>
        <div v-if="coinError" class="game-credit-error" role="alert"><p>{{ coinError }}</p><button v-if="pendingGameClaim" type="button" class="journey-next" :disabled="isSavingCoins" @click="onGameClaim(pendingGameClaim)">Riprova il salvataggio</button></div>
        <div class="game-stage">
          <ReactionTestGame
            v-if="activeGameId === 'reaction'"
            class="game-fill"
            :completion-reward="10"
            @claim="onGameClaim"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @exit="closeGame"
          />
          <QuickQuizGame
            v-else-if="activeGameId === 'quiz'"
            class="game-fill"
            :event-id="eventId"
            :completion-reward="QUIZ_REWARD"
            :claiming="isSavingCoins"
            @claim="onGameClaim"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @exit="closeGame"
          />
          <BasketballShootout
            v-else-if="activeGameId === 'basketball'"
            :key="eventId"
            class="game-fill"
            :event-id="eventId"
            :wallet-coins="totalCoins"
            :earned-coins="journey.rewards.basketball || 0"
            :wall-sponsors="wallSponsors"
            :club-name="homeName"
            :club-logo-url="homeLogoUrl"
            @credited="onBasketballCredit"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @saving-change="isBasketballSaving = $event"
          />
          <PlaymakerGame
            v-else-if="activeGameId === 'playmaker'"
            :key="eventId"
            class="game-fill"
            :event-id="eventId"
            :wallet-coins="totalCoins"
            :earned-coins="journey.rewards.playmaker || 0"
            :wall-sponsors="wallSponsors"
            :club-name="homeName"
            :club-logo-url="homeLogoUrl"
            :organization-slug="organizationSlug"
            @credited="onPlaymakerCredit"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @saving-change="isPlaymakerSaving = $event"
          />
          <CrossoverRushGame
            v-else-if="activeGameId === 'crossover'"
            :key="eventId"
            class="game-fill"
            :event-id="eventId"
            :wallet-coins="totalCoins"
            :earned-coins="journey.rewards.crossover || 0"
            :wall-sponsors="wallSponsors"
            :club-name="homeName"
            :club-logo-url="homeLogoUrl"
            :organization-slug="organizationSlug"
            @credited="onCrossoverCredit"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @saving-change="isCrossoverSaving = $event"
          />
          <VolleyballArena
            v-else-if="volleyballGameIds.includes(activeGameId)"
            :key="`${eventId}:${activeGameId}`"
            class="game-fill"
            :game-id="activeGameId"
            :event-id="eventId"
            :wallet-coins="totalCoins"
            :earned-coins="journey.rewards[activeGameId] || 0"
            :club-name="homeName"
            :club-logo-url="homeLogoUrl"
            @credited="onVolleyballCredit"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @saving-change="isVolleyballSaving = $event"
          />
          <component
            v-else-if="activeGameId === 'tap' || activeGameId === 'tap-volley'"
            :is="activeGameId === 'tap-volley' ? TapVolleyChallenge : TapChallenge"
            :key="`${eventId}:${activeGameId}`"
            class="game-fill"
            :event-id="eventId"
            :completion-reward="8"
            :claiming="isSavingCoins"
            @claim="onGameClaim"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @exit="closeGame"
          />
          <SponsorRushGame
            v-else-if="activeGameId === 'sponsor-rush'"
            class="game-fill"
            :completion-reward="12"
            :sponsor="sponsors[0] || null"
            :sponsor-analytics-match-id="eventId || 0"
            @claim="onGameClaim"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @exit="closeGame"
          />
          <MemoryFlashGame
            v-else-if="activeGameId === 'memory-flash'"
            class="game-fill"
            :wallet-coins="totalCoins"
            :claiming="isSavingCoins || Boolean(pendingGameClaim)"
            @claim="onGameClaim"
            @round-started="onReportGameStarted"
            @round-completed="onReportGameCompleted"
            @exit="closeGame"
          />
        </div>
      </div>

    </div>

    <FanMvpVoteFlight
      v-if="homeVoteFlight && homeVoteLanding"
      :player="votedPlayerCard"
      :club-logo-url="homeLogoUrl"
      :sponsors="postVoteSponsors"
      :show-auth-invite="!isRegistered"
      :source="homeVoteFlight.source"
      :destination="homeVoteLanding"
      @finished="finishHomeVoteFlight"
    />
    <div class="coin-flight-layer" aria-hidden="true"><span v-for="(flight, index) in coinFlights" :key="index" class="coin coin-flight" :style="flight"></span></div>
    <FanRegistrationPromptModal v-if="isRegistrationOpen" v-model="isRegistrationOpen" :trigger="registrationTrigger" :reward-label="registrationRewardLabel" :club-name="homeName" :organization-slug="organizationSlug" :theme-style="themeStyle" :wallet-coins="totalCoins" :on-submit="registerJourneyFan" :on-login="loginJourneyFan" @completed="completeRegistration" @dismissed="dismissRegistration" @telemetry="onFormTelemetry" />
    <FanMerchShop
      v-if="merchRequest"
      :key="organizationSlug"
      sheet
      :slug="organizationSlug"
      :club-name="homeName"
      :colors="organizationColors"
      :open-request="merchRequest"
    />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { normalizeOrganizationColors, organizationThemeStyle } from '../organizationTheme';
import { preloadFanHomeImages } from '../fanHomeAssets';
import { FAN_GAME_ARTWORK } from '../fanGameArtwork';
import { resolveGameWallSponsors } from '../games/gameWallSponsors';
import { findAssignedSponsor, postVoteSponsorSlots } from '../sponsorPlacements';
import { provideSponsorTracking, useSponsorPlacement } from '../useSponsorTracking.js';
import { sponsorAnalyticsFields } from '../sponsorAnalyticsService.js';
import SponsorBanner from '../components/SponsorBanner.vue';
import { createEventReportTelemetry, createReportSignalSender, reportSource } from '../eventReportTelemetry';
import { stopPostHogForNewExperience } from '../lib/posthog';
import { sanitizeFanAnalyticsMetadata } from '../fanAnalytics';
import { inlineFeedbackOptions, inlineFeedbackQuestions } from '../inlineFeedback';
import FanPlayerCard from '../components/FanPlayerCard.vue';
import MvpPlayerCarousel from '../components/MvpPlayerCarousel.vue';
import FanMvpRoster from '../components/FanMvpRoster.vue';
import FanMvpCelebration from '../components/FanMvpCelebration.vue';
import FanMvpVoteFlight from '../components/FanMvpVoteFlight.vue';
import FanCollectionPanel from '../components/FanCollectionPanel.vue';
import FanAvatar from '../components/FanAvatar.vue';
import FanMatchCard from '../components/FanMatchCard.vue';
import FanMerchLink from '../merch/FanMerchLink.vue';
const MatchChatEntry = defineAsyncComponent(() => import('../match-chat/MatchChatEntry.vue'));
const FanMerchShop = defineAsyncComponent(() => import('../merch/FanMerchShop.vue'));
const FanRegistrationPromptModal = defineAsyncComponent(() => import('../components/FanRegistrationPromptModal.vue'));
import { fanRegistrationAcceptance } from '../fanRegistration';
import { FIRST_FAN_GOAL, VOTE_REWARD, QUIZ_REWARD, FAN_BADGE_NAME, journeyStorageKey, normalizeJourney, journeyProgress, recordJourneyReward, newMinigameRoundId } from '../fanJourney';

// Giochi reali riusati da EarnCoinsModal (lazy-load: pesano).
const ReactionTestGame = defineAsyncComponent(() => import('../components/ReactionTestGame.vue'));
const QuickQuizGame = defineAsyncComponent(() => import('../components/QuickQuizGame.vue'));
const BasketballShootout = defineAsyncComponent(() => import('../components/minigames/BasketballShootout.vue'));
const PlaymakerGame = defineAsyncComponent(() => import('../components/minigames/PlaymakerGame.vue'));
const CrossoverRushGame = defineAsyncComponent(() => import('../components/minigames/CrossoverRushGame.vue'));
const VolleyballArena = defineAsyncComponent(() => import('../components/minigames/VolleyballArena.vue'));
const TapChallenge = defineAsyncComponent(() => import('../components/minigames/TapChallenge.vue'));
const TapVolleyChallenge = defineAsyncComponent(() => import('../components/minigames/TapVolleyChallenge.vue'));
const SponsorRushGame = defineAsyncComponent(() => import('../components/minigames/SponsorRushGame.vue'));
const MemoryFlashGame = defineAsyncComponent(() => import('../components/minigames/MemoryFlashGame.vue'));
import {
  apiClient,
  fetchFanProfile,
  fetchVoteStatus,
  saveEventFeedbackAnswer,
  fetchQuickQuiz,
  vote,
  fetchFanWallet,
  purchaseFanItem,
  equipFanItem,
  updateFanNickname,
  resolveApiUrl,
  getOrganizationSlug,
  getDeviceHeaders,
  getFanSessionHeaders,
  registerFanProfile,
  claimFanJourneyReward,
} from '../api';

const FAN_NICKNAME_MIN_LEN = 3;
const FAN_NICKNAME_MAX_LEN = 24;
const FAN_NICKNAME_PATTERN = /^[A-Za-z0-9._ -]+$/;

stopPostHogForNewExperience();

const props = defineProps({
  initialDataReady: { type: Boolean, default: true },
  organizationSlug: { type: String, default: '' },
  eventId: { type: Number, default: 0 },
  teamName: { type: String, default: 'TEAM' },
  teamLogoUrl: { type: String, default: '' },
  matchLabel: { type: String, default: 'Vota • Gioca • Vinci • Partecipa' },
  activeEvent: { type: Object, default: null },
  activeEventError: { type: Boolean, default: false },
  votedPlayerImageUrl: { type: String, default: '' },
  votedPlayerName: { type: String, default: '' },
  votedPlayerLastName: { type: String, default: '' },
  votedPlayerNumber: { type: [String, Number], default: '' },
  // Segnale dal parent dopo il primo voto (registrazione). Accettato per compat; TODO: prompt registrazione.
  registrationPromptSignal: { type: Number, default: 0 },
});
const sponsorMeasurement = provideSponsorTracking({
  url: resolveApiUrl,
  scope: () => ({ slug: props.organizationSlug, match: props.eventId || 0 }),
});
const vSponsorPlacement = useSponsorPlacement(sponsorMeasurement);

const WALLET_STORAGE_KEY = 'wallet:coins';
const emit = defineEmits(['loading-change', 'brand-change', 'retry-active-event']);
const isHomeLoading = ref(true);
let bootstrapRevision = 0;
let assetController;
let organizationThemeRequest = Promise.resolve();
watch(isHomeLoading, (loading) => emit('loading-change', loading), { immediate: true, flush: 'sync' });

const organizationColors = ref(normalizeOrganizationColors());
const organizationName = ref('');
const organizationLogoUrl = ref('');
const organizationSport = ref('');
const gameWallAssignments = ref({});
const postVoteAssignments = ref({});
const homeLogoUrl = computed(() => props.organizationSlug ? organizationLogoUrl.value : props.teamLogoUrl);
const themeStyle = computed(() => organizationThemeStyle(organizationColors.value));
watch(() => props.organizationSlug, (slug, previous, onCleanup) => {
  organizationColors.value = normalizeOrganizationColors();
  organizationName.value = '';
  organizationLogoUrl.value = '';
  organizationSport.value = '';
  gameWallAssignments.value = {};
  postVoteAssignments.value = {};
  if (!slug) {
    organizationThemeRequest = Promise.resolve();
    return;
  }
  const controller = new AbortController();
  onCleanup(() => controller.abort());
  organizationThemeRequest = loadOrganizationBrand(slug, controller.signal);
}, { immediate: true });
async function loadOrganizationBrand(slug, signal) {
  try {
    const { data } = await apiClient.get('/organization-theme', {
      params: { organization_slug: slug },
      signal,
    });
    if (!signal.aborted && slug === props.organizationSlug) {
      organizationColors.value = normalizeOrganizationColors(data);
      organizationName.value = String(data?.name || '').trim();
      organizationLogoUrl.value = String(data?.logo_url || '').trim();
      organizationSport.value = String(data?.sport || '').trim();
      gameWallAssignments.value = data?.game_wall_sponsors || {};
      postVoteAssignments.value = data?.post_vote_sponsors || {};
    }
  } catch (error) {
    if (!signal.aborted) console.error('Impossibile caricare i dati della società', error);
  }
}
watch([organizationColors, organizationName, homeLogoUrl], () => emit('brand-change', {
  colors: organizationColors.value,
  name: organizationName.value,
  logoUrl: homeLogoUrl.value,
}), { immediate: true });

/* ---------- Stato ---------- */
const totalCoins = ref(0);
const isRegistered = ref(false);
const fanNickname = ref('');
const fanLotteryTicket = ref(null);
const nicknameDraft = ref('');
const isNicknameEditorOpen = ref(false);
const isSavingNickname = ref(false);
const nicknameError = ref('');
const nicknameSuccess = ref('');
const hasVoted = ref(false);
const votedPlayerId = ref(null);
const localVoted = ref(null); // giocatore votato in questa sessione (feedback immediato)
const leaderboardRows = ref([]);
const leaderboardPeriod = ref('event');
const leaderboardUser = ref(null);
const liveEvent = ref(null);
const isRegistrationOpen = ref(false);
const registrationTrigger = ref('profile_overlay');
const registrationRewardLabel = ref('');
const registrationDestination = ref('profile');
const registrationOriginSheet = ref('');
const isLoadingLeaderboard = ref(false);
const leaderboardError = ref('');
const sponsorCatalog = ref([]);
const sponsorCatalogSlug = ref(null);
const currentSponsorCatalog = computed(() => sponsorCatalogSlug.value === props.organizationSlug ? sponsorCatalog.value : []);
const sponsors = computed(() => currentSponsorCatalog.value.filter((sponsor) => sponsor.isActive));

const rawPlayers = ref([]);
const isLoadingPlayers = ref(false);
const playersError = ref('');
const isLoadingVoteStatus = ref(false);
const voteStatusError = ref('');
const liveStatusError = ref(false);
let playersRequestRevision = 0;
let voteStatusRequestRevision = 0;
const homeMvpTitle = ref(null);
const homeMvpCarousel = ref(null);
const homeVoteLanding = ref(null);
const homeVoteFlight = ref(null);
let homeVoteBonusRequest = null;
const homeMvpSelectedId = ref(null);
const missionMvpCandidate = ref(null);
const mvpEditing = ref(false);
const highlightPlay = ref(false);
const homeVoteReward = ref(0);
const isVoting = ref(false);
const pendingVotePlayerId = ref(null);
const voteRevealRevision = ref(0);
const isVoteRevealRunning = ref(false);
let voteRequestRevision = 0;
let voteRevealTimer;
let pendingVoteCoinAnimation = false;
let sheetRevision = 0;
const voteError = ref('');
const coinError = ref('');
const isSavingCoins = ref(false);
const isBasketballSaving = ref(false);
const isPlaymakerSaving = ref(false);
const isCrossoverSaving = ref(false);
const isVolleyballSaving = ref(false);
const pendingGameClaim = ref(null);
const activeGameRoundId = ref('');
const voteBonusCoins = ref(0);
const celebrationKind = ref('vote');
const showVoteProfilePrompt = computed(() => celebrationKind.value === 'vote' && !isRegistered.value);
const celebrationName = ref('');
const isWalletReady = ref(false);
const journey = ref(normalizeJourney());
const progress = computed(() => journeyProgress(journey.value));
const quizAvailable = ref(false);
const walletEl = ref(null);
const voteResultEl = ref(null);
const voteBonusEl = ref(null);
const walletPulse = ref(false);
const coinFlights = ref([]);
let coinAnimationTimer;
let coinAnimationRevision = 0;
let journeyRewardRevision = 0;
let walletRevision = 0;
let loadedEventId = null;
let liveStatusTimer;
let liveRequestEventId = null;

const doneKeys = ref([]); // missioni completate (persistite per evento)

const activeSheet = ref('');
const platformInfoTriggerRef = ref(null);
const platformInfoRef = ref(null);
const profileBodyEl = ref(null);
const lotteryTicketEl = ref(null);
const activeGameId = ref(null);
const hasSubmittedFeedback = ref(false);
const feedbackDraft = ref({ responseId: '', answers: {} });
const feedbackPosition = ref(null);
const feedbackPendingAnswer = ref(null);
const feedbackSuggestion = ref('');
const feedbackError = ref('');
const feedbackQuestionTitle = ref(null);
const feedbackThanks = ref(false);
const homeChallengeButton = ref(null);
const homePlayButton = ref(null);
let feedbackThanksTimer;

// Registry giochi reali (titolo + reward indicativa mostrata nella lista).
const volleyballGameIds = ['volley-rush', 'setter-pro', 'monster-block'];
const games = {
  reaction: { title: 'Reaction Test', reward: 10 },
  quiz: { title: 'Quiz Lampo', reward: 15 },
  basketball: { title: 'Basket Rush', reward: 1 },
  playmaker: { title: 'Playmaker', reward: 1 },
  crossover: { title: 'Crossover Rush', reward: 1 },
  'volley-rush': { title: 'Volley Rush', reward: 1 },
  'setter-pro': { title: 'Setter Pro', reward: 1 },
  'monster-block': { title: 'Monster Block', reward: 1 },
  tap: { title: 'Tap Challenge', reward: 8 },
  'tap-volley': { title: 'Tap Volley Challenge', reward: 8 },
  'sponsor-rush': { title: 'Sponsor Rush', reward: 12 },
  'memory-flash': { title: 'Memory Flash', reward: 8 },
};
const redeeming = ref('');
const equipping = ref(false);
const isLoadingWallet = ref(false);
const walletError = ref('');
const shopMessage = ref('');
const catalog = ref([]);
const walletMovements = ref([]);
const participationPoints = ref(0);
const emptyCollection = () => ({ equipped_frame: '', items: [], badges: [], matches_followed: 0 });
const collection = ref(emptyCollection());
const hasLoadedCollection = ref(false);
const equippedFrame = computed(() => {
  if (!hasLoadedCollection.value) return leaderboardUser.value?.equipped_frame === 'club_frame' ? 'club_frame' : '';
  return collection.value.equipped_frame === 'club_frame' && collection.value.items.some((item) => item.key === 'club_frame' && item.kind === 'frame') ? 'club_frame' : '';
});
const selectedCollectible = ref(null);
const isCollectiblePreview = ref(false);
const isCollectibleNew = ref(false);
const collectibleOrigin = ref('spend');
const collectibleRevealRevision = ref(0);
const shopMatchMemory = computed(() => ({
  key: 'match_card', kind: 'match_card', event_id: props.eventId,
  club_name: homeName.value, opponent: String(eventState.value?.team2_name || '').trim(),
  event_date: eventState.value?.start_datetime || ((!props.activeEvent?.id || props.activeEvent.id === props.eventId) ? props.activeEvent?.start_datetime : '') || '',
}));
const shopBusy = computed(() => Boolean(redeeming.value) || equipping.value || isSavingCoins.value || isBasketballSaving.value || isPlaymakerSaving.value || isCrossoverSaving.value || isVolleyballSaving.value);
let walletRequestRevision = 0;
let leaderboardRequestRevision = 0;
let isDisposed = false;
let coinsStream = null;
const eventState = computed(() => {
  if (Number(liveEvent.value?.event_id) === props.eventId) return liveEvent.value;
  const activeEvent = props.activeEvent;
  return !activeEvent?.id || activeEvent.id === props.eventId ? activeEvent : null;
});
const matchChatEventLabel = computed(() => {
  const event = eventState.value;
  const match = event?.team2_name
    ? `${event.team1_name || homeName.value} – ${event.team2_name}`
    : `Partita #${props.eventId}`;
  const start = new Date(event?.start_datetime || '');
  if (Number.isNaN(start.getTime())) return match;
  const date = start.toLocaleString('it-IT', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
  return `${match} · ${date}`;
});
const availableMinigames = computed(() => liveEvent.value?.available_minigames ?? props.activeEvent?.available_minigames);
function isGameAvailable(id) {
  return !Array.isArray(availableMinigames.value) || availableMinigames.value.includes(id);
}
const matchConcluded = computed(() => eventState.value?.is_concluded === true);
const showMatchSummary = computed(() => Boolean(props.eventId) && (matchConcluded.value || eventState.value?.votes_closed === true));
const canVote = computed(() => Boolean(props.eventId) && Boolean(eventState.value) && eventState.value?.is_active !== false && !eventState.value?.votes_closed && !matchConcluded.value);
const showHomeVoteReceipt = computed(() => hasVoted.value && (!mvpEditing.value || !canVote.value));
const isScheduledMatch = computed(() => {
  const startsAt = eventState.value?.start_datetime
    || (Number(props.activeEvent?.id) === props.eventId ? props.activeEvent?.start_datetime : '');
  return !canVote.value && !showMatchSummary.value && Date.parse(startsAt) > Date.now();
});
const mvpConnectionError = computed(() => props.eventId ? liveStatusError.value : props.activeEventError);
const mvpStatusLabel = computed(() => {
  if (canVote.value) return 'Votazioni aperte';
  if (showMatchSummary.value) return 'Votazioni chiuse';
  if (isScheduledMatch.value) return 'Partita programmata';
  if (mvpConnectionError.value) return 'Stato non disponibile';
  return props.eventId ? 'Votazioni non aperte' : 'Nessuna partita';
});
const mvpUnavailableMessage = computed(() => {
  if (showMatchSummary.value) return 'Le votazioni per questa partita sono terminate.';
  if (mvpConnectionError.value) return 'Impossibile verificare lo stato delle votazioni.';
  if (!props.eventId) return 'Al momento non ci sono partite disponibili.';
  return 'Il club deve ancora aprire le votazioni.';
});
watch(canVote, (open) => {
  if (!open) cancelHomeVoteEdit();
});
const completedGames = computed(() => Object.keys(journey.value.rewards).filter((key) => key !== 'vote').length);
const featuredMission = computed(() => {
  if (!props.eventId || matchConcluded.value || eventState.value?.is_active === false) return null;
  const missions = earnMissions.value.filter((mission) => mission.game && (mission.key !== 'quiz' || quizAvailable.value));
  const unplayed = missions.filter((mission) => !doneKeys.value.includes(mission.key));
  return unplayed.find((mission) => mission.key === 'quiz') || unplayed[0] || missions[0] || null;
});
const nextQuizLabel = computed(() => {
  const startsAt = Date.parse(liveEvent.value?.quiz?.starts_at || '');
  if (!Number.isFinite(startsAt) || startsAt <= Date.now()) return '';
  return `Prossimo quiz alle ${new Date(startsAt).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })}`;
});
const finalMvpLabel = computed(() => {
  const winners = liveEvent.value?.mvp || [];
  return matchConcluded.value && winners.length ? winners.map((player) => `${player.first_name} ${player.last_name}`.trim()).join(' · ') : 'MVP del pubblico';
});
const finalMvpSubtitle = computed(() => {
  const winners = liveEvent.value?.mvp || [];
  return winners.length ? `${winners.length > 1 ? 'MVP a pari merito' : 'MVP della partita'} · ${winners[0].votes} voti` : 'Non sono stati espressi voti per questa partita.';
});
const journeyNextLabel = computed(() => {
  if (showMatchSummary.value) return 'La tua partita →';
  if (progress.value.unlocked) return 'Scopri i premi →';
  if (!hasVoted.value && canVote.value) return `Vota l’MVP · +${VOTE_REWARD}`;
  if (isGameAvailable('quiz') && quizAvailable.value && !doneKeys.value.includes('quiz')) return `Completa il quiz · +${QUIZ_REWARD}`;
  return 'Scegli una sfida →';
});
function openJourneyNext() {
  if (showMatchSummary.value) openSheet('match-summary');
  else if (progress.value.unlocked) openSheet('spend');
  else if (!hasVoted.value && canVote.value) openSheet('mvp');
  else if (isGameAvailable('quiz') && quizAvailable.value && !doneKeys.value.includes('quiz')) openGame('quiz');
  else openSheet('earn');
}

async function loadQuizAvailability() {
  const eventId = props.eventId;
  quizAvailable.value = false;
  if (!eventId) return;
  const { ok, data } = await fetchQuickQuiz(eventId);
  if (eventId === props.eventId) quizAvailable.value = ok && Array.isArray(data?.questions) && data.questions.length > 0;
}

async function refreshLiveStatus() {
  const eventId = props.eventId;
  if (!eventId || liveRequestEventId === eventId || (typeof document !== 'undefined' && document.visibilityState === 'hidden')) return;
  const scope = walletScope();
  liveRequestEventId = eventId;
  try {
    const { data } = await apiClient.get(`/events/${eventId}/fan-live`);
    if (isDisposed || scope !== walletScope() || eventId !== props.eventId || Number(data?.event_id) !== eventId) return;
    liveEvent.value = data;
    liveStatusError.value = false;
    quizAvailable.value = data.quiz?.available === true;
  } catch (error) {
    if (!isDisposed && scope === walletScope() && !eventState.value) liveStatusError.value = true;
  }
  finally { if (liveRequestEventId === eventId) liveRequestEventId = null; }
}

function openRegistration(trigger = 'profile_overlay', rewardLabel = '') {
  if (shopBusy.value && !(trigger === 'after_vote' && homeVoteBonusRequest)) return;
  if (!isRegistrationOpen.value && !isRegistered.value) reportTelemetry?.record('registration_started');
  registrationTrigger.value = trigger;
  registrationRewardLabel.value = rewardLabel;
  registrationOriginSheet.value = activeSheet.value;
  registrationDestination.value = ({ wallet: 'wallet', spend_redeem: 'spend', after_vote: 'leaderboard', leaderboard: 'leaderboard' })[trigger] || 'profile';
  closeSheets();
  isRegistrationOpen.value = true;
}
function completeRegistration() {
  if (!isRegistered.value) return;
  isRegistrationOpen.value = false;
  if (registrationDestination.value === 'profile') openProfile();
  else if (registrationDestination.value === 'leaderboard') openLeaderboard();
  else openSheet(registrationDestination.value);
}
function dismissRegistration() {
  isRegistrationOpen.value = false;
  if (['spend', 'collectible', 'vote-result', 'leaderboard'].includes(registrationOriginSheet.value)) openSheet(registrationOriginSheet.value);
}
async function registerJourneyFan(form) {
  const scope = walletScope();
  const registrationStartedAt = Date.now();
  const trigger = registrationTrigger.value;
  // The prompt can open while the guest's bonus is still being saved.
  if (homeVoteBonusRequest) await homeVoteBonusRequest;
  if (isDisposed || scope !== walletScope()) return { ok: false };
  const response = await registerFanProfile({ event_id: props.eventId, nickname: form.nickname, phone: form.phone, ...fanRegistrationAcceptance(form), guest_coins: totalCoins.value });
  if (isDisposed || scope !== walletScope()) return { ok: false };
  if (!response.ok) return response;
  recordFanEvent('registration_completed', { trigger, duration_ms: Math.max(0, Date.now() - registrationStartedAt) });
  isRegistered.value = true;
  fanNickname.value = String(response.data?.user?.nickname || form.nickname || '').trim();
  setCoins(Number(response.data?.wallet ?? totalCoins.value));
  await loadFanProfile();
  await refreshLeaderboard();
  return { ok: true, wallet: totalCoins.value };
}
async function loginJourneyFan() {
  const scope = walletScope();
  if (homeVoteBonusRequest) await homeVoteBonusRequest;
  if (isDisposed || scope !== walletScope()) return { ok: false };
  const profile = await loadFanProfile();
  if (isDisposed || scope !== walletScope()) return { ok: false };
  if (profile?.registration_pending || profile?.user?.registration_pending) {
    isRegistered.value = false;
    return { ok: false, code: 'REGISTRATION_INCOMPLETE' };
  }
  if (!isRegistered.value) return { ok: false, message: 'Impossibile caricare il profilo. Riprova.' };
  await refreshLeaderboard();
  return { ok: true, wallet: totalCoins.value };
}

function saveJourney() {
  try { window.localStorage.setItem(journeyStorageKey(props.organizationSlug, props.eventId), JSON.stringify(journey.value)); } catch (error) { /* storage non disponibile */ }
}
function loadJourney() {
  try { journey.value = normalizeJourney(JSON.parse(window.localStorage.getItem(journeyStorageKey(props.organizationSlug, props.eventId)) || 'null')); }
  catch (error) { journey.value = normalizeJourney(); }
}
function prefersReducedMotion() {
  return Boolean(window.matchMedia?.('(prefers-reduced-motion: reduce)').matches);
}
function clearEarnedCoinAnimation() {
  coinAnimationRevision += 1;
  window.clearTimeout(coinAnimationTimer);
  walletPulse.value = false;
  coinFlights.value = [];
}
function cancelVotePresentation() {
  window.clearTimeout(voteRevealTimer);
  homeVoteFlight.value = null;
  isVoteRevealRunning.value = false;
  pendingVoteCoinAnimation = false;
  clearEarnedCoinAnimation();
}
function startVotePresentation() {
  cancelVotePresentation();
  voteRevealRevision.value += 1;
  if (prefersReducedMotion()) return;
  isVoteRevealRunning.value = true;
  // Sequence the decorative flight after the card; never delay the confirmed balance.
  voteRevealTimer = window.setTimeout(() => {
    isVoteRevealRunning.value = false;
    if (pendingVoteCoinAnimation && activeSheet.value === 'vote-result' && celebrationKind.value === 'vote') animateEarnedCoins();
    pendingVoteCoinAnimation = false;
  }, 760);
}
function animateVoteReward() {
  if (activeSheet.value !== 'vote-result' || celebrationKind.value !== 'vote') return;
  if (isVoteRevealRunning.value) pendingVoteCoinAnimation = true;
  else animateEarnedCoins();
}
async function animateEarnedCoins() {
  if (isDisposed || prefersReducedMotion() || activeSheet.value !== 'vote-result') return;
  const revision = ++coinAnimationRevision;
  walletPulse.value = true;
  await nextTick();
  if (isDisposed || revision !== coinAnimationRevision || activeSheet.value !== 'vote-result') return;
  if (activeSheet.value === 'vote-result' && walletEl.value && voteResultEl.value) {
    const from = (voteBonusEl.value || voteResultEl.value).getBoundingClientRect();
    const to = walletEl.value.getBoundingClientRect();
    const startY = voteBonusEl.value ? from.top + from.height / 2 : from.top + 60;
    coinFlights.value = Array.from({ length: 6 }, (_, index) => ({
      left: `${from.left + from.width / 2}px`, top: `${startY}px`,
      '--flight-x': `${to.left + to.width / 2 - from.left - from.width / 2}px`,
      '--flight-y': `${to.top + to.height / 2 - startY}px`,
      '--flight-bend': `${(index % 2 ? 1 : -1) * (25 + index * 8)}px`,
      '--flight-lift': `${-38 - index * 9}px`,
      '--flight-spin': `${(index % 2 ? 1 : -1) * (160 + index * 35)}deg`,
      animationDelay: `${index * 50}ms`,
    }));
  }
  window.clearTimeout(coinAnimationTimer);
  coinAnimationTimer = window.setTimeout(() => { walletPulse.value = false; coinFlights.value = []; }, 1100);
}

/* ---------- Derivati header ---------- */
function initials(name) {
  const parts = String(name || '').trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return '—';
  return parts.slice(0, 3).map((p) => p[0]).join('').toUpperCase();
}
const homeName = computed(() =>
  String(organizationName.value || props.activeEvent?.organization_name || '').trim() || 'Società',
);
const merchRequest = ref(null);
function openMerchShop(trigger) {
  sheetRevision += 1;
  merchRequest.value = { trigger };
}
watch(() => props.organizationSlug, () => { merchRequest.value = null; });
const homeBadge = computed(() => initials(homeName.value));
const opponentLabel = computed(() => {
  const opponent = String(eventState.value?.team2_name || '').trim();
  return opponent ? `vs ${opponent}` : '';
});

/* ---------- Feedback dell'evento ---------- */
const feedbackSurvey = computed(() => {
  const survey = eventState.value?.feedback_survey ?? eventState.value?.feedbackSurvey ?? {};
  return {
    ...survey,
    suggestionPrompt: String(survey.suggestion_prompt ?? survey.suggestionPrompt ?? '').trim()
      || 'Se potessi migliorare qualcosa, cosa ti piacerebbe aggiungere o cambiare?',
  };
});
const feedbackQuestions = computed(() => inlineFeedbackQuestions(feedbackSurvey.value));
const hasFeedbackSurvey = computed(() =>
  Boolean(props.eventId)
  && eventState.value?.show_feedback_survey !== false
  && eventState.value?.showFeedbackSurvey !== false
  && feedbackQuestions.value.length > 0,
);
const feedbackStep = computed(() => {
  const unanswered = feedbackQuestions.value.findIndex(({ id }) => !Object.hasOwn(feedbackDraft.value.answers, id));
  const last = feedbackQuestions.value.length - 1;
  return Math.min(feedbackPosition.value ?? (unanswered < 0 ? last : unanswered), last);
});
const activeFeedbackQuestion = computed(() => feedbackQuestions.value[feedbackStep.value]);
const feedbackOptions = computed(() => inlineFeedbackOptions(activeFeedbackQuestion.value));
const feedbackSaving = computed(() => feedbackPendingAnswer.value !== null);
const feedbackStorageKey = (eventId) => `wcmvpvs-feedback:${eventId}`;
watch([() => props.eventId, () => props.organizationSlug], ([eventId]) => {
  hasSubmittedFeedback.value = false;
  feedbackDraft.value = { responseId: '', answers: {} };
  feedbackPosition.value = null;
  feedbackPendingAnswer.value = null;
  feedbackSuggestion.value = '';
  feedbackError.value = '';
  feedbackThanks.value = false;
  if (typeof window === 'undefined') return;
  window.clearTimeout(feedbackThanksTimer);
  if (!eventId) return;
  try {
    const saved = window.localStorage.getItem(feedbackStorageKey(eventId));
    if (saved === '1') {
      hasSubmittedFeedback.value = true;
    } else if (saved) {
      const draft = JSON.parse(saved);
      if (/^[0-9a-f]{32}$/.test(draft?.responseId) && draft.organizationSlug === props.organizationSlug
        && draft.answers && typeof draft.answers === 'object' && !Array.isArray(draft.answers)) {
        feedbackDraft.value = draft;
      }
    }
  } catch (error) {
    /* il flusso resta utilizzabile quando localStorage non è disponibile */
  }
}, { immediate: true });

function persistFeedback(value) {
  try {
    window.localStorage.setItem(feedbackStorageKey(props.eventId), value);
  } catch (error) {
    /* lo stato resta disponibile per la sessione corrente */
  }
}
function openInlineFeedback() {
  if (!hasFeedbackSurvey.value || hasSubmittedFeedback.value) return;
  closeSheets();
  nextTick(() => feedbackQuestionTitle.value?.focus());
}
function previousFeedbackQuestion() {
  if (feedbackSaving.value || feedbackStep.value <= 0) return;
  recordFanEvent('feedback_step', { trigger: 'feedback', count: feedbackStep.value + 1, outcome: 'back' });
  feedbackPosition.value = feedbackStep.value - 1;
  feedbackError.value = '';
  nextTick(() => feedbackQuestionTitle.value?.focus({ preventScroll: true }));
}
async function answerFeedbackInline(value) {
  const question = activeFeedbackQuestion.value;
  if (feedbackSaving.value || !hasFeedbackSurvey.value || hasSubmittedFeedback.value || !question) return;
  const optional = question.id === 'suggestion';
  if (!optional && !question.answers.some((answer) => answer.value === value)) return;
  const answer = value.trim();
  const draft = feedbackDraft.value;
  const step = feedbackStep.value;
  const scope = walletScope();
  if (!draft.responseId) {
    const random = globalThis.crypto.getRandomValues(new Uint8Array(16));
    draft.responseId = Array.from(random, (byte) => byte.toString(16).padStart(2, '0')).join('');
    draft.organizationSlug = props.organizationSlug;
    persistFeedback(JSON.stringify(draft));
    recordFanEvent('cta_clicked', { cta: 'feedback', placement: 'feedback_home' });
    recordFanEvent('feedback_opened', { trigger: 'feedback' });
  }
  feedbackError.value = '';
  feedbackPendingAnswer.value = answer;
  let result;
  try {
    result = await saveEventFeedbackAnswer(props.eventId, draft.responseId, question.id, answer);
  } catch (error) {
    result = { ok: false };
  }
  if (isDisposed || scope !== walletScope() || draft !== feedbackDraft.value) return;
  feedbackPendingAnswer.value = null;
  if (!result?.ok) {
    feedbackError.value = 'Impossibile salvare la risposta. Riprova.';
    recordFanEvent('feedback_failed', { trigger: 'feedback', count: step + 1, reason: 'save_failed', outcome: 'failure' });
    return;
  }
  recordFanEvent('feedback_step', { trigger: 'feedback', count: step + 1, outcome: optional && !answer ? 'skipped' : 'selected' });
  if (optional) {
    handleFeedbackSubmitted(draft.answers);
    nextTick(() => {
      if (isDisposed || scope !== walletScope() || draft !== feedbackDraft.value || activeSheet.value || isRegistrationOpen.value || activeGameId.value) return;
      (homeChallengeButton.value || homePlayButton.value)?.focus({ preventScroll: true });
    });
    return;
  }
  draft.answers[question.id] = answer;
  persistFeedback(JSON.stringify(draft));
  feedbackPosition.value = step + 1;
  nextTick(() => {
    if (!activeSheet.value && !isRegistrationOpen.value && !activeGameId.value && draft === feedbackDraft.value) {
      feedbackQuestionTitle.value?.focus({ preventScroll: true });
    }
  });
}
function handleFeedbackSubmitted(answers = {}) {
  reportTelemetry?.record('feedback_completed', { experience: answers.report_experience, perks_interest: answers.perks_interest });
  hasSubmittedFeedback.value = true;
  feedbackThanks.value = true;
  window.clearTimeout(feedbackThanksTimer);
  feedbackThanksTimer = window.setTimeout(() => { feedbackThanks.value = false; }, 2600);
  persistFeedback('1');
}

/* ---------- Giocatore votato (locale o da prop parent) ---------- */
const votedPlayerCard = computed(() => {
  const roster = players.value.find((player) => player.id === votedPlayerId.value);
  const local = localVoted.value;
  const name = local?.name || roster?.name || props.votedPlayerName || props.votedPlayerLastName || 'Il tuo MVP';
  const fullName = local?.fullName || roster?.fullName || props.votedPlayerName || name;
  const number = local?.number ?? roster?.number ?? props.votedPlayerNumber;
  return {
    id: votedPlayerId.value,
    name, fullName, number,
    lastName: local?.lastName || roster?.lastName || props.votedPlayerLastName,
    image: local?.image || roster?.image || props.votedPlayerImageUrl,
    subtitle: local?.subtitle || roster?.subtitle || (number !== '' && number != null ? `Maglia #${number}` : 'In campo'),
    badge: local?.badge || roster?.badge || initials(fullName),
  };
});
const displayVotedImage = computed(() => votedPlayerCard.value.image);
const displayVotedName = computed(() => votedPlayerCard.value.name);

/* ---------- Profilo (come LiveExperienceHome) ---------- */
const profileNickname = computed(() => {
  if (fanNickname.value.trim()) return fanNickname.value.trim();
  return isRegistered.value ? 'Tifoso' : 'Ospite';
});
const canSubmitNickname = computed(
  () => !isSavingNickname.value && nicknameDraft.value.trim() !== fanNickname.value.trim(),
);
const lotteryCode = computed(() => String(fanLotteryTicket.value?.ticket_code || '').trim());
const showLotteryTicket = computed(() => eventState.value?.show_lottery_ticket === true && Boolean(lotteryCode.value));
const showLotteryShortcut = computed(() => isRegistered.value && showLotteryTicket.value);
const lotteryQrUrl = computed(() => {
  if (!showLotteryTicket.value) return '';
  const qr = String(fanLotteryTicket.value?.qr_data || '').trim();
  return qr ? `https://api.qrserver.com/v1/create-qr-code/?size=260x260&data=${encodeURIComponent(qr)}` : '';
});

/* ---------- Giocatori (lista MVP reale) ---------- */
const players = computed(() => {
  const list = Array.isArray(rawPlayers.value) ? rawPlayers.value : [];
  const calledUp = list.filter((p) => p?.is_called_up === true);
  const source = calledUp.length ? calledUp : list;
  return source.map((p) => {
    const first = String(p?.first_name || '').trim();
    const last = String(p?.last_name || '').trim();
    const name = last ? `${first ? `${first[0]}. ` : ''}${last}` : first || 'Giocatore';
    const role = String(p?.role || p?.position || '').trim();
    const number = p?.jersey_number == null ? '' : String(p.jersey_number);
    return {
      id: Number(p?.id),
      name,
      fullName: [first, last].filter(Boolean).join(' ') || 'Giocatore',
      lastName: last,
      role,
      subtitle: role || (number ? `Maglia #${number}` : 'In campo'),
      number,
      badge: initials(`${first} ${last}`),
      image: String(p?.image_url || '').trim(),
    };
  });
});
const homeMvpPlayers = computed(() => players.value.map((player) => ({
  ...player,
  name: player.fullName,
  subtitle: player.role,
})));
const homeMvpSelectedPlayer = computed(() => homeMvpPlayers.value.find((player) => player.id === homeMvpSelectedId.value));
const canConfirmHomeVote = computed(() =>
  canVote.value && !isLoadingPlayers.value && !playersError.value && !isLoadingVoteStatus.value && !voteStatusError.value
  && !isVoting.value && !shopBusy.value && (!hasVoted.value || mvpEditing.value)
  && Boolean(homeMvpSelectedPlayer.value) && homeMvpSelectedId.value !== votedPlayerId.value,
);
function selectHomeMvp(player) {
  if (!canVote.value || isVoting.value || shopBusy.value || showHomeVoteReceipt.value) return;
  if (!homeMvpSelectedPlayer.value) {
    recordFanEvent('cta_clicked', { cta: 'mvp', placement: 'home_grid' });
    recordFanEvent('vote_opened', { is_change: hasVoted.value });
  }
  homeMvpSelectedId.value = player.id;
  voteError.value = '';
}
function confirmHomeVote(playerId) {
  if (canConfirmHomeVote.value && homeMvpSelectedId.value === playerId) {
    return castVote(homeMvpSelectedPlayer.value, { inline: true });
  }
}
function finishHomeVoteFlight() {
  homeVoteFlight.value = null;
  const scope = walletScope();
  const revision = sheetRevision;
  nextTick(() => {
    if (isDisposed || scope !== walletScope() || revision !== sheetRevision || activeSheet.value
      || isHomeLoading.value || isRegistrationOpen.value || activeGameId.value) return;
    homeVoteLanding.value?.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'instant' });
    homeMvpTitle.value?.focus({ preventScroll: true });
    if (!isRegistered.value) openRegistration('after_vote');
  });
}
function editHomeVote() {
  if (!hasVoted.value || !canVote.value || isVoting.value || shopBusy.value) return;
  mvpEditing.value = true;
  highlightPlay.value = false;
  voteError.value = '';
  nextTick(() => homeMvpTitle.value?.focus({ preventScroll: true }));
}
function cancelHomeVoteEdit() {
  const wasEditing = mvpEditing.value;
  mvpEditing.value = false;
  homeMvpSelectedId.value = null;
  if (wasEditing && !activeSheet.value) nextTick(() => homeMvpTitle.value?.focus({ preventScroll: true }));
}
function measureMvpCollapse(element) {
  element.inert = true;
  element.style.height = `${element.offsetHeight}px`;
}

const finalMvpPlayers = computed(() => matchConcluded.value ? (liveEvent.value?.mvp || []).map((winner) => {
  const roster = players.value.find((player) => player.id === Number(winner.player_id));
  const first = String(winner.first_name || '').trim();
  const last = String(winner.last_name || '').trim();
  const fullName = [first, last].filter(Boolean).join(' ') || roster?.fullName || 'Giocatore';
  return {
    id: Number(winner.player_id),
    name: roster?.name || (last ? `${first ? `${first[0]}. ` : ''}${last}` : fullName),
    fullName,
    number: roster?.number ?? '',
    image: winner.image_url || roster?.image || '',
    subtitle: `${Number(winner.votes) || 0} voti`,
    badge: roster?.badge || initials(fullName),
  };
}) : []);

/* ---------- Missioni "Guadagna" (aprono i giochi reali) ---------- */
const earnMissions = computed(() => [
  { key: 'vote', emoji: '🏆', title: "Vota l'MVP", subtitle: 'Un voto per partita', pts: 10, opensMvp: true },
  { key: 'reaction', emoji: '⚡', title: 'Reaction Test', subtitle: 'Testa i riflessi', pts: 10, game: 'reaction', repeatable: true },
  { key: 'quiz', emoji: '🧠', title: 'Quiz Lampo', subtitle: 'Domande sul match', pts: 15, game: 'quiz', repeatable: true },
  { key: 'volley-rush', emoji: '🏐', title: 'Volley Rush', subtitle: 'Supera il muro, trova il campo · Schiaccia e segna', pts: 1, game: 'volley-rush', repeatable: true, rewardUnit: 'punto' },
  { key: 'setter-pro', emoji: '🙌', title: 'Setter Pro', subtitle: 'Un tocco, l’alzata perfetta · Servi il tuo attaccante', pts: 1, game: 'setter-pro', repeatable: true, rewardUnit: 'alzata vincente' },
  { key: 'monster-block', emoji: '🧱', title: 'Monster Block', subtitle: 'Leggi l’attacco · Chiudi il muro al momento giusto', pts: 1, game: 'monster-block', repeatable: true, rewardUnit: 'muro vincente' },
  { key: 'basketball', emoji: '🏀', title: 'Basket Rush', subtitle: 'Trascina e tira · 1 moneta a canestro', pts: 1, game: 'basketball', repeatable: true, rewardUnit: 'canestro' },
  { key: 'playmaker', emoji: '🤝', title: 'Playmaker', subtitle: 'L’assist perfetto · Sessioni da 30 secondi', pts: 1, game: 'playmaker', repeatable: true, rewardUnit: 'assist' },
  { key: 'crossover', emoji: '💨', title: 'Crossover Rush', subtitle: 'Finta, cambia mano e schiaccia · 3 duelli', pts: 1, game: 'crossover', repeatable: true, rewardUnit: 'duello vinto' },
  { key: 'tap', emoji: '👆', title: 'Tap Challenge', subtitle: '10 secondi a martello', pts: 8, game: 'tap', repeatable: true },
  { key: 'tap-volley', emoji: '🏐', title: 'Tap Volley Challenge', subtitle: '10 secondi sotto rete · Tocca e insegui il pallone', pts: 8, game: 'tap-volley', repeatable: true },
  {
    key: 'sponsor-rush',
    emoji: '🏷️',
    title: rushSponsor.value ? `Sponsor Rush · ${rushSponsor.value.name}` : 'Sponsor Rush',
    subtitle: 'Prendi i loghi al volo',
    pts: 12,
    game: 'sponsor-rush',
    repeatable: true,
  },
  { key: 'memory-flash', emoji: '🧩', title: 'Memory Flash', subtitle: 'Memorizza le coppie', pts: 8, game: 'memory-flash', repeatable: true, rewardUnit: 'vittoria' },
].filter(mission => !mission.game || isGameAvailable(mission.game)));

/* ---------- Sponsor rush / partner ---------- */
const rushSponsor = computed(() => {
  const first = sponsors.value[0];
  if (!first) return null;
  return { name: first.name, badge: initials(first.name).slice(0, 1) };
});

/* ---------- Sheets ---------- */
function openSheet(id) {
  if (!isRegistered.value && ['wallet', 'profile'].includes(id)) {
    openRegistration(id === 'wallet' ? 'wallet' : 'profile_overlay');
    return;
  }
  sheetRevision += 1;
  if (id !== 'vote-result') cancelVotePresentation();
  recordSectionChange(id);
  activeSheet.value = id;
  if (id === 'mvp') {
    missionMvpCandidate.value = null;
    voteError.value = '';
    recordFanEvent('vote_opened', { is_change: hasVoted.value });
  }
  if (id === 'spend') reportTelemetry?.record('shop_view');
  if (['wallet', 'spend', 'profile'].includes(id)) return loadWallet();
}
function closeSheets() {
  const wasPlatformInfo = activeSheet.value === 'info';
  sheetRevision += 1;
  cancelVotePresentation();
  recordSectionChange('home');
  activeSheet.value = '';
  if (wasPlatformInfo) platformInfoTriggerRef.value?.focus({ preventScroll: true });
}
async function openPlatformInfo() {
  recordFanEvent('cta_clicked', { cta: 'info', placement: 'topbar' });
  openSheet('info');
  await nextTick();
  if (activeSheet.value === 'info') platformInfoRef.value?.focus({ preventScroll: true });
}
function trapPlatformInfoFocus(event) {
  const panel = platformInfoRef.value;
  const buttons = panel?.querySelectorAll('button:not(:disabled)');
  if (!buttons?.length) return;
  const first = buttons[0], last = buttons[buttons.length - 1];
  if (event.shiftKey && (document.activeElement === first || document.activeElement === panel)) {
    event.preventDefault(); last.focus({ preventScroll: true });
  } else if (!event.shiftKey && (document.activeElement === last || document.activeElement === panel)) {
    event.preventDefault(); first.focus({ preventScroll: true });
  }
}
function showCollectible(item, { preview = false, reveal = false, origin = 'spend' } = {}) {
  selectedCollectible.value = { ...item };
  isCollectiblePreview.value = preview;
  isCollectibleNew.value = reveal;
  collectibleOrigin.value = origin;
  collectibleRevealRevision.value += 1;
  openSheet('collectible');
}
function previewCollectible(item) {
  if (shopBusy.value || !catalog.value.some((entry) => entry.key === item?.key)) return;
  const owned = collection.value.items.find((entry) => entry.key === item.key && (entry.kind === 'frame' || Number(entry.event_id) === props.eventId));
  recordFanEvent('shop_item_viewed', { item_key: item.key, outcome: owned ? 'completed' : 'selected' });
  showCollectible(owned || { ...(item.kind === 'match_card' ? shopMatchMemory.value : {}), ...item }, { preview: !owned });
}
function inspectCollectible(memory) {
  if (shopBusy.value) return;
  const owned = collection.value.items.find((entry) => entry.key === memory?.key && Number(entry.event_id) === Number(memory?.event_id));
  if (owned) {
    recordFanEvent('shop_item_viewed', { item_key: owned.key, placement: 'profile', outcome: 'completed' });
    showCollectible(owned, { origin: 'profile' });
  }
}
function closeCollectible() { openSheet(collectibleOrigin.value); }

/* ---------- Wallet ---------- */
function readStoredCoins() {
  try {
    const raw = window.localStorage.getItem(`${WALLET_STORAGE_KEY}:${walletScope()}`);
    const parsed = Number.parseInt(raw ?? '', 10);
    return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
  } catch (error) {
    return 0;
  }
}
function writeStoredCoins(value) {
  try {
    window.localStorage.setItem(`${WALLET_STORAGE_KEY}:${walletScope()}`, String(value));
  } catch (error) {
    /* localStorage non disponibile */
  }
}
function setCoins(value) {
  const next = Math.max(0, Math.round(Number(value) || 0));
  totalCoins.value = next;
  walletRevision += 1;
  writeStoredCoins(next);
}

function walletScope() { return `${props.organizationSlug}:${props.eventId}`; }
function applyWallet(data, updateBalance = true) {
  if (!Number.isInteger(data?.wallet) || data.wallet < 0 || !Number.isInteger(data?.points) || data.points < 0 || !Array.isArray(data?.catalog) || !Array.isArray(data?.collection?.items)) return false;
  if (updateBalance) { setCoins(data.wallet); isWalletReady.value = true; }
  participationPoints.value = data.points;
  catalog.value = data.catalog.filter((item) => item && typeof item.key === 'string' && Number.isInteger(item.cost) && item.cost >= 0);
  walletMovements.value = Array.isArray(data.movements) ? data.movements : [];
  collection.value = { ...emptyCollection(), ...data.collection, badges: Array.isArray(data.collection.badges) ? data.collection.badges : [] };
  hasLoadedCollection.value = true;
  return true;
}
async function loadWallet({ collectionOnly = false } = {}) {
  if (redeeming.value || equipping.value) return;
  const request = ++walletRequestRevision;
  if (!props.eventId) { walletError.value = 'Disponibile con una partita del club.'; return; }
  const scope = walletScope();
  const revision = walletRevision;
  const startedAt = Date.now();
  isLoadingWallet.value = true;
  walletError.value = '';
  try {
    const response = await fetchFanWallet(props.eventId);
    if (isDisposed || request !== walletRequestRevision || scope !== walletScope()) return;
    if (revision !== walletRevision) { loadWallet({ collectionOnly }); return; }
    if (!response.ok || !applyWallet(response.data, !collectionOnly)) {
      walletError.value = response.message || 'Impossibile caricare il wallet. Riprova.';
      recordFanEvent('wallet_failed', { reason: analyticsFailureReason(response), duration_ms: Math.max(0, Date.now() - startedAt) });
    } else recordFanEvent('wallet_loaded', { duration_ms: Math.max(0, Date.now() - startedAt) });
  } catch (error) {
    if (!isDisposed && request === walletRequestRevision && scope === walletScope()) {
      walletError.value = 'Impossibile caricare il wallet. Riprova.';
      recordFanEvent('wallet_failed', { reason: 'network' });
    }
  } finally {
    if (request === walletRequestRevision) isLoadingWallet.value = false;
  }
}
function formatWalletDate(value) {
  const parsed = new Date(String(value || '').replace(' ', 'T') + (/Z$|[+-]\d\d:\d\d$/.test(String(value || '')) ? '' : 'Z'));
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleString('it-IT', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
}

async function loadFanProfile() {
  if (!props.eventId) return;
  const eventId = props.eventId;
  const scope = walletScope();
  const revision = walletRevision;
  const startedAt = Date.now();
  let response;
  try { response = await fetchFanProfile(eventId); }
  catch (error) {
    if (!isDisposed && scope === walletScope()) recordFanEvent('profile_failed', { reason: 'network' });
    throw error;
  }
  const { ok, data } = response;
  if (isDisposed || scope !== walletScope()) return;
  if (!ok || !data) { recordFanEvent('profile_failed', { reason: analyticsFailureReason(response) }); return; }
  isWalletReady.value = true;
  if (data.journey_rewards && typeof data.journey_rewards === 'object') {
    const rewards = revision === walletRevision ? {} : { ...journey.value.rewards };
    const restored = normalizeJourney({ rewards: data.journey_rewards });
    homeVoteReward.value = revision === walletRevision ? restored.rewards.vote || 0 : Math.max(homeVoteReward.value, restored.rewards.vote || 0);
    for (const [key, amount] of Object.entries(restored.rewards)) rewards[key] = Math.max(rewards[key] || 0, amount);
    journey.value = normalizeJourney({ ...journey.value, rewards });
    if (journey.value.rewards.vote) journey.value.pendingVoteBonus = false;
    if (revision === walletRevision) {
      doneKeys.value = Object.keys(journey.value.rewards);
      try { window.localStorage.setItem(doneStorageKey(), JSON.stringify(doneKeys.value)); } catch (error) { /* storage non disponibile */ }
    }
    for (const key of Object.keys(journey.value.rewards)) markDone(key);
    saveJourney();
  }
  isRegistered.value = Boolean(data.registered);
  if (revision === walletRevision && data.registered) {
    setCoins(Number(data.wallet) || 0);
  } else if (revision === walletRevision) {
    setCoins(Number(data.guest_coins) || 0);
  }
  fanNickname.value = String(data.user?.nickname || '').trim();
  if (!isNicknameEditorOpen.value) {
    nicknameDraft.value = fanNickname.value;
  }
  fanLotteryTicket.value = data.lottery_ticket || null;
  if (data.user_rank) {
    leaderboardUser.value = { ...data.user_rank, points: Number(data.user_rank.points ?? data.user_rank.coins ?? 0) };
  }
  recordFanEvent('profile_loaded', { registered: Boolean(data.registered), has_ticket: Boolean(data.lottery_ticket), duration_ms: Math.max(0, Date.now() - startedAt) });
  return data;
}

// Mostra l'accredito soltanto dopo la conferma del salvataggio del wallet.
async function grantCoins(delta, key, roundId = '') {
  const amount = Math.max(0, Math.floor(Number(delta) || 0));
  if (!props.eventId || !amount || shopBusy.value) return { ok: false, coins: 0 };
  if (key === 'vote' && (journey.value.rewards[key] || doneKeys.value.includes(key))) return { ok: true, coins: 0 };
  isSavingCoins.value = true;
  coinError.value = '';
  const eventId = props.eventId;
  const scope = walletScope();
  const request = ++journeyRewardRevision;
  try {
    const response = await claimFanJourneyReward(eventId, key, roundId);
    if (isDisposed || request !== journeyRewardRevision || scope !== walletScope()) return { ok: false, coins: 0 };
    const rewardAmount = response.data?.reward_amount;
    if (!response.ok || !Number.isInteger(response.data?.wallet) || response.data.wallet < 0 || !Number.isInteger(rewardAmount) || rewardAmount <= 0) {
      coinError.value = response.message || 'Le monete non sono ancora state salvate. Controlla la connessione e riprova.';
      return { ok: false, coins: 0 };
    }
    setCoins(response.data.wallet);
    isWalletReady.value = true;
    const total = response.data?.reward_total;
    if (key !== 'vote' && Number.isInteger(total) && total >= 0) {
      journey.value = normalizeJourney({ ...journey.value, rewards: { ...journey.value.rewards, [key]: Math.max(journey.value.rewards[key] || 0, total) } });
    } else {
      journey.value = recordJourneyReward(journey.value, key, key === 'vote' ? rewardAmount : Number(response.data?.coins_earned || 0));
    }
    saveJourney();
    markDone(key);
    refreshLeaderboard();
    return { ok: true, coins: Number(response.data?.coins_earned ?? 0) };
  } catch (error) {
    if (isDisposed || request !== journeyRewardRevision || scope !== walletScope()) return { ok: false, coins: 0 };
    coinError.value = 'Le monete non sono ancora state salvate. Controlla la connessione e riprova.';
    return { ok: false, coins: 0 };
  } finally { if (!isDisposed && request === journeyRewardRevision && scope === walletScope()) isSavingCoins.value = false; }
}

async function retryVoteBonus({ inline = false } = {}) {
  if (!journey.value.pendingVoteBonus) return;
  const scope = walletScope();
  const result = await grantCoins(VOTE_REWARD, 'vote');
  if (scope === walletScope()) {
    if (!result.ok) recordFanEvent('vote_reward_failed', { reason: 'save_failed' });
    else if (result.coins > 0) recordFanEvent('vote_reward_claimed', { amount: result.coins, outcome: 'success' });
  }
  if (isDisposed || scope !== walletScope()) return;
  if (result.ok) {
    journey.value.pendingVoteBonus = false;
    saveJourney();
    homeVoteReward.value = journey.value.rewards.vote || homeVoteReward.value;
    if (!inline) {
      voteBonusCoins.value = result.coins;
      if (result.coins) animateVoteReward();
    }
  }
}

/* ---------- Missioni completate (persistite per evento) ---------- */
function doneStorageKey() {
  return `cento:done:${props.eventId || 'anon'}`;
}
function loadDoneKeys() {
  try {
    const raw = window.localStorage.getItem(doneStorageKey());
    const parsed = raw ? JSON.parse(raw) : [];
    doneKeys.value = Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    doneKeys.value = [];
  }
}
function markDone(key) {
  if (doneKeys.value.includes(key)) return;
  doneKeys.value = [...doneKeys.value, key];
  try {
    window.localStorage.setItem(doneStorageKey(), JSON.stringify(doneKeys.value));
  } catch (error) {
    /* localStorage non disponibile */
  }
}

function onMission(mission, placement = 'game_catalog') {
  recordFanEvent('cta_clicked', { cta: mission.opensMvp ? 'mvp' : 'earn', game_id: mission.game, placement });
  // "Vota l'MVP" apre la lista giocatori; le monete arrivano col voto reale.
  if (mission.opensMvp) {
    openSheet('mvp');
    return;
  }
  if (mission.game) {
    openGame(mission.game);
  }
}

/* ---------- Giochi reali (overlay a schermo pieno) ---------- */
function openGame(id) {
  if (!games[id] || !isGameAvailable(id) || !props.eventId || shopBusy.value || matchConcluded.value || (id === 'quiz' && !quizAvailable.value)) return;
  coinError.value = '';
  pendingGameClaim.value = null;
  activeGameRoundId.value = newMinigameRoundId();
  closeSheets();
  activeGameId.value = id;
  analyticsGameOpenedAt = Date.now();
  recordFanEvent('game_opened', { game_id: id });
}
function closeGame(reason = 'user_closed') {
  if (isSavingCoins.value || isBasketballSaving.value || isPlaymakerSaving.value || isCrossoverSaving.value || isVolleyballSaving.value) return;
  const wasVolleyball = volleyballGameIds.includes(activeGameId.value);
  const wasGame = Boolean(activeGameId.value);
  if (wasGame) recordGameExit(typeof reason === 'string' ? reason : 'user_closed');
  activeGameId.value = null;
  if (wasGame) {
    openSheet('earn'); // torna alla lista giochi
    if (wasVolleyball) void loadFanProfile();
  }
}
function onBasketballCredit(receipt) {
  applyLiveGameCredit('basketball', receipt, 'basketball_earned');
}
function onPlaymakerCredit(receipt) {
  applyLiveGameCredit('playmaker', receipt, 'playmaker_earned');
}
function onCrossoverCredit(receipt) {
  applyLiveGameCredit('crossover', receipt, 'crossover_earned');
}
function onVolleyballCredit(receipt) {
  if (!volleyballGameIds.includes(receipt?.game_id) || receipt.game_id !== activeGameId.value) return;
  applyLiveGameCredit(receipt.game_id, receipt, 'game_earned');
}
function applyLiveGameCredit(key, receipt, totalField) {
  if (receipt?.event_id !== props.eventId) return;
  const wallet = Number(receipt.wallet);
  const earned = Number(receipt[totalField]);
  if (!Number.isInteger(wallet) || wallet < 0 || !Number.isInteger(earned) || earned < 0) return;
  const previousEarned = journey.value.rewards[key] || 0;
  setCoins(wallet);
  isWalletReady.value = true;
  journey.value = normalizeJourney({ ...journey.value, rewards: {
    ...journey.value.rewards, [key]: Math.max(journey.value.rewards[key] || 0, earned),
  } });
  saveJourney();
  if (earned > 0) markDone(key);
  if (Number(receipt.coins_earned) > 0) {
    if (earned > previousEarned) recordFanEvent('game_reward_claimed', { game_id: key, amount: Number(receipt.coins_earned), outcome: 'success' });
    animateEarnedCoins(); refreshLeaderboard();
  }
}
// Ogni gioco emette @claim { coins, keepOpen } al termine.
async function onGameClaim(payload) {
  if (!activeGameId.value || isSavingCoins.value) return;
  const key = activeGameId.value;
  const scope = walletScope();
  const launchRound = activeGameRoundId.value;
  const isCurrentClaim = () => !isDisposed && scope === walletScope() && key === activeGameId.value && launchRound === activeGameRoundId.value;
  const roundId = payload?.roundId || activeGameRoundId.value;
  pendingGameClaim.value = { ...payload, roundId };
  const coins = key === 'quiz' ? QUIZ_REWARD : Math.max(0, Number(payload?.coins) || 0);
  let earned = 0;
  if (coins && String(payload?.source || '').startsWith('tap_challenge')) {
    // Tap Challenge already credits its result server-side; only refresh the wallet here.
    const previousEarned = journey.value.rewards[key] || 0;
    isWalletReady.value = false;
    await loadFanProfile();
    if (!isCurrentClaim()) return;
    if (!isWalletReady.value) { coinError.value = 'Risultato salvato. Riprova a caricare il saldo.'; recordFanEvent('game_reward_failed', { game_id: key, round_id: roundId, reason: 'load_failed' }); return; }
    earned = Math.max(0, (journey.value.rewards[key] || 0) - previousEarned);
  } else if (coins) {
    const result = await grantCoins(coins, key, roundId);
    if (!isCurrentClaim()) return;
    if (!result.ok) { recordFanEvent('game_reward_failed', { game_id: key, round_id: roundId, reason: 'save_failed' }); return; }
    earned = result.coins;
  }
  pendingGameClaim.value = null;
  if (earned > 0) recordFanEvent('game_reward_claimed', { game_id: key, round_id: roundId, amount: earned, outcome: 'success', mode: payload?.keepOpen ? 'live' : 'singleplayer' });
  if (!payload?.keepOpen) {
    closeGame('completed');
    if (earned) {
      celebrationKind.value = key;
      celebrationName.value = games[key].title;
      voteBonusCoins.value = earned;
      openSheet('vote-result');
      animateEarnedCoins();
    }
  }
}
// I tentativi Memory Flash sono gratuiti: il saldo cambia solo sul server.

/* ---------- Voto MVP (lista reale + voto reale) ---------- */
async function loadPlayers() {
  const scope = walletScope();
  const request = ++playersRequestRevision;
  isLoadingPlayers.value = true;
  playersError.value = '';
  try {
    const { data } = await apiClient.get('/public/players');
    if (isDisposed || request !== playersRequestRevision || scope !== walletScope()) return;
    const payload = Array.isArray(data?.players) ? data.players : Array.isArray(data) ? data : [];
    rawPlayers.value = Array.isArray(payload) ? payload : [];
  } catch (error) {
    if (isDisposed || request !== playersRequestRevision || scope !== walletScope()) return;
    playersError.value = 'Impossibile caricare i giocatori. Riprova.';
    recordFanEvent('home_load_failed', { section: 'mvp', reason: 'load_failed' });
    rawPlayers.value = [];
  } finally {
    if (!isDisposed && request === playersRequestRevision && scope === walletScope()) isLoadingPlayers.value = false;
  }
}
async function loadVoteStatus() {
  if (!props.eventId) return;
  const eventId = props.eventId;
  const scope = walletScope();
  const request = ++voteStatusRequestRevision;
  isLoadingVoteStatus.value = true;
  voteStatusError.value = '';
  const status = await fetchVoteStatus(eventId).catch(() => ({ ok: false }));
  if (isDisposed || request !== voteStatusRequestRevision || scope !== walletScope()) return;
  isLoadingVoteStatus.value = false;
  if (isVoting.value || localVoted.value) return;
  if (!status.ok) {
    voteStatusError.value = 'Impossibile verificare il tuo voto. Riprova.';
    return;
  }
  hasVoted.value = Boolean(status?.hasVoted);
  votedPlayerId.value = status?.playerId ?? null;
  if (hasVoted.value && !journey.value.rewards.vote) {
    journey.value.pendingVoteBonus = true;
    saveJourney();
    await retryVoteBonus();
  }
}
async function castVote(player, { inline = false } = {}) {
  if (isDisposed || isVoting.value || shopBusy.value || !player?.id || !props.eventId) return;
  voteError.value = '';
  if (!canVote.value) { voteError.value = 'Le votazioni per questa partita sono chiuse.'; return; }
  isVoting.value = true;
  pendingVotePlayerId.value = player.id;
  const request = ++voteRequestRevision;
  const scope = walletScope();
  const presentationRevision = sheetRevision;
  const firstVote = !hasVoted.value && !doneKeys.value.includes('vote');
  const eventId = props.eventId;
  const startedAt = Date.now();
  const previousPlayerId = votedPlayerId.value;
  const isChange = hasVoted.value && String(previousPlayerId) !== String(player.id);
  recordFanEvent('vote_player_selected', { player_id: Number(player.id), previous_player_id: Number(previousPlayerId) || undefined, is_change: isChange });
  recordFanEvent('vote_submitted', { player_id: Number(player.id), is_change: isChange });
  try {
    const response = await vote({ eventId, playerId: player.id });
    if (isDisposed || request !== voteRequestRevision || scope !== walletScope()) return;
    if (!response?.ok) {
      voteError.value = response?.message || 'Il voto non è stato salvato. Controlla la connessione e riprova.';
      recordFanEvent('vote_failed', { player_id: Number(player.id), is_change: isChange, reason: analyticsFailureReason(response), duration_ms: Math.max(0, Date.now() - startedAt) });
      return;
    }
    hasVoted.value = true;
    if (isRegistered.value && String(response.vote?.code || '').trim()) {
      fanLotteryTicket.value = { ticket_code: String(response.vote.code).trim(), qr_data: String(response.vote.qr_data || '').trim() };
    }
    recordFanEvent('vote_completed', { player_id: Number(player.id), is_change: isChange, duration_ms: Math.max(0, Date.now() - startedAt) });
    if (isChange) recordFanEvent('vote_changed', { player_id: Number(player.id), previous_player_id: Number(previousPlayerId) || undefined, is_change: true });
    votedPlayerId.value = player.id;
    pendingVotePlayerId.value = null;
    localVoted.value = { ...player };
    if (!inline) {
      voteBonusCoins.value = 0;
      celebrationKind.value = 'vote';
    }
    if (firstVote) {
      journey.value.pendingVoteBonus = true;
      saveJourney();
    }
    mvpEditing.value = false;
    homeMvpSelectedId.value = null;
    if (inline) {
      highlightPlay.value = true;
      if (presentationRevision === sheetRevision && !activeSheet.value && !isHomeLoading.value
        && !isRegistrationOpen.value && !activeGameId.value && !shopBusy.value) {
        if (prefersReducedMotion()) {
          finishHomeVoteFlight();
        } else {
          homeVoteFlight.value = { source: homeMvpCarousel.value?.getPlayerCardRect(player.id) ?? null };
        }
      }
    } else if (presentationRevision === sheetRevision && (!activeSheet.value || activeSheet.value === 'mvp')) {
      startVotePresentation();
      openSheet('vote-result');
    }
    if (journey.value.pendingVoteBonus) {
      const reward = retryVoteBonus({ inline });
      if (inline) homeVoteBonusRequest = reward;
      try {
        await reward;
      } finally {
        if (homeVoteBonusRequest === reward) homeVoteBonusRequest = null;
      }
    }
  } catch (error) {
    if (isDisposed || request !== voteRequestRevision || scope !== walletScope()) return;
    voteError.value = 'Impossibile salvare il voto. Riprova tra poco.';
    recordFanEvent('vote_failed', { player_id: Number(player.id), is_change: isChange, reason: 'network', duration_ms: Math.max(0, Date.now() - startedAt) });
  } finally {
    if (!isDisposed && request === voteRequestRevision && scope === walletScope()) {
      isVoting.value = false;
      pendingVotePlayerId.value = null;
    }
  }
}

/* ---------- Leaderboard ---------- */
async function refreshLeaderboard() {
  const request = ++leaderboardRequestRevision;
  const scope = walletScope();
  const period = activeSheet.value === 'leaderboard' ? leaderboardPeriod.value : 'general';
  if (!props.eventId) {
    leaderboardRows.value = [];
    leaderboardUser.value = null;
    leaderboardError.value = '';
    isLoadingLeaderboard.value = false;
    return;
  }
  isLoadingLeaderboard.value = true;
  leaderboardError.value = '';
  try {
    const { data } = await apiClient.get(`/events/${props.eventId}/coins-leaderboard`, {
      headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() },
      // Fetch all rows only while the list is open, including live updates.
      params: activeSheet.value === 'leaderboard' ? { view: 'list', period } : {},
    });
    if (isDisposed || request !== leaderboardRequestRevision || scope !== walletScope()) return;
    const rank = data?.userRank || data?.user_rank;
    if (period === 'general') leaderboardUser.value = rank ? { ...rank, points: Number(rank.points ?? rank.coins ?? 0) } : null;
    const top = Array.isArray(data?.top3) ? data.top3 : Array.isArray(data?.top) ? data.top : [];
    // Older backend responses remain a single list with no repeated positions.
    const ranking = Array.isArray(data?.ranking) ? data.ranking : [
      ...top.map((row, index) => ({ ...row, rank: row.rank ?? index + 1 })),
      ...(Array.isArray(data?.nearby) ? data.nearby : []),
      ...(rank ? [{ ...rank, name: profileNickname.value, is_me: true }] : []),
    ];
    const positions = new Map();
    for (const row of ranking) {
      const position = Number(row.rank);
      if (!Number.isInteger(position) || position < 1) continue;
      positions.set(position, {
        rank: position,
        name: String(row.name || row.nickname || 'Tifoso'),
        points: Number(row.points ?? row.coins ?? 0),
        equipped_frame: row.equipped_frame === 'club_frame' ? 'club_frame' : '',
        is_me: row.is_me === true || position === Number(rank?.rank),
      });
    }
    leaderboardRows.value = [...positions.values()].sort((a, b) => a.rank - b.rank);
    if (activeSheet.value === 'leaderboard') recordFanEvent('leaderboard_loaded', { count: leaderboardRows.value.length });
  } catch (error) {
    if (!isDisposed && request === leaderboardRequestRevision && scope === walletScope()) {
      leaderboardError.value = 'Impossibile caricare la classifica. Riapri il pannello per riprovare.';
      recordFanEvent('leaderboard_failed', { reason: 'network' });
    }
  } finally {
    if (request === leaderboardRequestRevision) isLoadingLeaderboard.value = false;
  }
}
function openLeaderboard() {
  leaderboardPeriod.value = 'event';
  leaderboardRows.value = [];
  openSheet('leaderboard');
  refreshLeaderboard();
}
function selectLeaderboardPeriod(period) {
  if (!['general', 'event'].includes(period) || leaderboardPeriod.value === period) return;
  leaderboardPeriod.value = period;
  leaderboardRows.value = [];
  return refreshLeaderboard();
}

/* ---------- Profilo ---------- */
function openProfile() {
  if (!isRegistered.value) { openRegistration(); return; }
  nicknameDraft.value = fanNickname.value.trim();
  isNicknameEditorOpen.value = false;
  nicknameError.value = '';
  nicknameSuccess.value = '';
  const profileLoading = loadFanProfile();
  const walletLoading = openSheet('profile');
  return Promise.allSettled([profileLoading, walletLoading]);
}
async function openLotteryTicket() {
  if (!showLotteryShortcut.value) return;
  recordFanEvent('lottery_shortcut_clicked', { placement: 'lottery_shortcut', has_ticket: true });
  recordFanEvent('cta_clicked', { cta: 'lottery', placement: 'lottery_shortcut' });
  const scope = walletScope();
  const loading = openProfile();
  const revision = sheetRevision;
  await loading;
  await nextTick();
  if (isDisposed || scope !== walletScope() || revision !== sheetRevision || activeSheet.value !== 'profile' || !showLotteryShortcut.value) return;
  const body = profileBodyEl.value;
  const ticket = lotteryTicketEl.value;
  if (!body || !ticket) return;
  const top = body.scrollTop + ticket.getBoundingClientRect().top - body.getBoundingClientRect().top - 8;
  body.scrollTo({ top: Math.max(0, top), behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
  ticket.focus({ preventScroll: true });
}
function toggleNicknameEditor() {
  isNicknameEditorOpen.value = !isNicknameEditorOpen.value;
  if (isNicknameEditorOpen.value) recordFanEvent('profile_edit_started', { section: 'profile' });
  if (!isNicknameEditorOpen.value) {
    nicknameError.value = '';
    nicknameSuccess.value = '';
  }
}
function validateNickname(raw) {
  const n = String(raw || '').trim();
  if (!n) return { valid: false, message: 'Il nickname non può essere vuoto.' };
  if (n.length < FAN_NICKNAME_MIN_LEN) return { valid: false, message: 'Il nickname deve avere almeno 3 caratteri.' };
  if (n.length > FAN_NICKNAME_MAX_LEN) return { valid: false, message: 'Il nickname può avere massimo 24 caratteri.' };
  if (!FAN_NICKNAME_PATTERN.test(n)) return { valid: false, message: 'Usa solo lettere, numeri, spazio, punto, trattino o underscore.' };
  return { valid: true, nickname: n };
}
async function submitNickname() {
  if (isSavingNickname.value) return;
  nicknameError.value = '';
  nicknameSuccess.value = '';
  const check = validateNickname(nicknameDraft.value);
  if (!check.valid) {
    nicknameError.value = check.message;
    recordFanEvent('profile_update_failed', { reason: 'validation' });
    return;
  }
  if (check.nickname === fanNickname.value.trim()) {
    nicknameSuccess.value = 'Nessuna modifica da salvare.';
    return;
  }
  isSavingNickname.value = true;
  const scope = walletScope();
  const request = ++analyticsNicknameRevision;
  const isCurrent = () => !isDisposed && scope === walletScope() && request === analyticsNicknameRevision;
  try {
    const response = await updateFanNickname(check.nickname);
    if (!isCurrent()) return;
    if (!response?.ok) {
      nicknameError.value = response?.message || 'Impossibile aggiornare il nickname.';
      recordFanEvent('profile_update_failed', { reason: analyticsFailureReason(response) });
      return;
    }
    fanNickname.value = String(response.data?.user?.nickname || check.nickname).trim();
    nicknameDraft.value = fanNickname.value;
    nicknameSuccess.value = response.data?.message || 'Nickname aggiornato';
    isNicknameEditorOpen.value = false;
    recordFanEvent('profile_updated', { outcome: 'success' });
    refreshLeaderboard();
  } catch {
    if (isCurrent()) { nicknameError.value = 'Impossibile salvare il nickname. Riprova.'; recordFanEvent('profile_update_failed', { reason: 'network' }); }
  } finally { if (isCurrent()) isSavingNickname.value = false; }
}

/* ---------- Sponsors ---------- */
const wallSponsors = computed(() => resolveGameWallSponsors(
  gameWallAssignments.value,
  activeGameId.value,
  currentSponsorCatalog.value,
));
const postVoteSponsors = computed(() => postVoteSponsorSlots
  .map(({ id: placement }) => ({ placement, sponsor: findAssignedSponsor(postVoteAssignments.value[placement], currentSponsorCatalog.value) }))
  .filter(({ sponsor }) => sponsor));

// Stessa normalizzazione di LiveExperienceHome: sponsor = logo (logo_data/image_url).
function normalizeSponsor(item, index, matchId) {
  const imageUrl = String(item?.logo_data || item?.image_url || item?.imageUrl || '').trim();
  if (!imageUrl) return null;
  const priorityRaw = Number(item?.priority ?? item?.order_index ?? item?.order ?? item?.display_order);
  const analytics = sponsorAnalyticsFields(item, matchId);
  return {
    ...analytics,
    id: analytics.sponsorId || index + 1,
    name: String(item?.name || '').trim(),
    imageUrl,
    isActive: item?.is_active !== false,
    linkUrl: String(item?.link_url || item?.linkUrl || '').trim(),
    priority: Number.isFinite(priorityRaw) ? priorityRaw : Number.POSITIVE_INFINITY,
    insertedIndex: index,
  };
}
async function loadSponsors(signal) {
  const slug = props.organizationSlug;
  const matchId = props.eventId || 0;
  try {
    const { data } = await apiClient.get('/sponsors', {
      params: { include_placements: 1, analytics_match_id: matchId },
      signal,
      headers: slug ? { 'X-Organization-Slug': slug } : undefined,
    });
    if (signal.aborted || slug !== props.organizationSlug || matchId !== (props.eventId || 0)) return;
    const list = Array.isArray(data) ? data : Array.isArray(data?.sponsors) ? data.sponsors : [];
    sponsorCatalog.value = list
      .map((item, index) => normalizeSponsor(item, index, matchId))
      .filter(Boolean)
      .sort((a, b) => (a.priority !== b.priority ? a.priority - b.priority : a.insertedIndex - b.insertedIndex));
    sponsorCatalogSlug.value = slug;
  } catch (error) {
    if (!signal.aborted && slug === props.organizationSlug) sponsorCatalog.value = [];
  }
}
function onSponsorClick(sponsor) {
  if (sponsor?.linkUrl) {
    window.open(sponsor.linkUrl, '_blank', 'noopener');
  }
}

/* ---------- Riscatto premi ---------- */
async function redeem(reward) {
  if (shopBusy.value || !props.eventId || !reward || reward.owned) return;
  if (!isRegistered.value) { recordFanEvent('redemption_failed', { item_key: reward.key, reason: 'registration_required' }); openRegistration('spend_redeem', reward.title); return; }
  if (isLoadingWallet.value || walletError.value || totalCoins.value < reward.cost) {
    recordFanEvent('redemption_failed', { item_key: reward.key, reason: totalCoins.value < reward.cost ? 'insufficient_coins' : 'not_ready' });
    return;
  }
  const scope = walletScope();
  recordFanEvent('redemption_started', { item_key: reward.key, amount: reward.cost });
  let profileRequired = false;
  walletRequestRevision += 1;
  redeeming.value = reward.key;
  shopMessage.value = '';
  try {
    const response = await purchaseFanItem(props.eventId, reward.key);
    if (isDisposed || scope !== walletScope()) return;
    if (!response.ok && (response.status === 401 || response.code === 'PROFILE_REQUIRED')) {
      recordFanEvent('redemption_failed', { item_key: reward.key, reason: 'session_expired' });
      isRegistered.value = false;
      profileRequired = true;
      return;
    }
    if (!response.ok || !applyWallet(response.data)) { walletError.value = response.message || 'Acquisto non confermato. Riprova: gli oggetti già ottenuti non vengono addebitati due volte.'; recordFanEvent('redemption_failed', { item_key: reward.key, reason: analyticsFailureReason(response) }); return; }
    walletError.value = '';
    if (response.data.purchased !== false) recordFanEvent('redemption_completed', { item_key: reward.key, amount: reward.cost });
    shopMessage.value = response.data.purchased === false ? 'Questo oggetto è già nella tua collezione.' : `${reward.title} aggiunta alla tua collezione!`;
    const obtained = collection.value.items.find((item) => item.key === reward.key && (item.kind === 'frame' || Number(item.event_id) === props.eventId));
    if (obtained && ['spend', 'collectible'].includes(activeSheet.value)) showCollectible(obtained, { reveal: response.data.purchased !== false });
    refreshLeaderboard();
  } catch (error) {
    if (!isDisposed && scope === walletScope()) { walletError.value = 'Acquisto non confermato. Controlla la connessione e riprova.'; recordFanEvent('redemption_failed', { item_key: reward.key, reason: 'network' }); }
  } finally {
    if (!isDisposed && scope === walletScope()) {
      redeeming.value = '';
      if (profileRequired) openRegistration('spend_redeem', reward.title);
    }
  }
}
async function equipCollectionItem(key) {
  if (shopBusy.value || isLoadingWallet.value || !props.eventId) return;
  const scope = walletScope();
  walletRequestRevision += 1;
  equipping.value = true;
  walletError.value = '';
  try {
    const response = await equipFanItem(props.eventId, key);
    if (isDisposed || scope !== walletScope()) return;
    if (!response.ok || !applyWallet(response.data)) { walletError.value = response.message || 'Impossibile aggiornare la cornice. Riprova.'; recordFanEvent('frame_equip_failed', { item_key: 'club_frame', reason: analyticsFailureReason(response) }); }
    else { recordFanEvent('frame_equipped', { item_key: 'club_frame', outcome: key ? 'selected' : 'back' }); refreshLeaderboard(); }
  } catch (error) {
    if (!isDisposed && scope === walletScope()) { walletError.value = 'Impossibile aggiornare la cornice. Riprova.'; recordFanEvent('frame_equip_failed', { item_key: 'club_frame', reason: 'network' }); }
  } finally {
    if (scope === walletScope()) equipping.value = false;
  }
}

/* ---------- SSE classifica live ---------- */
function startCoinsStream() {
  if (!props.eventId || typeof window === 'undefined' || typeof EventSource === 'undefined') return;
  try {
    const slug = getOrganizationSlug();
    const base = resolveApiUrl(`/events/${props.eventId}/coins/stream`);
    const url = slug ? `${base}${base.includes('?') ? '&' : '?'}organization_slug=${encodeURIComponent(slug)}` : base;
    coinsStream = new EventSource(url);
    coinsStream.onmessage = () => refreshLeaderboard();
    coinsStream.onerror = () => {
      /* il browser ritenta da solo */
    };
  } catch (error) {
    coinsStream = null;
  }
}
function stopCoinsStream() {
  if (coinsStream) {
    coinsStream.close();
    coinsStream = null;
  }
}

/* ---------- Esc per chiudere ---------- */
function onKeydown(e) {
  if (e.key !== 'Escape') return;
  if (activeGameId.value) {
    closeGame();
    return;
  }
  closeSheets();
}

/* ---------- Lifecycle ---------- */
let reportTelemetry = null;
let reportTelemetryScope = '';
const reportRoundsStarted = new Set();
const reportRoundsCompleted = new Set();
const reportRoundTimes = new Map();
const reportRoundsAbandoned = new Set();
let analyticsSectionOpenedAt = Date.now();
let analyticsGameOpenedAt = 0;
let analyticsNicknameRevision = 0;
function recordFanEvent(name, metadata = {}) {
  try { reportTelemetry?.record(name, sanitizeFanAnalyticsMetadata(metadata)); } catch { /* Measurement must not interrupt the fan experience. */ }
}
function reportBrowserContext() {
  const ua = String(window.navigator?.userAgent || '');
  const width = Number(window.innerWidth);
  return {
    device_type: Number.isFinite(width) ? width < 768 ? 'mobile' : width < 1024 ? 'tablet' : 'desktop' : 'unknown',
    browser_family: !ua ? 'unknown' : /Edg\//.test(ua) ? 'Edge' : /Firefox\//.test(ua) ? 'Firefox' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : 'Other',
  };
}
function onFormTelemetry(event) {
  if (!['registration_step', 'registration_failed', 'registration_dismissed', 'login_started', 'login_completed', 'login_failed'].includes(event?.name)) return;
  recordFanEvent(event.name, { ...sanitizeFanAnalyticsMetadata(event.metadata), trigger: registrationTrigger.value });
}
function analyticsFailureReason(response) {
  const status = Number(response?.status ?? response?.error?.response?.status);
  return ({ 400: 'validation', 401: 'session_expired', 403: 'not_allowed', 404: 'not_found', 409: 'conflict', 429: 'rate_limited' })[status] || (status >= 500 ? 'server_error' : 'request_failed');
}
function recordSectionChange(next) {
  const previous = activeSheet.value || 'home';
  if (previous === next) return;
  recordFanEvent('section_closed', { section: previous, duration_ms: Math.max(0, Date.now() - analyticsSectionOpenedAt) });
  recordFanEvent('section_opened', { section: next, from_section: previous });
  analyticsSectionOpenedAt = Date.now();
}
function openHomeAction(id, placement = ['profile', 'wallet'].includes(id) ? 'topbar' : 'home_grid') {
  const cta = ({ spend: 'shop', 'match-summary': 'mvp' })[id] || id;
  recordFanEvent('cta_clicked', { cta, placement });
  if (id === 'profile') return openProfile();
  if (id === 'leaderboard') return openLeaderboard();
  return openSheet(id);
}
function recordGameExit(reason) {
  const gameId = activeGameId.value;
  if (!gameId) return;
  if (reason !== 'completed') {
    for (const [key, round] of reportRoundTimes) {
      if (round.gameId !== gameId || reportRoundsCompleted.has(key) || reportRoundsAbandoned.has(key)) continue;
      reportRoundsAbandoned.add(key);
      recordFanEvent('game_abandoned', { game_id: gameId, round_id: round.roundId, mode: round.mode, duration_ms: Math.max(0, Date.now() - round.startedAt), reason });
    }
  }
  recordFanEvent('game_closed', { game_id: gameId, duration_ms: Math.max(0, Date.now() - analyticsGameOpenedAt), outcome: reason === 'completed' ? 'completed' : 'abandoned', reason: reason === 'completed' ? undefined : reason });
}
function resetReportTelemetry() {
  const organizationSlug = props.organizationSlug;
  const scope = `${organizationSlug}:${props.eventId}`;
  if (scope === reportTelemetryScope && reportTelemetry) return;
  reportTelemetry?.stop();
  reportTelemetryScope = scope;
  reportRoundsStarted.clear(); reportRoundsCompleted.clear();
  reportRoundTimes.clear(); reportRoundsAbandoned.clear();
  analyticsSectionOpenedAt = Date.now();
  const deviceHeaders = getDeviceHeaders();
  reportTelemetry = props.eventId ? createEventReportTelemetry({
    eventId: props.eventId,
    identity: () => {
      let fanHeaders = {};
      try { fanHeaders = getFanSessionHeaders(organizationSlug); } catch { /* A guest remains measurable when storage is blocked. */ }
      return { ...deviceHeaders, ...fanHeaders, 'X-Organization-Slug': organizationSlug };
    },
    send: createReportSignalSender({ apiClient, resolveApiUrl }),
    context: () => ({
      organization_slug: organizationSlug, ...reportBrowserContext(),
      ...reportPageAvailability(), registered: isRegistered.value, has_voted: hasVoted.value,
      votes_closed: eventState.value?.votes_closed === true, is_concluded: matchConcluded.value,
      sport: organizationSport.value, section: activeSheet.value || 'home',
      game_id: activeGameId.value || undefined, has_ticket: showLotteryShortcut.value,
    }),
  }) : null;
}
function recordMatchChat(name) {
  if (name === 'match_chat_opened') sheetRevision += 1;
  reportTelemetry?.record(name);
}
function observeLotteryShortcut(element) {
  reportTelemetry?.observeVisibility?.(element, 'lottery_shortcut_view', { placement: 'lottery_shortcut', has_ticket: true });
}
function observeFeedbackInvitation(element) {
  reportTelemetry?.observeVisibility?.(element, 'cta_viewed', { cta: 'feedback', placement: 'feedback_home' });
}
function bindLotteryTicket(element) {
  lotteryTicketEl.value = element;
  reportTelemetry?.observeVisibility?.(element, 'lottery_qr_view', { placement: 'profile_ticket', has_ticket: true });
}
function observeReportMission(element, mission) {
  reportTelemetry?.observeVisibility?.(element, 'cta_viewed', { cta: mission.opensMvp ? 'mvp' : 'earn', game_id: mission.game, placement: 'game_catalog' }, { onceKey: `game_catalog:${mission.key}` });
}
function reportGameRound(name, payload = {}) {
  const gameId = activeGameId.value;
  const roundId = String(payload.roundId || '').trim();
  if (!gameId || !roundId) return;
  const key = `${gameId}:${roundId}`;
  const recorded = name === 'game_started' ? reportRoundsStarted : reportRoundsCompleted;
  if (recorded.has(key) || (name === 'game_completed' && !reportRoundsStarted.has(key))) return;
  recorded.add(key);
  const mode = payload.mode === 'live' ? 'live' : 'singleplayer';
  if (name === 'game_started') reportRoundTimes.set(key, { gameId, roundId, mode, startedAt: Date.now() });
  const round = reportRoundTimes.get(key);
  recordFanEvent(name, { game_id: gameId, round_id: roundId, mode: round?.mode || mode,
    ...(name === 'game_completed' ? { duration_ms: Math.max(0, Date.now() - (round?.startedAt || Date.now())), score: payload.score, taps: payload.taps, success: payload.success } : {}),
    ...(payload.sponsorId ? { sponsor_id: payload.sponsorId } : {}),
  });
}
function onReportGameStarted(payload) { reportGameRound('game_started', payload); }
function onReportGameCompleted(payload) { reportGameRound('game_completed', payload); }
function reportPageAvailability() {
  return { vote_available: canVote.value, games_available: !matchConcluded.value, feedback_available: hasFeedbackSurvey.value, source: reportSource(window.location?.search || '') };
}
watch([canVote, matchConcluded, hasFeedbackSurvey], () => {
  if (!isHomeLoading.value) reportTelemetry?.record('page_view', reportPageAvailability());
});

async function bootstrap() {
  resetReportTelemetry();
  const revision = ++bootstrapRevision;
  const loadingStartedAt = Date.now();
  recordFanEvent('home_loading_started', { phase: 'loading' });
  assetController?.abort();
  isHomeLoading.value = true;
  if (!props.initialDataReady) return;
  assetController = new AbortController();
  const { signal } = assetController;
  if (loadedEventId !== walletScope()) {
    loadedEventId = walletScope();
    voteRequestRevision += 1;
    journeyRewardRevision += 1;
    homeVoteBonusRequest = null;
    isVoting.value = false;
    isLoadingVoteStatus.value = false;
    voteStatusError.value = '';
    liveStatusError.value = false;
    rawPlayers.value = [];
    homeMvpSelectedId.value = null;
    mvpEditing.value = false;
    highlightPlay.value = false;
    homeVoteReward.value = 0;
    analyticsNicknameRevision += 1;
    isSavingNickname.value = false;
    pendingVotePlayerId.value = null;
    voteBonusCoins.value = 0;
    isSavingCoins.value = false;
    walletRequestRevision += 1;
    leaderboardRequestRevision += 1;
    isLoadingWallet.value = false;
    redeeming.value = '';
    equipping.value = false;
    walletError.value = '';
    shopMessage.value = '';
    catalog.value = [];
    collection.value = emptyCollection();
    hasLoadedCollection.value = false;
    selectedCollectible.value = null;
    isCollectiblePreview.value = false;
    isCollectibleNew.value = false;
    walletMovements.value = [];
    participationPoints.value = 0;
    isRegistered.value = false;
    isRegistrationOpen.value = false;
    registrationOriginSheet.value = '';
    registrationTrigger.value = 'profile_overlay';
    registrationRewardLabel.value = '';
    registrationDestination.value = 'profile';
    fanNickname.value = '';
    fanLotteryTicket.value = null;
    nicknameDraft.value = '';
    leaderboardRows.value = [];
    leaderboardUser.value = null;
    liveEvent.value = null;
    isWalletReady.value = false;
    hasVoted.value = false;
    localVoted.value = null;
    votedPlayerId.value = null;
    activeGameId.value = null;
    isBasketballSaving.value = false;
    isPlaymakerSaving.value = false;
    isCrossoverSaving.value = false;
    isVolleyballSaving.value = false;
    closeSheets();
    coinError.value = '';
    voteError.value = '';
  }
  totalCoins.value = props.eventId ? readStoredCoins() : 0;
  loadDoneKeys();
  loadJourney();
  stopCoinsStream();
  await Promise.allSettled([
    organizationThemeRequest,
    loadQuizAvailability(),
    refreshLiveStatus(),
    // Restore saved rewards before recovering a pending vote bonus.
    loadFanProfile().catch(() => {}).then(() => loadVoteStatus()),
    loadPlayers(),
    refreshLeaderboard(),
    loadSponsors(signal),
    ensureCentoFonts(),
  ]);
  if (revision !== bootstrapRevision || signal.aborted) return;
  await nextTick();
  await preloadFanHomeImages([
    homeLogoUrl.value,
    ...players.value.map((player) => player.image),
    ...currentSponsorCatalog.value.map((sponsor) => sponsor.imageUrl),
    displayVotedImage.value,
    lotteryQrUrl.value,
    ...Object.values(FAN_GAME_ARTWORK),
  ], { signal });
  if (revision !== bootstrapRevision || signal.aborted) return;
  isHomeLoading.value = false;
  recordFanEvent('home_ready', { phase: 'ready', duration_ms: Math.max(0, Date.now() - loadingStartedAt) });
  reportTelemetry?.start(reportPageAvailability());
  startCoinsStream();
  loadWallet({ collectionOnly: true });
}

function ensureCentoFonts() {
  if (typeof document === 'undefined') return;
  const id = 'cento-fonts-collectibles';
  let link = document.getElementById(id);
  if (!link) {
    link = document.createElement('link');
    link.id = id;
    link.rel = 'stylesheet';
    link.href =
      'https://fonts.googleapis.com/css2?family=Barlow+Condensed:wght@700;800&family=Fraunces:opsz,wght@9..144,600;9..144,750;9..144,900&family=Inter:wght@400;500;600;700;800&display=swap';
    const loaded = new Promise((resolve) => {
      link.onload = resolve;
      link.onerror = resolve;
    });
    document.head.appendChild(link);
    link.centoFontsReady = loaded;
  }
  return Promise.resolve(link.centoFontsReady).then(() => document.fonts?.ready);
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
  bootstrap();
  liveStatusTimer = window.setInterval(refreshLiveStatus, 30000);
});

onBeforeUnmount(() => {
  if (activeGameId.value) recordGameExit('navigation');
  reportTelemetry?.stop();
  isDisposed = true;
  voteRequestRevision += 1;
  journeyRewardRevision += 1;
  cancelVotePresentation();
  walletRequestRevision += 1;
  leaderboardRequestRevision += 1;
  bootstrapRevision += 1;
  assetController?.abort();
  window.removeEventListener('keydown', onKeydown);
  stopCoinsStream();
  window.clearTimeout(coinAnimationTimer);
  window.clearTimeout(feedbackThanksTimer);
  window.clearInterval(liveStatusTimer);
});

watch(
  [() => props.eventId, () => props.initialDataReady, () => props.organizationSlug],
  () => bootstrap(),
);

// Il parent aggiorna le prop del giocatore votato dopo il voto reale.
watch(
  () => props.votedPlayerName,
  (name) => {
    if (name) hasVoted.value = true;
  },
);
</script>

<style scoped>
.cento-root {
  position: fixed;
  inset: 0;
  z-index: 1;
  display: flex;
  justify-content: center;
  overflow: hidden;
  padding-inline: env(safe-area-inset-left) env(safe-area-inset-right);
  background: var(--outer-bg, #f3e9e4);
  --bg: #fff7f2;
  --card: #ffffff;
  --ink: #23161a;
  --muted: #8c7a80;
  --red: #e4212e;
  --red-deep: #c0121f;
  --red-soft: #ffe8e6;
  --red-line: #ffd2ce;
  --gold: #f5b321;
  --gold-deep: #d18e00;
  --gold-soft: #fff3d6;
  --line: #f1e4e0;
  --shadow: 0 3px 0 rgba(35, 22, 26, 0.1);
  --shadow-red: 0 4px 0 var(--red-deep);
  color: var(--ink);
  font-family: 'Inter', system-ui, sans-serif;
}
.cento-root * {
  box-sizing: border-box;
}

.phone {
  --home-pad: 10px;
  width: 100%;
  max-width: 430px;
  height: 100dvh;
  background: var(--bg);
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.home-scroll {
  --home-gap: 7px;
  --home-radius: 16px;
  --home-shadow: 0 1px 2px rgba(35, 22, 26, 0.07);
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  scroll-padding-top: calc(68px + env(safe-area-inset-top));
  padding-bottom: env(safe-area-inset-bottom);
}
.home-scroll.has-sponsors { padding-bottom: 0; }

.topbar {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: calc(60px + env(safe-area-inset-top));
  padding: calc(7px + env(safe-area-inset-top)) var(--home-pad) 7px;
  border-bottom: 1px solid var(--line);
  background: var(--bg);
}
.club {
  display: flex;
  flex: 1 1 7.5rem;
  min-width: 0;
  align-items: center;
  gap: 8px;
}
.club-logo {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: var(--red);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 900;
  font-size: 14px;
  color: var(--on-primary);
  overflow: hidden;
}
.club-logo.has-logo {
  background: var(--card);
  border: 1px solid var(--line);
}
.club-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.club-name {
  min-width: 0;
  line-height: 1.2;
  overflow-wrap: anywhere;
}
.club-name b {
  font-size: .875rem;
  font-weight: 800;
  display: block;
  letter-spacing: -0.01em;
}
.club-name span {
  display: block;
  margin-top: 2px;
  font-size: .75rem;
  color: var(--muted);
  font-weight: 600;
}
.wallet {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--gold-soft);
  border: 1px solid var(--line);
  border-radius: 999px;
  min-width: 44px;
  min-height: 44px;
  padding: 5px 9px;
  font-weight: 800;
  font-size: .875rem;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  color: var(--ink);
  cursor: pointer;
}
.coin {
  width: 17px;
  height: 17px;
  border-radius: 50%;
  flex: none;
  background: radial-gradient(circle at 34% 30%, #ffe293, #f0a81c 68%, #c67f00);
  box-shadow: inset 0 -1.5px 2px rgba(0, 0, 0, 0.22), 0 0 0 2px #fff;
}
.top-right {
  display: flex;
  flex: none;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  max-width: 100%;
}
.merch-orders-link {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  border-radius: 10px;
  background: var(--red-soft);
  color: var(--red-text);
  font-weight: 800;
  text-decoration: none;
}
.merch-orders-link { justify-content: space-between; width: 100%; margin-bottom: 10px; padding-inline: 10px; }
.merch-orders-link:focus-visible { outline: 2px solid var(--red-text); outline-offset: 2px; }
.platform-info-button { display: grid; place-items: center; flex: none; width: 44px; height: 44px; padding: 3px; border: 0; border-radius: 50%; background: transparent; color: var(--muted); cursor: pointer; -webkit-tap-highlight-color: transparent; }
.platform-info-button svg { width: 22px; height: 22px; }
.platform-info-button:hover, .platform-info-button[aria-expanded=true] { color: var(--red-text); background: var(--red-soft); }
.platform-info-button:focus-visible, .platform-info-actions button:focus-visible, .platform-info-sheet .close:focus-visible { outline: 3px solid var(--red-text); outline-offset: 3px; }
.profile-btn {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  flex: none;
  background: transparent;
  border: 0;
  padding: 0;
  color: var(--red-text);
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: none;
  transition: transform 0.12s ease, border-color 0.15s ease;
}
.profile-btn:active {
  transform: translateY(2px);
  box-shadow: none;
}
.profile-btn:hover {
  border-color: var(--red);
}
.guest-avatar { display: grid; place-items: center; width: 36px; height: 36px; border: 1px solid var(--red-line); border-radius: 50%; background: var(--red-soft); color: var(--red-text); }
.guest-avatar svg { width: 24px; height: 24px; }

.home-dashboard {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--home-gap);
  padding: var(--home-gap) var(--home-pad) var(--home-pad);
  line-height: 1.35;
}
.mvp-card, .home-shortcut, .moment-card, .journey-card, .rank-link {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: var(--home-radius);
  box-shadow: var(--home-shadow);
}
.mvp-card {
  display: grid;
  min-width: 0;
  padding: 9px 10px 10px;
  border-top: 3px solid var(--red);
}
.mvp-content {
  min-width: 0;
  padding-top: 4px;
}
.mvp-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px 8px;
}
.mvp-card h1 {
  min-width: 0;
  margin: 0;
  font-size: 1.125rem;
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -.025em;
  overflow-wrap: anywhere;
}
.mvp-heading h1 { flex: 1 1 7.5em; }
.mvp-card.is-confirmed {
  border-top-color: #b58a3f;
  background: linear-gradient(110deg, var(--gold-soft), var(--card) 62%);
}
.mvp-receipt {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 52px;
}
.mvp-receipt.is-new { animation: mvp-receipt-in .24s ease-out both; }
.mvp-receipt-copy { flex: 1; min-width: 0; }
.mvp-receipt h1 { font-size: .9375rem; }
.mvp-receipt .mvp-receipt-reward {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--ink);
  font-weight: 800;
}
.mvp-receipt-reward .coin { width: 13px; height: 13px; }
.mvp-thumbnail {
  flex: none;
  width: 44px;
  aspect-ratio: .66;
}
.mvp-thumbnail.is-in-flight { visibility: hidden; }
.mvp-thumbnail :deep(.mvp-player-number) { padding: 1px; font-size: 10px; }
.mvp-thumbnail :deep(.mvp-player-number small),
.mvp-thumbnail :deep(.mvp-player-caption) { display: none; }
.mvp-thumbnail :deep(.mvp-player-initials) { font-size: .75rem; }
.mvp-check {
  display: grid;
  height: 100%;
  place-items: center;
  border: 1px solid #b58a3f;
  border-radius: 8px;
  color: var(--red-text);
  background: var(--card);
  font-size: 1.25rem;
  font-weight: 800;
}
.mvp-secondary {
  min-height: 44px;
  max-width: 38%;
  padding: 4px 7px;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--card);
  color: var(--red-text);
  font-family: inherit;
  font-size: .75rem;
  font-weight: 800;
  line-height: 1.25;
  cursor: pointer;
  overflow-wrap: anywhere;
}
.mvp-content > .mvp-secondary { margin-top: 4px; max-width: 100%; }
.mvp-secondary:disabled { opacity: .55; cursor: default; }
.mvp-status {
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--bg);
  color: var(--ink);
  font-size: .6875rem;
  font-weight: 800;
}
.mvp-status.is-open {
  background: var(--red-soft);
}
.mvp-card p {
  margin: 0;
  color: var(--muted);
  font-size: .8125rem;
  overflow-wrap: anywhere;
}
.mvp-card .mvp-error { margin: 2px 0 5px; color: var(--red-text); font-weight: 700; }
.mvp-action {
  width: 100%;
  min-height: 44px;
  padding: 6px 9px;
  border: 1px solid var(--line);
  border-radius: 11px;
  background: var(--red-soft);
  color: var(--ink);
  font: inherit;
  font-size: .875rem;
  font-weight: 800;
  cursor: pointer;
  overflow-wrap: anywhere;
}
.mvp-content > .mvp-action:first-of-type { margin-top: 3px; }
.mvp-action:disabled {
  background: var(--bg);
  color: var(--muted);
  cursor: default;
}
.mvp-collapse-leave-active { overflow: hidden; transition: height .24s ease, padding .24s ease, opacity .16s ease; }
.mvp-collapse-leave-to { height: 0 !important; padding-top: 0; opacity: 0; }
@keyframes mvp-receipt-in { from { opacity: .35; transform: translateY(4px); } }
.home-shortcuts {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(0, 1fr);
  gap: var(--home-gap);
}
.home-shortcut.is-vote-next {
  border-color: var(--red-text);
  animation: home-play-in .52s ease .22s both;
}
@keyframes home-play-in { 45% { box-shadow: 0 0 0 3px var(--red-line); } }
.home-shortcut, .moment-card, .rank-link {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 9px;
  padding: 10px;
  color: var(--ink);
  font: inherit;
  text-align: left;
  text-decoration: none;
  cursor: pointer;
}
.home-shortcut { min-height: 72px; flex-wrap: wrap; }
.shortcut-copy { flex: 1 1 4.5rem; }
.home-shortcut:hover, .moment-card:hover, .rank-link:hover { border-color: var(--red-text); }
.home-icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: var(--red-soft);
  color: var(--ink);
  font-size: 1.125rem;
}
.home-icon svg { width: 19px; height: 19px; }
.shortcut-copy, .moment-copy {
  min-width: 0;
  overflow-wrap: anywhere;
}
.shortcut-copy b, .moment-copy b {
  display: block;
  font-size: .875rem;
  font-weight: 800;
}
.shortcut-copy > span, .moment-copy > span {
  display: block;
  color: var(--muted);
  font-size: .75rem;
}
.moment-card { min-height: 76px; }
.moment-copy { flex: 1; }
.moment-action { display: flex; flex: none; flex-direction: column; align-items: flex-end; gap: 5px; }
.moment-action .row-pts { color: var(--ink); font-size: .8125rem; }
.moment-action > b { color: var(--red-text); font-size: .8125rem; }
.topbar button:focus-visible, .home-dashboard :is(button, a):focus-visible, .sponsor-chip:focus-visible {
  outline: 2px solid var(--red-text);
  outline-offset: 2px;
}
.btn-vote {
  border: none;
  border-radius: 13px;
  padding: 12px;
  width: 100%;
  text-align: center;
  background: var(--red);
  color: var(--on-primary);
  font-family: inherit;
  font-size: 13.5px;
  font-weight: 800;
  box-shadow: var(--shadow-red);
  letter-spacing: 0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Leaderboard tile */
.lb-tile {
  flex: none;
}
.lb-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.lb-head b {
  font-size: 14px;
  font-weight: 800;
}
.lb-mini {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-height: 0;
  overflow: hidden;
}
.lb-mini-row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 11.5px;
  font-weight: 700;
}
.lb-empty {
  font-size: 11px;
  color: var(--muted);
  font-weight: 500;
}
.rank {
  width: 20px;
  height: 20px;
  border-radius: 7px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9.5px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  background: var(--bg);
  color: var(--muted);
  border: 2px solid var(--line);
}
.rank.r1 {
  background: var(--gold);
  color: var(--on-secondary);
  border: none;
  transform: rotate(-5deg);
}
.rank.r2 {
  background: #d7dee8;
  color: #3a4557;
  border: none;
  transform: rotate(3deg);
}
.rank.r3 {
  background: #e8a25f;
  color: #59300b;
  border: none;
  transform: rotate(-3deg);
}
.leaderboard-rank {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  font-size: 14px;
}
.lb-mini-name {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.lb-mini-pts {
  color: var(--gold-deep);
  font-variant-numeric: tabular-nums;
  font-weight: 800;
  font-size: 11px;
}

/* Traguardo e sfida in evidenza */
.journey-card {
  min-height: 62px;
  padding: 10px 12px;
}
.journey-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.journey-head b { min-width: 0; font-size: .875rem; font-weight: 800; overflow-wrap: anywhere; }
.journey-head small { margin-inline-start: 6px; color: var(--muted); font-size: .75rem; font-weight: 600; }
.journey-head strong { display: flex; align-items: center; gap: 7px; flex: none; font-size: .875rem; font-variant-numeric: tabular-nums; }
.journey-bar { width: 100%; height: 8px; border-radius: 99px; background: var(--red-soft); margin-top: 10px; overflow: hidden; }
.journey-bar > span { display: block; height: 100%; background: var(--red); border-radius: inherit; transition: width 0.4s ease; }
.journey-next { color: var(--red-text); background: transparent; border: 0; padding: 6px 0; font: inherit; font-size: 12px; font-weight: 800; cursor: pointer; }
.journey-card .journey-next { min-height: 44px; margin-top: 4px; font-size: .8125rem; text-align: left; }
.journey-badge { border-color: var(--gold); }
.journey-badge .row-copy span { white-space: normal; }
.match-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
.match-stats > div { padding: 12px 7px; background: var(--card); border: 2px solid var(--line); border-radius: 14px; text-align: center; }
.match-stats b { display: block; font-family: 'Fraunces', serif; font-size: 24px; }
.match-stats span { display: block; font-size: 11px; color: var(--muted); }
.mvp-vote-invite { position: relative; isolation: isolate; display: flex; align-items: center; gap: 10px; flex: none; overflow: hidden; margin: 0 0 5px; border: 1px solid #ebce87; border-radius: 14px; padding: 11px 12px; background: linear-gradient(115deg, #fff7e2, #ffedbd); color: #74501b; }
.mvp-vote-invite::after { content: ''; position: absolute; z-index: -1; inset: -60%; background: linear-gradient(110deg, transparent 44%, #ffffffa6 49%, transparent 54%); pointer-events: none; opacity: 0; }
.mvp-vote-invite.is-active::after { opacity: 1; animation: mvp-invite-shine 1100ms ease 250ms both; }
.mvp-invite-star { font-size: 24px; line-height: 1; color: #c69430; transform: rotate(-12deg); }
.mvp-vote-invite > span:nth-child(2) { flex: 1; min-width: 0; }
.mvp-vote-invite b, .mvp-vote-invite small { display: block; }
.mvp-vote-invite b { font-size: 12px; font-weight: 800; }
.mvp-vote-invite small { margin-top: 3px; font-size: 9px; line-height: 1.4; }
.mvp-vote-invite strong { display: flex; flex-wrap: wrap; align-items: center; justify-content: center; gap: 4px; flex: 0 0 54px; font-size: 20px; font-weight: 900; }
.mvp-vote-invite strong small { flex-basis: 100%; margin: 0; font-size: 9px; font-weight: 600; text-align: center; }
.mvp-vote-invite .coin { width: 15px; height: 15px; }
@keyframes mvp-invite-shine { from { transform: translateX(-70%); } to { transform: translateX(70%); } }
.sheet.vote-result-sheet { max-height: calc(100dvh - max(12px, env(safe-area-inset-top))); padding-top: 8px; padding-bottom: calc(12px + env(safe-area-inset-bottom)); }
.vote-result-sheet .grabber { margin-bottom: 6px; height: 4px; }
.vote-result-sheet .sheet-head { margin-bottom: 4px; }
.vote-result { align-items: center; text-align: center; padding: 0 0 6px; gap: 8px; overscroll-behavior-y: contain; }
.vote-result > * { flex-shrink: 0; }
.vote-result-summary { display: grid; grid-template-columns: minmax(96px, .85fr) minmax(0, 1.15fr); align-items: center; gap: 10px; width: 100%; }
.vote-result-copy { display: flex; flex-direction: column; gap: 8px; min-width: 0; }
.vote-result h2 { margin: 0; font: 750 clamp(18px, 5vw, 22px)/1.15 'Fraunces', serif; overflow-wrap: anywhere; }
.vote-result > p { margin: 0; font-size: 13px; }
.vote-result-status { min-height: 34px; display: flex; align-items: center; justify-content: center; width: 100%; }
.vote-result-status > p { margin: 0; font-size: 11px; line-height: 1.4; color: var(--muted); }
.vote-result-bonus { position: relative; isolation: isolate; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; overflow: hidden; width: 100%; padding: 8px; border: 1px solid #eac56f; border-radius: 13px; background: linear-gradient(110deg, #fff8e4, #ffe8ac); color: #74440c; box-shadow: 0 3px 0 #d5a84026; animation: mvp-reward-pop 550ms cubic-bezier(.16, 1, .3, 1) both; }
.vote-result-bonus::after { content: ''; position: absolute; z-index: -1; inset: -50%; background: linear-gradient(110deg, transparent 42%, #ffffffb3 50%, transparent 58%); pointer-events: none; animation: mvp-invite-shine 850ms ease 150ms both; }
.vote-reward-amount { display: flex; align-items: center; justify-content: center; gap: 6px; flex: none; font-family: 'Fraunces', serif; font-size: 30px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
.vote-reward-amount .coin { width: 20px; height: 20px; box-shadow: inset 0 -2px 3px #0003, 0 0 0 2px #ffffff80; }
.vote-reward-copy { display: flex; flex-direction: column; gap: 3px; min-width: 0; text-align: center; }
.vote-reward-copy b { font-size: 11px; font-weight: 850; line-height: 1.3; }
.vote-reward-copy small { font-size: 9px; line-height: 1.35; }
.vote-result-avatar { width: 72px; height: 72px; border: 2px solid var(--red-line); border-radius: 20px; background: var(--red-soft); overflow: hidden; display: flex; align-items: center; justify-content: center; justify-self: center; font-size: 36px; flex: none; }
.is-mvp-result .vote-result-copy > h2 { animation: mvp-copy-in 400ms ease 300ms both; }
.vote-result-actions { display: flex; flex: none; flex-direction: column; gap: 4px; padding-top: 8px; border-top: 1px solid var(--line); background: var(--bg); }
.vote-result-actions > .btn-vote { flex: none; min-height: 44px; margin: 0; }
.vote-result-secondary { display: flex; align-items: center; justify-content: center; gap: 8px; }
.vote-result-secondary > .journey-next { flex: 1; min-width: 0; min-height: 32px; padding: 4px 2px; font-size: 11px; line-height: 1.3; }
.vote-result .sheet-empty.error { padding: 6px 8px; font-size: 11px; line-height: 1.35; }
@keyframes mvp-copy-in { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
@keyframes mvp-reward-pop { 0% { opacity: 0; transform: translateY(9px) scale(.78); } 65% { opacity: 1; transform: translateY(-2px) scale(1.035); } 100% { opacity: 1; transform: none; } }
.final-mvp-cards { display: flex; flex-wrap: wrap; justify-content: center; gap: 14px 10px; width: 100%; padding: 6px 4px 12px; }
.final-mvp-cards > .fan-player-card { width: min(112px, calc((100% - 20px) / 3)); flex: none; }
.vote-result-goal { width: 100%; padding: 11px 12px; border-radius: 14px; background: var(--card); border: 1px solid var(--line); font-size: 12px; text-align: left; }
.vote-profile-prompt { width: 100%; border: 1px solid var(--red-line); border-radius: 15px; padding: 10px 12px; background: linear-gradient(120deg, var(--card), var(--red-soft)); text-align: left; }
.vote-profile-eyebrow { display: block; color: var(--red-text); font-size: 8px; font-weight: 850; letter-spacing: .1em; }
.vote-profile-prompt h3 { margin: 4px 0 8px; color: var(--ink); font: 750 16px/1.25 'Fraunces', serif; }
.vote-profile-prompt ul { display: flex; flex-direction: column; gap: 6px; margin: 0; padding: 0; list-style: none; }
.vote-profile-prompt li { display: flex; align-items: flex-start; gap: 8px; color: var(--ink); font-size: 11px; line-height: 1.45; }
.vote-profile-prompt li > span { flex: none; font-size: 14px; line-height: 1.2; }
.vote-profile-prompt li strong { font-weight: 800; }
@media (max-height: 600px) {
  .vote-result-sheet .sheet-title { font-size: 17px; }
  .vote-result { gap: 6px; }
  .vote-result-copy { gap: 6px; }
  .vote-result h2 { font-size: 18px; }
  .vote-reward-amount { font-size: 26px; }
  .vote-profile-prompt { padding: 8px 10px; }
  .vote-profile-prompt h3 { margin-bottom: 6px; font-size: 15px; }
  .vote-profile-prompt ul { gap: 4px; }
  .vote-profile-prompt li { font-size: 10px; line-height: 1.35; }
  .vote-result-actions { padding-top: 6px; }
}
@media (max-height: 450px) {
  .sheet.vote-result-sheet { padding-top: 6px; padding-bottom: calc(8px + env(safe-area-inset-bottom)); }
  .vote-result-sheet .grabber, .vote-profile-eyebrow { display: none; }
  .vote-result h2 { font-size: 16px; }
  .vote-result-bonus { padding: 5px 7px; gap: 2px; }
  .vote-reward-amount { font-size: 23px; }
  .vote-reward-copy b { display: none; }
  .vote-reward-copy small { font-size: 8px; }
  .vote-profile-prompt { padding: 6px 8px; }
  .vote-profile-prompt h3 { margin: 0 0 4px; font-size: 14px; }
  .vote-profile-prompt ul { gap: 3px; }
  .vote-profile-prompt li > span { font-size: 12px; line-height: 1.1; }
  .vote-result-secondary > .journey-next { min-height: 28px; font-size: 10px; }
}
.vote-goal-heading { display: flex; align-items: center; gap: 9px; }
.vote-goal-medal { display: grid; place-items: center; flex: none; width: 31px; height: 35px; border-radius: 10px; background: var(--gold-soft); color: var(--gold-deep); font-size: 20px; }
.vote-goal-heading > span:nth-child(2) { flex: 1; min-width: 0; }
.vote-goal-heading b, .vote-goal-heading small { display: block; }
.vote-goal-heading > span small { margin-bottom: 3px; color: var(--muted); font-size: 8px; font-weight: 750; letter-spacing: .08em; }
.vote-goal-heading b { font-size: 11px; line-height: 1.4; }
.vote-goal-heading > strong { flex: none; color: var(--red-text); font-size: 21px; font-weight: 900; font-variant-numeric: tabular-nums; }
.vote-goal-heading > strong small { display: inline; margin-left: 1px; color: var(--muted); font-size: 11px; font-weight: 600; }
.vote-result-goal .journey-bar { height: 9px; }
.vote-result-goal .journey-bar > span { background: linear-gradient(90deg, var(--red), var(--gold)); }
.vote-result-goal .journey-bar > span.is-rewarded { animation: mvp-goal-fill 800ms cubic-bezier(.16, 1, .3, 1) both; }
.vote-result-goal.is-unlocked { border-color: var(--gold); background: linear-gradient(115deg, var(--card), var(--gold-soft)); }
.vote-result-goal.is-unlocked .vote-goal-medal { animation: mvp-medal-pop 600ms cubic-bezier(.16, 1, .3, 1) both; }
@keyframes mvp-goal-fill { from { width: var(--goal-from, 0%); } }
@keyframes mvp-medal-pop { 0% { transform: scale(.6) rotate(-20deg); } 65% { transform: scale(1.18) rotate(8deg); } 100% { transform: none; } }
.game-credit-error { padding: 10px 14px; margin: 0; color: var(--red-text); background: var(--card); font-size: 13px; font-weight: 700; }
.coin-flight-layer { position: fixed; inset: 0; z-index: 140; pointer-events: none; }
.coin-flight { position: absolute; animation: coin-fly 0.8s cubic-bezier(.35, .05, .65, 1) both; }
.wallet-pulse { animation: wallet-pop 0.45s cubic-bezier(.16, 1, .3, 1) 0.55s both; }
@keyframes coin-fly {
  0% { opacity: 0; transform: translate(0, 0) scale(.45) rotateY(0); }
  18% { opacity: 1; }
  35% { opacity: 1; transform: translate(var(--flight-bend, 0px), var(--flight-lift, -45px)) scale(1.16) rotateY(90deg); }
  85% { opacity: 1; }
  100% { opacity: 0; transform: translate(var(--flight-x), var(--flight-y)) scale(.5) rotateY(var(--flight-spin, 360deg)); }
}
@keyframes wallet-pop { 0%, 100% { transform: scale(1); } 35% { transform: scale(1.17) rotate(-3deg); box-shadow: 0 0 0 7px #f5b32126, 0 3px 0 #d18e0040; } 65% { transform: scale(.97) rotate(1deg); } }

.feedback-cta {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 5px;
  background: var(--red);
  border: 1px solid transparent;
  border-radius: var(--home-radius);
  padding: 8px 10px;
  color: var(--on-primary);
  text-align: left;
  box-shadow: var(--home-shadow);
}
.feedback-inline h2, .feedback-inline h3, .feedback-inline p { margin: 0; overflow-wrap: anywhere; }
.feedback-inline h2 { font-size: .9375rem; font-weight: 800; }
.feedback-inline h3 { font-size: .9375rem; font-weight: 600; line-height: 1.35; }
.feedback-emoji-options {
  display: grid;
  grid-template-columns: repeat(4, minmax(44px, 1fr));
  gap: 6px;
  margin-top: 3px;
}
.feedback-emoji-options button,
.feedback-answer-options button,
.feedback-inline-actions button {
  min-height: 44px;
  border: 2px solid transparent;
  border-radius: 10px;
  background: #fffaf2;
  color: #29221e;
  font: inherit;
  cursor: pointer;
}
.feedback-emoji-options button {
  display: grid;
  place-items: center;
  padding: 2px;
  font: 1.4rem/1 system-ui;
  transition: transform .12s ease, background-color .12s ease;
}
.feedback-inline button[aria-pressed='true'] { border-color: #29221e; background: #ffdf93; }
.feedback-inline button:disabled { cursor: wait; }
.feedback-emoji-options button:active { transform: scale(.95); }
.home-dashboard .feedback-inline button:focus-visible,
.home-dashboard .feedback-inline textarea:focus-visible { outline-color: var(--on-primary); }
.feedback-answer-options { display: grid; gap: 5px; }
.feedback-answer-options button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 9px;
  font-size: .875rem;
  text-align: left;
  overflow-wrap: anywhere;
}
.feedback-inline-actions { display: flex; flex-wrap: wrap; gap: 5px; }
.feedback-inline-actions button { padding: 4px 9px; font-size: .8125rem; font-weight: 700; }
.feedback-suggestion textarea {
  width: 100%;
  resize: vertical;
  padding: 7px 9px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: #fffaf2;
  color: #29221e;
  font: inherit;
}
.feedback-suggestion small { display: block; text-align: right; }
.feedback-error { font-size: .8125rem; font-weight: 700; }
.feedback-thanks { margin: 0; padding: 8px; color: var(--ink); font-size: .875rem; }
.home-slot-enter-active { transition: opacity .15s ease, transform .15s ease; }
.home-slot-leave-active { transition: opacity .1s ease, transform .1s ease; pointer-events: none; }
.home-slot-enter-from, .home-slot-leave-to { opacity: 0; transform: translateY(4px); }
@media (prefers-reduced-motion: reduce) {
  .mvp-receipt.is-new, .home-shortcut.is-vote-next { animation: none; }
  .mvp-collapse-leave-active, .feedback-emoji-options button { transition: none; }
  .feedback-emoji-options button:active { transform: none; }
}

.sponsors {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 6px var(--home-pad) calc(8px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--line);
  background: var(--bg);
}
.sponsors-head {
  margin: 0;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.12em;
  color: var(--muted);
}
.sponsor-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.sponsor-chip {
  min-width: 0;
  height: 44px;
  border-radius: 11px;
  background: var(--card);
  border: 2px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-size: 12.5px;
  font-weight: 750;
  color: var(--ink);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 8px;
  box-shadow: var(--shadow);
  transition: transform 0.12s ease, border-color 0.15s ease;
}
.sponsor-chip:active {
  transform: translateY(2px);
  box-shadow: none;
}
.sponsor-chip:hover {
  border-color: var(--red-line);
}
.sponsor-chip img {
  max-width: 100%;
  max-height: 30px;
  object-fit: contain;
}
.powered {
  flex: none;
  text-align: center;
  font-size: 9.5px;
  color: var(--muted);
  padding: 6px 0 8px;
  letter-spacing: 0.04em;
  font-weight: 600;
}
.powered b {
  color: var(--red-text);
  font-weight: 800;
}

/* Sheets */
.overlay {
  position: absolute;
  inset: 0;
  z-index: 100;
  background: rgba(35, 22, 26, 0.42);
  backdrop-filter: blur(3px);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.22s ease;
}
.overlay.open {
  opacity: 1;
  pointer-events: auto;
}
.sheet {
  visibility: hidden;
  pointer-events: none;
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 101;
  background: var(--bg);
  border-radius: 26px 26px 0 0;
  padding: 10px 18px calc(18px + env(safe-area-inset-bottom));
  max-height: 82%;
  display: flex;
  flex-direction: column;
  transform: translateY(105%);
  transition: transform 0.28s cubic-bezier(0.32, 0.72, 0.28, 1);
  box-shadow: 0 -8px 30px rgba(35, 22, 26, 0.18);
}
.sheet.open {
  visibility: visible;
  pointer-events: auto;
  transform: translateY(0);
}
.grabber {
  width: 38px;
  height: 5px;
  border-radius: 99px;
  background: var(--red-line);
  margin: 0 auto 12px;
  flex: none;
}
.sheet-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex: none;
}
.sheet-title {
  font-family: 'Fraunces', serif;
  font-size: 19px;
  font-weight: 750;
  letter-spacing: -0.01em;
}
.sheet-sub {
  font-size: 11.5px;
  color: var(--muted);
  margin-top: 2px;
  font-weight: 500;
}
.close {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--line);
  background: var(--card);
  color: var(--muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  font-family: inherit;
  font-weight: 700;
}
.sheet-body {
  overflow-y: auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding-bottom: 4px;
}
.sheet.earn-sheet { max-height: calc(100dvh - max(12px, env(safe-area-inset-top))); }
.platform-info-sheet { max-height: calc(100% - 12px); outline: none; }
.platform-info-sheet .sheet-head { align-items: flex-start; gap: 12px; }
.platform-info-sheet .sheet-title { margin: 6px 0 0; line-height: 1.2; color: var(--ink); }
.platform-info-kicker { font-size: 9px; font-weight: 800; letter-spacing: .12em; color: var(--red-text); }
.platform-info-body { gap: 0; overscroll-behavior: contain; }
.platform-info-intro { margin: 0 0 14px; font-size: 12px; line-height: 1.6; color: var(--muted); }
.platform-info-features { list-style: none; margin: 0; padding: 0; }
.platform-info-features li { display: flex; align-items: flex-start; gap: 11px; padding: 13px 0; border-top: 1px solid var(--line); }
.platform-info-icon { display: grid; place-items: center; flex: none; width: 35px; height: 35px; border: 1px solid var(--red-line); border-radius: 12px; background: var(--red-soft); font-size: 18px; }
.platform-info-features h3 { margin: 0 0 5px; color: var(--ink); font-size: 13px; font-weight: 800; line-height: 1.35; }
.platform-info-features p { margin: 0; color: var(--muted); font-size: 11.5px; line-height: 1.65; }
.platform-info-actions { flex: none; padding-top: 13px; }
.platform-info-actions .btn-vote { width: 100%; margin: 0; }
.mission-grid { display: grid; flex: none; grid-template-columns: repeat(3, minmax(0, 1fr)); align-items: stretch; gap: 8px; padding: 3px 3px 6px; }
.mission-card { position: relative; isolation: isolate; overflow: hidden; display: flex; flex-direction: column; align-items: center; gap: 5px; min-width: 0; border: 1px solid #28314d; border-radius: 16px; padding: 10px 7px 9px; background: #090c1e; color: #fff; font-family: inherit; text-align: center; cursor: pointer; box-shadow: 0 3px 0 #090c1e1a; transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease; -webkit-tap-highlight-color: transparent; touch-action: manipulation; }
.mission-card::after { content: ''; position: absolute; z-index: -1; inset: 0; background: linear-gradient(180deg, #090c1e00 0%, #090c1e33 25%, #090c1ef5 43%, #090c1e 100%); pointer-events: none; }
.mission-card-cover { flex: none; width: 100%; height: 38px; }
.mission-card-artwork { position: absolute; z-index: -2; inset: 0; width: 100%; height: 100%; object-fit: cover; object-position: center top; pointer-events: none; transition: transform 220ms ease; }
.mission-card-title { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; width: 100%; min-height: 2.4em; margin-top: 2px; font-size: 12px; font-weight: 800; line-height: 1.2; overflow-wrap: anywhere; text-shadow: 0 1px 3px #090c1e; }
.mission-card-description { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; width: 100%; min-height: 2.6em; color: #d8dceb; font-size: 10px; line-height: 1.3; overflow-wrap: anywhere; }
.mission-card-reward { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 3px; width: 100%; min-height: 22px; margin-top: auto; border-radius: 8px; padding: 3px 2px; background: var(--gold-soft); color: var(--gold-deep); font-size: 11px; font-weight: 850; line-height: 1.2; }
.mission-card-reward .coin { width: 11px; height: 11px; box-shadow: inset 0 -1px 2px #0003; }
.mission-card-reward small { min-width: 0; font-size: 8px; font-weight: 600; }
.mission-card.is-completed { border-color: #659d80; box-shadow: none; }
.mission-card.is-completed .mission-card-reward { background: #eaf5ee; color: #2c714b; }
.mission-card:disabled { cursor: not-allowed; box-shadow: none; }
.mission-card:disabled .mission-card-artwork { filter: grayscale(.9) brightness(.65); }
.mission-card:disabled .mission-card-reward { background: #e1e4ec; color: #485062; }
.mission-card:focus-visible { outline: 2px solid var(--red-text); outline-offset: 2px; }
.mission-card:active:not(:disabled) { transform: translateY(2px) scale(.97); box-shadow: none; }
.mission-grid.is-active .mission-card { animation: mission-card-in 280ms ease-out var(--mission-delay, 0ms) backwards; }
@media (hover: hover) { .mission-card:hover:not(:disabled) { border-color: #c0a06a; box-shadow: 0 5px 12px #090c1e26; } .mission-card:hover:not(:disabled) .mission-card-artwork { transform: scale(1.05); } }
@keyframes mission-card-in { from { opacity: 0; transform: translateY(7px); } to { opacity: 1; transform: none; } }
@media (max-width: 350px) { .mission-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-height: 650px) {
  .earn-sheet .grabber { margin-bottom: 8px; }
  .earn-sheet .sheet-head { margin-bottom: 8px; }
  .mission-grid { gap: 6px; }
  .mission-card { gap: 4px; padding: 8px 6px; border-radius: 13px; }
  .mission-card-cover { height: 28px; }
  .mission-card-title { margin-top: 0; font-size: 11px; }
  .mission-card-description { font-size: 9px; }
  .mission-card-reward { min-height: 20px; padding: 2px; }
}
@media (prefers-reduced-motion: reduce) { .mission-card, .mission-card-artwork { transition: none; } .mission-grid.is-active .mission-card { animation: none; } .mission-card:active:not(:disabled), .mission-card:hover:not(:disabled) .mission-card-artwork { transform: none; } }
.sheet-empty {
  padding: 18px 4px;
  text-align: center;
  font-size: 12.5px;
  color: var(--muted);
  font-weight: 500;
}
.sheet-empty.error {
  color: var(--red-text);
}
.row-item {
  display: flex;
  align-items: center;
  gap: 11px;
  background: var(--card);
  border: 2px solid var(--line);
  border-radius: 15px;
  padding: 11px 13px;
  font-family: inherit;
  color: var(--ink);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, transform 0.1s ease;
  width: 100%;
  box-shadow: var(--shadow);
}
.row-item:active {
  transform: translateY(2px);
  box-shadow: none;
}
.row-item:hover {
  border-color: var(--red-line);
}
.row-item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}
.leaderboard-list { display: flex; flex-direction: column; gap: 10px; margin: 0; padding: 0; list-style: none; }
.leaderboard-periods { display: flex; gap: 6px; padding: 4px; border-radius: 12px; background: var(--red-soft); }
.leaderboard-periods button { flex: 1; padding: 10px 8px; border: 0; border-radius: 9px; background: transparent; color: inherit; font: inherit; font-weight: 800; cursor: pointer; }
.leaderboard-periods button.active { background: var(--red); color: #fff; }
.leaderboard-periods button:focus-visible { outline: 2px solid var(--red); outline-offset: 2px; }
.leaderboard-period-description { margin: 10px 0 14px; }
.leaderboard-row { cursor: default; }
.leaderboard-row:active { transform: none; box-shadow: var(--shadow); }
.leaderboard-self { background: var(--red-soft); }
.avatar {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  flex: none;
  background: var(--red-soft);
  border: 2px solid var(--red-line);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 750;
  font-size: 14px;
  color: var(--red-text);
  transform: rotate(-3deg);
  overflow: hidden;
}
.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.row-copy {
  flex: 1;
  min-width: 0;
}
.row-copy b {
  font-size: 13.5px;
  font-weight: 800;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.row-copy span {
  font-size: 11px;
  color: var(--muted);
  font-weight: 500;
}
.row-pts {
  display: flex;
  align-items: center;
  gap: 5px;
  flex: none;
  font-weight: 800;
  font-size: 13px;
  color: var(--gold-deep);
  font-variant-numeric: tabular-nums;
}
.row-pts .coin {
  width: 14px;
  height: 14px;
}
.row-pts small { font-size: 9px; font-weight: 600; }
.chip-mint {
  flex: none;
  font-size: 11px;
  font-weight: 800;
  color: var(--on-primary);
  background: var(--red);
  padding: 6px 12px;
  border-radius: 999px;
  box-shadow: 0 2px 0 var(--red-deep);
}
.chip-mint:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  box-shadow: none;
}

.rank-link { min-height: 46px; padding: 8px 12px; }
.rank-link-icon, .rank-chevron { width: 19px; height: 19px; flex: none; color: var(--red-text); }
.rank-link b { flex: 1; min-width: 0; font-size: .875rem; font-weight: 800; overflow-wrap: anywhere; }
.earn-collection-link { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 44px; padding: 8px 12px; border: 1px solid var(--line); border-radius: 12px; background: var(--card); color: var(--red-text); font: inherit; font-size: .875rem; font-weight: 800; cursor: pointer; }
.wallet-total { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 10px; padding: 22px 14px; border: 1px solid var(--gold); border-radius: 18px; background: var(--gold-soft); }
.wallet-total strong { font-family: 'Fraunces', serif; font-size: 42px; line-height: 1; color: var(--gold-deep); }
.wallet-total > span:last-child { flex-basis: 100%; text-align: center; font-size: 11px; color: var(--ink); }
.wallet-total .coin { width: 25px; height: 25px; }
.wallet-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.wallet-actions .btn-vote { margin: 0; }
.section-label { margin: 8px 0 0; font-size: 12px; font-weight: 800; }
.wallet-movement { display: flex; align-items: center; justify-content: space-between; gap: 12px; border-bottom: 1px solid var(--line); padding: 10px 0; }
.wallet-movement > span { flex: 1; min-width: 0; }
.wallet-movement b, .wallet-movement small { display: block; }
.wallet-movement b { font-size: 12px; overflow-wrap: anywhere; }
.wallet-movement small { margin-top: 4px; color: var(--muted); font-size: 10px; }
.wallet-movement strong { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.wallet-movement .is-credit { color: var(--gold-deep); }
.shop-balance { display: flex; justify-content: space-between; align-items: center; gap: 12px; border-radius: 14px; padding: 12px; background: var(--gold-soft); font-size: 12px; }
.shop-balance b { display: flex; align-items: center; gap: 6px; }
.shop-gallery { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: 12px; align-items: stretch; margin-top: 3px; }
.shop-item { display: flex; flex-direction: column; min-width: 0; padding: 10px; border: 1px solid var(--line); border-radius: 18px; background: var(--card); box-shadow: 0 3px 0 var(--line); }
.shop-artwork { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; position: relative; width: 100%; padding: 0 0 25px; border: 0; border-radius: 12px; background: transparent; color: var(--muted); font: inherit; cursor: pointer; }
.shop-artwork.frame { flex: 1; min-height: 218px; background: radial-gradient(ellipse at 50% 40%,var(--red-soft),transparent 70%); }
.shop-artwork.frame > :deep(.fan-avatar) { --avatar-size: clamp(98px,31vw,146px) !important; }
.shop-artwork.match_card > :deep(.fan-match-card) { transform: rotate(2deg); transition: transform .2s ease; }
.shop-artwork.match_card:active > :deep(.fan-match-card) { transform: rotate(0deg) scale(.98); }
.shop-artwork-hint { position: absolute; display: flex; justify-content: space-between; gap: 4px; right: 1px; left: 1px; bottom: 5px; font-size: 8px; font-weight: 700; }
.shop-wear-preview { display: flex; align-items: center; gap: 5px; width: 100%; padding: 5px; border: 1px solid var(--line); border-radius: 12px; background: var(--bg); }
.shop-wear-preview b { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 9px; color: var(--ink); }
.shop-item-copy { display: flex; flex: 1; flex-direction: column; padding-top: 12px; }
.shop-item-copy > b { font-size: 13px; }
.shop-item-copy p { flex: 1; margin: 5px 0 12px; color: var(--muted); font-size: 10px; line-height: 1.5; }
.shop-item-copy > small { display: block; margin-top: 6px; font-size: 10px; color: var(--muted); }
.shop-buy { display: inline-flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 6px; border: 0; border-radius: 10px; padding: 10px 12px; background: var(--red); color: var(--on-primary); font: inherit; font-size: 11px; font-weight: 800; cursor: pointer; }
.shop-buy .coin { width: 13px; height: 13px; }
.shop-buy:disabled { opacity: .5; cursor: default; }
.shop-message { border-radius: 12px; padding: 11px; background: var(--gold-soft); color: var(--ink); font-size: 12px; line-height: 1.5; }
.collectible-sheet { max-height: 94dvh; }
.collectible-body { align-items: stretch; padding: 4px 4px 12px; }
.collectible-stage { position: relative; display: flex; flex-direction: column; align-items: center; width: min(100%,290px); margin: 10px auto 8px; isolation: isolate; }
.collectible-stage.is-frame { min-height: 270px; justify-content: center; border-radius: 28px; background: radial-gradient(ellipse at 50% 40%,var(--red-soft),transparent 65%); }
.collectible-stage.is-new { animation: collectible-reveal .7s cubic-bezier(.18,.75,.22,1) both; }
.collectible-stage.is-new::after { content: ''; position: absolute; z-index: 10; inset: 0; border-radius: 14px; background: linear-gradient(110deg,transparent 30%,#ffffffa6 48%,transparent 64%); pointer-events: none; animation: collectible-glint .7s ease-out both; }
.collectible-fan-name { max-width: 100%; margin-top: 12px; font-size: 22px; font-weight: 900; overflow-wrap: anywhere; text-align: center; }
.collectible-club-name { margin-top: 4px; color: var(--muted); font-size: 11px; }
.collectible-description { margin: 7px 0 12px; color: var(--muted); font-size: 11px; line-height: 1.5; text-align: center; }
.shop-artwork:focus-visible { outline: 3px solid var(--red-text); outline-offset: 4px; }
.leaderboard-row { gap: 8px; }
.leaderboard-row .row-copy b { font-size: 12px; }
@keyframes collectible-reveal { 0% { opacity: 0; transform: translateY(20px) scale(.88) rotate(-4deg); } 65% { opacity: 1; transform: translateY(-3px) scale(1.015) rotate(1deg); } 100% { opacity: 1; transform: none; } }
@keyframes collectible-glint { 0% { opacity: 0; transform: translateX(-60%); } 25% { opacity: .5; } 100% { opacity: 0; transform: translateX(60%); } }
@media (max-width: 350px) { .shop-gallery { gap: 8px; } .shop-item { padding: 7px; } .shop-item-copy > b { font-size: 12px; } .shop-artwork.frame { min-height: 190px; } .shop-wear-preview { padding: 2px; gap: 2px; } .shop-wear-preview b { font-size: 8px; } }
@media (prefers-reduced-motion: reduce) { .collectible-stage.is-new, .collectible-stage.is-new::after { animation: none; } .collectible-stage.is-new::after { display: none; } .shop-artwork.match_card > :deep(.fan-match-card) { transition: none; transform: none; } }
.earn-collection-link:focus-visible, .shop-buy:focus-visible { outline: 3px solid var(--red-text); outline-offset: 3px; }

/* Profilo */
.profile-economy { flex: none; padding-bottom: 12px; }
.profile-balances { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
.profile-balance { display: flex; flex-direction: column; align-items: flex-start; gap: 5px; min-width: 0; border: 1px solid var(--line); border-radius: 15px; padding: 12px; background: var(--card); color: var(--ink); font-family: inherit; text-align: left; cursor: pointer; }
.profile-balance-coins { border-color: var(--gold); background: var(--gold-soft); color: var(--gold-deep); }
.profile-balance-points { border-color: var(--red-line); background: var(--red-soft); }
.profile-balance-label { display: flex; align-items: center; gap: 6px; width: 100%; font-size: 11px; font-weight: 800; }
.profile-balance-label .coin { width: 13px; height: 13px; box-shadow: inset 0 -1px 2px #0003; }
.balance-arrow { margin-left: auto; font-size: 14px; opacity: .65; }
.points-mark { color: var(--red-text); font-size: 16px; line-height: 1; }
.profile-balance strong { max-width: 100%; font-family: 'Fraunces', serif; font-size: 30px; font-weight: 800; line-height: 1.1; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.profile-balance small { font-size: 10px; font-weight: 500; line-height: 1.4; }
.profile-economy-hint { margin: 8px 2px 0; color: var(--muted); font-size: 10px; line-height: 1.5; }
.profile-body { gap: 10px; overscroll-behavior-y: contain; }
.profile-body > * { flex-shrink: 0; }
.profile-edit { margin-left: auto; min-height: 32px; padding: 4px 0 4px 8px; border: 0; background: transparent; color: var(--red-text); font: inherit; font-size: 10px; font-weight: 750; cursor: pointer; }
.profile-signin { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex: none; width: 100%; min-height: 62px; padding: 12px 14px; border: 0; border-radius: 13px; background: var(--red); color: var(--on-primary, #fff); font-family: inherit; text-align: left; cursor: pointer; }
.profile-signin > span:first-child { min-width: 0; }
.profile-signin strong, .profile-signin small { display: block; overflow-wrap: anywhere; }
.profile-signin strong { font-size: 13px; font-weight: 800; line-height: 1.4; }
.profile-signin small { margin-top: 3px; font-size: 10px; font-weight: 500; line-height: 1.4; }
.profile-signin > span:last-child { flex: none; font-size: 20px; }
.profile-balance:focus-visible, .profile-edit:focus-visible, .profile-signin:focus-visible { outline: 2px solid var(--red-text); outline-offset: 3px; }
.nick-editor {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}
.nick-field { display: flex; flex: 1; flex-direction: column; gap: 5px; min-width: 0; color: var(--muted); font-size: 11px; font-weight: 600; }
.nick-input {
  flex: 1;
  min-width: 0;
  border-radius: 13px;
  border: 2px solid var(--line);
  background: var(--card);
  padding: 11px 13px;
  font-family: inherit;
  font-size: 13.5px;
  font-weight: 700;
  color: var(--ink);
  outline: none;
  transition: border-color 0.15s ease;
}
.nick-input:focus {
  border-color: var(--red-line);
}
.nick-save {
  border: none;
  cursor: pointer;
  font-family: inherit;
}
.nick-msg {
  margin: 0;
  font-size: 11.5px;
  font-weight: 700;
  padding: 0 2px;
}
.nick-msg.error {
  color: var(--red-text);
}
.nick-msg.ok {
  color: var(--gold-deep);
}
.lottery-shortcut { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px 12px; width: 100%; min-height: 44px; flex: none; border: 1px solid var(--gold); border-radius: 12px; padding: 10px 12px; background: var(--gold-soft); color: var(--ink); font: inherit; font-size: 12px; text-align: left; cursor: pointer; }
.lottery-shortcut strong { margin-left: 4px; font-size: 15px; font-weight: 900; font-variant-numeric: tabular-nums; letter-spacing: .08em; }
.lottery-shortcut-action { display: flex; align-items: center; gap: 6px; color: var(--gold-deep); font-weight: 800; white-space: nowrap; }
.lottery-shortcut:focus-visible, .lottery-box:focus-visible { outline: 3px solid var(--red-text); outline-offset: 3px; }
.lottery-box {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 13px;
  padding: 0 12px 12px;
}
.lottery-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 48px;
  font-size: 12px;
  font-weight: 750;
  color: var(--ink);
}
.lottery-code {
  margin-bottom: 8px;
  font-family: 'Fraunces', serif;
  font-size: 18px;
  font-weight: 900;
  letter-spacing: 0.06em;
  color: var(--red-text);
}
.lottery-qr {
  display: block;
  margin: 0 auto;
  width: 168px;
  height: 168px;
  border-radius: 14px;
  border: 2px solid var(--line);
  padding: 8px;
  align-self: center;
  background: #fff;
}
.lottery-hint {
  font-size: 11px;
  font-weight: 700;
  color: var(--gold-deep);
  background: var(--gold-soft);
  border-radius: 10px;
  padding: 8px 11px;
}

/* Overlay gioco a schermo pieno */
.game-overlay {
  position: absolute;
  inset: 0;
  z-index: 150;
  background: var(--bg);
  display: flex;
  flex-direction: column;
}
.game-head {
  flex: none;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 2px solid var(--line);
}
.game-title {
  flex: 1;
  min-width: 0;
  text-align: center;
  font-family: 'Fraunces', serif;
  font-size: 15px;
  font-weight: 750;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.game-stage {
  flex: 1;
  min-height: 0;
  display: flex;
  padding: 12px;
  overflow: hidden;
}
.game-fill {
  width: 100%;
  height: 100%;
}

@media (prefers-reduced-motion: reduce) {
  .is-mvp-result .vote-result-copy > h2,
  .vote-result-bonus,
  .vote-result-goal .journey-bar > span.is-rewarded,
  .vote-result-goal.is-unlocked .vote-goal-medal { animation: none; }
  .mvp-vote-invite::after,
  .vote-result-bonus::after { display: none; }
  .coin-flight { display: none; }
  .wallet-pulse { animation: none; }
  .journey-bar > span { transition: none; }
  .home-slot-enter-from, .home-slot-leave-to { opacity: 1; transform: none; }
  .sheet,
  .overlay,
  .home-slot-enter-active,
  .home-slot-leave-active,
  .mvp-player,
  .row-item {
    transition: none;
  }
}
</style>

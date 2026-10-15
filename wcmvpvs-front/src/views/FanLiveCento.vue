<template>
  <div class="cento-root" :class="{ 'reduce-motion': false }" :style="themeStyle">
    <div class="phone" :inert="isHomeLoading" :aria-hidden="isHomeLoading || undefined" :aria-busy="isHomeLoading">
      <!-- Top bar -->
      <header class="topbar">
        <div class="club">
          <div class="club-logo" :class="{ 'has-logo': homeLogoUrl }">
            <img v-if="homeLogoUrl" :src="homeLogoUrl" :alt="homeName" />
            <template v-else>{{ homeBadge }}</template>
          </div>
          <div class="club-name">
            <b>{{ homeName }}</b>
            <span>{{ opponentLabel }}</span>
          </div>
        </div>
        <div class="top-right">
          <button ref="walletEl" class="wallet" :class="{ 'wallet-pulse': walletPulse }" type="button" aria-label="Saldo monete" @click="openSheet('earn')">
            <span class="coin"></span> <span>{{ totalCoins }}</span>
          </button>
          <button class="profile-btn" type="button" aria-label="Profilo" @click="openProfile">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
              <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
            </svg>
          </button>
        </div>
      </header>

      <main>
        <!-- Grid -->
        <div class="grid">
          <!-- MVP: sheet inline con giocatori reali + voto reale -->
          <button class="tile tile-mvp" type="button" :disabled="!eventId" @click="showMatchSummary ? openSheet('match-summary') : openSheet('mvp')">
            <b>{{ hasVoted ? 'Il tuo MVP' : "Vota l'MVP del pubblico" }}</b>
            <p>{{ hasVoted ? 'Grazie per il voto!' : canVote ? 'Il tuo voto vale +10 monete' : 'In attesa della prossima votazione' }}</p>
            <div class="mvp-photo" :class="{ 'has-player': hasVoted }">
              <FanPlayerCard v-if="hasVoted" class="mvp-home-card" :player="votedPlayerCard" :club-logo-url="homeLogoUrl" selected selected-label="Voto registrato" loading="eager" />
              <template v-else>
                <span class="status">{{ canVote ? 'Votazioni aperte' : 'Votazioni chiuse' }}</span>
                <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--red-text)" stroke-width="1.5">
                  <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
                </svg>
              </template>
            </div>
            <span class="btn-vote">{{ showMatchSummary ? 'Vedi il riepilogo' : hasVoted ? votedButtonLabel : canVote ? 'Vota ora' : 'Votazioni chiuse' }}</span>
          </button>

          <button class="tile" type="button" @click="openSheet('earn')">
            <div class="tile-icon">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 4 14h6l-1 8 9-12h-6l1-8z" /></svg>
            </div>
            <b>Gioca e guadagna</b>
            <p>Quiz e sfide sponsor</p>
            <div class="spacer"></div>
            <span class="tile-cta">Gioca →</span>
          </button>

          <button class="tile" type="button" @click="openSheet('spend')">
            <div class="tile-icon">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 12v8H4v-8M2 7h20v5H2zM12 7v13" /></svg>
            </div>
            <b>Scopri i premi</b>
            <p>Premi e merch del club</p>
            <div class="spacer"></div>
            <span class="tile-cta">Riscatta →</span>
          </button>
        </div>

        <section v-if="eventId" class="journey-card" aria-label="Il tuo primo traguardo">
          <div class="journey-head">
            <div>
              <b>{{ progress.unlocked ? `Badge ${FAN_BADGE_NAME} sbloccato!` : 'Il tuo primo traguardo' }}</b>
              <span>{{ progress.unlocked ? 'Il badge è nel tuo profilo' : `Mancano ${progress.remaining} monete al badge ${FAN_BADGE_NAME}` }}</span>
            </div>
            <strong>{{ progress.current }}/{{ FIRST_FAN_GOAL }} <span class="coin" aria-hidden="true"></span></strong>
          </div>
          <div class="journey-bar" role="progressbar" aria-label="Monete guadagnate per il primo badge" :aria-valuenow="progress.current" :aria-valuemin="0" :aria-valuemax="FIRST_FAN_GOAL">
            <span :style="{ width: `${progress.percent}%` }"></span>
          </div>
          <div class="journey-footer">
            <small>Vota e gioca · le monete restano nel wallet</small>
            <button type="button" class="journey-next" @click="openJourneyNext">{{ journeyNextLabel }}</button>
          </div>
          <button v-if="journey.pendingVoteBonus" class="journey-next" type="button" :disabled="isSavingCoins" @click="retryVoteBonus">{{ isSavingCoins ? 'Salvataggio…' : 'Riprova a salvare il bonus voto' }}</button>
        </section>

        <button v-if="featuredMission" class="moment-card" type="button" @click="onMission(featuredMission)">
          <span class="avatar" aria-hidden="true">{{ featuredMission.emoji }}</span>
          <span class="row-copy"><span>Sfida del momento</span><b>{{ featuredMission.title }}</b><span>{{ featuredMission.repeatable ? featuredMission.subtitle : nextQuizLabel || 'Una breve sfida da giocare durante le pause' }}</span></span>
          <span class="row-pts">+{{ featuredMission.pts }} <span class="coin" aria-hidden="true"></span></span>
        </button>
        <button v-if="showMatchSummary" class="moment-card" type="button" @click="openSheet('match-summary')">
          <span class="avatar" aria-hidden="true">🏀</span>
          <span class="row-copy"><b>La tua partita</b><span>{{ matchConcluded ? 'Risultato MVP e riepilogo della serata' : 'Votazioni chiuse · guarda il tuo riepilogo' }}</span></span>
          <span class="tile-cta">Apri →</span>
        </button>

        <!-- Leaderboard tile -->
        <button class="tile lb-tile" type="button" @click="openLeaderboard">
          <div class="lb-head">
            <b>Classifica tifosi</b>
            <span class="tile-cta">Vedi tutti →</span>
          </div>
          <div class="lb-mini">
            <div v-for="(row, i) in leaderboardTop3" :key="i" class="lb-mini-row">
              <span class="rank" :class="`r${i + 1}`">{{ i + 1 }}</span>
              <span class="lb-mini-name">{{ row.name }}</span>
              <span class="lb-mini-pts">{{ row.coins }}</span>
            </div>
            <div v-if="!leaderboardTop3.length" class="lb-empty">Gioca e vota per popolare la classifica</div>
          </div>
        </button>

        <!-- Feedback dell'evento: domande e suggerimento aperto finale -->
        <button class="feedback-cta" type="button" :disabled="!hasFeedbackSurvey || hasSubmittedFeedback" @click="openFeedbackModal">
          <span class="feedback-cta-icon" aria-hidden="true">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.5 8.5 0 0 1-8.5 8.5H5l-4 3V11.5A8.5 8.5 0 0 1 9.5 3h3a8.5 8.5 0 0 1 8.5 8.5Z" /><path d="M7 9h8M7 13h5" /></svg>
          </span>
          <span class="feedback-cta-copy">
            <b>{{ hasSubmittedFeedback ? 'Feedback inviato' : 'Lascia un feedback' }}</b>
            <span>{{ feedbackSubtitle }}</span>
          </span>
          <span class="feedback-cta-action">{{ hasSubmittedFeedback ? 'Fatto ✓' : 'Rispondi →' }}</span>
        </button>

        <!-- Sponsors (spazio ampliato al posto dello scoreboard) -->
        <section v-if="sponsors.length" class="sponsors">
          <div class="sponsors-head">
            <span class="label">PARTNER UFFICIALI</span>
          </div>
          <div class="sponsor-grid">
            <button
              v-for="sponsor in sponsors.slice(0, 6)"
              :key="sponsor.id"
              class="sponsor-chip"
              type="button"
              @click="onSponsorClick(sponsor)"
            >
              <img v-if="sponsor.imageUrl" :src="sponsor.imageUrl" :alt="sponsor.name || 'Sponsor'" />
              <span v-else>{{ sponsor.name }}</span>
            </button>
          </div>
        </section>
      </main>

      <!-- Overlay condiviso dai pannelli della home -->
      <div class="overlay" :class="{ open: activeSheet }" @click="closeSheets"></div>

      <!-- MVP: lista giocatori reale -->
      <div class="sheet" :class="{ open: activeSheet === 'mvp' }" role="dialog" aria-label="Vota MVP">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Vota l'MVP del pubblico</div>
            <div class="sheet-sub">Tocca la card del tuo MVP · +10 monete al primo voto</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div v-if="isLoadingPlayers" class="sheet-empty">Caricamento giocatori…</div>
          <div v-else-if="playersError" class="sheet-empty error">{{ playersError }}</div>
          <div v-else-if="!players.length" class="sheet-empty">Nessun giocatore disponibile.</div>
          <p v-if="voteError" class="sheet-empty error" role="alert">{{ voteError }}</p>
          <div v-if="players.length && !isLoadingPlayers && !playersError" class="mvp-player-grid" :aria-busy="isVoting">
            <button
              v-for="player in players"
              :key="player.id"
              class="mvp-player"
              :class="{ 'is-selected': votedPlayerId === player.id }"
              type="button"
              :aria-label="`Vota ${player.fullName}${player.number ? `, maglia ${player.number}` : ''}`"
              :aria-pressed="votedPlayerId === player.id"
              :disabled="isVoting || !canVote"
              @click="castVote(player)"
            >
              <FanPlayerCard :player="player" :club-logo-url="homeLogoUrl" :selected="votedPlayerId === player.id" />
            </button>
          </div>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'vote-result' }" role="dialog" :aria-label="celebrationKind === 'vote' ? 'Conferma del voto' : 'Monete guadagnate'" :aria-hidden="activeSheet !== 'vote-result'" :inert="activeSheet !== 'vote-result'">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div class="sheet-title">{{ celebrationKind === 'vote' ? 'Il tuo voto conta!' : 'Sfida completata!' }}</div>
          <button class="close" type="button" aria-label="Chiudi conferma voto" @click="closeSheets">✕</button>
        </div>
        <div ref="voteResultEl" class="sheet-body vote-result">
          <FanPlayerCard v-if="celebrationKind === 'vote'" class="vote-result-player" :player="votedPlayerCard" :club-logo-url="homeLogoUrl" selected loading="eager" />
          <span v-else class="vote-result-avatar" aria-hidden="true">{{ progress.unlocked ? '🏅' : '🎉' }}</span>
          <h2>{{ celebrationKind === 'vote' ? `Hai votato ${displayVotedName || 'il tuo MVP'}!` : `${celebrationName} completato!` }}</h2>
          <p v-if="voteBonusCoins" class="vote-result-bonus">+{{ voteBonusCoins }} <span class="coin" aria-hidden="true"></span> monete salvate</p>
          <p v-else-if="journey.pendingVoteBonus">Voto registrato. Il bonus di {{ VOTE_REWARD }} monete è ancora da salvare.</p>
          <p v-else>Voto aggiornato. Il bonus del primo voto è già stato assegnato.</p>
          <div class="vote-result-goal">
            <b>{{ progress.unlocked ? `Hai sbloccato il badge ${FAN_BADGE_NAME}!` : `${progress.current}/${FIRST_FAN_GOAL} · Mancano ${progress.remaining} monete al primo badge` }}</b>
            <div class="journey-bar"><span :style="{ width: `${progress.percent}%` }"></span></div>
          </div>
          <p v-if="coinError" class="sheet-empty error" role="alert">{{ coinError }}</p>
          <button v-if="journey.pendingVoteBonus" class="btn-vote" type="button" :disabled="isSavingCoins" @click="retryVoteBonus">{{ isSavingCoins ? 'Salvataggio…' : 'Riprova a salvare le monete' }}</button>
          <button v-else class="btn-vote" type="button" @click="openJourneyNext">{{ journeyNextLabel }}</button>
          <button class="journey-next" type="button" @click="closeSheets">Torna alla partita</button>
        </div>
      </div>

      <!-- Guadagna: missioni del match -->
      <div class="sheet" :class="{ open: activeSheet === 'earn' }" role="dialog" aria-label="Guadagna monete">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Guadagna monete</div>
            <div class="sheet-sub">Completa le missioni del match</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <button
            v-for="mission in earnMissions"
            :key="mission.key"
            class="row-item"
            type="button"
            :disabled="matchConcluded || (mission.key === 'quiz' && !quizAvailable)"
            @click="onMission(mission)"
          >
            <span class="avatar">{{ mission.emoji }}</span>
            <span class="row-copy"><b>{{ mission.title }}</b><span>{{ !mission.repeatable && doneKeys.includes(mission.key) ? 'Bonus già ricevuto per questa partita' : mission.subtitle }}</span></span>
            <span class="row-pts">
              <template v-if="mission.repeatable">+1 <span class="coin" aria-hidden="true"></span><small>/ canestro</small></template>
              <template v-else-if="doneKeys.includes(mission.key) || (mission.key === 'vote' && hasVoted)">✓</template>
              <template v-else-if="mission.key === 'quiz' && !quizAvailable">Non disponibile</template>
              <template v-else>+{{ mission.pts }} <span class="coin"></span></template>
            </span>
          </button>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'spend' }" role="dialog" aria-label="Spendi monete">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Spendi le tue monete</div>
            <div class="sheet-sub">Premi offerti dal club e dagli sponsor</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div class="row-item leaderboard-row journey-badge">
            <span class="avatar" aria-hidden="true">🏅</span>
            <span class="row-copy"><b>Badge {{ FAN_BADGE_NAME }}</b><span>{{ progress.unlocked ? 'Sbloccato! Lo trovi nel tuo profilo.' : `Guadagna ${FIRST_FAN_GOAL} monete in questa partita, votando o giocando, per sbloccarlo.` }}</span></span>
          </div>
          <button
            v-for="reward in rewards"
            :key="reward.key"
            class="row-item"
            type="button"
            :disabled="redeeming === reward.key || totalCoins < reward.cost"
            @click="redeem(reward)"
          >
            <span class="avatar">{{ reward.emoji }}</span>
            <span class="row-copy"><b>{{ reward.title }}</b><span>{{ reward.subtitle }}</span></span>
            <span class="row-pts">{{ reward.cost }} <span class="coin"></span></span>
          </button>
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
            <div class="sheet-sub">I tifosi con più monete</div>
          </div>
          <button class="close" type="button" aria-label="Chiudi classifica tifosi" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div v-if="isLoadingLeaderboard && !leaderboardTop3.length" class="sheet-empty">Caricamento classifica…</div>
          <div v-else-if="leaderboardError" class="sheet-empty error" role="status">{{ leaderboardError }}</div>
          <div v-else-if="!leaderboardTop3.length" class="sheet-empty">
            {{ eventId ? 'Gioca e vota per popolare la classifica.' : 'La classifica sarà disponibile con un evento attivo.' }}
          </div>
          <div v-for="(row, index) in leaderboardTop3" :key="index" class="row-item leaderboard-row">
            <span class="rank leaderboard-rank" :class="`r${index + 1}`">{{ index + 1 }}</span>
            <span class="row-copy"><b>{{ row.name }}</b><span>{{ index + 1 }}° posto</span></span>
            <span class="row-pts">{{ row.coins }} <span class="coin" aria-hidden="true"></span><span class="sr-only">monete</span></span>
          </div>
          <div v-if="leaderboardUser" class="row-item leaderboard-row leaderboard-self">
            <span class="avatar">{{ leaderboardUser.rank }}</span>
            <span class="row-copy"><b>La tua posizione</b><span>{{ profileNickname }}</span></span>
            <span class="row-pts">{{ leaderboardUser.coins }} <span class="coin" aria-hidden="true"></span><span class="sr-only">monete</span></span>
          </div>
          <template v-if="leaderboardNearby.length">
            <div class="sheet-sub">Vicino a te · classifica della società</div>
            <p v-if="rankGapLabel" class="rank-gap">{{ rankGapLabel }}</p>
            <div v-for="row in leaderboardNearby" :key="row.rank" class="row-item leaderboard-row" :class="{ 'leaderboard-self': row.is_me }">
              <span class="rank leaderboard-rank">{{ row.rank }}</span>
              <span class="row-copy"><b>{{ row.is_me ? 'Tu' : row.name }}</b></span>
              <span class="row-pts">{{ row.coins }} <span class="coin" aria-hidden="true"></span></span>
            </div>
          </template>
          <button v-else-if="!isRegistered" class="btn-vote" type="button" @click="openRegistration">Entra in classifica con il tuo nickname</button>
        </div>
      </div>

      <div class="sheet" :class="{ open: activeSheet === 'match-summary' }" role="dialog" aria-label="La tua partita" :aria-hidden="activeSheet !== 'match-summary'" :inert="activeSheet !== 'match-summary'">
        <div class="grabber"></div>
        <div class="sheet-head"><div><div class="sheet-title">La tua partita</div><div class="sheet-sub">{{ homeName }} · {{ opponentLabel }}</div></div><button class="close" type="button" aria-label="Chiudi riepilogo" @click="closeSheets">✕</button></div>
        <div class="sheet-body">
          <div class="match-stats"><div><b>{{ hasVoted ? '✓' : '—' }}</b><span>Voto MVP</span></div><div><b>{{ completedGames }}</b><span>Sfide completate</span></div><div><b>{{ progress.earned }}</b><span>Monete guadagnate</span></div></div>
          <div class="row-item leaderboard-row"><span class="avatar" aria-hidden="true">🏆</span><span class="row-copy"><b>{{ finalMvpLabel }}</b><span>{{ matchConcluded ? finalMvpSubtitle : 'Il risultato finale sarà disponibile quando la partita sarà conclusa.' }}</span></span></div>
          <div v-if="finalMvpPlayers.length" class="final-mvp-cards">
            <FanPlayerCard v-for="player in finalMvpPlayers" :key="player.id" :player="player" :club-logo-url="homeLogoUrl" />
          </div>
          <div v-if="progress.unlocked" class="row-item leaderboard-row journey-badge"><span class="avatar" aria-hidden="true">🏅</span><span class="row-copy"><b>{{ FAN_BADGE_NAME }}</b><span>Il tuo badge digitale della partita</span></span></div>
          <p class="sheet-sub">{{ leaderboardUser ? `La tua posizione nella società: #${leaderboardUser.rank}` : 'Grazie per aver partecipato alla serata.' }}</p>
          <button v-if="hasFeedbackSurvey && !hasSubmittedFeedback" class="btn-vote" type="button" @click="openFeedbackModal">Com’è andata? Lascia il tuo feedback</button>
          <p v-else-if="hasSubmittedFeedback" class="sheet-sub">Feedback ricevuto, grazie!</p>
        </div>
      </div>

      <!-- Profilo (come LiveExperienceHome: nickname + saldo + QR lotteria) -->
      <div class="sheet" :class="{ open: activeSheet === 'profile' }" role="dialog" aria-label="Profilo">
        <div class="grabber"></div>
        <div class="sheet-head">
          <div>
            <div class="sheet-title">Il tuo profilo</div>
            <div class="sheet-sub">{{ isRegistered ? 'Account tifoso' : 'Ospite · non registrato' }}</div>
          </div>
          <button class="close" type="button" @click="closeSheets">✕</button>
        </div>
        <div class="sheet-body">
          <div class="profile-hero">
            <div class="profile-avatar">
              <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="var(--red-text)" stroke-width="1.8" stroke-linecap="round">
                <circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 3.6-7 8-7s8 3 8 7" />
              </svg>
            </div>
            <div>
              <div class="profile-label">Nickname</div>
              <div class="profile-nick">{{ profileNickname }}</div>
            </div>
          </div>

          <button class="row-item profile-toggle" type="button" @click="toggleNicknameEditor">
            <b>Modifica nickname</b>
          </button>
          <div v-if="isNicknameEditorOpen" class="nick-editor">
            <input
              v-model.trim="nicknameDraft"
              type="text"
              minlength="3"
              maxlength="24"
              placeholder="Inserisci nickname"
              class="nick-input"
              @keyup.enter="submitNickname"
            />
            <button
              class="chip-mint nick-save"
              type="button"
              :disabled="isSavingNickname || !canSubmitNickname"
              @click="submitNickname"
            >
              {{ isSavingNickname ? 'Salvataggio…' : 'Aggiorna' }}
            </button>
          </div>
          <p v-if="nicknameError" class="nick-msg error">{{ nicknameError }}</p>
          <p v-else-if="nicknameSuccess" class="nick-msg ok">{{ nicknameSuccess }}</p>

          <div class="coin-balance">
            <span class="coin-balance-label">Saldo monete</span>
            <span class="coin-balance-val">{{ totalCoins }} <span class="coin"></span></span>
          </div>
          <button v-if="!isRegistered" class="btn-vote" type="button" @click="openRegistration">Accedi o crea il tuo profilo tifoso</button>
          <div v-if="progress.unlocked" class="row-item leaderboard-row journey-badge">
            <span class="avatar" aria-hidden="true">🏅</span>
            <span class="row-copy"><b>{{ FAN_BADGE_NAME }}</b><span>Badge della partita · {{ FIRST_FAN_GOAL }} monete guadagnate</span></span>
          </div>

          <div class="lottery-box">
            <div class="lottery-title">QR Lotteria MVP</div>
            <template v-if="lotteryCode">
              <div class="lottery-code">{{ lotteryCode }}</div>
              <img v-if="lotteryQrUrl" :src="lotteryQrUrl" alt="QR lotteria" class="lottery-qr" />
              <p class="lottery-hint">Resta fino a fine partita per ritirare il premio.</p>
            </template>
            <p v-else class="lottery-empty">Vota l'MVP per ottenere il tuo QR lotteria personale.</p>
          </div>
        </div>
      </div>

      <!-- Overlay gioco a schermo pieno -->
      <div v-if="activeGameId" class="game-overlay">
        <header class="game-head">
          <button class="close" type="button" aria-label="Indietro" :disabled="isBasketballSaving" @click="closeGame">←</button>
          <div class="game-title">{{ games[activeGameId]?.title }}</div>
          <button class="close" type="button" aria-label="Chiudi" :disabled="isBasketballSaving" @click="closeGame">✕</button>
        </header>
        <div v-if="coinError" class="game-credit-error" role="alert"><p>{{ coinError }}</p><button v-if="pendingGameClaim" type="button" class="journey-next" :disabled="isSavingCoins" @click="onGameClaim(pendingGameClaim)">Riprova il salvataggio</button></div>
        <div class="game-stage">
          <ReactionTestGame
            v-if="activeGameId === 'reaction'"
            class="game-fill"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <QuickQuizGame
            v-else-if="activeGameId === 'quiz'"
            class="game-fill"
            :event-id="eventId"
            :completion-reward="QUIZ_REWARD"
            :claiming="isSavingCoins"
            :already-rewarded="doneKeys.includes('quiz')"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <BasketballShootout
            v-else-if="activeGameId === 'basketball'"
            :key="eventId"
            class="game-fill"
            :event-id="eventId"
            :wallet-coins="totalCoins"
            :earned-coins="journey.rewards.basketball || 0"
            :club-name="homeName"
            :club-logo-url="homeLogoUrl"
            @credited="onBasketballCredit"
            @saving-change="isBasketballSaving = $event"
          />
          <TapChallenge
            v-else-if="activeGameId === 'tap'"
            class="game-fill"
            :event-id="eventId"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <SponsorRushGame
            v-else-if="activeGameId === 'sponsor-rush'"
            class="game-fill"
            @claim="onGameClaim"
            @exit="closeGame"
          />
          <MemoryFlashGame
            v-else-if="activeGameId === 'memory-flash'"
            class="game-fill"
            :wallet-coins="totalCoins"
            @claim="onGameClaim"
            @spend="onGameSpend"
            @exit="closeGame"
          />
        </div>
      </div>

    </div>

    <div class="coin-flight-layer" aria-hidden="true"><span v-for="(flight, index) in coinFlights" :key="index" class="coin coin-flight" :style="flight"></span></div>
    <EventFeedbackModal
      v-model="isFeedbackModalOpen"
      :event-id="eventId"
      :feedback-survey="feedbackSurvey"
      :theme-style="themeStyle"
      @submitted="handleFeedbackSubmitted"
    />
    <FanRegistrationPromptModal v-if="isRegistrationOpen" v-model="isRegistrationOpen" trigger="profile_overlay" :wallet-coins="totalCoins" :on-submit="registerJourneyFan" :on-login="loginJourneyFan" />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { normalizeOrganizationColors, organizationThemeStyle } from '../organizationTheme';
import { preloadFanHomeImages } from '../fanHomeAssets';
import EventFeedbackModal from '../components/EventFeedbackModal.vue';
import FanPlayerCard from '../components/FanPlayerCard.vue';
const FanRegistrationPromptModal = defineAsyncComponent(() => import('../components/FanRegistrationPromptModal.vue'));
import { FIRST_FAN_GOAL, VOTE_REWARD, QUIZ_REWARD, FAN_BADGE_NAME, journeyStorageKey, normalizeJourney, journeyProgress, recordJourneyReward } from '../fanJourney';

// Giochi reali riusati da EarnCoinsModal (lazy-load: pesano).
const ReactionTestGame = defineAsyncComponent(() => import('../components/ReactionTestGame.vue'));
const QuickQuizGame = defineAsyncComponent(() => import('../components/QuickQuizGame.vue'));
const BasketballShootout = defineAsyncComponent(() => import('../components/minigames/BasketballShootout.vue'));
const TapChallenge = defineAsyncComponent(() => import('../components/minigames/TapChallenge.vue'));
const SponsorRushGame = defineAsyncComponent(() => import('../components/minigames/SponsorRushGame.vue'));
const MemoryFlashGame = defineAsyncComponent(() => import('../components/minigames/MemoryFlashGame.vue'));
import {
  apiClient,
  fetchFanProfile,
  fetchVoteStatus,
  fetchQuickQuiz,
  vote,
  syncGuestCoins,
  redeemFanReward,
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

const props = defineProps({
  initialDataReady: { type: Boolean, default: true },
  organizationSlug: { type: String, default: '' },
  eventId: { type: Number, default: 0 },
  teamName: { type: String, default: 'TEAM' },
  teamLogoUrl: { type: String, default: '' },
  matchLabel: { type: String, default: 'Vota • Gioca • Vinci • Partecipa' },
  activeEvent: { type: Object, default: null },
  votedPlayerImageUrl: { type: String, default: '' },
  votedPlayerName: { type: String, default: '' },
  votedPlayerLastName: { type: String, default: '' },
  votedPlayerNumber: { type: [String, Number], default: '' },
  // Segnale dal parent dopo il primo voto (registrazione). Accettato per compat; TODO: prompt registrazione.
  registrationPromptSignal: { type: Number, default: 0 },
});

const WALLET_STORAGE_KEY = 'wallet:coins';
const emit = defineEmits(['loading-change', 'brand-change']);
const isHomeLoading = ref(true);
let bootstrapRevision = 0;
let assetController;
let organizationThemeRequest = Promise.resolve();
watch(isHomeLoading, (loading) => emit('loading-change', loading), { immediate: true, flush: 'sync' });

const organizationColors = ref(normalizeOrganizationColors());
const organizationName = ref('');
const organizationLogoUrl = ref('');
const homeLogoUrl = computed(() => props.organizationSlug ? organizationLogoUrl.value : props.teamLogoUrl);
const themeStyle = computed(() => organizationThemeStyle(organizationColors.value));
watch(() => props.organizationSlug, (slug, previous, onCleanup) => {
  organizationColors.value = normalizeOrganizationColors();
  organizationName.value = '';
  organizationLogoUrl.value = '';
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
    if (!signal.aborted) {
      organizationColors.value = normalizeOrganizationColors(data);
      organizationName.value = String(data?.name || '').trim();
      organizationLogoUrl.value = String(data?.logo_url || '').trim();
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
const leaderboardTop3 = ref([]);
const leaderboardUser = ref(null);
const leaderboardNearby = ref([]);
const leaderboardGap = ref(null);
const liveEvent = ref(null);
const isRegistrationOpen = ref(false);
const isLoadingLeaderboard = ref(false);
const leaderboardError = ref('');
const sponsors = ref([]);

const rawPlayers = ref([]);
const isLoadingPlayers = ref(false);
const playersError = ref('');
const isVoting = ref(false);
const voteError = ref('');
const coinError = ref('');
const isSavingCoins = ref(false);
const isBasketballSaving = ref(false);
const pendingGameClaim = ref(null);
const voteBonusCoins = ref(0);
const celebrationKind = ref('vote');
const celebrationName = ref('');
const isWalletReady = ref(false);
const journey = ref(normalizeJourney());
const progress = computed(() => journeyProgress(journey.value));
const quizAvailable = ref(false);
const walletEl = ref(null);
const voteResultEl = ref(null);
const walletPulse = ref(false);
const coinFlights = ref([]);
let coinAnimationTimer;
let walletRevision = 0;
let loadedEventId = null;
let liveStatusTimer;
let liveRequestEventId = null;

const doneKeys = ref([]); // missioni completate (persistite per evento)

const activeSheet = ref('');
const activeGameId = ref(null);
const isFeedbackModalOpen = ref(false);
const hasSubmittedFeedback = ref(false);

// Registry giochi reali (titolo + reward indicativa mostrata nella lista).
const games = {
  reaction: { title: 'Reaction Test', reward: 10 },
  quiz: { title: 'Quiz Lampo', reward: 15 },
  basketball: { title: 'Basket Rush', reward: 1 },
  tap: { title: 'Tap Challenge', reward: 8 },
  'sponsor-rush': { title: 'Sponsor Rush', reward: 12 },
  'memory-flash': { title: 'Memory Flash', reward: 8 },
};
const redeeming = ref('');
let coinsStream = null;
const eventState = computed(() => liveEvent.value || (!props.activeEvent?.id || props.activeEvent.id === props.eventId ? props.activeEvent : null));
const matchConcluded = computed(() => eventState.value?.is_concluded === true);
const showMatchSummary = computed(() => Boolean(props.eventId) && (matchConcluded.value || eventState.value?.votes_closed === true));
const canVote = computed(() => Boolean(props.eventId) && Boolean(eventState.value) && eventState.value?.is_active !== false && !eventState.value?.votes_closed && !matchConcluded.value);
const completedGames = computed(() => Object.keys(journey.value.rewards).filter((key) => key !== 'vote').length);
const featuredMission = computed(() => {
  if (!props.eventId || matchConcluded.value || eventState.value?.is_active === false) return null;
  const missions = earnMissions.value.filter((mission) => mission.game && (mission.repeatable || !doneKeys.value.includes(mission.key)));
  return (quizAvailable.value && missions.find((mission) => mission.key === 'quiz')) || missions.find((mission) => mission.key !== 'quiz') || null;
});
const nextQuizLabel = computed(() => {
  const startsAt = Date.parse(liveEvent.value?.quiz?.starts_at || '');
  if (!Number.isFinite(startsAt) || startsAt <= Date.now()) return '';
  return `Prossimo quiz alle ${new Date(startsAt).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })}`;
});
const rankGapLabel = computed(() => {
  const above = leaderboardNearby.value.find((row) => row.rank === Number(leaderboardUser.value?.rank) - 1);
  if (!above || !leaderboardUser.value) return Number(leaderboardUser.value?.rank) === 1 ? 'Sei in testa alla classifica della società!' : '';
  const gap = leaderboardGap.value ?? Math.max(1, above.coins - Number(leaderboardUser.value.coins) + 1);
  return `${gap === 1 ? 'Manca 1 moneta' : `Mancano ${gap} monete`} per superare il ${above.rank}° posto`;
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
  if (quizAvailable.value && !doneKeys.value.includes('quiz')) return `Completa il quiz · +${QUIZ_REWARD}`;
  return 'Scegli una sfida →';
});
function openJourneyNext() {
  if (showMatchSummary.value) openSheet('match-summary');
  else if (progress.value.unlocked) openSheet('spend');
  else if (!hasVoted.value && canVote.value) openSheet('mvp');
  else if (quizAvailable.value && !doneKeys.value.includes('quiz')) openGame('quiz');
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
  liveRequestEventId = eventId;
  try {
    const { data } = await apiClient.get(`/events/${eventId}/fan-live`);
    if (eventId !== props.eventId || Number(data?.event_id) !== eventId) return;
    liveEvent.value = data;
    quizAvailable.value = data.quiz?.available === true;
  } catch (error) { /* conserva le ultime informazioni ricevute se la rete non è disponibile */ }
  finally { if (liveRequestEventId === eventId) liveRequestEventId = null; }
}

function openRegistration() { closeSheets(); isRegistrationOpen.value = true; }
async function registerJourneyFan(form) {
  const response = await registerFanProfile({ event_id: props.eventId, nickname: form.nickname, gender: form.gender, phone: form.phone, accepted_terms: form.acceptedTerms, guest_coins: totalCoins.value, enter_lottery: false });
  if (!response.ok) return response;
  isRegistered.value = true;
  fanNickname.value = String(response.data?.user?.nickname || form.nickname || '').trim();
  setCoins(Number(response.data?.wallet ?? totalCoins.value));
  await loadFanProfile();
  isRegistrationOpen.value = false;
  await refreshLeaderboard();
  openSheet('leaderboard');
  return { ok: true, wallet: totalCoins.value };
}
async function loginJourneyFan() {
  await loadFanProfile();
  if (!isRegistered.value) return { ok: false, message: 'Impossibile caricare il profilo. Riprova.' };
  isRegistrationOpen.value = false;
  await refreshLeaderboard();
  openSheet('leaderboard');
  return { ok: true };
}

function saveJourney() {
  try { window.localStorage.setItem(journeyStorageKey(props.organizationSlug, props.eventId), JSON.stringify(journey.value)); } catch (error) { /* storage non disponibile */ }
}
function loadJourney() {
  try { journey.value = normalizeJourney(JSON.parse(window.localStorage.getItem(journeyStorageKey(props.organizationSlug, props.eventId)) || 'null')); }
  catch (error) { journey.value = normalizeJourney(); }
}
async function animateEarnedCoins() {
  walletPulse.value = true;
  await nextTick();
  if (activeSheet.value === 'vote-result' && walletEl.value && voteResultEl.value) {
    const from = voteResultEl.value.getBoundingClientRect();
    const to = walletEl.value.getBoundingClientRect();
    coinFlights.value = Array.from({ length: 6 }, (_, index) => ({
      left: `${from.left + from.width / 2}px`, top: `${from.top + 60}px`,
      '--flight-x': `${to.left + to.width / 2 - from.left - from.width / 2}px`,
      '--flight-y': `${to.top + to.height / 2 - from.top - 60}px`,
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
const homeBadge = computed(() => initials(homeName.value));
const opponentLabel = computed(() => {
  const opponent = String(eventState.value?.team2_name || '').trim();
  return opponent ? `vs ${opponent}` : 'Esperienza ufficiale';
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
const hasFeedbackSurvey = computed(() =>
  Boolean(props.eventId)
  && eventState.value?.show_feedback_survey !== false
  && eventState.value?.showFeedbackSurvey !== false
  && Array.isArray(feedbackSurvey.value.questions)
  && feedbackSurvey.value.questions.length > 0,
);
const feedbackSubtitle = computed(() => {
  if (hasSubmittedFeedback.value) return 'Grazie per aver condiviso la tua esperienza!';
  if (!props.eventId) return 'Disponibile con un evento attivo';
  if (!hasFeedbackSurvey.value) return 'Sondaggio non disponibile per questa partita';
  return 'Rispondi alle domande e lascia un suggerimento';
});
const feedbackStorageKey = (eventId) => `wcmvpvs-feedback:${eventId}`;
watch(() => props.eventId, (eventId) => {
  isFeedbackModalOpen.value = false;
  hasSubmittedFeedback.value = false;
  if (!eventId || typeof window === 'undefined') return;
  try {
    hasSubmittedFeedback.value = window.localStorage.getItem(feedbackStorageKey(eventId)) === '1';
  } catch (error) {
    /* il flusso resta utilizzabile quando localStorage non è disponibile */
  }
}, { immediate: true });
function openFeedbackModal() {
  if (!hasFeedbackSurvey.value || hasSubmittedFeedback.value) return;
  closeSheets();
  isFeedbackModalOpen.value = true;
}
function handleFeedbackSubmitted() {
  hasSubmittedFeedback.value = true;
  try {
    window.localStorage.setItem(feedbackStorageKey(props.eventId), '1');
  } catch (error) {
    /* lo stato di conferma resta valido per la sessione corrente */
  }
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
const votedButtonLabel = computed(() => {
  const number = votedPlayerCard.value.number;
  const num = number === '' || number == null ? '' : `#${number} `;
  const label = votedPlayerCard.value.lastName || (hasVoted.value ? votedPlayerCard.value.name : '');
  return label ? `${num}${label}`.trim() : 'Modifica voto';
});

/* ---------- Profilo (come LiveExperienceHome) ---------- */
const profileNickname = computed(() => {
  if (fanNickname.value.trim()) return fanNickname.value.trim();
  return isRegistered.value ? 'Tifoso' : 'Ospite';
});
const canSubmitNickname = computed(
  () => !isSavingNickname.value && nicknameDraft.value.trim() !== fanNickname.value.trim(),
);
const lotteryCode = computed(() => String(fanLotteryTicket.value?.ticket_code || '').trim());
const lotteryQrUrl = computed(() => {
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
      subtitle: role || (number ? `Maglia #${number}` : 'In campo'),
      number,
      badge: initials(`${first} ${last}`),
      image: String(p?.image_url || '').trim(),
    };
  });
});

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
  { key: 'reaction', emoji: '⚡', title: 'Reaction Test', subtitle: 'Testa i riflessi', pts: 10, game: 'reaction' },
  { key: 'quiz', emoji: '🧠', title: 'Quiz Lampo', subtitle: 'Domande sul match', pts: 15, game: 'quiz' },
  { key: 'basketball', emoji: '🏀', title: 'Basket Rush', subtitle: 'Trascina e tira · 1 moneta a canestro', pts: 1, game: 'basketball', repeatable: true },
  { key: 'tap', emoji: '👆', title: 'Tap Challenge', subtitle: '10 secondi a martello', pts: 8, game: 'tap' },
  {
    key: 'sponsor-rush',
    emoji: '🏷️',
    title: rushSponsor.value ? `Sponsor Rush · ${rushSponsor.value.name}` : 'Sponsor Rush',
    subtitle: 'Prendi i loghi al volo',
    pts: 12,
    game: 'sponsor-rush',
  },
  { key: 'memory-flash', emoji: '🧩', title: 'Memory Flash', subtitle: 'Memorizza le coppie', pts: 8, game: 'memory-flash' },
]);

/* ---------- Sponsor rush / partner ---------- */
const rushSponsor = computed(() => {
  const first = sponsors.value[0];
  if (!first) return null;
  return { name: first.name, badge: initials(first.name).slice(0, 1) };
});

/* ---------- Premi (riscatto reale via /rewards/redeem) ---------- */
const rewards = ref([
  { key: 'sponsor_discount', emoji: '🎁', title: 'Sconto 10% · Partner', subtitle: 'Codice valido 7 giorni', cost: 80 },
  { key: 'signed_ball', emoji: '🏀', title: 'Pallone autografato', subtitle: 'Estrazione a fine partita', cost: 150 },
  { key: 'next_ticket', emoji: '🎟️', title: 'Biglietto prossimo match', subtitle: 'Settore tifosi', cost: 300 },
]);

/* ---------- Sheets ---------- */
function openSheet(id) {
  activeSheet.value = id;
}
function closeSheets() {
  activeSheet.value = '';
}

/* ---------- Wallet ---------- */
function readStoredCoins() {
  try {
    const raw = window.localStorage.getItem(WALLET_STORAGE_KEY);
    const parsed = Number.parseInt(raw ?? '', 10);
    return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
  } catch (error) {
    return 0;
  }
}
function writeStoredCoins(value) {
  try {
    window.localStorage.setItem(WALLET_STORAGE_KEY, String(value));
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

async function loadFanProfile() {
  if (!props.eventId) return;
  const eventId = props.eventId;
  const revision = walletRevision;
  const { ok, data } = await fetchFanProfile(eventId);
  if (!ok || !data || eventId !== props.eventId) return;
  isWalletReady.value = true;
  if (data.journey_rewards && typeof data.journey_rewards === 'object') {
    const rewards = { ...journey.value.rewards };
    const restored = normalizeJourney({ rewards: data.journey_rewards });
    for (const [key, amount] of Object.entries(restored.rewards)) rewards[key] = Math.max(rewards[key] || 0, amount);
    journey.value = normalizeJourney({ ...journey.value, rewards });
    if (journey.value.rewards.vote) journey.value.pendingVoteBonus = false;
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
    leaderboardUser.value = data.user_rank;
  }
}

// Mostra l'accredito soltanto dopo la conferma del salvataggio del wallet.
async function grantCoins(delta, key) {
  const amount = Math.max(0, Math.floor(Number(delta) || 0));
  if (!props.eventId || !amount || isSavingCoins.value) return { ok: false, coins: 0 };
  if (journey.value.rewards[key] || doneKeys.value.includes(key)) return { ok: true, coins: 0 };
  isSavingCoins.value = true;
  coinError.value = '';
  const eventId = props.eventId;
  try {
    const serverReward = key === 'vote' || key === 'quiz';
    if (!isWalletReady.value && !serverReward) await loadFanProfile();
    if (eventId !== props.eventId) return { ok: false, coins: 0 };
    if (!isWalletReady.value && !serverReward) {
      coinError.value = 'Impossibile leggere il saldo. Controlla la connessione e riprova.';
      return { ok: false, coins: 0 };
    }
    const nextBalance = totalCoins.value + amount;
    const response = serverReward ? await claimFanJourneyReward(eventId, key) : await syncGuestCoins(eventId, nextBalance);
    if (eventId !== props.eventId) return { ok: false, coins: 0 };
    if (!response.ok) {
      coinError.value = response.message || 'Le monete non sono ancora state salvate. Controlla la connessione e riprova.';
      return { ok: false, coins: 0 };
    }
    setCoins(Number(response.data?.wallet ?? nextBalance));
    isWalletReady.value = true;
    journey.value = recordJourneyReward(journey.value, key, amount);
    saveJourney();
    markDone(key);
    refreshLeaderboard();
    return { ok: true, coins: serverReward ? Number(response.data?.coins_earned ?? amount) : amount };
  } catch (error) {
    coinError.value = 'Le monete non sono ancora state salvate. Controlla la connessione e riprova.';
    return { ok: false, coins: 0 };
  } finally { isSavingCoins.value = false; }
}

async function retryVoteBonus() {
  if (!journey.value.pendingVoteBonus) return;
  const result = await grantCoins(VOTE_REWARD, 'vote');
  if (result.ok) {
    journey.value.pendingVoteBonus = false;
    saveJourney();
    voteBonusCoins.value = result.coins;
    if (result.coins) animateEarnedCoins();
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

function onMission(mission) {
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
  if (!games[id] || !props.eventId || isSavingCoins.value || matchConcluded.value || (id === 'quiz' && !quizAvailable.value)) return;
  coinError.value = '';
  pendingGameClaim.value = null;
  activeSheet.value = '';
  activeGameId.value = id;
}
function closeGame() {
  if (isSavingCoins.value || isBasketballSaving.value) return;
  const wasGame = Boolean(activeGameId.value);
  activeGameId.value = null;
  if (wasGame) {
    openSheet('earn'); // torna alla lista giochi
  }
}
function onBasketballCredit(receipt) {
  if (receipt?.event_id !== props.eventId) return;
  const wallet = Number(receipt.wallet);
  const earned = Number(receipt.basketball_earned);
  if (!Number.isInteger(wallet) || wallet < 0 || !Number.isInteger(earned) || earned < 0) return;
  setCoins(wallet);
  isWalletReady.value = true;
  journey.value = normalizeJourney({ ...journey.value, rewards: {
    ...journey.value.rewards, basketball: Math.max(journey.value.rewards.basketball || 0, earned),
  } });
  saveJourney();
  if (earned > 0) markDone('basketball');
  if (Number(receipt.coins_earned) > 0) { animateEarnedCoins(); refreshLeaderboard(); }
}
// Ogni gioco emette @claim { coins, keepOpen } al termine.
async function onGameClaim(payload) {
  if (!activeGameId.value || isSavingCoins.value) return;
  const key = activeGameId.value;
  pendingGameClaim.value = payload;
  const coins = key === 'quiz' ? QUIZ_REWARD : Math.max(0, Number(payload?.coins) || 0);
  let earned = 0;
  if (coins && String(payload?.source || '').startsWith('tap_challenge')) {
    // Tap Challenge already credits its result server-side; only refresh the wallet here.
    isWalletReady.value = false;
    await loadFanProfile();
    if (!isWalletReady.value) { coinError.value = 'Risultato salvato. Riprova a caricare il saldo.'; return; }
    if (!journey.value.rewards[key]) {
      journey.value = recordJourneyReward(journey.value, key, coins);
      saveJourney(); markDone(key); earned = coins;
    }
  } else if (coins) {
    const result = await grantCoins(coins, key);
    if (!result.ok) return;
    earned = result.coins;
  }
  pendingGameClaim.value = null;
  if (!payload?.keepOpen) {
    closeGame();
    if (earned) {
      celebrationKind.value = key;
      celebrationName.value = games[key].title;
      voteBonusCoins.value = earned;
      openSheet('vote-result');
      animateEarnedCoins();
    }
  }
}
// Memory Flash può spendere monete (@spend).
async function onGameSpend(payload) {
  const coins = Math.max(0, Number(payload?.coins) || 0);
  if (!coins) return;
  setCoins(totalCoins.value - coins);
  if (props.eventId) {
    await syncGuestCoins(props.eventId, totalCoins.value);
  }
}

/* ---------- Voto MVP (lista reale + voto reale) ---------- */
async function loadPlayers() {
  isLoadingPlayers.value = true;
  playersError.value = '';
  try {
    const { data } = await apiClient.get('/public/players');
    const payload = Array.isArray(data?.players) ? data.players : Array.isArray(data) ? data : [];
    rawPlayers.value = Array.isArray(payload) ? payload : [];
  } catch (error) {
    playersError.value = 'Impossibile caricare i giocatori. Riprova.';
    rawPlayers.value = [];
  } finally {
    isLoadingPlayers.value = false;
  }
}
async function loadVoteStatus() {
  if (!props.eventId) return;
  const eventId = props.eventId;
  const status = await fetchVoteStatus(eventId);
  if (eventId !== props.eventId || !status.ok || isVoting.value || localVoted.value) return;
  hasVoted.value = Boolean(status?.hasVoted);
  votedPlayerId.value = status?.playerId ?? null;
  if (hasVoted.value && !journey.value.rewards.vote) {
    journey.value.pendingVoteBonus = true;
    saveJourney();
    await retryVoteBonus();
  }
}
async function castVote(player) {
  if (isVoting.value || !player?.id || !props.eventId) return;
  voteError.value = '';
  if (!canVote.value) { voteError.value = 'Le votazioni per questa partita sono chiuse.'; return; }
  isVoting.value = true;
  const firstVote = !hasVoted.value && !doneKeys.value.includes('vote');
  const eventId = props.eventId;
  try {
    const response = await vote({ eventId, playerId: player.id });
    if (eventId !== props.eventId) return;
    if (!response?.ok) {
      voteError.value = response?.message || 'Il voto non è stato salvato. Controlla la connessione e riprova.';
      return;
    }
    hasVoted.value = true;
    votedPlayerId.value = player.id;
    localVoted.value = { ...player };
    voteBonusCoins.value = 0;
    celebrationKind.value = 'vote';
    if (firstVote) {
      journey.value.pendingVoteBonus = true;
      saveJourney();
    }
    openSheet('vote-result');
    if (journey.value.pendingVoteBonus) await retryVoteBonus();
  } catch (error) {
    voteError.value = 'Impossibile salvare il voto. Riprova tra poco.';
  } finally {
    isVoting.value = false;
  }
}

/* ---------- Leaderboard ---------- */
async function refreshLeaderboard() {
  if (!props.eventId) {
    leaderboardTop3.value = [];
    leaderboardUser.value = null;
    leaderboardNearby.value = [];
    leaderboardGap.value = null;
    leaderboardError.value = '';
    return;
  }
  isLoadingLeaderboard.value = true;
  leaderboardError.value = '';
  try {
    const { data } = await apiClient.get(`/events/${props.eventId}/coins-leaderboard`, { headers: { ...getDeviceHeaders(), ...getFanSessionHeaders() } });
    const top = Array.isArray(data?.top3) ? data.top3 : Array.isArray(data?.top) ? data.top : [];
    leaderboardTop3.value = top.slice(0, 3).map((row) => ({
      name: String(row?.name || row?.nickname || 'Tifoso'),
      coins: Number(row?.coins ?? row?.points ?? 0),
    }));
    const rank = data?.userRank || data?.user_rank;
    leaderboardUser.value = rank || null;
    leaderboardNearby.value = Array.isArray(data?.nearby) ? data.nearby : [];
    leaderboardGap.value = Number.isFinite(data?.next_rank_gap) && data.next_rank_gap > 0 ? data.next_rank_gap : null;
  } catch (error) {
    leaderboardError.value = 'Impossibile caricare la classifica. Riapri il pannello per riprovare.';
  } finally {
    isLoadingLeaderboard.value = false;
  }
}
function openLeaderboard() {
  openSheet('leaderboard');
  refreshLeaderboard();
}

/* ---------- Profilo ---------- */
function openProfile() {
  nicknameDraft.value = fanNickname.value.trim();
  isNicknameEditorOpen.value = false;
  nicknameError.value = '';
  nicknameSuccess.value = '';
  loadFanProfile();
  openSheet('profile');
}
function toggleNicknameEditor() {
  isNicknameEditorOpen.value = !isNicknameEditorOpen.value;
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
  nicknameError.value = '';
  nicknameSuccess.value = '';
  const check = validateNickname(nicknameDraft.value);
  if (!check.valid) {
    nicknameError.value = check.message;
    return;
  }
  if (check.nickname === fanNickname.value.trim()) {
    nicknameSuccess.value = 'Nessuna modifica da salvare.';
    return;
  }
  isSavingNickname.value = true;
  const response = await updateFanNickname(check.nickname);
  isSavingNickname.value = false;
  if (!response?.ok) {
    nicknameError.value = response?.message || 'Impossibile aggiornare il nickname.';
    return;
  }
  fanNickname.value = String(response.data?.user?.nickname || check.nickname).trim();
  nicknameDraft.value = fanNickname.value;
  nicknameSuccess.value = response.data?.message || 'Nickname aggiornato';
  isNicknameEditorOpen.value = false;
  refreshLeaderboard();
}

/* ---------- Sponsors ---------- */
// Stessa normalizzazione di LiveExperienceHome: sponsor = logo (logo_data/image_url).
function normalizeSponsor(item, index) {
  const imageUrl = String(item?.logo_data || item?.image_url || item?.imageUrl || '').trim();
  if (!imageUrl) return null;
  const priorityRaw = Number(item?.priority ?? item?.order_index ?? item?.order ?? item?.display_order);
  return {
    id: Number(item?.id) || index + 1,
    name: String(item?.name || '').trim(),
    imageUrl,
    linkUrl: String(item?.link_url || item?.linkUrl || '').trim(),
    priority: Number.isFinite(priorityRaw) ? priorityRaw : Number.POSITIVE_INFINITY,
    insertedIndex: index,
  };
}
async function loadSponsors() {
  try {
    const { data } = await apiClient.get('/sponsors');
    const list = Array.isArray(data) ? data : Array.isArray(data?.sponsors) ? data.sponsors : [];
    sponsors.value = list
      .map((item, index) => normalizeSponsor(item, index))
      .filter(Boolean)
      .sort((a, b) => (a.priority !== b.priority ? a.priority - b.priority : a.insertedIndex - b.insertedIndex));
  } catch (error) {
    sponsors.value = [];
  }
}
function onSponsorClick(sponsor) {
  if (props.eventId && sponsor?.id) {
    apiClient
      .post(`/events/${props.eventId}/sponsors/${sponsor.id}/click`, { at: new Date().toISOString() })
      .catch(() => {});
  }
  if (sponsor?.linkUrl) {
    window.open(sponsor.linkUrl, '_blank', 'noopener');
  }
}

/* ---------- Riscatto premi ---------- */
async function redeem(reward) {
  if (redeeming.value || isSavingCoins.value) return;
  if (totalCoins.value < reward.cost) {
    return;
  }
  redeeming.value = reward.key;
  try {
    const { ok, data } = await redeemFanReward(props.eventId, reward.key, reward.cost);
    if (ok && data && data.wallet != null) {
      setCoins(Number(data.wallet));
    } else if (ok) {
      setCoins(totalCoins.value - reward.cost);
    } else {
      return;
    }
    closeSheets();
  } catch (error) {
    // riscatto non riuscito: nessuna azione
  } finally {
    redeeming.value = '';
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
async function bootstrap() {
  const revision = ++bootstrapRevision;
  assetController?.abort();
  isHomeLoading.value = true;
  if (!props.initialDataReady) return;
  assetController = new AbortController();
  const { signal } = assetController;
  if (loadedEventId !== props.eventId) {
    loadedEventId = props.eventId;
    liveEvent.value = null;
    isWalletReady.value = false;
    hasVoted.value = false;
    localVoted.value = null;
    votedPlayerId.value = null;
    activeGameId.value = null;
    isBasketballSaving.value = false;
    closeSheets();
    coinError.value = '';
    voteError.value = '';
  }
  totalCoins.value = readStoredCoins();
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
    loadSponsors(),
    ensureCentoFonts(),
  ]);
  if (revision !== bootstrapRevision || signal.aborted) return;
  await nextTick();
  await preloadFanHomeImages([
    homeLogoUrl.value,
    ...players.value.map((player) => player.image),
    ...sponsors.value.map((sponsor) => sponsor.imageUrl),
    displayVotedImage.value,
    lotteryQrUrl.value,
  ], { signal });
  if (revision !== bootstrapRevision || signal.aborted) return;
  isHomeLoading.value = false;
  startCoinsStream();
}

function ensureCentoFonts() {
  if (typeof document === 'undefined') return;
  const id = 'cento-fonts';
  let link = document.getElementById(id);
  if (!link) {
    link = document.createElement('link');
    link.id = id;
    link.rel = 'stylesheet';
    link.href =
      'https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,600;9..144,750;9..144,900&family=Inter:wght@400;500;600;700;800&display=swap';
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
  bootstrapRevision += 1;
  assetController?.abort();
  window.removeEventListener('keydown', onKeydown);
  stopCoinsStream();
  window.clearTimeout(coinAnimationTimer);
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
  width: 100%;
  max-width: 430px;
  height: 100dvh;
  background:
    radial-gradient(90% 40% at 50% -6%, var(--hero-glow), transparent 62%),
    var(--bg);
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

/* Top bar */
.topbar {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}
.club {
  display: flex;
  align-items: center;
  gap: 9px;
}
.club-logo {
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
  box-shadow: var(--shadow-red);
  transform: rotate(-4deg);
  overflow: hidden;
}
.club-logo.has-logo {
  transform: none;
}
.club-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.club-name {
  line-height: 1.15;
}
.club-name b {
  font-size: 13px;
  font-weight: 800;
  display: block;
  letter-spacing: -0.01em;
}
.club-name span {
  font-size: 10px;
  color: var(--muted);
  font-weight: 600;
}
.wallet {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--gold-soft);
  border: 2px solid var(--gold);
  border-radius: 999px;
  padding: 6px 13px 6px 8px;
  font-weight: 800;
  font-size: 14px;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  color: var(--gold-deep);
  cursor: pointer;
  box-shadow: 0 3px 0 rgba(209, 142, 0, 0.25);
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
  align-items: center;
  gap: 8px;
}
.profile-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  flex: none;
  background: var(--card);
  border: 2px solid var(--red-line);
  color: var(--red-text);
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow);
  transition: transform 0.12s ease, border-color 0.15s ease;
}
.profile-btn:active {
  transform: translateY(2px);
  box-shadow: none;
}
.profile-btn:hover {
  border-color: var(--red);
}

main {
  flex: 1;
  min-height: 0;
  padding: 6px 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.75); }
}

/* Grid */
.grid {
  flex: 1 0 280px;
  min-height: 0;
  display: grid;
  grid-template-columns: 1.15fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 10px;
}
.tile {
  background: var(--card);
  border: 2px solid var(--line);
  border-radius: 20px;
  padding: 13px;
  display: flex;
  flex-direction: column;
  text-align: left;
  color: var(--ink);
  font-family: inherit;
  cursor: pointer;
  transition: transform 0.12s ease, border-color 0.15s ease;
  min-height: 0;
  overflow: hidden;
  box-shadow: var(--shadow);
}
.tile:active {
  transform: scale(0.98) translateY(2px);
  box-shadow: none;
}
.tile:hover {
  border-color: var(--red-line);
}
.tile-mvp {
  grid-row: 1 / 3;
  border-color: var(--red-line);
  background: linear-gradient(175deg, var(--card) 55%, var(--red-soft));
  position: relative;
}
.tile-icon {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--red-soft);
  color: var(--red-text);
  margin-bottom: 9px;
  transform: rotate(-4deg);
}
.tile b {
  font-size: 14px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.tile p {
  font-size: 11px;
  color: var(--muted);
  line-height: 1.4;
  margin-top: 3px;
  font-weight: 500;
}
.tile .spacer {
  flex: 1;
  min-height: 4px;
}
.tile-cta {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 800;
  color: var(--red-text);
}
.mvp-photo {
  flex: 1;
  min-height: 0;
  margin: 9px 0;
  border-radius: 14px;
  background:
    radial-gradient(80% 80% at 50% 20%, var(--red-soft), var(--red-line)),
    var(--red-soft);
  border: 2px solid var(--red-line);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}
.mvp-photo svg {
  opacity: 0.55;
}
.mvp-photo.has-player {
  border: 0;
  border-radius: 0;
  background: transparent;
  overflow: visible;
}
.mvp-home-card {
  height: 100%;
  width: auto;
  max-width: 100%;
  max-height: 190px;
  flex: none;
}
.status {
  position: absolute;
  top: 8px;
  left: 8px;
  font-size: 9.5px;
  font-weight: 800;
  color: var(--on-primary);
  background: var(--red);
  padding: 4px 9px;
  border-radius: 999px;
  display: flex;
  align-items: center;
  gap: 5px;
}
.status::before {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--card);
  animation: pulse 1.5s ease-in-out infinite;
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

/* Box feedback */
.journey-card {
  flex: none;
  padding: 12px;
  border: 2px solid var(--red-line);
  background: var(--card);
  border-radius: 18px;
  box-shadow: var(--shadow);
}
.journey-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.journey-head > div { min-width: 0; }
.journey-head b { display: block; font-size: 13px; font-weight: 800; }
.journey-head span:not(.coin) { display: block; font-size: 11px; color: var(--muted); margin-top: 3px; }
.journey-head strong { display: flex; align-items: center; gap: 5px; flex: none; font-size: 14px; }
.journey-bar { width: 100%; height: 8px; border-radius: 99px; background: var(--red-soft); margin-top: 10px; overflow: hidden; }
.journey-bar > span { display: block; height: 100%; background: var(--red); border-radius: inherit; transition: width 0.4s ease; }
.journey-footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-top: 7px; flex-wrap: wrap; }
.journey-footer small { color: var(--muted); font-size: 10px; }
.journey-next { color: var(--red-text); background: transparent; border: 0; padding: 6px 0; font: inherit; font-size: 12px; font-weight: 800; cursor: pointer; }
.journey-badge { border-color: var(--gold); }
.journey-badge .row-copy span { white-space: normal; }
.moment-card { display: flex; align-items: center; gap: 10px; flex: none; width: 100%; background: var(--card); border: 2px solid var(--line); border-radius: 18px; padding: 11px 12px; color: var(--ink); text-align: left; font-family: inherit; cursor: pointer; box-shadow: var(--shadow); }
.moment-card .row-copy span { display: block; white-space: normal; }
.rank-gap { margin: 0; padding: 10px 12px; font-size: 12.5px; font-weight: 800; background: var(--card); border: 2px solid var(--red-line); border-radius: 14px; }
.match-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
.match-stats > div { padding: 12px 7px; background: var(--card); border: 2px solid var(--line); border-radius: 14px; text-align: center; }
.match-stats b { display: block; font-family: 'Fraunces', serif; font-size: 24px; }
.match-stats span { display: block; font-size: 11px; color: var(--muted); }
.vote-result { align-items: center; text-align: center; padding: 8px 0; gap: 13px; }
.vote-result h2 { margin: 0; font: 750 23px/1.2 'Fraunces', serif; }
.vote-result > p { margin: 0; font-size: 13px; }
.vote-result .vote-result-bonus { display: flex; align-items: center; gap: 6px; color: var(--gold-deep); font-size: 17px; font-weight: 800; }
.vote-result-avatar { width: 72px; height: 72px; border: 2px solid var(--red-line); border-radius: 20px; background: var(--red-soft); overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 36px; flex: none; }
.vote-result-player { width: 112px; flex: none; }
.final-mvp-cards { display: flex; flex-wrap: wrap; justify-content: center; gap: 14px 10px; width: 100%; padding: 6px 4px 12px; }
.final-mvp-cards > .fan-player-card { width: min(112px, calc((100% - 20px) / 3)); flex: none; }
.vote-result-goal { width: 100%; padding: 12px; border-radius: 14px; background: var(--card); border: 2px solid var(--line); font-size: 12px; }
.game-credit-error { padding: 10px 14px; margin: 0; color: var(--red-text); background: var(--card); font-size: 13px; font-weight: 700; }
.coin-flight-layer { position: fixed; inset: 0; z-index: 140; pointer-events: none; }
.coin-flight { position: absolute; animation: coin-fly 0.7s ease-in both; }
.wallet-pulse { animation: wallet-pop 0.55s ease; }
@keyframes coin-fly {
  0% { opacity: 0; transform: translate(0, 0) scale(0.5); }
  20% { opacity: 1; }
  90% { opacity: 1; }
  100% { opacity: 0; transform: translate(var(--flight-x), var(--flight-y)) scale(0.6); }
}
@keyframes wallet-pop { 50% { transform: scale(1.1); } }

.feedback-cta {
  flex: none;
  display: flex;
  align-items: center;
  gap: 11px;
  background: var(--red);
  border: none;
  border-radius: 20px;
  padding: 12px 14px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  font-family: inherit;
  color: var(--on-primary);
  text-align: left;
  transition: transform 0.12s ease;
  box-shadow: 0 4px 0 #0d0608;
}
.feedback-cta:active {
  transform: scale(0.98) translateY(2px);
  box-shadow: none;
}
.feedback-cta:disabled {
  cursor: default;
  opacity: 0.7;
}
.feedback-cta::before {
  content: '';
  position: absolute;
  right: -18px;
  top: -22px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: var(--on-primary);
  opacity: 0.08;
}
.feedback-cta-icon {
  width: 36px;
  height: 36px;
  border-radius: 11px;
  flex: none;
  background: var(--card);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Fraunces', serif;
  font-weight: 900;
  font-size: 16px;
  color: var(--ink);
  transform: rotate(-4deg);
}
.feedback-cta-copy {
  flex: 1;
  min-width: 0;
  position: relative;
}
.feedback-cta-copy b {
  font-size: 12.5px;
  font-weight: 800;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.feedback-cta-copy span {
  font-size: 10.5px;
  color: inherit;
  opacity: 0.85;
  font-weight: 500;
}
.feedback-cta-action {
  display: flex;
  align-items: center;
  gap: 5px;
  flex: none;
  background: var(--card);
  padding: 6px 11px;
  border-radius: 999px;
  font-weight: 800;
  font-size: 12.5px;
  color: var(--red-text);
  font-variant-numeric: tabular-nums;
  position: relative;
}

/* Sponsors (blocco ampliato) */
.sponsors {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 2px 2px 0;
}
.sponsors-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.sponsors-head .label {
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.12em;
  color: var(--muted);
}
.sponsor-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
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
.mvp-player-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 14px 10px;
  padding: 6px 4px 12px;
  flex-shrink: 0;
}
.mvp-player {
  position: relative;
  flex: 0 0 calc((100% - 20px) / 3);
  min-width: 0;
  padding: 0;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: #39270f;
  font: inherit;
  text-align: center;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: transform 0.16s ease, filter 0.16s ease;
}
.mvp-player:focus-visible {
  outline: 3px solid var(--red-text);
  outline-offset: 3px;
}
@media (hover: hover) {
  .mvp-player:hover:not(:disabled) { transform: translateY(-3px); }
}
.mvp-player:active:not(:disabled) { transform: scale(0.97); }
.mvp-player:disabled { cursor: default; }
.mvp-player:disabled:not(.is-selected) { opacity: 0.65; }
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

/* Profilo */
.profile-hero {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0 6px;
}
.profile-avatar {
  width: 66px;
  height: 66px;
  border-radius: 50%;
  flex: none;
  background: var(--red-soft);
  border: 2px solid var(--red-line);
  display: flex;
  align-items: center;
  justify-content: center;
  transform: rotate(-3deg);
}
.profile-label {
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--muted);
}
.profile-nick {
  font-family: 'Fraunces', serif;
  font-size: 22px;
  font-weight: 750;
  color: var(--ink);
  line-height: 1.1;
}
.profile-toggle {
  justify-content: center;
}
.nick-editor {
  display: flex;
  gap: 8px;
  align-items: center;
}
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
.coin-balance {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--gold-soft);
  border: 2px solid var(--gold);
  border-radius: 15px;
  padding: 12px 15px;
  box-shadow: 0 3px 0 rgba(209, 142, 0, 0.2);
}
.coin-balance-label {
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--gold-deep);
}
.coin-balance-val {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: 'Fraunces', serif;
  font-size: 24px;
  font-weight: 900;
  color: var(--gold-deep);
  font-variant-numeric: tabular-nums;
}
.lottery-box {
  background: var(--card);
  border: 2px solid var(--line);
  border-radius: 15px;
  padding: 14px 15px;
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}
.lottery-title {
  font-family: 'Fraunces', serif;
  font-size: 15px;
  font-weight: 750;
  color: var(--ink);
}
.lottery-code {
  font-family: 'Fraunces', serif;
  font-size: 18px;
  font-weight: 900;
  letter-spacing: 0.06em;
  color: var(--red-text);
}
.lottery-qr {
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
.lottery-empty {
  font-size: 12px;
  color: var(--muted);
  font-weight: 500;
  line-height: 1.5;
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
  .coin-flight { display: none; }
  .wallet-pulse { animation: none; }
  .journey-bar > span { transition: none; }
  .status::before {
    animation: none;
  }
  .sheet,
  .overlay,
  .tile,
  .feedback-cta,
  .mvp-player,
  .row-item {
    transition: none;
  }
}
</style>

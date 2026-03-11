<template>
  <div class="live-experience relative h-[100dvh] overflow-hidden text-white">
    <div class="arena-bg absolute inset-0" aria-hidden="true" />
    <div class="vignette absolute inset-0" aria-hidden="true" />

    <main class="relative z-10 flex h-full flex-col px-3 pb-3 pt-3 sm:px-4">
      <LiveHeader
        :team-name="teamName"
        :team-logo-url="teamLogoUrl"
        :is-live="isLive"
        :profile-avatar-url="profileAvatarUrl"
        :sponsor-line="sponsorLine"
        @profile-click="openProfileOverlay"
      >
        <StoriesBar
          v-if="activeStories.length"
          :stories="activeStories"
          :seen-ids="seenStoryIds"
          :loading-story-id="loadingStoryId"
          @open="openStory"
        />
        <template v-else>
          <p class="truncate text-center text-[clamp(0.86rem,2.8vw,1.16rem)] font-extrabold tracking-tight text-white">
            LIVE EXPERIENCE UFFICIALE
          </p>
          <p class="mt-1 truncate text-center text-[clamp(0.62rem,2.1vw,0.84rem)] text-slate-200/90">
            {{ sponsorLine }}
          </p>
        </template>
      </LiveHeader>

      <section class="hero animate-on-enter mt-[3.2vh] text-center">
        <h1 class="font-black uppercase leading-[0.92] tracking-tight drop-shadow-[0_4px_14px_rgba(0,0,0,0.85)]">
          <span class="block text-[clamp(2rem,10vw,4rem)]">ENTRA NELLA</span>
          <span class="block text-[clamp(2.7rem,12vw,4.8rem)] text-amber-400">PARTITA</span>
        </h1>
        <p class="mx-auto mt-2 max-w-[92%] border-t border-amber-300/50 pt-2 text-[clamp(0.9rem,3.8vw,1.4rem)] font-extrabold tracking-tight text-slate-100/95 drop-shadow-md">
          {{ matchLabel }}
        </p>
      </section>

      <section class="animate-on-enter mt-[3.2vh] grid grid-cols-3 gap-2.5">
        <FeatureCard
          v-bind="voteFeature"
          @select="onFeatureSelect"
        />

        <div class="flex min-h-[220px] flex-col gap-2">
          <article
            class="mini-feature mini-feature--earn"
            role="button"
            tabindex="0"
            aria-label="Apri guadagna monete"
            @click="onFeatureSelect('game-live')"
            @keydown.enter.prevent="onFeatureSelect('game-live')"
            @keydown.space.prevent="onFeatureSelect('game-live')"
          >
            <div class="mini-feature__content">
              <div class="text-center">
                <p id="wallet-coin-target" class="mini-feature__coins">🪙 {{ totalCoins }}</p>
                <p v-if="coinBoostActive" class="mini-feature__boost">BOOST x2 · {{ coinBoostCountdownLabel }}</p>
              </div>
            </div>
            <button type="button" class="mini-feature__cta mini-feature__cta--earn" @click.stop="onFeatureSelect('game-live')">
              GUADAGNA
            </button>
          </article>

          <article
            class="mini-feature mini-feature--spend"
            role="button"
            tabindex="0"
            aria-label="Apri premi e utilizza monete"
            @click="openSpendPreview"
            @keydown.enter.prevent="openSpendPreview"
            @keydown.space.prevent="openSpendPreview"
          >
            <div class="mini-feature__content">
              <p class="mini-feature__icons" aria-hidden="true">🎁 🏷️ ⚡</p>
            </div>
            <button type="button" class="mini-feature__cta mini-feature__cta--spend" @click.stop="openSpendPreview">
              SPENDI
            </button>
          </article>
        </div>

        <article
          class="leaderboard-preview"
          role="button"
          tabindex="0"
          aria-label="Apri classifica tifosi"
          @click="openLeaderboard"
          @keydown.enter.prevent="openLeaderboard"
          @keydown.space.prevent="openLeaderboard"
        >
          <p class="leaderboard-preview__title">CLASSIFICA TIFOSI</p>
          <ul class="leaderboard-preview__list">
            <li v-for="(entry, index) in leaderboardTop3" :key="`${entry.name}-${index}`" class="leaderboard-preview__item">
              <span>{{ medals[index] }} {{ entry.name }}</span>
              <strong>{{ entry.coins }} 🪙</strong>
            </li>
          </ul>
          <p v-if="isRegisteredFan && leaderboardUser" class="leaderboard-preview__you">Tu: #{{ leaderboardUser.rank }} • {{ leaderboardUser.coins }} 🪙</p>
          <button type="button" class="leaderboard-preview__cta" @click.stop="openLeaderboard">CLASSIFICA</button>
        </article>
      </section>

      <section v-if="showFeedbackCta" class="feedback-area__cta-slot animate-on-enter mt-[2.8vh] mb-[2vh] w-full">
        <ExperienceFeedbackCta
          :disabled="!hasFeedbackSurvey || hasSubmittedFeedback"
          :submitted="hasSubmittedFeedback"
          @select="openFeedbackModal"
        />
      </section>

      <section class="feedback-area__sponsors-slot animate-on-enter mb-[1.2vh] w-full">
        <SponsorsMarquee
          v-if="showSponsorsBox"
          :sponsors="sponsors"
          :event-id="eventId"
          @sponsor-click="handleSponsorClick"
        />
      </section>
    </main>

    <button
      type="button"
      class="fixed bottom-5 right-4 z-[140] inline-flex items-center gap-2 rounded-full border border-amber-200/60 bg-amber-400 px-4 py-3 text-sm font-black uppercase tracking-wide text-slate-950 shadow-[0_12px_30px_rgba(251,191,36,0.45)]"
      @click="openBarOrdering"
    >
      🍺 Bar
    </button>

    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div v-if="isBarModalOpen" class="fixed inset-0 z-[220] bg-slate-100 text-slate-900">
          <div class="mx-auto flex h-full w-full max-w-3xl flex-col">
            <header class="sticky top-0 z-20 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur">
              <div class="flex items-center justify-between gap-3">
                <button v-if="barCanGoBack" type="button" class="rounded-full border border-slate-300 px-3 py-1 text-xs font-bold uppercase" @click="goBackBarStep">Indietro</button>
                <span v-else class="w-16" />
                <div class="text-center">
                  <p class="text-[0.7rem] font-semibold uppercase tracking-[0.2em] text-slate-500">Bar ordering</p>
                  <h3 class="text-base font-black sm:text-lg">{{ barStepTitle }}</h3>
                </div>
                <button type="button" class="rounded-full border border-slate-300 px-3 py-1 text-xs font-bold uppercase" @click="isBarModalOpen = false">Chiudi</button>
              </div>
            </header>

            <div class="flex-1 overflow-y-auto px-4 pb-28 pt-4">
              <section v-if="barStep === 'start'" class="space-y-4">
                <div class="text-center">
                  <h2 class="text-4xl font-black leading-tight text-slate-900">Ordina dal BAR</h2>
                  <p class="mt-2 text-sm text-slate-600">Ritira al banco o ricevi istruzioni per il ritiro.</p>
                </div>
                <article v-for="mode in barOrderModes" :key="mode.id" class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm" :class="barOrderMode === mode.id ? 'ring-2 ring-amber-400' : ''" @click="barOrderMode = mode.id">
                  <p class="text-xs font-bold uppercase text-slate-500">{{ mode.label }}</p>
                  <p class="mt-2 text-3xl">{{ mode.emoji }}</p>
                </article>
                <button type="button" class="w-full rounded-2xl bg-slate-900 py-4 text-lg font-black text-white" @click="goBarStep('categories')">Inizia ordine</button>
              </section>

              <section v-else-if="barStep === 'categories'" class="space-y-4">
                <article v-for="category in barCategories" :key="category.id" class="overflow-hidden rounded-3xl border border-slate-200 bg-white" @click="openBarCategory(category.id)">
                  <div class="p-4"><p class="text-lg font-black">{{ category.name }}</p></div>
                  <img :src="category.image" :alt="category.name" class="h-40 w-full object-cover">
                </article>
              </section>

              <section v-else-if="barStep === 'products'" class="space-y-4">
                <article v-for="product in barProductsByCategory" :key="product.id" class="rounded-3xl border border-slate-200 bg-white p-4 shadow-sm">
                  <img :src="product.image" :alt="product.name" class="h-40 w-full rounded-2xl object-cover">
                  <div class="mt-3 flex items-start justify-between gap-3">
                    <div>
                      <p class="text-xl font-black">{{ product.name }}</p>
                      <p class="text-sm text-slate-500">{{ product.description || 'Ricetta del bar.' }}</p>
                      <p class="mt-2 text-lg font-bold">€ {{ (product.price_cents / 100).toFixed(2) }}</p>
                      <span v-if="product.badge" class="mt-2 inline-block rounded-full bg-amber-100 px-2 py-1 text-xs font-bold text-amber-700">{{ product.badge }}</span>
                    </div>
                    <button type="button" class="rounded-xl bg-slate-900 px-4 py-3 text-sm font-bold text-white" @click="openProductDetail(product.id)">Vedi</button>
                  </div>
                </article>
              </section>

              <section v-else-if="barStep === 'detail' && selectedBarProduct" class="space-y-4">
                <img :src="selectedBarProduct.image" :alt="selectedBarProduct.name" class="h-52 w-full rounded-3xl object-cover">
                <div class="rounded-3xl bg-white p-5">
                  <h4 class="text-2xl font-black">{{ selectedBarProduct.name }}</h4>
                  <p class="mt-2 text-sm text-slate-600">{{ selectedBarProduct.description || 'Seleziona varianti ed extra per completare il tuo ordine.' }}</p>
                  <p class="mt-3 text-xl font-black">€ {{ (selectedBarProduct.price_cents / 100).toFixed(2) }}</p>
                </div>
                <div class="rounded-3xl bg-white p-5">
                  <p class="text-sm font-bold uppercase text-slate-500">Extra</p>
                  <div class="mt-3 space-y-2">
                    <div v-for="extra in barExtras" :key="extra.id" class="flex items-center justify-between rounded-xl border border-slate-200 px-3 py-2">
                      <p class="font-semibold">{{ extra.label }} (+€ {{ extra.price.toFixed(2) }})</p>
                      <button type="button" class="rounded-lg border border-slate-300 px-3 py-1 text-xs font-bold" @click="toggleBarExtra(extra.id)">{{ selectedBarExtras.includes(extra.id) ? 'Rimuovi' : 'Aggiungi' }}</button>
                    </div>
                  </div>
                </div>
                <div class="flex items-center justify-between rounded-3xl bg-white p-5">
                  <div class="flex items-center gap-3">
                    <button type="button" class="h-10 w-10 rounded-full border border-slate-300 text-xl" @click="changeDetailQty(-1)">-</button>
                    <span class="w-8 text-center text-xl font-black">{{ barDetailQty }}</span>
                    <button type="button" class="h-10 w-10 rounded-full border border-slate-300 text-xl" @click="changeDetailQty(1)">+</button>
                  </div>
                  <button type="button" class="rounded-2xl bg-slate-900 px-4 py-3 font-black text-white" @click="addProductFromDetail">Aggiungi al carrello</button>
                </div>
              </section>

              <section v-else-if="barStep === 'upsell'" class="space-y-4">
                <h4 class="text-2xl font-black">Vuoi aggiungere patatine?</h4>
                <article v-if="barUpsellProduct" class="rounded-3xl border border-slate-200 bg-white p-4">
                  <img :src="barUpsellProduct.image" :alt="barUpsellProduct.name" class="h-48 w-full rounded-2xl object-cover">
                  <div class="mt-3 flex items-center justify-between">
                    <div>
                      <p class="text-xl font-black">{{ barUpsellProduct.name }}</p>
                      <p class="text-sm text-slate-500">€ {{ (barUpsellProduct.price_cents / 100).toFixed(2) }}</p>
                    </div>
                    <button type="button" class="rounded-xl bg-amber-400 px-4 py-2 font-black" @click="addUpsellProduct">Aggiungi</button>
                  </div>
                </article>
                <button type="button" class="w-full rounded-2xl border border-slate-300 bg-white py-3 font-bold" @click="goBarStep('products')">Salta e continua</button>
              </section>

              <section v-else-if="barStep === 'cart'" class="space-y-4">
                <article v-for="item in barCartItems" :key="item.id" class="rounded-3xl bg-white p-4">
                  <div class="flex items-center justify-between">
                    <div>
                      <p class="text-lg font-black">{{ item.name }}</p>
                      <p class="text-sm text-slate-500">€ {{ (item.price_cents / 100).toFixed(2) }}</p>
                    </div>
                    <div class="flex items-center gap-2">
                      <button type="button" class="h-9 w-9 rounded-full border" @click="decreaseBarQty(item.id)">-</button>
                      <span class="w-6 text-center font-bold">{{ item.qty }}</span>
                      <button type="button" class="h-9 w-9 rounded-full border" @click="increaseBarQty(item.id)">+</button>
                      <button type="button" class="rounded-lg border px-2 py-1 text-xs" @click="removeBarProduct(item.id)">Rimuovi</button>
                    </div>
                  </div>
                </article>
                <div class="rounded-3xl bg-white p-4">
                  <p class="font-semibold">Totale ordine</p>
                  <p class="text-2xl font-black">€ {{ barOrderTotalLabel }}</p>
                </div>
                <div class="grid grid-cols-2 gap-2">
                  <button type="button" class="rounded-xl border border-slate-300 bg-white py-3 font-bold" @click="goBarStep('categories')">Torna al menu</button>
                  <button type="button" class="rounded-xl bg-slate-900 py-3 font-bold text-white" @click="goBarStep('checkout')">Vai al checkout</button>
                </div>
              </section>

              <section v-else-if="barStep === 'checkout'" class="space-y-3">
                <div class="rounded-3xl bg-white p-4">
                  <p class="text-sm text-slate-500">Riepilogo ordine</p>
                  <p class="mt-1 font-semibold">{{ barOrderSummaryLabel }}</p>
                  <p class="mt-2 text-2xl font-black">€ {{ barOrderTotalLabel }}</p>
                  <p class="mt-2 text-sm">Modalità: <strong>{{ barOrderModeLabel }}</strong></p>
                </div>
                <div v-if="barOrderMode === 'seat'" class="grid grid-cols-2 gap-2 text-sm">
                  <input v-model.trim="barDelivery.sector" class="rounded-xl border border-slate-300 bg-white px-3 py-3" placeholder="Settore">
                  <input v-model.trim="barDelivery.row" class="rounded-xl border border-slate-300 bg-white px-3 py-3" placeholder="Fila">
                  <input v-model.trim="barDelivery.seat" class="rounded-xl border border-slate-300 bg-white px-3 py-3" placeholder="Posto">
                  <input v-model.trim="barDelivery.notes" class="rounded-xl border border-slate-300 bg-white px-3 py-3" placeholder="Note">
                </div>
                <textarea v-else v-model.trim="barDelivery.notes" rows="3" class="w-full rounded-xl border border-slate-300 bg-white px-3 py-3 text-sm" placeholder="Note (opzionale)"></textarea>
                <button type="button" class="w-full rounded-2xl bg-slate-900 py-4 text-lg font-black text-white" @click="goBarStep('payment')">Vai al pagamento</button>
              </section>

              <section v-else-if="barStep === 'payment'" class="space-y-4">
                <div class="rounded-3xl bg-white p-4">
                  <p class="text-sm text-slate-500">Riepilogo ordine</p>
                  <p class="mt-2 font-semibold">{{ barOrderSummaryLabel }}</p>
                  <p class="mt-2 text-2xl font-black">Totale € {{ barOrderTotalLabel }}</p>
                </div>
                <p v-if="barCheckoutError" class="text-sm text-rose-600">{{ barCheckoutError }}</p>
                <button type="button" class="w-full rounded-2xl bg-emerald-500 py-4 text-lg font-black text-white disabled:opacity-60" :disabled="isBarCheckoutLoading || barTotalCents <= 0" @click="startBarCheckout">
                  {{ isBarCheckoutLoading ? 'Caricamento checkout...' : 'Conferma e paga' }}
                </button>
              </section>

              <section v-else-if="barStep === 'confirmation'" class="space-y-3 rounded-3xl bg-white p-5 text-center">
                <p class="text-2xl font-black text-emerald-600">Ordine ricevuto ✅</p>
                <p class="text-sm">Numero ordine: <strong>#{{ barConfirmedOrderNumber }}</strong></p>
                <p class="text-sm">Prodotti: {{ barOrderSummaryLabel }}</p>
                <p class="text-sm">Stato ordine: <strong>Ricevuto</strong></p>
                <p class="text-sm text-slate-600">Il tuo ordine è stato ricevuto. Ti avviseremo quando sarà pronto.</p>
              </section>
            </div>

            <div v-if="barShowCartBar" class="fixed inset-x-0 bottom-0 z-30 border-t border-slate-200 bg-white p-3">
              <button type="button" class="mx-auto flex w-full max-w-3xl items-center justify-between rounded-2xl bg-slate-900 px-4 py-4 text-white" @click="goBarStep('cart')">
                <span>{{ barCartCount }} articoli</span>
                <strong>€ {{ barOrderTotalLabel }}</strong>
                <span class="text-sm font-bold">Vai al carrello</span>
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <StoryModal
      :open="isStoryModalOpen"
      :current-story="currentStory"
      :show-prev="activeStoryIndex > 0"
      @close="closeStoryModal"
      @next="goToNextStory"
      @prev="goToPrevStory"
    />

    <EarnCoinsModal
      v-model="isEarnModalOpen"
      :event-id="eventId"
      :wallet-target-el="walletTargetEl"
      :wallet-coins="totalCoins"
      :free-retry="freeRetry"
      @coins-earned="addCoinsFromMinigame"
      @consume-free-retry="consumeFreeRetry"
    />

    <CoinCollectAnimation ref="coinAnimationRef" />

    <FansLeaderboardModal
      v-model="isLeaderboardModalOpen"
      :top-list="leaderboardTop3"
      :user-rank="leaderboardUser"
    />

    <FanRegistrationPromptModal
      v-model="isRegistrationPromptOpen"
      :trigger="registrationTrigger"
      :earned-coins="lastEarnedCoins"
      :wallet-coins="totalCoins"
      :reward-label="selectedRewardLabel"
      :on-submit="handleRegistrationSubmit"
      :on-login="handleExistingFanLogin"
      @dismissed="markPromptDismissed"
    />



    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div
          v-if="isProfileOverlayOpen"
          class="fixed inset-0 z-[130] flex"
          role="dialog"
          aria-modal="true"
          aria-label="Profilo tifoso"
          @click.self="closeProfileOverlay"
        >
          <div class="absolute inset-0 bg-slate-950/95 backdrop-blur-md" aria-hidden="true" />

          <div class="relative flex h-full w-full flex-col overflow-hidden">
            <header class="sticky top-0 z-10 border-b border-white/10 bg-slate-950/85 px-4 py-4 backdrop-blur md:px-6">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <p class="text-xs font-bold uppercase tracking-[0.2em] text-amber-300/90">Profilo utente</p>
                  <h2 class="mt-1 text-2xl font-black text-white md:text-3xl">Il tuo account</h2>
                </div>
                <button
                  type="button"
                  class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/20 bg-white/5 text-2xl leading-none text-white transition hover:bg-white/15"
                  aria-label="Chiudi profilo"
                  @click="closeProfileOverlay"
                >
                  ×
                </button>
              </div>
            </header>

            <div class="flex-1 overflow-y-auto px-4 pb-10 pt-5 md:px-6">
              <div class="mx-auto grid w-full max-w-5xl gap-4 lg:grid-cols-2">
                <section class="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-[0_10px_30px_rgba(15,23,42,0.4)] lg:col-span-1">
                  <div class="flex items-center gap-3">
                    <div class="flex h-16 w-16 items-center justify-center overflow-hidden rounded-full border border-amber-300/60 bg-slate-800 text-2xl">
                      <img v-if="profileAvatarUrl" :src="profileAvatarUrl" alt="Avatar profilo" class="h-full w-full object-cover">
                      <span v-else aria-hidden="true">👤</span>
                    </div>
                    <div>
                      <p class="text-xs uppercase tracking-[0.18em] text-slate-300">Nickname</p>
                      <p class="text-xl font-extrabold text-white">{{ profileNickname }}</p>
                    </div>
                  </div>

                  <div class="mt-5 rounded-xl border border-emerald-300/25 bg-emerald-400/10 p-4">
                    <p class="text-xs uppercase tracking-[0.2em] text-emerald-200/90">Saldo monete</p>
                    <p class="mt-1 text-3xl font-black text-emerald-300">{{ totalCoins }} 🪙</p>
                  </div>
                </section>

                <section class="rounded-2xl border border-white/15 bg-white/10 p-5 shadow-[0_10px_30px_rgba(15,23,42,0.4)] lg:col-span-1">
                  <h3 class="text-lg font-extrabold text-white">QR Lotteria MVP</h3>
                  <div v-if="displayLotteryCode" class="mt-3">
                    <p class="text-sm text-slate-300">Codice lotteria:</p>
                    <p class="text-base font-black tracking-wider text-amber-300">{{ displayLotteryCode }}</p>
                    <img
                      :src="displayLotteryQrUrl"
                      alt="QR lotteria utente"
                      class="mt-3 h-44 w-44 rounded-xl border border-white/20 bg-white p-2"
                    >
                    <p class="mt-3 rounded-xl border border-amber-300/25 bg-amber-300/10 px-3 py-2 text-xs font-semibold text-amber-100">
                      Resta fino a fine partita per ritirare il premio.
                    </p>
                  </div>
                  <p v-else class="mt-3 rounded-xl border border-dashed border-white/20 bg-slate-900/35 px-3 py-4 text-sm text-slate-300">
                    Vota l'MVP per ottenere il tuo QR lotteria personale.
                  </p>
                </section>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div
          v-if="isSpendPreviewOpen"
          class="fixed inset-0 z-[120] flex"
          role="dialog"
          aria-modal="true"
          aria-label="Spendi monete"
          @click.self="closeSpendPreview"
        >
          <div class="absolute inset-0 bg-slate-950/90 backdrop-blur-sm" aria-hidden="true" />

          <div class="relative flex h-full w-full flex-col overflow-hidden">
            <header class="sticky top-0 z-10 border-b border-white/10 bg-slate-950/85 px-4 py-4 backdrop-blur md:px-6">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <h2 class="text-2xl font-black text-white md:text-3xl">Spendi Monete</h2>
                  <p class="mt-1 text-sm text-slate-300 md:text-base">I guest possono vedere il catalogo, ma per riscattare serve il profilo tifoso.</p>
                </div>
                <button
                  type="button"
                  class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/20 bg-white/5 text-2xl leading-none text-white transition hover:bg-white/15"
                  aria-label="Chiudi modale Spendi Monete"
                  @click="closeSpendPreview"
                >
                  ×
                </button>
              </div>
            </header>

            <div class="flex-1 overflow-y-auto px-4 pb-8 pt-5 md:px-6">
              <div class="mx-auto max-w-6xl">
                <section class="mystery-box rounded-2xl border border-violet-200/25 bg-violet-500/10 p-4 shadow-[0_10px_28px_rgba(76,29,149,0.35)]">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <p class="text-xs font-bold uppercase tracking-[0.2em] text-violet-200">NUOVO</p>
                      <h3 class="mt-1 text-2xl font-black text-white">Mystery Box</h3>
                      <p class="mt-1 text-sm text-slate-200">Costo apertura: {{ MYSTERY_BOX_COST }} 🪙</p>
                    </div>
                    <button
                      type="button"
                      class="mystery-box__open-btn"
                      :disabled="isOpeningMysteryBox || !canOpenMysteryBox"
                      @click="openMysteryBox"
                    >
                      {{ mysteryBoxButtonLabel }}
                    </button>
                  </div>

                  <div ref="mysteryBoxEl" class="mystery-box__chest" :class="mysteryBoxAnimationClass" aria-hidden="true">📦</div>

                  <p v-if="isMysteryBoxCooldownActive" class="mt-2 text-sm font-semibold text-amber-200">
                    Potrai aprire un'altra mystery box tra {{ mysteryBoxCooldownLabel }}.
                  </p>
                  <p v-else-if="totalCoins < MYSTERY_BOX_COST" class="mt-2 text-sm font-semibold text-rose-200">Monete insufficienti per aprire il box.</p>
                  <p v-if="mysteryBoxStatusText" class="mt-3 text-sm text-slate-200">{{ mysteryBoxStatusText }}</p>

                  <Transition name="reveal-fade">
                    <div v-if="mysteryBoxReward" class="mystery-box__reward">
                      <p class="text-xs font-bold uppercase tracking-[0.2em] text-violet-200">Hai trovato</p>
                      <p class="mt-1 text-xl font-black text-white">{{ mysteryBoxReward.label }}</p>
                      <button type="button" class="mt-3 rounded-full border border-violet-200/40 bg-violet-400/20 px-4 py-2 text-sm font-bold text-white" :disabled="isOpeningMysteryBox || !canOpenMysteryBox" @click="openMysteryBox">
                        Apri un’altra
                      </button>
                    </div>
                  </Transition>
                </section>

                <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <button
                    v-for="coupon in spendCouponPreview"
                    :key="coupon.id"
                    type="button"
                    class="group rounded-2xl border border-white/15 bg-white/10 p-4 text-left shadow-[0_10px_28px_rgba(15,23,42,0.45)] backdrop-blur transition hover:-translate-y-0.5 hover:bg-white/15"
                    @click="attemptRedeem(coupon.id, coupon.cost, coupon.label)"
                  >
                    <div class="flex items-start justify-between gap-2">
                      <span class="text-2xl" aria-hidden="true">🎟️</span>
                      <span class="rounded-full border border-amber-300/40 bg-amber-300/15 px-2 py-0.5 text-xs font-bold text-amber-200">
                        {{ coupon.cost }} 🪙
                      </span>
                    </div>
                    <h3 class="mt-3 text-lg font-extrabold text-white">{{ coupon.label }}</h3>
                    <p class="mt-1 text-sm text-slate-300">{{ coupon.description }}</p>
                    <p class="mt-4 text-xs font-semibold uppercase tracking-wide text-emerald-300">Riscatta</p>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <EventFeedbackModal
      v-model="isFeedbackModalOpen"
      :event-id="props.eventId"
      :feedback-survey="props.activeEvent?.feedback_survey ?? props.activeEvent?.feedbackSurvey"
      @submitted="handleFeedbackSubmitted"
    />

  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import EarnCoinsModal from '../components/EarnCoinsModal.vue';
import CoinCollectAnimation from '../components/CoinCollectAnimation.vue';
import EventFeedbackModal from '../components/EventFeedbackModal.vue';
import ExperienceFeedbackCta from '../components/ExperienceFeedbackCta.vue';
import FansLeaderboardModal from '../components/FansLeaderboardModal.vue';
import FeatureCard from '../components/FeatureCard.vue';
import FanRegistrationPromptModal from '../components/FanRegistrationPromptModal.vue';
import LiveHeader from '../components/LiveHeader.vue';
import SponsorsMarquee from '../components/SponsorsMarquee.vue';
import StoriesBar from '../components/StoriesBar.vue';
import StoryModal from '../components/StoryModal.vue';
import { apiClient, fetchFanProfile, fetchVoteStatus, redeemFanReward, registerFanProfile, syncGuestCoins } from '../api';
import { getOrCreateDeviceId } from '../deviceId';

const anonymousAvatarSvg = encodeURIComponent(
  `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 320 220'>
    <defs>
      <linearGradient id='bg' x1='0' x2='1' y1='0' y2='1'>
        <stop offset='0%' stop-color='#1e293b'/>
        <stop offset='100%' stop-color='#0f172a'/>
      </linearGradient>
      <linearGradient id='ring' x1='0' x2='1' y1='0' y2='1'>
        <stop offset='0%' stop-color='#fde68a'/>
        <stop offset='100%' stop-color='#f97316'/>
      </linearGradient>
    </defs>
    <rect width='320' height='220' fill='url(#bg)'/>
    <circle cx='160' cy='88' r='44' fill='#334155' stroke='url(#ring)' stroke-width='6'/>
    <path d='M74 200c0-40 39-66 86-66s86 26 86 66' fill='#334155' stroke='url(#ring)' stroke-width='6' stroke-linecap='round'/>
  </svg>`,
);
const anonymousAvatarDataUrl = `data:image/svg+xml,${anonymousAvatarSvg}`;
const medals = ['🥇', '🥈', '🥉'];
const rewardLabelMap = {
  'coupon-match': 'Coupon Match Day',
  'coupon-merch': 'Sconto Merch 20%',
  'coupon-upgrade': 'Upgrade posto',
  'coupon-photo': 'Foto Team Edition',
};

const MYSTERY_BOX_COST = 10;
const MYSTERY_BOX_COOLDOWN_MS = 2 * 60 * 1000;
const COIN_BOOST_DURATION_MS = 10 * 60 * 1000;
const storageKeys = {
  wallet: 'wallet:coins',
  coinBoostActive: 'coinBoostActive',
  coinBoostEndTime: 'coinBoostEndTime',
  freeRetry: 'freeRetry',
  mysteryBoxCooldownEndTime: 'mysteryBoxCooldownEndTime',
};
const mysteryRewards = [
  { id: 'coins-6', type: 'coins', amount: 6, label: '+6 monete' },
  { id: 'coins-12', type: 'coins', amount: 12, label: '+12 monete' },
  { id: 'coins-20', type: 'coins', amount: 20, label: '+20 monete' },
  { id: 'boost', type: 'boost', label: 'BOOST MONETE 10 MINUTI' },
  { id: 'free-retry', type: 'freeRetry', label: 'RETRY GRATIS MINIGIOCO' },
];

const props = defineProps({
  eventId: {
    type: Number,
    default: 0,
  },
  teamName: {
    type: String,
    default: 'TEAM',
  },
  teamLogoUrl: {
    type: String,
    default: '',
  },
  isLive: {
    type: Boolean,
    default: true,
  },
  sponsorLine: {
    type: String,
    default: 'Powered by MVP System',
  },
  matchLabel: {
    type: String,
    default: 'Vota • Gioca • Vinci • Partecipa',
  },
  activeEvent: {
    type: Object,
    default: null,
  },
  features: {
    type: Array,
    default: () => [
      {
        id: 'vote-mvp',
        title: 'VOTA L\'MVP',
        subtitle: 'del pubblico',
        description: 'Votazioni aperte',
        actionLabel: 'CLICCA ORA',
        icon: '◔',
        theme: 'orange',
      },
      {
        id: 'game-live',
        title: 'GUADAGNA MONETE',
        subtitle: '',
        description: 'Gioca per guadagnarle',
        actionLabel: 'GIOCA ORA',
        centerBadge: '🪙 0',
        icon: '⚡',
        theme: 'blue',
      },
      {
        id: 'lottery-live',
        title: 'PREMI',
        subtitle: 'utilizza le tue monete',
        description: 'X premi disponibili',
        actionLabel: 'SCOPRI',
        centerBadge: '🎁',
        icon: '🎁',
        theme: 'green',
      },
    ],
  },
  votedPlayerImageUrl: {
    type: String,
    default: '',
  },
  votedPlayerName: {
    type: String,
    default: '',
  },
  votedPlayerLastName: {
    type: String,
    default: '',
  },
  votedPlayerNumber: {
    type: [String, Number],
    default: '',
  },
  gainedCoins: {
    type: Number,
    default: 0,
  },
  registrationPromptSignal: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['feature-select']);
const isEarnModalOpen = ref(false);
const isLeaderboardModalOpen = ref(false);
const isRegistrationPromptOpen = ref(false);
const registrationTrigger = ref('after_vote');
const isRegisteredFan = ref(false);
const fanSessionToken = ref('');
const fanNickname = ref('');
const fanId = ref(0);
const isProfileOverlayOpen = ref(false);
const fanRewardRedemptions = ref([]);
const fanLotteryTicket = ref(null);
const hasVotedMvp = ref(false);
const hasHandledFirstVoteFlow = ref(false);
const shouldOpenProfileAfterAuth = ref(false);
const lastEarnedCoins = ref(0);
const isSpendPreviewOpen = ref(false);
const isBarModalOpen = ref(false);
const isBarCheckoutLoading = ref(false);
const barCheckoutError = ref('');
const barOrderConfirmed = ref(false);
const barProducts = ref([]);
const barCart = ref({});
const barDelivery = ref({ sector: '', row: '', seat: '', notes: '' });
const barStep = ref('start');
const barStepHistory = ref([]);
const barOrderMode = ref('counter');
const selectedCategoryId = ref('all');
const selectedBarProductId = ref(null);
const barDetailQty = ref(1);
const selectedBarExtras = ref([]);
const barConfirmedOrderNumber = ref('—');

const barExtras = [
  { id: 'fries', label: 'Aggiungi patatine', price: 2.5 },
  { id: 'drink', label: 'Aggiungi bibita', price: 2 },
  { id: 'sauce', label: 'Aggiungi salsa', price: 0.5 },
];

const barOrderModes = [
  { id: 'counter', label: 'Ritiro al banco', emoji: '🥤' },
  { id: 'seat', label: 'Consegna al posto', emoji: '🪑' },
];

let stripeClientPromise;

function ensureStripeClient() {
  const key = String(import.meta.env.VITE_STRIPE_PUBLIC_KEY || '').trim();
  if (!key || typeof window === 'undefined') {
    return Promise.resolve(null);
  }
  if (stripeClientPromise) {
    return stripeClientPromise;
  }
  stripeClientPromise = new Promise((resolve) => {
    if (window.Stripe) {
      resolve(window.Stripe(key));
      return;
    }

    const script = document.createElement('script');
    script.src = 'https://js.stripe.com/v3/';
    script.async = true;
    script.onload = () => resolve(window.Stripe ? window.Stripe(key) : null);
    script.onerror = () => resolve(null);
    document.head.appendChild(script);
  });

  return stripeClientPromise;
}
const experienceFormStorageKey = 'experienceFormSubmitted';
const selectedRewardLabel = ref('Premio riscattato');
const spendCouponPreview = [];
const totalCoins = ref(0);
const walletTargetEl = ref(null);
const coinAnimationRef = ref(null);
const mysteryBoxEl = ref(null);
const isOpeningMysteryBox = ref(false);
const mysteryBoxStep = ref('idle');
const mysteryBoxStatusText = ref('');
const mysteryBoxReward = ref(null);
const mysteryBoxCooldownEndTime = ref(0);
const coinBoostActive = ref(false);
const coinBoostEndTime = ref(0);
const boostTick = ref(Date.now());
const freeRetry = ref(0);
const isFeedbackModalOpen = ref(false);
const hasSubmittedFeedback = ref(false);
const sponsors = ref([]);
const leaderboardTop3 = ref([
  { name: 'TIFO1', coins: 320 },
  { name: 'TIFO2', coins: 275 },
  { name: 'TIFO3', coins: 249 },
]);
const leaderboardUser = ref(null);
let leaderboardPollingTimer = null;
let boostCountdownTimer = null;
let isLeaderboardRequestInFlight = false;
let hasPendingLeaderboardRefresh = false;
const hasSponsors = computed(() => sponsors.value.length > 0);
const showSponsorsBox = computed(() => hasSponsors.value);
const showFeedbackCta = computed(() => hasFeedbackSurvey.value);
const hasFeedbackSurvey = computed(() => {
  const survey = props.activeEvent?.feedback_survey ?? props.activeEvent?.feedbackSurvey;
  return Array.isArray(survey?.questions) && survey.questions.length > 0;
});

const boostRemainingMs = computed(() => Math.max(0, coinBoostEndTime.value - boostTick.value));
const coinBoostCountdownLabel = computed(() => {
  const totalSeconds = Math.ceil(boostRemainingMs.value / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
});

const mysteryBoxCooldownRemainingMs = computed(() => Math.max(0, mysteryBoxCooldownEndTime.value - boostTick.value));
const isMysteryBoxCooldownActive = computed(() => mysteryBoxCooldownRemainingMs.value > 0);
const canOpenMysteryBox = computed(() => totalCoins.value >= MYSTERY_BOX_COST && !isMysteryBoxCooldownActive.value);
const mysteryBoxCooldownLabel = computed(() => {
  const totalSeconds = Math.ceil(mysteryBoxCooldownRemainingMs.value / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
});
const mysteryBoxAnimationClass = computed(() => `mystery-box__chest--${mysteryBoxStep.value}`);
const mysteryBoxButtonLabel = computed(() => {
  if (isOpeningMysteryBox.value) return 'Apertura...';
  if (isMysteryBoxCooldownActive.value) return `Attendi ${mysteryBoxCooldownLabel.value}`;
  if (!canOpenMysteryBox.value) return `Servono ${MYSTERY_BOX_COST} 🪙`;
  return 'APRI';
});

const profileAvatarUrl = computed(() => {
  return anonymousAvatarDataUrl;
});

const profileNickname = computed(() => {
  if (fanNickname.value.trim()) {
    return fanNickname.value.trim();
  }
  return isRegisteredFan.value ? 'Tifoso' : 'Guest';
});

const accountRedemptions = computed(() =>
  fanRewardRedemptions.value.map((entry, index) => ({
    id: Number(entry?.id) || index + 1,
    label: rewardLabelMap[String(entry?.reward_key || '').trim()] || String(entry?.reward_key || 'Reward').replace(/-/g, ' '),
    costCoins: Math.max(0, Number(entry?.cost_coins) || 0),
    createdAt: String(entry?.created_at || '').replace('T', ' ').slice(0, 16) || 'Data non disponibile',
  })),
);

const lotteryTicketCode = computed(() => String(fanLotteryTicket.value?.ticket_code || '').trim());
const lotteryQrData = computed(() => String(fanLotteryTicket.value?.qr_data || '').trim());
const displayLotteryCode = computed(() => lotteryTicketCode.value);
const displayLotteryQrUrl = computed(() =>
  lotteryQrData.value
    ? `https://api.qrserver.com/v1/create-qr-code/?size=260x260&data=${encodeURIComponent(lotteryQrData.value)}`
    : '',
);

const voteFeature = computed(() => {
  const baseFeature = props.features.find((feature) => feature.id === 'vote-mvp') || props.features[0];
  const hasVotedPlayer = Boolean(props.votedPlayerImageUrl);
  const playerLastName = String(props.votedPlayerLastName || '').trim();
  const fallbackName = String(props.votedPlayerName || '').trim();
  const titleLabel = hasVotedPlayer ? (playerLastName || fallbackName || baseFeature.title) : baseFeature.title;
  const hasPlayerNumber = String(props.votedPlayerNumber || '').trim() !== '';
  const subtitleLabel = hasVotedPlayer
    ? (hasPlayerNumber ? `#${String(props.votedPlayerNumber).trim()}` : '')
    : baseFeature.subtitle;

  return {
    ...baseFeature,
    title: titleLabel,
    subtitle: subtitleLabel,
    actionLabel: hasVotedPlayer ? 'MODIFICA' : baseFeature.actionLabel,
    previewImageUrl: hasVotedPlayer ? props.votedPlayerImageUrl : anonymousAvatarDataUrl,
    previewImageFit: hasVotedPlayer ? 'contain' : 'cover',
    previewAlt: props.votedPlayerName
      ? `MVP selezionato: ${props.votedPlayerName}`
      : 'Avatar anonimo MVP',
  };
});

function normalizeLeaderboardUser(entry) {
  const rank = Number(entry?.rank ?? entry?.position ?? entry?.user_rank);
  if (!Number.isFinite(rank)) {
    return null;
  }

  return {
    rank,
    coins: Math.max(0, Number(entry?.coins ?? entry?.wallet ?? entry?.score) || 0),
  };
}

function openLeaderboard() {
  isLeaderboardModalOpen.value = true;
  if (!isRegisteredFan.value) {
    openRegistrationPrompt('leaderboard');
  }
}

function syncWalletTargetEl() {
  if (typeof document === 'undefined') {
    return;
  }

  walletTargetEl.value = document.getElementById('wallet-coin-target');
}

function persistPowerUps() {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(storageKeys.coinBoostActive, coinBoostActive.value ? '1' : '0');
  window.localStorage.setItem(storageKeys.coinBoostEndTime, String(coinBoostEndTime.value || 0));
  window.localStorage.setItem(storageKeys.freeRetry, String(Math.max(0, freeRetry.value)));
  window.localStorage.setItem(storageKeys.mysteryBoxCooldownEndTime, String(mysteryBoxCooldownEndTime.value || 0));
}

function hydratePowerUps() {
  if (typeof window === 'undefined') {
    return;
  }

  const storedBoostActive = window.localStorage.getItem(storageKeys.coinBoostActive) === '1';
  const storedBoostEndTime = Number.parseInt(window.localStorage.getItem(storageKeys.coinBoostEndTime) || '0', 10);
  const storedFreeRetry = Number.parseInt(window.localStorage.getItem(storageKeys.freeRetry) || '0', 10);
  const storedMysteryBoxCooldownEndTime = Number.parseInt(window.localStorage.getItem(storageKeys.mysteryBoxCooldownEndTime) || '0', 10);

  coinBoostEndTime.value = Number.isFinite(storedBoostEndTime) ? storedBoostEndTime : 0;
  coinBoostActive.value = storedBoostActive && coinBoostEndTime.value > Date.now();
  if (!coinBoostActive.value) {
    coinBoostEndTime.value = 0;
  }

  freeRetry.value = Math.max(0, Number.isFinite(storedFreeRetry) ? storedFreeRetry : 0);
  mysteryBoxCooldownEndTime.value = Number.isFinite(storedMysteryBoxCooldownEndTime)
    ? Math.max(Date.now(), storedMysteryBoxCooldownEndTime)
    : 0;
  if (mysteryBoxCooldownEndTime.value <= Date.now()) {
    mysteryBoxCooldownEndTime.value = 0;
  }
  persistPowerUps();
}

function tickBoostState() {
  boostTick.value = Date.now();
  if (coinBoostActive.value && boostRemainingMs.value <= 0) {
    coinBoostActive.value = false;
    coinBoostEndTime.value = 0;
    persistPowerUps();
  }
}

onMounted(async () => {
  await loadBarProducts();
  await confirmBarOrderFromQuery();
  if (typeof window === 'undefined') {
    return;
  }

  const stored = Number.parseInt(window.localStorage.getItem(storageKeys.wallet) || '0', 10);
  totalCoins.value = Number.isFinite(stored) && stored > 0 ? stored : 0;
  hydratePowerUps();
  await nextTick();
  syncWalletTargetEl();

  try {
    const parsed = JSON.parse(window.localStorage.getItem(storyStorageKey.value) || '[]');
    seenStoryIds.value = Array.isArray(parsed) ? parsed.filter((id) => Number.isFinite(Number(id))).map((id) => Number(id)) : [];
  } catch (error) {
    seenStoryIds.value = [];
  }

  hasSubmittedFeedback.value = window.localStorage.getItem(experienceFormStorageKey) === '1';

  await loadFanProfile();
  loadEventStories();
  loadSponsors();
  await loadLeaderboardPreview();
  startLeaderboardPolling();

  if (boostCountdownTimer === null) {
    boostCountdownTimer = window.setInterval(() => {
      tickBoostState();
    }, 1000);
  }
});

async function addCoins(amount, options = {}) {
  const parsed = Number(amount) || 0;
  const source = options.source || 'generic';
  const shouldBoost = parsed > 0 && source === 'minigame' && coinBoostActive.value && boostRemainingMs.value > 0;
  const finalAmount = shouldBoost ? parsed * 2 : parsed;

  totalCoins.value = Math.max(0, totalCoins.value + finalAmount);

  if (typeof window !== 'undefined') {
    window.localStorage.setItem(storageKeys.wallet, String(totalCoins.value));
  }

  await nextTick();
  syncWalletTargetEl();

  if (props.eventId) {
    await syncGuestCoins(props.eventId, totalCoins.value);
  }

  if (!isRegisteredFan.value && props.eventId && finalAmount > 0) {
    lastEarnedCoins.value = finalAmount;
  }

  if (props.eventId) {
    await refreshLeaderboardPreview();
  }
}

function addCoinsFromMinigame(amount) {
  return addCoins(amount, { source: 'minigame' });
}

async function refreshLeaderboardPreview() {
  if (!props.eventId) {
    return;
  }

  if (isLeaderboardRequestInFlight) {
    hasPendingLeaderboardRefresh = true;
    return;
  }

  await loadLeaderboardPreview();
}

async function loadLeaderboardPreview() {
  if (!props.eventId || isLeaderboardRequestInFlight) {
    return;
  }

  isLeaderboardRequestInFlight = true;

  try {
    // TODO: sostituire endpoint placeholder con leaderboard ufficiale quando disponibile.
    const { data } = await apiClient.get(`/events/${props.eventId}/coins-leaderboard`);
    const top = Array.isArray(data?.top3) ? data.top3 : Array.isArray(data?.top) ? data.top : [];
    leaderboardTop3.value = top.slice(0, 3).map((entry, index) => ({
      name: String(entry?.name || `TIFO${index + 1}`).slice(0, 10).toUpperCase(),
      coins: Math.max(0, Number(entry?.coins) || 0),
    }));

    if (isRegisteredFan.value) {
      leaderboardUser.value = normalizeLeaderboardUser(data?.userRank ?? data?.user_rank);
    }
  } catch (error) {
    // placeholder fallback, keep static UI preview
  } finally {
    isLeaderboardRequestInFlight = false;

    if (hasPendingLeaderboardRefresh) {
      hasPendingLeaderboardRefresh = false;
      await loadLeaderboardPreview();
    }
  }
}

function stopLeaderboardPolling() {
  if (leaderboardPollingTimer !== null && typeof window !== 'undefined') {
    window.clearInterval(leaderboardPollingTimer);
  }
  leaderboardPollingTimer = null;
}

function startLeaderboardPolling() {
  stopLeaderboardPolling();
  if (typeof window === 'undefined' || !props.eventId) {
    return;
  }

  leaderboardPollingTimer = window.setInterval(() => {
    refreshLeaderboardPreview();
  }, 5000);
}

function openFeedbackModal() {
  if (!hasFeedbackSurvey.value || hasSubmittedFeedback.value) {
    return;
  }
  isFeedbackModalOpen.value = true;
}

function handleFeedbackSubmitted() {
  hasSubmittedFeedback.value = true;
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(experienceFormStorageKey, '1');
  }
}

function normalizeSponsor(item, index) {
  const imageUrl = String(item?.logo_data || item?.image_url || item?.imageUrl || '').trim();
  if (!imageUrl) {
    return null;
  }

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
    sponsors.value = Array.isArray(data)
      ? data
          .map((item, index) => normalizeSponsor(item, index))
          .filter(Boolean)
          .sort((a, b) => {
            if (a.priority !== b.priority) {
              return a.priority - b.priority;
            }
            return a.insertedIndex - b.insertedIndex;
          })
      : [];
  } catch (error) {
    sponsors.value = [];
  }
}

function handleSponsorClick(sponsor) {
  const eventId = Number(props.eventId) || 0;
  const sponsorId = Number(sponsor?.id) || 0;
  if (!eventId || !sponsorId) {
    return;
  }

  apiClient.post(`/events/${eventId}/sponsors/${sponsorId}/click`, {
    device_id: getOrCreateDeviceId(),
    at: new Date().toISOString(),
  }).catch(() => {});
}


const eventStories = ref([]);
const seenStoryIds = ref([]);
const isStoryModalOpen = ref(false);
const activeStoryIndex = ref(0);
const loadingStoryId = ref(0);

const preloadedStoryUrls = new Set();
const preloadPromises = new Map();

const activeStories = computed(() =>
  eventStories.value
    .filter((story) => story && story.is_active !== false)
    .sort((a, b) => (Number(a.order_index) || 0) - (Number(b.order_index) || 0)),
);

const currentStory = computed(() => activeStories.value[activeStoryIndex.value] || null);

const storyStorageKey = computed(() => `mvp:stories:seen:event:${props.eventId || 0}`);

function persistSeenStories() {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(storyStorageKey.value, JSON.stringify(Array.from(new Set(seenStoryIds.value))));
}

function markStorySeen(storyId) {
  if (!storyId || seenStoryIds.value.includes(storyId)) {
    return;
  }
  seenStoryIds.value = [...seenStoryIds.value, storyId];
  persistSeenStories();
}

async function loadEventStories() {
  if (!props.eventId) {
    eventStories.value = [];
    return;
  }
  try {
    const { data } = await apiClient.get(`/events/${props.eventId}/stories`);
    eventStories.value = Array.isArray(data) ? data : [];
  } catch (error) {
    eventStories.value = [];
  }
}

function openStory(index) {
  preloadAndOpenStory(index);
}

function preloadStoryVideo(url) {
  const targetUrl = String(url || '').trim();
  if (!targetUrl) {
    return Promise.resolve();
  }

  if (preloadedStoryUrls.has(targetUrl)) {
    return Promise.resolve();
  }

  if (preloadPromises.has(targetUrl)) {
    return preloadPromises.get(targetUrl);
  }

  const preloadPromise = new Promise((resolve) => {
    if (typeof document === 'undefined') {
      preloadedStoryUrls.add(targetUrl);
      resolve();
      return;
    }

    const preloader = document.createElement('video');
    preloader.preload = 'auto';
    preloader.src = targetUrl;

    const cleanup = () => {
      preloader.removeEventListener('canplaythrough', onReady);
      preloader.removeEventListener('loadeddata', onReady);
      preloader.removeEventListener('error', onReady);
      preloader.removeAttribute('src');
      preloader.load();
    };

    const onReady = () => {
      preloadedStoryUrls.add(targetUrl);
      preloadPromises.delete(targetUrl);
      cleanup();
      resolve();
    };

    preloader.addEventListener('canplaythrough', onReady, { once: true });
    preloader.addEventListener('loadeddata', onReady, { once: true });
    preloader.addEventListener('error', onReady, { once: true });
    preloader.load();
  });

  preloadPromises.set(targetUrl, preloadPromise);
  return preloadPromise;
}

function preloadOtherStories(excludeIndex) {
  activeStories.value.forEach((story, index) => {
    if (!story || index === excludeIndex) {
      return;
    }
    preloadStoryVideo(story.video_url);
  });
}

async function preloadAndOpenStory(index) {
  const safeIndex = Math.max(0, Math.min(index, activeStories.value.length - 1));
  const selectedStory = activeStories.value[safeIndex];
  if (!selectedStory) {
    return;
  }

  loadingStoryId.value = Number(selectedStory.id) || 0;
  await preloadStoryVideo(selectedStory.video_url);
  activeStoryIndex.value = safeIndex;
  isStoryModalOpen.value = true;
  loadingStoryId.value = 0;
  preloadOtherStories(safeIndex);
}

function closeStoryModal() {
  isStoryModalOpen.value = false;
}

function goToNextStory() {
  if (activeStoryIndex.value >= activeStories.value.length - 1) {
    closeStoryModal();
    return;
  }
  activeStoryIndex.value += 1;
}

function goToPrevStory() {
  if (activeStoryIndex.value <= 0) {
    return;
  }
  activeStoryIndex.value -= 1;
}

watch(
  () => props.eventId,
  () => {
    if (typeof window === 'undefined') {
      return;
    }
    try {
      const parsed = JSON.parse(window.localStorage.getItem(storyStorageKey.value) || '[]');
      seenStoryIds.value = Array.isArray(parsed) ? parsed.filter((id) => Number.isFinite(Number(id))).map((id) => Number(id)) : [];
    } catch (error) {
      seenStoryIds.value = [];
    }
    loadFanProfile();
    loadEventStories();
    loadSponsors();
    loadLeaderboardPreview();
    startLeaderboardPolling();
  },
);


watch(() => props.registrationPromptSignal, (value, previous) => {
  if (value !== previous) {
    loadFanProfile().finally(() => {
      handleAfterVoteLotteryFlow();
    });
  }
});

watch([currentStory, isStoryModalOpen], ([story, isOpen]) => {
  if (!isOpen || !story) {
    return;
  }
  markStorySeen(Number(story.id));
});

watch([isStoryModalOpen, isProfileOverlayOpen], ([storyOpen, profileOpen]) => {
  if (typeof document === 'undefined') {
    return;
  }
  document.body.style.overflow = storyOpen || profileOpen ? 'hidden' : '';
});

onBeforeUnmount(() => {
  stopLeaderboardPolling();
  if (typeof window !== 'undefined' && boostCountdownTimer !== null) {
    window.clearInterval(boostCountdownTimer);
    boostCountdownTimer = null;
  }
  if (typeof document !== 'undefined') {
    document.body.style.overflow = '';
  }
  isProfileOverlayOpen.value = false;
});



function markPromptDismissed(trigger) {
  if (typeof window === 'undefined') return;
  window.sessionStorage.setItem(`fan:prompt:${trigger}`, '1');
}

function openRegistrationPrompt(trigger) {
  if (isRegisteredFan.value || typeof window === 'undefined') return;
  if (trigger === 'spend_redeem') {
    registrationTrigger.value = trigger;
    isRegistrationPromptOpen.value = true;
    return;
  }
  const key = `fan:prompt:${trigger}`;
  if (window.sessionStorage.getItem(key) === '1') return;
  registrationTrigger.value = trigger;
  isRegistrationPromptOpen.value = true;
  window.sessionStorage.setItem(key, '1');
}

function handleAfterVoteLotteryFlow() {
  if (!hasVotedMvp.value || hasHandledFirstVoteFlow.value) {
    return;
  }

  hasHandledFirstVoteFlow.value = true;
  if (isRegisteredFan.value) {
    loadFanProfile().finally(() => {
      isProfileOverlayOpen.value = true;
    });
    return;
  }

  shouldOpenProfileAfterAuth.value = true;
  openRegistrationPrompt('after_vote');
}

async function loadFanProfile() {
  if (!props.eventId) return;

  const voteStatus = await fetchVoteStatus(props.eventId);
  hasVotedMvp.value = Boolean(voteStatus?.ok && voteStatus.hasVoted);
  if (!hasVotedMvp.value) {
    hasHandledFirstVoteFlow.value = false;
  }

  const response = await fetchFanProfile(props.eventId);
  if (!response?.ok) return;
  const data = response.data || {};
  isRegisteredFan.value = Boolean(data.registered);
  if (data.session_token) {
    fanSessionToken.value = data.session_token;
  }
  if (data.registered) {
    fanId.value = Number(data.user?.id) || 0;
    fanNickname.value = data.user?.nickname || '';
    totalCoins.value = Math.max(0, Number(data.wallet) || 0);
    leaderboardUser.value = normalizeLeaderboardUser(data.user_rank ?? data.userRank);
    fanRewardRedemptions.value = Array.isArray(data.reward_redemptions) ? data.reward_redemptions : [];
    fanLotteryTicket.value = data.lottery_ticket || null;
  } else if (Number.isFinite(Number(data.guest_coins))) {
    totalCoins.value = Math.max(totalCoins.value, Number(data.guest_coins) || 0);
    fanRewardRedemptions.value = [];
    fanLotteryTicket.value = null;
  }
}


async function handleExistingFanLogin() {
  await loadFanProfile();
  if (!isRegisteredFan.value) {
    return { ok: false, message: 'Impossibile trovare un profilo associato a questo numero.' };
  }
  if (shouldOpenProfileAfterAuth.value) {
    shouldOpenProfileAfterAuth.value = false;
    isProfileOverlayOpen.value = true;
  }
  return { ok: true };
}

async function handleRegistrationSubmit(form) {
  const response = await registerFanProfile({
    event_id: props.eventId,
    nickname: form.nickname,
    gender: form.gender,
    phone: form.phone,
    accepted_terms: form.acceptedTerms,
    guest_coins: totalCoins.value,
    enter_lottery: form.trigger === 'after_vote',
  });
  if (!response?.ok) {
    return { ok: false, message: response.message };
  }
  isRegisteredFan.value = true;
  fanId.value = Number(response.data?.user?.id) || 0;
  fanNickname.value = response.data?.user?.nickname || '';
  totalCoins.value = Math.max(0, Number(response.data?.wallet) || totalCoins.value);
  isRegistrationPromptOpen.value = false;
  await loadLeaderboardPreview();
  await loadFanProfile();
  if (shouldOpenProfileAfterAuth.value) {
    shouldOpenProfileAfterAuth.value = false;
    isProfileOverlayOpen.value = true;
  }
  return { ok: true, wallet: totalCoins.value };
}

function openProfileOverlay() {
  if (!isRegisteredFan.value) {
    openRegistrationPrompt('profile_overlay');
    return;
  }
  isProfileOverlayOpen.value = true;
}

function closeProfileOverlay() {
  isProfileOverlayOpen.value = false;
}

function randomMysteryReward() {
  const index = Math.floor(Math.random() * mysteryRewards.length);
  return mysteryRewards[index];
}

function wait(ms) {
  return new Promise((resolve) => {
    if (typeof window === 'undefined') {
      resolve();
      return;
    }
    window.setTimeout(resolve, ms);
  });
}

async function creditMysteryCoins(amount) {
  await addCoins(amount, { source: 'mystery-box' });

  if (coinAnimationRef.value?.play && mysteryBoxEl.value && walletTargetEl.value) {
    await coinAnimationRef.value.play({
      fromEl: mysteryBoxEl.value,
      toEl: walletTargetEl.value,
      count: 16,
      amount,
    });
  }
}

function activateCoinBoost() {
  coinBoostActive.value = true;
  coinBoostEndTime.value = Date.now() + COIN_BOOST_DURATION_MS;
  boostTick.value = Date.now();
  persistPowerUps();
}

function grantFreeRetry() {
  freeRetry.value = 1;
  persistPowerUps();
}

function consumeFreeRetry() {
  if (freeRetry.value <= 0) {
    return;
  }
  freeRetry.value = 0;
  persistPowerUps();
}

async function executeMysteryReward(reward) {
  if (!reward) {
    return;
  }

  if (reward.type === 'coins') {
    await creditMysteryCoins(reward.amount || 0);
    return;
  }

  if (reward.type === 'boost') {
    activateCoinBoost();
    return;
  }

  if (reward.type === 'freeRetry') {
    grantFreeRetry();
  }
}

async function openMysteryBox() {
  if (isOpeningMysteryBox.value || !canOpenMysteryBox.value) {
    return;
  }

  isOpeningMysteryBox.value = true;
  mysteryBoxReward.value = null;
  mysteryBoxStep.value = 'opening';
  mysteryBoxStatusText.value = 'Apertura in corso...';

  await addCoins(-MYSTERY_BOX_COST, { source: 'mystery-box' });
  await wait(900);

  mysteryBoxStep.value = 'revealing';
  mysteryBoxStatusText.value = 'Rivelazione premio...';
  await wait(850);

  const reward = randomMysteryReward();
  mysteryBoxReward.value = reward;
  mysteryBoxStatusText.value = '';

  await executeMysteryReward(reward);

  mysteryBoxCooldownEndTime.value = Date.now() + MYSTERY_BOX_COOLDOWN_MS;
  persistPowerUps();

  mysteryBoxStep.value = 'idle';
  isOpeningMysteryBox.value = false;
}

function openSpendPreview() {
  isSpendPreviewOpen.value = true;
  mysteryBoxStep.value = 'idle';
  mysteryBoxStatusText.value = '';
}

function closeSpendPreview() {
  isSpendPreviewOpen.value = false;
}

async function attemptRedeem(rewardKey, costCoins, rewardLabel) {
  if (!isRegisteredFan.value) {
    selectedRewardLabel.value = rewardLabel || `${String(rewardKey).replace('-', ' ').toUpperCase()} · ${costCoins} 🪙`;
    openRegistrationPrompt('spend_redeem');
    return;
  }
  const response = await redeemFanReward(props.eventId, rewardKey, costCoins);
  if (response?.ok) {
    totalCoins.value = Number(response.data?.wallet) || totalCoins.value;
  }
}


const barCategoryImageMap = {
  Panini: 'https://images.unsplash.com/photo-1568901346375-23c9450c58cd?auto=format&fit=crop&w=1000&q=80',
  Bibite: 'https://images.unsplash.com/photo-1596803244618-8ea0578f81f0?auto=format&fit=crop&w=1000&q=80',
  Patatine: 'https://images.unsplash.com/photo-1576107232684-1279f390859f?auto=format&fit=crop&w=1000&q=80',
  Snack: 'https://images.unsplash.com/photo-1515003197210-e0cd71810b5f?auto=format&fit=crop&w=1000&q=80',
  Dolci: 'https://images.unsplash.com/photo-1551024506-0bccd828d307?auto=format&fit=crop&w=1000&q=80',
  'Menu combo': 'https://images.unsplash.com/photo-1561758033-d89a9ad46330?auto=format&fit=crop&w=1000&q=80',
};

const productImageFallback = 'https://images.unsplash.com/photo-1550547660-d9450f859349?auto=format&fit=crop&w=1000&q=80';

const normalizedBarProducts = computed(() => {
  return barProducts.value.map((product) => {
    const name = String(product?.name || 'Prodotto BAR');
    const lower = name.toLowerCase();
    let category = String(product?.category || '').trim();
    if (!category) {
      category = 'Snack';
      if (lower.includes('panin') || lower.includes('burger') || lower.includes('hot dog')) category = 'Panini';
      else if (lower.includes('cola') || lower.includes('bibita') || lower.includes('acqua') || lower.includes('drink')) category = 'Bibite';
      else if (lower.includes('patatin')) category = 'Patatine';
      else if (lower.includes('dolc') || lower.includes('cookie') || lower.includes('torta')) category = 'Dolci';
      else if (lower.includes('menu') || lower.includes('combo')) category = 'Menu combo';
    }

    const image = product?.image_url || product?.image || product?.category_image_url || barCategoryImageMap[category] || productImageFallback;
    const badge = Number(product?.is_best_seller) ? 'Più venduto' : (lower.includes('menu') ? 'Combo' : 'Più venduto');
    return { ...product, category, image, categoryImage: product?.category_image_url || barCategoryImageMap[category] || image, badge };
  });
});

const barCategories = computed(() => {
  const map = new Map();
  for (const p of normalizedBarProducts.value) {
    if (!map.has(p.category)) {
      map.set(p.category, { id: p.category, name: p.category, image: p.categoryImage || p.image || productImageFallback });
    }
  }
  return Array.from(map.values());
});

const barProductsByCategory = computed(() => {
  return normalizedBarProducts.value.filter((p) => selectedCategoryId.value === 'all' || p.category === selectedCategoryId.value);
});

const selectedBarProduct = computed(() => normalizedBarProducts.value.find((p) => String(p.id) === String(selectedBarProductId.value)) || null);
const barUpsellProduct = computed(() => normalizedBarProducts.value.find((p) => p.category === 'Patatine') || normalizedBarProducts.value[0] || null);

const barCartCount = computed(() => Object.values(barCart.value).reduce((sum, q) => sum + Math.max(0, Number(q) || 0), 0));
const barCartItems = computed(() => {
  return normalizedBarProducts.value
    .map((p) => ({ ...p, qty: Math.max(0, Number(barCart.value?.[p.id] || 0)) }))
    .filter((p) => p.qty > 0);
});

const barTotalCents = computed(() => {
  return normalizedBarProducts.value.reduce((sum, product) => {
    const qty = Number(barCart.value?.[product.id] || 0);
    if (!Number.isFinite(qty) || qty <= 0) return sum;
    return sum + qty * Number(product.price_cents || 0);
  }, 0);
});

const barOrderTotalLabel = computed(() => (barTotalCents.value / 100).toFixed(2));
const barOrderModeLabel = computed(() => (barOrderMode.value === 'seat' ? 'Consegna al posto' : 'Ritiro al banco'));
const barStepTitle = computed(() => ({
  start: 'Inizio ordine',
  categories: 'Categorie',
  products: selectedCategoryId.value || 'Prodotti',
  detail: 'Dettaglio prodotto',
  upsell: 'Suggerimenti',
  cart: 'Carrello',
  checkout: 'Checkout',
  payment: 'Pagamento',
  confirmation: 'Conferma ordine',
}[barStep.value] || 'Ordina dal BAR'));
const barCanGoBack = computed(() => !['start', 'confirmation'].includes(barStep.value));
const barShowCartBar = computed(() => !['start', 'confirmation', 'cart', 'checkout', 'payment'].includes(barStep.value) && barCartCount.value > 0);

const barOrderSummaryLabel = computed(() => {
  const labels = [];
  for (const product of normalizedBarProducts.value) {
    const qty = Number(barCart.value?.[product.id] || 0);
    if (qty > 0) labels.push(`${product.name} x${qty}`);
  }
  return labels.length ? labels.join(' · ') : 'Nessun prodotto selezionato';
});

function goBarStep(nextStep) {
  if (barStep.value !== nextStep) barStepHistory.value.push(barStep.value);
  barStep.value = nextStep;
}

function goBackBarStep() {
  const prev = barStepHistory.value.pop();
  barStep.value = prev || 'start';
}

function openBarOrdering() {
  barOrderConfirmed.value = false;
  barCheckoutError.value = '';
  barStep.value = 'start';
  barStepHistory.value = [];
  isBarModalOpen.value = true;
}

function openBarCategory(categoryId) {
  selectedCategoryId.value = categoryId;
  goBarStep('products');
}

function openProductDetail(productId) {
  selectedBarProductId.value = productId;
  barDetailQty.value = 1;
  selectedBarExtras.value = [];
  goBarStep('detail');
}

function toggleBarExtra(extraId) {
  selectedBarExtras.value = selectedBarExtras.value.includes(extraId)
    ? selectedBarExtras.value.filter((id) => id !== extraId)
    : [...selectedBarExtras.value, extraId];
}

function changeDetailQty(delta) {
  barDetailQty.value = Math.max(1, barDetailQty.value + delta);
}

function increaseBarQty(productId) {
  const current = Number(barCart.value?.[productId] || 0);
  barCart.value = { ...barCart.value, [productId]: current + 1 };
}

function decreaseBarQty(productId) {
  const current = Number(barCart.value?.[productId] || 0);
  if (current <= 0) return;
  barCart.value = { ...barCart.value, [productId]: current - 1 };
}

function removeBarProduct(productId) {
  barCart.value = { ...barCart.value, [productId]: 0 };
}

function addProductFromDetail() {
  if (!selectedBarProduct.value) return;
  const productId = selectedBarProduct.value.id;
  const current = Number(barCart.value?.[productId] || 0);
  barCart.value = { ...barCart.value, [productId]: current + barDetailQty.value };
  goBarStep('upsell');
}

function addUpsellProduct() {
  if (barUpsellProduct.value) increaseBarQty(barUpsellProduct.value.id);
  goBarStep('products');
}

async function loadBarProducts() {
  try {
    const { data } = await apiClient.get('/bar/products');
    barProducts.value = Array.isArray(data) ? data : [];
  } catch (error) {
    barProducts.value = [];
  }
}

async function startBarCheckout() {
  barCheckoutError.value = '';
  if (barOrderMode.value === 'seat' && (!barDelivery.value.sector || !barDelivery.value.row || !barDelivery.value.seat)) {
    barCheckoutError.value = 'Inserisci settore, fila e posto.';
    return;
  }

  const items = Object.entries(barCart.value)
    .map(([product_id, quantity]) => ({ product_id, quantity: Number(quantity) }))
    .filter((entry) => entry.quantity > 0);

  if (!items.length) {
    barCheckoutError.value = 'Seleziona almeno un prodotto.';
    return;
  }

  isBarCheckoutLoading.value = true;
  try {
    const { data } = await apiClient.post('/bar/checkout/session', {
      items,
      sector: barOrderMode.value === 'seat' ? barDelivery.value.sector : '',
      row: barOrderMode.value === 'seat' ? barDelivery.value.row : '',
      seat: barOrderMode.value === 'seat' ? barDelivery.value.seat : '',
      notes: barDelivery.value.notes,
    });

    if (!data?.checkout_url || !data?.session_id) {
      barCheckoutError.value = 'Checkout non disponibile.';
      return;
    }

    if (typeof window !== 'undefined') {
      window.localStorage.setItem('bar:last_session_id', String(data.session_id));
    }

    const stripe = await ensureStripeClient();
    if (stripe && data?.session_id) {
      const result = await stripe.redirectToCheckout({ sessionId: data.session_id });
      if (result?.error?.message) barCheckoutError.value = result.error.message;
      return;
    }

    if (typeof window !== 'undefined' && data?.checkout_url) {
      window.location.href = data.checkout_url;
    }
  } catch (error) {
    barCheckoutError.value = error?.response?.data?.message || 'Errore durante la creazione del pagamento.';
  } finally {
    isBarCheckoutLoading.value = false;
  }
}

async function confirmBarOrderFromQuery() {
  if (typeof window === 'undefined') return;

  const params = new URLSearchParams(window.location.search || '');
  const success = params.get('barOrderSuccess');
  const sessionFromQuery = params.get('session_id');
  const storedSession = window.localStorage.getItem('bar:last_session_id') || '';
  const sessionId = sessionFromQuery || storedSession;

  if (success !== '1' || !sessionId) return;

  try {
    const { data } = await apiClient.post('/bar/checkout/confirm', { session_id: sessionId });
    if (data?.confirmed) {
      barOrderConfirmed.value = true;
      barConfirmedOrderNumber.value = String(data?.order_id || sessionId).slice(-6);
      isBarModalOpen.value = true;
      barStep.value = 'confirmation';
      barCart.value = {};
      barDelivery.value = { sector: '', row: '', seat: '', notes: '' };
    }
  } catch (error) {
    // silent
  }
}

function onFeatureSelect(featureId) {
  if (featureId === 'game-live') {
    isEarnModalOpen.value = true;
    return;
  }

  if (featureId === 'leaderboard-live') {
    openLeaderboard();
    return;
  }

  emit('feature-select', featureId);
}
</script>

<style scoped>
.live-experience {
  background:
    radial-gradient(circle at 50% -15%, rgba(59, 130, 246, 0.35), transparent 55%),
    radial-gradient(circle at 85% 26%, rgba(251, 191, 36, 0.32), transparent 38%),
    radial-gradient(circle at 14% 28%, rgba(255, 255, 255, 0.21), transparent 28%),
    linear-gradient(180deg, #030712 0%, #0f172a 45%, #030712 100%);
}

.arena-bg {
  background:
    radial-gradient(circle at 20% 24%, rgba(255, 255, 255, 0.46), transparent 13%),
    radial-gradient(circle at 83% 27%, rgba(255, 180, 92, 0.5), transparent 16%),
    radial-gradient(circle at 50% 68%, rgba(255, 255, 255, 0.16), transparent 35%),
    linear-gradient(180deg, rgba(15, 23, 42, 0.72) 0%, rgba(2, 6, 23, 0.85) 100%);
  filter: blur(1.8px);
}

.vignette {
  background: radial-gradient(circle at center, transparent 45%, rgba(2, 6, 23, 0.8) 100%);
}

.feedback-area {
  padding-top: clamp(0.5rem, 1.5vh, 0.8rem);
  padding-bottom: clamp(0.5rem, 1.5vh, 0.8rem);
  gap: clamp(0.5rem, 1.5vh, 0.8rem);
}

.feedback-area__cta-slot,
.feedback-area__sponsors-slot {
  width: 100%;
}

.mini-feature {
  position: relative;
  display: flex;
  min-height: 106px;
  flex: 1;
  cursor: pointer;
  flex-direction: column;
  justify-content: space-between;
  overflow: hidden;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.35);
  padding: 0.5rem;
  transition: transform 0.15s ease;
}

.mini-feature:active {
  transform: scale(0.99);
}

.mini-feature--earn {
  background: linear-gradient(180deg, rgba(96, 165, 250, 0.26), rgba(23, 37, 84, 0.9));
  box-shadow: 0 0 22px rgba(59, 130, 246, 0.44);
}

.mini-feature--spend {
  background: linear-gradient(180deg, rgba(110, 231, 183, 0.24), rgba(20, 83, 45, 0.9));
  box-shadow: 0 0 22px rgba(34, 197, 94, 0.35);
}

.mini-feature__content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: rgba(0, 0, 0, 0.22);
  margin-bottom: 0.45rem;
}

.mini-feature__coins {
  font-size: clamp(1.3rem, 5vw, 2rem);
  font-weight: 900;
  letter-spacing: -0.02em;
}

.mini-feature__boost {
  margin-top: 0.2rem;
  font-size: 0.66rem;
  font-weight: 900;
  letter-spacing: 0.06em;
  color: #fde047;
}

.mystery-box__open-btn {
  border-radius: 9999px;
  border: 1px solid rgba(196, 181, 253, 0.45);
  background: linear-gradient(180deg, rgba(167, 139, 250, 0.4), rgba(126, 34, 206, 0.5));
  padding: 0.5rem 1.1rem;
  font-weight: 900;
  color: #fff;
}

.mystery-box__open-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.mystery-box__chest {
  margin-top: 0.9rem;
  display: flex;
  justify-content: center;
  font-size: clamp(3rem, 8vw, 4.4rem);
  transform-origin: center;
}

.mystery-box__chest--idle {
  animation: mystery-idle 2.1s ease-in-out infinite;
}

.mystery-box__chest--opening {
  animation: mystery-opening 0.6s ease-in-out infinite;
}

.mystery-box__chest--revealing {
  animation: mystery-reveal 0.45s ease forwards;
}

.mystery-box__reward {
  margin-top: 0.8rem;
  border-radius: 0.95rem;
  border: 1px solid rgba(221, 214, 254, 0.35);
  background: rgba(46, 16, 101, 0.35);
  padding: 0.9rem;
  text-align: center;
}

@keyframes mystery-idle {
  0%,
  100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}

@keyframes mystery-opening {
  0%,
  100% { transform: rotate(0deg) scale(1); }
  25% { transform: rotate(-8deg) scale(1.02); }
  75% { transform: rotate(8deg) scale(1.02); }
}

@keyframes mystery-reveal {
  from { transform: scale(0.88); opacity: 0.65; }
  to { transform: scale(1.08); opacity: 1; }
}

.reveal-fade-enter-active,
.reveal-fade-leave-active {
  transition: opacity 0.25s ease;
}

.reveal-fade-enter-from,
.reveal-fade-leave-to {
  opacity: 0;
}

.mini-feature__icons {
  font-size: clamp(1.2rem, 4.5vw, 1.7rem);
  letter-spacing: 0.2rem;
}

.mini-feature__cta {
  width: 100%;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.25);
  padding: 0.38rem 0.5rem;
  font-size: clamp(0.72rem, 2.2vw, 0.88rem);
  font-weight: 900;
  letter-spacing: 0.04em;
  color: #fff;
}

.mini-feature__cta--earn {
  background: linear-gradient(180deg, #60a5fa, #1d4ed8);
}

.mini-feature__cta--spend {
  background: linear-gradient(180deg, #a3e635, #15803d);
}

.leaderboard-preview {
  display: flex;
  min-height: 220px;
  cursor: pointer;
  flex-direction: column;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.35);
  background: linear-gradient(180deg, rgba(250, 204, 21, 0.18), rgba(120, 53, 15, 0.84));
  padding: 0.55rem;
  box-shadow: 0 0 22px rgba(245, 158, 11, 0.35);
}

.leaderboard-preview__title {
  text-align: center;
  font-size: clamp(0.7rem, 2vw, 0.9rem);
  font-weight: 900;
}

.leaderboard-preview__list {
  margin-top: 0.4rem;
  display: flex;
  flex-direction: column;
  gap: 0.33rem;
}

.leaderboard-preview__item {
  display: flex;
  justify-content: space-between;
  border-radius: 0.4rem;
  background: rgba(2, 6, 23, 0.4);
  padding: 0.24rem 0.34rem;
  font-size: clamp(0.65rem, 1.95vw, 0.8rem);
  font-weight: 700;
}

.leaderboard-preview__you {
  margin-top: auto;
  border-radius: 0.45rem;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(2, 6, 23, 0.35);
  padding: 0.28rem;
  text-align: center;
  font-size: clamp(0.62rem, 1.8vw, 0.74rem);
  font-weight: 700;
}

.leaderboard-preview__cta {
  margin-top: 0.4rem;
  width: 100%;
  border-radius: 0.45rem;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: linear-gradient(180deg, #f59e0b, #b45309);
  padding: 0.36rem 0.45rem;
  font-size: clamp(0.66rem, 1.9vw, 0.78rem);
  font-weight: 900;
}
</style>

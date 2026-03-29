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
                <p v-if="nextGameMultiplier > 1" class="mini-feature__boost mini-feature__boost--next">x{{ nextGameMultiplier }} prossima vincita</p>
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
      v-if="isBarFeatureEnabled"
      type="button"
      class="fixed bottom-5 right-4 z-[140] inline-flex items-center gap-2 rounded-full border border-amber-200/60 bg-amber-400 px-4 py-3 text-sm font-black uppercase tracking-wide text-slate-950 shadow-[0_12px_30px_rgba(251,191,36,0.45)]"
      @click="openBarOrdering"
    >
      🍺 Bar
    </button>

    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div v-if="isBarFeatureEnabled && isBarModalOpen" class="fixed inset-0 z-[220] bg-slate-100 text-slate-900">
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
                <div class="grid grid-cols-2 gap-3">
                  <button
                    v-for="mode in barOrderModes"
                    :key="mode.id"
                    type="button"
                    class="aspect-square rounded-3xl border border-slate-200 bg-white p-4 text-left shadow-sm transition hover:border-amber-300 hover:shadow-md"
                    :class="barOrderMode === mode.id ? 'ring-2 ring-amber-400' : ''"
                    @click="selectBarMode(mode.id)"
                  >
                    <p class="text-xs font-bold uppercase text-slate-500">{{ mode.label }}</p>
                    <p class="mt-3 text-5xl">{{ mode.emoji }}</p>
                  </button>
                </div>
              </section>

              <section v-else-if="barStep === 'categories'" class="space-y-4">
                <article v-for="category in barCategories" :key="category.id" class="overflow-hidden rounded-3xl border border-slate-200 bg-white" @click="openBarCategory(category.id)">
                  <div class="p-4"><p class="text-lg font-black">{{ category.name }}</p></div>
                  <img :src="category.image" :alt="category.name" class="h-40 w-full object-cover">
                </article>
              </section>

              <section v-else-if="barStep === 'products'" class="space-y-4">
                <article v-for="product in barProductsByCategory" :key="product.id" class="rounded-3xl border border-slate-200 bg-white p-4 shadow-sm">
                  <div class="flex items-start gap-3">
                    <img :src="product.image" :alt="product.name" class="h-24 w-24 shrink-0 rounded-2xl object-cover">
                    <div class="min-w-0 flex-1">
                      <p class="text-xl font-black">{{ product.name }}</p>
                      <p class="text-sm text-slate-500">{{ product.description || 'Ricetta del bar.' }}</p>
                      <p class="mt-2 text-lg font-bold">€ {{ (product.price_cents / 100).toFixed(2) }}</p>
                      <span v-if="product.badge" class="mt-2 inline-block rounded-full bg-amber-100 px-2 py-1 text-xs font-bold text-amber-700">{{ product.badge }}</span>
                    </div>
                    <button type="button" class="self-start rounded-xl bg-slate-900 px-4 py-3 text-sm font-bold text-white" @click="openProductDetail(product.id)">Vedi</button>
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
                <h4 class="text-2xl font-black">{{ barSuggestionTitle }}</h4>
                <article v-for="suggestion in barSuggestionItems" :key="`s-${suggestion.id}`" class="rounded-3xl border border-slate-200 bg-white p-4">
                  <img :src="suggestion.image" :alt="suggestion.name" class="h-48 w-full rounded-2xl object-cover">
                  <div class="mt-3 flex items-center justify-between">
                    <div>
                      <p class="text-xl font-black">{{ suggestion.name }}</p>
                      <p class="text-sm text-slate-500">€ {{ (suggestion.price_cents / 100).toFixed(2) }}</p>
                    </div>
                    <button type="button" class="rounded-xl bg-amber-400 px-4 py-2 font-black" @click="addSuggestedProduct(suggestion.id)">Aggiungi</button>
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
                <div v-if="isWheelSpinning" class="mb-4 rounded-xl border border-amber-300/40 bg-amber-300/15 px-4 py-2 text-center text-sm font-bold text-amber-100">
                  Spin in corso... input bloccato
                </div>

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
                      :disabled="isWheelSpinning || isOpeningMysteryBox || !canOpenMysteryBox"
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
                      <button type="button" class="mt-3 rounded-full border border-violet-200/40 bg-violet-400/20 px-4 py-2 text-sm font-bold text-white" :disabled="isWheelSpinning || isOpeningMysteryBox || !canOpenMysteryBox" @click="openMysteryBox">
                        Apri un’altra
                      </button>
                    </div>
                  </Transition>
                </section>

                <section class="wheel-card mt-4 rounded-2xl border border-amber-200/30 bg-amber-500/10 p-4 shadow-[0_10px_28px_rgba(146,64,14,0.35)]">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <p class="text-xs font-bold uppercase tracking-[0.2em] text-amber-200">Ruota della Fortuna</p>
                      <h3 class="mt-1 text-2xl font-black text-white">Ruota della Fortuna</h3>
                      <p class="mt-1 text-sm text-slate-200">Gira la ruota e vinci premi.</p>
                      <p class="mt-1 text-sm text-amber-100">Costo: {{ FORTUNE_WHEEL_COST }} monete</p>
                    </div>
                    <button
                      type="button"
                      class="wheel-card__spin-btn"
                      :disabled="isWheelSpinning || !canSpinWheel"
                      @click="openWheelModal"
                    >
                      {{ canSpinWheel ? 'GIRA' : `Servono ${FORTUNE_WHEEL_COST} 🪙` }}
                    </button>
                  </div>
                </section>

                <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <button
                    v-for="coupon in spendCouponPreview"
                    :key="coupon.id"
                    type="button"
                    class="group rounded-2xl border border-white/15 bg-white/10 p-4 text-left shadow-[0_10px_28px_rgba(15,23,42,0.45)] backdrop-blur transition hover:-translate-y-0.5 hover:bg-white/15"
                    :disabled="isWheelSpinning"
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

    <Teleport to="body">
      <Transition name="earn-modal-fade">
        <div
          v-if="isWheelModalOpen"
          class="wheel-modal fixed inset-0 z-[160] flex items-center justify-center"
          role="dialog"
          aria-modal="true"
        >
          <div class="wheel-modal__content w-full h-full px-4 py-6">
            <div class="mx-auto flex h-full w-full max-w-3xl flex-col items-center justify-center">
              <p class="mb-4 rounded-full border border-amber-300/40 bg-amber-300/15 px-4 py-2 text-sm font-black text-amber-100">Saldo: {{ totalCoins }} 🪙</p>
              <div class="wheel-wrap wheel-wrap--modal" :class="{ 'wheel-wrap--result': Boolean(wheelResult) }">
                <!-- Pointer indicator -->
                <div class="wheel-pointer" aria-hidden="true">
                  <div class="wheel-pointer__triangle"></div>
                  <div class="wheel-pointer__stem"></div>
                </div>

                <!-- Outer decorative ring -->
                <div class="wheel-outer-ring" :class="{ 'wheel-outer-ring--jackpot': wheelResult?.type === 'jackpot' }"></div>

                <div
                  ref="wheelEl"
                  class="fortune-wheel fortune-wheel--modal"
                  :class="{ 'fortune-wheel--spinning': isWheelSpinning, 'fortune-wheel--jackpot': wheelResult?.type === 'jackpot' }"
                  :style="{ transform: `rotate(${wheelRotationDeg}deg)` }"
                >
                  <!-- Segment divider lines -->
                  <svg class="fortune-wheel__dividers" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
                    <line v-for="i in 8" :key="i"
                      x1="50" y1="50" x2="50" y2="0"
                      stroke="rgba(255,255,255,0.45)" stroke-width="0.7"
                      :transform="`rotate(${(i - 1) * 45}, 50, 50)`"
                    />
                  </svg>

                  <!-- Segment labels -->
                  <span
                    v-for="(segment, index) in wheelSegments"
                    :key="segment.id"
                    class="fortune-wheel__label"
                    :class="{ 'fortune-wheel__label--winner': wheelWinningIndex === index }"
                    :style="segmentLabelStyle(index)"
                  >
                    {{ segment.shortLabel }}
                  </span>

                  <!-- Center hub -->
                  <div class="fortune-wheel__hub">
                    <span class="fortune-wheel__hub-icon">🍀</span>
                  </div>
                </div>
              </div>

              <Transition name="reveal-fade">
                <div v-if="wheelResult" class="wheel-reveal mt-6 text-center" :class="{ 'wheel-reveal--jackpot': wheelResult.type === 'jackpot' }">
                  <p class="wheel-reveal__text">{{ wheelResultRevealLabel }}</p>
                </div>
              </Transition>

              <div v-if="wheelResult" class="mt-6 flex w-full max-w-sm gap-3">
                <button v-if="canSpinWheel" type="button" class="wheel-card__spin-btn flex-1" @click="spinFortuneWheel">
                  GIRA ANCORA
                </button>
                <button type="button" class="wheel-modal__close-btn flex-1" @click="closeWheelModal">
                  CHIUDI
                </button>
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

    <InAppAiPopup
      v-model="isAiPopupOpen"
      :popup="activeAiPopup"
      @dismiss="handleAiPopupDismiss"
      @cta="handleAiPopupCTA"
    />

  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import EarnCoinsModal from '../components/EarnCoinsModal.vue';
import CoinCollectAnimation from '../components/CoinCollectAnimation.vue';
import EventFeedbackModal from '../components/EventFeedbackModal.vue';
import ExperienceFeedbackCta from '../components/ExperienceFeedbackCta.vue';
import InAppAiPopup from '../components/InAppAiPopup.vue';
import FansLeaderboardModal from '../components/FansLeaderboardModal.vue';
import FeatureCard from '../components/FeatureCard.vue';
import FanRegistrationPromptModal from '../components/FanRegistrationPromptModal.vue';
import LiveHeader from '../components/LiveHeader.vue';
import SponsorsMarquee from '../components/SponsorsMarquee.vue';
import StoriesBar from '../components/StoriesBar.vue';
import StoryModal from '../components/StoryModal.vue';
import { apiClient, fetchFanProfile, fetchVoteStatus, redeemFanReward, registerFanProfile, syncGuestCoins, resolveApiUrl, getOrganizationSlug, generateAIBarUpsell, generateAIPopup, trackAIInteraction } from '../api';
import { getOrCreateDeviceId } from '../deviceId';
import { safeTrackEvent } from '../tracking';
import { endTrackingLifecycle, startTrackingLifecycle, trackAppEvent, trackSectionView, updateTrackingContext } from '../eventTracking';

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
const FORTUNE_WHEEL_COST = 6;
const WHEEL_SEGMENT_DEG = 45;
const WHEEL_SPINS = 5;
const storageKeys = {
  wallet: 'wallet:coins',
  coinBoostActive: 'coinBoostActive',
  coinBoostEndTime: 'coinBoostEndTime',
  freeRetry: 'freeRetry',
  nextGameMultiplier: 'nextGameMultiplier',
  mysteryBoxCooldownEndTime: 'mysteryBoxCooldownEndTime',
};
const mysteryRewards = [
  { id: 'coins-6', type: 'coins', amount: 6, label: '+6 monete' },
  { id: 'coins-12', type: 'coins', amount: 12, label: '+12 monete' },
  { id: 'coins-20', type: 'coins', amount: 20, label: '+20 monete' },
  { id: 'boost', type: 'boost', label: 'BOOST MONETE 10 MINUTI' },
  { id: 'free-retry', type: 'freeRetry', label: 'RETRY GRATIS MINIGIOCO' },
];
const wheelSegments = [
  { id: 'coins-3-a', type: 'coins', amount: 3, label: '+3 monete', shortLabel: '🪙+3', weight: 24 },
  { id: 'coins-5', type: 'coins', amount: 5, label: '+5 monete', shortLabel: '🪙+5', weight: 20 },
  { id: 'coins-8', type: 'coins', amount: 8, label: '+8 monete', shortLabel: '🪙+8', weight: 16 },
  { id: 'coins-12', type: 'coins', amount: 12, label: '+12 monete', shortLabel: '🪙+12', weight: 11 },
  { id: 'free-retry-wheel', type: 'freeRetry', label: 'Retry minigioco', shortLabel: '🔄 Retry', weight: 8 },
  { id: 'x2-next-win', type: 'nextMultiplier', amount: 2, label: 'x2 prossima vincita minigioco', shortLabel: '⚡×2', weight: 10 },
  { id: 'coins-3-b', type: 'coins', amount: 3, label: '+3 monete', shortLabel: '🪙+3', weight: 24 },
  { id: 'jackpot-25', type: 'jackpot', amount: 25, label: 'JACKPOT +25 monete', shortLabel: '💎JACK', weight: 3 },
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
const isAiPopupOpen = ref(false);
const activeAiPopup = ref(null);
const aiPopupHistory = ref([]);
const lastActivityAt = ref(Date.now());
let inactivityTimer = null;
const barOrderConfirmed = ref(false);
const barProducts = ref([]);
const barCategoriesData = ref([]);
const barSuggestionsCache = ref({});
let barCatalogPreloadPromise = null;
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
let wheelTickTimer = null;
let wheelAudioContext = null;

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
const wheelEl = ref(null);
const isOpeningMysteryBox = ref(false);
const mysteryBoxStep = ref('idle');
const mysteryBoxStatusText = ref('');
const mysteryBoxReward = ref(null);
const mysteryBoxCooldownEndTime = ref(0);
const coinBoostActive = ref(false);
const coinBoostEndTime = ref(0);
const boostTick = ref(Date.now());
const freeRetry = ref(0);
const nextGameMultiplier = ref(1);
const isWheelSpinning = ref(false);
const wheelRotationDeg = ref(0);
const wheelResult = ref(null);
const isWheelModalOpen = ref(false);
const wheelWinningIndex = ref(-1);
const isFeedbackModalOpen = ref(false);
const hasSubmittedFeedback = ref(false);
const sponsors = ref([]);
const leaderboardTop3 = ref([
  { name: 'TIFO1', coins: 320 },
  { name: 'TIFO2', coins: 275 },
  { name: 'TIFO3', coins: 249 },
]);
const leaderboardUser = ref(null);
let leaderboardSseSource = null;
let boostCountdownTimer = null;
let isLeaderboardRequestInFlight = false;
let hasPendingLeaderboardRefresh = false;
const hasSponsors = computed(() => sponsors.value.length > 0);
const showSponsorsBox = computed(() => hasSponsors.value);
const showFeedbackCta = computed(() => hasFeedbackSurvey.value);
const isBarFeatureEnabled = computed(() =>
  props.activeEvent?.organization_bar_enabled !== false &&
  props.activeEvent?.organizationBarEnabled !== false,
);
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
const canSpinWheel = computed(() => totalCoins.value >= FORTUNE_WHEEL_COST);
const mysteryBoxButtonLabel = computed(() => {
  if (isOpeningMysteryBox.value) return 'Apertura...';
  if (isMysteryBoxCooldownActive.value) return `Attendi ${mysteryBoxCooldownLabel.value}`;
  if (!canOpenMysteryBox.value) return `Servono ${MYSTERY_BOX_COST} 🪙`;
  return 'APRI';
});
const wheelResultRevealLabel = computed(() => {
  if (!wheelResult.value) return '';
  if (wheelResult.value.type === 'jackpot') return 'JACKPOT';
  if (wheelResult.value.type === 'coins') return `+${wheelResult.value.amount || 0} MONETE`;
  if (wheelResult.value.type === 'freeRetry') return 'RETRY MINIGIOCO';
  if (wheelResult.value.type === 'nextMultiplier') return 'X2 PROSSIMA VINCITA';
  return wheelResult.value.label.toUpperCase();
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
  trackAppEvent('navigation.leaderboard_opened', { modal: 'fans_leaderboard', registered_fan: isRegisteredFan.value }, 'navigation');
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
  window.localStorage.setItem(storageKeys.nextGameMultiplier, String(Math.max(1, nextGameMultiplier.value)));
  window.localStorage.setItem(storageKeys.mysteryBoxCooldownEndTime, String(mysteryBoxCooldownEndTime.value || 0));
}

function hydratePowerUps() {
  if (typeof window === 'undefined') {
    return;
  }

  const storedBoostActive = window.localStorage.getItem(storageKeys.coinBoostActive) === '1';
  const storedBoostEndTime = Number.parseInt(window.localStorage.getItem(storageKeys.coinBoostEndTime) || '0', 10);
  const storedFreeRetry = Number.parseInt(window.localStorage.getItem(storageKeys.freeRetry) || '0', 10);
  const storedMultiplier = Number.parseInt(window.localStorage.getItem(storageKeys.nextGameMultiplier) || '1', 10);
  const storedMysteryBoxCooldownEndTime = Number.parseInt(window.localStorage.getItem(storageKeys.mysteryBoxCooldownEndTime) || '0', 10);

  coinBoostEndTime.value = Number.isFinite(storedBoostEndTime) ? storedBoostEndTime : 0;
  coinBoostActive.value = storedBoostActive && coinBoostEndTime.value > Date.now();
  if (!coinBoostActive.value) {
    coinBoostEndTime.value = 0;
  }

  freeRetry.value = Math.max(0, Number.isFinite(storedFreeRetry) ? storedFreeRetry : 0);
  nextGameMultiplier.value = Math.max(1, Number.isFinite(storedMultiplier) ? storedMultiplier : 1);
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
  if (isBarFeatureEnabled.value) {
    await preloadBarCatalog();
    await confirmBarOrderFromQuery();
  }
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

  startTrackingLifecycle({ eventId: props.eventId, page: 'newui_live_experience', source: 'newui' });
  updateTrackingContext({
    loginState: isRegisteredFan.value ? 'logged_in' : 'guest',
    profileState: isRegisteredFan.value ? 'registered' : 'guest',
  });
  trackSectionView('home.hero', { match_label: matchLabel.value });
  await loadFanProfile();
  loadEventStories();
  loadSponsors();
  await loadLeaderboardPreview();
  startLeaderboardPolling();
  registerUserActivity();

  if (typeof window !== 'undefined') {
    const activityEvents = ['click', 'touchstart', 'keydown'];
    activityEvents.forEach((eventName) => window.addEventListener(eventName, registerUserActivity, { passive: true }));
    inactivityTimer = window.setInterval(() => {
      const idleSeconds = Math.round((Date.now() - lastActivityAt.value) / 1000);
      if (idleSeconds >= 75 && !isAiPopupOpen.value && !isBarModalOpen.value) {
        maybeOpenAiPopup('inactive_user', 'far_esplorare_una_feature', { idleSeconds });
      }
    }, 15000);
  }

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
  const nextMultiplier = parsed > 0 && source === 'minigame' ? Math.max(1, Number(nextGameMultiplier.value) || 1) : 1;
  const finalAmount = parsed * (shouldBoost ? 2 : 1) * nextMultiplier;

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

  if (source === 'minigame' && parsed > 0 && nextMultiplier > 1) {
    nextGameMultiplier.value = 1;
    persistPowerUps();
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
  if (leaderboardSseSource) {
    leaderboardSseSource.close();
    leaderboardSseSource = null;
  }
}

function startLeaderboardPolling() {
  stopLeaderboardPolling();
  if (typeof window === 'undefined' || !props.eventId || typeof EventSource === 'undefined') {
    return;
  }
  const base = resolveApiUrl(`/events/${props.eventId}/coins/stream`);
  const slug = getOrganizationSlug();
  const url = slug ? base + (base.includes('?') ? '&' : '?') + 'organization_slug=' + encodeURIComponent(slug) : base;
  leaderboardSseSource = new EventSource(url);
  leaderboardSseSource.addEventListener('message', () => {
    refreshLeaderboardPreview();
  });
}

function openFeedbackModal() {
  if (!hasFeedbackSurvey.value || hasSubmittedFeedback.value) {
    trackAppEvent('feedback.cta_ignored', { already_submitted: hasSubmittedFeedback.value, available: hasFeedbackSurvey.value }, 'feedback');
    return;
  }
  trackAppEvent('feedback.modal_opened', { survey_enabled: hasFeedbackSurvey.value }, 'feedback');
  isFeedbackModalOpen.value = true;
}

function handleFeedbackSubmitted() {
  trackAppEvent('feedback.submitted', { event_id: props.eventId }, 'feedback');
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
    if (sponsors.value.length) {
      trackSectionView('sponsors.marquee', { sponsor_count: sponsors.value.length });
    }
  } catch (error) {
    trackAppEvent('error.sponsors_load_failed', { message: error?.message || 'unknown_error' }, 'error');
    sponsors.value = [];
  }
}

function handleSponsorClick(sponsor) {
  trackAppEvent('sponsor.clicked', { sponsor_id: Number(sponsor?.id) || 0, sponsor_name: String(sponsor?.name || ''), sponsor_link: String(sponsor?.linkUrl || sponsor?.link_url || '') }, 'sponsor');
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
  const story = activeStories.value[index] || null;
  trackAppEvent('content.story_opened', { story_id: Number(story?.id) || 0, story_index: index, story_title: String(story?.title || story?.headline || '') }, 'content');
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

watch(isBarFeatureEnabled, (enabled) => {
  if (enabled) {
    return;
  }
  isBarModalOpen.value = false;
  barStep.value = 'start';
  barStepHistory.value = [];
  barCart.value = {};
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
  endTrackingLifecycle('live_experience_unmounted');
  stopLeaderboardPolling();
  stopWheelTickLoop();
  if (typeof window !== 'undefined' && boostCountdownTimer !== null) {
    window.clearInterval(boostCountdownTimer);
    boostCountdownTimer = null;
  }
  if (typeof window !== 'undefined' && inactivityTimer !== null) {
    window.clearInterval(inactivityTimer);
    inactivityTimer = null;
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
  updateTrackingContext({
    fanId: Number(data?.user?.id) || 0,
    fanSessionToken: data.session_token || fanSessionToken.value || undefined,
    loginState: data.registered ? 'logged_in' : 'guest',
    profileState: data.registered ? 'registered' : 'guest',
  });
  trackAppEvent('fan.profile_loaded', { registered: Boolean(data.registered), wallet: Number(data.wallet || data.guest_coins || 0) }, 'fan');
  if (data.registered) {
    fanId.value = Number(data.user?.id) || 0;
    fanNickname.value = data.user?.nickname || '';
    totalCoins.value = Math.max(0, Number(data.wallet) || 0);
    leaderboardUser.value = normalizeLeaderboardUser(data.user_rank ?? data.userRank);
    fanRewardRedemptions.value = Array.isArray(data.reward_redemptions) ? data.reward_redemptions : [];
    fanLotteryTicket.value = data.lottery_ticket || null;
    if (totalCoins.value >= 8 && totalCoins.value < 15) {
      void maybeOpenAiPopup('coins_near_reward', 'far_spendere_monete', { wallet: totalCoins.value });
    }
  } else if (Number.isFinite(Number(data.guest_coins))) {
    totalCoins.value = Math.max(totalCoins.value, Number(data.guest_coins) || 0);
    fanRewardRedemptions.value = [];
    fanLotteryTicket.value = null;
  }
}


async function handleExistingFanLogin() {
  await loadFanProfile();
  if (!isRegisteredFan.value) {
    trackAppEvent('fan.login_failed', { reason: 'fan_profile_not_found' }, 'fan');
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
    trackAppEvent('fan.registration_failed', { trigger: form.trigger, message: response.message || 'registration_failed' }, 'fan');
    return { ok: false, message: response.message };
  }
  trackAppEvent('fan.registration_completed', { trigger: form.trigger, nickname: form.nickname, accepted_terms: form.acceptedTerms }, 'fan');
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
  trackAppEvent('fan.profile_opened', { registered_fan: isRegisteredFan.value }, 'fan');
  if (!isRegisteredFan.value) {
    openRegistrationPrompt('profile_overlay');
    return;
  }
  isProfileOverlayOpen.value = true;
}

function closeProfileOverlay() {
  trackAppEvent('fan.profile_closed', {}, 'fan');
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

function segmentLabelStyle(index) {
  const angle = index * WHEEL_SEGMENT_DEG + WHEEL_SEGMENT_DEG / 2;
  return {
    transform: `translate(-50%, -50%) rotate(${angle}deg) translateY(-86px) rotate(${-angle}deg)`,
  };
}

function pickWheelSegment() {
  const totalWeight = wheelSegments.reduce((sum, segment) => sum + (segment.weight || 1), 0);
  let random = Math.random() * totalWeight;
  for (const segment of wheelSegments) {
    random -= segment.weight || 1;
    if (random <= 0) return segment;
  }
  return wheelSegments[0];
}

async function creditWheelCoins(amount) {
  await addCoins(amount, { source: 'wheel' });

  if (coinAnimationRef.value?.play && walletTargetEl.value) {
    await coinAnimationRef.value.play({
      fromEl: wheelEl.value || walletTargetEl.value,
      toEl: walletTargetEl.value,
      count: 18,
      amount,
    });
  }
}


function triggerWheelStopVibration() {
  if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') {
    navigator.vibrate(120);
  }
}


function playWheelTick() {
  if (typeof window === 'undefined') {
    return;
  }
  const AudioCtx = window.AudioContext || window.webkitAudioContext;
  if (!AudioCtx) {
    return;
  }
  wheelAudioContext = wheelAudioContext || new AudioCtx();
  const oscillator = wheelAudioContext.createOscillator();
  const gain = wheelAudioContext.createGain();
  oscillator.type = 'square';
  oscillator.frequency.value = 920;
  gain.gain.value = 0.02;
  oscillator.connect(gain);
  gain.connect(wheelAudioContext.destination);
  oscillator.start();
  oscillator.stop(wheelAudioContext.currentTime + 0.03);
}

function startWheelTickLoop() {
  stopWheelTickLoop();
  wheelTickTimer = window.setInterval(() => {
    playWheelTick();
  }, 120);
}

function stopWheelTickLoop() {
  if (wheelTickTimer) {
    window.clearInterval(wheelTickTimer);
    wheelTickTimer = null;
  }
}

async function executeWheelReward(segment) {
  if (segment.type === 'coins' || segment.type === 'jackpot') {
    await creditWheelCoins(segment.amount || 0);
    return;
  }

  if (segment.type === 'nextMultiplier') {
    nextGameMultiplier.value = 2;
    persistPowerUps();
    return;
  }

  if (segment.type === 'freeRetry') {
    grantFreeRetry();
    return;
  }

}

async function spinFortuneWheel() {
  if (isWheelSpinning.value || !isWheelModalOpen.value || !canSpinWheel.value) {
    return;
  }

  isWheelSpinning.value = true;
  wheelResult.value = null;
  wheelWinningIndex.value = -1;

  await addCoins(-FORTUNE_WHEEL_COST, { source: 'wheel' });

  const winningSegment = pickWheelSegment();
  const targetIndex = wheelSegments.findIndex((segment) => segment.id === winningSegment.id);
  const targetCenter = targetIndex * WHEEL_SEGMENT_DEG + WHEEL_SEGMENT_DEG / 2;
  // Account for the current accumulated rotation so the pointer always lands
  // precisely on the winning segment, even after multiple spins.
  const currentEffectiveAngle = wheelRotationDeg.value % 360;
  const landingRotation = ((360 - targetCenter) - currentEffectiveAngle + 360) % 360;

  startWheelTickLoop();
  wheelRotationDeg.value += WHEEL_SPINS * 360 + landingRotation;
  await wait(3800);
  stopWheelTickLoop();

  wheelWinningIndex.value = targetIndex;
  wheelResult.value = winningSegment;
  triggerWheelStopVibration();
  await executeWheelReward(winningSegment);
  isWheelSpinning.value = false;
}

function closeWheelModal() {
  if (isWheelSpinning.value) {
    return;
  }
  isWheelModalOpen.value = false;
}

function openWheelModal() {
  if (isWheelSpinning.value || !canSpinWheel.value) {
    return;
  }
  isWheelModalOpen.value = true;
  wheelResult.value = null;
  wheelWinningIndex.value = -1;
  nextTick(() => {
    spinFortuneWheel();
  });
}



function openSpendPreview() {
  registerUserActivity();
  trackAppEvent('coins.store_opened', { wallet_coins: totalCoins.value }, 'coins');
  isSpendPreviewOpen.value = true;
  mysteryBoxStep.value = 'idle';
  mysteryBoxStatusText.value = '';
}

function closeSpendPreview() {
  if (isWheelSpinning.value) {
    trackAppEvent('coins.store_close_blocked', { reason: 'wheel_spinning' }, 'coins');
    return;
  }
  trackAppEvent('coins.store_closed', { wallet_coins: totalCoins.value }, 'coins');
  isSpendPreviewOpen.value = false;
}

async function attemptRedeem(rewardKey, costCoins, rewardLabel) {
  if (!isRegisteredFan.value) {
    trackAppEvent('coins.reward_redeem_blocked', { reward_key: rewardKey, cost_coins: costCoins, reason: 'registration_required' }, 'coins');
    selectedRewardLabel.value = rewardLabel || `${String(rewardKey).replace('-', ' ').toUpperCase()} · ${costCoins} 🪙`;
    openRegistrationPrompt('spend_redeem');
    return;
  }
  trackAppEvent('coins.reward_redeem_attempted', { reward_key: rewardKey, cost_coins: costCoins, reward_label: rewardLabel }, 'coins');
  const response = await redeemFanReward(props.eventId, rewardKey, costCoins);
  if (response?.ok) {
    trackAppEvent('coins.reward_redeem_completed', { reward_key: rewardKey, cost_coins: costCoins, wallet_after: Number(response.data?.wallet) || totalCoins.value }, 'coins');
    totalCoins.value = Number(response.data?.wallet) || totalCoins.value;
    return;
  }
  trackAppEvent('coins.reward_redeem_failed', { reward_key: rewardKey, cost_coins: costCoins, message: response?.message || 'redeem_failed' }, 'coins');
}

function registerUserActivity() {
  lastActivityAt.value = Date.now();
  updateTrackingContext({ section: isBarModalOpen.value ? `bar.${barStep.value}` : 'home' });
}

async function maybeOpenAiPopup(triggerType, objective, extra = {}) {
  try {
    const sessionId = getOrCreateDeviceId();
    const popup = await generateAIPopup({
      trigger_type: triggerType,
      objective,
      session_id: sessionId,
      event_id: props.eventId,
      event_phase: props.isLive ? 'live' : 'prematch',
      sessions_count: isRegisteredFan.value ? 2 : 1,
      games_played: Number(extra.gamesPlayed || 0),
      coins: totalCoins.value,
      inactive_seconds: Math.max(0, Math.round((Date.now() - lastActivityAt.value) / 1000)),
      cart_items_count: barCartCount.value,
      cart_total_cents: barTotalCents.value,
      popup_history_session: aiPopupHistory.value,
      extra,
    });
    if (!popup || popup.source === 'suppressed' || !popup.popup_title) {
      return;
    }
    activeAiPopup.value = { ...popup, trigger_type: triggerType };
    isAiPopupOpen.value = true;
    aiPopupHistory.value = [...aiPopupHistory.value, { trigger_type: triggerType, interaction_id: popup.interaction_id || 0, shown_at: new Date().toISOString() }].slice(-10);
    safeTrackEvent('ai', 'popup_shown', triggerType, { source: popup.source, interactionId: popup.interaction_id || 0 });
    if (popup.interaction_id) {
      await trackAIInteraction(popup.interaction_id, 'shown', { session_id: sessionId, trigger: triggerType });
    }
  } catch (error) {
    // silent fallback
  }
}

async function handleAiPopupDismiss() {
  const interactionId = Number(activeAiPopup.value?.interaction_id) || 0;
  safeTrackEvent('ai', 'popup_dismissed', activeAiPopup.value?.trigger_type || 'popup', { interactionId });
  if (interactionId) {
    await trackAIInteraction(interactionId, 'dismissed', { session_id: getOrCreateDeviceId() }).catch(() => {});
  }
}

async function handleAiPopupCTA() {
  const trigger = String(activeAiPopup.value?.trigger_type || 'popup');
  const interactionId = Number(activeAiPopup.value?.interaction_id) || 0;
  safeTrackEvent('ai', 'popup_clicked', trigger, { interactionId });
  if (interactionId) {
    await trackAIInteraction(interactionId, 'clicked', { session_id: getOrCreateDeviceId() }).catch(() => {});
  }
  if (trigger === 'cart_abandon_risk' || trigger === 'bar.step_viewed') {
    openBarOrdering();
  } else if (trigger === 'coins_near_reward') {
    openSpendPreview();
  } else {
    isEarnModalOpen.value = true;
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
  if (barCategoriesData.value.length) {
    return barCategoriesData.value.map((category) => ({
      id: String(category.id),
      name: String(category.name || 'Categoria BAR'),
      image: String(category.image_url || productImageFallback),
    }));
  }

  const map = new Map();
  for (const p of normalizedBarProducts.value) {
    if (!map.has(p.category)) {
      map.set(p.category, { id: p.category, name: p.category, image: p.categoryImage || p.image || productImageFallback });
    }
  }
  return Array.from(map.values());
});

const selectedBarCategory = computed(() => barCategories.value.find((category) => String(category.id) === String(selectedCategoryId.value)) || null);

const barProductsByCategory = computed(() => {
  return normalizedBarProducts.value.filter((p) => {
    if (selectedCategoryId.value === 'all') return true;
    if (p.category_id !== undefined && p.category_id !== null && Number(p.category_id) > 0) {
      return String(p.category_id) === String(selectedCategoryId.value);
    }
    if (selectedBarCategory.value?.name) {
      return String(p.category || '').trim().toLowerCase() === String(selectedBarCategory.value.name || '').trim().toLowerCase();
    }
    return p.category === selectedCategoryId.value;
  });
});

const selectedBarProduct = computed(() => normalizedBarProducts.value.find((p) => String(p.id) === String(selectedBarProductId.value)) || null);
const barSuggestionsPayload = ref({ title: '', items: [], source: 'none', enabled: false });
const barSuggestionItems = computed(() => (Array.isArray(barSuggestionsPayload.value?.items) ? barSuggestionsPayload.value.items : []).map((item) => {
  const normalized = normalizedBarProducts.value.find((product) => String(product.id) === String(item.id));
  return normalized || item;
}).slice(0, 3));
const barSuggestionTitle = computed(() => String(barSuggestionsPayload.value?.title || 'Completa il tuo ordine'));

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
  products: selectedBarCategory.value?.name || selectedCategoryId.value || 'Prodotti',
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
  registerUserActivity();
  trackAppEvent('bar.step_viewed', { step: nextStep, previous_step: barStep.value, cart_items_count: barCartCount.value, cart_total_cents: barTotalCents.value }, 'bar');
  if (barStep.value !== nextStep) barStepHistory.value.push(barStep.value);
  barStep.value = nextStep;
  if (nextStep === 'cart' && barCartCount.value > 0) {
    maybeOpenAiPopup('bar.step_viewed', 'far_completare_ordine', { cartSummary: barOrderSummaryLabel.value, tracked_event_name: 'bar.step_viewed', tracked_step: nextStep });
  }
}

function goBackBarStep() {
  const prev = barStepHistory.value.pop();
  barStep.value = prev || 'start';
}

function openBarOrdering() {
  if (!isBarFeatureEnabled.value) {
    return;
  }
  registerUserActivity();
  trackAppEvent('bar.menu_opened', { cart_items_count: barCartCount.value, cart_total_cents: barTotalCents.value }, 'bar');
  barOrderConfirmed.value = false;
  barCheckoutError.value = '';
  barStep.value = 'start';
  barStepHistory.value = [];
  selectedCategoryId.value = 'all';
  void preloadBarCatalog({ force: true });
  isBarModalOpen.value = true;
}

function selectBarMode(modeId) {
  trackAppEvent('bar.order_mode_selected', { order_mode: modeId }, 'bar');
  barOrderMode.value = modeId;
  goBarStep('categories');
}

function openBarCategory(categoryId) {
  trackAppEvent('bar.category_viewed', { category_id: String(categoryId) }, 'bar');
  selectedCategoryId.value = String(categoryId);
  goBarStep('products');
}

function openProductDetail(productId) {
  const product = normalizedBarProducts.value.find((item) => String(item.id) === String(productId));
  trackAppEvent('bar.product_viewed', { product_id: productId, product_name: product?.name || '' }, 'bar');
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
  trackAppEvent('bar.cart_quantity_changed', { product_id: productId, delta: 1, previous_qty: Number(barCart.value?.[productId] || 0) }, 'bar');
  const current = Number(barCart.value?.[productId] || 0);
  barCart.value = { ...barCart.value, [productId]: current + 1 };
}

function decreaseBarQty(productId) {
  const current = Number(barCart.value?.[productId] || 0);
  trackAppEvent('bar.cart_quantity_changed', { product_id: productId, delta: -1, previous_qty: current }, 'bar');
  if (current <= 0) return;
  barCart.value = { ...barCart.value, [productId]: current - 1 };
}

function removeBarProduct(productId) {
  trackAppEvent('bar.cart_item_removed', { product_id: productId, previous_qty: Number(barCart.value?.[productId] || 0) }, 'bar');
  barCart.value = { ...barCart.value, [productId]: 0 };
}

async function addProductFromDetail() {
  if (!selectedBarProduct.value) return;
  trackAppEvent('bar.added_to_cart', { product_id: selectedBarProduct.value.id, product_name: selectedBarProduct.value.name, quantity: barDetailQty.value, extras_count: selectedBarExtras.value.length }, 'bar');
  const productId = selectedBarProduct.value.id;
  const current = Number(barCart.value?.[productId] || 0);
  barCart.value = { ...barCart.value, [productId]: current + barDetailQty.value };
  await loadBarSuggestionsForProduct(productId);
  goBarStep('upsell');
}

async function addSuggestedProduct(productId) {
  trackAppEvent('bar.upsell_accepted', { product_id: productId, interaction_id: Number(barSuggestionsPayload.value?.interaction_id) || 0 }, 'bar');
  if (productId) increaseBarQty(productId);
  const interactionId = Number(barSuggestionsPayload.value?.interaction_id) || 0;
  safeTrackEvent('ai', 'upsell_added_to_cart', String(productId || ''), { interactionId });
  if (interactionId) {
    await trackAIInteraction(interactionId, 'upsell_added_to_cart', { session_id: getOrCreateDeviceId() }).catch(() => {});
  }
  goBarStep('products');
}


function defaultBarSuggestionsPayload() {
  return { title: '', items: [], source: 'none', enabled: false };
}

function normalizeBarSuggestionsPayload(data) {
  return data && typeof data === 'object' ? data : defaultBarSuggestionsPayload();
}

async function fetchBarSuggestionsForProduct(productID, options = {}) {
  const numericId = Number(String(productID || '').replace('product:', ''));
  if (!numericId) {
    return defaultBarSuggestionsPayload();
  }

  const cacheKey = String(numericId);
  if (!options.force && barSuggestionsCache.value[cacheKey]) {
    return barSuggestionsCache.value[cacheKey];
  }

  try {
    const { data } = await apiClient.get(`/bar/suggestions/${numericId}`);
    const payload = normalizeBarSuggestionsPayload(data);
    barSuggestionsCache.value = { ...barSuggestionsCache.value, [cacheKey]: payload };
    return payload;
  } catch (error) {
    const payload = defaultBarSuggestionsPayload();
    barSuggestionsCache.value = { ...barSuggestionsCache.value, [cacheKey]: payload };
    return payload;
  }
}

async function preloadBarSuggestions(products, options = {}) {
  const productIds = (Array.isArray(products) ? products : [])
    .map((product) => String(product?.id || ''))
    .filter((id) => id.startsWith('product:'));

  if (!productIds.length) {
    return;
  }

  await Promise.all(productIds.map((productId) => fetchBarSuggestionsForProduct(productId, options)));
}

async function preloadBarCatalog(options = {}) {
  if (!isBarFeatureEnabled.value) {
    return;
  }
  if (barCatalogPreloadPromise && !options.force) {
    return barCatalogPreloadPromise;
  }

  barCatalogPreloadPromise = (async () => {
    await Promise.all([loadBarProducts(), loadBarCategories()]);
    await preloadBarSuggestions(barProducts.value, options);
  })();

  try {
    await barCatalogPreloadPromise;
  } finally {
    barCatalogPreloadPromise = null;
  }
}

async function loadBarSuggestionsForProduct(productID) {
  const selectedItems = barCartItems.value.map((item) => ({
    product_id: String(item.id),
    product_name: item.name,
    category: item.category,
    quantity: item.qty,
  }));
  try {
    const aiPayload = await generateAIBarUpsell({
      trigger: 'bar.added_to_cart',
      event_id: props.eventId,
      event_phase: props.isLive ? 'live' : 'prematch',
      cart: selectedItems,
      available_products: normalizedBarProducts.value.map((product) => ({
        product_id: String(product.id),
        name: product.name,
        category: product.category,
        available: true,
        visible: true,
        priority: String(product.badge || '').toLowerCase().includes('venduto'),
        price_cents: Number(product.price_cents || 0),
      })),
    });
    const suggestions = Array.isArray(aiPayload?.suggestions) ? aiPayload.suggestions : [];
    if (suggestions.length) {
      barSuggestionsPayload.value = {
        title: 'Potrebbe piacerti anche',
        source: aiPayload?.source || 'ai',
        enabled: true,
        interaction_id: aiPayload?.interaction_id || 0,
        items: suggestions.map((suggestion) => {
          const matched = normalizedBarProducts.value.find((product) => product.name === suggestion.product_name || String(product.id) === String(suggestion.product_id));
          return matched ? { ...matched, reason: suggestion.reason, marketing_text: suggestion.marketing_text } : suggestion;
        }),
      };
      if (aiPayload?.interaction_id) {
        await trackAIInteraction(aiPayload.interaction_id, 'shown', { session_id: getOrCreateDeviceId(), trigger: 'bar.added_to_cart' }).catch(() => {});
      }
      trackAppEvent('ai.upsell_shown', { source: aiPayload?.source || 'ai', interaction_id: aiPayload?.interaction_id || 0, suggestions_count: suggestions.length }, 'ai');
      safeTrackEvent('ai', 'upsell_shown', 'bar.added_to_cart', { source: aiPayload?.source || 'ai', interactionId: aiPayload?.interaction_id || 0 });
      return;
    }
  } catch (error) {
    // fallback below
  }
  barSuggestionsPayload.value = await fetchBarSuggestionsForProduct(productID);
}

async function loadBarCategories() {
  if (!isBarFeatureEnabled.value) {
    barCategoriesData.value = [];
    return;
  }
  try {
    const { data } = await apiClient.get('/bar/categories');
    barCategoriesData.value = Array.isArray(data) ? data : [];
  } catch (error) {
    barCategoriesData.value = [];
  }
}

async function loadBarProducts() {
  if (!isBarFeatureEnabled.value) {
    barProducts.value = [];
    return;
  }
  try {
    const { data } = await apiClient.get('/bar/products');
    barProducts.value = Array.isArray(data) ? data : [];
  } catch (error) {
    barProducts.value = [];
  }
}

async function startBarCheckout() {
  barCheckoutError.value = '';
  trackAppEvent('bar.checkout_started', { order_mode: barOrderMode.value, cart_items_count: barCartCount.value, cart_total_cents: barTotalCents.value }, 'bar');
  if (barOrderMode.value === 'seat' && (!barDelivery.value.sector || !barDelivery.value.row || !barDelivery.value.seat)) {
    barCheckoutError.value = 'Inserisci settore, fila e posto.';
    return;
  }

  const items = Object.entries(barCart.value)
    .map(([product_id, quantity]) => ({ product_id, quantity: Number(quantity) }))
    .filter((entry) => entry.quantity > 0);

  if (!items.length) {
    barCheckoutError.value = 'Seleziona almeno un prodotto.';
    trackAppEvent('bar.checkout_failed', { reason: 'empty_cart' }, 'bar');
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
      trackAppEvent('bar.checkout_failed', { reason: 'missing_checkout_url' }, 'bar');
      return;
    }

    if (typeof window !== 'undefined') {
      window.localStorage.setItem('bar:last_session_id', String(data.session_id));
    }
    trackAppEvent('bar.checkout_redirected', { session_id: String(data.session_id), checkout_url_present: Boolean(data?.checkout_url) }, 'bar');

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
    trackAppEvent('bar.checkout_failed', { reason: 'request_error', message: barCheckoutError.value }, 'bar');
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
      trackAppEvent('bar.order_completed', { order_id: data?.order_id || 0, session_id: sessionId, cart_total_cents: barTotalCents.value }, 'bar');
      barOrderConfirmed.value = true;
      barConfirmedOrderNumber.value = String(data?.order_id || sessionId).slice(-6);
      isBarModalOpen.value = true;
      barStep.value = 'confirmation';
      barCart.value = {};
      barDelivery.value = { sector: '', row: '', seat: '', notes: '' };
    }
  } catch (error) {
    trackAppEvent('bar.order_confirmation_failed', { session_id: sessionId, message: error?.response?.data?.message || 'confirmation_failed' }, 'bar');
  }
}


function onFeatureSelect(featureId) {
  registerUserActivity();
  trackAppEvent('navigation.feature_selected', { feature_id: featureId }, 'navigation');
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

.mini-feature__boost--next {
  color: #86efac;
}

.wheel-card {
  position: relative;
}

.wheel-card__spin-btn {
  border-radius: 9999px;
  border: 1px solid rgba(252, 211, 77, 0.45);
  background: linear-gradient(180deg, rgba(251, 191, 36, 0.45), rgba(180, 83, 9, 0.55));
  padding: 0.6rem 1.25rem;
  font-weight: 900;
  color: #fff;
}

.wheel-card__spin-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.wheel-modal {
  background: radial-gradient(circle at top, rgba(71, 85, 105, 0.3), rgba(2, 6, 23, 0.96) 70%);
}

.wheel-modal__content {
  backdrop-filter: blur(2px);
}

.wheel-wrap {
  position: relative;
  display: flex;
  justify-content: center;
}

.wheel-wrap--modal {
  height: min(70vh, 420px);
  width: min(70vh, 420px);
  align-items: center;
}

/* ── Pointer ─────────────────────────────────────── */
.wheel-pointer {
  position: absolute;
  top: -1.1rem;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none;
}

.wheel-pointer__triangle {
  width: 0;
  height: 0;
  border-left: 14px solid transparent;
  border-right: 14px solid transparent;
  border-top: 24px solid #fde68a;
  filter: drop-shadow(0 0 8px rgba(251, 191, 36, 0.9)) drop-shadow(0 2px 4px rgba(0,0,0,0.6));
}

.wheel-pointer__stem {
  width: 6px;
  height: 12px;
  background: linear-gradient(180deg, #fde68a, #f59e0b);
  border-radius: 0 0 3px 3px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.4);
}

/* ── Outer decorative ring ────────────────────────── */
.wheel-outer-ring {
  position: absolute;
  inset: -6px;
  border-radius: 9999px;
  border: 5px solid transparent;
  background: conic-gradient(
    #fbbf24, #f59e0b, #ef4444, #a855f7, #3b82f6, #22c55e, #f97316, #eab308, #fbbf24
  ) border-box;
  -webkit-mask: linear-gradient(#fff 0 0) padding-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: destination-out;
  mask-composite: exclude;
  opacity: 0.75;
  pointer-events: none;
  transition: opacity 0.4s, box-shadow 0.4s;
}

.wheel-outer-ring--jackpot {
  opacity: 1;
  box-shadow: 0 0 24px rgba(250, 204, 21, 0.9);
  animation: ring-jackpot 0.6s ease infinite alternate;
}

@keyframes ring-jackpot {
  from { opacity: 0.85; }
  to   { opacity: 1; box-shadow: 0 0 40px rgba(250,204,21,1); }
}

/* ── Wheel ────────────────────────────────────────── */
.fortune-wheel {
  position: relative;
  height: 100%;
  width: 100%;
  border-radius: 9999px;
  border: 6px solid rgba(255, 255, 255, 0.7);
  /* White 1-deg separators between each coloured segment */
  background: conic-gradient(
    #f59e0b    0deg  44deg,
    #fff      44deg  45deg,
    #a855f7   45deg  89deg,
    #fff      89deg  90deg,
    #22c55e   90deg 134deg,
    #fff     134deg 135deg,
    #ef4444  135deg 179deg,
    #fff     179deg 180deg,
    #14b8a6  180deg 224deg,
    #fff     224deg 225deg,
    #3b82f6  225deg 269deg,
    #fff     269deg 270deg,
    #f97316  270deg 314deg,
    #fff     314deg 315deg,
    #eab308  315deg 359deg,
    #fff     359deg 360deg
  );
  box-shadow:
    0 0 0 3px rgba(255,255,255,0.15),
    0 0 32px rgba(251, 191, 36, 0.4),
    0 8px 32px rgba(0,0,0,0.5);
  transition: transform 3.8s cubic-bezier(0.22, 1, 0.36, 1);
  overflow: hidden;
}

.fortune-wheel::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 9999px;
  background: radial-gradient(
    circle at 38% 28%,
    rgba(255,255,255,0.18) 0%,
    transparent 60%
  );
  pointer-events: none;
}

.fortune-wheel--jackpot {
  box-shadow:
    0 0 0 3px rgba(255,255,255,0.2),
    0 0 60px rgba(250, 204, 21, 1),
    0 8px 32px rgba(0,0,0,0.5);
}

/* ── SVG dividers ─────────────────────────────────── */
.fortune-wheel__dividers {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 1;
}

/* ── Labels ───────────────────────────────────────── */
.fortune-wheel__label {
  position: absolute;
  left: 50%;
  top: 50%;
  z-index: 3;
  font-size: clamp(0.62rem, 2.1vw, 0.82rem);
  font-weight: 900;
  color: #fff;
  text-shadow: 0 1px 6px rgba(0, 0, 0, 0.85);
  white-space: nowrap;
  background: rgba(0,0,0,0.22);
  padding: 0.1em 0.35em;
  border-radius: 999px;
  backdrop-filter: blur(1px);
  line-height: 1.3;
}

.fortune-wheel__label--winner {
  color: #fde68a;
  background: rgba(0,0,0,0.45);
  text-shadow: 0 0 16px rgba(250, 204, 21, 1), 0 1px 3px rgba(0,0,0,0.9);
  box-shadow: 0 0 10px rgba(250,204,21,0.7);
}

/* ── Center hub ───────────────────────────────────── */
.fortune-wheel__hub {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: clamp(36px, 11%, 52px);
  height: clamp(36px, 11%, 52px);
  border-radius: 50%;
  background: radial-gradient(circle at 40% 35%, #fef9c3, #f59e0b 60%, #b45309);
  border: 3px solid rgba(255,255,255,0.85);
  box-shadow:
    0 0 14px rgba(251,191,36,0.7),
    0 3px 10px rgba(0,0,0,0.5),
    inset 0 1px 3px rgba(255,255,255,0.5);
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: center;
}

.fortune-wheel__hub-icon {
  font-size: clamp(0.9rem, 3%, 1.3rem);
  line-height: 1;
  filter: drop-shadow(0 1px 2px rgba(0,0,0,0.4));
}

.wheel-reveal {
  border-radius: 1rem;
  border: 1px solid rgba(252, 211, 77, 0.5);
  background: rgba(15, 23, 42, 0.75);
  padding: 0.8rem 1.2rem;
  box-shadow: 0 0 20px rgba(250, 204, 21, 0.28);
}

.wheel-reveal--jackpot {
  box-shadow: 0 0 30px rgba(250, 204, 21, 0.9);
}

.wheel-reveal__text {
  font-size: clamp(1.5rem, 6vw, 2.5rem);
  font-weight: 900;
  color: #fff;
  animation: prize-bounce 0.65s ease;
}

.wheel-modal__close-btn {
  border-radius: 9999px;
  border: 1px solid rgba(148, 163, 184, 0.55);
  background: rgba(15, 23, 42, 0.85);
  padding: 0.6rem 1.25rem;
  font-weight: 900;
  color: #e2e8f0;
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

@keyframes prize-bounce {
  0% { transform: scale(0.75); }
  60% { transform: scale(1.12); }
  100% { transform: scale(1); }
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
@media (max-width: 640px) {
  .wheel-wrap--modal {
    height: min(70vh, 88vw);
    width: min(70vh, 88vw);
  }
}
</style>

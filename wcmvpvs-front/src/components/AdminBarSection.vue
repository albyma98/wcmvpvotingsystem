<template>
  <section class="bar-admin">
    <header class="bar-hero">
      <div class="bar-hero__identity">
        <div class="bar-hero__icon">🍺</div>
        <div>
          <h1 class="bar-hero__title">{{ settings.partnerName }}</h1>
          <p class="bar-hero__subtitle">Gestione BAR — {{ settings.companyName }}</p>
        </div>
        <div class="bar-hero__live-dot" :class="{ 'bar-hero__live-dot--active': !isLoading }">
          <span class="bar-hero__live-pulse"></span>
          <span class="bar-hero__live-label">{{ isLoading ? 'Aggiornamento…' : 'Live' }}</span>
        </div>
      </div>

      <nav class="bar-nav" aria-label="Sezioni bar">
        <div class="bar-nav__group" v-for="group in tabGroups" :key="group.label">
          <span class="bar-nav__group-label">{{ group.label }}</span>
          <div class="bar-nav__tabs">
            <button
              v-for="t in group.tabs"
              :key="t.id"
              class="bar-nav__tab"
              :class="{ 'bar-nav__tab--active': activeTab === t.id }"
              type="button"
              @click="activeTab = t.id"
              :aria-current="activeTab === t.id ? 'page' : undefined"
            >
              <span class="bar-nav__tab-icon">{{ t.icon }}</span>
              <span class="bar-nav__tab-label">{{ t.label }}</span>
            </button>
          </div>
        </div>
      </nav>
    </header>

    <div v-if="activeTab==='dashboard'" class="tab-panel">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">📊</span>
        <div>
          <h2 class="section-hero-bar__title">Dashboard</h2>
          <p class="section-hero-bar__desc">KPI operativi in tempo reale per questo evento.</p>
        </div>
      </div>

      <div class="kpi-grid">
        <article class="kpi-card" v-for="(k, idx) in kpis" :key="k.label" :data-kpi-index="idx">
          <div class="kpi-card__icon">{{ kpiMeta[idx]?.icon || '📌' }}</div>
          <div class="kpi-card__body">
            <p class="kpi-card__label">{{ k.label }}</p>
            <p class="kpi-card__value">{{ k.value }}</p>
          </div>
        </article>
      </div>

      <div class="info-card">
        <div class="info-card__header">
          <h4 class="info-card__title">Ultimi ordini</h4>
        </div>
        <ul class="order-list">
          <li v-for="o in overview.latest_orders||[]" :key="o.id" class="order-list__row">
            <span class="order-list__id">#{{ o.id }}</span>
            <span class="order-list__status">{{ statusLabel(o.order_status) }}</span>
            <span class="order-list__price">€ {{ (o.total_cents/100).toFixed(2) }}</span>
            <span class="order-list__seat muted">{{ seatingLabel(o) }}</span>
          </li>
          <li v-if="!(overview.latest_orders||[]).length" class="muted" style="padding:.5rem 0; font-size:.875rem;">Nessun ordine recente.</li>
        </ul>
      </div>
    </div>

    <div v-else-if="activeTab==='live'" class="live-board-wrap">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">🔴</span>
        <div>
          <h2 class="section-hero-bar__title">Ordini Live</h2>
          <p class="section-hero-bar__desc">Vista operativa Kanban — solo ordini aperti, con priorità visuale e azioni rapide.</p>
        </div>
      </div>

      <header class="live-toolbar">
        <div class="live-toolbar__actions">
          <span class="muted">Aggiornato {{ lastRefreshLabel }}</span>
          <button class="btn outline" type="button" @click="load" :disabled="isLoading">{{ isLoading ? 'Aggiornamento…' : 'Aggiorna' }}</button>
        </div>
      </header>

      <div class="live-columns" role="list" aria-label="Ordini live per stato">
        <section v-for="column in liveColumns" :key="column.id" class="live-column" role="listitem">
          <header class="live-column__header" :class="`live-column__header--${column.id}`">
            <div class="live-column__header-left">
              <span class="live-column__header-icon">{{ liveColumnMeta[column.id]?.icon }}</span>
              <h4 class="live-column__title">{{ column.title }}</h4>
            </div>
            <span class="live-column__count" :class="`live-column__count--${column.id}`">{{ column.orders.length }}</span>
          </header>

          <div v-if="!column.orders.length" class="live-column__empty">
            <p>{{ column.emptyText }}</p>
          </div>

          <article
            v-for="order in column.orders"
            :key="order.id"
            class="order-card"
            :class="[`order-card--${order.order_status}`, urgencyClass(order.elapsedMinutes)]"
            @click="openOrderDetail(order)"
          >
            <header class="order-card__header">
              <div>
                <p class="order-card__id">Ordine #{{ order.id }}</p>
                <p class="order-card__time">{{ formatOrderTime(order.created_at) }}</p>
              </div>
              <div class="order-card__badges">
                <span class="badge" :class="`badge--${paymentClass(order.payment_status)}`">{{ paymentLabel(order.payment_status) }}</span>
                <span class="badge" :class="urgencyBadgeClass(order.elapsedMinutes)">{{ order.elapsedMinutes }} min</span>
              </div>
            </header>

            <div class="order-card__meta">
              <p><strong>Totale:</strong> € {{ euro(order.total_cents) }}</p>
              <p><strong>Cliente:</strong> {{ customerLabel(order) }}</p>
              <p><strong>Posizione:</strong> {{ seatingLabel(order) }}</p>
            </div>

            <ul class="order-card__items">
              <li v-for="item in order.parsedItems" :key="`${order.id}-${item.id}`">
                <strong>x{{ item.quantity }}</strong> {{ item.name }}
              </li>
              <li v-if="!order.parsedItems.length" class="muted">Prodotti non disponibili</li>
            </ul>

            <p v-if="order.notes" class="order-card__notes">📝 {{ order.notes }}</p>

            <footer class="order-card__actions" @click.stop>
              <button
                v-if="nextStatusAction(order.order_status)"
                class="btn primary"
                type="button"
                @click="setStatus(order.id, nextStatusAction(order.order_status).status)"
              >
                {{ nextStatusAction(order.order_status).label }}
              </button>
              <button
                v-if="canCancel(order.order_status)"
                class="btn danger"
                type="button"
                @click="setStatus(order.id, 'annullato')"
              >
                Annulla
              </button>
              <button class="btn outline" type="button" @click="openOrderDetail(order)">Dettaglio</button>
            </footer>
          </article>
        </section>
      </div>
    </div>

    <div v-else-if="activeTab==='history'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">📋</span>
        <div>
          <h2 class="section-hero-bar__title">Storico ordini</h2>
          <p class="section-hero-bar__desc">Controlla gli ordini passati con filtri rapidi.</p>
        </div>
      </div>

      <div class="filter-card">
        <p class="filter-card__label">Filtri</p>
        <div class="filter-grid">
        <label>
          Data
          <input v-model="historyDate" class="input" type="date" />
        </label>
        <label>
          Stato ordine
          <select v-model="historyStatus" class="input">
            <option value="">Tutti</option>
            <option value="new">Nuovo</option>
            <option value="in_preparazione">In preparazione</option>
            <option value="pronto">Pronto</option>
            <option value="completato">Completato</option>
            <option value="annullato">Annullato</option>
          </select>
        </label>
        <label>
          Cerca ordine
          <input v-model="historySearch" class="input" type="search" placeholder="Es: 1042" />
        </label>
        </div>
      </div>

      <div class="simple-list">
        <button v-for="order in historyOrders" :key="`history-${order.id}`" class="simple-list__row" type="button" @click="openOrderDetail(order)">
          <strong>#{{ order.id }}</strong>
          <span>{{ formatOrderTime(order.created_at, true) }}</span>
          <span>€ {{ euro(order.total_cents) }}</span>
          <span>{{ statusLabel(order.order_status) }}</span>
          <span>{{ paymentLabel(order.payment_status) }}</span>
          <span>{{ customerLabel(order) }}</span>
        </button>
        <p v-if="!historyOrders.length" class="muted">Nessun ordine trovato con i filtri selezionati.</p>
      </div>
    </div>

    <div v-else-if="activeTab==='menu'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">🍽️</span>
        <div>
          <h2 class="section-hero-bar__title">Menu bar</h2>
          <p class="section-hero-bar__desc">Crea/elimina prodotti e crea pacchetti menù scontati.</p>
        </div>
      </div>

      <article class="info-card">
        <h4>Nuovo prodotto</h4>
        <div class="filter-grid">
          <label>Nome<input v-model="newProduct.name" class="input" type="text" /></label>
          <label>Prezzo (€)<input v-model.number="newProduct.priceEuro" class="input" type="number" min="0" step="0.01" /></label>
          <label>Descrizione<input v-model="newProduct.description" class="input" type="text" /></label>
          <label>Immagine prodotto
            <input class="input" type="file" accept="image/*" @change="onProductImageSelect" />
          </label>
          <fieldset class="category-radio-group">
            <legend>Categoria</legend>
            <p v-if="!barCategories.length" class="muted">Nessuna categoria disponibile. Crea prima una categoria.</p>
            <label v-for="category in barCategories" :key="`product-category-${category.id}`" class="category-radio-option">
              <input v-model.number="newProduct.categoryId" type="radio" name="new-product-category" :value="category.id" />
              <span>{{ category.name }}</span>
            </label>
          </fieldset>
        </div>
        <p v-if="newProduct.image_url" class="muted">Immagine prodotto selezionata ✅</p>
        <button class="btn primary" type="button" @click="createProduct">Crea prodotto</button>
      </article>

      <div class="menu-grid">
        <article v-for="product in productCards" :key="product.id" class="menu-card">
          <h4>{{ product.name }}</h4>
          <p class="menu-card__price">€ {{ euro(product.price_cents) }}</p>
          <p class="muted">{{ product.description || 'Prodotto bar' }}</p>
          <img v-if="product.image_url" :src="product.image_url" :alt="product.name" class="h-24 w-full rounded object-cover" />
          <p class="muted"><strong>Categoria:</strong> {{ product.category || 'Non assegnata' }}</p>
          <div class="menu-card__actions">
            <button class="btn danger" type="button" @click="deleteProduct(product.id)">Elimina definitivamente</button>
          </div>
        </article>
      </div>

      <article class="info-card">
        <h4>Nuovo menù (pacchetto)</h4>
        <div class="filter-grid">
          <label>Nome<input v-model="newMenu.name" class="input" type="text" /></label>
          <label>Prezzo menù (€)<input v-model.number="newMenu.priceEuro" class="input" type="number" min="0" step="0.01" /></label>
          <label>Descrizione<input v-model="newMenu.description" class="input" type="text" /></label>
        </div>
        <div class="simple-list">
          <label v-for="product in productCards" :key="`menu-item-${product.id}`" class="setting-row">
            <input type="checkbox" :checked="isMenuProductSelected(product.id)" @change="toggleMenuProduct(product.id, $event.target.checked)" />
            {{ product.name }}
            <input class="input" type="number" min="1" style="max-width:90px" :disabled="!isMenuProductSelected(product.id)" :value="menuQty(product.id)" @input="setMenuQty(product.id, $event.target.value)" />
          </label>
        </div>
        <button class="btn primary" type="button" @click="createMenu">Crea menù</button>
      </article>

      <article class="info-card">
        <h4>Menù esistenti</h4>
        <div class="simple-list">
          <div v-for="menu in barMenus" :key="`bar-menu-${menu.id}`" class="simple-list__row">
            <strong>{{ menu.name }}</strong>
            <span>€ {{ euro(menu.price_cents) }}</span>
            <span>{{ menu.description || '—' }}</span>
            <span>{{ (menu.items || []).length }} prodotti</span>
            <button class="btn danger" type="button" @click="deleteMenu(menu.id)">Elimina definitivamente</button>
          </div>
        </div>
      </article>
    </div>


    <div v-else-if="activeTab==='categories'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">🗂️</span>
        <div>
          <h2 class="section-hero-bar__title">Categorie</h2>
          <p class="section-hero-bar__desc">CRUD categorie prodotti BAR con immagine obbligatoria.</p>
        </div>
      </div>

      <article class="info-card">
        <h4>{{ editingCategoryId ? 'Modifica categoria' : 'Nuova categoria' }}</h4>
        <div class="filter-grid">
          <label>Nome<input v-model="newCategory.name" class="input" type="text" /></label>
          <label>Immagine categoria (obbligatoria)
            <input class="input" type="file" accept="image/*" @change="onCategoryImageSelect" />
          </label>
        </div>
        <p v-if="newCategory.image_url" class="muted">Immagine selezionata ✅</p>
        <div class="menu-card__actions">
          <button class="btn primary" type="button" @click="saveCategory">{{ editingCategoryId ? 'Salva modifiche' : 'Crea categoria' }}</button>
          <button v-if="editingCategoryId" class="btn outline" type="button" @click="resetCategoryForm">Annulla modifica</button>
        </div>
      </article>

      <div class="simple-list">
        <div v-for="category in barCategories" :key="`bar-category-${category.id}`" class="simple-list__row">
          <strong>{{ category.name }}</strong>
          <img :src="category.image_url" :alt="category.name" class="h-12 w-16 rounded object-cover" />
          <button class="btn outline" type="button" @click="editCategory(category)">Modifica</button>
          <button class="btn danger" type="button" @click="deleteCategory(category.id)">Elimina</button>
        </div>
        <p v-if="!barCategories.length" class="muted">Nessuna categoria disponibile.</p>
      </div>
    </div>

    <div v-else-if="activeTab==='suggestions'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">💡</span>
        <div>
          <h2 class="section-hero-bar__title">Suggerimenti post-aggiunta</h2>
          <p class="section-hero-bar__desc">Priorità prodotto, fallback categoria. Max 3 suggerimenti.</p>
        </div>
      </div>

      <article class="info-card">
        <h4>Suggerimenti per prodotto</h4>
        <div class="simple-list">
          <div v-for="product in productCards" :key="`sugg-prod-${product.id}`" class="simple-list__row" style="align-items:flex-start;">
            <div style="min-width: 220px;">
              <strong>{{ product.name }}</strong>
              <p class="muted">{{ product.category || 'Senza categoria' }}</p>
            </div>
            <label class="suggestion-toggle" :class="{ 'is-checked': productSuggestionState(product.id).enabled }">
              <input type="checkbox" :checked="productSuggestionState(product.id).enabled" @change="setProductSuggestionEnabled(product.id, $event.target.checked)" />
              Attivi
            </label>
            <input class="input" type="text" placeholder="Titolo box" :value="productSuggestionState(product.id).title" @input="setProductSuggestionTitle(product.id, $event.target.value)" style="min-width:220px" />
            <select class="input" :value="productSuggestionState(product.id).max_items" @change="setProductSuggestionMax(product.id, $event.target.value)">
              <option :value="2">2</option>
              <option :value="3">3</option>
            </select>
            <div>
              <label
                v-for="candidate in productCards.filter((p) => p.id !== product.id)"
                :key="`ps-${product.id}-${candidate.id}`"
                class="suggestion-choice"
                :class="{ 'is-checked': productSuggestionState(product.id).suggestion_ids.includes(candidate.id) }"
              >
                <input type="checkbox" :checked="productSuggestionState(product.id).suggestion_ids.includes(candidate.id)" @change="toggleProductSuggestion(product.id, candidate.id, $event.target.checked)" />
                {{ candidate.name }}
              </label>
            </div>
            <button class="btn primary" type="button" @click="saveProductSuggestion(product.id)">Salva</button>
          </div>
        </div>
      </article>

      <article class="info-card">
        <h4>Fallback per categoria</h4>
        <div class="simple-list">
          <div v-for="category in barCategories" :key="`sugg-cat-${category.id}`" class="simple-list__row" style="align-items:flex-start;">
            <div style="min-width: 220px;"><strong>{{ category.name }}</strong></div>
            <label class="suggestion-toggle" :class="{ 'is-checked': categorySuggestionState(category.id).enabled }">
              <input type="checkbox" :checked="categorySuggestionState(category.id).enabled" @change="setCategorySuggestionEnabled(category.id, $event.target.checked)" />
              Attivi
            </label>
            <input class="input" type="text" placeholder="Titolo fallback" :value="categorySuggestionState(category.id).title" @input="setCategorySuggestionTitle(category.id, $event.target.value)" style="min-width:220px" />
            <select class="input" :value="categorySuggestionState(category.id).max_items" @change="setCategorySuggestionMax(category.id, $event.target.value)">
              <option :value="2">2</option>
              <option :value="3">3</option>
            </select>
            <div>
              <label
                v-for="candidate in productCards"
                :key="`cs-${category.id}-${candidate.id}`"
                class="suggestion-choice"
                :class="{ 'is-checked': categorySuggestionState(category.id).suggestion_ids.includes(candidate.id) }"
              >
                <input type="checkbox" :checked="categorySuggestionState(category.id).suggestion_ids.includes(candidate.id)" @change="toggleCategorySuggestion(category.id, candidate.id, $event.target.checked)" />
                {{ candidate.name }}
              </label>
            </div>
            <button class="btn primary" type="button" @click="saveCategorySuggestion(category.id)">Salva</button>
          </div>
        </div>
      </article>
    </div>


    <div v-else-if="activeTab==='quick'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">⚡</span>
        <div>
          <h2 class="section-hero-bar__title">Disponibilità rapida</h2>
          <p class="section-hero-bar__desc">Aggiorna la disponibilità in pochi secondi durante la partita.</p>
        </div>
      </div>

      <div class="quick-list">
        <article v-for="product in productCards" :key="`quick-${product.id}`" class="quick-card">
          <div>
            <h4>{{ product.name }}</h4>
            <p class="muted">Stato attuale: <strong>{{ product.available ? 'Disponibile' : 'Esaurito' }}</strong></p>
          </div>
          <div class="quick-card__actions">
            <button class="btn quick-btn" :class="product.available ? 'primary' : 'outline'" type="button" @click="setProductAvailability(product.id, true)">Disponibile</button>
            <button class="btn quick-btn" :class="!product.available ? 'danger' : 'outline'" type="button" @click="setProductAvailability(product.id, false)">Esaurito</button>
          </div>
        </article>
      </div>
    </div>

    <div v-else-if="activeTab==='stats'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">📈</span>
        <div>
          <h2 class="section-hero-bar__title">Statistiche</h2>
          <p class="section-hero-bar__desc">Panoramica rapida vendite bar per evento.</p>
        </div>
      </div>

      <div class="kpi-grid">
        <article class="kpi-card" v-for="(card, idx) in statsCards" :key="card.label" :data-kpi-index="idx">
          <div class="kpi-card__icon">{{ statsKpiMeta[idx]?.icon || '📊' }}</div>
          <div class="kpi-card__body">
            <p class="kpi-card__label">{{ card.label }}</p>
            <p class="kpi-card__value">{{ card.value }}</p>
          </div>
        </article>
      </div>

      <article class="info-card">
        <h4>Prodotti più venduti</h4>
        <ul class="item-list compact">
          <li v-for="item in topProducts" :key="item.id">{{ item.name }} • {{ item.qty }} pz</li>
          <li v-if="!topProducts.length" class="muted">Ancora nessun dato disponibile.</li>
        </ul>
      </article>
    </div>

    <div v-else-if="activeTab==='clients'" class="stacked-section clients-layout">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">👥</span>
        <div>
          <h2 class="section-hero-bar__title">Clienti</h2>
          <p class="section-hero-bar__desc">Riepilogo clienti e storico ordini.</p>
        </div>
      </div>

      <div class="clients-grid">
        <article class="info-card">
          <h4>Lista clienti</h4>
          <button
            v-for="client in clientRows"
            :key="client.id"
            class="simple-list__row"
            type="button"
            @click="selectedClientId = client.id"
          >
            <strong>{{ client.name }}</strong>
            <span>{{ client.orders }} ordini</span>
            <span>€ {{ euro(client.totalCents) }}</span>
            <span>Ultimo: {{ formatOrderTime(client.lastOrderAt, true) }}</span>
          </button>
          <p v-if="!clientRows.length" class="muted">Nessun cliente disponibile.</p>
        </article>

        <article class="info-card">
          <h4>Storico cliente</h4>
          <p v-if="!selectedClient" class="muted">Seleziona un cliente per vedere gli ordini.</p>
          <template v-else>
            <p><strong>{{ selectedClient.name }}</strong> • {{ selectedClient.orders }} ordini • € {{ euro(selectedClient.totalCents) }}</p>
            <ul class="item-list compact">
              <li v-for="order in selectedClientOrders" :key="`client-order-${order.id}`">
                <button type="button" class="link-btn" @click="openOrderDetail(order)">
                  #{{ order.id }} • {{ formatOrderTime(order.created_at, true) }} • {{ statusLabel(order.order_status) }} • € {{ euro(order.total_cents) }}
                </button>
              </li>
            </ul>
          </template>
        </article>
      </div>
    </div>

    <div v-else-if="activeTab==='cash'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">💰</span>
        <div>
          <h2 class="section-hero-bar__title">Cassa</h2>
          <p class="section-hero-bar__desc">Riepilogo chiusura giornata o evento.</p>
        </div>
      </div>

      <div class="kpi-grid">
        <article class="kpi-card" v-for="(card, idx) in cashCards" :key="card.label" :data-kpi-index="idx">
          <div class="kpi-card__icon">{{ cashKpiMeta[idx]?.icon || '💰' }}</div>
          <div class="kpi-card__body">
            <p class="kpi-card__label">{{ card.label }}</p>
            <p class="kpi-card__value">{{ card.value }}</p>
          </div>
        </article>
      </div>
    </div>

    <div v-else-if="activeTab==='settings'" class="stacked-section">
      <div class="section-hero-bar">
        <span class="section-hero-bar__icon">⚙️</span>
        <div>
          <h2 class="section-hero-bar__title">Impostazioni BAR</h2>
          <p class="section-hero-bar__desc">Configura preferenze operative del bar.</p>
        </div>
      </div>

      <div class="settings-grid">
        <article class="info-card">
          <h4>Dati bar</h4>
          <p><strong>Partner BAR:</strong> {{ settings.partnerName }}</p>
          <p><strong>Società associata:</strong> {{ settings.companyName }}</p>
          <p class="muted">Accesso dati: ruolo BAR vede solo il proprio bar, super admin vede tutti i bar.</p>
        </article>

        <article class="info-card">
          <h4>Preferenze operative</h4>
          <label class="setting-row">
            <input v-model="settings.notifyNewOrders" type="checkbox" />
            Notifiche nuovi ordini
          </label>
          <label class="setting-row">
            <input v-model="settings.soundAlerts" type="checkbox" />
            Avvisi sonori priorità ordini
          </label>
          <label class="setting-row">
            <input v-model="settings.autoRefreshLive" type="checkbox" />
            Aggiornamento automatico ordini live
          </label>
        </article>
      </div>
    </div>

    <div v-else class="muted">Sezione non disponibile.</div>

    <div v-if="selectedOrder" class="order-modal" role="dialog" aria-modal="true" aria-label="Dettaglio ordine">
      <div class="order-modal__backdrop" @click="closeOrderDetail"></div>
      <div class="order-modal__panel">
        <header class="order-modal__header">
          <div>
            <p class="muted">Dettaglio ordine</p>
            <h3>#{{ selectedOrder.id }} • {{ statusLabel(selectedOrder.order_status) }}</h3>
          </div>
          <button class="btn outline" type="button" @click="closeOrderDetail">Chiudi</button>
        </header>

        <div class="order-modal__grid">
          <p><strong>Creato alle:</strong> {{ formatOrderTime(selectedOrder.created_at, true) }}</p>
          <p><strong>Aggiornato alle:</strong> {{ formatOrderTime(selectedOrder.updated_at, true) }}</p>
          <p><strong>Pagamento:</strong> {{ paymentLabel(selectedOrder.payment_status) }}</p>
          <p><strong>Totale:</strong> € {{ euro(selectedOrder.total_cents) }}</p>
          <p><strong>Cliente:</strong> {{ customerLabel(selectedOrder) }}</p>
          <p><strong>Posizione:</strong> {{ seatingLabel(selectedOrder) }}</p>
        </div>

        <section>
          <h4>Prodotti</h4>
          <ul class="order-modal__items">
            <li v-for="item in selectedOrder.parsedItems" :key="`detail-${selectedOrder.id}-${item.id}`">
              <span>{{ item.name }} • € {{ euro(item.priceCents) }}</span>
              <strong>x{{ item.quantity }}</strong>
            </li>
          </ul>
        </section>

        <section>
          <h4>Note</h4>
          <p class="order-modal__notes">{{ selectedOrder.notes || 'Nessuna nota' }}</p>
        </section>

        <section>
          <h4>Cronologia</h4>
          <ul class="order-modal__timeline">
            <li>
              <span>Creato</span>
              <strong>{{ formatOrderTime(selectedOrder.created_at, true) }}</strong>
            </li>
            <li>
              <span>Ultimo aggiornamento</span>
              <strong>{{ formatOrderTime(selectedOrder.updated_at, true) }}</strong>
            </li>
            <li>
              <span>Stato corrente</span>
              <strong>{{ statusLabel(selectedOrder.order_status) }}</strong>
            </li>
          </ul>
        </section>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { apiClient } from '../api';

const props = defineProps({ authHeaders: { type: Object, required: true }, isSuperAdmin: { type: Boolean, default: false } });
const activeTab = ref('dashboard');
const overview = ref({});
const orders = ref([]);
const isLoading = ref(false);
const selectedOrder = ref(null);
const lastRefreshAt = ref(null);
const productsCatalog = ref([]);
const productState = ref({});
const barMenus = ref([]);
const barCategories = ref([]);
const editingCategoryId = ref(0);
const newCategory = ref({ name: '', image_url: '' });
const newProduct = ref({ name: '', description: '', priceEuro: 0, categoryId: 0, image_url: '' });
const newMenu = ref({ name: '', description: '', priceEuro: 0, items: {} });
const productSuggestions = ref({});
const categorySuggestions = ref({});
const historyDate = ref('');
const historyStatus = ref('');
const historySearch = ref('');
const selectedClientId = ref('');
const settings = ref({
  partnerName: 'BAR Partner',
  companyName: 'Società evento',
  notifyNewOrders: true,
  soundAlerts: true,
  autoRefreshLive: true,
});

const RECENT_CANCELLED_MINUTES = 20;
const refreshEveryMs = 15000;
let refreshTimer = null;

const tabs = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'live', label: 'Ordini live' },
  { id: 'history', label: 'Storico ordini' },
  { id: 'menu', label: 'Menu' },
  { id: 'categories', label: 'Categorie' },
  { id: 'suggestions', label: 'Suggerimenti' },
  { id: 'quick', label: 'Disponibilità rapida' },
  { id: 'stats', label: 'Statistiche' },
  { id: 'clients', label: 'Clienti' },
  { id: 'cash', label: 'Cassa' },
  { id: 'settings', label: 'Impostazioni' },
];

const tabGroups = [
  { label: 'Operativo', tabs: [
    { id: 'dashboard', label: 'Dashboard',   icon: '📊' },
    { id: 'live',      label: 'Live',         icon: '🔴' },
    { id: 'history',   label: 'Storico',      icon: '📋' },
  ]},
  { label: 'Catalogo', tabs: [
    { id: 'menu',        label: 'Menu',          icon: '🍽️' },
    { id: 'categories',  label: 'Categorie',     icon: '🗂️' },
    { id: 'suggestions', label: 'Suggerimenti',  icon: '💡' },
    { id: 'quick',       label: 'Disponibilità', icon: '⚡' },
  ]},
  { label: 'Analisi', tabs: [
    { id: 'stats',   label: 'Statistiche', icon: '📈' },
    { id: 'clients', label: 'Clienti',     icon: '👥' },
    { id: 'cash',    label: 'Cassa',       icon: '💰' },
  ]},
  { label: 'Config', tabs: [
    { id: 'settings', label: 'Impostazioni', icon: '⚙️' },
  ]},
];

const kpiMeta      = [{ icon: '📦' }, { icon: '⏳' }, { icon: '🔧' }, { icon: '✅' }, { icon: '🏁' }, { icon: '💰' }, { icon: '🎫' }];
const statsKpiMeta = [{ icon: '💰' }, { icon: '📦' }, { icon: '🎫' }, { icon: '⏰' }];
const cashKpiMeta  = [{ icon: '💰' }, { icon: '✅' }, { icon: '❌' }, { icon: '🍺' }];
const liveColumnMeta = { new: { icon: '🆕' }, in_preparazione: { icon: '🔧' }, pronto: { icon: '🟢' }, annullato: { icon: '❌' } };

const kpis = computed(() => {
  const o = overview.value || {};
  return [
    { label: 'Ordini ricevuti', value: o.orders_received || 0 },
    { label: 'In attesa', value: o.orders_pending || 0 },
    { label: 'In preparazione', value: o.orders_in_preparation || 0 },
    { label: 'Pronti', value: o.orders_ready || 0 },
    { label: 'Completati', value: o.orders_completed || 0 },
    { label: 'Incasso', value: `€ ${((o.revenue_cents||0)/100).toFixed(2)}` },
    { label: 'Ticket medio', value: `€ ${((o.avg_ticket_cents||0)/100).toFixed(2)}` },
  ];
});

const parsedOrders = computed(() => {
  return orders.value.map((order) => {
    const status = normalizeStatus(order.order_status);
    const createdAt = parseDate(order.created_at);
    const elapsedMinutes = Math.max(0, Math.round((Date.now() - createdAt.getTime()) / 60000));
    return {
      ...order,
      order_status: status,
      elapsedMinutes,
      parsedItems: parseItems(order),
    };
  }).sort((a, b) => parseDate(b.created_at) - parseDate(a.created_at));
});

const historyOrders = computed(() => parsedOrders.value.filter((order) => {
  if (historyStatus.value && order.order_status !== historyStatus.value) {
    return false;
  }
  if (historyDate.value) {
    const day = parseDate(order.created_at).toISOString().slice(0, 10);
    if (day !== historyDate.value) {
      return false;
    }
  }
  if (historySearch.value) {
    return String(order.id).includes(historySearch.value.trim());
  }
  return true;
}));

const liveColumns = computed(() => {
  const active = parsedOrders.value.filter((order) => {
    if (['new', 'in_preparazione', 'pronto'].includes(order.order_status)) {
      return true;
    }
    return order.order_status === 'annullato' && order.elapsedMinutes <= RECENT_CANCELLED_MINUTES;
  });

  const byStatus = {
    new: active.filter((o) => o.order_status === 'new'),
    in_preparazione: active.filter((o) => o.order_status === 'in_preparazione'),
    pronto: active.filter((o) => o.order_status === 'pronto'),
    annullato: active.filter((o) => o.order_status === 'annullato'),
  };

  return [
    { id: 'new', title: 'Nuovi', orders: byStatus.new, emptyText: 'Nessun nuovo ordine' },
    { id: 'in_preparazione', title: 'In preparazione', orders: byStatus.in_preparazione, emptyText: 'Nessun ordine in preparazione' },
    { id: 'pronto', title: 'Pronti', orders: byStatus.pronto, emptyText: 'Nessun ordine pronto al ritiro' },
    { id: 'annullato', title: 'Annullati recenti', orders: byStatus.annullato, emptyText: 'Nessun annullamento recente' },
  ];
});

const productCards = computed(() => productsCatalog.value.map((product) => {
  const state = productState.value[product.id] || { active: true, available: true };
  return { ...product, ...state };
}));

const topProducts = computed(() => {
  const map = new Map();
  parsedOrders.value.forEach((order) => {
    order.parsedItems.forEach((item) => {
      map.set(item.id, { id: item.id, name: item.name, qty: (map.get(item.id)?.qty || 0) + item.quantity });
    });
  });
  return Array.from(map.values()).sort((a, b) => b.qty - a.qty).slice(0, 5);
});

const busiestHour = computed(() => {
  const byHour = new Map();
  parsedOrders.value.forEach((order) => {
    const hour = parseDate(order.created_at).getHours();
    byHour.set(hour, (byHour.get(hour) || 0) + 1);
  });
  const sorted = Array.from(byHour.entries()).sort((a, b) => b[1] - a[1]);
  if (!sorted.length) return 'N/D';
  const [hour, count] = sorted[0];
  return `${String(hour).padStart(2, '0')}:00 • ${count} ordini`;
});

const statsCards = computed(() => {
  const total = parsedOrders.value.reduce((acc, order) => acc + Number(order.total_cents || 0), 0);
  const count = parsedOrders.value.length;
  return [
    { label: 'Incasso totale evento', value: `€ ${euro(total)}` },
    { label: 'Numero totale ordini', value: count },
    { label: 'Ticket medio', value: `€ ${count ? euro(total / count) : '0.00'}` },
    { label: 'Fascia oraria con più ordini', value: busiestHour.value },
  ];
});

const clientRows = computed(() => {
  const map = new Map();
  parsedOrders.value.forEach((order) => {
    const id = customerId(order);
    const current = map.get(id) || {
      id,
      name: customerLabel(order),
      orders: 0,
      totalCents: 0,
      lastOrderAt: order.created_at,
    };
    current.orders += 1;
    current.totalCents += Number(order.total_cents || 0);
    if (parseDate(order.created_at) > parseDate(current.lastOrderAt)) {
      current.lastOrderAt = order.created_at;
    }
    map.set(id, current);
  });
  return Array.from(map.values()).sort((a, b) => b.orders - a.orders);
});

const selectedClient = computed(() => clientRows.value.find((client) => client.id === selectedClientId.value) || null);
const selectedClientOrders = computed(() => {
  if (!selectedClient.value) return [];
  return parsedOrders.value.filter((order) => customerId(order) === selectedClient.value.id);
});

const cashCards = computed(() => {
  const completed = parsedOrders.value.filter((order) => order.order_status === 'completato').length;
  const cancelled = parsedOrders.value.filter((order) => order.order_status === 'annullato').length;
  const soldProducts = parsedOrders.value.reduce((acc, order) => acc + order.parsedItems.reduce((inner, item) => inner + item.quantity, 0), 0);
  const total = parsedOrders.value.reduce((acc, order) => acc + Number(order.total_cents || 0), 0);
  return [
    { label: 'Incasso totale', value: `€ ${euro(total)}` },
    { label: 'Ordini completati', value: completed },
    { label: 'Ordini annullati', value: cancelled },
    { label: 'Prodotti venduti', value: soldProducts },
  ];
});

const lastRefreshLabel = computed(() => {
  if (!lastRefreshAt.value) {
    return 'mai';
  }
  return new Intl.DateTimeFormat('it-IT', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(lastRefreshAt.value);
});

async function loadProducts() {
  const { data } = await apiClient.get('/admin/bar/products', props.authHeaders);
  productsCatalog.value = Array.isArray(data) ? data : [];
  const next = {};
  productsCatalog.value.forEach((product) => {
    next[product.id] = productState.value[product.id] || { active: true, available: true };
  });
  productState.value = next;
}

async function loadCategories() {
  const { data } = await apiClient.get('/admin/bar/categories', props.authHeaders);
  barCategories.value = Array.isArray(data) ? data : [];
}

async function loadMenus() {
  const { data } = await apiClient.get('/admin/bar/menus', props.authHeaders);
  barMenus.value = Array.isArray(data) ? data : [];
}

function normalizeSuggestionState(raw = {}) {
  return {
    enabled: Boolean(raw.enabled),
    title: String(raw.title || ''),
    max_items: [2, 3].includes(Number(raw.max_items)) ? Number(raw.max_items) : 2,
    suggestion_ids: Array.isArray(raw.suggestion_ids) ? raw.suggestion_ids.map((id) => Number(id)).filter((id) => id > 0).slice(0, 3) : [],
  };
}

async function loadSuggestions() {
  const [productResponse, categoryResponse] = await Promise.all([
    apiClient.get('/admin/bar/suggestions/products', props.authHeaders),
    apiClient.get('/admin/bar/suggestions/categories', props.authHeaders),
  ]);
  const nextProduct = {};
  for (const item of (Array.isArray(productResponse.data) ? productResponse.data : [])) {
    nextProduct[Number(item.product_id)] = normalizeSuggestionState(item);
  }
  productSuggestions.value = nextProduct;

  const nextCategory = {};
  for (const item of (Array.isArray(categoryResponse.data) ? categoryResponse.data : [])) {
    nextCategory[Number(item.category_id)] = normalizeSuggestionState(item);
  }
  categorySuggestions.value = nextCategory;
}

function productSuggestionState(productId) {
  return productSuggestions.value[productId] || { enabled: false, title: '', max_items: 2, suggestion_ids: [] };
}

function categorySuggestionState(categoryId) {
  return categorySuggestions.value[categoryId] || { enabled: false, title: '', max_items: 2, suggestion_ids: [] };
}

function patchProductSuggestion(productId, patch) {
  productSuggestions.value = {
    ...productSuggestions.value,
    [productId]: { ...productSuggestionState(productId), ...patch },
  };
}

function patchCategorySuggestion(categoryId, patch) {
  categorySuggestions.value = {
    ...categorySuggestions.value,
    [categoryId]: { ...categorySuggestionState(categoryId), ...patch },
  };
}

function setProductSuggestionEnabled(productId, value) { patchProductSuggestion(productId, { enabled: Boolean(value) }); }
function setProductSuggestionTitle(productId, value) { patchProductSuggestion(productId, { title: String(value || '') }); }
function setProductSuggestionMax(productId, value) { patchProductSuggestion(productId, { max_items: Number(value) === 3 ? 3 : 2 }); }

function setCategorySuggestionEnabled(categoryId, value) { patchCategorySuggestion(categoryId, { enabled: Boolean(value) }); }
function setCategorySuggestionTitle(categoryId, value) { patchCategorySuggestion(categoryId, { title: String(value || '') }); }
function setCategorySuggestionMax(categoryId, value) { patchCategorySuggestion(categoryId, { max_items: Number(value) === 3 ? 3 : 2 }); }

function toggleProductSuggestion(productId, suggestionId, checked) {
  const current = productSuggestionState(productId).suggestion_ids;
  const next = checked ? [...new Set([...current, suggestionId])].slice(0, 3) : current.filter((id) => id !== suggestionId);
  patchProductSuggestion(productId, { suggestion_ids: next });
}

function toggleCategorySuggestion(categoryId, suggestionId, checked) {
  const current = categorySuggestionState(categoryId).suggestion_ids;
  const next = checked ? [...new Set([...current, suggestionId])].slice(0, 3) : current.filter((id) => id !== suggestionId);
  patchCategorySuggestion(categoryId, { suggestion_ids: next });
}

async function saveProductSuggestion(productId) {
  await apiClient.put(`/admin/bar/suggestions/products/${productId}`, productSuggestionState(productId), props.authHeaders);
  await loadSuggestions();
}

async function saveCategorySuggestion(categoryId) {
  await apiClient.put(`/admin/bar/suggestions/categories/${categoryId}`, categorySuggestionState(categoryId), props.authHeaders);
  await loadSuggestions();
}

async function load() {
  isLoading.value = true;
  try {
    const [{ data: ov }, { data: os }] = await Promise.all([
      apiClient.get('/admin/bar/overview', props.authHeaders),
      apiClient.get('/admin/bar/orders', props.authHeaders),
    ]);
    overview.value = ov || {};
    orders.value = Array.isArray(os) ? os : [];
    lastRefreshAt.value = new Date();
    if (selectedOrder.value) {
      selectedOrder.value = parsedOrders.value.find((order) => order.id === selectedOrder.value.id) || null;
    }
    if (!selectedClientId.value && clientRows.value.length) {
      selectedClientId.value = clientRows.value[0].id;
    }
  } finally {
    isLoading.value = false;
  }
}

async function setStatus(orderId, status) {
  await apiClient.put(`/admin/bar/orders/${orderId}/status`, { status }, props.authHeaders);
  await load();
}

async function createProduct() {
  const payload = {
    name: String(newProduct.value.name || '').trim(),
    description: String(newProduct.value.description || '').trim(),
    price_cents: Math.round(Number(newProduct.value.priceEuro || 0) * 100),
    image_url: String(newProduct.value.image_url || '').trim(),
    category_id: Number(newProduct.value.categoryId || 0),
  };
  if (!payload.name || payload.price_cents <= 0 || payload.category_id <= 0) return;
  await apiClient.post('/admin/bar/products', payload, props.authHeaders);
  newProduct.value = { name: '', description: '', priceEuro: 0, categoryId: 0, image_url: '' };
  await loadProducts();
}

async function onProductImageSelect(event) {
  const file = event?.target?.files?.[0];
  if (!file) return;
  newProduct.value.image_url = await fileToDataUrl(file);
}

async function deleteProduct(productId) {
  await apiClient.delete(`/admin/bar/products/${productId}`, props.authHeaders);
  await Promise.all([loadProducts(), loadMenus()]);
}

function isMenuProductSelected(productId) {
  return Number(newMenu.value.items[productId] || 0) > 0;
}

function toggleMenuProduct(productId, checked) {
  newMenu.value.items = {
    ...newMenu.value.items,
    [productId]: checked ? Math.max(1, Number(newMenu.value.items[productId] || 1)) : 0,
  };
}

function setMenuQty(productId, value) {
  const qty = Math.max(1, Number(value || 1));
  newMenu.value.items = { ...newMenu.value.items, [productId]: qty };
}

function menuQty(productId) {
  return Math.max(1, Number(newMenu.value.items[productId] || 1));
}

async function createMenu() {
  const items = Object.entries(newMenu.value.items)
    .filter(([, qty]) => Number(qty) > 0)
    .map(([productId, qty]) => ({ product_id: Number(productId), quantity: Number(qty) }));
  const payload = {
    name: String(newMenu.value.name || '').trim(),
    description: String(newMenu.value.description || '').trim(),
    price_cents: Math.round(Number(newMenu.value.priceEuro || 0) * 100),
    items,
  };
  if (!payload.name || payload.price_cents <= 0 || !payload.items.length) return;
  await apiClient.post('/admin/bar/menus', payload, props.authHeaders);
  newMenu.value = { name: '', description: '', priceEuro: 0, items: {} };
  await loadMenus();
}


async function onCategoryImageSelect(event) {
  const file = event?.target?.files?.[0];
  if (!file) return;
  newCategory.value.image_url = await fileToDataUrl(file);
}

function fileToDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

function editCategory(category) {
  editingCategoryId.value = Number(category.id || 0);
  newCategory.value = { name: String(category.name || ''), image_url: String(category.image_url || '') };
}

function resetCategoryForm() {
  editingCategoryId.value = 0;
  newCategory.value = { name: '', image_url: '' };
}

async function saveCategory() {
  const payload = { name: String(newCategory.value.name || '').trim(), image_url: String(newCategory.value.image_url || '').trim() };
  if (!payload.name || !payload.image_url) return;
  if (editingCategoryId.value > 0) {
    await apiClient.put(`/admin/bar/categories/${editingCategoryId.value}`, payload, props.authHeaders);
  } else {
    await apiClient.post('/admin/bar/categories', payload, props.authHeaders);
  }
  resetCategoryForm();
  await loadCategories();
}

async function deleteCategory(categoryId) {
  await apiClient.delete(`/admin/bar/categories/${categoryId}`, props.authHeaders);
  await Promise.all([loadCategories(), loadProducts()]);
}

async function deleteMenu(menuId) {
  await apiClient.delete(`/admin/bar/menus/${menuId}`, props.authHeaders);
  await loadMenus();
}

function setProductAvailability(productId, available) {
  const prev = productState.value[productId] || { active: true, available: true };
  productState.value = {
    ...productState.value,
    [productId]: { ...prev, available },
  };
}

function parseItems(order) {
  const products = safeJson(order.products || order.products_json || '[]');
  const quantities = safeJson(order.quantities || order.quantities_json || '[]');

  const quantityMap = new Map();
  quantities.forEach((row) => {
    if (row && row.id) {
      quantityMap.set(String(row.id), Number(row.quantity || 0));
    }
  });

  return products.map((product) => {
    const id = String(product.id || product.name || Math.random());
    const priceCents = Number(product.price_cents || product.priceCents || 0);
    const quantity = Math.max(1, quantityMap.get(id) || 1);
    return {
      id,
      name: product.name || product.id || 'Prodotto',
      quantity,
      priceCents,
      lineTotalCents: priceCents * quantity,
    };
  });
}

function safeJson(raw) {
  try {
    const parsed = JSON.parse(String(raw || '[]'));
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    return [];
  }
}

function parseDate(raw) {
  const parsed = new Date(raw || Date.now());
  if (Number.isNaN(parsed.getTime())) {
    return new Date();
  }
  return parsed;
}

function normalizeStatus(status) {
  const s = String(status || '').toLowerCase().trim();
  if (s === 'pending' || s === 'paid') {
    return 'new';
  }
  return s || 'new';
}

function statusLabel(status) {
  const labels = {
    new: 'Nuovo',
    pending: 'Nuovo',
    in_preparazione: 'In preparazione',
    pronto: 'Pronto',
    completato: 'Completato',
    annullato: 'Annullato',
  };
  return labels[status] || status;
}

function paymentLabel(status) {
  const labels = {
    paid: 'Pagato',
    pending: 'Pagamento in attesa',
    cancelled: 'Pagamento annullato',
    expired: 'Pagamento scaduto',
  };
  return labels[String(status || '').toLowerCase()] || 'Pagamento n/d';
}

function paymentClass(status) {
  const s = String(status || '').toLowerCase();
  if (s === 'paid') {
    return 'ok';
  }
  if (s === 'pending') {
    return 'wait';
  }
  return 'alert';
}

function euro(value) {
  return (Number(value || 0) / 100).toFixed(2);
}

function formatOrderTime(raw, withDate = false) {
  const date = parseDate(raw);
  return new Intl.DateTimeFormat('it-IT', {
    day: withDate ? '2-digit' : undefined,
    month: withDate ? '2-digit' : undefined,
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function customerId(order) {
  return String(order.customer_name || order.stripe_reference || `ordine-${order.id}`);
}

function customerLabel(order) {
  if (order.customer_name) {
    return order.customer_name;
  }
  return `Ticket ${String(order.stripe_reference || order.id).slice(-8)}`;
}

function seatingLabel(order) {
  const parts = [order.sector, order.row, order.seat].filter(Boolean);
  return parts.length ? parts.join(' • ') : 'Non disponibile';
}

function urgencyClass(minutes) {
  if (minutes >= 18) {
    return 'order-card--urgent';
  }
  if (minutes >= 10) {
    return 'order-card--attention';
  }
  return 'order-card--fresh';
}

function urgencyBadgeClass(minutes) {
  if (minutes >= 18) {
    return 'badge--urgent';
  }
  if (minutes >= 10) {
    return 'badge--attention';
  }
  return 'badge--ok';
}

function nextStatusAction(status) {
  const actions = {
    new: { status: 'in_preparazione', label: 'In preparazione' },
    in_preparazione: { status: 'pronto', label: 'Segna pronto' },
    pronto: { status: 'completato', label: 'Completa' },
  };
  return actions[status] || null;
}

function canCancel(status) {
  return ['new', 'in_preparazione'].includes(status);
}

function openOrderDetail(order) {
  selectedOrder.value = order;
}

function closeOrderDetail() {
  selectedOrder.value = null;
}

onMounted(async () => {
  await Promise.all([load(), loadCategories(), loadProducts(), loadMenus(), loadSuggestions()]);
  refreshTimer = window.setInterval(() => {
    if (settings.value.autoRefreshLive) {
      load();
    }
  }, refreshEveryMs);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
});
</script>

<style scoped>
/* ═══════════════════════════════════════════════════
   1. SCOPED ROOT
═══════════════════════════════════════════════════ */
.bar-admin {
  display: grid;
  gap: 0;
  background: var(--bg-base, #f1f5f9);
  min-height: 600px;
  border-radius: var(--radius-lg, 16px);
  overflow: hidden;
}

/* ═══════════════════════════════════════════════════
   2. PAGE HERO HEADER
═══════════════════════════════════════════════════ */
.bar-hero {
  background: var(--bg-card, #ffffff);
  border-bottom: 1px solid var(--border, rgba(15,23,42,0.1));
  padding: 1.25rem 1.75rem 0;
  display: grid;
  gap: 0;
}

.bar-hero__identity {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding-bottom: 1rem;
}

.bar-hero__icon {
  font-size: 2rem;
  line-height: 1;
  flex-shrink: 0;
}

.bar-hero__title {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.6rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-primary, #0f172a);
  margin: 0 0 0.15rem;
}

.bar-hero__subtitle {
  font-size: 0.8rem;
  color: var(--text-secondary, #475569);
  margin: 0;
}

.bar-hero__live-dot {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.3rem 0.75rem;
  border-radius: 999px;
  border: 1px solid rgba(148,163,184,0.25);
  background: #f8fafc;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted, #94a3b8);
  flex-shrink: 0;
}

.bar-hero__live-dot--active {
  border-color: rgba(34,197,94,0.35);
  background: rgba(34,197,94,0.08);
  color: #166534;
}

.bar-hero__live-pulse {
  display: block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #94a3b8;
  flex-shrink: 0;
}

.bar-hero__live-dot--active .bar-hero__live-pulse {
  background: #22c55e;
  animation: pulse-dot 1.8s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% { box-shadow: 0 0 0 0 rgba(34,197,94,0.5); }
  50%       { box-shadow: 0 0 0 5px rgba(34,197,94,0); }
}

.bar-hero__live-label { letter-spacing: 0.04em; }

/* ═══════════════════════════════════════════════════
   3. GROUPED TAB NAVIGATION
═══════════════════════════════════════════════════ */
.bar-nav {
  display: flex;
  gap: 0;
  overflow-x: auto;
  scrollbar-width: none;
}

.bar-nav::-webkit-scrollbar { display: none; }

.bar-nav__group {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border, rgba(15,23,42,0.1));
  padding: 0 0.25rem;
  min-width: 0;
}

.bar-nav__group:last-child { border-right: none; }

.bar-nav__group-label {
  font-size: 0.6rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted, #94a3b8);
  padding: 0.5rem 0.5rem 0.25rem;
  white-space: nowrap;
}

.bar-nav__tabs {
  display: flex;
  gap: 0.1rem;
  align-items: flex-end;
}

.bar-nav__tab {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
  padding: 0.5rem 0.7rem 0.6rem;
  border: none;
  background: transparent;
  border-radius: 8px 8px 0 0;
  cursor: pointer;
  position: relative;
  transition: background 0.15s, color 0.15s;
  white-space: nowrap;
  color: var(--text-secondary, #475569);
}

.bar-nav__tab:hover {
  background: rgba(2,132,199,0.06);
  color: var(--accent, #0284c7);
}

.bar-nav__tab--active {
  background: var(--bg-base, #f1f5f9);
  color: var(--accent, #0284c7);
}

.bar-nav__tab--active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0.5rem;
  right: 0.5rem;
  height: 2px;
  background: var(--accent, #0284c7);
  border-radius: 2px 2px 0 0;
}

.bar-nav__tab-icon {
  font-size: 1.1rem;
  line-height: 1;
}

.bar-nav__tab-label {
  font-size: 0.68rem;
  font-weight: 600;
  letter-spacing: 0.04em;
}

/* ═══════════════════════════════════════════════════
   4. TAB PANEL WRAPPERS
═══════════════════════════════════════════════════ */
.tab-panel,
.stacked-section,
.live-board-wrap {
  padding: 1.5rem 1.75rem;
  display: grid;
  gap: 1.25rem;
}

/* ═══════════════════════════════════════════════════
   5. SECTION HERO BAR (per-tab inline header)
═══════════════════════════════════════════════════ */
.section-hero-bar {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
}

.section-hero-bar__icon {
  font-size: 1.75rem;
  flex-shrink: 0;
  line-height: 1;
}

.section-hero-bar__title {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.2rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-primary, #0f172a);
  margin: 0 0 0.15rem;
}

.section-hero-bar__desc {
  font-size: 0.8rem;
  color: var(--text-secondary, #475569);
  margin: 0;
  line-height: 1.4;
}

/* ═══════════════════════════════════════════════════
   6. KPI CARDS
═══════════════════════════════════════════════════ */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(175px, 1fr));
  gap: 0.85rem;
}

.kpi-card {
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-left-width: 4px;
  border-left-color: var(--accent, #0284c7);
  border-radius: var(--radius-md, 12px);
  padding: 1.1rem 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
}

.kpi-card__icon {
  font-size: 1.5rem;
  line-height: 1;
  flex-shrink: 0;
}

.kpi-card__body { min-width: 0; }

.kpi-card__label {
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: var(--text-muted, #94a3b8);
  margin: 0 0 0.2rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kpi-card__value {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.6rem;
  font-weight: 800;
  color: var(--text-primary, #0f172a);
  margin: 0;
  line-height: 1;
  letter-spacing: 0.02em;
}

/* Per-index KPI left-border colors */
.kpi-card[data-kpi-index="0"] { border-left-color: #0284c7; }
.kpi-card[data-kpi-index="1"] { border-left-color: #d97706; }
.kpi-card[data-kpi-index="2"] { border-left-color: #f97316; }
.kpi-card[data-kpi-index="3"] { border-left-color: #22c55e; }
.kpi-card[data-kpi-index="4"] { border-left-color: #6366f1; }
.kpi-card[data-kpi-index="5"] { border-left-color: #eab308; }
.kpi-card[data-kpi-index="6"] { border-left-color: #38bdf8; }

/* ═══════════════════════════════════════════════════
   7. INFO CARDS
═══════════════════════════════════════════════════ */
.info-card {
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  padding: 1.25rem;
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
  display: grid;
  gap: 0.85rem;
}

.info-card h4,
.info-card__title {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-primary, #0f172a);
  margin: 0;
}

.info-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

/* ═══════════════════════════════════════════════════
   8. FILTER CARD
═══════════════════════════════════════════════════ */
.filter-card {
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  padding: 1rem 1.25rem;
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
}

.filter-card__label {
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-muted, #94a3b8);
  margin: 0 0 0.75rem;
}

.filter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.85rem;
}

/* ═══════════════════════════════════════════════════
   9. FORM ELEMENTS
═══════════════════════════════════════════════════ */
.input {
  width: 100%;
  border: 1px solid rgba(2,132,199,0.2);
  border-radius: var(--radius-sm, 8px);
  padding: 0.55rem 0.8rem;
  background: #f8fafc;
  font-family: 'IBM Plex Sans', system-ui, sans-serif;
  font-size: 0.875rem;
  color: var(--text-primary, #0f172a);
  margin-top: 0.3rem;
  transition: border-color 0.18s, box-shadow 0.18s;
}

.input:focus {
  outline: none;
  border-color: var(--accent, #0284c7);
  box-shadow: 0 0 0 3px rgba(2,132,199,0.1);
}

.category-radio-group {
  border: 1px solid rgba(2,132,199,0.2);
  border-radius: var(--radius-sm, 8px);
  padding: 0.65rem 0.85rem;
  background: #f8fafc;
  display: grid;
  gap: 0.45rem;
}

.category-radio-group legend {
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-secondary, #475569);
  padding: 0 0.25rem;
}

.category-radio-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
}

.category-radio-option input { margin: 0; }

/* ═══════════════════════════════════════════════════
   10. LIST ROWS
═══════════════════════════════════════════════════ */
.simple-list {
  display: grid;
  gap: 0.45rem;
}

.simple-list__row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
  gap: 0.5rem;
  align-items: center;
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-sm, 8px);
  padding: 0.7rem 0.9rem;
  text-align: left;
  font-size: 0.875rem;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
  width: 100%;
}

.simple-list__row:hover {
  border-color: rgba(2,132,199,0.3);
  box-shadow: 0 2px 8px rgba(2,132,199,0.08);
}

/* Order list (dashboard latest orders) */
.order-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 0.4rem;
}

.order-list__row {
  display: grid;
  grid-template-columns: 3.5rem 1fr 5rem 1fr;
  gap: 0.65rem;
  align-items: center;
  padding: 0.55rem 0.75rem;
  border-radius: var(--radius-sm, 8px);
  background: #f8fafc;
  border: 1px solid var(--border, rgba(15,23,42,0.06));
  font-size: 0.82rem;
}

.order-list__id {
  font-weight: 700;
  color: var(--accent, #0284c7);
}

.order-list__price {
  font-weight: 700;
  color: var(--text-primary, #0f172a);
  text-align: right;
}

.order-list__status {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary, #475569);
}

.order-list__seat {
  font-size: 0.75rem;
}

/* ═══════════════════════════════════════════════════
   11. MENU GRID & CARDS
═══════════════════════════════════════════════════ */
.menu-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.85rem;
}

.menu-card {
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  padding: 1rem;
  display: grid;
  gap: 0.6rem;
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
  transition: border-color 0.15s, transform 0.15s;
}

.menu-card:hover {
  border-color: rgba(2,132,199,0.25);
  transform: translateY(-2px);
}

.menu-card h4 {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-primary, #0f172a);
  margin: 0;
}

.menu-card__price {
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--accent-gold, #d97706);
  margin: 0;
}

.menu-card__badges,
.menu-card__actions {
  display: flex;
  gap: 0.45rem;
  flex-wrap: wrap;
}

/* ═══════════════════════════════════════════════════
   12. QUICK AVAILABILITY
═══════════════════════════════════════════════════ */
.quick-list {
  display: grid;
  gap: 0.65rem;
}

.quick-card {
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  padding: 1rem 1.25rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
}

.quick-card h4 {
  font-weight: 700;
  margin: 0 0 0.2rem;
  color: var(--text-primary, #0f172a);
}

.quick-card__actions {
  display: flex;
  gap: 0.5rem;
}

.quick-btn {
  font-size: 0.95rem;
  min-width: 130px;
  min-height: 42px;
}

/* ═══════════════════════════════════════════════════
   13. LIVE BOARD
═══════════════════════════════════════════════════ */
.live-toolbar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
  background: var(--bg-card, #ffffff);
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  padding: 0.75rem 1.25rem;
  box-shadow: var(--shadow-card, 0 8px 32px rgba(15,23,42,0.08));
}

.live-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.live-columns {
  display: grid;
  grid-template-columns: repeat(4, minmax(260px, 1fr));
  gap: 1rem;
  overflow-x: auto;
  padding-bottom: 0.5rem;
}

.live-column {
  background: #f8fafc;
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: var(--radius-md, 12px);
  min-height: 320px;
  padding: 0.85rem;
  display: grid;
  gap: 0.75rem;
  align-content: start;
}

.live-column__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.65rem 0.85rem;
  border-radius: var(--radius-sm, 8px);
}

.live-column__header-left {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.live-column__header-icon {
  font-size: 1rem;
  line-height: 1;
}

.live-column__title {
  margin: 0;
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 0.95rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.live-column__header--new             { background: #dbeafe; color: #1e3a5f; }
.live-column__header--in_preparazione { background: #fef3c7; color: #78350f; }
.live-column__header--pronto          { background: #dcfce7; color: #14532d; }
.live-column__header--annullato       { background: #f1f5f9; color: var(--text-secondary, #475569); }

.live-column__count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.75rem;
  height: 1.75rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 800;
  flex-shrink: 0;
}

.live-column__count--new             { background: #1e40af; color: #ffffff; }
.live-column__count--in_preparazione { background: #92400e; color: #ffffff; }
.live-column__count--pronto          { background: #166534; color: #ffffff; }
.live-column__count--annullato       { background: var(--text-muted, #94a3b8); color: #ffffff; }

.live-column__empty {
  border: 1px dashed rgba(15,23,42,0.15);
  border-radius: var(--radius-sm, 8px);
  padding: 1.25rem;
  color: var(--text-muted, #94a3b8);
  text-align: center;
  font-size: 0.85rem;
}

/* ═══════════════════════════════════════════════════
   14. ORDER CARDS
═══════════════════════════════════════════════════ */
.order-card {
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-left-width: 4px;
  border-radius: var(--radius-sm, 8px);
  background: var(--bg-card, #ffffff);
  padding: 0.85rem;
  display: grid;
  gap: 0.6rem;
  cursor: pointer;
  transition: box-shadow 0.15s, transform 0.15s;
}

.order-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 16px rgba(15,23,42,0.1);
}

.order-card__header { display: flex; justify-content: space-between; gap: 0.5rem; }
.order-card__id    { margin: 0; font-size: 1.1rem; font-weight: 800; color: var(--text-primary, #0f172a); }
.order-card__time  { margin: 0; color: var(--text-muted, #94a3b8); font-size: 0.85rem; }
.order-card__badges { display: flex; gap: 0.35rem; flex-wrap: wrap; justify-content: flex-end; }
.order-card__meta p { margin: 0 0 0.2rem; font-size: 0.85rem; color: var(--text-primary, #0f172a); }
.order-card__items  { margin: 0; padding-left: 1rem; display: grid; gap: 0.18rem; font-size: 0.85rem; }
.order-card__notes  { margin: 0; padding: 0.5rem 0.65rem; border-radius: 6px; background: #fffbeb; color: #854d0e; font-size: 0.85rem; }
.order-card__actions { display: flex; gap: 0.4rem; flex-wrap: wrap; }

.order-card--new            { border-left-color: #3b82f6; }
.order-card--in_preparazione { border-left-color: #f59e0b; }
.order-card--pronto         { border-left-color: #22c55e; }
.order-card--annullato      { border-left-color: #94a3b8; }

.order-card--attention { box-shadow: 0 0 0 2px rgba(245,158,11,0.22); }
.order-card--urgent    {
  box-shadow: 0 0 0 2px rgba(239,68,68,0.28);
  animation: urgent-pulse 2s ease infinite;
}

@keyframes urgent-pulse {
  0%, 100% { box-shadow: 0 0 0 2px rgba(239,68,68,0.28); }
  50%       { box-shadow: 0 0 0 4px rgba(239,68,68,0.12); }
}

/* ═══════════════════════════════════════════════════
   15. BADGES
═══════════════════════════════════════════════════ */
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0.2rem 0.5rem;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.badge--ok        { background: #dcfce7; color: #166534; }
.badge--wait      { background: #fef3c7; color: #92400e; }
.badge--alert     { background: #fee2e2; color: #b91c1c; }
.badge--attention { background: #ffedd5; color: #9a3412; }
.badge--urgent    { background: #fee2e2; color: #b91c1c; }

/* ═══════════════════════════════════════════════════
   16. SUGGESTIONS
═══════════════════════════════════════════════════ */
.suggestion-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.3rem 0.6rem;
  border: 1px solid #cbd5e1;
  border-radius: 999px;
  background: #f8fafc;
  font-weight: 600;
  color: var(--text-secondary, #475569);
  cursor: pointer;
  transition: all 0.15s ease;
}

.suggestion-toggle.is-checked {
  background: #dcfce7;
  border-color: #22c55e;
  color: #166534;
  box-shadow: 0 0 0 2px rgba(34,197,94,0.14);
}

.suggestion-choice {
  display: block;
  font-size: 0.78rem;
  padding: 0.25rem 0.45rem;
  border: 1px solid transparent;
  border-radius: 6px;
  transition: all 0.15s ease;
  color: #334155;
  cursor: pointer;
}

.suggestion-choice.is-checked {
  background: #eff6ff;
  border-color: #93c5fd;
  color: #1e40af;
  font-weight: 600;
}

/* ═══════════════════════════════════════════════════
   17. CLIENTS & SETTINGS
═══════════════════════════════════════════════════ */
.clients-layout { display: grid; gap: 1.25rem; }
.clients-grid   { display: grid; grid-template-columns: 1.1fr 1fr; gap: 0.85rem; }
.settings-grid  { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 0.85rem; }

.setting-row {
  display: flex;
  gap: 0.6rem;
  align-items: center;
  padding: 0.45rem 0;
  font-size: 0.875rem;
  color: var(--text-primary, #0f172a);
  border-bottom: 1px solid var(--border, rgba(15,23,42,0.06));
}

.setting-row:last-child { border-bottom: none; }

/* ═══════════════════════════════════════════════════
   18. ORDER MODAL
═══════════════════════════════════════════════════ */
.order-modal {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: grid;
  place-items: center;
}

.order-modal__backdrop {
  position: absolute;
  inset: 0;
  background: rgba(15,23,42,0.55);
  backdrop-filter: blur(2px);
}

.order-modal__panel {
  position: relative;
  width: min(860px, 96vw);
  max-height: 92vh;
  overflow: auto;
  background: var(--bg-card, #ffffff);
  border-radius: var(--radius-lg, 16px);
  padding: 1.5rem;
  display: grid;
  gap: 1.25rem;
  box-shadow: 0 24px 80px rgba(15,23,42,0.18);
}

.order-modal__header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border, rgba(15,23,42,0.1));
}

.order-modal__header h3 {
  font-family: 'Barlow Condensed', 'Impact', sans-serif;
  font-size: 1.3rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  margin: 0;
}

.order-modal__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.65rem;
}

.order-modal__grid p {
  margin: 0;
  font-size: 0.875rem;
  padding: 0.5rem 0.65rem;
  background: #f8fafc;
  border-radius: 6px;
  border: 1px solid var(--border, rgba(15,23,42,0.06));
}

.order-modal__items {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.45rem;
}

.order-modal__items li {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0.65rem;
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  border-radius: 6px;
  font-size: 0.875rem;
}

.order-modal__notes {
  margin: 0;
  padding: 0.75rem;
  border-radius: var(--radius-sm, 8px);
  background: #f8fafc;
  border: 1px solid var(--border, rgba(15,23,42,0.1));
  font-size: 0.875rem;
  color: var(--text-secondary, #475569);
}

.order-modal__timeline {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.4rem;
}

.order-modal__timeline li {
  display: flex;
  justify-content: space-between;
  border-bottom: 1px dashed var(--border, rgba(15,23,42,0.1));
  padding-bottom: 0.35rem;
  font-size: 0.875rem;
}

/* ═══════════════════════════════════════════════════
   19. UTILITIES & RESPONSIVE
═══════════════════════════════════════════════════ */
.muted { color: var(--text-muted, #94a3b8); }

.link-btn {
  border: none;
  background: none;
  color: var(--accent, #0284c7);
  padding: 0;
  text-decoration: underline;
  cursor: pointer;
  font-size: 0.875rem;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  border-radius: var(--radius-sm, 8px);
  border: 0;
  padding: 0.55rem 1.15rem;
  font-family: 'IBM Plex Sans', system-ui, sans-serif;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s ease;
  white-space: nowrap;
}

.btn:disabled { opacity: 0.5; cursor: not-allowed; }

.btn.primary {
  background: linear-gradient(135deg, #0284c7, #4f46e5);
  color: #fff;
  box-shadow: 0 4px 20px rgba(2,132,199,0.3);
}

.btn.primary:hover {
  box-shadow: 0 6px 28px rgba(2,132,199,0.5);
  transform: translateY(-1px);
}

.btn.danger {
  background: rgba(239,68,68,0.1);
  color: #ef4444;
  border: 1px solid rgba(239,68,68,0.2);
}

.btn.danger:hover { background: rgba(239,68,68,0.2); }

.btn.outline {
  background: transparent;
  color: var(--text-muted, #94a3b8);
  border: 1px solid rgba(148,163,184,0.3);
}

.btn.outline:hover {
  border-color: rgba(2,132,199,0.35);
  color: var(--accent, #0284c7);
}

.item-list.compact {
  margin: 0;
  padding-left: 1.25rem;
  display: grid;
  gap: 0.25rem;
  font-size: 0.85rem;
}

.item-list.compact li { color: var(--text-secondary, #475569); }

@media (max-width: 1100px) {
  .live-columns { grid-template-columns: repeat(4, minmax(290px, 1fr)); }
}

@media (max-width: 900px) {
  .bar-nav { flex-direction: column; }
  .bar-nav__group {
    flex-direction: row;
    align-items: center;
    border-right: none;
    border-bottom: 1px solid var(--border, rgba(15,23,42,0.1));
    flex-wrap: wrap;
    padding: 0.25rem 0;
  }
  .bar-nav__group:last-child { border-bottom: none; }
  .bar-nav__group-label { min-width: 80px; }
  .bar-nav__tabs { flex-wrap: wrap; }
  .simple-list__row { grid-template-columns: 1fr; }
  .clients-grid { grid-template-columns: 1fr; }
  .kpi-grid { grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); }
  .order-list__row { grid-template-columns: 3.5rem 1fr 5rem; }
}

@media (max-width: 640px) {
  .bar-hero__identity { flex-wrap: wrap; }
  .bar-hero__live-dot { margin-left: 0; }
  .stacked-section,
  .tab-panel,
  .live-board-wrap { padding: 1rem; }
}
</style>

<template>
  <div class="master-portal">
    <header class="master-header">
      <div>
        <p class="eyebrow">Portale master</p>
        <h1>Controllo centrale</h1>
        <p class="subtitle">Monitora le società e lo stato dei voti in tutta la piattaforma.</p>
      </div>
      <div class="header-actions" v-if="isSuperAdmin">
        <a class="btn ghost" href="/admin" title="Vai al pannello società">Portale società</a>
        <button class="btn outline" type="button" @click="logout">Esci</button>
      </div>
    </header>

    <section v-if="!isAuthenticated" class="card login-card">
      <h2>Accedi come super admin</h2>
      <form class="form-grid" @submit.prevent="login">
        <label>
          Username
          <input v-model.trim="loginForm.username" type="text" autocomplete="username" required />
        </label>
        <label>
          Password
          <input v-model="loginForm.password" type="password" autocomplete="current-password" required />
        </label>
        <button class="btn primary" type="submit" :disabled="isLoggingIn">
          {{ isLoggingIn ? "Accesso in corso…" : "Entra" }}
        </button>
      </form>
      <p v-if="loginError" class="error">{{ loginError }}</p>
    </section>

    <section v-else-if="!isSuperAdmin" class="card warning-card">
      <h2>Accesso limitato</h2>
      <p>Solo gli utenti con ruolo <strong>superadmin</strong> possono accedere al portale master.</p>
      <div class="warning-actions">
        <a class="btn ghost" href="/admin">Vai al portale società</a>
        <button class="btn outline" type="button" @click="logout">Esci</button>
      </div>
    </section>

    <section v-else class="master-shell">
      <nav class="master-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['nav-btn', { active: activeSection === tab.id }]"
          type="button"
          @click="switchSection(tab.id)"
        >
          {{ tab.label }}
        </button>
        <button
          v-if="selectedOrganizationId && activeSection === 'organization-detail'"
          class="nav-btn"
          type="button"
          @click="switchSection('organizations')"
        >
          Torna alla lista
        </button>
      </nav>

      <div class="master-content">
        <div v-if="activeSection === 'dashboard'" class="dashboard-view">
          <div class="grid-cards">
            <article class="stat-card" aria-live="polite">
              <p class="label">Società registrate</p>
              <p class="value">{{ summary.total_organizations ?? 0 }}</p>
              <button class="btn ghost" type="button" @click="refreshDashboard" :disabled="isLoadingSummary || isLoadingAnalytics">
                {{ isLoadingSummary || isLoadingAnalytics ? 'Aggiornamento…' : 'Aggiorna' }}
              </button>
            </article>
            <article class="stat-card">
              <p class="label">Totale voti</p>
              <p class="value">{{ summary.total_votes ?? 0 }}</p>
              <small>Storico complessivo</small>
            </article>
            <article class="stat-card">
              <p class="label">Voti ultimi 7 giorni</p>
              <p class="value">{{ summary.votes_last_7_days ?? 0 }}</p>
              <small>Monitoraggio attività recente</small>
            </article>
            <article class="stat-card">
              <p class="label">Totale partite</p>
              <p class="value">{{ summary.total_events ?? 0 }}</p>
              <small>Eventi registrati nel sistema</small>
            </article>
            <article class="stat-card">
              <p class="label">Impression sponsor</p>
              <p class="value">{{ masterAnalytics.sponsor_stats.total_impressions.toLocaleString('it-IT') }}</p>
              <small>Totale esposizioni registrate</small>
            </article>
            <article class="stat-card">
              <p class="label">CTR medio sponsor</p>
              <p class="value">{{ formatPercent(masterAnalytics.sponsor_stats.average_ctr) }}</p>
              <small>Click-through rate complessivo</small>
            </article>
            <article class="stat-card">
              <p class="label">Tempo totale utenti</p>
              <p class="value">
                {{ formatDurationSeconds(masterAnalytics.engagement.total_duration_seconds) }}
              </p>
              <small>Somma del tempo trascorso sulle pagine evento</small>
            </article>
            <article class="stat-card">
              <p class="label">Tempo medio per utente</p>
              <p class="value">
                {{ formatDurationSeconds(masterAnalytics.engagement.average_duration_per_user) }}
              </p>
              <small>Sessione media per tifoso</small>
            </article>
            <article class="stat-card">
              <p class="label">Tempo medio per partita</p>
              <p class="value">
                {{ formatDurationSeconds(masterAnalytics.engagement.average_duration_per_match) }}
              </p>
              <small>Tempo medio complessivo per match</small>
            </article>
            <article class="stat-card highlight">
              <p class="label">Voti mese corrente</p>
              <p class="value">{{ masterAnalytics.monthly_summary.current.votes.toLocaleString('it-IT') }}</p>
              <small>
                {{ masterAnalytics.monthly_summary.current.month || 'mese' }} ·
                <span :class="['delta', resolveDeltaClass(masterAnalytics.monthly_summary.votes_change.absolute)]">
                  {{ formatDelta(masterAnalytics.monthly_summary.votes_change) }}
                </span>
              </small>
            </article>
          </div>

          <p v-if="analyticsError" class="error">{{ analyticsError }}</p>

          <section class="card analytics-card">
            <header class="section-header">
              <div>
                <h2>Classifica società</h2>
                <p>Andamento voti e crescita negli ultimi 7 giorni.</p>
              </div>
              <button class="btn outline" type="button" @click="refreshDashboard" :disabled="isLoadingAnalytics">
                {{ isLoadingAnalytics ? 'Aggiornamento…' : 'Aggiorna dati' }}
              </button>
            </header>
            <div class="table-wrapper">
              <table>
                <thead>
                  <tr>
                    <th>Società</th>
                    <th>Voti totali</th>
                    <th>Ultimi 7 giorni</th>
                    <th>Eventi</th>
                    <th>Crescita</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="entry in masterAnalytics.organization_leaderboard" :key="entry.organization_id">
                    <td>
                      <div class="org-cell">
                        <img v-if="entry.logo_url" :src="entry.logo_url" :alt="`Logo ${entry.name}`" />
                        <div>
                          <p class="org-name">{{ entry.name }}</p>
                          <small class="muted">{{ entry.city || '—' }}</small>
                        </div>
                      </div>
                    </td>
                    <td>{{ entry.total_votes.toLocaleString('it-IT') }}</td>
                    <td>{{ entry.votes_last_7_days.toLocaleString('it-IT') }}</td>
                    <td>{{ entry.total_events.toLocaleString('it-IT') }}</td>
                    <td>
                      <span :class="['delta', resolveDeltaClass(entry.growth_percentage)]">
                        {{ formatPercent(entry.growth_percentage) }}
                      </span>
                    </td>
                  </tr>
                  <tr v-if="!masterAnalytics.organization_leaderboard.length">
                    <td colspan="5" class="muted">Nessuna società disponibile.</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="card analytics-card">
            <header class="section-header">
              <div>
                <h2>Andamento voti (ultimi 30 giorni)</h2>
                <p>Trend giornaliero globale e per singola società.</p>
              </div>
            </header>
            <div class="trend-grid">
              <div class="trend-panel">
                <h3>Totale piattaforma</h3>
                <VoteTrendChart
                  :points="buildChartPoints(masterAnalytics.vote_trends.global)"
                  :width="520"
                  :height="180"
                  accessible-label="Andamento voti ultimi 30 giorni - globale"
                />
              </div>
              <div class="trend-panel">
                <h3>Per società</h3>
                <div class="trend-list">
                  <div
                    v-for="orgTrend in masterAnalytics.vote_trends.per_organization"
                    :key="orgTrend.organization_id"
                    class="trend-list__item"
                  >
                    <div class="trend-list__header">
                      <strong>{{ orgTrend.name || 'Società' }}</strong>
                      <small class="muted">{{ orgTrend.slug || `ID ${orgTrend.organization_id}` }}</small>
                    </div>
                    <VoteTrendChart
                      :points="buildChartPoints(orgTrend.data)"
                      :width="360"
                      :height="120"
                      :accessible-label="`Andamento voti ultimi 30 giorni - ${orgTrend.name || 'società'}`"
                    />
                  </div>
                  <p v-if="!masterAnalytics.vote_trends.per_organization.length" class="muted">
                    Nessun dato disponibile.
                  </p>
                </div>
              </div>
            </div>
          </section>

          <div class="dual-grid">
            <section class="card analytics-card">
              <header class="section-header">
                <div>
                  <h2>Eventi più votati</h2>
                  <p>Storico e ultimi 7 giorni.</p>
                </div>
              </header>
              <div class="top-events">
                <div class="top-events__column">
                  <h3>Di sempre</h3>
                  <ul>
                    <li v-for="event in masterAnalytics.top_events.all_time" :key="event.event_id">
                      <div>
                        <strong>{{ event.label }}</strong>
                        <p class="muted">{{ event.organization_name || '—' }}</p>
                        <small class="muted">{{ formatDate(event.start_date) }}</small>
                      </div>
                      <span class="badge">{{ event.total_votes.toLocaleString('it-IT') }} voti</span>
                    </li>
                    <li v-if="!masterAnalytics.top_events.all_time.length" class="muted">Nessun dato.</li>
                  </ul>
                </div>
                <div class="top-events__column">
                  <h3>Ultimi 7 giorni</h3>
                  <ul>
                    <li v-for="event in masterAnalytics.top_events.last_7_days" :key="`${event.event_id}-recent`">
                      <div>
                        <strong>{{ event.label }}</strong>
                        <p class="muted">{{ event.organization_name || '—' }}</p>
                        <small class="muted">{{ formatDate(event.start_date) }}</small>
                      </div>
                      <span class="badge">{{ event.total_votes.toLocaleString('it-IT') }} voti</span>
                    </li>
                    <li v-if="!masterAnalytics.top_events.last_7_days.length" class="muted">Nessun dato.</li>
                  </ul>
                </div>
              </div>
            </section>

            <section class="card analytics-card">
              <header class="section-header">
                <div>
                  <h2>Statistiche sponsor</h2>
                  <p>Impression e click totali e per società.</p>
                </div>
              </header>
              <div class="sponsor-stats">
                <div class="sponsor-stats__summary">
                  <p><strong>Totale impression:</strong> {{ masterAnalytics.sponsor_stats.total_impressions.toLocaleString('it-IT') }}</p>
                  <p><strong>Totale click:</strong> {{ masterAnalytics.sponsor_stats.total_clicks.toLocaleString('it-IT') }}</p>
                  <p><strong>CTR medio:</strong> {{ formatPercent(masterAnalytics.sponsor_stats.average_ctr) }}</p>
                </div>
                <div class="table-wrapper compact">
                  <table>
                    <thead>
                      <tr>
                        <th>Società</th>
                        <th>Impression</th>
                        <th>Click</th>
                        <th>CTR</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="stat in masterAnalytics.sponsor_stats.organizations" :key="stat.organization_id">
                        <td>{{ stat.name }}</td>
                        <td>{{ stat.impressions.toLocaleString('it-IT') }}</td>
                        <td>{{ stat.clicks.toLocaleString('it-IT') }}</td>
                        <td>{{ formatPercent(stat.ctr) }}</td>
                      </tr>
                      <tr v-if="!masterAnalytics.sponsor_stats.organizations.length">
                        <td colspan="4" class="muted">Nessun dato disponibile.</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </section>

            <section class="card analytics-card">
              <header class="section-header">
                <div>
                  <h2>Tempo medio per società</h2>
                  <p>Tempo totale sulle pagine evento, medio per partita e per tifoso.</p>
                </div>
              </header>
              <div class="table-wrapper compact">
                <table>
                  <thead>
                    <tr>
                      <th>Società</th>
                      <th>Tempo totale</th>
                      <th>Tempo medio / partita</th>
                      <th>Tempo medio / utente</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="stat in masterAnalytics.engagement.organizations"
                      :key="stat.organization_id"
                    >
                      <td>
                        <div class="org-cell">
                          <div>
                            <p class="org-name">{{ stat.name || 'Società' }}</p>
                            <small class="muted">{{ stat.slug || `ID ${stat.organization_id}` }}</small>
                          </div>
                        </div>
                      </td>
                      <td>{{ formatDurationSeconds(stat.total_duration_seconds) }}</td>
                      <td>{{ formatDurationSeconds(stat.average_duration_per_match) }}</td>
                      <td>{{ formatDurationSeconds(stat.average_duration_per_user) }}</td>
                    </tr>
                    <tr v-if="!masterAnalytics.engagement.organizations.length">
                      <td colspan="4" class="muted">Nessun dato disponibile.</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>
          </div>

          <section class="card analytics-card">
            <header class="section-header">
              <div>
                <h2>Riepilogo mensile</h2>
                <p>Voti, partite e utenti unici con confronto rispetto al mese precedente.</p>
              </div>
            </header>
            <div class="monthly-summary">
              <div class="monthly-summary__column">
                <p class="label">Mese corrente</p>
                <h3>{{ masterAnalytics.monthly_summary.current.month || '—' }}</h3>
                <p><strong>Voti:</strong> {{ masterAnalytics.monthly_summary.current.votes.toLocaleString('it-IT') }}</p>
                <p><strong>Partite:</strong> {{ masterAnalytics.monthly_summary.current.events.toLocaleString('it-IT') }}</p>
                <p><strong>Utenti unici:</strong> {{ masterAnalytics.monthly_summary.current.unique_users.toLocaleString('it-IT') }}</p>
              </div>
              <div class="monthly-summary__column">
                <p class="label">Mese precedente</p>
                <h3>{{ masterAnalytics.monthly_summary.previous.month || '—' }}</h3>
                <p><strong>Voti:</strong> {{ masterAnalytics.monthly_summary.previous.votes.toLocaleString('it-IT') }}</p>
                <p><strong>Partite:</strong> {{ masterAnalytics.monthly_summary.previous.events.toLocaleString('it-IT') }}</p>
                <p><strong>Utenti unici:</strong> {{ masterAnalytics.monthly_summary.previous.unique_users.toLocaleString('it-IT') }}</p>
              </div>
              <div class="monthly-summary__column deltas">
                <p class="label">Variazioni</p>
                <p>
                  <strong>Voti:</strong>
                  <span :class="['delta', resolveDeltaClass(masterAnalytics.monthly_summary.votes_change.absolute)]">
                    {{ formatDelta(masterAnalytics.monthly_summary.votes_change) }}
                  </span>
                </p>
                <p>
                  <strong>Partite:</strong>
                  <span :class="['delta', resolveDeltaClass(masterAnalytics.monthly_summary.events_change.absolute)]">
                    {{ formatDelta(masterAnalytics.monthly_summary.events_change) }}
                  </span>
                </p>
                <p>
                  <strong>Utenti unici:</strong>
                  <span :class="['delta', resolveDeltaClass(masterAnalytics.monthly_summary.unique_users_change.absolute)]">
                    {{ formatDelta(masterAnalytics.monthly_summary.unique_users_change) }}
                  </span>
                </p>
              </div>
            </div>
          </section>
        </div>

        <div v-else-if="activeSection === 'organizations'" class="organizations-view">
          <header class="section-header">
            <div>
              <h2>Società</h2>
              <p>Gestisci anagrafiche e stato delle società.</p>
            </div>
            <div class="section-actions">
              <button class="btn outline" type="button" @click="fetchOrganizations" :disabled="isLoadingOrganizations">
                {{ isLoadingOrganizations ? 'Aggiornamento…' : 'Aggiorna elenco' }}
              </button>
              <button class="btn primary" type="button" @click="openCreateOrganization">
                Nuova società
              </button>
            </div>
          </header>

          <div v-if="organizationFormVisible" class="card form-card">
            <header>
              <h3>{{ organizationFormMode === 'create' ? 'Crea società' : 'Modifica società' }}</h3>
              <button class="btn ghost" type="button" @click="closeOrganizationForm">Chiudi</button>
            </header>
            <form class="form-grid" @submit.prevent="submitOrganizationForm">
              <label>
                Nome
                <input v-model.trim="organizationForm.name" type="text" required />
              </label>
              <label>
                Slug / URL pubblico
                <input
                  v-model.trim="organizationForm.slug"
                  type="text"
                  required
                  placeholder="es. volley-milano o https://mia-societa.it"
                />
                <small class="help-text">Puoi inserire uno slug (verrà normalizzato) oppure un URL completo.</small>
              </label>
              <label>
                Città / Descrizione
                <input v-model.trim="organizationForm.city" type="text" placeholder="Es. Milano" />
              </label>
              <label>
                Logo (URL)
                <input v-model.trim="organizationForm.logo_url" type="url" placeholder="https://…" />
              </label>
              <label class="switch-field">
                <input type="checkbox" v-model="organizationForm.is_active" />
                <span>Società attiva</span>
              </label>
              <div class="form-actions">
                <button class="btn outline" type="button" @click="closeOrganizationForm">Annulla</button>
                <button class="btn primary" type="submit" :disabled="isSavingOrganization">
                  {{ isSavingOrganization ? 'Salvataggio…' : 'Salva' }}
                </button>
              </div>
              <p v-if="organizationFormError" class="error">{{ organizationFormError }}</p>
            </form>
          </div>

          <div class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Società</th>
                  <th>Slug / URL</th>
                  <th>Città</th>
                  <th>Stato</th>
                  <th>Creata il</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="org in organizations" :key="org.id">
                  <td>
                    <div class="org-cell">
                      <img v-if="org.logo_url" :src="org.logo_url" :alt="`Logo ${org.name}`" />
                      <div>
                        <p class="org-name">{{ org.name }}</p>
                        <small>ID {{ org.id }}</small>
                      </div>
                    </div>
                  </td>
                  <td>
                    <div class="slug-cell">
                      <div v-if="org.slug" class="slug-links">
                        <a
                          :href="resolvePublicLink(org.slug)"
                          class="slug-link"
                          target="_blank"
                          rel="noreferrer"
                        >
                          {{ org.slug }}
                        </a>
                        <a
                          :href="resolveAdminLink(org.slug)"
                          class="slug-link admin"
                          target="_blank"
                          rel="noreferrer"
                        >
                          Admin
                        </a>
                      </div>
                      <span v-else class="muted">—</span>
                    </div>
                  </td>
                  <td>{{ org.city || '—' }}</td>
                  <td>
                    <span :class="['status-pill', org.is_active ? 'active' : 'inactive']">
                      {{ org.is_active ? 'Attiva' : 'Disattiva' }}
                    </span>
                  </td>
                  <td>{{ formatDate(org.created_at) }}</td>
                  <td class="actions">
                    <button class="btn ghost" type="button" @click="viewOrganization(org.id)">
                      Dettagli
                    </button>
                    <button class="btn outline" type="button" @click="openEditOrganization(org)">
                      Modifica
                    </button>
                  </td>
                </tr>
                <tr v-if="!organizations.length && !isLoadingOrganizations">
                  <td colspan="4" class="empty">Nessuna società registrata.</td>
                </tr>
                <tr v-if="isLoadingOrganizations">
                  <td colspan="4" class="empty">Caricamento in corso…</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-else-if="activeSection === 'qr-redirects'" class="qr-redirects-view">
          <header class="section-header">
            <div>
              <h2>QR Redirect</h2>
              <p>Collega un percorso QR generico a quello desiderato e monitora gli accessi.</p>
            </div>
            <div class="section-actions">
              <button class="btn outline" type="button" @click="fetchQRRedirects" :disabled="isLoadingQrRedirects">
                {{ isLoadingQrRedirects ? 'Aggiornamento…' : 'Aggiorna elenco' }}
              </button>
            </div>
          </header>

          <div class="card form-card">
            <header>
              <h3>Nuovo collegamento</h3>
            </header>
            <form class="form-grid" @submit.prevent="submitQRRedirect">
              <label>
                Path QR (origine)
                <input v-model.trim="qrRedirectForm.source_path" type="text" placeholder="/qrred" required />
              </label>
              <label>
                Path destinazione (redirect)
                <input v-model.trim="qrRedirectForm.target_path" type="text" placeholder="/joy-volley" required />
              </label>
              <div class="form-actions">
                <button class="btn primary" type="submit" :disabled="isSavingQrRedirect">
                  {{ isSavingQrRedirect ? 'Salvataggio…' : 'Collega' }}
                </button>
                <button class="btn outline" type="button" @click="resetQRRedirectForm">Pulisci</button>
              </div>
              <p v-if="qrRedirectError" class="error">{{ qrRedirectError }}</p>
            </form>
          </div>

          <div class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Path QR</th>
                  <th>Redirect</th>
                  <th>Collegato il</th>
                  <th>Arrivi da QR</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="redirect in qrRedirects" :key="redirect.id">
                  <td><code>{{ redirect.source_path }}</code></td>
                  <td><code>{{ redirect.target_path }}</code></td>
                  <td>{{ formatDate(redirect.created_at) }}</td>
                  <td>{{ (redirect.hits ?? 0).toLocaleString('it-IT') }}</td>
                </tr>
                <tr v-if="!qrRedirects.length && !isLoadingQrRedirects">
                  <td colspan="4" class="empty">Nessun collegamento creato.</td>
                </tr>
                <tr v-if="isLoadingQrRedirects">
                  <td colspan="4" class="empty">Caricamento in corso…</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-else-if="activeSection === 'organization-detail'" class="detail-view">
          <div v-if="organizationDetail">
            <header class="section-header">
              <div>
                <h2>{{ organizationDetail.organization.name }}</h2>
                <p>{{ organizationDetail.organization.city || 'Nessuna descrizione disponibile' }}</p>
              </div>
              <div class="section-actions">
                <a
                  v-if="organizationDetail.organization.slug"
                  :href="resolvePublicLink(organizationDetail.organization.slug)"
                  class="btn outline"
                  target="_blank"
                  rel="noreferrer"
                >
                  Pagina pubblica
                </a>
                <a :href="resolveSocietyLink(organizationDetail.organization)" class="btn ghost" target="_blank">
                  Apri pannello società
                </a>
                <button class="btn outline" type="button" @click="switchSection('organizations')">Torna alla lista</button>
              </div>
            </header>

            <div class="detail-grid">
              <article class="card info-card">
                <div class="logo-preview" v-if="organizationDetail.organization.logo_url">
                  <img :src="organizationDetail.organization.logo_url" :alt="organizationDetail.organization.name" />
                </div>
                <dl>
                  <div>
                    <dt>ID</dt>
                    <dd>{{ organizationDetail.organization.id }}</dd>
                  </div>
                  <div>
                    <dt>Slug / URL pubblico</dt>
                    <dd>
                      <div v-if="organizationDetail.organization.slug" class="slug-links">
                        <a
                          :href="resolvePublicLink(organizationDetail.organization.slug)"
                          class="slug-link"
                          target="_blank"
                          rel="noreferrer"
                        >
                          {{ organizationDetail.organization.slug }}
                        </a>
                        <a
                          :href="resolveAdminLink(organizationDetail.organization.slug)"
                          class="slug-link admin"
                          target="_blank"
                          rel="noreferrer"
                        >
                          Admin
                        </a>
                      </div>
                      <span v-else class="muted">—</span>
                    </dd>
                  </div>
                  <div>
                    <dt>Stato</dt>
                    <dd>
                      <span :class="['status-pill', organizationDetail.organization.is_active ? 'active' : 'inactive']">
                        {{ organizationDetail.organization.is_active ? 'Attiva' : 'Disattiva' }}
                      </span>
                    </dd>
                  </div>
                  <div>
                    <dt>Creato il</dt>
                    <dd>{{ formatDate(organizationDetail.organization.created_at) }}</dd>
                  </div>
                </dl>
              </article>

              <article class="card info-card">
                <h3>Riepilogo voti</h3>
                <div class="stat-line">
                  <span>Totale voti</span>
                  <strong>{{ organizationDetail.stats.total_votes }}</strong>
                </div>
                <div class="stat-line">
                  <span>Partite totali</span>
                  <strong>{{ organizationDetail.stats.total_matches }}</strong>
                </div>
                <div class="stat-line">
                  <span>Ultima partita</span>
                  <div>
                    <strong>{{ organizationDetail.stats.last_match_votes }}</strong>
                    <small>{{ formatDate(organizationDetail.stats.last_match_date) }}</small>
                  </div>
                </div>
              </article>
            </div>
          </div>
          <p v-else class="empty">
            {{ isLoadingDetail ? 'Caricamento dettagli…' : 'Seleziona una società per vedere i dettagli.' }}
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { apiClient } from '../api';
import VoteTrendChart from './VoteTrendChart.vue';

const token = ref(localStorage.getItem('adminToken') || '');
const activeUsername = ref(localStorage.getItem('adminUsername') || '');
const activeRole = ref(localStorage.getItem('adminRole') || '');
const isAuthenticated = computed(() => Boolean(token.value));
const isSuperAdmin = computed(() => activeRole.value === 'superadmin');

const tabs = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'organizations', label: 'Società' },
  { id: 'qr-redirects', label: 'QR Redirect' },
];
const activeSection = ref('dashboard');

const loginForm = reactive({ username: '', password: '' });
const isLoggingIn = ref(false);
const loginError = ref('');

const summary = reactive({ total_organizations: 0, total_votes: 0, votes_last_7_days: 0, total_events: 0 });
const isLoadingSummary = ref(false);
const summaryLoaded = ref(false);

const createEmptyAnalytics = () => ({
  organization_leaderboard: [],
  vote_trends: { global: [], per_organization: [] },
  top_events: { all_time: [], last_7_days: [] },
  sponsor_stats: { total_impressions: 0, total_clicks: 0, average_ctr: 0, organizations: [] },
  monthly_summary: {
    current: { month: '', votes: 0, events: 0, unique_users: 0 },
    previous: { month: '', votes: 0, events: 0, unique_users: 0 },
    votes_change: { absolute: 0, percent: 0 },
    events_change: { absolute: 0, percent: 0 },
    unique_users_change: { absolute: 0, percent: 0 },
  },
  engagement: {
    total_duration_seconds: 0,
    average_duration_per_match: 0,
    average_duration_per_user: 0,
    organizations: [],
  },
});

const masterAnalytics = ref(createEmptyAnalytics());
const isLoadingAnalytics = ref(false);
const analyticsLoaded = ref(false);
const analyticsError = ref('');

const organizations = ref([]);
const isLoadingOrganizations = ref(false);
const organizationsLoaded = ref(false);

const selectedOrganizationId = ref(0);
const organizationDetail = ref(null);
const isLoadingDetail = ref(false);

const organizationForm = reactive({ id: 0, name: '', slug: '', city: '', logo_url: '', is_active: true });
const organizationFormVisible = ref(false);
const organizationFormMode = ref('create');
const isSavingOrganization = ref(false);
const organizationFormError = ref('');

const qrRedirectForm = reactive({ source_path: '', target_path: '' });
const qrRedirects = ref([]);
const isLoadingQrRedirects = ref(false);
const qrRedirectsLoaded = ref(false);
const isSavingQrRedirect = ref(false);
const qrRedirectError = ref('');

const authHeaders = computed(() => ({
  headers: { Authorization: token.value ? `Bearer ${token.value}` : '' },
}));

function resetOrganizationForm() {
  organizationForm.id = 0;
  organizationForm.name = '';
  organizationForm.slug = '';
  organizationForm.city = '';
  organizationForm.logo_url = '';
  organizationForm.is_active = true;
  organizationFormError.value = '';
}

function resetQRRedirectForm() {
  qrRedirectForm.source_path = '';
  qrRedirectForm.target_path = '';
  qrRedirectError.value = '';
}

function openCreateOrganization() {
  organizationFormMode.value = 'create';
  resetOrganizationForm();
  organizationFormVisible.value = true;
}

function openEditOrganization(org) {
  organizationFormMode.value = 'edit';
  organizationForm.id = org.id;
  organizationForm.name = org.name;
  organizationForm.slug = org.slug || '';
  organizationForm.city = org.city || '';
  organizationForm.logo_url = org.logo_url || '';
  organizationForm.is_active = Boolean(org.is_active);
  organizationFormVisible.value = true;
}

function closeOrganizationForm() {
  organizationFormVisible.value = false;
  organizationFormError.value = '';
}

function switchSection(section) {
  activeSection.value = section;
}

function resolveAdminLink(slug) {
  const baseLink = resolvePublicLink(slug);
  if (!baseLink) return '';

  try {
    const url = new URL(baseLink, typeof window !== 'undefined' ? window.location.origin : undefined);
    url.pathname = `${url.pathname.replace(/\/+$/, '')}/admin`;
    return url.toString();
  } catch (error) {
    const sanitized = baseLink.replace(/\/+$/, '');
    return `${sanitized}/admin`;
  }
}

function resolveSocietyLink(org) {
  if (org?.slug) {
    return resolveAdminLink(org.slug);
  }
  const id = typeof org === 'object' ? org?.id : org;
  return `/admin?society=${id ?? ''}`;
}

function resolvePublicLink(slug) {
  if (!slug) return '';
  if (/^https?:\/\//i.test(slug)) {
    return slug;
  }
  if (typeof window === 'undefined' || !window.location?.origin) {
    return `/${slug}`;
  }
  return `${window.location.origin}/${slug}`;
}

function formatDate(value) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('it-IT');
}

function formatPercent(value) {
  const parsed = Number(value ?? 0);
  if (!Number.isFinite(parsed)) {
    return '0%';
  }
  return `${parsed.toFixed(1)}%`;
}

function formatDurationSeconds(value) {
  const total = Math.max(0, Math.floor(Number(value ?? 0)));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  if (hours > 0) {
    return `${hours.toLocaleString('it-IT')}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m`;
  }
  return `${total}s`;
}

function resolveDeltaClass(value) {
  if (value > 0) return 'positive';
  if (value < 0) return 'negative';
  return 'neutral';
}

function formatDelta(delta) {
  const absolute = Number(delta?.absolute ?? 0);
  const percent = Number(delta?.percent ?? 0);
  const sign = absolute > 0 ? '+' : '';
  const absoluteLabel = `${sign}${absolute.toLocaleString('it-IT')}`;
  return `${absoluteLabel} (${formatPercent(percent)})`;
}

function buildChartPoints(points) {
  if (!Array.isArray(points)) {
    return [];
  }
  return points
    .map((point) => {
      const date = point?.date ? new Date(point.date) : null;
      if (!(date instanceof Date) || Number.isNaN(date.valueOf())) {
        return null;
      }
      const value = Number(point?.votes ?? 0);
      if (!Number.isFinite(value)) {
        return null;
      }
      return {
        date,
        value,
        label: date.toLocaleDateString('it-IT', { day: '2-digit', month: '2-digit' }),
        tooltip: `${value.toLocaleString('it-IT')} voti · ${date.toLocaleDateString('it-IT')}`,
      };
    })
    .filter(Boolean);
}

async function login() {
  if (!loginForm.username || !loginForm.password) {
    loginError.value = 'Inserisci username e password';
    return;
  }
  loginError.value = '';
  isLoggingIn.value = true;
  try {
    const { data } = await apiClient.post('/admin/login', {
      username: loginForm.username,
      password: loginForm.password,
    });
    token.value = data?.token || '';
    activeUsername.value = data?.username || '';
    activeRole.value = data?.role || '';
    localStorage.setItem('adminToken', token.value);
    localStorage.setItem('adminUsername', activeUsername.value);
    localStorage.setItem('adminRole', activeRole.value);
    loginForm.username = '';
    loginForm.password = '';
    if (!isSuperAdmin.value) {
      loginError.value = 'Account privo di privilegi master.';
    } else {
      ensureSectionData(activeSection.value);
    }
  } catch (error) {
    if (error?.response?.status === 401) {
      loginError.value = 'Credenziali non valide';
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
  organizations.value = [];
  organizationDetail.value = null;
  summaryLoaded.value = false;
  analyticsLoaded.value = false;
  analyticsError.value = '';
  masterAnalytics.value = createEmptyAnalytics();
  organizationsLoaded.value = false;
  selectedOrganizationId.value = 0;
  activeSection.value = 'dashboard';
  qrRedirects.value = [];
  qrRedirectsLoaded.value = false;
  resetQRRedirectForm();
}

async function fetchSummary() {
  if (!isSuperAdmin.value || !token.value) return;
  isLoadingSummary.value = true;
  try {
    const { data } = await apiClient.get('/admin/master/summary', authHeaders.value);
    summary.total_organizations = data?.total_organizations ?? 0;
    summary.total_votes = data?.total_votes ?? 0;
    summary.votes_last_7_days = data?.votes_last_7_days ?? 0;
    summary.total_events = data?.total_events ?? 0;
    summaryLoaded.value = true;
  } catch (error) {
    console.error('Impossibile caricare la dashboard master', error);
  } finally {
    isLoadingSummary.value = false;
  }
}

function normalizeMasterAnalytics(raw) {
  const normalized = createEmptyAnalytics();
  if (!raw || typeof raw !== 'object') {
    return normalized;
  }

  const numberValue = (value) => {
    const parsed = Number(value ?? 0);
    return Number.isFinite(parsed) ? parsed : 0;
  };

  const parseLeaderboard = Array.isArray(raw.organization_leaderboard)
    ? raw.organization_leaderboard.map((entry) => ({
        organization_id: numberValue(entry.organization_id),
        name: entry?.name || '',
        slug: entry?.slug || '',
        city: entry?.city || '',
        logo_url: entry?.logo_url || '',
        total_votes: numberValue(entry?.total_votes),
        votes_last_7_days: numberValue(entry?.votes_last_7_days),
        total_events: numberValue(entry?.total_events),
        growth_percentage: Number(entry?.growth_percentage ?? 0) || 0,
      }))
    : [];

  normalized.organization_leaderboard = parseLeaderboard;

  const normalizePoints = (points) =>
    Array.isArray(points)
      ? points.map((point) => ({ date: point?.date || '', votes: numberValue(point?.votes) })).filter((p) => p.date)
      : [];

  normalized.vote_trends = {
    global: normalizePoints(raw?.vote_trends?.global),
    per_organization: Array.isArray(raw?.vote_trends?.per_organization)
      ? raw.vote_trends.per_organization.map((org) => ({
          organization_id: numberValue(org?.organization_id),
          name: org?.name || '',
          slug: org?.slug || '',
          data: normalizePoints(org?.data),
        }))
      : [],
  };

  const normalizeTopEvents = (list) =>
    Array.isArray(list)
      ? list.map((event) => ({
          event_id: numberValue(event?.event_id),
          organization_id: numberValue(event?.organization_id),
          organization_name: event?.organization_name || '',
          organization_slug: event?.organization_slug || '',
          label: event?.label || '',
          start_date: event?.start_date || '',
          total_votes: numberValue(event?.total_votes),
        }))
      : [];

  normalized.top_events = {
    all_time: normalizeTopEvents(raw?.top_events?.all_time),
    last_7_days: normalizeTopEvents(raw?.top_events?.last_7_days),
  };

  normalized.sponsor_stats = {
    total_impressions: numberValue(raw?.sponsor_stats?.total_impressions),
    total_clicks: numberValue(raw?.sponsor_stats?.total_clicks),
    average_ctr: Number(raw?.sponsor_stats?.average_ctr ?? 0) || 0,
    organizations: Array.isArray(raw?.sponsor_stats?.organizations)
      ? raw.sponsor_stats.organizations.map((org) => ({
          organization_id: numberValue(org?.organization_id),
          name: org?.name || '',
          slug: org?.slug || '',
          impressions: numberValue(org?.impressions),
          clicks: numberValue(org?.clicks),
          ctr: Number(org?.ctr ?? 0) || 0,
        }))
      : [],
  };

  normalized.engagement = {
    total_duration_seconds: numberValue(raw?.engagement?.total_duration_seconds),
    average_duration_per_match: Number(raw?.engagement?.average_duration_per_match ?? 0) || 0,
    average_duration_per_user: Number(raw?.engagement?.average_duration_per_user ?? 0) || 0,
    organizations: Array.isArray(raw?.engagement?.organizations)
      ? raw.engagement.organizations.map((org) => ({
          organization_id: numberValue(org?.organization_id),
          name: org?.name || '',
          slug: org?.slug || '',
          total_duration_seconds: numberValue(org?.total_duration_seconds),
          average_duration_per_match: Number(org?.average_duration_per_match ?? 0) || 0,
          average_duration_per_user: Number(org?.average_duration_per_user ?? 0) || 0,
        }))
      : [],
  };

  const defaultMonthly = normalized.monthly_summary;
  const monthly = raw?.monthly_summary || {};
  normalized.monthly_summary = {
    current: {
      month: monthly?.current?.month || defaultMonthly.current.month,
      votes: numberValue(monthly?.current?.votes ?? defaultMonthly.current.votes),
      events: numberValue(monthly?.current?.events ?? defaultMonthly.current.events),
      unique_users: numberValue(monthly?.current?.unique_users ?? defaultMonthly.current.unique_users),
    },
    previous: {
      month: monthly?.previous?.month || defaultMonthly.previous.month,
      votes: numberValue(monthly?.previous?.votes ?? defaultMonthly.previous.votes),
      events: numberValue(monthly?.previous?.events ?? defaultMonthly.previous.events),
      unique_users: numberValue(monthly?.previous?.unique_users ?? defaultMonthly.previous.unique_users),
    },
    votes_change: {
      absolute: numberValue(monthly?.votes_change?.absolute ?? defaultMonthly.votes_change.absolute),
      percent: Number(monthly?.votes_change?.percent ?? defaultMonthly.votes_change.percent) || 0,
    },
    events_change: {
      absolute: numberValue(monthly?.events_change?.absolute ?? defaultMonthly.events_change.absolute),
      percent: Number(monthly?.events_change?.percent ?? defaultMonthly.events_change.percent) || 0,
    },
    unique_users_change: {
      absolute: numberValue(monthly?.unique_users_change?.absolute ?? defaultMonthly.unique_users_change.absolute),
      percent: Number(monthly?.unique_users_change?.percent ?? defaultMonthly.unique_users_change.percent) || 0,
    },
  };

  return normalized;
}

async function fetchAnalytics() {
  if (!isSuperAdmin.value || !token.value) return;
  isLoadingAnalytics.value = true;
  analyticsError.value = '';
  try {
    const { data } = await apiClient.get('/admin/master/analytics', authHeaders.value);
    masterAnalytics.value = normalizeMasterAnalytics(data);
    analyticsLoaded.value = true;
  } catch (error) {
    if (error?.response?.status === 401) {
      analyticsError.value = 'Sessione scaduta, effettua nuovamente il login.';
    } else {
      analyticsError.value = 'Impossibile caricare le analytics.';
    }
  } finally {
    isLoadingAnalytics.value = false;
  }
}

async function refreshDashboard() {
  await Promise.all([fetchSummary(), fetchAnalytics()]);
}

async function fetchOrganizations() {
  if (!isSuperAdmin.value || !token.value) return;
  isLoadingOrganizations.value = true;
  try {
    const { data } = await apiClient.get('/admin/master/organizations', authHeaders.value);
    organizations.value = Array.isArray(data) ? data : [];
    organizationsLoaded.value = true;
  } catch (error) {
    console.error('Impossibile caricare le società', error);
  } finally {
    isLoadingOrganizations.value = false;
  }
}

async function fetchOrganizationDetail(id = selectedOrganizationId.value) {
  if (!isSuperAdmin.value || !token.value || !id) return;
  isLoadingDetail.value = true;
  try {
    const { data } = await apiClient.get(`/admin/master/organizations/${id}`, authHeaders.value);
    organizationDetail.value = data || null;
  } catch (error) {
    console.error('Impossibile caricare il dettaglio società', error);
    organizationDetail.value = null;
  } finally {
    isLoadingDetail.value = false;
  }
}

async function submitOrganizationForm() {
  if (!organizationForm.name) {
    organizationFormError.value = 'Il nome è obbligatorio';
    return;
  }
  organizationFormError.value = '';
  isSavingOrganization.value = true;
  try {
    const payload = {
      name: organizationForm.name,
      slug: organizationForm.slug,
      city: organizationForm.city,
      logo_url: organizationForm.logo_url,
      is_active: organizationForm.is_active,
    };
    if (organizationFormMode.value === 'create') {
      await apiClient.post('/admin/master/organizations', payload, authHeaders.value);
    } else {
      await apiClient.put(`/admin/master/organizations/${organizationForm.id}`, payload, authHeaders.value);
    }
    closeOrganizationForm();
    fetchOrganizations();
    if (organizationFormMode.value === 'edit' && selectedOrganizationId.value === organizationForm.id) {
      fetchOrganizationDetail();
    }
  } catch (error) {
    if (error?.response?.status === 400) {
      organizationFormError.value = 'Dati non validi. Verifica i campi obbligatori.';
    } else if (error?.response?.status === 404) {
      organizationFormError.value = 'Società non trovata.';
    } else {
      organizationFormError.value = 'Errore durante il salvataggio. Riprova.';
    }
  } finally {
    isSavingOrganization.value = false;
  }
}

async function fetchQRRedirects() {
  if (!isSuperAdmin.value || !token.value) return;
  isLoadingQrRedirects.value = true;
  try {
    const { data } = await apiClient.get('/admin/master/qr-redirects', authHeaders.value);
    qrRedirects.value = Array.isArray(data) ? data : [];
    qrRedirectsLoaded.value = true;
  } catch (error) {
    console.error('Impossibile caricare i redirect QR', error);
  } finally {
    isLoadingQrRedirects.value = false;
  }
}

async function submitQRRedirect() {
  if (!qrRedirectForm.source_path || !qrRedirectForm.target_path) {
    qrRedirectError.value = 'Compila entrambi i percorsi';
    return;
  }
  qrRedirectError.value = '';
  isSavingQrRedirect.value = true;
  try {
    await apiClient.post(
      '/admin/master/qr-redirects',
      { source_path: qrRedirectForm.source_path, target_path: qrRedirectForm.target_path },
      authHeaders.value
    );
    resetQRRedirectForm();
    fetchQRRedirects();
  } catch (error) {
    if (error?.response?.status === 400) {
      qrRedirectError.value = 'Percorsi non validi. Usa path come /qrred e /joy-volley.';
    } else {
      qrRedirectError.value = 'Errore durante il salvataggio. Riprova.';
    }
  } finally {
    isSavingQrRedirect.value = false;
  }
}

function viewOrganization(id) {
  selectedOrganizationId.value = id;
  organizationDetail.value = null;
  activeSection.value = 'organization-detail';
}

function ensureSectionData(section) {
  if (!isSuperAdmin.value) return;
  if (section === 'dashboard' && !summaryLoaded.value) {
    fetchSummary();
  }
  if (section === 'dashboard' && !analyticsLoaded.value && !isLoadingAnalytics.value) {
    fetchAnalytics();
  }
  if (section === 'organizations' && !organizationsLoaded.value) {
    fetchOrganizations();
  }
  if (section === 'qr-redirects' && !qrRedirectsLoaded.value) {
    fetchQRRedirects();
  }
  if (section === 'organization-detail' && selectedOrganizationId.value && !organizationDetail.value && !isLoadingDetail.value) {
    fetchOrganizationDetail();
  }
}

watch(activeSection, ensureSectionData);
watch(isSuperAdmin, (value) => {
  if (value) {
    ensureSectionData(activeSection.value);
  }
});

watch(selectedOrganizationId, (value) => {
  if (value && activeSection.value === 'organization-detail') {
    fetchOrganizationDetail(value);
  }
});

onMounted(() => {
  if (isSuperAdmin.value) {
    ensureSectionData(activeSection.value);
  }
});
</script>

<style scoped>
.master-portal {
  padding: clamp(1.5rem, 4vw, 3rem);
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  color: #0f172a;
}

.master-header {
  background: linear-gradient(135deg, #0f172a, #1e293b);
  color: #fff;
  padding: clamp(1.5rem, 3vw, 2.5rem);
  border-radius: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.master-header .subtitle {
  color: rgba(255, 255, 255, 0.8);
  margin: 0.25rem 0 0;
}

.master-header .eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.2em;
  font-size: 0.75rem;
  margin: 0;
  color: rgba(255, 255, 255, 0.7);
}

.master-header h1 {
  margin: 0.2rem 0;
  font-size: clamp(1.5rem, 4vw, 2.5rem);
}

.header-actions {
  display: flex;
  gap: 0.75rem;
}

.master-shell {
  background: #fff;
  border-radius: 1.5rem;
  padding: 1.5rem;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.12);
}

.master-nav {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1.25rem;
}

.nav-btn {
  border: 1px solid rgba(15, 23, 42, 0.15);
  background: #f8fafc;
  color: #0f172a;
  padding: 0.6rem 1.4rem;
  border-radius: 999px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s ease;
}

.nav-btn.active,
.nav-btn:hover {
  background: #0f172a;
  color: #fff;
}

.master-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.grid-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: #0f172a;
  color: #fff;
  padding: 1.5rem;
  border-radius: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.1);
}

.stat-card .label {
  text-transform: uppercase;
  font-size: 0.8rem;
  letter-spacing: 0.1em;
  color: rgba(255, 255, 255, 0.8);
}

.stat-card .value {
  font-size: clamp(2rem, 5vw, 2.8rem);
  margin: 0;
}

.stat-card.highlight {
  background: linear-gradient(145deg, #0f172a, #0ea5e9);
  box-shadow: 0 10px 40px rgba(14, 165, 233, 0.35);
}

.analytics-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.delta {
  font-weight: 700;
  padding: 0.15rem 0.55rem;
  border-radius: 999px;
}

.delta.positive {
  background: rgba(22, 163, 74, 0.12);
  color: #15803d;
}

.delta.negative {
  background: rgba(220, 38, 38, 0.12);
  color: #b91c1c;
}

.delta.neutral {
  background: rgba(148, 163, 184, 0.2);
  color: #475569;
}

.trend-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1rem;
}

.trend-panel {
  background: #0f172a;
  color: #fff;
  padding: 1rem;
  border-radius: 1rem;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.1);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.trend-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-height: 360px;
  overflow: auto;
}

.trend-list__item {
  background: rgba(255, 255, 255, 0.08);
  padding: 0.75rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.trend-list__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.35rem;
}

.dual-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1rem;
}

.top-events {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1rem;
}

.top-events__column ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.top-events__column li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: #f8fafc;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 0.9rem;
}

.badge {
  background: #0ea5e9;
  color: #fff;
  padding: 0.3rem 0.75rem;
  border-radius: 999px;
  font-weight: 700;
  font-size: 0.9rem;
}

.sponsor-stats {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.sponsor-stats__summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.5rem;
}

.table-wrapper.compact table td,
.table-wrapper.compact table th {
  padding: 0.75rem;
}

.monthly-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
  align-items: stretch;
}

.monthly-summary__column {
  background: #f8fafc;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 1rem;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.monthly-summary__column.deltas {
  background: #0f172a;
  color: #fff;
  border: none;
}

.monthly-summary__column .label {
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.78rem;
  color: #475569;
}

.monthly-summary__column.deltas .label {
  color: rgba(255, 255, 255, 0.7);
}

.organizations-view .section-header,
.detail-view .section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.section-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.table-wrapper {
  overflow-x: auto;
  background: #fff;
  border-radius: 1rem;
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.table-wrapper table {
  width: 100%;
  border-collapse: collapse;
}

.table-wrapper th,
.table-wrapper td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
}

.table-wrapper th {
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #475569;
}

.org-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.org-cell img {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid rgba(15, 23, 42, 0.1);
}

.org-name {
  margin: 0;
  font-weight: 600;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 600;
}

.status-pill.active {
  background: rgba(16, 185, 129, 0.15);
  color: #047857;
}

.status-pill.inactive {
  background: rgba(248, 113, 113, 0.15);
  color: #b91c1c;
}

.slug-cell {
  display: flex;
  align-items: center;
  min-height: 2rem;
}

.slug-links {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.slug-link {
  color: #2563eb;
  font-weight: 600;
  text-decoration: none;
}

.slug-link.admin {
  color: #0f172a;
  background: rgba(59, 130, 246, 0.12);
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  font-weight: 700;
  border: 1px solid rgba(59, 130, 246, 0.35);
}

.slug-link:hover,
.slug-link:focus {
  text-decoration: underline;
}

.muted {
  color: #94a3b8;
}

.actions {
  display: flex;
  gap: 0.5rem;
}

.empty {
  text-align: center;
  color: #64748b;
}

.form-card header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  font-weight: 600;
}

.form-grid input {
  padding: 0.65rem 0.8rem;
  border-radius: 0.65rem;
  border: 1px solid rgba(15, 23, 42, 0.2);
}

.help-text {
  font-size: 0.75rem;
  color: #64748b;
}

.switch-field {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}

.form-actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.login-card,
.warning-card,
.card {
  background: #fff;
  border-radius: 1.25rem;
  padding: 1.5rem;
  box-shadow: 0 15px 40px rgba(15, 23, 42, 0.1);
}

.warning-card {
  border: 1px solid rgba(248, 113, 113, 0.4);
}

.warning-actions {
  display: flex;
  gap: 1rem;
  margin-top: 1rem;
}

.btn {
  border: none;
  border-radius: 999px;
  padding: 0.5rem 1.25rem;
  font-weight: 600;
  cursor: pointer;
}

.btn.primary {
  background: linear-gradient(135deg, #0ea5e9, #2563eb);
  color: #fff;
}

.btn.outline {
  border: 1px solid rgba(15, 23, 42, 0.2);
  background: transparent;
  color: #0f172a;
}

.btn.ghost {
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid transparent;
  color: inherit;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1.25rem;
}

.info-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.logo-preview {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  border: 2px solid rgba(15, 23, 42, 0.1);
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

dl {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem 1rem;
}

dl dt {
  font-size: 0.8rem;
  text-transform: uppercase;
  color: #94a3b8;
}

dl dd {
  margin: 0;
  font-weight: 600;
}

.stat-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
}

.stat-line strong {
  font-size: 1.5rem;
}

.error {
  color: #b91c1c;
  margin-top: 0.5rem;
}

@media (max-width: 640px) {
  .master-header,
  .section-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-actions,
  .section-actions {
    width: 100%;
    flex-direction: column;
  }
}
</style>

<script setup>
// Sezione "Tornei" del MasterPortal — mondo parallelo alle società.
// Il master fa solo provisioning: crea il contenitore, consegna link e
// credenziali all'organizzatore. Squadre/calendario/sponsor sono affare
// del pannello admin torneo (/ta/:slug), non del master.
import { onMounted, reactive, ref } from 'vue'

const tournaments = ref([])
const loading = ref(false)
const error = ref('')
const created = ref(null) // output post-creazione (credenziali mostrate UNA volta)

const form = reactive({ name: '', format: 'BEACH VOLLEY 4X4', dateLabel: '', location: '' })
const submitting = ref(false)

// Reset password: riga aperta + input custom + esito (password mostrata UNA volta).
const pwOpen = ref(null)      // eventId della riga con pannello aperto
const pwInput = ref('')       // password custom (vuota = generata dal server)
const pwBusy = ref(false)
const pwResult = ref(null)    // { eventId, adminUsername, adminPassword }
const rowError = ref('')      // errore azioni per-riga
const deleting = ref(null)    // eventId in cancellazione

function togglePw (t) {
  rowError.value = ''
  pwResult.value = null
  pwInput.value = ''
  pwOpen.value = pwOpen.value === t.eventId ? null : t.eventId
}

async function resetPassword (t) {
  pwBusy.value = true
  rowError.value = ''
  try {
    const res = await fetch(`/api/admin/master/tournaments/${t.eventId}/password`, {
      method: 'PUT',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: pwInput.value.trim() })
    })
    const data = await res.json().catch(() => ({}))
    if (res.status === 400 && data.error === 'password_too_short') {
      rowError.value = 'La password deve avere almeno 6 caratteri.'; return
    }
    if (!res.ok) throw new Error(data.error || res.status)
    pwResult.value = { eventId: t.eventId, ...data }
    pwOpen.value = null
    pwInput.value = ''
  } catch (e) {
    rowError.value = 'Reset password fallito.'
  } finally {
    pwBusy.value = false
  }
}

async function remove (t) {
  if (!window.confirm(`Eliminare definitivamente il torneo "${t.name}" e TUTTI i suoi dati (squadre, partite, sponsor, admin)?\nL'operazione non è reversibile.`)) return
  deleting.value = t.eventId
  rowError.value = ''
  try {
    const res = await fetch(`/api/admin/master/tournaments/${t.eventId}`, {
      method: 'DELETE',
      credentials: 'include'
    })
    if (!res.ok) throw new Error(res.status)
    if (pwResult.value?.eventId === t.eventId) pwResult.value = null
    await load()
  } catch (e) {
    rowError.value = 'Eliminazione fallita.'
  } finally {
    deleting.value = null
  }
}

async function load () {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('/api/admin/master/tournaments', { credentials: 'include' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    tournaments.value = (await res.json()).tournaments ?? []
  } catch (e) {
    error.value = 'Impossibile caricare i tornei.'
  } finally {
    loading.value = false
  }
}

async function create () {
  if (!form.name.trim()) { error.value = 'Il nome è obbligatorio.'; return }
  submitting.value = true
  error.value = ''
  try {
    const res = await fetch('/api/admin/master/tournaments', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form)
    })
    const data = await res.json().catch(() => ({}))
    if (res.status === 409) { error.value = 'Slug già esistente: cambia nome.'; return }
    if (!res.ok) throw new Error(data.error || res.status)
    created.value = data
    form.name = ''; form.dateLabel = ''; form.location = ''
    await load()
  } catch (e) {
    error.value = 'Creazione fallita.'
  } finally {
    submitting.value = false
  }
}

function copy (text) { navigator.clipboard?.writeText(text) }
const abs = path => `${window.location.origin}${path}`

onMounted(load)
</script>

<template>
  <div class="mt-section">
    <section class="mt-card">
      <h2>Nuovo torneo</h2>
      <p class="hint">
        Il master crea il contenitore e consegna le credenziali. Squadre,
        calendario e sponsor li gestisce l'organizzatore dal suo pannello.
      </p>
      <div class="mt-form">
        <label>Nome <input v-model="form.name" placeholder="Sunset Beach Cup" /></label>
        <label>Formato <input v-model="form.format" placeholder="BEACH VOLLEY 4X4" /></label>
        <label>Date <input v-model="form.dateLabel" placeholder="8 - 11 GIUGNO 2026" /></label>
        <label>Luogo <input v-model="form.location" placeholder="LIDO DI CLASSE, RA" /></label>
      </div>
      <button class="mt-btn" :disabled="submitting" @click="create">
        {{ submitting ? 'Creazione…' : 'Crea torneo' }}
      </button>
      <p v-if="error" class="mt-error">{{ error }}</p>

      <div v-if="created" class="mt-created">
        <h3>✓ Torneo creato</h3>
        <p class="warn">Le credenziali sono mostrate SOLO ORA: copiale e consegnale all'organizzatore.</p>
        <div class="row"><span>Portale tifosi</span><code>{{ abs(created.publicPath) }}</code><button @click="copy(abs(created.publicPath))">Copia</button></div>
        <div class="row"><span>Pannello admin</span><code>{{ abs(created.adminPath) }}</code><button @click="copy(abs(created.adminPath))">Copia</button></div>
        <div class="row"><span>Username</span><code>{{ created.adminUsername }}</code><button @click="copy(created.adminUsername)">Copia</button></div>
        <div class="row"><span>Password</span><code>{{ created.adminPassword }}</code><button @click="copy(created.adminPassword)">Copia</button></div>
      </div>
    </section>

    <section class="mt-card">
      <h2>Tornei</h2>
      <p v-if="loading" class="hint">Caricamento…</p>
      <p v-else-if="!tournaments.length" class="hint">Nessun torneo ancora creato.</p>
      <table v-else class="mt-table">
        <thead><tr><th>Nome</th><th>Date</th><th>Fase</th><th>Squadre</th><th>Partite</th><th>Admin</th><th>Link</th><th>Azioni</th></tr></thead>
        <tbody>
          <template v-for="t in tournaments" :key="t.eventId">
            <tr>
              <td><b>{{ t.name }}</b><br /><small>{{ t.slug }}</small></td>
              <td>{{ t.dateLabel }}</td>
              <td>{{ t.phaseLabel }}</td>
              <td>{{ t.teamsCount }}</td>
              <td>{{ t.matchesCount }}</td>
              <td class="admin-cell">
                <code v-if="t.adminUsername">{{ t.adminUsername }}</code>
                <span v-else class="muted">—</span>
                <button v-if="t.adminUsername" class="ico" title="Copia username" @click="copy(t.adminUsername)">⧉</button>
              </td>
              <td class="links">
                <a :href="`/t/${t.slug}`" target="_blank" rel="noopener">tifosi</a>
                <a :href="`/ta/${t.slug}`" target="_blank" rel="noopener">admin</a>
              </td>
              <td class="actions">
                <button class="mt-mini" @click="togglePw(t)">🔑 Password</button>
                <button class="mt-mini danger" :disabled="deleting === t.eventId" @click="remove(t)">
                  {{ deleting === t.eventId ? '…' : '🗑 Elimina' }}
                </button>
              </td>
            </tr>

            <!-- Pannello reset password -->
            <tr v-if="pwOpen === t.eventId" :key="`pw-${t.eventId}`" class="pw-row">
              <td colspan="8">
                <div class="pw-panel">
                  <p class="hint">
                    La password attuale non è recuperabile: se l'organizzatore l'ha persa,
                    assegnane una nuova qui. Lascia vuoto per generarne una casuale.
                  </p>
                  <div class="pw-form">
                    <input v-model="pwInput" type="text" placeholder="Nuova password (vuoto = casuale)" />
                    <button class="mt-btn" :disabled="pwBusy" @click="resetPassword(t)">
                      {{ pwBusy ? 'Salvataggio…' : 'Salva password' }}
                    </button>
                    <button class="mt-mini" @click="togglePw(t)">Annulla</button>
                  </div>
                </div>
              </td>
            </tr>

            <!-- Esito reset: mostrato UNA volta -->
            <tr v-if="pwResult && pwResult.eventId === t.eventId" :key="`res-${t.eventId}`" class="pw-row">
              <td colspan="8">
                <div class="mt-created">
                  <h3>✓ Nuova password impostata</h3>
                  <p class="warn">Mostrata SOLO ORA: copiala e consegnala all'organizzatore. Le sessioni attive sono state disconnesse.</p>
                  <div class="row"><span>Username</span><code>{{ pwResult.adminUsername }}</code><button @click="copy(pwResult.adminUsername)">Copia</button></div>
                  <div class="row"><span>Password</span><code>{{ pwResult.adminPassword }}</code><button @click="copy(pwResult.adminPassword)">Copia</button></div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
      <p v-if="rowError" class="mt-error">{{ rowError }}</p>
    </section>
  </div>
</template>

<style scoped>
.mt-section { display: flex; flex-direction: column; gap: 18px; }
.mt-card { background: rgba(15,23,42,.75); border: 1px solid rgba(148,163,184,.18); border-radius: 14px; padding: 18px 20px; color: #e2e8f0; }
.mt-card h2 { margin: 0 0 6px; font-size: 17px; }
.hint { color: #94a3b8; font-size: 13px; margin: 0 0 14px; }
.mt-form { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px; margin-bottom: 14px; }
.mt-form label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: #94a3b8; }
.mt-form input { background: rgba(2,6,23,.6); border: 1px solid rgba(148,163,184,.25); border-radius: 8px; padding: 9px 11px; color: #f1f5f9; font-size: 14px; }
.mt-btn { background: #f2b928; color: #111; border: none; border-radius: 9px; padding: 10px 18px; font-weight: 800; cursor: pointer; }
.mt-btn:disabled { opacity: .6; cursor: default; }
.mt-error { color: #f87171; font-size: 13px; margin-top: 10px; }
.mt-created { margin-top: 16px; border: 1px solid rgba(242,185,40,.4); border-radius: 10px; padding: 14px; }
.mt-created h3 { margin: 0 0 4px; color: #f2b928; font-size: 15px; }
.mt-created .warn { color: #fbbf24; font-size: 12px; margin: 0 0 10px; }
.mt-created .row { display: flex; align-items: center; gap: 10px; padding: 4px 0; font-size: 13px; }
.mt-created .row span { width: 110px; color: #94a3b8; flex: none; }
.mt-created code { background: rgba(2,6,23,.6); padding: 4px 8px; border-radius: 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mt-created button { background: transparent; border: 1px solid rgba(148,163,184,.35); color: #cbd5e1; border-radius: 6px; padding: 3px 10px; cursor: pointer; font-size: 12px; }
.mt-table { width: 100%; border-collapse: collapse; font-size: 13.5px; }
.mt-table th { text-align: left; color: #94a3b8; font-weight: 600; padding: 8px 10px; border-bottom: 1px solid rgba(148,163,184,.18); }
.mt-table td { padding: 10px; border-bottom: 1px solid rgba(148,163,184,.1); vertical-align: top; }
.mt-table small { color: #64748b; }
.links a { color: #f2b928; margin-right: 10px; }
.mt-table .muted { color: #64748b; }
.admin-cell { white-space: nowrap; }
.admin-cell code { background: rgba(2,6,23,.6); padding: 2px 6px; border-radius: 5px; font-size: 12px; }
.admin-cell .ico { background: transparent; border: none; color: #94a3b8; cursor: pointer; font-size: 13px; padding: 0 4px; }
.actions { white-space: nowrap; }
.mt-mini { background: transparent; border: 1px solid rgba(148,163,184,.35); color: #cbd5e1; border-radius: 6px; padding: 4px 9px; cursor: pointer; font-size: 12px; margin-right: 6px; }
.mt-mini:disabled { opacity: .6; cursor: default; }
.mt-mini.danger { border-color: rgba(248,113,113,.5); color: #f87171; }
.mt-mini.danger:hover { background: rgba(248,113,113,.12); }
.pw-row td { background: rgba(2,6,23,.35); }
.pw-panel .hint { margin: 4px 0 10px; }
.pw-form { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.pw-form input { flex: 1 1 260px; background: rgba(2,6,23,.6); border: 1px solid rgba(148,163,184,.25); border-radius: 8px; padding: 8px 11px; color: #f1f5f9; font-size: 14px; }
</style>

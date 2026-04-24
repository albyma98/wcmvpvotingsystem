# wcmvpvotingsystem

## Creazione di un nuovo admin

### 1. Creare il primo amministratore

Le API protette richiedono che esista almeno un amministratore per poter accedere al pannello `/admin`. Se il database è vuoto è necessario inserirne uno manualmente nella tabella `admins` del file SQLite predefinito `wcmvpvs-back/service/database/mvpvs.db`.

```bash
# Genera l'hash SHA-256 della password che vuoi assegnare
PASSWORD_HASH=$(echo -n 'scegli-una-password-sicura' | sha256sum | awk '{print $1}')

# Inserisci l'admin iniziale (sostituisci username/ruolo se necessario)
sqlite3 wcmvpvs-back/service/database/mvpvs.db "INSERT INTO admins (username, password_hash, role) VALUES ('admin', '$PASSWORD_HASH', 'superadmin');"
```

L'API utilizza l'hash SHA-256 esadecimale delle password, come mostrato dalle funzioni `hashAdminPassword` e `CreateAdmin` nel backend.【F:wcmvpvs-back/service/api/admin-crud.go†L240-L266】【F:wcmvpvs-back/service/database/database.go†L362-L387】

### 2. Accedere al portale amministrativo

Una volta creato il primo amministratore, accedi al portale visitando `http://localhost:3000/admin` (oppure la porta configurata nel deployment) e autentica l'utente creato con la combinazione username/password scelta. Il portale è gestito dal componente `AdminPortal.vue` e consente di amministrare squadre, giocatori, eventi, voti e account admin.【F:wcmvpvs-front/src/App.vue†L3-L24】【F:wcmvpvs-front/src/components/AdminPortal.vue†L156-L210】

### 3. Creare ulteriori amministratori

Nel portale seleziona la sezione **Admin**, compila il form con username, password e ruolo del nuovo utente e premi **Crea admin**. La chiamata inviata al backend utilizza l'endpoint protetto `POST /admins`, che salva l'account e rende immediatamente disponibile l'accesso con le nuove credenziali.【F:wcmvpvs-front/src/components/AdminPortal.vue†L156-L205】【F:wcmvpvs-back/service/api/admin-crud.go†L226-L289】

## Deploy su un server Ubuntu con Docker Compose

La repository include tutto il necessario per eseguire l'applicazione (backend Go + frontend Vue) su un server Ubuntu 22.04/24.04 con Docker e Docker Compose.

1. **Installare i prerequisiti**

   ```bash
   sudo apt-get update
   sudo apt-get install -y docker.io docker-compose-plugin
   sudo systemctl enable --now docker
   ```

2. **Preparare la configurazione**

   ```bash
   git clone https://github.com/albyma98/wcmvpvotingsystem.git
   cd wcmvpvotingsystem
   cp .env.example .env
   ```

   Modifica il file `.env` impostando:

   - `VOTE_SECRET`: stringa usata per firmare i ticket di voto;
   - `BOOTSTRAP_ADMIN_*`: dati per creare automaticamente il primo amministratore.

   Per generare l'hash SHA-256 della password iniziale:

   ```bash
   PASSWORD_HASH=$(echo -n 'scegli-una-password-sicura' | sha256sum | awk '{print $1}')
   sed -i "s/^BOOTSTRAP_ADMIN_PASSWORD_HASH=.*/BOOTSTRAP_ADMIN_PASSWORD_HASH=${PASSWORD_HASH}/" .env
   ```

3. **Avviare i container**

   ```bash
   docker compose up --build -d
   ```

   - Il backend espone le API su `http://<server>:3000` e salva il database SQLite nel volume `backend-data` (montato in `/data`).
   - Il frontend è raggiungibile su `http://<server>:8080` e, grazie alla configurazione di nginx (`wcmvpvs-front/nginx/default.conf`), inoltra automaticamente le richieste `GET/POST` a `/api/*` verso il backend interno.【F:wcmvpvs-front/nginx/default.conf†L1-L20】

4. **Aggiornare o riavviare il servizio**

   ```bash
   docker compose pull
   docker compose up --build -d
   ```

   I dati (squadre, eventi, voti) rimangono persistenti grazie al volume Docker nominato `backend-data` collegato al file SQLite del backend.【F:docker-compose.yml†L6-L37】

5. **Verificare lo stato**

   ```bash
   docker compose ps
   docker compose logs -f backend
   ```

   Assicurati che l'admin di bootstrap sia stato creato controllando i log del backend (messaggio `admin ... creato`). È possibile disabilitare la creazione automatica impostando `BOOTSTRAP_ADMIN_ENABLED=false` o lasciando vuoto `BOOTSTRAP_ADMIN_PASSWORD_HASH`.【F:wcmvpvs-back/cmd/webapi/main.go†L42-L70】

## OTP SMS (Twilio Verify) - API examples

Il backend espone gli endpoint OTP sotto `/api/auth/*` (proxy frontend) / `/auth/*` (backend diretto).

```bash
# Start register (crea utente pending + invia OTP)
curl -i -X POST http://localhost:3000/auth/start \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+393331234567","mode":"register"}'

# Start login (utente già esistente + invia OTP)
curl -i -X POST http://localhost:3000/auth/start \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+393331234567","mode":"login"}'

# Verify code (approva OTP e ritorna token sessione)
curl -i -X POST http://localhost:3000/auth/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+393331234567","code":"123456"}'

# Resend OTP
curl -i -X POST http://localhost:3000/auth/resend \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+393331234567"}'
```

---

## Feature: Branded Mini-Game Modal

Un formato di advertising interattivo che permette agli sponsor di presentare un mini-game brandizzato durante la partita. I fan giocano, vincono coin, e vengono esposti al brand sponsor.

### Come si configura lato admin

1. Apri il portale admin → sezione **Eventi**
2. In creazione o modifica evento, individua il blocco **Branded Mini-Game** (punto arancione)
3. Attiva il toggle "Attiva mini-game sponsor"
4. Compila il form che appare:

| Campo | Descrizione | Obbligatorio |
|---|---|---|
| ID Sponsor | Identificativo interno (es. `acme-2025`) | ✅ |
| Nome Sponsor | Nome visualizzato ai fan (es. `ACME Sport`) | ✅ |
| URL Logo | Link pubblico all'immagine logo | No |
| Colore primario | Sfondo header e CTA (hex) | No |
| Colore secondario | Testo su sfondo primario (hex) | No |
| Tipo gioco | `Tap Battle`, `Memory Flash`, `Sponsor Rush` | ✅ |
| CTA Label | Testo pulsante link sponsor (es. "Scopri di più") | No |
| CTA URL | URL https dove mandare il fan | No |
| Tipo reward | `coins` o `nessun premio` | ✅ |
| Coin reward | Monete assegnate al completamento | Se reward=coins |
| Partite max | Quante volte può giocare per evento (default 1) | ✅ |

5. Clicca **Salva impostazioni** — la config viene serializzata come JSON nella colonna `branded_game_config` dell'evento.

La preview a destra del form si aggiorna in tempo reale mostrando il pulsante di entry come lo vedrà il fan.

### Struttura JSON `branded_game_config`

```json
{
  "sponsor_id": "acme-2025",
  "sponsor_name": "ACME Sport",
  "sponsor_logo_url": "https://cdn.example.com/acme-logo.png",
  "primary_color": "#1a73e8",
  "secondary_color": "#ffffff",
  "game_type": "tap_challenge",
  "cta_label": "Scopri ACME Sport",
  "cta_url": "https://acmesport.it",
  "reward_type": "coins",
  "reward_coins": 50,
  "max_plays_per_user": 1
}
```

### API endpoints

```
GET  /events/{eventId}/branded-game
     → config pubblica + { can_play, plays_used, plays_remaining }
     → 404 se show_branded_game=false

POST /events/{eventId}/branded-game/result
     Body: { score, duration_ms, completed, payload, session_id }
     → { rewarded_coins, remaining_plays }
     → 409 se plays esauriti
     → 429 se rate limit superato (20 req/min per device)
```

### Come sviluppare un nuovo `game_type`

1. **Crea il componente** in `wcmvpvs-front/src/components/minigames/NuovoGioco.vue` rispettando il contratto:
   ```vue
   <script setup>
   const props = defineProps({
     eventId: { type: Number },
     walletCoins: { type: Number, default: 0 },
   });
   const emit = defineEmits(['claim', 'exit']);
   // emit('claim', { coins: N, keepOpen?: false })
   // emit('exit')
   </script>
   ```

2. **Registra il tipo** in `BrandedGameModal.vue`:
   ```js
   const gameComponentMap = {
     tap_challenge: defineAsyncComponent(() => import('./minigames/TapChallenge.vue')),
     memory_flash:  defineAsyncComponent(() => import('./minigames/MemoryFlashGame.vue')),
     sponsor_rush:  defineAsyncComponent(() => import('./minigames/SponsorRushGame.vue')),
     nuovo_gioco:   defineAsyncComponent(() => import('./minigames/NuovoGioco.vue')), // ← aggiungi
   };
   ```

3. **Aggiungi il tipo** alla whitelist backend in `branded_game.go`:
   ```go
   var validGameTypes = map[string]struct{}{
     "tap_challenge": {},
     "memory_flash":  {},
     "sponsor_rush":  {},
     "nuovo_gioco":   {}, // ← aggiungi
   }
   ```

4. **Aggiorna il select** nell'admin form in `AdminPortal.vue`:
   ```html
   <option value="nuovo_gioco">Nuovo Gioco 🎯</option>
   ```

5. Aggiungi la label nel `gameTypeLabel` computed di `BrandedGameEntry.vue` e `BrandedGameModal.vue`.

Il mini-game non deve sapere nulla di branded game: riceve `eventId` e `walletCoins`, emette `claim` o `exit`. Il wrapper `BrandedGameModal` gestisce branding, tracking, e submit result.

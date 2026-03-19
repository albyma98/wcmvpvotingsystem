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


## Configurazione modulo AI (upsell BAR + popup in-app)

Il modulo AI viene eseguito **solo lato backend Go**: il frontend non parla mai direttamente con il provider LLM. Il provider viene usato esclusivamente per generare copy/suggerimenti, mentre disponibilità prodotti, prezzi, trigger reali e regole di business restano sotto controllo applicativo.

### Quale file devo usare davvero?

Se stai avviando il progetto come previsto in questo repository, cioè con **Docker Compose**, devi configurare il modulo AI nel file **`.env`** partendo da `.env.example`.

Il file `wcmvpvs-back/demo/config.yml` **non è il file principale del deploy**: è solo un esempio di configurazione YAML per esecuzioni manuali/locali del backend fuori da Docker. Quindi, nella pratica:

- **usa `.env`** se lanci il progetto con `docker compose up`;
- **usa `wcmvpvs-back/demo/config.yml` solo se avvii il backend Go manualmente** e vuoi passare la configurazione via YAML invece che via env.

### Variabili da impostare

Nel file `.env` puoi configurare il modulo con queste variabili:

```bash
# Attiva il modulo AI
CFG_AI_ENABLED=true

# Provider LLM (OpenAI-compatible)
AI_PROVIDER_BASE_URL=https://api.openai.com/v1
AI_API_KEY=sk-...
AI_MODEL=gpt-4o-mini

# Guardrail runtime backend
CFG_AI_REQUESTTIMEOUT=4s
CFG_AI_CACHETTL=90s
CFG_AI_MAXPOPUPSSESSION=3
CFG_AI_POPUPCOOLDOWN=8m
```

### Significato delle variabili

- `CFG_AI_ENABLED`: abilita/disabilita totalmente il modulo AI; se `false`, il sistema usa solo fallback statici.
- `AI_PROVIDER_BASE_URL`: URL base del provider LLM compatibile con endpoint `/chat/completions`.
- `AI_API_KEY`: chiave API del provider.
- `AI_MODEL`: nome modello da usare per copy upsell e popup.
- `CFG_AI_REQUESTTIMEOUT`: timeout massimo per singola chiamata LLM.
- `CFG_AI_CACHETTL`: cache breve lato backend per evitare richieste duplicate inutili.
- `CFG_AI_MAXPOPUPSSESSION`: massimo popup AI mostrabili nella stessa sessione.
- `CFG_AI_POPUPCOOLDOWN`: attesa minima tra popup consecutivi nella stessa sessione.

### Note operative

- Se usi `docker-compose.yml`, le variabili AI vengono già inoltrate al backend tramite `environment`, quindi ti basta valorizzarle nel file `.env`.
- Se `CFG_AI_ENABLED=true` ma `AI_API_KEY` è vuota, il backend non va in errore: usa il fallback statico.
- Gli endpoint AI esposti dal backend sono:
  - `POST /api/ai/bar/upsell`
  - `POST /api/ai/popups/generate`
  - `POST /api/ai/interactions/{id}/track`
- Le interazioni AI vengono salvate nella tabella SQLite `ai_interactions` per tracking di `shown`, `dismissed`, `clicked`, `converted`.
- Per produzione conviene partire con timeout stretti e cache breve, poi tarare valori e prompt con i dati reali.

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

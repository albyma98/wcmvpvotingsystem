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

## Verificare che lo scraping del live score funzioni

Il backend espone il live score con l'endpoint pubblico `GET /public/events/{eventId}/live-score`, che usa il campo `live_score_url` dell'evento e fa fetch della pagina LegaVolley per estrarre set/punteggio correnti.【F:wcmvpvs-back/service/api/api-handler.go†L19-L24】【F:wcmvpvs-back/service/api/live-score.go†L56-L88】

### 1) Configura un evento con URL LegaVolley valido

Nel portale Admin, in sezione Eventi:

- incolla un URL nel formato `https://ww2.legavolley.it/tabellini_online/gara_<id>.html`;
- salva l'evento;
- usa il pulsante **Test link** per vedere un'anteprima rapida (`Set X: Y-Z`).

La validazione front-end accetta solo quel pattern e il bottone **Test link** chiama proprio l'endpoint pubblico del backend.【F:wcmvpvs-front/src/components/AdminPortal.vue†L392-L423】【F:wcmvpvs-front/src/components/AdminPortal.vue†L5829-L5840】

### 2) Verifica da API con curl

```bash
curl -i http://localhost:3000/public/events/<EVENT_ID>/live-score
```

Se lo scraping è corretto, ricevi `200 OK` con payload JSON (es. `homeName`, `guestName`, `setsHome`, `setsGuest`, `currentSet`, `homePoints`, `guestPoints`, `updatedAt`). In caso di URL mancante o pagina non raggiungibile ottieni rispettivamente `409` o `502`.【F:wcmvpvs-back/service/api/live-score.go†L56-L88】

### 3) Controlla i log backend quando fallisce

Quando il fetch/parsing non riesce il backend scrive un warning `cannot fetch live score` con `event_id`, utile per capire subito quale evento ha un link errato o non più disponibile.【F:wcmvpvs-back/service/api/live-score.go†L77-L80】

```bash
docker compose logs -f backend
```

### 4) Nota sulla cache

Il risultato è cache-ato per 4 secondi per evento: chiamate ripetute immediate possono restituire lo stesso snapshot. Per testare un refresh reale attendi almeno 5 secondi tra due richieste consecutive.【F:wcmvpvs-back/service/api/live-score.go†L19-L20】【F:wcmvpvs-back/service/api/live-score.go†L92-L110】

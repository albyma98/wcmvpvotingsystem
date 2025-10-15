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

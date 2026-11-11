# Match Chat

## Attivazione

Apri **/{slug}/admin/new → Eventi → Configura → Match Chat**. Attiva **Abilita Match Chat per questo evento**. Lo switch salva subito la configurazione di quella partita. Quando la chat è attiva, la homepage **/{slug}/new** mostra **Chat Live** sopra la classifica tifosi, con un’anteprima dell’ultimo messaggio visibile. Premi sul box per aprire la conversazione.

Per un evento concluso, apri **Archivio e report**, espandi l’evento e scegli **Match Chat**.

Gli ospiti possono leggere una chat attiva. Per scrivere, reagire o segnalare serve il profilo della stessa società con verifica OTP completata. La chat usa il nickname. L’admin può rimuovere messaggi dalla vista dei tifosi, esaminare le segnalazioni, impostare o annullare una sospensione e pubblicare come società.

## SQLite e conservazione

`database.New` applica una migrazione additiva e idempotente in una transazione. Gli eventi esistenti e nuovi partono disattivati in assenza di una riga in `event_chat_settings`. Il primo salvataggio registra stato, data e amministratore. Le tabelle `event_chat_messages`, `event_chat_reactions`, `event_chat_reports` e `event_chat_mutes` sono indicizzate per evento. Le reazioni e le segnalazioni referenziano il messaggio con la coppia evento–messaggio. L’identità dei tifosi resta in `fan_profiles`.

La disattivazione conserva i dati. La rimozione di un messaggio lo nasconde ai tifosi e lo conserva per la moderazione. La cancellazione esplicita di un evento o della sua società segue il flusso di eliminazione esistente e rimuove anche i dati della chat. La chat non applica scadenze automatiche alla cronologia.

Per tornare a una versione precedente, arresta il servizio e conserva una copia del database prima di applicare [20261111_match_chat.down.sql](../wcmvpvs-back/migrations/20261111_match_chat.down.sql). Il rollback rimuove le tabelle della chat e l’indice delle aperture, conservando le misurazioni già raccolte nel sistema di report ABX. Una versione che include la chat ricrea lo schema al riavvio.

## API

Il prefisso pubblico è `/api/events/{eventId}/match-chat`. Le richieste HTTP usano `X-Organization-Slug`. Le scritture dei tifosi usano anche `X-Fan-Session`.

| Metodo | Percorso relativo | Operazione |
| --- | --- | --- |
| GET | `/messages?before={id}` | Cronologia visibile e reazioni. Il cursore iniziale è facoltativo. |
| POST | `/messages` | Pubblica `{text, request_id}`. Riusa lo stesso ID quando ripeti un invio incerto. |
| PUT | `/messages/{id}/reaction` | Imposta o rimuove `{emoji, active}`. |
| POST | `/messages/{id}/report` | Segnala con `{reason}`, tra `abuse`, `spam`, `other`. |
| GET, upgrade WebSocket | `/status/stream` | Comunica la disponibilità anche mentre la chat è disattivata. |
| GET, upgrade WebSocket | `/stream` | Riceve i contenuti in diretta di una chat attiva. |

Il prefisso admin è `/api/admin/events/{eventId}/match-chat`. Le rotte usano l’autenticazione admin esistente e verificano la proprietà dell’evento.

| Metodo | Percorso relativo | Operazione |
| --- | --- | --- |
| GET | Percorso base | Restituisce stato e statistiche. |
| PUT | `/settings` | Salva `{enabled}`. |
| GET | `/messages?before={id}` | Include i messaggi rimossi. |
| POST | `/messages` | Pubblica un messaggio ufficiale con `{text, request_id}` quando la chat è attiva. |
| DELETE | `/messages/{id}` | Nasconde un messaggio. |
| GET | `/reports?before={id}` | Elenca le segnalazioni. |
| PUT | `/reports/{id}` | Segna una segnalazione come esaminata. |
| GET | `/mutes?after={fanId}` | Elenca le sospensioni attive. |
| PUT | `/mutes/{fanId}` | Sospende un autore per `{minutes}`, da 1 a 1440. |
| DELETE | `/mutes/{fanId}` | Annulla una sospensione. |

Le pagine contengono fino a 30 elementi e restituiscono `next_before` oppure `next_after`. I timestamp sono millisecondi Unix. Una chat disattivata risponde `409 chat_disabled` alla cronologia pubblica, alle scritture dei tifosi con sessione valida e all’apertura del canale messaggi.

## WebSocket e deployment

I due canali richiedono origine HTTP(S) corrispondente al sito e `organization_slug` nella query. Il server verifica l’associazione tra società ed evento. I frame sono in sola lettura. Il canale disponibilità invia `{type:"status", enabled}`. Con `preview=1` include l’ultimo messaggio visibile nel frame iniziale e invia `{type:"preview", message}` quando l’anteprima viene aggiornata. Il campo `message` è omesso quando non ci sono messaggi visibili. Il canale messaggi apre con lo stato, poi invia `message`, `reaction` con conteggio assoluto e `removed` con l’ID del messaggio. Alla disattivazione invia lo stato e chiude le sottoscrizioni alla conversazione. Il client risincronizza la cronologia recente dopo la riconnessione e usa il cursore per recuperare le pagine precedenti.

Il deployment Compose esistente inoltra gli upgrade via Caddy. Ciascun canale ammette fino a 2.000 connessioni per evento e 5.000 per processo. Il broadcast vive in memoria nel processo Go e presuppone una sola istanza backend. Prima di scalare a più istanze serve un trasporto condiviso per gli aggiornamenti, oltre al coordinamento delle scritture e dei limiti tra le istanze.

## Moderazione e misurazione

Il server limita i messaggi a 180 caratteri Unicode, applica un intervallo e un tetto per minuto per autore ed evento e rifiuta ripetizioni ravvicinate. Le impostazioni di processo sono `MATCH_CHAT_MIN_INTERVAL_SECONDS` (5) e `MATCH_CHAT_MESSAGES_PER_MINUTE` (8). Un limite aggiuntivo di 24 scritture al minuto per profilo protegge invii, reazioni e segnalazioni. HTML e URL sono visualizzati come testo, senza collegamenti o allegati. Il controllo di moderazione precede la pubblicazione. Il tipo `ai` è riservato a future integrazioni, senza operazioni di pubblicazione AI.

L’analytics ABX registra aperture, chiusure, invii, reazioni aggiunte, segnalazioni e cambi di stato per evento. L’apertura unica è attribuita al profilo riconosciuto o al browser ospite pseudonimizzato. Se lo stesso browser apre anche con un profilo riconosciuto, l’apertura ospite viene ricondotta a un profilo di quel browser. Il tasso di partecipazione divide gli utenti tracciati in apertura che hanno scritto per tutte le aperture uniche. Messaggi, autori, reazioni e segnalazioni sono conteggiati dai dati SQLite. I testi e i telefoni non sono raccolti dall’analytics della chat.

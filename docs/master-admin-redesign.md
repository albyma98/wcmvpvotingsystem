# ArenaBoostX Master Admin

Il nuovo workspace si apre da `/admin/master/new`. Le schede con un ID hanno un indirizzo diretto e la navigazione tra le sezioni usa la cronologia del browser. Su desktop la barra laterale si comprime. Su tablet e mobile si apre dal menu della barra superiore. Le tabelle larghe scorrono nella propria area, con controlli di ricerca, filtri, ordinamento e paginazione dove applicabili.

## Inventario e mappatura

| Funzione nel Master originale | Collocazione nel nuovo workspace | Operazioni e dati |
| --- | --- | --- |
| Dashboard e aggiornamento | Panoramica | Società registrate, voti complessivi e negli ultimi 7 giorni, partite. I contatori di ClubHub attivi e tornei derivano dai rispettivi elenchi. |
| Analisi delle votazioni | Analytics | Classifica per società, serie temporali globali e per società, partite con più voti nello storico e negli ultimi 7 giorni. |
| Sponsorizzazioni, engagement e riepilogo mensile | Analytics | Impression, clic, CTR, durata e medie di fruizione, dettaglio per società, confronto mensile per voti, partite e utenti unici. |
| Elenco e anagrafica società | Società sportive | Logo, nome, ID, sport, città, slug o URL, stato, BAR, costo e credito SMS, data di creazione. Link pubblici e amministrativi, totali di voti e partite e ultimo evento nella scheda. |
| Creazione e modifica società | Società sportive → Nuova società o Modifica | Anagrafica, sport, logo con URL o file, tre colori, stato, BAR, SMS e arena. I valori storici restano leggibili anche quando lo sport è assente. |
| Associazione e cancellazione società | Scheda società → Arena 3D e azioni della scheda | Assegna o scollega un’arena. L’eliminazione richiede la password del superadmin e mostra l’eventuale esito parziale della pulizia dei file. |
| Arene 3D | Arene → Catalogo, scheda ed editor | Nome, slug, città, squadra, capienza, stato e JSON della scena. Creazione e aggiornamento con validazione della configurazione. |
| Elenco e amministrazione ClubHub | ClubHub → Elenco, creazione e scheda | Stato, identificativo società Live, accesso al sito e al CMS, squadra/contenuti e visite totali e uniche. Il form comprende città, stagione, tagline, logo e copertina, colori e categorie. Generazione e copia credenziali, reset manuale o generato e cancellazione con conferma. |
| Domini personalizzati ClubHub | Scheda ClubHub → Domini personalizzati | Associazione e alias `www`, record DNS copiabili, verifica e aggiornamento periodico, dominio primario e rimozione con conferma. |
| Elenco e amministrazione tornei | Tornei → Elenco, creazione e scheda | Stato, fase, squadre, partite e username. Creazione con nome, formato, date e luogo. Link e credenziali copiabili, reset con revoca delle sessioni e cancellazione con conferma. La gestione operativa si apre nel portale del torneo. |
| Redirect QR | Configurazione → Redirect QR | Origine, destinazione, stato, data e accessi. Creazione, modifica dell’origine o della destinazione, attivazione, disattivazione ed eliminazione con conferma. |
| Minigiochi sotto la scheda Società | Configurazione → Minigiochi | Associazioni per disciplina, opzione esclusiva Tutti, selezione vuota per disattivare il gioco e salvataggio per singolo gioco. |
| Marchio e report | Configurazione → Report e marchio | Nome emittente, caricamento, anteprima e rimozione logo, ripristino dei valori salvati e salvataggio. |
| Accesso e sessione | Accesso al workspace e barra superiore | Login, controllo dei privilegi tramite endpoint Master protetto, ritorno al login alla scadenza, logout e collegamento al portale società. |

## Contratti e relazioni

Tutte le richieste del workspace usano il client HTTP della piattaforma con il cookie `admin_session`. La verifica al caricamento usa `GET /admin/master/summary`, che richiede il ruolo `superadmin` anche quando la sessione è valida. Login e logout usano `/admin/login` e `/admin/logout`. Le informazioni di ruolo nel browser servono alla compatibilità con gli altri pannelli. L’autorizzazione delle richieste resta sul server.

I percorsi seguenti sono relativi a `/api/admin/master`.

| Risorsa | Metodi e percorsi |
| --- | --- |
| Sintesi e analisi | `GET /summary`, `GET /analytics` |
| Società | `GET/POST /organizations`, `GET/PUT/DELETE /organizations/{id}`, `PATCH /organizations/{id}/arena` |
| Arene | `GET/POST /arenas`, `GET/PUT /arenas/{id}` |
| ClubHub | `GET/POST /clubs`, `PUT /clubs/{id}/password`, `DELETE /clubs/{id}` |
| Domini ClubHub | `GET/POST /clubs/{id}/domains`, `POST /clubs/{id}/domains/verify`, `PUT /clubs/{id}/domains/{domainID}/primary`, `DELETE /clubs/{id}/domains/{domainID}` |
| Tornei | `GET/POST /tournaments`, `PUT /tournaments/{eventId}/password`, `DELETE /tournaments/{eventId}` |
| Redirect | `GET/POST /qr-redirects`, `PUT /qr-redirects/{id}/active`, `DELETE /qr-redirects/{id}` |
| Minigiochi | `GET /minigames`, `PUT /minigames/{gameId}` |
| Marchio dei report | `GET/PUT /report-branding` |

La società mantiene l’associazione facoltativa `arena_id`. ClubHub espone `organizationId` e ha una propria identità, credenziali, contenuti e domini. La cancellazione ClubHub usa il proprio ID e il servizio conserva la società Live. Le operazioni sul torneo usano `eventId` per la risorsa amministrativa e lo slug per i collegamenti pubblici e operativi.

La rinomina di un redirect richiede due richieste con l’API esistente. Se il nuovo percorso viene salvato e la rimozione del precedente fallisce, il form indica il salvataggio parziale e il tentativo successivo ripete la sola rimozione. La creazione ClubHub può lasciare la società Live creata quando una fase successiva fallisce lato server. Il workspace mostra l’errore restituito dal servizio e lascia l’operatore in controllo del tentativo successivo.

## Collaudo

- Build Vite e 178 test frontend superati, inclusi 5 casi di regressione sul routing e sulla rinomina dei redirect.
- In Edge a 1440, 768 e 390 px sono stati verificati la navigazione, il controllo del ruolo, il caricamento da link diretti, ricerca, ordinamento e paginazione, protezione delle modifiche, selettori, finestre di conferma e aggiornamento delle schermate dopo i salvataggi. Il percorso `/admin/master` ha aperto il pannello originale.
- Il collaudo in Edge ha intercettato le richieste con dati locali di prova. Ha coperto creazione, modifica ed eliminazione di società, assegnazione di un’arena, creazione e modifica di arene, creazione ClubHub e associazione di un dominio, creazione, reset password ed eliminazione di tornei, creazione di un redirect e salvataggio di minigiochi e marchio report.
- I test Go mirati per la conferma con password e la normalizzazione dei domini sono passati. La verifica delle scritture su SQLite richiede un ambiente con CGO e un compilatore compatibile con `go-sqlite3`. La risoluzione DNS reale richiede un dominio di prova configurato.

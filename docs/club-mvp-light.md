# MVP Light per ClubHub

## Configurazione

1. In **Calendario**, salva la partita con la divisione corretta. Apri **MVP Light** dalla partita o dalla console della diretta.
2. In **Configura convocati**, seleziona i giocatori della divisione e premi **Salva convocati**. Puoi farlo prima di avviare la diretta.
3. Avvia la diretta e premi **Attiva votazione**. Se sono votabili più partite, usa **Associa al QR ClubHub** sulla partita da aprire con il QR.
4. Apri **Classifica live** per seguire i risultati. La vista si aggiorna via SSE mentre è attiva. Quando concludi la partita, torna alla sua voce **MVP Light** nel calendario e apri **Classifica finale**. Gli ex aequo rimangono visibili.

Sul dominio ClubHub il QR punta a `/mvp`. Sul dominio della piattaforma il percorso è `/club/<slug>/mvp`. Il percorso apre automaticamente una singola votazione disponibile. Se ne trova più di una, usa la partita associata al QR. Se la scelta è ambigua o nessuna partita è votabile, torna alla home. I link diretti terminano con `/mvp/<id-partita>`.

## Dati e configurazione tecnica

All'avvio del backend, `EnsureClubTables` applica la migrazione SQLite idempotente. La migrazione crea le tabelle per configurazioni, convocazioni e voti, genera la chiave persistente di firma dei cookie e installa i trigger per chiusura e conservazione dello storico. Il passaggio di una partita a conclusa chiude la votazione nella stessa scrittura del risultato. Non servono variabili d'ambiente aggiuntive. I domini personalizzati continuano a usare la configurazione ClubHub.

Il primo voto blocca la convocazione. Il salvataggio del roster conserva gli ID dei giocatori esistenti. Se un convocato viene rimosso dal roster, scompare dalla scheda pubblica e i voti precedenti restano associati ai dati del giocatore al momento della convocazione. Dal calendario si può eliminare una partita anche se ha ricevuto voti. La conferma elimina anche convocazioni, voti e classifica MVP Light della partita. Per eliminare una divisione dal CMS, occorre prima eliminare le sue partite con voti.

Il backend consente un voto per partita e per cookie anonimo firmato. Usa un vincolo SQLite per gestire invii simultanei e limita la frequenza dei tentativi. La protezione è riferita al browser e al dominio. Browser diversi, navigazione privata o cancellazione dei cookie permettono di ottenere una nuova identità.

## Verifica manuale

1. Salva 12 convocati da un roster di 18. Avvia la diretta, attiva MVP Light e apri il QR su uno smartphone. Verifica che compaiano i 12 convocati e che il punteggio segua la console.
2. Seleziona una card, conferma, poi ricarica nello stesso browser. Attendi la conferma del server e verifica lo stato **Hai già votato** dopo la ricarica. Controlla che l'admin mostri un solo voto e che la lista sia bloccata.
3. Apri la votazione per un secondo incontro della stessa divisione con convocati differenti. Verifica i due link diretti, l'associazione del QR e la possibilità di votare di nuovo per il secondo incontro.
4. Concludi la partita mentre la pagina di voto è aperta. Verifica il ritorno alla home e consulta la classifica finale dal calendario.

# Report post partita della home /:slug/new

> Per le query realmente usate dal pulsante report e dalle sue 20 KPI, leggere [Query e calcolo delle KPI del report evento](engagement-report-queries.md). Questo documento riguarda le 10 query manuali sulle fonti operative. Le limitazioni sulla mancanza di impression e visite si riferiscono ai dati storici senza i segnali dedicati della nuova telemetria.

Le [10 query SQL](new-fan-post-match.sql) sono scritte per SQLite e per lo schema dell'ultima versione del backend presente nel repository. Non richiedono PostHog e non interrogano il database locale dell'ambiente di sviluppo. Non e' stata verificata la versione effettivamente distribuita sul server della partita.

Ogni blocco numerato contiene una query indipendente di sola lettura. Il file non crea tabelle o viste e non modifica dati. Eseguire i blocchi separatamente nel client SQL collegato al database da analizzare. Impostare i parametri nel client, oppure sostituirli con valori SQL letterali.

| Parametro | Valore da inserire |
| --- | --- |
| `:event_id` | ID numerico della partita conclusa |
| `:club_slug` | Slug del club, ossia la parte prima di `/new`, tra apici se sostituito direttamente nel testo |
| `:from_utc` | Solo query 02: inizio del periodo da analizzare, formato `YYYY-MM-DD HH:MM:SS`, UTC |
| `:to_utc` | Solo query 02: fine del periodo, stesso formato, UTC; questo estremo e' escluso |

Scegliere per la query 02 un intervallo che includa l'apertura della piattaforma e il prepartita, se si vogliono contare anche le registrazioni precedenti all'inizio del gioco. La registrazione non salva l'ID evento: il periodo scelto non dimostra che una registrazione sia stata causata dalla partita.

Eseguire prima la query 01: deve restituire una riga con il club e l'evento corretti. Se non restituisce righe, controllare ID e slug prima di interpretare gli zeri delle altre query.

| Query | Risultato |
| --- | --- |
| 01 | Riepilogo della partita: partecipanti osservabili, profili, dispositivi ospiti, voti, feedback e click sponsor |
| 02 | Nuovi profili creati e completati nel periodo, raggruppati per ora UTC |
| 03 | Risultati MVP, quote percentuali, pari merito e associazione attuale dei voti a profili o ospiti |
| 04 | Reaction Test, Quick Quiz, Tap Challenge, Sponsor Rush e Memory Flash: premi confermati, partecipanti, monete e ripetizioni |
| 05 | Basketball Shootout, Playmaker e Crossover Rush: partecipanti, sessioni disponibili, tentativi, esiti, percentuali di successo e monete |
| 06 | Monete accreditate/spese nell'evento, distinte per causale del ledger |
| 07 | Cornici e card: acquisti, acquirenti distinti, costo medio e monete spese nell'evento |
| 08 | Distribuzione delle risposte al sondaggio e percentuale di feedback con suggerimento libero |
| 09 | Click per sponsor, dispositivi riconoscibili, click senza identificativo e quota dei click ricevuti |
| 10 | Classifica attuale del club, saldo, punti, badge dell'evento, cornice indossata e card possedute |

## Interpretazione dei risultati

- **Attribuzione a /new.** Le query usano le fonti che alimentano le funzionalita di questa pagina. Le tabelle condivise di voti, feedback, sponsor, economia e profili non conservano sempre la provenienza della UI. Se per la stessa partita sono state usate altre interfacce, i risultati possono includere anche quelle attivita. Non e' possibile ricostruire a posteriori una provenienza mancante.
- **Persone e dispositivi.** Un partecipante e' un profilo registrato oppure un dispositivo ancora ospite. L'adozione dei dati dell'ospite durante la registrazione puo' associare le attivita precedenti al profilo. La suddivisione profili/ospiti descrive quindi le associazioni attuali, non necessariamente lo stato al momento dell'azione. Più dispositivi ospiti possono appartenere alla stessa persona; dispositivi condivisi non distinguono sempre persone diverse.
- **Copertura della query 01.** Sono osservabili voti, premi, ticket basket, sessioni Playmaker/Crossover e acquisti. Chi ha soltanto aperto la home, letto le istruzioni, consultato la classifica o il profilo non viene contato. I click sponsor e i feedback sono riepilogati separatamente e non si sommano ai partecipanti distinti.
- **Registrazioni.** La query 02 conta solo profili creati nel periodo, ora completi, con la prima ricevuta di accettazione anch'essa nel periodo. Non conta ogni versione dell'informativa come una nuova registrazione. Profili preesistenti, registrazioni incomplete e creazioni precedenti alla finestra sono esclusi. Non misura la conversione da visitatore a registrato.
- **Voti.** La query 03 legge la scelta conservata per ogni dispositivo. Cambiare voto aggiorna `created_at`: i due estremi temporali non ricostruiscono la cronologia originale. Gli ospiti poi registrati possono risultare tra i voti associati a profili.
- **Giochi a premio fisso.** La query 04 conta premi confermati, non tutte le partite iniziate, perse o completate senza ritirare le monete. Un retry dello stesso `request_id` non genera una nuova riga e non viene contato due volte. I premi legacy vengono tenuti separati; non rappresentano una cronologia completa delle vecchie giocate. `premi_oltre_il_primo` descrive ripetizioni premiate, non una misura delle aperture del gioco.
- **Giochi con validazione server.** La query 05 non somma sessioni e tentativi. I canestri, gli assist e i duelli vinti valgono attualmente una moneta ciascuno. Il successo percentuale usa solo i tentativi con esito: ticket basket scaduti o pendenti non sono automaticamente tiri sbagliati o abbandoni. Una sessione creata non dimostra la sua durata effettiva o il completamento.
- **Monete e acquisti.** Le query 06 e 07 usano il ledger con l'ID evento e le causali delle funzioni analizzate. Saldi iniziali e trasferimenti non sono guadagni della partita. Non sommare i loro accrediti alle ricevute delle query 04/05: descrivono le stesse monete. La cornice ha `scope_event_id = 0` nell'inventario; la spesa della partita si identifica dal movimento. Unione delle collezioni e possesso attuale non equivalgono al numero storico di acquisti. Il report riguarda monete virtuali.
- **Feedback.** Sono risposte ricevute, non rispondenti unici verificabili: la tabella non conserva un identificativo del tifoso. I testi delle domande possono essere personalizzati; le chiavi `experience`, `team_spirit`, `perks_interest` e `mini_games_interest` vanno confrontate con la configurazione del sondaggio dell'evento. Non viene calcolato un NPS.
- **Sponsor.** La query 09 mostra click e dispositivi dove presenti. `quota_click_percento` e' la quota dei click totali, non il CTR. La home /new non invia le impression dei banner, quindi manca il denominatore per un CTR della pagina. La lista usa i record sponsor ancora esistenti.
- **Classifica, badge e collezione.** La query 10 riproduce l'ordinamento del club per punti cumulativi, con ID crescente a pari punti. Saldo, classifica, cornice e inventario sono attuali: non esiste uno snapshot automatico alla chiusura dell'evento. I punti del badge dell'evento vengono ricavati dalle ricevute dei premi, non dal saldo spendibile. Per rispettare il calcolo del profilo, vengono incluse anche eventuali ricevute Tap Live/branded condivise se presenti; tali giochi non vengono presentati come funzionalita della home /new nelle query 04/05.

Non sono ricostruibili da queste fonti le visite complete alla home, il percorso tra pannelli, i click sull'icona informazioni, le aperture di profilo/classifica, il tempo reale di permanenza, gli abbandoni completi di tutti i minigiochi, i tempi di reazione della nuova UI o la conversione visita/registrazione. Le query non esportano numeri di telefono, token SMS/sessione o suggerimenti liberi.

## Fonti del codice

- `wcmvpvs-back/service/database/database.go`: schema, voti, profili e classifica.
- `wcmvpvs-back/service/database/fan_economy.go`: ledger, trasferimenti e inventario.
- `wcmvpvs-back/service/database/fan_minigame_rewards.go` e `fan_journey.go`: ricevute premi e idempotenza.
- `wcmvpvs-back/service/database/basketball_shots.go`, `playmaker.go`, `crossover.go`: azioni dei tre giochi con validazione server.
- `wcmvpvs-back/service/database/fan_shop.go`: acquisti e calcolo del badge.
- `wcmvpvs-front/src/views/FanLiveCento.vue`: funzionalita effettivamente accessibili da /new.

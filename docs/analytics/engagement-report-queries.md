# Query e calcolo delle 30 KPI del report evento

Questo documento descrive il calcolo eseguito dal pulsante report in **Storico eventi**, per la home `/:slug/new`. Il report combina i segnali di `engagement_report_signals` con le registrazioni operative delle funzionalità della home. Non usa i riepiloghi già calcolati nello storico eventi.

Il caso segnalato riguarda **l'evento 135 in produzione**. Il codice è stato esaminato nel repository; non è stata eseguita una lettura del database di produzione. L'ID della società deve essere ricavato dall'evento e verificato rispetto all'amministratore autenticato: non si presume che sia 17.

## Parametri e lettura dei risultati

Il backend usa SQLite e parametri posizionali `?`. Negli esempi SQL del documento i parametri hanno nomi espliciti per poterli eseguire separatamente in un client SQL:

| Parametro | Significato |
| --- | --- |
| `:event_id` | Evento richiesto; nel caso segnalato, `135` |
| `:organization_id` | Società proprietaria dell'evento, validata rispetto alla sessione admin |
| `:event_start` | Valore `start_datetime` dell'evento richiesto |
| `:previous_event_id` | Evento precedente della stessa società, scelto dalla query dedicata |

Le query riportate sono di sola lettura. Il flusso applicativo salva anche una bozza di calcolo e, dopo la conferma nella modale, una versione approvata del report: questi salvataggi non sono fonti di misurazione delle KPI.

Non esiste una query distinta per ognuna delle 30 KPI. Il backend carica le osservazioni, riconcilia le identità e le azioni duplicate, costruisce i fatti di base e applica le formule in Go. Una stessa query contribuisce quindi a più KPI. Le dieci KPI aggiuntive riutilizzano le stesse fonti: non introducono query di estrazione ulteriori.

`null`, mostrato come **N/D**, significa dato non ricostruibile oppure denominatore assente o nullo. Non viene sostituito automaticamente con zero. Una percentuale viene calcolata soltanto se entrambi i fatti sono disponibili e il denominatore è maggiore di zero.

## Identificazione dell'evento

Prima di leggere i dati, il backend ricava la società con questa query e verifica che coincida con quella dell'amministratore autenticato:

```sql
SELECT organization_id
FROM events
WHERE id = :event_id;
```

Per l'evento 135, il risultato di questa query fornisce il valore di `:organization_id`. La query dei metadati applica nuovamente entrambi i vincoli:

```sql
SELECT e.id,
       e.start_datetime,
       COALESCE(e.location, ''),
       COALESCE(t1.name, ''),
       COALESCE(t2.name, '')
FROM events e
LEFT JOIN teams t1 ON t1.id = e.team1_id
LEFT JOIN teams t2 ON t2.id = e.team2_id
WHERE e.id = :event_id
  AND e.organization_id = :organization_id;
```

La query serve a titolo, luogo, data e contesto del report. Il numero dei presenti non viene dedotto da squadre, capienza o affluenza presunta.

## Segnali dedicati della home

```sql
SELECT signal_id,
       name,
       session_id,
       device_key,
       fan_id,
       occurred_at,
       metadata
FROM engagement_report_signals
WHERE event_id = :event_id
ORDER BY occurred_at, signal_id;
```

Il controllo della società è già stato applicato all'evento. `metadata` viene decodificato da JSON. I segnali riconosciuti sono `page_view`, `active_time`, `vote_completed`, `game_started`, `game_completed`, `registration_started`, `registration_completed`, `redemption_completed`, `sponsor_view`, `sponsor_click`, `sponsored_game_view`, `shop_view` e `feedback_completed`.

La chiave primaria `(event_id, signal_id)` rende idempotente la ricezione dello stesso segnale. I segnali non vengono contati con un semplice `COUNT(*)`: un voto, un giocatore o un round possono avere più osservazioni riferite alla stessa azione.

## Fonti operative: 14 letture, inclusa la convalida dei profili

Il caricatore `loadReportOperationalData(eventID, orgID)` legge le tabelle di seguito anche quando `engagement_report_signals` è vuota. Il suo risultato viene trasformato in osservazioni interne: non vengono inserite righe retroattive nella tabella dei segnali.

Prima delle 13 letture, questa query convalida gli ID dei profili rispetto alla società e scarta i profili ancora in registrazione:

```sql
SELECT p.id
FROM fan_profiles p
JOIN events e ON e.organization_id = p.organization_id
WHERE e.id = :event_id
  AND e.organization_id = :organization_id
  AND p.registration_pending = 0;
```

La query **non** attribuisce tutte le anagrafiche del club a quell'evento: serve esclusivamente a convalidare gli ID incontrati nelle attività. Non vengono letti telefono, token o altri dati personali per il report.

### 1. Voti MVP

```sql
SELECT v.id,
       COALESCE(v.device_id, ''),
       COALESCE(v.user_id, 0),
       COALESCE(v.created_at, '')
FROM votes v
JOIN events e ON e.id = v.event_id
WHERE v.event_id = :event_id
  AND e.organization_id = :organization_id;
```

Ogni riga prova un voto salvato. L'insieme dei votanti viene deduplicato per identità; il conteggio delle righe resta disponibile separatamente come `details.operational.votes`. Il backend legge il voto attualmente conservato: non ricostruisce la cronologia dei cambi di voto.

### 2. Ricevute dei minigiochi con ID round

```sql
SELECT request_id, device_id, fan_id, reward_key, coins, created_at
FROM fan_minigame_rewards
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Le ricevute positive di Reaction Test, Quick Quiz, Tap Challenge, Sponsor Rush e Memory Flash provano una giocata premiata. `request_id` è l'ID round, condiviso con la telemetria: ricezione del premio e `game_completed` dello stesso round non aggiungono due completamenti. I retry della stessa richiesta non sono nuove giocate.

### 3. Ricevute legacy dei giochi

```sql
SELECT device_id || ':' || reward_key,
       device_id,
       fan_id,
       reward_key,
       coins,
       created_at
FROM fan_journey_rewards
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Vengono usate soltanto le ricevute positive dei cinque giochi indicati sopra. La causale voto non diventa una giocata. Una ricevuta legacy non ha un ID round: prova al massimo una giocata completata per identità e gioco. Se esiste già un completamento moderno della stessa coppia, non ne aggiunge un altro. Se esistono soltanto avvii moderni, conserva la prova minima di un completamento senza aggiungere un avvio ulteriore o simulare un rigioco. È una misura minima confermata, non una ricostruzione di tutte le vecchie partite.

### 4. Risposte del quiz con ID round

```sql
SELECT device_id || ':' || request_id || ':' || question_id,
       device_id,
       request_id
FROM fan_minigame_quiz_answers
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Le risposte dimostrano l'avvio del quiz; più risposte dello stesso `request_id` appartengono a un solo round. Non certificano un completamento senza la relativa prova. Questa tabella non ha un timestamp di creazione o un ID profilo: non ne viene inventato uno.

### 5. Risposte legacy del quiz

```sql
SELECT device_id || ':' || question_id,
       device_id,
       ''
FROM fan_journey_quiz_answers
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Dimostrano partecipazione al quiz, con un solo identificativo convenzionale legacy per identità. Più domande non equivalgono a più partite. Valgono gli stessi limiti della ricevuta legacy sul confronto con i round moderni.

### 6. Tiri del Basketball Shootout

```sql
SELECT shot_id, device_id, fan_id, status, issued_at
FROM basketball_shots
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Solo i tiri con esito `made` o `miss` provano una giocata effettiva. Un ticket `pending` o `expired` non prova un tiro né un abbandono. Nel Basketball Shootout la telemetria definisce il singolo tiro come round: `shot_id` viene usato sia dal gioco sia dal caricatore operativo. Un tiro confermato vale quindi un round completato e non viene contato due volte quando è presente anche il corrispondente segnale.

### 7. Sessioni Playmaker

```sql
SELECT session_id, device_id, fan_id, started_at
FROM playmaker_sessions
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

La sessione prova un avvio. La scadenza della sessione non prova il completamento. `session_id` serve a ricondurre sessione e passaggi allo stesso round.

### 8. Passaggi Playmaker

```sql
SELECT session_id || ':' || attempt,
       device_id,
       fan_id,
       resolved_at,
       session_id
FROM playmaker_passes
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Più passaggi della stessa sessione non diventano più round. Le azioni confermano partecipazione al gioco; il numero di passaggi non è il numero di partite completate.

### 9. Sessioni Crossover Rush

```sql
SELECT session_id, device_id, fan_id, started_at
FROM crossover_sessions
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

Come per Playmaker, la sessione prova l'avvio e non il completamento. L'identificativo è `session_id`.

### 10. Duelli Crossover Rush

```sql
SELECT session_id || ':' || duel,
       device_id,
       fan_id,
       resolved_at,
       session_id
FROM crossover_duels
WHERE event_id = :event_id
  AND organization_id = :organization_id;
```

I duelli di una stessa sessione non diventano round aggiuntivi. Un duello registrato prova partecipazione; l'intero round resta distinto dal singolo duello.

### 11. Ledger di monete e riscatti

```sql
SELECT id, device_id, fan_id, source, delta, created_at
FROM fan_wallet_movements
WHERE event_id = :event_id
  AND organization_id = :organization_id
  AND source IN (
    'basketball', 'playmaker', 'crossover',
    'journey:vote', 'journey:reaction', 'journey:quiz', 'journey:tap',
    'journey:sponsor-rush', 'journey:memory-flash', 'journey:basketball',
    'journey:playmaker', 'journey:crossover',
    'minigame:reaction', 'minigame:quiz', 'minigame:tap',
    'minigame:sponsor-rush', 'minigame:memory-flash', 'minigame:basketball',
    'minigame:playmaker', 'minigame:crossover',
    'shop:club_frame', 'shop:match_card'
  );
```

Gli accrediti positivi alimentano le monete ottenute; gli addebiti alimentano le monete spese. Le causali `shop:` con `delta < 0` provano il riscatto nell'evento. L'accredito `journey:vote` conferma il voto senza aggiungere un secondo votante alla relativa riga di `votes`. Per i giochi riconosciuti l'accredito prova partecipazione, mentre il conteggio dei round resta affidato agli ID delle ricevute o delle sessioni. La lista esplicita esclude saldi iniziali, trasferimenti e causali di giochi estranei alla home.

Il saldo attuale di `fan_wallets` o `guest_wallets` non è un guadagno della partita. Il possesso di una cornice in `fan_shop_items`, che può avere ambito club, non ne prova l'acquisto in questo evento. Per questo l'attribuzione economica usa il ledger con `event_id`; non somma il saldo ai movimenti né somma una moneta del ledger alla stessa moneta della ricevuta premio. Le cifre sono monete virtuali, non euro o ricavi monetari.

### 12. Risposte anonime al feedback

```sql
SELECT f.perks_interest
FROM event_feedback f
JOIN events e ON e.id = f.event_id
WHERE f.event_id = :event_id
  AND e.organization_id = :organization_id;
```

Tutte le righe alimentano il numero di risposte ricevute. Per l'interesse premi sono valide `yes`, `maybe`, `little`, `no`; soltanto `yes` è positivo. Quando sono presenti risposte operative valide, si usano queste per la KPI 20 senza sommarvi le copie telemetriche.

La tabella non contiene un'identità del rispondente: non si fabbricano utenti unici né si sommano le risposte anonime alle risposte identificate. Quando sono presenti segnali `feedback_completed`, la KPI 18 usa soltanto il campione identificabile tracciato, senza aggiungere le righe anonime. Il totale delle risposte operative resta un dato distinto. Il campo storico `experience` esprimeva la facilità del voto, quindi non viene convertito nella soddisfazione complessiva della KPI 19. Quella KPI usa la domanda dedicata nei nuovi segnali.

### 13. Clic sponsor

```sql
SELECT c.id,
       COALESCE(c.device_id, ''),
       c.sponsor_id,
       COALESCE(NULLIF(s.report_name, ''), s.name),
       c.clicked_at
FROM sponsor_clicks c
JOIN events e ON e.id = c.event_id
JOIN sponsors s
  ON s.id = c.sponsor_id
 AND s.organization_id = e.organization_id
WHERE c.event_id = :event_id
  AND e.organization_id = :organization_id;
```

Il riepilogo distingue clic totali e clic privi di identità. I clic con dispositivo possono contribuire all'insieme dei cliccatori, ma diventano numeratore del CTR solo se la stessa identità ha un'esposizione qualificata allo stesso sponsor. Un clic non viene trasformato in un'impression. Senza esposizioni non si ricava un CTR a partire dai soli clic. Anche nel dettaglio per sponsor, `exposure_measured = false` comporta la visualizzazione N/D delle misure di esposizione, anziché uno zero che suggerirebbe una rilevazione effettiva. La query considera gli sponsor tuttora presenti nella tabella e appartenenti alla società.

## Riconciliazione fra fonti

1. Il `device_id` operativo viene normalizzato nella stessa chiave della telemetria: SHA-256 di `organization_id + ":" + device_id`. Non si confronta un ID grezzo con un hash.
2. Un'attività con profilo valido usa l'identità `fan:<id>`. Un'attività ospite viene collegata al profilo solo se, nello stesso evento, quel dispositivo è associato in modo non ambiguo a un solo profilo. Altrimenti resta un'identità dispositivo. Un dispositivo condiviso non consente di riconoscere tutte le persone fisiche che l'hanno usato.
3. Votanti, giocatori, riscattanti e partecipanti sono insiemi di identità, non somme dei conteggi delle tabelle. Per esempio voto salvato e segnale `vote_completed` dello stesso partecipante producono un solo votante.
4. I round si riconoscono per identità, gioco e ID round. Premio, risposte quiz e telemetria con lo stesso ID descrivono lo stesso round; nel basket l'ID condiviso è `shot_id`. Il ledger conferma la partecipazione e non aggiunge un nuovo round alla ricevuta. La prova legacy può confermare almeno un completamento mancante, ma non viene aggiunta a completamenti moderni già confermati per la stessa identità e gioco e non crea una ripetizione quando sono già riconoscibili avvii moderni.
5. Le tabelle operative condivise non conservano sempre la pagina di provenienza. Le attività storiche sono attribuibili all'evento e alla società; non è possibile certificare retroattivamente che siano avvenute soltanto in `/new` se anche un'altra interfaccia scriveva nelle stesse tabelle.

`details.sources` indica quante righe sono state lette per ogni fonte, anche se alcune non producono un partecipante o un round valido. Comprende le 13 letture operative, `fan_profiles` per la convalida e `engagement_report_signals`. Il numero dei profili consultati per convalidare la società non è il numero dei partecipanti all'evento. Questi conteggi sono diagnostici: non coincidono necessariamente con il valore di una KPI.

## Evento precedente per il ritorno dei profili

```sql
SELECT id
FROM events
WHERE organization_id = :organization_id
  AND id <> :event_id
  AND (
    julianday(start_datetime) < julianday(:event_start)
    OR (
      julianday(start_datetime) = julianday(:event_start)
      AND id < :event_id
    )
  )
ORDER BY julianday(start_datetime) DESC, id DESC
LIMIT 1;
```

Per l'evento trovato vengono lette le stesse fonti di misurazione. Il confronto riguarda i profili identificati attivi: non si collegano dispositivi ospiti appartenenti a eventi diversi. Senza un precedente comparabile o senza un denominatore identificabile, il ritorno resta N/D.

## Formule delle 30 KPI

I nomi tra backtick sono le chiavi dei fatti restituiti dall'API e modificabili nella modale. Le percentuali sono moltiplicate per 100.

| N. | KPI | Formula eseguita |
| --- | --- | --- |
| 1 | Visitatori digitali unici; con fonti operative, Partecipanti digitali osservabili | `visitors` |
| 2 | Adozione del pubblico presente | `onsite_visitors / attendees × 100` |
| 3 | Partecipazione attiva | `active_visitors / engagement_eligible_visitors × 100` |
| 4 | Tempo attivo medio misurato | `active_seconds / timed_visitors`, in secondi |
| 5 | Partecipazione al voto MVP | `voters / vote_eligible_visitors × 100` |
| 6 | Partecipazione ai giochi | `players / game_eligible_visitors × 100` |
| 7 | Round di gioco completati; con fonti operative, Giocate completate confermate | `rounds_completed` |
| 8 | Tasso di rigioco | `repeat_players / players × 100` |
| 9 | Uso di più esperienze | `multi_experience_visitors / active_visitors × 100` |
| 10 | Nuovi profili verificati | `new_profiles` |
| 11 | Conversione alla registrazione | `new_profiles / registration_starters × 100` |
| 12 | Ritorno dall'evento precedente | `returning_profiles / previous_event_profiles × 100` |
| 13 | Conversione al riscatto | `redeemers / shop_eligible_profiles × 100` |
| 14 | Copertura unica sponsor | `sponsor_reach` |
| 15 | Impression sponsor visibili | `sponsor_impressions` |
| 16 | CTR sponsor su utenti esposti | `sponsor_clickers / sponsor_reach × 100` |
| 17 | Partecipazione ai giochi sponsorizzati | `sponsored_players / sponsored_exposed_visitors × 100` |
| 18 | Risposta al feedback | `feedback_respondents / feedback_eligible_visitors × 100` |
| 19 | Esperienza positiva | `experience_positive / experience_responses × 100` |
| 20 | Interesse per premi e vantaggi | `perks_positive / perks_responses × 100` |
| 21 | Tempo totale stimato sulla piattaforma | `estimated_seconds`, in secondi |
| 22 | Tempo medio stimato per partecipante | `estimated_seconds / estimated_timed_visitors`, in secondi |
| 23 | Partecipanti identificati | `identified_visitors` |
| 24 | Partecipanti ospiti | `guest_visitors` |
| 25 | Clic complessivi verso gli sponsor | `total_sponsor_clicks` |
| 26 | Partecipanti attivi con gli sponsor | `sponsor_interactors` |
| 27 | Coinvolgimento del pubblico esposto agli sponsor | `sponsor_engaged_exposed / sponsor_reach × 100` |
| 28 | Riscatti di premi virtuali | `shop_redemptions` |
| 29 | Nuovi profili sul pubblico osservabile | `new_profiles / visitors × 100` |
| 30 | Feedback raccolti | `feedback_collected` |

### Fonte e denominatore di ciascuna KPI

| N. | Dato ricostruito e condizioni di disponibilità |
| --- | --- |
| 1 | Unione delle identità dei segnali e delle attività operative riconoscibili. Con dati storici operativi è un **minimo osservabile**, perché non ricostruisce chi ha soltanto aperto la pagina. Non è un conteggio certificato di persone fisiche. |
| 2 | `onsite_visitors` richiede un `page_view` con `source = arena`; `attendees` deve essere confermato dalla società nella modale. Senza entrambi, N/D. L'indicazione arena non costituisce una verifica della presenza fisica. |
| 3 | Le azioni di voto, giochi, riscatto e clic sponsor possono provenire da entrambe le fonti; i feedback identificati arrivano dai segnali. Il denominatore `engagement_eligible_visitors` coincide con le identità osservabili solo quando è stato rilevato almeno un `page_view`. **Senza alcun `page_view` il denominatore resta N/D**, per evitare un artificiale 100% ottenuto da una popolazione composta solo da azioni salvate. Il numero di attivi rimane disponibile per gli altri confronti. |
| 4 | Il numeratore usa esclusivamente gli intervalli `active_time`; `timed_visitors` conta le identità con almeno un intervallo valido. Intervalli sovrapposti della stessa identità vengono uniti e ogni heartbeat vale al massimo 15 secondi. La media riguarda questo campione, escludendo le identità senza misurazioni del tempo. I timestamp di voto, premio o sessione non vengono convertiti in tempo di permanenza. Senza intervalli validi, N/D. |
| 5 | I votanti provengono da `votes` e dai segnali di voto, deduplicati. Il denominatore richiede la disponibilità osservata della votazione nei `page_view`; i votanti vengono inclusi nell'audience idonea quando questa è misurabile. I soli voti salvati non ricostruiscono chi poteva votare ma non lo ha fatto: il tasso resta N/D senza denominatore. |
| 6 | I giocatori provengono da avvii/completamenti, ricevute, risposte quiz, tiri risolti, sessioni, passaggi, duelli e accrediti gioco pertinenti. Il denominatore richiede i `page_view` con disponibilità giochi osservata; i soli giocatori non ricostruiscono il pubblico potenziale. |
| 7 | Completamenti telemetrici, ricevute positive dei cinque minigiochi e tiri basket risolti `made/miss`, deduplicati per round. Nel basket un tiro confermato è un round secondo la stessa definizione usata dalla telemetria. Le sessioni Playmaker/Crossover non diventano automaticamente completamenti. I dati legacy sono una prova minima: non rappresentano tutte le giocate non premiate. |
| 8 | Il numeratore richiede almeno due ID round distinti per la stessa identità, anche su giochi diversi. Una serie di passaggi o duelli nella stessa sessione resta un solo round. Le azioni senza ID round non dimostrano una ripetizione aggiuntiva. Con dati operativi incompleti il tasso riguarda i round osservabili, non tutte le ripetizioni reali. |
| 9 | Vengono riconosciute almeno due funzionalità tra voto, giochi, riscatto, clic sponsor e feedback identificato. Il denominatore è la popolazione attiva osservabile, quindi questo rapporto può essere ricostruito anche dalle sole azioni operative. Il risultato descrive le attività salvate, che potrebbero non includere tutte le esperienze realmente usate. |
| 10 | Servono segnali `registration_completed` associati a un profilo identificato. `fan_profiles` è un'anagrafica del club e non conserva l'evento di registrazione: essere un profilo attivo nell'evento non dimostra essersi registrato durante quell'evento. Nessuna finestra temporale arbitraria viene usata per attribuire nuove registrazioni. |
| 11 | Inizi registrazione e completamenti provengono dal percorso tracciato. Un completamento dimostra anche l'avvio dello stesso percorso e ne recupera l'eventuale segnale mancante. Le sole anagrafiche non ricostruiscono gli inizi o gli abbandoni. Senza il percorso attribuito all'evento, N/D. |
| 12 | Si confrontano i profili identificati con attività nei due eventi della stessa società, usando segnali e fonti operative. Il numeratore è l'intersezione dei profili attivi dei due eventi; il denominatore sono i profili attivi nell'evento precedente. Gli ospiti non vengono collegati fra eventi. |
| 13 | I riscatti sono ricavati dai segnali e dagli addebiti `shop:` del ledger. Il denominatore richiede profili identificati osservati in `page_view` o `shop_view`; il possesso attuale di un oggetto non ricostruisce il pubblico dello shop. Senza questa audience, N/D. |
| 14 | Solo esposizioni qualificate `sponsor_view` o `sponsored_game_view`, deduplicate per identità; più sponsor non moltiplicano il partecipante nel totale. I clic storici non sostituiscono le esposizioni. Senza esposizioni, N/D. |
| 15 | Esposizioni qualificate distinte per identità, sessione, sponsor, tipo di segnale e posizionamento. Il criterio client richiede visibilità di almeno il 50% per un secondo. Non si assume che ogni visita o gioco equivalga a un'impression. Senza esposizioni, N/D. |
| 16 | Cliccatori riconoscibili, da segnali o `sponsor_clicks`, che hanno anche un'esposizione allo **stesso sponsor**. Un'identità viene contata una sola volta nel totale. I clic anonimi restano nel riepilogo dei clic e non diventano utenti unici. Senza copertura sponsor misurata, N/D. |
| 17 | Il numeratore richiede avvio o completamento del gioco attribuito allo stesso sponsor visto in `sponsored_game_view`. Le ricevute senza attribuzione sponsor non permettono di inventare quella relazione. Senza il pubblico esposto al gioco sponsorizzato, N/D. |
| 18 | Se esistono segnali `feedback_completed`, il numeratore conta le identità rispondenti del **solo campione tracciato**; il denominatore richiede il pubblico con feedback disponibile osservato nei `page_view`. Le risposte anonime di `event_feedback` non vengono sommate e non rendono indisponibile il campione identificato. Se mancano i segnali di risposta o il relativo denominatore, N/D. Il tasso non viene presentato come copertura di tutte le risposte storiche; il totale operativo resta distinto. |
| 19 | Prima risposta valida per identità alla domanda dedicata: `awesome` o `good` sono positive; `ok` e `bad` sono valide non positive. La domanda storica sulla facilità del voto non entra né nel numeratore né nel denominatore. Senza risposte alla domanda dedicata, N/D. |
| 20 | Con risposte operative valide si usa `event_feedback`: `yes` positivo, `yes/maybe/little/no` denominatore. Non si sommano le copie telemetriche. Se tali risposte operative mancano, si usano le risposte identificate dei segnali. Il risultato riguarda il campione rispondente; non misura l'interesse di tutti i visitatori. |
| 21 | Stima ricavata dagli intervalli tra azioni con timestamp valido e dagli intervalli `active_time`, dopo la riconciliazione delle identità ospite/profilo e delle prove duplicate. Gli intervalli sovrapposti vengono uniti. Senza alcuna durata ricostruibile, N/D. Il metodo e la copertura sono specificati nella sezione seguente. |
| 22 | Il numeratore è la KPI 21; il denominatore comprende soltanto identità con almeno un intervallo temporale valido, sia ospiti sia profili. Le identità prive di intervalli non valgono zero e sono escluse dalla media. Non è il tempo medio di tutti i visitatori, di tutti gli spettatori o del solo pubblico registrato. |
| 23 | Identità osservate riconducibili a un profilo verificato della società dopo la riconciliazione locale all'evento. Le azioni precedenti da ospite non creano un partecipante aggiuntivo quando il dispositivo è collegato univocamente a quel profilo. L'identificazione non implica un consenso marketing. |
| 24 | Identità osservabili rimaste prive di un collegamento univoco a un profilo verificato. Possono essere dispositivi o sessioni; non certificano il numero di persone fisiche. Le KPI 23 e 24 ripartiscono il medesimo insieme osservabile della KPI 1. |
| 25 | Se il totale operativo di `sponsor_clicks` è positivo, viene usato quel totale, inclusi i clic senza identità, senza sommarvi copie telemetriche. In assenza di clic operativi si usa il minimo dei clic identificabili osservati nei segnali, deduplicati per identità, sponsor e ID dell'azione. Non è un numero di utenti unici e non dimostra acquisti. |
| 26 | Unione delle identità con almeno un clic sponsor oppure un avvio/completamento di gioco esplicitamente attribuito a uno sponsor. Ogni identità conta una sola volta anche se interagisce con più sponsor. Le prove senza identità e i giochi senza attribuzione sponsor sono esclusi. Non richiede un'esposizione osservata, quindi può essere disponibile anche quando la KPI 27 è N/D. |
| 27 | Per ogni sponsor si intersecano le identità esposte con quelle che hanno cliccato o avviato/completato un suo gioco; l'unione di queste intersezioni costituisce `sponsor_engaged_exposed`. Più sponsor non moltiplicano la stessa identità nel totale. Senza esposizioni qualificate e copertura unica misurata, N/D: un'interazione non viene trasformata retroattivamente in un'impression. |
| 28 | Se presenti, gli addebiti operativi `shop:club_frame` e `shop:match_card` con `delta < 0`, attribuiti all'evento, costituiscono il totale prioritario. Le copie telemetriche non si aggiungono. In assenza di addebiti si usa il minimo osservato nei segnali, una volta per identità e oggetto. Un riscatto è un'azione in monete virtuali: non equivale a ricavi in euro né a un'offerta commerciale realmente utilizzata. |
| 29 | Nuove registrazioni attribuite all'evento come nella KPI 10, divise per l'insieme delle identità digitali osservabili della KPI 1. Senza nuove registrazioni attribuibili o senza pubblico osservabile, N/D. Con dati storici privi di visite passive, il denominatore è un minimo verificabile; non è una conversione sul pubblico totale presente. |
| 30 | Se esistono risposte in `event_feedback`, si usa il totale delle righe operative senza aggiungervi le copie telemetriche. In loro assenza si usa il minimo delle identità rispondenti nei segnali. Le risposte operative possono essere anonime e non permettono di certificare utenti unici; il conteggio delle risposte resta distinto dalla soddisfazione della KPI 19 e dal tasso della KPI 18. |

### Stima del tempo: metodo riproducibile e copertura

Le KPI 21 e 22 sono marcate `source = estimated`, mostrato come **Stima da azioni**. Restano separate dal tempo attivo misurato della KPI 4. Il calcolo usa le osservazioni browser e operative già caricate, senza query aggiuntive e senza assegnare a priori una durata convenzionale a un voto, a un gioco o a un premio.

Per ciascuna identità riconciliata nell'evento il backend:

1. Deduplica le osservazioni e le prove della stessa azione. Gli ID di round legano avvio, ricevuta e completamento; prove legacy e accrediti non aggiungono una nuova azione se esiste già una prova datata più specifica. Una prova senza timestamp non elimina una ricevuta datata e un avvio moderno conserva il completamento legacy quando manca quello moderno datato. Clic vicini sullo stesso sponsor vengono ricondotti alla stessa osservazione quando distano al massimo due secondi. Per voti e altri duplicati viene preferito il timestamp browser rispetto alla ricevuta operativa tardiva.
2. Ordina i punti temporali validi. Fra due azioni consecutive con distanza positiva **non superiore a 300 secondi**, considera solo l'intervallo che parte dalla prima azione e dura `min(distanza, 60 secondi)`. Gap più lunghi non generano durata. Una tabella senza timestamp non riceve un timestamp inventato.
3. Recupera gli intervalli `active_time` validi: ogni heartbeat copre al massimo **15 secondi**, usando `started_at` e `ended_at` quando disponibili e rispettando il limite dichiarato dal segnale.
4. Unisce gli intervalli ricostruiti con quelli misurati per la stessa identità. Sovrapposizioni, schede multiple e copie della stessa azione non sommano secondi duplicati. Non aggiunge durata prima della prima azione o dopo l'ultima; un heartbeat valido costituisce un intervallo esplicitamente osservato.
5. Somma le durate unite e arrotonda il totale ai secondi. Un'identità entra nel campione della media soltanto se possiede almeno un intervallo positivo. Un'unica azione senza heartbeat valido non genera una durata: quell'identità viene esclusa dal campione, anziché trattata come una visita di zero secondi.

Esempio: due azioni a 75 secondi di distanza contribuiscono con i primi 60 secondi. Un heartbeat di 10 secondi interamente contenuto in questi 60 non porta il totale a 70. Due azioni distanti dieci minuti non consentono invece di riempire il periodo intermedio. I timestamp operativi sono quelli conservati dalle tabelle, che possono descrivere il voto attuale o una ricevuta; per questo il risultato è una stima di attività osservabile, non una misura certificata di permanenza continua.

`details.time_estimate` contiene i seguenti metadati, mostrati insieme alla copertura nella modale e usati per dichiarare il campione del report:

| Campo | Significato |
| --- | --- |
| `method` | Testo del metodo applicato |
| `covered_visitors` | Identità con almeno un intervallo valido; coincide con `estimated_timed_visitors` quando la stima è disponibile |
| `total_visitors` | Identità osservabili complessive dell'evento, ospiti e profili riconciliati |
| `excluded_visitors` | `total_visitors - covered_visitors`; identità prive di durata ricostruibile |
| `gap_limit_seconds` | `300`: distanza massima fra due azioni consecutive utilizzabile per la stima |
| `interval_cap_seconds` | `60`: durata massima ricostruita per una coppia di azioni |
| `measured_seconds` | Secondi misurati dagli heartbeat, uniti per identità e arrotondati |
| `inferred_seconds` | Parte aggiunta dalla ricostruzione, al netto delle sovrapposizioni con i tempi misurati: totale stimato arrotondato meno `measured_seconds` |

Se nessuna identità ha un intervallo valido, `estimated_seconds` ed `estimated_timed_visitors` restano `null`, entrambe le KPI sono N/D e il campione escluso resta dichiarato. La mancanza di durata non dimostra disinteresse né una permanenza nulla.

### Riepilogo operativo aggiuntivo

Oltre alle 30 KPI, `details.operational` espone i conteggi effettivamente ricostruibili: voti salvati, profili identificati con attività operativa, ricevute premio dei giochi ammessi, riscatti, monete ottenute e spese, risposte feedback, clic sponsor e clic sponsor privi di identità. Alcuni totali alimentano anche le KPI aggiuntive; il riepilogo conserva la distinzione fra dati operativi e campioni tracciati. Questi valori permettono di consultare dati utili anche quando manca il denominatore di una percentuale.

`details.coverage_notes` accompagna il report con i limiti delle fonti. I valori derivati dai segnali sono marcati `new_tracking`, quelli operativi `operational`, le combinazioni `mixed` e le stime temporali `estimated`; le modifiche approvate sono `manual` e i valori non ricostruibili `unavailable`. La provenienza indica come è stato ottenuto il dato, non certifica una copertura completa del pubblico.

## Revisione e versione approvata

L'apertura della modale prepara un nuovo calcolo con 30 KPI disponibili e le 20 originarie preselezionate. Ogni KPI dispone della casella **Includi nel PDF**; la modale permette anche di selezionarle tutte, selezionare soltanto quelle con dati o deselezionarle. Sono consentite **da 1 a 20 KPI nel one pager** e **da 1 a 30 nel report completo di tre pagine**. Una selezione troppo ampia richiede di ridurla o cambiare formato: non viene troncata automaticamente. Anche una KPI N/D può essere selezionata e, se necessario, verificata o modificata.

La richiesta di approvazione include `selected_kpi_ids`. Il backend rifiuta una selezione vuota, ID sconosciuti, duplicati e il superamento del limite del formato; poi ordina la selezione secondo l'ordine delle KPI del calcolo originale. Questa selezione canonica viene congelata nello snapshot e inclusa nell'hash dell'approvazione insieme a formato, fatti, modifiche, descrizioni e note. Cambiare soltanto l'ordine dei medesimi ID non crea una revisione diversa; cambiare gli ID selezionati cambia la revisione. Un retry della stessa approvazione restituisce la versione salvata.

Valori, modifiche e descrizioni delle KPI non selezionate restano archiviati nel report approvato; il PDF stampa soltanto la selezione. La modifica della selezione nella modale invalida l'approvazione corrente del form. Scaricare nuovamente una versione già approvata usa i dati, le descrizioni e la selezione congelati, senza rileggere i dati live.

Le bozze e le versioni precedenti con 20 KPI mantengono il proprio insieme originale: non ricevono dieci KPI sintetiche dal nuovo codice. Se `selected_kpi_ids` manca, vengono usati gli ID congelati nella bozza oppure, per le vecchie bozze prive di selezione, tutti gli indicatori originali. Anche un PDF storico senza selezione esplicita conserva tutti i suoi indicatori.

Le modifiche manuali vengono separate dai fatti originari; non è richiesto un motivo della correzione. Il salvataggio approvato conserva dati, provenienza, descrizioni, selezione e formato.

Ogni KPI dispone inoltre di una mini descrizione facoltativa e modificabile (`display_description`). Il testo può essere svuotato per ometterlo dal PDF e può contenere al massimo 120 caratteri. La descrizione sintetica di stampa è separata dalla descrizione tecnica della formula (`description`): modificarla non cambia i fatti o il calcolo. Le descrizioni approvate vengono conservate nella versione del report e partecipano all'identificazione della revisione, come i valori modificati. Nel PDF le correzioni non vengono segnalate con asterischi e non vengono stampati i riquadri esplicativi verdi a fondo pagina.

La query che recupera l'ultima versione approvata è:

```sql
SELECT payload
FROM engagement_report_snapshots
WHERE event_id = :event_id
ORDER BY approved_at DESC
LIMIT 1;
```

Il logo dell'emittente firma il PDF; il nome dell'emittente non viene stampato nella firma.

## Codice di riferimento

- `wcmvpvs-back/service/api/engagement_reports.go`: autenticazione dell'ambito, caricamento, bozza e approvazione.
- `wcmvpvs-back/service/api/engagement_report_metrics.go`: identità, deduplicazione, fatti e formule delle KPI.
- `wcmvpvs-back/service/api/engagement_report_sources.go`: query operative e loro normalizzazione.
- `wcmvpvs-back/service/api/engagement_report_combined.go`: unione delle fonti, provenienza e limiti dei denominatori.
- `wcmvpvs-back/service/api/engagement_report_time.go`: stima temporale, unione degli intervalli e copertura del campione.
- `wcmvpvs-back/service/api/engagement_report_selection.go`: selezione valida e ordine canonico delle KPI nel PDF.
- `wcmvpvs-front/src/eventReportTelemetry.js`: osservazioni inviate dalla home.
- `wcmvpvs-front/src/views/FanLiveCento.vue`: azioni e disponibilità effettivamente osservate nella pagina.

Le [query manuali post partita](new-fan-post-match.md) sono un'analisi distinta. Non rappresentano una query per ciascuna KPI del report e hanno limiti specifici per i dati raccolti prima dei segnali dedicati.

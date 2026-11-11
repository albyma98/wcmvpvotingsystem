# Sponsor Analytics su /:slug/new

## Ricognizione

`FanLiveCento.vue` carica il catalogo sponsor e le assegnazioni dall'API esistente. Le assegnazioni in `game_wall_sponsors` e `post_vote_sponsors` consentono di usare uno sponsor nascosto dalla home. I cartelloni SVG riutilizzano `GameWallSponsors.vue` e `SvgSponsorBoard.vue`.

| Superficie | Placement | Creatività e misurazione |
| --- | --- | --- |
| Match Day, sezione Partner ufficiali | `home_footer` | Sponsor attivi dal catalogo, misurati singolarmente. Clic su creatività con destinazione HTTP(S). |
| Conferma MVP | `post_vote_left`, `post_vote_right` | Logo o nome di fallback. Finestra di animazione esistente di 4,36 s. Full view con quattro secondi continuativi ad almeno il 50%. |
| Basket Rush, Playmaker, Crossover Rush | `game_wall_left`, `game_wall_right` | Due cartelloni per gioco, distinti da `game_id`. Campionamento delle coordinate SVG trasformate in pixel CSS, con hit test per coperture e strati SVG sovrapposti. |
| Modale Guadagna monete | `sponsor_rush_entry` | Titolo con il nome del primo sponsor attivo. L'apertura del gioco è un'azione della piattaforma. |
| Sponsor Rush | `sponsor_rush_logo` | Un'esposizione geometrica per logo in caduta realmente visibile. Raccolta, punteggio e schermata finale sono azioni di gioco. |

Gli stemmi club e squadra, le fotografie e le card dei giocatori, i marchi di ArenaBoostX, le illustrazioni di gioco, la testata del merchandising e le immagini prodotto sono contenuti istituzionali o di prodotto. L'audit di roster, risultato del voto, riepilogo partita, wallet, collezione, profilo, classifiche, feedback, chat, registrazione, shop e schermate degli altri giochi non ha trovato ulteriori creatività sponsor configurate. I componenti Branded Game, la pagina club e lo SponsorStrip del torneo appartengono a flussi distinti. Nella superficie analizzata non è presente una rotazione automatica. Un cambio dell'identità della creatività sullo stesso nodo termina l'esposizione precedente.

## Fonti e compatibilità

- `sponsor_sessions`, `sponsor_exposures`, `sponsor_clicks` e gli endpoint storici servono ancora la vecchia esperienza e l'admin. La vecchia viewability di sezione e i clic storici hanno definizioni e granularità diverse. Le righe restano consultabili nello storico e in **Risultati → Impatto degli sponsor**.
- `event_report_signals` continua a raccogliere telemetria operativa di `/new`. Prima del nuovo motore registrava il footer come `home_partner`, il titolo Sponsor Rush come `sponsored_game_view` e, per alcuni clic, anche una richiesta separata a `sponsor_clicks`. Le nuove esposizioni e i clic commerciali transitano attraverso il solo batch sponsor. Gli eventi delle partite ai minigiochi mantengono la propria funzione operativa.
- PostHog è già disattivato su `/new`. Le statistiche commerciali provengono da `sponsor_measured_exposures` e `sponsor_measured_event_ids` in SQLite.
- **Risultati → Sponsor Analytics** offre filtri, riepilogo, dettaglio, trend e aggregazione per la stagione dal 1° luglio al 30 giugno, in UTC. La stessa struttura aggregata è disponibile ai report marketing in `commercial_sponsor_analytics`. I PDF esistenti conservano le definizioni e le misurazioni storiche.

## Definizioni e copertura

- L'impression inizia al primo campione positivo di una creatività caricata, con scheda visibile. Per HTML, la viewability richiede almeno il 50% per un secondo continuo. Scroll, schermate sovrapposte e scheda nascosta interrompono la continuità. Il campionamento avviene ogni 125 ms e scarta intervalli con ritardi oltre 350 ms. Il totale dei tempi usa solo esposizioni chiuse e intervalli consecutivi sopra soglia. La media usa i tempi positivi di questo insieme. Una chiusura mancante lascia l'impression e le qualificazioni già ricevute.
- La viewability rate usa le impression HTML misurabili. Le impression geometriche sono disponibili per SVG e oggetti di gioco e hanno viewability e full-view rate N/D. I test geometrici stimano l'area con una griglia e il browser hit test. L'occlusione da elementi estranei al flusso, dipinti con `pointer-events: none`, e gli effetti di compositing complessi possono sfuggire al test. [Linee guida IAB/MRC per l'intrinsic in-game](https://www.mediaratingcouncil.org/sites/default/files/Standards/In-Game%20Advertising%20Measurement%20Guidelines%2008-31-22.pdf)
- Reach stimata e clic unici usano l'identificativo tecnico già presente nel browser, con HMAC per società lato server. Il browser non crea un identificativo per questa funzione. La reach è distinta sull'intero intervallo e l'aggregato non somma i valori dei singoli giorni o posizionamenti. Browser, dispositivi e origini differenti possono rappresentare la stessa persona. Se una parte delle esposizioni non ha un identificativo riutilizzabile, la reach dell'aggregato è N/D. La frequenza usa viewable / reach, il CTR usa clic / viewable. Full view usa la coorte MVP con esito ricevuto. Le conversioni sono N/D.
- Il catalogo fornisce una firma per società, partita, sponsor, creatività e placement. La firma vale 24 ore e permette di completare una misura iniziata con la creatività precedente. Il servizio usa il segreto di voto già configurato, con chiavi logiche separate per firma e HMAC di reach. Con il segreto di esempio i banner rimangono disponibili e la raccolta sponsor è disattivata. La rotazione del segreto interrompe la deduplicazione della reach tra i due periodi.
- L'identità della creatività deriva dal logo nel catalogo o dal nome nel placement testuale. Un'immagine remota sovrascritta allo stesso URL mantiene l'identità precedente. I filtri attribuiscono gli eventi all'ora di inizio dell'esposizione. Per un clic senza impression usano l'ora del clic.
- Negli eventi di chiusura, `visibility_percentage` è il picco stimato dell'esposizione. Negli altri eventi rappresenta il campione disponibile in quel momento.
- Il client accoda al massimo 480 eventi in memoria. Invia batch fino a 24 ogni cinque secondi o alla chiusura, con fetch keepalive e retry. Un browser terminato bruscamente o offline fino alla chiusura può perdere il batch residuo. Il server limita per browser, IP, società e istanza, valida la sequenza e accetta ogni ID una sola volta. Le misure dipendono dalla telemetria del browser e non costituiscono una verifica antifrode indipendente.

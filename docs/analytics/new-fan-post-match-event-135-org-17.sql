-- ArenaBoostX: 10 query SQLite per il report post partita di /:slug/new.
-- Schema di riferimento: ultima versione del backend nel repository.
-- Ogni query e' indipendente ed esegue solo letture.
-- Evento: 135. Organizzazione: 17. Nessuno slug richiesto.
-- La query 02 richiede anche :from_utc e :to_utc (estremo finale escluso).
-- Le tabelle condivise NON permettono di distinguere sempre /new dalle altre UI.
-- Leggere new-fan-post-match.md per significato e limiti dei risultati.

-- 01. RIEPILOGO EVENTO E PARTECIPANTI OSSERVABILI
-- Un partecipante e' un profilo, oppure un dispositivo ancora ospite.
-- Non include chi ha solamente visitato la home, letto le istruzioni o aperto il profilo.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id
    FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), raw_actors AS (
    SELECT v.user_id AS fan_id, v.device_id
    FROM votes v JOIN scope s ON s.event_id = v.event_id
    UNION ALL
    SELECT r.fan_id, r.device_id FROM fan_minigame_rewards r
    JOIN scope s ON s.event_id = r.event_id AND s.organization_id = r.organization_id
    UNION ALL
    SELECT r.fan_id, r.device_id FROM fan_journey_rewards r
    JOIN scope s ON s.event_id = r.event_id AND s.organization_id = r.organization_id
    UNION ALL
    SELECT b.fan_id, b.device_id FROM basketball_shots b
    JOIN scope s ON s.event_id = b.event_id AND s.organization_id = b.organization_id
    UNION ALL
    SELECT p.fan_id, p.device_id FROM playmaker_sessions p
    JOIN scope s ON s.event_id = p.event_id AND s.organization_id = p.organization_id
    UNION ALL
    SELECT c.fan_id, c.device_id FROM crossover_sessions c
    JOIN scope s ON s.event_id = c.event_id AND s.organization_id = c.organization_id
    UNION ALL
    SELECT m.fan_id, m.device_id FROM fan_wallet_movements m
    JOIN scope s ON s.event_id = m.event_id AND s.organization_id = m.organization_id
    WHERE m.source LIKE 'shop:%'
), actors AS (
    SELECT DISTINCT CASE WHEN fan_id > 0 THEN 'fan:' || fan_id
        WHEN NULLIF(TRIM(device_id), '') IS NOT NULL THEN 'guest:' || TRIM(device_id)
        END AS actor
    FROM raw_actors
)
SELECT s.event_id, o.name AS club, e.start_datetime AS inizio_evento,
       e.is_concluded AS evento_concluso,
       (SELECT COUNT(actor) FROM actors) AS partecipanti_osservabili,
       (SELECT COUNT(*) FROM actors WHERE actor LIKE 'fan:%') AS profili_con_attivita,
       (SELECT COUNT(*) FROM actors WHERE actor LIKE 'guest:%') AS dispositivi_ancora_ospiti,
       (SELECT COUNT(*) FROM votes v WHERE v.event_id = s.event_id) AS voti_mvp,
       (SELECT COUNT(*) FROM event_feedback f WHERE f.event_id = s.event_id) AS feedback_ricevuti,
       (SELECT COUNT(*) FROM sponsor_clicks c WHERE c.event_id = s.event_id) AS click_sponsor
FROM scope s JOIN events e ON e.id = s.event_id
JOIN organizations o ON o.id = s.organization_id;

-- 02. NUOVI PROFILI CREATI E COMPLETATI NEL PERIODO, PER ORA UTC
-- La registrazione non conserva l'event_id: attribuzione al club e al periodo.
-- Si usa la prima ricevuta di accettazione, senza contare ogni versione legale.
-- Sono esclusi i profili incompleti e quelli creati prima del periodo scelto.
WITH scope AS (
    SELECT e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), first_acceptance AS (
    SELECT a.fan_id, MIN(julianday(a.accepted_at)) AS accepted_jd
    FROM fan_registration_acceptances a
    JOIN scope s ON s.organization_id = a.organization_id
    GROUP BY a.fan_id
), completed AS (
    SELECT p.id, a.accepted_jd
    FROM fan_profiles p JOIN scope s ON s.organization_id = p.organization_id
    JOIN first_acceptance a ON a.fan_id = p.id
    WHERE p.registration_pending = 0 AND p.accepted_terms = 1
      AND julianday(p.created_at) >= julianday(:from_utc)
      AND julianday(p.created_at) < julianday(:to_utc)
      AND a.accepted_jd >= julianday(:from_utc)
      AND a.accepted_jd < julianday(:to_utc)
)
SELECT strftime('%Y-%m-%d %H:00:00', accepted_jd) AS ora_utc,
       COUNT(*) AS nuovi_profili_completati,
       SUM(COUNT(*)) OVER () AS totale_nel_periodo
FROM completed
GROUP BY ora_utc ORDER BY ora_utc;

-- 03. RISULTATI MVP, QUOTE E VOTI ASSOCIATI A PROFILI/OSPITI
-- La tabella conserva la scelta attuale, non la cronologia dei cambi di voto.
-- created_at viene aggiornato quando si cambia voto: non e' sempre il primo voto.
-- user_id puo' essere valorizzato anche dopo il voto, quando l'ospite si registra.
WITH scope AS (
    SELECT e.id AS event_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), results AS (
    SELECT v.player_id, COUNT(*) AS voti,
           SUM(CASE WHEN v.user_id > 0 THEN 1 ELSE 0 END) AS voti_associati_a_profili,
           SUM(CASE WHEN COALESCE(v.user_id, 0) = 0 THEN 1 ELSE 0 END) AS voti_ancora_ospiti,
           MIN(datetime(v.created_at)) AS min_data_voto_conservata_utc,
           MAX(datetime(v.created_at)) AS max_data_voto_conservata_utc
    FROM votes v JOIN scope s ON s.event_id = v.event_id
    GROUP BY v.player_id
)
SELECT DENSE_RANK() OVER (ORDER BY r.voti DESC) AS posizione_mvp,
       TRIM(p.first_name || ' ' || p.last_name) AS giocatore,
       r.voti, ROUND(100.0 * r.voti / SUM(r.voti) OVER (), 2) AS quota_voti_percento,
       r.voti_associati_a_profili, r.voti_ancora_ospiti,
       r.min_data_voto_conservata_utc, r.max_data_voto_conservata_utc
FROM results r JOIN players p ON p.id = r.player_id
ORDER BY r.voti DESC, r.player_id;

-- 04. I CINQUE MINIGIOCHI A PREMIO FISSO: PREMI, PARTECIPANTI E RIPETIZIONI
-- Conta i premi confermati, non tutte le partite avviate o completate senza ritiro.
-- Il premio legacy vale al massimo una volta per partecipante e gioco.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), games(game_key, gioco) AS (
    VALUES ('reaction', 'Reaction Test'), ('quiz', 'Quick Quiz'),
           ('tap', 'Tap Challenge'), ('sponsor-rush', 'Sponsor Rush'),
           ('memory-flash', 'Memory Flash')
), raw_rewards AS (
    SELECT r.reward_key,
           CASE WHEN r.fan_id > 0 THEN 'fan:' || r.fan_id
                WHEN NULLIF(TRIM(r.device_id), '') IS NOT NULL THEN 'guest:' || TRIM(r.device_id) END AS actor,
           r.coins, 1 AS premio_nuovo, 0 AS premio_legacy
    FROM fan_minigame_rewards r
    JOIN scope s ON s.event_id = r.event_id AND s.organization_id = r.organization_id
    WHERE r.coins > 0
), legacy AS (
    SELECT r.reward_key,
           CASE WHEN r.fan_id > 0 THEN 'fan:' || r.fan_id
                WHEN NULLIF(TRIM(r.device_id), '') IS NOT NULL THEN 'guest:' || TRIM(r.device_id) END AS actor,
           MAX(r.coins) AS coins, 0 AS premio_nuovo, 1 AS premio_legacy
    FROM fan_journey_rewards r
    JOIN scope s ON s.event_id = r.event_id AND s.organization_id = r.organization_id
    WHERE r.coins > 0
    GROUP BY r.reward_key, actor
), rewards AS (
    SELECT * FROM raw_rewards UNION ALL SELECT * FROM legacy
)
SELECT g.gioco, COUNT(DISTINCT r.actor) AS partecipanti_premiati,
       COALESCE(SUM(r.premio_nuovo), 0) AS giocate_premiate_nuovo_flusso,
       COALESCE(SUM(r.premio_legacy), 0) AS premi_legacy,
       COALESCE(SUM(r.coins), 0) AS monete_accreditate,
       ROUND(1.0 * SUM(r.premio_nuovo + r.premio_legacy) /
             NULLIF(COUNT(DISTINCT r.actor), 0), 2) AS premi_per_partecipante,
       MAX(0, COALESCE(SUM(r.premio_nuovo + r.premio_legacy), 0) -
              COUNT(DISTINCT r.actor)) AS premi_oltre_il_primo
FROM games g LEFT JOIN rewards r ON r.reward_key = g.game_key
GROUP BY g.game_key, g.gioco ORDER BY monete_accreditate DESC, g.gioco;

-- 05. BASKET, PLAYMAKER E CROSSOVER: ESITI VALIDATI DAL SERVER
-- Basket: i ticket emessi possono includere un tiro mai effettuato.
-- Ticket pending/expired non sono una misura certa degli abbandoni.
-- Playmaker/Crossover: una sessione creata non prova che sia stata completata.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), games(gioco) AS (
    VALUES ('Basketball Shootout'), ('Playmaker'), ('Crossover Rush')
), actions AS (
    SELECT 'Basketball Shootout' AS gioco,
           CASE WHEN b.status = 'made' THEN 1 WHEN b.status = 'miss' THEN 0 END AS successo,
           CASE WHEN b.status IN ('made', 'miss') THEN b.clean ELSE 0 END AS clean
    FROM basketball_shots b
    JOIN scope s ON s.event_id = b.event_id AND s.organization_id = b.organization_id
    UNION ALL
    SELECT 'Playmaker', p.assisted, p.clean FROM playmaker_passes p
    JOIN scope s ON s.event_id = p.event_id AND s.organization_id = p.organization_id
    UNION ALL
    SELECT 'Crossover Rush', c.won, c.clean FROM crossover_duels c
    JOIN scope s ON s.event_id = c.event_id AND s.organization_id = c.organization_id
), raw_participants AS (
    SELECT 'Basketball Shootout' AS gioco, b.fan_id, b.device_id, NULL AS session_id
    FROM basketball_shots b
    JOIN scope s ON s.event_id = b.event_id AND s.organization_id = b.organization_id
    UNION ALL
    SELECT 'Playmaker', p.fan_id, p.device_id, p.session_id FROM playmaker_sessions p
    JOIN scope s ON s.event_id = p.event_id AND s.organization_id = p.organization_id
    UNION ALL
    SELECT 'Crossover Rush', c.fan_id, c.device_id, c.session_id FROM crossover_sessions c
    JOIN scope s ON s.event_id = c.event_id AND s.organization_id = c.organization_id
), participants AS (
    SELECT gioco,
           COUNT(DISTINCT CASE WHEN fan_id > 0 THEN 'fan:' || fan_id
               WHEN NULLIF(TRIM(device_id), '') IS NOT NULL THEN 'guest:' || TRIM(device_id) END) AS partecipanti,
           COUNT(DISTINCT session_id) AS sessioni_create
    FROM raw_participants GROUP BY gioco
), outcomes AS (
    SELECT gioco, COUNT(*) AS record_tentativi, COUNT(successo) AS tentativi_con_esito,
           COALESCE(SUM(successo), 0) AS successi,
           SUM(CASE WHEN successo = 1 AND clean = 1 THEN 1 ELSE 0 END) AS successi_clean
    FROM actions GROUP BY gioco
)
SELECT g.gioco, COALESCE(p.partecipanti, 0) AS partecipanti,
       CASE WHEN g.gioco = 'Basketball Shootout' THEN NULL ELSE COALESCE(p.sessioni_create, 0) END AS sessioni_create,
       COALESCE(a.record_tentativi, 0) AS record_tentativi,
       COALESCE(a.tentativi_con_esito, 0) AS tentativi_con_esito,
       COALESCE(a.record_tentativi - a.tentativi_con_esito, 0) AS ticket_senza_esito,
       COALESCE(a.successi, 0) AS canestri_assist_o_duelli_vinti,
       ROUND(100.0 * a.successi / NULLIF(a.tentativi_con_esito, 0), 2) AS successo_percento,
       COALESCE(a.successi_clean, 0) AS successi_clean,
       COALESCE(a.successi, 0) AS monete_accreditate
FROM games g LEFT JOIN participants p ON p.gioco = g.gioco
LEFT JOIN outcomes a ON a.gioco = g.gioco ORDER BY g.gioco;

-- 06. MONETE ACCREDITATE E SPESE NELL'EVENTO, PER CAUSALE
-- Il ledger e' la fonte monetaria: non sommare nuovamente le ricevute dei giochi.
-- Saldi iniziali e trasferimenti non sono nuovi guadagni.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
)
SELECT m.source AS causale, COUNT(*) AS movimenti,
       COUNT(DISTINCT m.owner_key) AS partecipanti,
       SUM(CASE WHEN m.delta > 0 THEN m.delta ELSE 0 END) AS monete_accreditate,
       SUM(CASE WHEN m.delta < 0 THEN -m.delta ELSE 0 END) AS monete_spese,
       SUM(m.delta) AS variazione_netta,
       MIN(datetime(m.created_at)) AS primo_movimento_utc,
       MAX(datetime(m.created_at)) AS ultimo_movimento_utc
FROM fan_wallet_movements m
JOIN scope s ON s.event_id = m.event_id AND s.organization_id = m.organization_id
WHERE m.source IN ('basketball', 'playmaker', 'crossover')
   OR m.source LIKE 'journey:%' OR m.source LIKE 'minigame:%' OR m.source LIKE 'shop:%'
GROUP BY m.source ORDER BY monete_accreditate DESC, monete_spese DESC;

-- 07. SHOP TIFOSO: ACQUISTI E MONETE SPESE PER OGGETTO
-- La cornice ha scope_event_id=0 nell'inventario: per attribuire la spesa
-- alla partita si usa il movimento con event_id, non il possesso della cornice.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), catalog(source, oggetto) AS (
    VALUES ('shop:club_frame', 'Cornice della societa'),
           ('shop:match_card', 'Card ricordo della partita')
), purchases AS (
    SELECT m.* FROM fan_wallet_movements m
    JOIN scope s ON s.event_id = m.event_id AND s.organization_id = m.organization_id
    WHERE m.delta < 0 AND m.source IN ('shop:club_frame', 'shop:match_card')
)
SELECT c.oggetto, COUNT(p.id) AS operazioni_di_acquisto,
       COUNT(DISTINCT p.owner_key) AS acquirenti_distinti,
       COALESCE(SUM(-p.delta), 0) AS monete_spese,
       ROUND(AVG(-p.delta), 2) AS costo_medio_in_monete,
       MIN(datetime(p.created_at)) AS primo_acquisto_utc,
       MAX(datetime(p.created_at)) AS ultimo_acquisto_utc
FROM catalog c LEFT JOIN purchases p ON p.source = c.source
GROUP BY c.source, c.oggetto ORDER BY monete_spese DESC, c.oggetto;

-- 08. FEEDBACK: DISTRIBUZIONE DELLE RISPOSTE E PRESENZA DI SUGGERIMENTI
-- Le chiavi delle domande vanno lette insieme alla configurazione del sondaggio.
-- Nessuna esportazione dei suggerimenti liberi o attribuzione a persone.
WITH scope AS (
    SELECT e.id AS event_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), feedback AS (
    SELECT f.* FROM event_feedback f JOIN scope s ON s.event_id = f.event_id
), answers AS (
    SELECT 'experience' AS domanda, experience AS risposta FROM feedback
    UNION ALL SELECT 'team_spirit', team_spirit FROM feedback
    UNION ALL SELECT 'perks_interest', perks_interest FROM feedback
    UNION ALL SELECT 'mini_games_interest', mini_games_interest FROM feedback
    UNION ALL SELECT 'suggestion', CASE WHEN NULLIF(TRIM(suggestion), '') IS NULL
        THEN 'assente' ELSE 'presente' END FROM feedback
)
SELECT domanda, risposta, COUNT(*) AS risposte,
       SUM(COUNT(*)) OVER (PARTITION BY domanda) AS totale_risposte_domanda,
       ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (PARTITION BY domanda), 2) AS quota_percento
FROM answers GROUP BY domanda, risposta ORDER BY domanda, risposte DESC, risposta;

-- 09. SPONSOR: CLICK, DISPOSITIVI IDENTIFICABILI E QUOTA DEI CLICK
-- quota_click_percento NON e' il CTR: /new non invia le impression dei banner.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), results AS (
    SELECT sp.id, COALESCE(NULLIF(TRIM(sp.report_name), ''), sp.name) AS sponsor,
           COUNT(c.id) AS click_totali,
           COUNT(DISTINCT NULLIF(TRIM(c.device_id), '')) AS dispositivi_identificabili,
           SUM(CASE WHEN c.id IS NOT NULL AND NULLIF(TRIM(c.device_id), '') IS NULL THEN 1 ELSE 0 END) AS click_senza_device,
           MIN(datetime(c.clicked_at)) AS primo_click_utc,
           MAX(datetime(c.clicked_at)) AS ultimo_click_utc
    FROM scope s JOIN sponsors sp ON sp.organization_id = s.organization_id
    LEFT JOIN sponsor_clicks c ON c.sponsor_id = sp.id AND c.event_id = s.event_id
    GROUP BY sp.id, sp.report_name, sp.name
)
SELECT sponsor, click_totali, dispositivi_identificabili, click_senza_device,
       ROUND(100.0 * click_totali / NULLIF(SUM(click_totali) OVER (), 0), 2) AS quota_click_percento,
       primo_click_utc, ultimo_click_utc
FROM results ORDER BY click_totali DESC, id;

-- 10. CLASSIFICA DEL CLUB, COLLEZIONI E BADGE DELLA PARTITA
-- Foto ATTUALE del club: non esiste uno snapshot automatico alla chiusura.
-- Il calcolo badge riproduce le fonti del profilo, incluse eventuali attivita
-- condivise Tap Live / branded, che non vengono analizzate come giochi di /new.
WITH scope AS (
    SELECT e.id AS event_id, e.organization_id FROM events e
    WHERE e.id = 135 AND e.organization_id = 17
), reward_parts AS (
    SELECT r.fan_id, MAX(r.coins) AS coins FROM fan_journey_rewards r
    JOIN scope s ON s.event_id = r.event_id AND s.organization_id = r.organization_id
    WHERE r.fan_id > 0 GROUP BY r.fan_id, r.reward_key
    UNION ALL
    SELECT r.fan_id, SUM(r.coins) FROM fan_minigame_rewards r
    JOIN scope s ON s.event_id = r.event_id AND s.organization_id = r.organization_id
    WHERE r.fan_id > 0 GROUP BY r.fan_id
    UNION ALL
    SELECT b.fan_id, COUNT(*) FROM basketball_shots b
    JOIN scope s ON s.event_id = b.event_id AND s.organization_id = b.organization_id
    WHERE b.fan_id > 0 AND b.status = 'made' GROUP BY b.fan_id
    UNION ALL
    SELECT p.fan_id, COUNT(*) FROM playmaker_passes p
    JOIN scope s ON s.event_id = p.event_id AND s.organization_id = p.organization_id
    WHERE p.fan_id > 0 AND p.assisted = 1 GROUP BY p.fan_id
    UNION ALL
    SELECT c.fan_id, COUNT(*) FROM crossover_duels c
    JOIN scope s ON s.event_id = c.event_id AND s.organization_id = c.organization_id
    WHERE c.fan_id > 0 AND c.won = 1 GROUP BY c.fan_id
    UNION ALL
    SELECT t.fan1_id, t.fan1_coins FROM tap_live_matches t
    JOIN scope s ON s.event_id = t.event_id AND s.organization_id = t.organization_id
    WHERE t.status = 'finished' AND t.fan1_id > 0
    UNION ALL
    SELECT t.fan2_id, t.fan2_coins FROM tap_live_matches t
    JOIN scope s ON s.event_id = t.event_id AND s.organization_id = t.organization_id
    WHERE t.status = 'finished' AND t.fan2_id > 0
    UNION ALL
    SELECT b.user_id, b.rewarded_coins FROM branded_game_participations b
    JOIN scope s ON s.event_id = b.event_id
    WHERE b.completed = 1 AND b.user_id > 0
), event_rewards AS (
    SELECT fan_id, SUM(coins) AS coins FROM reward_parts GROUP BY fan_id
), inventory AS (
    SELECT i.fan_id,
           SUM(CASE WHEN i.kind = 'match_card' THEN 1 ELSE 0 END) AS card_nella_collezione_club,
           MAX(CASE WHEN i.item_key = 'match_card' AND i.scope_event_id = s.event_id THEN 1 ELSE 0 END) AS possiede_card_evento
    FROM fan_shop_items i JOIN scope s ON s.organization_id = i.organization_id
    WHERE i.fan_id > 0 GROUP BY i.fan_id
), ranked AS (
    SELECT ROW_NUMBER() OVER (ORDER BY COALESCE(a.points, w.coins) DESC, p.id ASC) AS posizione_club,
           p.id AS fan_id, p.nickname, COALESCE(a.points, w.coins) AS punti_club_attuali,
           w.coins AS saldo_monete_attuale,
           COALESCE(r.coins, 0) AS punti_badge_evento,
           CASE WHEN COALESCE(r.coins, 0) >= 25 THEN 1 ELSE 0 END AS badge_tifoso_attivo_evento,
           CASE WHEN sp.equipped_frame = 'club_frame' THEN 1 ELSE 0 END AS cornice_indossata_ora,
           COALESCE(i.card_nella_collezione_club, 0) AS card_nella_collezione_club,
           COALESCE(i.possiede_card_evento, 0) AS possiede_card_evento
    FROM scope s JOIN fan_profiles p ON p.organization_id = s.organization_id
    JOIN fan_wallets w ON w.fan_id = p.id
    LEFT JOIN fan_economy_accounts a ON a.organization_id = s.organization_id AND a.owner_key = 'fan:' || p.id
    LEFT JOIN fan_shop_profiles sp ON sp.organization_id = s.organization_id AND sp.holder_key = 'fan:' || p.id
    LEFT JOIN event_rewards r ON r.fan_id = p.id
    LEFT JOIN inventory i ON i.fan_id = p.id
)
SELECT posizione_club, nickname, punti_club_attuali, saldo_monete_attuale,
       punti_badge_evento, badge_tifoso_attivo_evento, cornice_indossata_ora,
       card_nella_collezione_club, possiede_card_evento
FROM ranked ORDER BY posizione_club;

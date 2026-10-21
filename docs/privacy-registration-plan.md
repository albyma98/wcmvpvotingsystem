# ArenaBoostX — Piano privacy della registrazione tifosi

Data: 21 ottobre 2026.

Documento di progettazione basato sulle risposte di Alberto Marra. Le decisioni
confermate sono distinte dalle raccomandazioni e dai punti da completare. Questo
documento non è l'informativa da pubblicare e non modifica il funzionamento
dell'applicazione.

La [specifica delle spunte](privacy-registration-checkboxes.md) definisce testi,
comportamenti e dati da registrare per la futura implementazione.

## 1. Decisioni confermate

| Ambito | Decisione |
| --- | --- |
| Gestore | Alberto Marra, persona fisica; piattaforma ArenaBoostX |
| Contatto privacy | alberto@arenaboostx.com |
| Esperienza interessata | Registrazione e utilizzo tifosi di `/:slug/new` |
| Account | Profili separati per club, gestiti dalla piattaforma |
| Dati inseriti dal tifoso | Nickname e numero di telefono |
| Verifica del numero | SMS OTP tramite Twilio per registrazione e accesso |
| Ospiti | Voto MVP, minigiochi e feedback disponibili senza account |
| Dati di servizio | Monete, progressi, ultimo voto e dati dei giochi necessari al funzionamento |
| Classifica | Nickname e punteggi visibili; telefono mai pubblico |
| Marketing | Nessun invio promozionale, nessuna profilazione commerciale |
| Sponsor | Nessun accesso ai dati dei tifosi e nessuna esportazione |
| WhatsApp della piattaforma | Non previsto |
| Foto e selfie | Non previsti nella nuova esperienza |
| Pagamenti | Nessun acquisto con denaro; eventuali utilizzi di monete per elementi virtuali estetici |
| Premi fisici | Esempi: maglia, bevanda e altro gadget; selezione casuale o per classifica |
| Partecipazione ai gadget | Automatica con il profilo tifoso, secondo il regolamento; nessuna spunta facoltativa dedicata |
| Consegna dei premi | Alberto trasmette i tre numeri selezionati a un referente dello staff del club, che contatta i selezionati |
| Infrastruttura dichiarata | AWS in Francia; Twilio per SMS; PostHog da configurare per sole statistiche aggregate |
| Durata del profilo | Tra più stagioni, fino alla cancellazione richiesta o a 24 mesi dall'ultimo accesso autenticato |
| Cancellazione | Rimozione del profilo e dei relativi riferimenti personali da classifiche e premi; tempi per log e backup da definire |
| Utenti precedenti | Nessun account reale da migrare |

Il modello descritto indica Alberto Marra come possibile titolare del trattamento
degli account, in quanto ne decide finalità e mezzi. Il ruolo del club per la
selezione e la consegna dei premi va qualificato in base alle attività effettive;
la sola assenza di contratti non determina i ruoli GDPR.

## 2. Raccomandazione sull'età

Proposta da adottare nelle specifiche, non ancora confermata dall'utente:

- registrazione disponibile dai 14 anni;
- coinvolgimento del genitore o tutore per l'assegnazione e il ritiro dei premi
  fisici a minorenni, con modalità definite nel regolamento;
- informativa e spiegazioni comprensibili anche a utenti di 14–17 anni;
- accertamento dell'età proporzionato ai rischi, evitando di raccogliere per
  impostazione predefinita documenti o date di nascita complete;
- la sola dichiarazione dell'età non va presentata come verifica certa;
- la verifica OTP attesta l'accesso al numero, non l'età o l'identità anagrafica.

In Italia i 14 anni sono la soglia per prestare autonomamente il consenso al
trattamento nell'offerta diretta di servizi della società dell'informazione
quando il trattamento si fonda sul consenso. Non costituiscono una regola
generale sulla capacità contrattuale o sulla partecipazione ai premi. Scegliere
16 anni lascerebbe comunque utenti minorenni e non eliminerebbe queste esigenze.

La procedura proporzionata per accertare l'età e le condizioni applicabili ai
minorenni vanno definite prima di pubblicare i testi definitivi.

## 3. Documenti e basi giuridiche

Preparare:

1. informativa privacy di ArenaBoostX per tifosi registrati e ospiti;
2. termini del servizio, con profilo separato per club, nickname pubblico,
   progressi, monete virtuali, età e cancellazione;
3. informativa su cookie e altre memorie del dispositivo, aderente alla
   configurazione effettiva;
4. regole delle iniziative con premi fisici e relativa spiegazione dei dati
   comunicati al club;
5. accordo con il club che descriva ruoli, accessi e uso dei soli contatti
   selezionati; eventuali atti ex art. 28 o art. 26 solo se coerenti con i ruoli;
6. verifica degli accordi e delle condizioni di trattamento con i fornitori.

Non utilizzare un unico "consenso GDPR" per tutte le operazioni.

| Trattamento | Impostazione proposta, da finalizzare |
| --- | --- |
| Account, autenticazione e progressi | Valutare l'esecuzione del servizio richiesto, nei limiti di effettiva necessità e validità dei termini, anche per minorenni |
| Antifrode e sicurezza | Valutare e documentare il legittimo interesse, necessità e bilanciamento, considerando anche i minori |
| Classifica con nickname | Descrivere chiaramente il funzionamento e qualificare la base coerentemente con la funzione richiesta |
| Selezione e consegna dei gadget | Partecipazione inclusa nel profilo. Valutare l'esecuzione del servizio per i trattamenti oggettivamente necessari, senza dedurre la necessità dalla sola clausola contrattuale; verificare in particolare il passaggio del numero al club e le alternative meno invasive |
| Marketing e profilazione commerciale | Non previsti; l'accettazione dei termini non li autorizza |
| Analisi statistiche | Distinguere dati di servizio identificabili e statistiche rese effettivamente anonime; definire separatamente le operazioni necessarie per produrle |

L'informativa va resa prima della raccolta. L'eventuale registrazione della sua
versione o presa visione non costituisce consenso generale al trattamento.

## 4. Flusso di registrazione proposto

1. Presentare l'età minima e applicare la procedura di verifica definita.
2. Raccogliere il nickname, spiegando che può apparire nelle classifiche del club.
3. Mostrare, prima del numero, una breve spiegazione degli usi effettivi e i link
   all'informativa e ai termini.
4. Raccogliere il numero di telefono.
5. Richiedere «Accetto i Termini del servizio» con casella non preselezionata.
   Le regole dei gadget e i dettagli della consegna sono incorporati in una
   sezione dei termini o in un loro allegato accessibile prima della registrazione.
6. Indicare i gadget fra le funzioni del profilo e mostrare un collegamento
   descrittivo all'informativa sull'uso del numero. Spostare il paragrafo
   esplicativo nei documenti collegati, senza una spunta o un blocco esteso
   dedicato ai gadget nel modulo.
7. Inviare l'OTP su richiesta del tifoso, verificarlo e completare la registrazione.
8. Rendere disponibili nel profilo informativa, termini, regole delle iniziative
   con gadget e richiesta di cancellazione.

Bozza di spiegazione nel [frammento dell'informativa](legal/uso-numero-e-gadget.md),
consultabile dal collegamento vicino al numero:

> Il profilo include automaticamente le selezioni dei gadget. Usiamo il numero
> per l'accesso via SMS e, solo se selezionato, lo comunichiamo al referente di
> [nome del club] per la consegna.

Esempio di accettazione contrattuale:

> Accetto i Termini del servizio.

Collegamento separato sempre visibile, senza ripetere il paragrafo nel modulo:

> Informativa privacy — SMS e contatto del club per i gadget

L'informativa può condividere la pagina o finestra dei termini, purché abbia
una sezione riconoscibile e sia raggiungibile direttamente dal proprio link.

Non inserire caselle per marketing, sponsor, WhatsApp o foto, perché queste
finalità non sono previste nella nuova esperienza. Il mezzo usato personalmente
dal referente del club per contattare il selezionato resta da descrivere
nell'accordo operativo, senza confonderlo con una funzione della piattaforma.

## 5. Passaggio dei contatti allo staff

La comunicazione dei numeri al club è un trattamento da descrivere, anche quando
avviene manualmente. Prima di finalizzare il flusso:

- identificare il club e il referente incaricato;
- definire chi organizza la selezione e chi decide come usare i dati per la
  consegna, così da qualificare correttamente i ruoli;
- fornire solo numero e riferimenti strettamente necessari al premio, senza
  accesso all'elenco completo dei tifosi o alle loro attività;
- limitare l'uso al contatto e alla consegna, senza riutilizzo promozionale;
- concordare protezione dei contatti, termine di conservazione dopo la consegna
  e gestione delle richieste di cancellazione;
- definire il coinvolgimento del genitore o tutore per i minorenni, se si adotta
  la raccomandazione sui premi;
- valutare se un codice di ritiro o un contatto avviato dall'utente possa ridurre
  in futuro la necessità di trasmettere numeri.

## 6. PostHog e dati di utilizzo

Obiettivo confermato: analisi aggregate, senza interesse a ricostruire il
comportamento del singolo tifoso.

Situazione del codice al momento della verifica:

- `src/main.js` inizializza l'integrazione PostHog;
- `src/lib/posthog.js` usa un identificatore persistente di dispositivo e
  `localStorage`, se è configurata la chiave;
- il codice non disabilita il session recording; l'effettiva registrazione
  dipende anche dalla configurazione del servizio;
- `src/eventTracking.ts` gestisce eventi con riferimenti a dispositivo,
  sessione e, quando disponibile, tifoso;
- queste proprietà non descrivono un sistema che invia soltanto conteggi anonimi.

Direzione tecnica proposta:

- mantenere nel sistema della piattaforma solo lo stato di servizio necessario:
  monete, progressi, ultimo voto e dati dei giochi;
- produrre sul server statistiche aggregate per club, evento, periodo e
  funzionalità;
- inviare a PostHog solo aggregati sufficientemente ampi, senza numero,
  nickname, fan ID, device ID, session ID o cronologia individuale;
- non usare il browser del tifoso per inviare questi aggregati e non abilitarvi
  session replay per la nuova esperienza;
- evitare granularità, gruppi molto piccoli o combinazioni che consentano di
  risalire al singolo;
- verificare se rimangono altri accessi o scritture nel dispositivo per scopi
  analitici prima di decidere l'eventuale necessità del consenso al tracciamento.

Aggregare una dashboard dopo aver raccolto eventi individuali non rende anonima
la raccolta. Anche un identificatore privo di nome può essere dato personale.
La valutazione sui cookie e altri strumenti di tracciamento riguarda anche gli
ospiti e dipende dalla configurazione effettiva, non dalla registrazione.

L'integrazione è condivisa con altre parti dell'applicazione: l'intervento
tecnico dovrà delimitare la nuova esperienza e valutare gli effetti sugli altri
flussi prima di modificare componenti globali.

## 7. Conservazione, diritti e registrazioni delle scelte

- Account: cancellazione richiesta dall'utente o dopo 24 mesi dall'ultimo accesso
  autenticato, con controllo automatico periodico.
- Stagioni: distinguere l'eventuale azzeramento di classifiche e progressi dalla
  cancellazione del profilo.
- Dati di utilizzo: evitare una cronologia identificabile indefinita quando
  basta lo stato corrente del servizio.
- Selezionati: definire un periodo limitato alla consegna e alle necessità
  documentate di gestione dell'iniziativa.
- Log di sicurezza, tentativi OTP, dati di prova delle scelte e backup: stabilire
  tempi separati, motivati e coerenti con le effettive impostazioni dei fornitori.
- Cancellazione: revocare le sessioni, rimuovere il profilo e i riferimenti
  personali nelle tabelle collegate, comprese classifiche e premi. Coordinare
  anche la rimozione presso i destinatari quando applicabile.
- Eventuali conservazioni residue per obblighi applicabili o difesa di diritti
  devono essere limitate e descritte; non promettere cancellazione istantanea di
  tutte le copie di backup se non è tecnicamente assicurata.
- Conservare le versioni dei testi presentati, le date delle azioni e le scelte
  effettive, distinguendo accettazione dei termini, inclusione automatica ai
  premi derivante dal profilo ed eventuali futuri consensi.
- Gestire le richieste tramite il profilo e `alberto@arenaboostx.com`, con
  verifica dell'identità proporzionata.

## 8. Allineamenti applicativi e punti da completare

Il processo di registrazione è ora implementato nel frontend e nel backend:
due dichiarazioni distinte, documenti versionati consultabili, prova della
verifica OTP, registrazione dell'accettazione e inclusione automatica ai gadget.
La [specifica delle spunte](privacy-registration-checkboxes.md) descrive lo stato
attuato e i file da modificare per aggiornare i testi.

L'implementazione del modulo non completa tutte le attività di questo piano:
pulizia per inattività, cancellazioni coordinate, esportazione di statistiche
aggregate e formalizzazione dei rapporti con i club rimangono attività separate.

Allineamenti applicati o ancora da completare secondo l'ambito sopra descritto:

- sostituire l'attuale testo generico di consenso per profilo e premi;
- impedire che `accepted_terms` abiliti messaggi promozionali;
- limitare gli accessi ai telefoni: gli endpoint marketing attuali sono
  disponibili agli amministratori ordinari autorizzati per l'organizzazione,
  mentre il modello desiderato riserva i contatti ad Alberto e, per i soli
  selezionati, al referente del club;
- mantenere l'attuale separazione dei profili per organizzazione;
- prevedere versioni dei documenti, registrazione delle scelte, cancellazione
  coordinata e processo di rimozione per inattività;
- adeguare PostHog all'obiettivo di aggregazione prima di descriverlo come tale.

Da completare per testi pubblicabili:

1. conferma della raccomandazione su età e coinvolgimento dei genitori;
2. nome e ruolo del club, referente, modalità di contatto e tempi di uso dei
   numeri dei selezionati;
3. qualificazione dell'iniziativa con premi, organizzatore, valore dei premi e
   regolamento: una maglia non è automaticamente un oggetto di minimo valore
   escluso dalla disciplina delle manifestazioni a premio;
4. tempi effettivi per log, backup e dati dei premi;
5. inventario dei servizi AWS, configurazioni Twilio/PostHog, accordi di
   trattamento e condizioni di eventuali trasferimenti internazionali;
6. verifica dei termini applicabili ai minorenni e delle basi giuridiche per
   ciascun trattamento, senza considerare questo piano una validazione legale.

## Riferimenti consultati

- [Garante — Minori: soglia italiana e distinzione dal diritto dei contratti](https://www.garanteprivacy.it/temi/minori)
- [EDPB — Statement 1/2025 on Age Assurance](https://www.edpb.europa.eu/our-work-tools/our-documents/statements/statement-12025-age-assurance_en)
- [EDPB — Consenso, sintesi di aprile 2026](https://www.edpb.europa.eu/system/files/2026-04/edpb-summary-consent_en.pdf)
- [EDPB — Linee guida sui ruoli di titolare e responsabile](https://www.edpb.europa.eu/documents/guideline/guidelines-072020-on-the-concepts-of-controller-and-processor-in-the-gdpr_en)
- [Garante — Principi fondamentali del trattamento](https://garanteprivacy.it/web/guest/home/principi-fondamentali-del-trattamento)
- [Garante — FAQ su cookie e strumenti di tracciamento](https://www.garanteprivacy.it/faq/cookie)
- [PostHog — Privacy controls](https://posthog.com/docs/privacy)
- [MIMIT — Manifestazioni escluse, art. 6](https://www.mimit.gov.it/index.php/it/component/content/article?id=2013741)
- [MIMIT — Concorsi a premio](https://www.mimit.gov.it/it/mercato-e-consumatori/concorrenza-e-commercio/manifestazioni-a-premio/concorsi-a-premio)
- [Twilio — Data Protection Addendum](https://www.twilio.com/en-us/legal/data-protection-addendum)
- [EDPB — Necessità contrattuale nei servizi online](https://www.edpb.europa.eu/system/files/documents/files/file1/edpb_guidelines-art_6-1-b-adopted_after_public_consultation_en.pdf)

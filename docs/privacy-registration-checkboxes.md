# ArenaBoostX — Spunte e testi della registrazione

Data: 21 ottobre 2026. Specifica di interfaccia per `/:slug/new`.
La soglia di 14 anni e il coinvolgimento del genitore per i gadget sono la
raccomandazione progettuale descritta nel piano privacy; non sono una
verifica dell'età o una validazione dei termini per minorenni.

## Testi e posizione

Il testo sull'uso del numero è raccolto nel frammento
[Uso del numero e partecipazione ai gadget](legal/uso-numero-e-gadget.md), da
includere nella sezione dedicata dell'informativa e richiamare nei termini.
Il paragrafo non viene ripetuto nel modulo. Il collegamento visibile è:

> Informativa privacy — SMS e contatto del club per i gadget

Rendere disponibili da questo punto i collegamenti all'informativa e ai
termini. Il testo completo dell'informativa deve essere
consultabile prima della raccolta del numero, anche all'inizio del percorso.

### Dichiarazione di età

- Testo: **Dichiaro di avere almeno 14 anni.**
- Obbligatoria per questo percorso di registrazione; inizialmente non selezionata.
- Natura: dichiarazione di ammissibilità, non consenso privacy.
- La dichiarazione non va chiamata «età verificata»; il controllo adeguato e
  proporzionato va definito nella procedura di accesso dei minorenni.
- Nessuna data di nascita o copia di documento raccolta per impostazione predefinita.

La soglia va presentata già all'inizio del percorso, prima delle verifiche
del nickname e dell'invio dell'OTP, non soltanto dopo aver raccolto i dati.

### Accettazione dei termini

- Testo: **Accetto i Termini del servizio.**
- «Termini del servizio» è un link al documento pertinente e versionato.
- Le regole dei gadget sono una sezione riconoscibile dei termini o un loro
  allegato accessibile dal medesimo documento prima della registrazione.
- Obbligatoria per creare l'account; inizialmente non selezionata.
- Natura: accettazione delle condizioni del servizio.
- Non abilita messaggi promozionali o trattamento per finalità ulteriori.

### Gadget inclusi nel profilo, descritti nei termini

- Nessuna casella o blocco esteso dedicato nel modulo. Alberto ha confermato che
  la partecipazione automatica è una funzione del profilo tifoso e ha chiesto
  di incorporare i dettagli nei termini.
- La partecipazione rimane indicata fra le funzioni del profilo. Per l'uso del
  numero, il collegamento descrittivo apre direttamente l'informativa che
  contiene la spiegazione, senza un paragrafo aggiuntivo nel modulo.
- Nella sezione «Gadget del club» dei termini, includere:

> Creando il profilo tifoso partecipi automaticamente alle selezioni dei gadget
> di **[nome del club]**, secondo il regolamento. Solo se vieni selezionato,
> ArenaBoostX comunicherà il tuo numero al referente del club per organizzare
> la consegna del gadget.

- Nella medesima sezione, la nota relativa ai minorenni, se adottata nel regolamento:

> Se hai meno di 18 anni, per la consegna sarà coinvolto un genitore o tutore.

- Non creare un consenso facoltativo preselezionato o trasformato in obbligatorio.
  La registrazione richiede il servizio del profilo, che include la partecipazione.
- Valutare l'esecuzione del servizio richiesto come base per i trattamenti
  oggettivamente necessari alla partecipazione e alla consegna. Inserire una
  clausola nei termini non rende da solo necessario qualsiasi trattamento:
  il passaggio del telefono al club richiede una valutazione specifica delle
  alternative meno invasive e del ruolo del club.
- Il numero è comunicato solo per la consegna ai selezionati, senza riuso per
  promozioni, sponsor o analisi individuali.
- Applicare l'inclusione automatica sia alle estrazioni casuali sia ai premi
  per classifica previsti dal regolamento.
- Definire nel regolamento le iniziative e i requisiti cui si riferisce la
  partecipazione del profilo, senza promettere l'assegnazione di un premio.
- Consentire al selezionato di rifiutare il singolo gadget senza perdere account,
  monete o progressi; non imporre di accettare materialmente un premio.
- La cancellazione del profilo interrompe le inclusioni future; gestire le
  eventuali consegne in corso secondo le regole pubblicate.
- Usare il nome effettivo del club del profilo; non lasciare un destinatario
  generico o un segnaposto nell'applicazione pubblicata.

## Informativa senza spunta aggiuntiva

Mostrare il link **Informativa privacy — SMS e contatto del club per i gadget**
fuori dalle etichette delle caselle. Non inserire un'accettazione obbligatoria dell'informativa come se
fosse il consenso a qualsiasi uso dei dati.

L'informativa può essere ospitata nella stessa pagina o finestra dei termini,
ma deve essere una sezione chiaramente riconoscibile, con accesso diretto dal
link «Informativa privacy». Non nasconderla in un testo contrattuale indistinto.

La presentazione della versione dell'informativa può essere registrata per
documentare il flusso; apertura del link e lettura completa non sono equivalenti.

Il collegamento descrittivo non sostituisce l'informativa completa: la sua
accessibilità e la trasparenza dell'intero percorso vanno verificate prima
della pubblicazione, anche in relazione agli utenti minorenni.

## Pulsanti e comportamento

- Pulsante principale: **Ricevi il codice SMS**.
- Abilitarlo con numero valido, dichiarazione di età e termini accettati.
- Sono presenti due sole caselle: dichiarazione di età e termini comprensivi
  delle regole dei gadget; nessuna scelta separata di partecipazione.
- Prima dell'invio, spiegare in modo testuale eventuali errori; non affidarsi
  soltanto al colore o al pulsante disabilitato.
- Il link ai documenti non deve selezionare o deselezionare una casella.
- Le scelte rimangono quando si apre e chiude un documento o si torna
  dall'OTP per modificare il numero.
- Cambio del club o nuova registrazione: tutte le spunte tornano vuote.
- Accesso a un account esistente: numero e OTP; non riproporre le spunte di
  nuova registrazione a ogni accesso. Eventuali variazioni dei termini hanno
  un flusso dedicato.
- Il tifoso può uscire e continuare come ospite; i diritti privacy riguardano
  anche gli ospiti.
- Rendere interamente cliccabili le etichette, usare checkbox native e
  mantenere accessibilità da tastiera e descrizioni associate ai campi.

## Dati da registrare e controlli del server

Nomi indicativi per il contratto applicativo:

| Campo | Significato |
| --- | --- |
| `age_eligibility_declared` | Dichiarazione della soglia richiesta, non età certificata |
| `minimum_age_declared` | Soglia della dichiarazione, inizialmente proposta a 14 |
| `terms_version` / `terms_accepted_at` | Versione delle condizioni effettivamente accettate e data |
| `privacy_notice_version` | Versione dell'informativa resa disponibile nel flusso |
| `gadget_rules_version` | Versione delle regole presentate per quella iniziativa |
| `gadget_enrolment_source` / `gadget_enrolled_at` | Inclusione automatica derivante dalla registrazione completata e data |
| `organization_id` | Club cui si riferiscono profilo e scelta |

Le date vanno prodotte dal server. I documenti devono avere versioni disponibili
e riconosciute dal server; non accettare versioni arbitrarie inviate dal browser.
Non memorizzare un campo «consenso marketing» come conseguenza dei termini.

Il server deve validare le dichiarazioni obbligatorie, la verifica del numero e
la corrispondenza del club e attivare l'inclusione automatica solo al completamento
dell'account. La partecipazione non deve dipendere da un flag facoltativo del
browser, da un consenso marketing o dalla semplice richiesta di OTP.
Un controllo nel solo browser non basta a garantire questo comportamento.

L'invio OTP precede il completamento dell'account: gestire separatamente i dati
temporanei del tentativo e la loro scadenza, senza conservare indefinitamente
registrazioni incomplete.

## Casi di accettazione della futura implementazione

1. Età o termini non dichiarati: non proseguire nel percorso di nuova registrazione.
2. Età e termini dichiarati, numero verificato e registrazione completata:
   account creato e inclusione automatica nelle selezioni previste per il club.
3. Numero non verificato o registrazione incompleta: nessuna inclusione.
4. Apertura dei documenti: le scelte non cambiano e la navigazione torna al modulo.
5. Rifiuto del gadget assegnato: nessuna perdita di monete o progressi.
6. Login successivo: nessuna nuova adesione o duplicazione dell'inclusione.
7. La dichiarazione di età e l'OTP non sono mostrati come verifica anagrafica.
8. Il club riceve solo i contatti dei selezionati necessari alla consegna.

## Stato della progettazione

È disponibile un'anteprima interattiva delle spunte con dati dimostrativi;
non invia SMS e non modifica account. Il flusso è ora implementato anche nel
componente reale `FanRegistrationPromptModal.vue`, utilizzato da `/:slug/new`.

I testi visualizzati sono `wcmvpvs-front/src/legal/fan-terms.md` e
`wcmvpvs-front/src/legal/fan-privacy.md`. La versione dell'intero fascicolo è
`2026-10-21.1`, condivisa fra `src/fanRegistration.js` e
`service/database/fan_registration.go`; modificarla su entrambi i lati quando
cambiano i documenti. Il campo `legal_version` identifica insieme termini,
informativa e sezione delle regole dei gadget.

L'API valida le dichiarazioni prima dell'invio OTP e del completamento del
profilo. La registrazione richiede una sessione ottenuta con OTP del medesimo
numero e club. La tabella `fan_registration_acceptances` conserva versione,
soglia dichiarata, data prodotta dal server e stato della verifica del numero.
La partecipazione all'evento viene registrata automaticamente senza un flag
facoltativo dal browser. I profili precedenti non ricevono accettazioni inventate.

I documenti sono raggiungibili anche dalla home /new dopo la registrazione.
PostHog nel browser non viene inizializzato su /new; non è stato implementato
in questo intervento un servizio alternativo di esportazione di aggregati.
I termini non abilitano gli SMS promozionali: il relativo flag è separato e
inizialmente falso, e i contatti completi sono riservati al ruolo master.

L'implementazione non costituisce una validazione legale: restano da definire
il ruolo del club, il regolamento concreto delle iniziative, la procedura
proporzionata di verifica dell'età e i tempi operativi di log e backup. La
pulizia automatica dopo 24 mesi e la gestione completa delle cancellazioni
non fanno parte di questo intervento sul processo di registrazione.

Riferimenti:

- [EDPB — Condizioni del consenso](https://www.edpb.europa.eu/system/files/2026-04/edpb-summary-consent_en.pdf)
- [Garante — Minori](https://www.garanteprivacy.it/temi/minori)
- [EDPB — Necessità contrattuale nei servizi online](https://www.edpb.europa.eu/system/files/documents/files/file1/edpb_guidelines-art_6-1-b-adopted_after_public_consultation_en.pdf)
- [EDPB — Trasparenza e informativa su più livelli](https://www.edpb.europa.eu/documents/guideline/article-29-working-party-guidelines-on-transparency-under-regulation-2016679_en)


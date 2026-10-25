# Chiusura votazioni, premi e WhatsApp

Nella dashboard dell'admin società, il superadmin può salvare:

- uno o più organizzatori con nome, telefono internazionale e ricezione abilitata;
- il Content Template SID Twilio della società (`HX` e 32 caratteri esadecimali);
- la modalità casuale oppure i tre migliori nella classifica tifosi tra i votanti.

Le impostazioni appartengono alla singola società. Al login lo staff apre
direttamente Chiusura votazioni e accede soltanto a questa schermata, con i
comandi sullo stato delle votazioni e gli esiti delle notifiche. Dashboard,
Risultati e Lotteria non sono disponibili allo staff, che non può modificare
destinatari o template.

Sotto il Template SID, il campo **Numero WhatsApp di prova** e il pulsante
**Testa** permettono al superadmin di inviare un messaggio reale di prova.
Si usa il SID attualmente inserito, anche prima di salvare la configurazione,
con risultati di esempio e il giocatore “Mario Rossi (TEST)”. Il numero di prova
non viene salvato né aggiunto agli organizzatori; il test non modifica eventi,
premi o notifiche della chiusura. Il mittente WhatsApp è lo stesso dell'invio
agli organizzatori. Il form segnala l'accettazione da parte di Twilio oppure
l'errore restituito; la ricezione va verificata sul telefono di prova.

## Al clic su “Chiudi votazioni”

Un'unica transazione chiude i voti, salva MVP e percentuale sul totale dei voti
dell'evento, assegna i premi in ordine di posizione e accoda i messaggi. Anche i
voti anonimi contano per l'MVP; per i premi occorre un codice numerico di quattro
cifre collegato, tramite le sessioni del dispositivo, a un tifoso della società
con telefono verificato, come nella lotteria. Sono ammessi anche i voti espressi
prima di completare il profilo, se al momento della chiusura il profilo è valido.

- **Casuale:** estrazione uniforme dei ticket idonei senza ripetere un ticket,
  come nella lotteria esistente.
- **Classifica:** punti della classifica generale della società, filtrata sui
  tifosi idonei che hanno votato a questa partita. Il primo riceve il primo
  premio e così via; uno stesso tifoso non riceve più premi. Si usa il punteggio
  cumulativo già esposto dalla classifica, che non diminuisce quando si spendono
  monete. A parità vince chi precede nella classifica (ID tifoso crescente).
  Questa modalità assegna al massimo i primi tre premi.

I premi già assegnati manualmente vengono conservati. Con pochi partecipanti i
premi rimanenti sono indicati come “Non assegnato”; con zero voti l'MVP è
“Nessun voto” e la percentuale “0,00”. La chiusura resta possibile anche senza
destinatari o credenziali WhatsApp configurate.

I risultati della prima chiusura sono definitivi: clic ripetuti, chiamate
concorrenti o riattivazioni dell'evento non ripetono estrazione e notifiche.
La lotteria mostra gli stessi premi e codici e non può annullare vincitori già
inclusi nella chiusura automatica. I voti arrivati dopo la chiusura vengono
rifiutati anche se la richiesta era già in corso.

## Twilio

Il server usa `ACCOUNT_SID`, `AUTH_TOKEN` e `WHATSAPP_MESSAGING_SERVICE_ID`
già supportati. Il servizio deve avere un mittente WhatsApp. Il Template SID
salvato nella dashboard deve appartenere allo stesso account Twilio e riferirsi
a un template WhatsApp approvato con queste otto variabili:

| Variabile | Valore |
| --- | --- |
| 1 | Nome e cognome MVP |
| 2 | Percentuale su tutti i voti dell'evento, due decimali, senza `%` |
| 3, 5, 7 | Nomi dei primi tre premi in ordine |
| 4, 6, 8 | Codici vincitori, conservando gli zeri iniziali |

Per gli slot premio non presenti si inviano “Premio non previsto” e “-”.
I valori non contengono interruzioni di riga. Le richieste usano `ContentSid`
e `ContentVariables`; non usano `Body`.

La notifica al singolo vincitore continua a usare il template globale opzionale
`CONTENT_TEMPLATE_WINNER_SID`, distinto dal riepilogo agli organizzatori.

Il worker invia la coda persistente in background e riprende le notifiche ancora
in attesa dopo un riavvio. La sezione Chiusura votazioni mostra gli esiti per
organizzatore: in coda, invio, accettato da Twilio, fallito, configurazione
mancante o esito incerto. “Accettato da Twilio” non attesta la consegna al
telefono: non è stato aggiunto un webhook per gli aggiornamenti di consegna.

“Riprova notifiche non inviate” riaccoda solo gli organizzatori falliti o saltati
che sono ancora abilitati, usando il Template SID attualmente salvato e gli
stessi risultati. Non ripete i messaggi già accettati e non riestrae i premi.
Timeout di rete e invii interrotti richiedono una verifica nel registro Twilio:
non vengono ritentati automaticamente per evitare messaggi duplicati.

Le nuove tabelle vengono create automaticamente all'avvio del backend.
I test usano database temporanei e un trasporto HTTP simulato, senza WhatsApp reali.

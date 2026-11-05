# Admin V2: integrazione con la logica esistente

Il vecchio `AdminPortal.vue` e i suoi componenti restano invariati. La V2 non
monta né incorpora la loro interfaccia. Il plugin Vite
`build/adminV2Controllers.js` legge gli script setup originali ed espone un
composable tramite la query `?admin-v2-controller`:

```js
import { useController } from '../../components/AdminPortal.vue?admin-v2-controller';
const state = useController(reactiveProps, emit);
```

La trasformazione usa l'AST e i binding del compilatore ufficiale Vue già
presente nel progetto. Rimuove esclusivamente import di componenti `.vue` e
macro di compilazione, mantiene gli import dei servizi e restituisce lo stato
e le funzioni applicative. Ogni invocazione crea un'istanza isolata. Template,
CSS e layout precedenti non entrano nel modulo generato. CRUD, payload,
normalizzazione, validazioni, notifiche operative e permessi conservano quindi
un'unica fonte nel codice attuale.

Per il solo controller principale il plugin applica sette punti di integrazione
protetti da controlli di struttura: verifica sessione, loading iniziale, errori
delle richieste, logout, login, bootstrap e cambio società. Il bootstrap è affidato a
`useAdminV2Controller`: nessuna risorsa viene caricata prima del completamento
di `/admin/me`. Un errore 401, 403 o di rete interrompe il caricamento; il ruolo
salvato in localStorage non consente di superare questa verifica. Le credenziali
continuano a essere i cookie del client API esistente. Non viene introdotto un
sistema di autenticazione parallelo.

Gli errori 401 concorrenti possono arrivare sia dall'interceptor della V2 sia
dal gestore delle richieste condiviso. Il controller V2 mantiene una sola
richiesta di logout pendente: pulisce immediatamente lo stato locale e fa
convergere i gestori sulla stessa operazione. Un nuovo login attende che tale
richiesta termini prima di inviare le credenziali, evitando che una risposta di
logout tardiva cancelli la nuova sessione. La sequenza, comprese le chiamate
API, è verificata da un test di concorrenza. Il controller legacy non cambia.

La radice dell'applicazione è ricreata al cambio slug: stato, watcher e polling
del workspace precedente vengono rilasciati. Le rotte usano lo stesso routing
History API del resto del progetto; Vue Router non è una dipendenza esistente.
Il routing dei domini ClubHub mantiene la precedenza e non viene modificato.

Se i punti di integrazione originali cambiano, il build fallisce esplicitamente
invece di generare un adapter non verificato. I test in `adminV2.test.js`
controllano i controller realmente generati e le chiamate API dei percorsi di
sessione e di autorizzazione, oltre ai confini delle rotte.

Eseguire dalla directory frontend:

```text
node --test src/admin-v2/adminV2.test.js
npm run build
```

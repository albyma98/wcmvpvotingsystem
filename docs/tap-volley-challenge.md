# Tap Volley Challenge

`tap-volley` è una voce separata nel catalogo minigiochi del master. È assegnata di default a **Pallavolo**; il master può modificarne gli sport o disabilitarla indipendentemente da Tap Challenge (`tap`). Le assegnazioni esistenti di Tap Challenge sono conservate.

Nella pagina `/<società>/new`, il gioco compare in “Gioca e guadagna” e può comparire come “Sfida del momento” se consentito dalla configurazione dello sport. Usa un pallone giallo e blu, un campo con rete da pallavolo, testi dedicati e la copertina `wcmvpvs-front/src/assets/fan-games/tap-volley.webp`.

Una partita singola dura 10 secondi. Il pallone cambia posizione e accelera a ogni tocco. Ogni partita completata dà 8 monete, con importo stabilito dal server e una ricevuta per impedire un doppio accredito dello stesso round. Saldo, punti e progressi sono ripristinati dal server.

La sfida live resta riservata ai tifosi registrati. I giocatori sono abbinati solo nello stesso gioco, evento e società. Countdown, punteggio, risultato, abbandono e rivincita riutilizzano il motore di Tap Challenge. Gli endpoint volley sono sotto `/events/{eventId}/tap-volley-live/`; gli endpoint basket restano sotto `/tap-live/`. Le operazioni di pulizia e lettura rimangono accessibili per terminare una partita già iniziata anche se il master disabilita successivamente il gioco.

All’avvio viene aggiunta la colonna `tap_live_matches.game_id`. Le partite precedenti rimangono attribuite a `tap`; le nuove partite volley usano `tap-volley`. Risultati live e progressi vengono conteggiati separatamente, mentre monete e punti confluiscono nello stesso wallet del tifoso.

La variante Vue `TapVolleyChallenge.vue` condivide il motore con `TapChallenge.vue`, così regole, salvataggi e gestione delle sfide live restano coerenti. Il catalogo, la grafica, i permessi dello sport, le ricevute dei premi e l’abbinamento live identificano separatamente le due versioni.

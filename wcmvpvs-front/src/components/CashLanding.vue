<script setup>
import cashLogo from '../assets/cash-logo.svg'
</script>

<template>
  <main class="landing-page">
    <section class="hero" data-animate>
      <div class="hero__logo" data-animate>
        <h1 class="hero__title">Welcome to Cash</h1>
        <img :src="cashLogo" alt="Cash brand logo" class="logo-image" />
        <p class="hero__subtitle">Sport. Tech. Motivation.</p>
      </div>
      <span class="divider" aria-hidden="true">⸻</span>
    </section>

    <section class="content" data-animate>
      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">La visione</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p>CASH nasce per unire tre mondi che spesso si sfiorano ma raramente si incontrano davvero: lo sport, la tecnologia e la motivazione personale.</p>
          <p>Dietro il nome Cash c’è una storia sportiva reale, fatta di passione, fatica, crescita e obiettivi inseguiti giorno dopo giorno. Da quella esperienza nasce una filosofia: trasformare l’energia dello sport in un modo di vivere.</p>
          <p>CASH non è solo un brand, ma una mentalità. Un invito a spingersi oltre, a credere nelle proprie capacità e a trasformare ogni punto, ogni allenamento, ogni sfida in una possibilità di crescita.</p>
        </div>
      </article>

      <span class="divider" aria-hidden="true">⸻</span>

      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">La missione</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p>Creare un ecosistema dove la motivazione incontra l’azione. Dove la grinta dell’atleta, la partecipazione del pubblico e la visione degli sponsor diventano un unico linguaggio: quello della connessione vera.</p>
          <p>CASH costruisce ponti tra chi vive lo sport e chi lo sostiene, tra chi indossa i valori e chi li mette in campo. Ogni progetto nasce per ispirare, coinvolgere e misurare ciò che conta davvero: l’impatto umano dietro la performance.</p>
        </div>
      </article>

      <span class="divider" aria-hidden="true">⸻</span>

      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">Le due anime del brand</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p class="content__highlight">🖤 Wearing Cash</p>
          <p>La parte emotiva e visiva del brand. Una linea streetwear che racconta la determinazione di chi crede nei propri sogni e non smette di inseguirli. Ogni capo è un simbolo: rappresenta la costanza, la resilienza, l’identità sportiva di chi vive con passione e disciplina.</p>
          <p class="content__quote">“Wear your grind. Chase your dreams.”<br />Non è uno slogan: è una promessa.</p>
        </div>
      </article>

      <span class="divider" aria-hidden="true">⸻</span>

      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">⚡ Cash MVP System</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p>La parte tecnologica e interattiva del mondo Cash. Una piattaforma che trasforma i tifosi in protagonisti, permettendo loro di votare in tempo reale il miglior giocatore della partita e di vivere un’esperienza digitale dentro al palazzetto.</p>
          <p>Cash MVP System è pensato per le società sportive che vogliono dare più valore ai propri sponsor e coinvolgere il pubblico come mai prima. Ogni voto, ogni interazione, ogni dato raccolto racconta qualcosa di più grande: l’energia condivisa dello sport.</p>
          <p class="content__quote">“Diamo voce ai tifosi.<br />Diamo dati concreti alle società.<br />Diamo valore a chi crede nella passione sportiva.”</p>
        </div>
      </article>

      <span class="divider" aria-hidden="true">⸻</span>

      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">Un unico linguaggio</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p>CASH rappresenta la fusione tra performance e ispirazione. Che si tratti di una felpa o di un sistema digitale, il messaggio è lo stesso: credi, spingi, rendilo possibile.</p>
          <p>Lo sport è l’origine, la tecnologia è il mezzo, la motivazione è il collante. Insieme, diventano esperienza.</p>
        </div>
      </article>

      <span class="divider" aria-hidden="true">⸻</span>

      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">Il significato di CASH</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p>Per qualcuno è solo una parola. Per noi è un simbolo. Un nome che nasce dallo sport e che oggi rappresenta un modo di pensare:</p>
          <p>Cash è costanza, ambizione, sacrificio e cuore.</p>
          <p>È la memoria di chi lavora ogni giorno per un obiettivo, e la visione di chi crede che ogni piccolo passo costruisca un risultato più grande.</p>
        </div>
      </article>

      <span class="divider" aria-hidden="true">⸻</span>

      <article class="content__block">
        <header class="content__header">
          <h2 class="content__title">Inside &amp; Outside the Game</h2>
          <span class="content__underline"></span>
        </header>
        <div class="content__body">
          <p>CASH vive dentro e fuori dal campo. Nel sudore, nelle scelte, nei dettagli. In chi lotta per migliorarsi e in chi non smette di ispirare gli altri.</p>
          <p>È l’energia che scorre tra atleti, tifosi, brand e comunità. Un ecosistema che trasforma la passione in movimento e la motivazione in impatto reale.</p>
          <p class="content__highlight">CASH — Sport. Tech. Motivation.<br />Wear it. Live it. Power it.</p>
        </div>
      </article>
    </section>

    <span class="divider divider--footer" aria-hidden="true">⸻</span>

    <footer class="footer" data-animate>
      <p class="footer__location">📍 #PoweredByCash</p>
      <p class="footer__hashtags">#WearingCash • #CashMVPSystem</p>
      <p class="footer__hashtags">#PoweredByCash • #WearingCash • #CashMVPSystem</p>
    </footer>
  </main>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Montserrat:wght@400;500;600;700;800&display=swap');

:global(body) {
  margin: 0;
  background-color: #0c0c0d;
}

.landing-page {
  font-family: 'Montserrat', 'Nexa', sans-serif;
  background: #0c0c0d;
  color: #f5f5f5;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 4rem 1.5rem 5rem;
  gap: 3rem;
}

.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 1rem;
}

.hero__logo {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  animation: fadeIn 1.2s ease forwards;
  opacity: 0;
}

.hero__title {
  font-size: clamp(1.3rem, 3vw, 1.6rem);
  letter-spacing: 0.35rem;
  text-transform: uppercase;
  color: #ff5c33;
  margin: 0;
}

.logo-image {
  max-width: 100%;
  width: clamp(180px, 45vw, 360px);
  height: auto;
  filter: drop-shadow(0 12px 20px rgba(255, 44, 32, 0.35));
}

.hero__subtitle {
  font-size: clamp(1.1rem, 3vw, 1.6rem);
  letter-spacing: 0.4rem;
  text-transform: uppercase;
  color: #bdbdbd;
  margin: 0;
}

.divider {
  display: inline-block;
  color: rgba(255, 255, 255, 0.25);
  letter-spacing: 1rem;
  font-size: 1.1rem;
  margin: 1.5rem auto 0;
  text-transform: uppercase;
  animation: fadeIn 1.1s ease forwards;
  opacity: 0;
}

.divider--footer {
  margin-top: 2rem;
}

.content {
  width: min(900px, 100%);
  display: flex;
  flex-direction: column;
  gap: 3rem;
}

.content__block {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  background: rgba(255, 255, 255, 0.03);
  padding: 2.5rem 2rem;
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 18px 45px rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(4px);
  animation: fadeIn 1.1s ease forwards;
  opacity: 0;
}

.content__header {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.content__title {
  font-size: clamp(1.6rem, 4vw, 2.4rem);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.3rem;
  margin: 0;
  transition: color 0.3s ease, transform 0.3s ease;
  cursor: default;
}

.content__title:hover {
  color: #ff5c33;
  transform: translateY(-4px);
}

.content__underline {
  width: 80px;
  height: 3px;
  background: linear-gradient(90deg, #ff5c33 0%, rgba(255, 92, 51, 0.15) 100%);
  border-radius: 999px;
}

.content__body {
  color: #d6d6d6;
  font-size: 1.05rem;
  line-height: 1.9;
}

.content__body p {
  margin: 0 0 1rem;
}

.content__body p:last-child {
  margin-bottom: 0;
}

.content__highlight {
  color: #ff5c33;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.content__quote {
  font-style: italic;
  color: #ffffff;
  font-weight: 500;
}

.footer {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  align-items: center;
  text-transform: uppercase;
  letter-spacing: 0.4rem;
  color: #ff5c33;
  font-weight: 600;
  font-size: clamp(0.9rem, 2.5vw, 1.1rem);
  animation: fadeIn 1.3s ease forwards;
  opacity: 0;
  text-align: center;
}

.footer__location {
  margin: 0;
}

.footer__hashtags {
  margin: 0;
  color: #f5f5f5;
  letter-spacing: 0.3rem;
}

[data-animate] {
  animation: fadeIn 1s ease forwards;
  opacity: 0;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(24px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 640px) {
  .landing-page {
    padding: 3rem 1.25rem 4rem;
  }

  .content__block {
    padding: 2rem 1.5rem;
  }

  .content__title {
    letter-spacing: 0.2rem;
  }

  .footer {
    letter-spacing: 0.25rem;
  }

  .footer__hashtags {
    letter-spacing: 0.2rem;
  }
}
</style>

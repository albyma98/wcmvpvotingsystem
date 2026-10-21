// Static URLs let Vite fingerprint and bundle each lightweight cover.
export const FAN_GAME_ARTWORK = Object.freeze({
  vote: new URL('./assets/fan-games/mvp.webp', import.meta.url).href,
  reaction: new URL('./assets/fan-games/reaction.webp', import.meta.url).href,
  quiz: new URL('./assets/fan-games/quiz.webp', import.meta.url).href,
  basketball: new URL('./assets/fan-games/basketball.webp', import.meta.url).href,
  playmaker: new URL('./assets/fan-games/playmaker.webp', import.meta.url).href,
  crossover: new URL('./assets/fan-games/crossover.webp', import.meta.url).href,
  tap: new URL('./assets/fan-games/tap.webp', import.meta.url).href,
  'sponsor-rush': new URL('./assets/fan-games/sponsor-rush.webp', import.meta.url).href,
  'memory-flash': new URL('./assets/fan-games/memory-flash.webp', import.meta.url).href,
});

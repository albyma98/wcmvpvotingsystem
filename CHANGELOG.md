# Changelog

## [Unreleased] — Branded Game Modal (Task 1–4)

### Added

#### Backend
- **`branded_game_participations` table** — Nuova tabella SQLite per registrare le partecipazioni dei fan ai branded game (event_id, device_id, user_id nullable, score, completed, rewarded_coins).
- **`events.show_branded_game`** — Nuova colonna BOOLEAN sulla tabella `events` (DEFAULT 0).
- **`events.branded_game_config`** — Nuova colonna TEXT (JSON) sulla tabella `events`.
- **`GET /events/{eventId}/branded-game`** — Endpoint pubblico che ritorna la config del branded game e `can_play` / `plays_used` / `plays_remaining` basati sul device_id.
- **`POST /events/{eventId}/branded-game/result`** — Endpoint pubblico che registra il risultato, accredita coin (fan_wallets o guest_wallets), scrive tracking events `branded_game.completed` e `branded_game.reward_claimed`. Rate limit: 20 req/min per device.
- **`BrandedGameConfig` struct** con validazione completa (sponsor_id required, game_type whitelist, reward_type whitelist, reward_coins ≥ 0, max_plays_per_user ≥ 1, cta_url no javascript:/data: XSS).
- **30 unit test** in `branded_game_test.go` (validazione, parsing, play limit, coin reward, JSON round-trip, XSS URL).

#### Frontend — Admin
- **`defaultBrandedGameConfig()`** — Factory con valori di default per la config branded game.
- **`validateBrandedGameConfigClient()`** — Validazione client-side prima del save.
- **Form branded game** nel blocco "Crea nuovo evento" — toggle + form inline espandibile con preview live.
- **Form branded game** nella lista eventi esistenti — stesso pattern, integrato nel tab "Impostazioni".
- **CSS classi `bg-*`** aggiunte in `admin-theme.css` (~100 righe).
- Tutti e 3 i payload PUT/POST evento aggiornati per includere `show_branded_game` e `branded_game_config`.

#### Frontend — Player
- **`BrandedGameEntry.vue`** — Pill full-width con logo sponsor, nome, game type, badge reward. IntersectionObserver per impression tracking (once per session via sessionStorage); fallback su click per browser senza IO. Emits: `@open`.
- **`BrandedGameModal.vue`** — Teleport to body, z-[125]. Header brandizzato con primary/secondary color. 5 fasi: `intro → playing → submitting → result / exhausted / error`. Game lazy-loaded via `defineAsyncComponent` per tipo. `onErrorCaptured` per crash del mini-game. Tracking eventi: `opened`, `started`, `completed`, `reward_claimed`, `cta_clicked`, `dismissed`. ESC listener + `aria-modal`.
- **`LiveExperienceHome.vue`** — `loadBrandedGame()` su mount, `BrandedGameEntry` posizionato dopo la grid 3-col, `BrandedGameModal` nel blocco Teleport.

### Security fixes (parte di questa release)
- Token admin spostato da `localStorage` a `httpOnly cookie` (`admin_session`).
- Rate limit login admin/partner: 5 tentativi/username, 10/IP in 10 min → 429.
- Bundle splitting: AdminPortal e MasterPortal in chunk separati, non più scaricati dagli utenti fan.
- Password admin migrata da SHA256 a bcrypt (cost=12) con upgrade silenzioso al login.
- Security headers: `X-Frame-Options`, `HSTS`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`.
- CORS ristretto da `*` a `https://mvp.wearingcash.it` (configurabile via `ALLOWED_ORIGIN`).
- Rimosso codice shop (frontend + backend) non utilizzato.
- Rate limit su `POST /events/{eventId}/branded-game/result`: 20 req/min per device.
- Validazione `cta_url`: rifiuta `javascript:` e `data:` URI.

### Changed
- `Event` struct: aggiunto `ShowBrandedGame bool` e `BrandedGameConfig string`.
- 5 query SQL eventi aggiornate (INSERT, UPDATE, 2× SELECT list, GetActiveEvent).
- `vite.config.js`: aggiunto Vite proxy per dev (`/api` → backend), rimossi `manualChunks` che causavano preloading admin bundle agli utenti.
- `api.ts`: `withCredentials: true` su axios, dev usa path relativo `/api`.

### Fixed (QA Task 5)
- **BrandedGameEntry**: fallback impression tracking su click se `IntersectionObserver` non disponibile.
- **BrandedGameModal**: `onErrorCaptured` per crash del mini-game → fase `error` invece di crash silenzioso.
- **Backend rate limit** su branded game result endpoint.
- **XSS**: validazione `cta_url` per bloccare `javascript:` e `data:` URI.

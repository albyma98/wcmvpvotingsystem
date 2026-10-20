# Fan wallet and collection: first phase

The fan home at `/:slug/new` uses a spendable coin balance and separate cumulative participation points. Spending coins does not reduce points or match badges. The home has one contextual panel (first goal, suggested game, or match recap), a compact leaderboard entry, a wallet sheet, and a digital collection in the profile.

Guests can vote, earn coins and browse the shop and item previews. Opening the wallet or profile prompts them to create a fan profile or sign in. After an MVP vote, guests see the profile invitation in place of the next badge goal. Shop purchases require a signed-in, completed profile; guest, invalid and incomplete sessions cannot spend through either purchase API. The client returns to the shop after authentication and waits for a new purchase click.

## Catalog

| Key | Cost | Ownership | Use |
| --- | ---: | --- | --- |
| `club_frame` | 20 | Once per club and holder | Equip or remove the profile frame. |
| `match_card` | 15 | Once per club, match and holder | Keep a match souvenir with actual club, opponent and date. |

Prices are defined on the server. Retrying an already completed purchase does not debit the balance again. There are no paid top-ups or physical sponsor offers in this phase. Existing lottery prizes are separate from the shop.

## API

All three routes require the organization scope and device headers; authenticated requests also send the fan session header.

- `GET /events/{eventId}/fan-wallet`: confirmed balance, points, latest 30 movements, catalog and collection.
- `POST /events/{eventId}/fan-shop/purchase`, body `{ "item_key": "club_frame" }` or `match_card`.
- `POST /events/{eventId}/fan-shop/equip`, body `{ "item_key": "club_frame" }` or an empty string to remove it.

Mutation responses have the same state shape as the GET. The client never changes ownership or subtracts a purchase price before acknowledgement. Reads and mutations from a previous event or club are discarded by the home.

The purchase route and legacy `rewards/redeem` require a fan session for a completed profile. Missing or incomplete profiles receive `401` with code `PROFILE_REQUIRED`; invalid sessions receive `401`. Wallet/catalog reads remain available to guests.

## Persistence and migration

Economy tables are initialized with the existing fan tables when the backend starts. Wallet triggers record persistent deltas, including the existing server-side game paths. Existing balances have an explicit opening entry; the migration does not invent a transaction history. Registered fans' initial points also include known recorded redemptions and support donations. Unrecorded historical spending cannot be reconstructed.

Anonymous funds, points, purchases and participation are adopted once when that device authenticates or registers with the same club. Values supplied in `guest_coins` are not accepted as funds. A previously registered phone needs its verified session rather than obtaining access through a new registration request.

The `Tifoso Attivo` badge is based on 25 earned coins for a match and is independent of purchases. Collection participation counts represent recorded votes or rewarded activity, not physical attendance.

## Older client compatibility

`guest-coins` is now a balance read: clients cannot overwrite the wallet with an arbitrary total. `rewards/redeem` only accepts the server catalog. Older home implementations using local rewards, multipliers, wheel outcomes or arbitrary reward keys must migrate to server actions before those rewards can persist. The new home uses fixed, once-per-match journey claims for MVP, quiz, Reaction Test, Sponsor Rush, Memory Flash and solo Tap Challenge. Basketball games and live Tap Challenge keep their authoritative game receipts. Live Tap earnings also restore cumulative match progress. Memory Flash retries are free.

Deploy the frontend and backend together. A frontend-only release cannot use the wallet/shop routes.

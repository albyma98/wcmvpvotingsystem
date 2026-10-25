# Sports collectibles artwork

Generated with the built-in `image_gen` tool from the approved two-object concept. The source concept is a preview, not a club photograph. The arena is generic illustrative artwork. Club logos, names, opponents, dates and avatar initials are rendered by Vue from actual application data; no match information is baked into these images.

- `avatar-frame.png`: 1254 × 1254 RGBA. Transparent outside and through the center, verified after generation.
- `match-card.png`: 1086 × 1448 RGBA, a 3:4 background without typography.

Production WebP exports preserve transparency and are the files imported by Vue:

| File | Dimensions | Size |
| --- | --- | ---: |
| `avatar-frame-small.webp` | 256 × 256 | 22,526 bytes |
| `avatar-frame.webp` | 640 × 640 | 95,582 bytes |
| `match-card.webp` | 768 × 1024 | 287,804 bytes |
| `match-card-volleyball.webp` | 768 × 1024 | 150,960 bytes |

Mechanical export used Sharp with WebP quality 90 for the frames, 88 for the card, alpha quality 100, effort 6. Sharp is an export tool only and is not a project dependency. The avatar uses responsive `srcset`/`sizes`; original PNGs are not imported into the application build.

Components share these assets across shop, profile and object detail. Shop/profile collectible artwork is mounted when the corresponding sheet opens; the avatar frame is only requested when displayed as framed. Both assets use neutral metal panels, with club color accents supplied by the interface.

The volleyball variant was generated with the built-in `image_gen` tool using `match-card.webp` as the edit target. `match-card-volleyball.png` is the 1086 × 1448 source; its production WebP uses the same 768 × 1024 size and quality 88 export as the basketball card. `FanMatchCard` selects it for the society sport `Pallavolo`, supplied by `/organization-theme`, across the shop, preview, purchase reveal and profile collection. The footer ball icon also follows the sport. The existing background remains the default for other or missing sports.

## Final prompts

### Volleyball match card

Create a volleyball variant of the provided existing basketball collectible card background. This image is the edit target: preserve its portrait 3:4 aspect ratio, silver beveled metal perimeter and corner shapes, black textured empty top plaque and empty bottom plaque, and premium photorealistic dark indoor sports arena style. Replace the basketball court image in the central window with a professional indoor volleyball court: orange playing surface with clear white volleyball boundary lines and teal surrounding free zone, taut regulation volleyball net spanning across the middle of the court with two padded posts and slim antennae. Similar low courtside wide-angle viewpoint, empty stadium seating and dramatic arena lights, sharp realistic floor reflections. The net must be prominent in the central area and remain recognizable at thumbnail size. Remove basketball hoops, backboards and all basketball court markings. No people, no balls, no text, no lettering, no logos, no watermarks. Keep the empty plaques and overall visual composition of the reference for UI text overlays. Generate a finished production image for the app, not a mockup.

### Avatar frame

Use case: background-extraction / precise-object-edit. Reference image is the approved two-object sports concept. Produce ONLY the circular avatar frame from the left half as a production-ready standalone PNG asset, viewed straight-on without perspective. Keep its sophisticated machined brushed chrome silver segments, deep graphite inset panels, beveled edges, directional realistic highlights and small basketball medallion at bottom. Adapt all red enamel panels to neutral charcoal graphite so club-specific color accents can be added in the application. The circular opening in the middle MUST BE FULLY TRANSPARENT, with no TU letters, no grey fill, no white disk, no glass. All the area OUTSIDE the metal ring MUST BE FULLY TRANSPARENT. No cast shadow painted on an opaque background; tiny natural self-shadow on the metal only. A square canvas, ring centered and almost edge-to-edge, outer diameter 94% of canvas width, center hole diameter about 62% of canvas width. Compact clean silhouette and strong highlights that remain visible at 40px. Absolutely no text, prices, profile panel, logos, labels, second small ring, objects, or border around canvas. The output is a single isolated transparent wearable chrome basketball avatar frame matching the approved reference.

### Match card

Use case: precise-object-edit / product-mockup. Reference image is the approved two-object concept. Produce ONLY the right-hand collectible basketball card as a production-ready standalone artwork background. Straight-on, flat, ZERO perspective or tilt, vertical 3:4 card. Keep its premium graphite leather/micrograin panels, crisp machined silver chrome frame, slightly cut corners, realistic directional lighting, and beautifully lit photoreal generic EMPTY indoor basketball court. In the top 58% of the card show a rich arena interior in dramatic warm light with detailed parquet/court lines and a basketball hoop on the right, no people. Leave an empty black graphite header panel (top14%) for a real club logo/title and a generous clean empty black graphite lower panel (bottom30%) for dynamic match text. All red inlays in the reference must become NEUTRAL CHARCOAL GRAPHITE, so real club colors will be applied in code. Absolutely NO letters, logos, dates, statistics, numbers, serials, prices, placeholder text or watermarks anywhere on the card. The dynamic texts will be real HTML in the application. Fill the canvas with the card edge-to-edge, outside tiny cut corners genuinely transparent; no surrounding studio background or cast shadow, no second card or avatar. Fine metallic edging but preserve large clean text areas, high-quality sports collectible proportions.

const sans = 'ui-sans-serif, system-ui, -apple-system, sans-serif'
const serif = 'Georgia, "Times New Roman", serif'
// ENDURA is distributed as a licensed webfont, separately from Google Fonts.
const enduraStylesheet = import.meta.env?.VITE_CLUB_ENDURA_STYLESHEET || ''

export const CLUB_FONTS = Object.freeze([
  { name: 'Inter', fallback: sans, weights: '400;500;600;700;800;900' },
  { name: 'Montserrat', fallback: sans, weights: '400;500;600;700;800;900' },
  { name: 'Poppins', fallback: sans, weights: '400;500;600;700;800;900' },
  { name: 'Roboto', fallback: sans, weights: '400;500;600;700;800;900' },
  { name: 'Open Sans', fallback: sans, weights: '400;500;600;700;800' },
  { name: 'Lato', fallback: sans, weights: '400;700;900' },
  { name: 'Barlow Condensed', fallback: sans, weights: '400;500;600;700;800;900' },
  { name: 'Oswald', fallback: sans, weights: '400;500;600;700' },
  { name: 'Merriweather', fallback: serif, weights: '400;500;600;700;800;900' },
  { name: 'Playfair Display', fallback: serif, weights: '400;500;600;700;800;900' },
  { name: 'ENDURA', fallback: sans, stylesheet: enduraStylesheet, disabled: !enduraStylesheet },
  { name: 'Arial', fallback: sans },
  { name: 'Georgia', fallback: serif },
])

export function clubTypography(club = {}, fonts = CLUB_FONTS) {
  const findFont = name => fonts.find(font => !font.disabled && font.name === name) || fonts[0]
  const title = findFont(club.titleFont)
  const body = findFont(club.bodyFont)
  const webFonts = [...new Set([title, body])].filter(font => font.weights)
  const stylesheet = webFonts.length
    ? `https://fonts.googleapis.com/css2?${webFonts.map(font => `family=${encodeURIComponent(font.name).replace(/%20/g, '+')}:wght@${font.weights}`).join('&')}&display=swap`
    : ''
  return {
    style: {
      '--club-font-title': `"${title.name}", ${title.fallback}`,
      '--club-font-body': `"${body.name}", ${body.fallback}`,
    },
    stylesheet,
    stylesheets: [...new Set([stylesheet, title.stylesheet, body.stylesheet].filter(Boolean))],
  }
}

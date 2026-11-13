const qrLabels = ['qr', 'qr-code', 'qrcode'];
const socialLabels = ['social', 'organic-social', 'paid-social', 'instagram', 'facebook', 'fb', 'ig', 'tiktok', 'x', 'twitter', 'linkedin', 'youtube', 'whatsapp', 'telegram'];
const socialDomains = ['facebook.com', 'instagram.com', 'tiktok.com', 'x.com', 'twitter.com', 'linkedin.com', 'youtube.com', 'youtu.be', 'wa.me', 'whatsapp.com', 't.co', 'fb.com', 't.me', 'telegram.org'];
const placements = new Map([
  ['led', 'led'], ['led-arena', 'led'], ['arena-led', 'led'], ['ledwall', 'led'], ['led-wall', 'led'], ['maxischermo', 'led'],
  ['giornalino', 'match_programme'], ['match-programme', 'match_programme'], ['programma-partita', 'match_programme'], ['programma', 'match_programme'], ['magazine', 'match_programme'],
  ['arena', 'arena'], ['remote', 'remote'], ['direct', 'direct'],
]);

function channelKey(value) {
  return (value || '').toLowerCase().trim().split(/[\s_-]+/).join('-');
}

export function matchdayAttribution(href, referrer = '') {
  let entry;
  try { entry = new URL(href); } catch { return { channel: 'unknown', qr: false, arrival: '' }; }
  const params = entry.searchParams;
  const source = channelKey(params.get('utm_source') || params.get('source'));
  const medium = channelKey(params.get('utm_medium'));
  const qr = qrLabels.includes(source) || qrLabels.includes(medium) || source.startsWith('qr-') || source.endsWith('-qr');
  const tag = params.get('abx_qr') || '';
  const arrival = /^[a-f0-9]{32}$/.test(tag) ? tag : '';
  const placement = placements.get(source.replace(/^qr-|-qr$/g, ''));
  let channel = placement;
  if (!channel) {
    if (qr) channel = 'qr';
    else if (socialLabels.includes(source) || socialLabels.includes(medium)) channel = 'social';
    else if (medium === 'email' || ['email', 'newsletter'].includes(source)) channel = 'email';
    else if (['cpc', 'paid', 'paid-search'].includes(medium)) channel = 'paid';
    else if (medium === 'organic' || ['google', 'bing', 'duckduckgo'].includes(source)) channel = 'search';
    else if ((source && source !== 'unknown') || medium || params.has('utm_campaign')) channel = 'campaign';
    else channel = referrerChannel(entry.hostname, referrer);
  }
  return { channel, qr, arrival };
}

function referrerChannel(hostname, referrer) {
  if (!referrer) return 'direct';
  let referrerHost;
  try { referrerHost = new URL(referrer).hostname; } catch { return 'unknown'; }
  if (!referrerHost) return 'unknown';
  if (hostname === referrerHost) return 'internal';
  if (socialDomains.some(domain => referrerHost === domain || referrerHost.endsWith(`.${domain}`))) return 'social';
  if (/^(www\.)?google\.[a-z.]+$/.test(referrerHost) || referrerHost === 'bing.com' || referrerHost.endsWith('.bing.com') || referrerHost === 'duckduckgo.com') return 'search';
  return 'referral';
}

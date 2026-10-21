// Bump together with database.FanLegalVersion whenever the published texts change.
export const FAN_LEGAL_VERSION = '2026-10-21.1';
export const FAN_MINIMUM_AGE = 14;

export function fanRegistrationAcceptance(form) {
  return {
    accepted_terms: form.acceptedTerms === true,
    age_eligibility_declared: form.ageEligibilityDeclared === true,
    minimum_age_declared: FAN_MINIMUM_AGE,
    legal_version: FAN_LEGAL_VERSION,
  };
}

export function legalTextBlocks(source, clubName) {
  const text = String(source).replaceAll('{{clubName}}', String(clubName || 'il club selezionato'));
  return text.trim().split(/\r?\n\s*\r?\n/).map((paragraph) => {
    if (paragraph.startsWith('# ')) return { type: 'title', text: paragraph.slice(2).trim() };
    if (paragraph.startsWith('## ')) return { type: 'heading', text: paragraph.slice(3).trim() };
    return { type: 'paragraph', text: paragraph.replace(/\r?\n/g, ' ').trim() };
  });
}

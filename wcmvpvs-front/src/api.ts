export async function vote({ eventId, playerId }) {
  console.info('Mock vote request', { eventId, playerId });
  await new Promise((resolve) => setTimeout(resolve, 350));
  return { ok: true };
}

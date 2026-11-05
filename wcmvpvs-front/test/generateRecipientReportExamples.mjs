import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createEngagementReportPdf } from '../src/engagementReportPdf.js';
import { recipientReportFixture } from './recipientReportFixture.js';

const out = resolve(process.argv[2] || '../output/pdf');
mkdirSync(out, { recursive: true });
for (const id of ['executive', 'marketing', 'sponsor', 'analytics']) {
  const doc = createEngagementReportPdf(recipientReportFixture(id));
  writeFileSync(resolve(out, `arenaboostx-${id}-esempio.pdf`), Buffer.from(doc.output('arraybuffer')));
  console.log(`${id}: ${doc.getNumberOfPages()} pagine, dati sintetici`);
}

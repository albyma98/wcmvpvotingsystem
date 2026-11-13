import assert from 'node:assert/strict';
import test from 'node:test';
import { matchdayAttribution } from './matchdayAttribution.js';

test('matchday attribution sends bounded channel labels and an opaque managed QR token', () => {
  const token = 'ab'.repeat(16);
  const address = `https://club.test/club/new?utm_source=LED%20arena&utm_medium=QR&abx_qr=${token}&ticket=1234`;
  assert.deepEqual(matchdayAttribution(address, 'https://social.test/private?id=42'), { channel: 'led', qr: true, arrival: token });
  assert.deepEqual(matchdayAttribution('https://club.test/club/new?abx_qr=invalid', 'https://m.instagram.com/private?id=42'), { channel: 'social', qr: false, arrival: '' });
  assert.equal(matchdayAttribution('https://club.test/club/new?utm_source=unknown-name').channel, 'campaign');
  assert.equal(matchdayAttribution('https://club.test/club/new').channel, 'direct');
});

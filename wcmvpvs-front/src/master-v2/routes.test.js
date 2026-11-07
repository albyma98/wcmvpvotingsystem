import assert from 'node:assert/strict';
import { test } from 'node:test';
import { isMasterWorkspacePath, masterBase, masterSectionPath, masterSections, readMasterRoute } from './routes.js';

test('the new namespace does not claim the legacy Master or fan URLs', () => {
  for (const path of ['/admin/master', '/admin/master/newer', '/sport/new', '/sport/admin/v2']) {
    assert.equal(isMasterWorkspacePath(path), false, path);
    assert.equal(readMasterRoute(path), null, path);
  }
  assert.equal(readMasterRoute(masterBase + '/').section, 'overview');
  for (const section of masterSections) {
    assert.equal(readMasterRoute(masterSectionPath(section.id)).section, section.id);
  }
});

test('deep links map supported entity IDs and tabs to their actual screens', () => {
  assert.equal(readMasterRoute(`${masterBase}/organizations/create`).mode, 'create');
  assert.equal(readMasterRoute(`${masterBase}/organizations/17/edit`).mode, 'edit');
  assert.equal(readMasterRoute(`${masterBase}/organizations/17/arena`).tab, 'arena');
  assert.equal(readMasterRoute(`${masterBase}/clubs/9/domains`).id, 9);
  assert.equal(readMasterRoute(`${masterBase}/tournaments/20`).mode, 'detail');
  assert.equal(readMasterRoute(`${masterBase}/arenas/32/edit`).id, 32);
  for (const path of ['/clubs/1/edit', '/tournaments/0', '/arenas/1.5', '/organizations/17/arena/extra', '/qr-redirects/create', '/organizations/9007199254740993']) {
    assert.equal(readMasterRoute(masterBase + path), null, path);
  }
});

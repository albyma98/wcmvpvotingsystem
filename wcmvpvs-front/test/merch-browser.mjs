// In-memory browser smoke test. Run with MERCH_TEST_BROWSER set to a local Chromium or Edge executable.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { once } from 'node:events';
import { setTimeout as delay } from 'node:timers/promises';
import { createServer } from 'vite';

const executable = process.env.MERCH_TEST_BROWSER;
assert.ok(executable, 'Set MERCH_TEST_BROWSER to a local Chromium/Edge executable');
const base = '/api/v1/clubs/merch-fixture/merch';
const token = 'a'.repeat(64);
const receivedBodies = [];
const settings = {
  enabled: true, visible: true, online_enabled: true, offline_enabled: true,
  default_mode: 'offline', pickup_enabled: true, shipping_enabled: false, contact_requirement: 'email',
  pickup_point: 'Palazzetto di prova', payment_instructions: 'Pagamento alla cassa',
};
const products = [{
  id: 'fixture-shirt', name: 'Maglia ufficiale di prova', category: 'Abbigliamento', featured: true,
  stock: 2, max_quantity: 2, price_cents: 3200, online_enabled: true, offline_enabled: true,
  variants: [{ id: 'm', size: 'M', color: 'Blu', stock: 2 }, { id: 'l', size: 'L', color: 'Blu', stock: 0 }],
}];
const quote = {
  items: [{ product_id: 'fixture-shirt', variant_id: 'm', quantity: 1, name: products[0].name, variant_name: 'M / Blu', unit_cents: 3200 }],
  subtotal_cents: 3200, shipping_cents: 0, total_cents: 3200,
};
const order = {
  ...quote, id: 'fixture-order', code: 'FIXTURE-123', mode: 'offline', payment_status: 'UNPAID_OFFLINE',
  fulfillment: 'pickup', fulfillment_status: 'PENDING', pickup_point: settings.pickup_point,
  payment_instructions: settings.payment_instructions, created_at: 1785000000, expires_at: 1785100000,
};

function reply(response, status, body) {
  response.writeHead(status, { 'content-type': 'application/json' });
  response.end(JSON.stringify(body));
}

const fixture = {
  name: 'merch-test-fixture',
  configureServer(server) {
    server.middlewares.use(async (request, response, next) => {
      const route = new URL(request.url, 'http://localhost').pathname;
      if (!route.startsWith('/api/')) return next();
      if (route === '/api/v1/club-domain') return reply(response, 204);
      if (route === '/api/organization-theme') return reply(response, 200, { name: 'Società di prova', primary_color: '#234eb0' });
      if (route === '/api/fan/me') return reply(response, 200, { registered: false });
      if (route === base) return reply(response, 200, { settings, products });
      if (route === base + '/events') return reply(response, 204);
      if (route === base + '/quote') return reply(response, 200, quote);
      if (route === base + '/orders') {
        if (request.method === 'GET') return reply(response, 200, { orders: [] });
        const chunks = [];
        for await (const chunk of request) chunks.push(chunk);
        receivedBodies.push(JSON.parse(Buffer.concat(chunks).toString()));
        return reply(response, 201, { order, access_token: token });
      }
      if (route === base + '/orders/' + order.id && request.headers['x-merch-access'] === token) {
        return reply(response, 200, { order });
      }
      return reply(response, 404, { message: 'Fixture non disponibile' });
    });
  },
};

async function waitFor(check, label) {
  for (let tries = 0; tries < 480; tries++) {
    const result = await check();
    if (result) return result;
    await delay(125);
  }
  throw new Error(`Timeout: ${label}`);
}

const profile = await mkdtemp(path.join(tmpdir(), 'arenaboost-merch-'));
const server = await createServer({
  root: fileURLToPath(new URL('../', import.meta.url)),
  plugins: [fixture],
  server: { host: '127.0.0.1', port: 0 },
});
let browser;
let socket;
try {
  await server.listen();
  const origin = 'http://127.0.0.1:' + server.httpServer.address().port;
  browser = spawn(executable, [
    '--headless=new', '--disable-gpu', '--no-first-run', '--disable-background-networking',
    '--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE localhost, EXCLUDE 127.0.0.1',
    '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank',
  ], { windowsHide: true, stdio: 'ignore' });
  let launchError;
  browser.on('error', error => { launchError = error; });
  const port = await waitFor(async () => {
    if (launchError) throw launchError;
    if (browser.exitCode !== null) throw new Error(`Browser stopped with code ${browser.exitCode}`);
    try { return (await readFile(path.join(profile, 'DevToolsActivePort'), 'utf8')).split(/\r?\n/)[0]; }
    catch { return ''; }
  }, 'browser launch');
  const tabs = await (await fetch('http://127.0.0.1:' + port + '/json/list')).json();
  socket = new WebSocket(tabs.find(tab => tab.type === 'page').webSocketDebuggerUrl);
  await once(socket, 'open');
  const pending = new Map();
  let sequence = 0;
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data);
    const action = pending.get(message.id);
    if (!action) return;
    pending.delete(message.id);
    if (message.error) action.reject(Error(message.error.message));
    else action.resolve(message.result);
  });
  function send(method, params = {}) {
    const id = ++sequence;
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      socket.send(JSON.stringify({ id, method, params }));
    });
  }
  async function run(expression) {
    const result = await send('Runtime.evaluate', { expression, returnByValue: true, userGesture: true });
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text);
    return result.result.value;
  }
  const shop = origin + '/merch-fixture/new/shop';
  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  const navigation = await send('Page.navigate', { url: shop });
  assert.equal(navigation.errorText, undefined);
  try {
    await waitFor(() => run('Boolean(document.querySelector(".product-card"))'), 'mobile catalog');
  } catch (error) {
    throw new Error(error.message + ' · ' + await run('document.body?.innerText.slice(0, 600)'));
  }
  assert.equal(await run('document.documentElement.scrollWidth === innerWidth'), true);
  await run('document.querySelector(".product-card").click()');
  await waitFor(() => run('Boolean(document.querySelector("dialog[open]"))'), 'product bottom sheet');
  assert.equal(await run('Math.abs(document.querySelector("dialog").getBoundingClientRect().bottom - innerHeight) < 2'), true);
  assert.equal(await run('document.querySelector(".product-detail .primary").disabled'), true);
  await run('document.querySelector(".variant-list button:not(:disabled)").click()');
  await run('document.querySelector(".product-detail .primary").click()');
  await waitFor(() => run('Boolean(document.querySelector(".quote .grand-total"))'), 'offline quote');
  await run(`
    for (const [selector, value] of [['[autocomplete=name]', 'Tifoso di prova'], ['input[type=email]', 'tifoso@example.invalid']]) {
      const input = document.querySelector(selector);
      input.value = value;
      input.dispatchEvent(new Event('input', { bubbles: true }));
    }
    document.querySelector('.checkout-form button[type=submit]').click();
  `);
  await waitFor(() => run('document.body.innerText.includes("Prenotazione ricevuta")'), 'unpaid receipt');
  assert.equal(await run('document.body.innerText.includes("Pagamento da effettuare al club")'), true);
  assert.equal(receivedBodies.length, 1);
  assert.equal(receivedBodies[0].expected_total_cents, 3200);
  assert.equal(receivedBodies[0].items[0].variant_id, 'm');
  await send('Emulation.setDeviceMetricsOverride', { width: 1200, height: 850, deviceScaleFactor: 1, mobile: false });
  await send('Page.navigate', { url: shop + '/fixture-shirt' });
  await waitFor(() => run('Boolean(document.querySelector("dialog[open]"))'), 'desktop product deep link');
  assert.equal(await run('document.querySelector("dialog").getBoundingClientRect().width < innerWidth'), true);
  console.log('Browser merch QA passed: mobile guest reservation and desktop product dialog.');
} finally {
  socket?.close();
  if (browser && browser.exitCode === null && !browser.killed) browser.kill();
  await server.close();
  const resolved = path.resolve(profile);
  if (path.dirname(resolved) === path.resolve(tmpdir()) && path.basename(resolved).startsWith('arenaboost-merch-')) {
    await rm(resolved, { recursive: true, force: true, maxRetries: 3 }).catch(() => {});
  }
}

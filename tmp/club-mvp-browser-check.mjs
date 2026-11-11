import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { join, extname, resolve } from 'node:path'
import { setTimeout as delay } from 'node:timers/promises'

const profile = await mkdtemp(join(process.env.TEMP, 'club-mvp-browser-'))
const dist = resolve('wcmvpvs-front/dist')
const players = Array.from({ length: 12 }, (_, index) => ({
  id: index + 1, firstName: 'Maria Luisa', lastName: index ? 'De Luca' : 'Rossi Della Rovere', jerseyNumber: index + 1, role: 'Schiacciatrice',
}))
const match = { id: 31, status: 'live', home: true, revision: 1, opponent: 'Ospiti', scoreFor: 1, scoreAgainst: 0, pointsFor: 9, pointsAgainst: 6, bestOf: 3, setsJson: '["25-22"]' }
const club = { name: 'Aurora Volley', primaryColor: '#203561', secondaryColor: '#f59e0b' }
const ballot = { club, team: { name: 'Under 19' }, match, players }
const sockets = new Set()
const adminStreams = new Set()
const roster = [...players, ...players.slice(0, 6).map((player, index) => ({ ...player, id: 13 + index, jerseyNumber: 13 + index }))]
const admin = { match, team: ballot.team, roster, callups: [], enabled: false, open: false, canOpen: false, totalVotes: 0, playersVoted: 0 }
let voted
let closed = false
let posts = 0
let customDomain = false
const server = createServer(async (req, res) => {
  const path = new URL(req.url, 'http://localhost').pathname
  const json = (code, data) => { res.writeHead(code, { 'Content-Type': 'application/json' }); res.end(JSON.stringify(data)) }
  if (path === '/api/v1/club-domain') return customDomain
    ? json(200, { clubSlug: 'aurora', primaryHostname: '127.0.0.1' })
    : res.writeHead(204).end()
  if (path.startsWith('/api/v1/club-admin/aurora')) {
    const team = { id: 1, name: 'Under 19', active: true }
    if (path.endsWith('/overview')) return json(200, { club, teams: [team], matches: [{ ...match, teamId: 1 }], counts: {} })
    if (path.endsWith('/teams')) return json(200, { teams: [team], players: { 1: roster } })
    if (path.endsWith('/mvp/stream')) {
      res.writeHead(200, { 'Content-Type': 'text/event-stream' })
      adminStreams.add(res)
      res.on('close', () => adminStreams.delete(res))
      return res.write('data: {}\n\n')
    }
    if (path.endsWith('/mvp')) {
      if (req.method === 'PUT') {
        let body = ''
        for await (const chunk of req) body += chunk
        const change = JSON.parse(body)
        if (change.playerIds) {
          admin.callups = roster.filter(player => change.playerIds.includes(player.id)).map(player => ({ ...player, eligible: true, votes: 0 }))
          admin.canOpen = Boolean(admin.callups.length)
        }
        if (change.enabled !== undefined) admin.open = admin.enabled = change.enabled
      }
      return json(200, admin)
    }
  }
  if (path.endsWith('/mvp/vote')) {
    posts++
    let body = ''
    for await (const chunk of req) body += chunk
    voted = players.find(player => player.id === JSON.parse(body).playerId)
    return json(201, { voted })
  }
  if (path.startsWith('/api/') && path.endsWith('/mvp')) return closed ? json(409, { error: 'mvp_closed' }) : json(200, { ...ballot, voted })
  if (path.endsWith('/home')) return json(200, { club, teams: [], posts: [], gallery: [], matches: [], sponsors: [] })
  if (path.startsWith('/api/')) return json(200, {})
  const file = path.startsWith('/assets/') ? join(dist, path) : join(dist, 'index.html')
  const types = { '.css': 'text/css', '.js': 'application/javascript', '.html': 'text/html' }
  try { res.writeHead(200, { 'Content-Type': types[extname(file)] || 'application/octet-stream' }).end(await readFile(file)) }
  catch { res.end() }
})
const message = data => {
  const payload = Buffer.from(JSON.stringify(data))
  const header = payload.length < 126 ? Buffer.from([0x81, payload.length]) : Buffer.from([0x81, 126, payload.length >> 8, payload.length & 255])
  return Buffer.concat([header, payload])
}
server.on('upgrade', (req, socket) => {
  const accept = createHash('sha1').update(req.headers['sec-websocket-key'] + '258EAFA5-E914-47DA-95CA-C5AB0DC85B11').digest('base64')
  socket.write('HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: ' + accept + '\r\n\r\n')
  sockets.add(socket)
  socket.on('close', () => sockets.delete(socket))
  socket.on('error', () => sockets.delete(socket))
  socket.write(message({ matches: [match] }))
})
await new Promise((resolve, reject) => { server.on('error', reject); server.listen(0, '127.0.0.1', resolve) })
const base = `http://127.0.0.1:${server.address().port}`
const browser = spawn('C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', [
  '--headless=new', '--no-first-run', '--disable-extensions', '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank',
], { stdio: 'ignore', windowsHide: true })
let launchError
browser.on('error', error => { launchError = error })

async function until(fn, detail) {
  for (let attempt = 0; attempt < 100; attempt++) {
    const result = await fn()
    if (result) return result
    await delay(100)
  }
  throw new Error(detail)
}

let ws
try {
  const port = await until(async () => {
    if (launchError) throw launchError
    try { return (await readFile(join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0] } catch { return null }
  }, 'Edge did not expose DevTools')
  const targets = await fetch(`http://127.0.0.1:${port}/json/list`).then(res => res.json())
  ws = new WebSocket(targets.find(target => target.type === 'page').webSocketDebuggerUrl)
  await new Promise((resolve, reject) => { ws.addEventListener('open', resolve); ws.addEventListener('error', reject) })
  let sequence = 0
  const pending = new Map()
  ws.addEventListener('message', event => {
    const { id, error, result } = JSON.parse(event.data)
    if (pending.has(id)) {
      const { resolve, reject } = pending.get(id)
      pending.delete(id)
      if (error) reject(error)
      else resolve(result)
    }
  })
  const cdp = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++sequence
    pending.set(id, { resolve, reject })
    ws.send(JSON.stringify({ id, method, params }))
  })
  const evaluate = async expression => {
    const { result, exceptionDetails } = await cdp('Runtime.evaluate', { expression, returnByValue: true })
    if (exceptionDetails) throw new Error(exceptionDetails.exception?.description || exceptionDetails.text)
    return result.value
  }
  for (const width of [320, 280, 768]) {
    await cdp('Emulation.setDeviceMetricsOverride', { width, height: 568, deviceScaleFactor: 1, mobile: true })
    await cdp('Page.navigate', { url: base + '/club/aurora/mvp' })
    await until(() => evaluate('document.querySelectorAll(".mvp-card").length === 12'), 'The ballot did not render')
    const layout = await evaluate(`(() => {
      const cards = document.querySelectorAll('.mvp-card')
      const button = document.querySelector('.mvp-confirm').getBoundingClientRect()
      return { width: innerWidth, documentWidth: document.documentElement.scrollWidth, sameRow: cards[0].offsetTop === cards[1].offsetTop, bottom: button.bottom, height: innerHeight, loadedApp: performance.getEntriesByType('resource').some(entry => entry.name.includes('/assets/App-')) }
    })()`)
    assert.equal(layout.width, width)
    assert.equal(layout.documentWidth, width)
    assert.equal(layout.sameRow, width !== 280)
    assert.ok(layout.bottom <= layout.height, JSON.stringify(layout))
    assert.equal(layout.loadedApp, false)
    if (width === 320) {
      const shot = await cdp('Page.captureScreenshot', { format: 'png' })
      await writeFile('tmp/club-mvp-320.png', Buffer.from(shot.data, 'base64'))
    }
    console.log(`Mobile ballot at ${width}px: 12 cards, visible submit button, no overflow`)
  }
  await evaluate('document.querySelector(".mvp-card input").click()')
  await until(() => evaluate('!document.querySelector(".mvp-confirm").disabled'), 'Selecting a card did not enable confirmation')
  await evaluate('document.querySelector(".mvp-confirm").click()')
  await until(() => evaluate('document.querySelector(".mvp-receipt h1")?.textContent === "Voto registrato!"'), 'The confirmation did not render')
  assert.equal(posts, 1)
  await cdp('Page.navigate', { url: base + '/club/aurora/mvp/31' })
  await until(() => evaluate('document.querySelector(".mvp-receipt h1")?.textContent === "Hai già votato"'), 'The restored receipt did not render')
  await until(() => sockets.size > 0, 'The score WebSocket was not connected')
  closed = true
  for (const socket of sockets) socket.write(message({ match: { ...match, status: 'completed', revision: 2 } }))
  await until(() => evaluate('location.pathname === "/club/aurora"'), 'The WebSocket completion did not return to the club')
  await cdp('Page.navigate', { url: base + '/club/aurora/mvp' })
  await until(() => evaluate('location.pathname === "/club/aurora"'), 'The closed QR did not return to the club')
  console.log('Confirmed vote, restored receipt, live closure and QR fallback')
  await cdp('Emulation.setDeviceMetricsOverride', { width: 320, height: 568, deviceScaleFactor: 1, mobile: true })
  await cdp('Page.navigate', { url: base + '/club-admin/aurora' })
  async function click(text) {
    await until(() => evaluate(`(() => {
      const button = [...document.querySelectorAll('button')].find(item => item.textContent.trim().startsWith(${JSON.stringify(text)}) && !item.disabled);
      if (!button) return false;
      button.click();
      return true;
    })()`), `Missing button: ${text}`)
  }
  await click('Gestisci il calendario')
  await click('MVP Light')
  await click('Configura convocati')
  await click('Seleziona tutti')
  await until(() => evaluate('document.querySelectorAll(".mvp-admin-list input:checked").length === 18'), 'Select all did not select 18 athletes')
  for (let index = 12; index < 18; index++) await evaluate(`document.querySelectorAll(".mvp-admin-list input")[${index}].click()`)
  await click('Salva convocati')
  await until(() => admin.callups.length === 12, 'The admin did not save the 12 call-ups')
  await click('Attiva votazione')
  await click('Classifica live')
  await until(() => adminStreams.size > 0 && admin.enabled, 'The admin did not activate voting or connect the stream')
  admin.totalVotes = admin.playersVoted = 1
  Object.assign(admin.callups[0], { votes: 1, percentage: 100, leader: true })
  for (const stream of adminStreams) stream.write('data: {}\n\n')
  await until(() => evaluate('document.querySelector(".mvp-ranking progress")?.value === 100'), 'The SSE update did not refresh the ranking')
  assert.equal(await evaluate('document.documentElement.scrollWidth'), 320)
  console.log('Admin at 320px: selected and saved 12 of 18, activated voting, and received the live ranking update')
  customDomain = true
  closed = false
  voted = undefined
  await cdp('Page.navigate', { url: base + '/mvp' })
  await until(() => evaluate('document.querySelectorAll(".mvp-card").length === 12'), 'The custom-domain QR did not show the ballot')
  closed = true
  await cdp('Page.navigate', { url: base + '/mvp/31' })
  await until(() => evaluate('location.pathname === "/"'), 'The custom-domain direct link did not return to the home')
  console.log('Custom-domain /mvp routing and home fallback')
} finally {
  ws?.close()
  browser.kill()
  for (const socket of sockets) socket.destroy()
  for (const stream of adminStreams) stream.end()
  server.close()
}

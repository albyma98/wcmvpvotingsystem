import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import * as clubHub from '../clubHub.js'

const matchCard = loadComponent('./ClubMatchCard.vue', 'club-match-card-test', { vue: Vue, '../clubHub': clubHub })
const imports = {
  vue: Vue,
  '../clubHub': clubHub,
  '../clubHub.js': clubHub,
  './ClubMatchCard': matchCard,
  './ClubMatchCard.vue': matchCard,
}
function loadComponent(filename, id, imports) {
  const { descriptor } = parse(readFileSync(new URL(filename, import.meta.url), 'utf8'))
  const script = compileScript(descriptor, { id })
  const source = script.content
    .replace(/^import\s+([\s\S]*?)\s+from\s+(['"])([^'"]+)\2;?\r?\n/gm, (_, binding, quote, name) =>
      `const ${binding.trim().replace(/\s+as\s+/g, ': ')} = imports[${JSON.stringify(name)}];\n`)
    .replace('export default', 'return')
  const component = new Function('imports', source)(imports)
  const template = compileTemplate({
    source: descriptor.template.content,
    id,
    compilerOptions: { bindingMetadata: script.bindings },
  })
  assert.deepEqual(template.errors, [])
  const helpers = {}
  const renderSource = template.code.replace(/^import \{([^}]+)\} from "vue"\r?$/gm, (_, bindings) => {
    for (const binding of bindings.split(',')) {
      const [name, alias = name] = binding.trim().split(/\s+as\s+/)
      helpers[alias] = Vue[name]
    }
    return ''
  }).replace('export function render', 'return function render')
  component.render = new Function(...Object.keys(helpers), renderSource)(...Object.values(helpers))
  return component
}
const lightbox = loadComponent('./ClubGalleryLightbox.vue', 'club-gallery-lightbox-test', { vue: Vue })
const component = loadComponent('./ClubContentModal.vue', 'club-content-modal-test', { ...imports, './ClubGalleryLightbox.vue': lightbox })

// A minimal Vue host exercises the real rendered labels, links and click handlers.
// Native dialog methods are supplied because no browser DOM is required here.
function hostNode(tag, text = '') {
  return Vue.markRaw({
    tag, text, props: {}, children: [], parent: null, open: false,
    showModal() { this.open = true },
    close() { this.open = false },
    focus() {},
    querySelector() { return null },
    querySelectorAll() { return [] },
    getBoundingClientRect() { return { left: 24, top: 24, right: 800, bottom: 700 } },
  })
}
const renderer = Vue.createRenderer({
  createElement: tag => hostNode(tag),
  createText: text => hostNode('#text', text),
  createComment: text => hostNode('#comment', text),
  setText: (node, text) => { node.text = text },
  setElementText: (node, text) => { node.text = text; node.children = [] },
  parentNode: node => node.parent,
  nextSibling: node => node.parent?.children[node.parent.children.indexOf(node) + 1] ?? null,
  patchProp: (node, key, previous, value) => { node.props[key] = value },
  insert(node, parent, anchor = null) {
    if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1)
    const index = anchor ? parent.children.indexOf(anchor) : -1
    parent.children.splice(index < 0 ? parent.children.length : index, 0, node)
    node.parent = parent
  },
  remove(node) {
    if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1)
    node.parent = null
  },
})

function nodes(root, predicate) {
  return [root, ...root.children.flatMap(child => nodes(child, () => true))].filter(predicate)
}
function textOf(node) {
  return node.tag === '#comment' ? '' : node.text + node.children.map(textOf).join('')
}
const find = (root, predicate) => nodes(root, predicate)[0]
const click = node => node.props.onClick({ target: node, currentTarget: node })

function fixture(t, overrides = {}) {
  const team = {
    name: 'Under 18 Aurora Volley', category: 'Under 18', championship: 'Campionato regionale',
    description: 'Descrizione duplicata da non mostrare sopra il menu', logoUrl: '/team-logo.png',
  }
  const props = Vue.reactive({
    panel: 'team',
    club: { name: 'Club Aurora', contacts: [{ name: 'Segreteria', role: 'Informazioni', phone: '+39 (333) 123-4567', email: 'info@aurora.test', address: 'Via dello Sport 10', website: 'https://aurora.test' }] },
    data: {
      matches: [{ id: 1, opponent: 'Ospiti club', scheduledAt: '2026-10-25T15:00:00Z' }],
      posts: [{ id: 2, kind: 'announcement', title: 'Avviso del club', body: 'Testo completo avviso' }, { id: 3, kind: 'news', title: 'Notizia del club', excerpt: 'Estratto notizia' }],
      gallery: [{ id: 4, imageUrl: '/club-photo.jpg', title: 'Foto del club' }],
    },
    teamData: {
      team,
      matches: [{ id: 11, opponent: 'Ospiti squadra', scheduledAt: '2026-10-26T15:00:00Z', status: 'scheduled' }],
      players: [{ id: 12, firstName: 'Alessandro', lastName: 'Della Valle Campolungo', jerseyNumber: 17, role: 'Palleggiatore' }],
      posts: [{ id: 13, kind: 'news', title: 'Notizia senza immagine', excerpt: 'Il testo resta disponibile senza fotografia' }],
      gallery: [{ id: 14, imageUrl: '/team-photo.jpg', title: 'Foto della squadra', caption: 'In campo' }],
    },
    teamTitle: team.name,
    teamLoading: false,
    teamError: '',
    section: 'overview',
    ...overrides,
  })
  const events = []
  props.onClose = () => events.push(['close'])
  props['onUpdate:section'] = section => events.push(['update:section', section])
  const root = hostNode('root')
  const app = renderer.createApp({ render: () => Vue.h(component, props) })
  app.mount(root)
  t.after(() => app.unmount())
  return { root, props, events }
}

test('team modal has one title beside close and a menu without the repeated team introduction', t => {
  const { root, events } = fixture(t)
  const dialog = find(root, node => node.tag === 'dialog')
  assert.ok(dialog)
  const header = find(dialog, node => node.tag === 'header')
  assert.equal(textOf(find(header, node => node.tag === 'h2')), 'Under 18 Aurora Volley')
  assert.equal(find(header, node => node.tag === 'h2').props.id, 'club-content-title')
  assert.equal(nodes(header, node => node.tag === 'button').length, 1)
  const close = find(header, node => node.tag === 'button')
  assert.equal(close.props['aria-label'], 'Chiudi')
  assert.ok(!textOf(header).includes('Club Aurora'))
  assert.ok(!textOf(dialog).includes('Descrizione duplicata'))
  assert.ok(!nodes(dialog, node => node.tag === 'img').some(node => node.props.src === '/team-logo.png'))
  const nav = find(dialog, node => node.props['aria-label'] === 'Sezioni squadra')
  const buttons = nodes(nav, node => node.tag === 'button')
  assert.deepEqual(buttons.map(textOf), ['Panoramica', 'Partite', 'Roster', 'News', 'Gallery'])
  assert.deepEqual(buttons.map(node => node.props['aria-current'] ?? null), ['page', null, null, null, null])
  click(buttons[2])
  click(close)
  assert.deepEqual(events, [['update:section', 'roster'], ['close']])
})

test('loading and error states retain the selected team title and expose no premature menu', async t => {
  const { root, props } = fixture(t, { teamData: null, teamTitle: 'Prima squadra femminile', teamLoading: true })
  assert.equal(textOf(find(root, node => node.tag === 'h2')), 'Prima squadra femminile')
  assert.ok(!find(root, node => node.props['aria-label'] === 'Sezioni squadra'))
  props.teamLoading = false
  props.teamError = 'Dettagli temporaneamente non disponibili'
  await Vue.nextTick()
  assert.ok(textOf(root).includes(props.teamError))
  assert.equal(textOf(find(root, node => node.tag === 'h2')), 'Prima squadra femminile')
  assert.ok(!find(root, node => node.props['aria-label'] === 'Sezioni squadra'))
})

test('native dialog opens and dismisses through Escape or the backdrop without closing for content clicks', t => {
  const { root, events } = fixture(t)
  const dialog = find(root, node => node.tag === 'dialog')
  assert.equal(dialog.open, true)
  const title = find(dialog, node => node.tag === 'h2')
  dialog.props.onClick({ target: title, clientX: 40, clientY: 40 })
  dialog.props.onClick({ target: dialog, clientX: 100, clientY: 100 })
  assert.deepEqual(events, [])
  dialog.props.onClick({ target: dialog, clientX: 5, clientY: 100 })
  let prevented = false
  dialog.props.onCancel({ preventDefault() { prevented = true } })
  assert.equal(prevented, true)
  assert.deepEqual(events, [['close'], ['close']])
})

test('overview statistics navigate to the requested section and show the featured match', t => {
  const { root, events } = fixture(t)
  const overview = find(root, node => node.props['aria-label'] === 'Panoramica squadra')
  assert.ok(textOf(overview).includes('Ospiti squadra'))
  const buttons = nodes(overview, node => node.tag === 'button')
  assert.deepEqual(buttons.map(node => textOf(find(node, item => item.tag === 'span'))), ['Partite', 'Atleti', 'Notizie', 'Foto'])
  for (const button of buttons) {
    assert.equal(textOf(find(button, node => node.tag === 'b')), '1')
    click(button)
  }
  assert.deepEqual(events, [
    ['update:section', 'matches'], ['update:section', 'roster'],
    ['update:section', 'news'], ['update:section', 'gallery'],
  ])
})

test('club and team scoreboards show the society name with correct home and away scores', async t => {
  for (const [panel, section] of [['matches', 'matches'], ['team', 'matches'], ['team', 'overview']]) {
    const { root, props } = fixture(t, { panel, section })
    const match = (panel === 'team' ? props.teamData : props.data).matches[0]
    Object.assign(match, { status: 'completed', scoreFor: 3, scoreAgainst: 1 })
    for (const home of [true, false]) {
      match.home = home
      await Vue.nextTick()
      const scoreboard = find(root, node => node.props.class === 'club-match-scoreboard')
      const names = nodes(scoreboard, node => node.tag === 'strong').map(textOf)
      assert.deepEqual(names, home ? ['Club Aurora', match.opponent] : [match.opponent, 'Club Aurora'])
      assert.deepEqual(nodes(scoreboard, node => node.tag === 'b').map(textOf), home ? ['3', '1'] : ['1', '3'])
      assert.ok(!textOf(scoreboard).includes('Under 18'))
    }
    props.club.name = 'Aurora Volley'
    await Vue.nextTick()
    assert.ok(textOf(find(root, node => node.props.class === 'club-match-scoreboard')).includes('Aurora Volley'))
  }
})

test('switching controlled team sections renders matches, full roster names, news without images and photos', async t => {
  const { root, props } = fixture(t)
  for (const [section, expected] of [
    ['matches', 'Ospiti squadra'],
    ['roster', 'Alessandro Della Valle Campolungo'],
    ['news', 'Notizia senza immagine'],
    ['gallery', 'Foto della squadra'],
  ]) {
    props.section = section
    await Vue.nextTick()
    assert.ok(textOf(root).includes(expected), `${section} content missing`)
    const nav = find(root, node => node.props['aria-label'] === 'Sezioni squadra')
    const selected = nodes(nav, node => node.props['aria-current'] === 'page')
    assert.equal(selected.length, 1)
    if (section === 'roster') assert.ok(textOf(root).includes('17'))
    if (section === 'news') assert.ok(textOf(root).includes('Il testo resta disponibile senza fotografia'))
    if (section === 'gallery') assert.ok(find(root, node => node.tag === 'img' && node.props.src === '/team-photo.jpg'))
  }
})

test('empty team sections and sponsor panels retain useful explanatory content', async t => {
  const { root, props } = fixture(t, { teamData: { team: { name: 'Categoria' }, matches: [], players: [], posts: [], gallery: [] } })
  for (const [section, label] of [
    ['matches', 'Calendario e risultati'], ['roster', 'Roster squadra'],
    ['news', 'Notizie'], ['gallery', 'Foto del club'],
  ]) {
    props.section = section
    await Vue.nextTick()
    const content = find(root, node => node.tag === 'section' && node.props['aria-label'] === label)
    assert.ok(find(content, node => node.tag === 'h3'), `${section} empty heading missing`)
    assert.ok(textOf(find(content, node => node.tag === 'p')).trim(), `${section} empty explanation missing`)
  }
  props.panel = 'sponsors'
  await Vue.nextTick()
  assert.equal(textOf(find(root, node => node.tag === 'h2')), 'I nostri Sponsor')
  assert.match(textOf(root), /sponsor saranno disponibili/)
})

test('club panels render their own content and useful contact links', async t => {
  const { root, props } = fixture(t, { panel: 'contacts', teamData: null })
  const links = nodes(root, node => node.tag === 'a').map(node => node.props.href)
  assert.ok(links.includes('tel:+393331234567'))
  assert.ok(links.includes('mailto:info@aurora.test'))
  assert.ok(links.includes('https://aurora.test'))
  assert.ok(textOf(root).includes('Via dello Sport 10'))
  for (const [panel, expected] of [
    ['matches', 'Ospiti club'],
    ['announcements', 'Avviso del club'],
    ['news', 'Notizia del club'],
    ['gallery', 'Foto del club'],
  ]) {
    props.panel = panel
    await Vue.nextTick()
    assert.ok(textOf(root).includes(expected), `${panel} content missing`)
    assert.ok(!find(root, node => node.props['aria-label'] === 'Sezioni squadra'))
  }
})

test('club and category calendars show nearer dates before later dates and react to edits', async t => {
  for (const panel of ['matches', 'team']) {
    const { root, props } = fixture(t, { panel, section: 'matches' })
    const payload = panel === 'team' ? props.teamData : props.data
    payload.matches = [
      { id: 3, scheduledAt: '2026-11-15T18:00:00Z', opponent: 'Terza', status: 'scheduled' },
      { id: 1, scheduledAt: '2026-10-28T18:00:00Z', opponent: 'Prima', status: 'completed' },
      { id: 2, scheduledAt: '2026-10-31T18:00:00Z', opponent: 'Seconda', status: 'scheduled' },
    ]
    await Vue.nextTick()
    const dates = () => nodes(root, node => node.tag === 'time').map(node => node.props.datetime)
    assert.deepEqual(dates(), ['2026-10-28T18:00:00Z', '2026-10-31T18:00:00Z', '2026-11-15T18:00:00Z'])
    assert.deepEqual(payload.matches.map(match => match.id), [3, 1, 2])
    payload.matches[0].scheduledAt = '2026-10-30T18:00:00Z'
    await Vue.nextTick()
    assert.deepEqual(dates(), ['2026-10-28T18:00:00Z', '2026-10-30T18:00:00Z', '2026-10-31T18:00:00Z'])
  }
})

test('category overview prioritizes live games, then the next fixture, then the latest result', async t => {
  const { root, props } = fixture(t)
  props.teamData.matches = [
    { id: 3, scheduledAt: '2026-11-15T18:00:00Z', status: 'scheduled' },
    { id: 1, scheduledAt: '2026-10-20T18:00:00Z', status: 'completed' },
    { id: 2, scheduledAt: '2026-10-31T18:00:00Z', status: 'scheduled' },
  ]
  const featuredDate = () => find(root, node => node.tag === 'time').props.datetime
  await Vue.nextTick()
  assert.equal(featuredDate(), '2026-10-31T18:00:00Z')
  props.teamData.matches[0].status = 'live'
  await Vue.nextTick()
  assert.equal(featuredDate(), '2026-11-15T18:00:00Z')
  props.teamData.matches.forEach(match => { match.status = 'completed' })
  await Vue.nextTick()
  assert.equal(featuredDate(), '2026-11-15T18:00:00Z')
})

test('announcements display attached photos and retain text-only posts', async t => {
  const { root, props } = fixture(t, { panel: 'announcements' })
  props.data.posts = [
    { id: 1, kind: 'announcement', title: 'Avviso con foto', body: 'Dettagli della comunicazione', imageUrl: '/announcement.jpg', pinned: true },
    { id: 2, kind: 'announcement', title: 'Avviso senza foto', excerpt: 'Comunicazione testuale' },
    { id: 3, kind: 'news', title: 'Notizia separata', imageUrl: '/news.jpg' },
  ]
  await Vue.nextTick()
  const section = find(root, node => node.props['aria-label'] === 'Annunci del club')
  const posts = nodes(section, node => node.tag === 'article')
  assert.equal(posts.length, 2)
  const photo = find(posts[0], node => node.tag === 'img')
  assert.equal(photo.props.src, '/announcement.jpg')
  assert.equal(photo.props.alt, 'Avviso con foto')
  assert.equal(photo.props.loading, 'lazy')
  assert.match(textOf(posts[0]), /In evidenza.*Dettagli della comunicazione/)
  assert.ok(!find(posts[1], node => node.tag === 'img'))
  assert.match(textOf(posts[1]), /Comunicazione testuale/)
  props.data.posts[0].imageUrl = ''
  await Vue.nextTick()
  assert.equal(nodes(section, node => node.tag === 'img').length, 0)
  assert.match(textOf(posts[0]), /Avviso con foto/)
})

test('sponsor cards display logos and optional descriptions with one external link per sponsor', t => {
  const sponsors = [
    { id: 21, name: 'Sport e Passione', logoUrl: '/sport-logo.png', website: 'https://sport.example', description: 'Al fianco del club.\nDa oltre venti anni.' },
    { id: 22, name: 'Partner senza descrizione', logoUrl: 'https://images.example/partner.webp', website: 'https://partner.example', description: '' },
  ]
  const { root } = fixture(t, { panel: 'sponsors', teamData: null, data: { sponsors } })
  const section = find(root, node => node.props['aria-label'] === 'Sponsor del club')
  assert.ok(section)
  const cards = nodes(section, node => node.tag === 'a')
  assert.equal(cards.length, 2)
  for (const [index, card] of cards.entries()) {
    const sponsor = sponsors[index]
    assert.equal(card.props.href, sponsor.website)
    assert.equal(card.props.target, '_blank')
    assert.equal(card.props.rel, 'noopener noreferrer')
    assert.ok(card.props['aria-label'].includes(sponsor.name))
    assert.ok(card.props['aria-label'].includes('nuova scheda'))
    assert.equal(textOf(find(card, node => node.tag === 'h3')), sponsor.name)
    const logo = find(card, node => node.tag === 'img')
    assert.equal(logo.props.src, sponsor.logoUrl)
    assert.equal(logo.props.alt, sponsor.name)
    assert.equal(logo.props.loading, 'lazy')
    assert.ok(textOf(card).includes('Visita il sito'))
  }
  assert.equal(textOf(find(cards[0], node => node.tag === 'p')), sponsors[0].description)
  assert.ok(!find(cards[1], node => node.tag === 'p'))
  assert.ok(!textOf(root).includes('sponsor saranno disponibili'))
})

test('sponsor modal updates from the empty state when the club payload changes', async t => {
  const { root, props } = fixture(t, { panel: 'sponsors', teamData: null })
  assert.match(textOf(root), /sponsor saranno disponibili/)
  props.data = { sponsors: [{ id: 1, name: 'Nuovo sponsor', logoUrl: '/new.png', website: 'https://new.example', description: '' }] }
  await Vue.nextTick()
  assert.ok(textOf(root).includes('Nuovo sponsor'))
  assert.ok(!textOf(root).includes('sponsor saranno disponibili'))
  props.panel = 'contacts'
  await Vue.nextTick()
  assert.ok(!textOf(root).includes('Nuovo sponsor'))
  assert.ok(textOf(root).includes('Segreteria'))
})

test('club and team gallery thumbnails open the complete image and close without dismissing the gallery', async t => {
  for (const panel of ['gallery', 'team']) {
    const { root, events } = fixture(t, { panel, section: 'gallery' })
    const galleryDialog = find(root, node => node.tag === 'dialog')
    const thumbnail = find(root, node => node.props['aria-haspopup'] === 'dialog')
    assert.match(thumbnail.props['aria-label'], /Ingrandisci immagine:/)
    const expectedSource = panel === 'team' ? '/team-photo.jpg' : '/club-photo.jpg'

    for (const method of ['button', 'escape', 'backdrop']) {
      click(thumbnail)
      await Vue.nextTick()
      const viewer = find(root, node => node.tag === 'dialog' && node !== galleryDialog)
      assert.ok(viewer.open)
      assert.match(viewer.props['aria-label'], /Immagine ingrandita:/)
      assert.equal(find(viewer, node => node.tag === 'img').props.src, expectedSource)
      if (panel === 'team') assert.equal(textOf(find(viewer, node => node.tag === 'figcaption')), 'In campo')
      viewer.props.onClick({ target: viewer, clientX: 100, clientY: 100 })
      viewer.props.onClick({ target: find(viewer, node => node.tag === 'img'), clientX: 100, clientY: 100 })
      await Vue.nextTick()
      assert.ok(viewer.open)

      if (method === 'button') click(find(viewer, node => node.props['aria-label'] === 'Chiudi immagine ingrandita'))
      if (method === 'escape') {
        let prevented = false
        viewer.props.onCancel({ preventDefault() { prevented = true } })
        assert.equal(prevented, true)
      }
      if (method === 'backdrop') viewer.props.onClick({ target: viewer, clientX: 5, clientY: 100 })
      await Vue.nextTick()
      assert.equal(viewer.open, false)
      assert.equal(nodes(root, node => node.tag === 'dialog').length, 1)
      assert.equal(galleryDialog.open, true)
      assert.deepEqual(events, [])
    }
  }
})

test('leaving the gallery clears the enlarged image, and reopening a different photo shows its own content', async t => {
  const { root, props } = fixture(t, { section: 'gallery' })
  click(find(root, node => node.props['aria-haspopup'] === 'dialog'))
  await Vue.nextTick()
  const viewer = nodes(root, node => node.tag === 'dialog')[1]
  props.section = 'roster'
  await Vue.nextTick()
  assert.equal(viewer.open, false)
  props.section = 'gallery'
  props.teamData.gallery.push({ id: 15, imageUrl: '/second-photo.jpg', caption: 'La seconda foto' })
  await Vue.nextTick()
  assert.equal(nodes(root, node => node.tag === 'dialog').length, 1)
  const thumbnails = nodes(root, node => node.props['aria-haspopup'] === 'dialog')
  assert.match(thumbnails[1].props['aria-label'], /La seconda foto/)
  click(thumbnails[1])
  await Vue.nextTick()
  const secondViewer = nodes(root, node => node.tag === 'dialog')[1]
  assert.equal(find(secondViewer, node => node.tag === 'img').props.src, '/second-photo.jpg')
  find(secondViewer, node => node.tag === 'img').props.onError()
  await Vue.nextTick()
  assert.match(textOf(secondViewer), /Impossibile caricare/)
  assert.ok(find(secondViewer, node => node.props['aria-label'] === 'Chiudi immagine ingrandita'))
})

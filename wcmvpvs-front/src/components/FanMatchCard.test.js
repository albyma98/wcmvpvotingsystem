import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parse, compileTemplate } from '@vue/compiler-sfc';
import * as Vue from 'vue';
import * as VueSSR from 'vue/server-renderer';
import { setupSfc, testVue } from '../../test/sfcSetup.js';

const url = new URL('./FanMatchCard.vue', import.meta.url);
const { descriptor } = parse(readFileSync(url, 'utf8'));
const template = compileTemplate({ source: descriptor.template.content, id: 'match-card-test', ssr: true, ssrCssVars: [] });
assert.deepEqual(template.errors, []);
const helpers = {};
const renderCode = template.code.replace(/^import \{([^}]+)\} from "([^"]+)"\r?$/gm, (_, imports, module) => {
  for (const item of imports.split(',')) {
    const [name, alias = name] = item.trim().split(/\s+as\s+/);
    helpers[alias] = (module === 'vue' ? Vue : VueSSR)[name];
  }
  return '';
}).replace('export function ssrRender', 'return function ssrRender');
const ssrRender = new Function(...Object.keys(helpers), renderCode)(...Object.values(helpers));

test('shop and collection card sizes select the volleyball court from the society sport and keep match details', async t => {
  for (const compact of [false, true]) {
    const { state, props, scope } = setupSfc(url, {
      vue: testVue,
      '../assets/fan-collectibles/match-card.webp': '/assets/basketball.webp',
      '../assets/fan-collectibles/match-card-volleyball.webp': '/assets/volleyball.webp',
    }, {
      sport: 'Pallavolo', compact, clubName: 'Volley Club', clubLogoUrl: '/club.png',
      memory: { club_name: 'Volley Club', opponent: 'Avversaria', event_date: '2026-10-25T18:00:00Z' },
    });
    t.after(() => scope.stop());
    const html = () => VueSSR.renderToString(Vue.createSSRApp({ setup: () => ({ ...Vue.proxyRefs(state), ...props }), ssrRender }));
    let rendered = await html();
    assert.match(rendered, /src="\/assets\/volleyball.webp"/);
    assert.doesNotMatch(rendered, /basketball.webp/);
    assert.match(rendered, /Volley Club/);
    assert.match(rendered, /Avversaria/);
    assert.match(rendered, /datetime="2026-10-25"/);
    assert.match(rendered, /src="\/club.png"/);
    for (const sport of ['Pallacanestro', '', null]) {
      props.sport = sport;
      rendered = await html();
      assert.match(rendered, /src="\/assets\/basketball.webp"/);
      assert.doesNotMatch(rendered, /volleyball.webp/);
    }
    props.sport = ' Pallavolo ';
    assert.match(await html(), /src="\/assets\/volleyball.webp"/);
  }
});

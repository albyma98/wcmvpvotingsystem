import { readFileSync } from 'node:fs';
import { parse, compileScript } from '@vue/compiler-sfc';
import * as Vue from 'vue';

// Load actual component logic with HTTP and lifecycle dependencies supplied by each test.
export function setupSfc(url, imports, initialProps, globals = {}) {
  const { descriptor } = parse(readFileSync(url, 'utf8'));
  const source = compileScript(descriptor, { id: 'sfc-behavior-test' }).content
    .replace(/^import\s+([\s\S]*?)\s+from\s+(['"])([^'"]+)\2;?\r?\n/gm, (_, binding, quote, name) =>
      `const ${binding.trim()} = imports[${JSON.stringify(name)}];\n`)
    .replace('export default', 'return');
  const component = new Function('imports', ...Object.keys(globals), source)(imports, ...Object.values(globals));
  const props = Vue.reactive(initialProps);
  const events = [];
  const scope = Vue.effectScope();
  const state = scope.run(() => component.setup(props, { expose() {}, emit: (...args) => events.push(args) }));
  return { state, props, scope, events };
}

export function testWindow(storage = new Map()) {
  const timers = new Map();
  let nextTimer = 1;
  return {
    storage, timers,
    localStorage: { getItem: (key) => storage.get(key) ?? null, setItem: (key, value) => storage.set(key, value) },
    setTimeout: (callback) => { const id = nextTimer++; timers.set(id, callback); return id; },
    clearTimeout: (id) => timers.delete(id),
    setInterval: () => nextTimer++, clearInterval() {},
    addEventListener() {}, removeEventListener() {},
  };
}

export const testVue = { ...Vue, onMounted() {}, onBeforeUnmount() {} };

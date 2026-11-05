import { readFileSync } from 'node:fs';
import path from 'node:path';
import { compileScript, parse } from '@vue/compiler-sfc';

const query = '?admin-v2-controller';
const virtualPrefix = '\0admin-v2-controller:';
const macros = new Set(['defineProps', 'defineEmits', 'defineExpose', 'defineOptions']);

/**
 * Reuse the existing, tested application controllers without importing their UI.
 * Only presentation imports and Vue compiler macros are removed. The source SFC
 * remains byte-for-byte unchanged. AST guards deliberately fail a build if the
 * security integration points in AdminPortal change upstream.
 */
export function compileAdminV2Controller(source, filename) {
  const { descriptor, errors } = parse(source, { filename });
  if (errors.length || !descriptor.scriptSetup) throw new Error(`Missing valid script setup: ${filename}`);
  const compiled = compileScript(descriptor, { id: 'admin-v2-controller' });
  const script = descriptor.scriptSetup.content;
  const edits = [];
  const imports = [];
  const removedBindings = new Set();
  const portal = path.basename(filename) === 'AdminPortal.vue';
  const bar = path.basename(filename) === 'AdminBarSection.vue';
  const guards = new Set();
  const replace = (node, value) => edits.push({ start: node.start, end: node.end, value });
  const text = node => script.slice(node.start, node.end);

  for (const node of compiled.scriptSetupAst) {
    if (node.type === 'ImportDeclaration') {
      const module = node.source.value;
      if (module.endsWith('.vue')) {
        node.specifiers.forEach(specifier => removedBindings.add(specifier.local.name));
      } else {
        const retained = node.specifiers.filter(specifier => !macros.has(specifier.imported?.name));
        node.specifiers.filter(specifier => !retained.includes(specifier)).forEach(specifier => removedBindings.add(specifier.local.name));
        const resolved = module.startsWith('.') ? path.resolve(path.dirname(filename), module).replaceAll('\\', '/') : module;
        if (!node.specifiers.length) imports.push(`import ${JSON.stringify(resolved)};`);
        else if (retained.length) {
          const defaults = retained.filter(item => item.type === 'ImportDefaultSpecifier').map(item => item.local.name);
          const namespaces = retained.filter(item => item.type === 'ImportNamespaceSpecifier').map(item => `* as ${item.local.name}`);
          const named = retained.filter(item => item.type === 'ImportSpecifier').map(item => item.imported.name === item.local.name ? item.local.name : `${item.imported.name} as ${item.local.name}`);
          const bindings = [...defaults, ...namespaces, ...(named.length ? [`{ ${named.join(', ')} }`] : [])].join(', ');
          imports.push(`import ${bindings} from ${JSON.stringify(resolved)};`);
        }
      }
      replace(node, '');
    } else if (node.type === 'VariableDeclaration') {
      for (const declaration of node.declarations) {
        if (declaration.init?.type !== 'CallExpression') continue;
        const macro = declaration.init.callee.name;
        if (macro === 'defineProps') replace(declaration.init, '__props');
        if (macro === 'defineEmits') replace(declaration.init, '__emit');
      }
    } else if (node.type === 'ExpressionStatement' && macros.has(node.expression.callee?.name)) {
      replace(node, '');
    }
    if (node.type === 'ExpressionStatement' && node.expression.callee?.name === 'onMounted') {
      const callback = node.expression.arguments[0];
      if (callback && ['ArrowFunctionExpression', 'FunctionExpression'].includes(callback.type)) {
        let callbackSource = text(callback);
        if (bar) {
          const firstAwait = callback.body.body.find(statement => statement.type === 'ExpressionStatement' && statement.expression.type === 'AwaitExpression');
          if (!firstAwait || !text(firstAwait).includes('Promise.all')) throw new Error('Admin V2: BAR lifecycle contract changed');
          const offset = firstAwait.end - callback.start;
          callbackSource = callbackSource.slice(0, offset) + '\nif (__v2Disposed || __adapter.isDisposed?.()) return;\n' + callbackSource.slice(offset);
        }
        const wrapped = `(...args) => { __adapter.onLifecycleStart?.(); try { return Promise.resolve((${callbackSource})(...args)).then(result => { ${bar ? 'if (__v2Disposed) return;' : ''} __adapter.onLifecycleComplete?.(); return result; }).catch(error => { if (__adapter.onLifecycleError) __adapter.onLifecycleError(error); else console.error(error); }); } catch (error) { if (__adapter.onLifecycleError) __adapter.onLifecycleError(error); else console.error(error); } }`;
        if (bar) replace(node, `const __v2InitializeBar = ${wrapped};\nonMounted(__v2InitializeBar);`);
        else replace(callback, wrapped);
      }
    }

    if (!portal) continue;
    if (node.type === 'FunctionDeclaration' && node.id.name === 'loadOrganizationName') {
      const attempt = node.body.body.find(item => item.type === 'TryStatement');
      if (!attempt?.handler || !text(attempt.block).includes('apiClient.get("/admin/me"')) throw new Error('Admin V2: session verification contract changed');
      let originalTry = text(attempt.block).slice(1, -1);
      const response = attempt.block.body[0];
      const offset = response.end - attempt.block.start - 1;
      originalTry = originalTry.slice(0, offset) + '\nif (!data || typeof data.role !== "string" || !data.role.trim()) throw new Error("Risposta della sessione non valida. Riprova l’accesso.");\n' + originalTry.slice(offset);
      replace(node.body, `{ try { ${originalTry}\n__adapter.onSessionVerified?.(data);\n} catch (error) { __adapter.onSessionError?.(error); if (error?.response?.status === 401) handleUnauthorized(); throw error; } }`);
      guards.add('session');
    }
    if (node.type === 'FunctionDeclaration' && node.id.name === 'loadAll') {
      if (!text(node.body).includes('await loadOrganizationName()')) throw new Error('Admin V2: initial loading contract changed');
      replace(node.body, `{ __adapter.onLoading?.(true); try { ${text(node.body).slice(1, -1)} } catch (error) { __adapter.onLoadError?.(error); throw error; } finally { __adapter.onLoading?.(false); } }`);
      guards.add('loading');
    }
    if (node.type === 'FunctionDeclaration' && node.id.name === 'secureRequest') {
      const attempt = node.body.body.find(item => item.type === 'TryStatement');
      if (!attempt?.handler) throw new Error('Admin V2: request error contract changed');
      replace(attempt.handler.body, `{ __adapter.onRequestError?.(error); ${text(attempt.handler.body).slice(1, -1)} }`);
      guards.add('requests');
    }
    if (node.type === 'FunctionDeclaration' && node.id.name === 'logout') {
      const request = node.body.body[0];
      if (request?.type !== 'TryStatement' || !text(request).includes('apiClient.post("/admin/logout")')) throw new Error('Admin V2: logout serialization contract changed');
      const clearSession = script.slice(request.end, node.body.end - 1);
      replace(node.body, `{ if (__v2LogoutPromise) return __v2LogoutPromise; __adapter.onSessionEnded?.(); ${clearSession}\n__v2LogoutPromise = (async () => { ${text(request)} })().finally(() => { __v2LogoutPromise = null; }); return __v2LogoutPromise; }`);
      guards.add('logout');
    }
    if (node.type === 'FunctionDeclaration' && node.id.name === 'login') {
      const attempt = node.body.body.find(item => item.type === 'TryStatement');
      if (!attempt || !text(attempt.block).includes('apiClient.post("/admin/login"')) throw new Error('Admin V2: login serialization contract changed');
      replace(attempt.block, `{ if (__v2LogoutPromise) await __v2LogoutPromise; ${text(attempt.block).slice(1, -1)} }`);
      guards.add('login');
    }
    if (node.type === 'IfStatement' && text(node.test) === 'isAuthenticated.value' && text(node).includes('loadAll();')) {
      replace(node, '');
      guards.add('bootstrap');
    }
    // The V2 application is remounted on slug changes; the legacy watcher only
    // updates the organization name and must not start a second partial load.
    if (node.type === 'ExpressionStatement' && /^watch\(\(\) => props\.organizationSlug/.test(text(node))) {
      replace(node, '');
      guards.add('slug');
    }
  }
  if (portal && guards.size !== 7) throw new Error(`Admin V2: missing controller integration points (${[...guards].join(', ')})`);
  let output = script;
  for (const edit of edits.sort((a, b) => b.start - a.start)) output = output.slice(0, edit.start) + edit.value + output.slice(edit.end);
  const bindings = Object.entries(compiled.bindings)
    .filter(([name, kind]) => !removedBindings.has(name) && kind !== 'props' && kind !== 'props-aliased')
    .map(([name]) => name);
  const lifecycle = bar ? 'let __v2Disposed = false; onBeforeUnmount(() => { __v2Disposed = true; });\n' : portal ? 'let __v2LogoutPromise = null;\n' : '';
  const extraBindings = bar ? ', retryLifecycle: () => { if (__v2Disposed) return; if (refreshTimer) { window.clearInterval(refreshTimer); refreshTimer = null; } stopOrdersSSE(); return __v2InitializeBar(); }' : '';
  return `${imports.join('\n')}\nexport function useController(__props = {}, __emit = () => {}, __adapter = {}) {\n${lifecycle}${output}\nreturn { ${bindings.join(', ')}${extraBindings} };\n}\n`;
}

export function adminV2Controllers() {
  return {
    name: 'arenaboostx-admin-v2-controllers',
    enforce: 'pre',
    async resolveId(source, importer) {
      if (!source.endsWith(query)) return null;
      const resolved = await this.resolve(source.slice(0, -query.length), importer, { skipSelf: true });
      if (!resolved) throw new Error(`Admin V2 controller not found: ${source}`);
      return virtualPrefix + resolved.id;
    },
    load(id) {
      if (!id.startsWith(virtualPrefix)) return null;
      const filename = id.slice(virtualPrefix.length);
      this.addWatchFile(filename);
      return compileAdminV2Controller(readFileSync(filename, 'utf8'), filename);
    },
  };
}

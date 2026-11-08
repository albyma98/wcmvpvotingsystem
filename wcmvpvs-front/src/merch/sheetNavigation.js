// The browser history stores an opaque entry ID. Checkout data stays in memory.
export function createSheetNavigation(browser, onChange, isBusy) {
  const session = globalThis.crypto?.randomUUID?.() || String(Math.random());
  const entries = new Map();
  let active;
  let sequence = 0;
  let traversing = false;
  let restoring = false;
  let pendingView;

  function select(entry) {
    if (entry === active) return;
    active = entry;
    onChange(entry?.view ?? null);
  }

  function show(view, replace = false) {
    if (restoring) {
      pendingView = view;
      return;
    }
    if (traversing) return;
    if (!active) entries.clear();
    const id = `${session}:${++sequence}`;
    const entry = { view, depth: replace ? active.depth : (active?.depth || 0) + 1 };
    const state = { ...browser.history.state, fanMerchSheet: id };
    if (replace) browser.history.replaceState(state, '');
    else browser.history.pushState(state, '');
    entries.set(id, entry);
    select(entry);
  }

  function pop(event) {
    const entry = entries.get(event.state?.fanMerchSheet);
    if (restoring) {
      if (entry !== active) {
        browser.history.go(active.depth - (entry?.depth || 0));
        return;
      }
      restoring = false;
      if (pendingView) {
        show(pendingView, true);
        pendingView = undefined;
      }
      return;
    }
    traversing = false;
    if (active && entry !== active && isBusy()) {
      restoring = true;
      browser.history.go(active.depth - (entry?.depth || 0));
      return;
    }
    select(entry);
  }

  function back(close = false) {
    if (!active || traversing || restoring || isBusy()) return;
    traversing = true;
    browser.history.go(close ? -active.depth : -1);
  }

  browser.addEventListener('popstate', pop);
  return { show, back, dispose: () => browser.removeEventListener('popstate', pop) };
}

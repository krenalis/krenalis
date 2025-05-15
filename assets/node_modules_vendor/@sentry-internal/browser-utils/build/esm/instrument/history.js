import { addHandler, maybeInstrument, triggerHandlers, supportsHistory, fill } from '@sentry/core';
import { WINDOW } from '../types.js';

let lastHref;

/**
 * Add an instrumentation handler for when a fetch request happens.
 * The handler function is called once when the request starts and once when it ends,
 * which can be identified by checking if it has an `endTimestamp`.
 *
 * Use at your own risk, this might break without changelog notice, only used internally.
 * @hidden
 */
function addHistoryInstrumentationHandler(handler) {
  const type = 'history';
  addHandler(type, handler);
  maybeInstrument(type, instrumentHistory);
}

/**
 * Exported just for testing
 */
function instrumentHistory() {
  // The `popstate` event may also be triggered on `pushState`, but it may not always reliably be emitted by the browser
  // Which is why we also monkey-patch methods below, in addition to this
  WINDOW.addEventListener('popstate', () => {
    const to = WINDOW.location.href;
    // keep track of the current URL state, as we always receive only the updated state
    const from = lastHref;
    lastHref = to;

    if (from === to) {
      return;
    }

    const handlerData = { from, to } ;
    triggerHandlers('history', handlerData);
  });

  // Just guard against this not being available, in weird environments
  if (!supportsHistory()) {
    return;
  }

  function historyReplacementFunction(originalHistoryFunction) {
    return function ( ...args) {
      const url = args.length > 2 ? args[2] : undefined;
      if (url) {
        // coerce to string (this is what pushState does)
        const from = lastHref;
        const to = String(url);
        // keep track of the current URL state, as we always receive only the updated state
        lastHref = to;

        if (from === to) {
          return originalHistoryFunction.apply(this, args);
        }

        const handlerData = { from, to } ;
        triggerHandlers('history', handlerData);
      }
      return originalHistoryFunction.apply(this, args);
    };
  }

  fill(WINDOW.history, 'pushState', historyReplacementFunction);
  fill(WINDOW.history, 'replaceState', historyReplacementFunction);
}

export { addHistoryInstrumentationHandler, instrumentHistory };
//# sourceMappingURL=history.js.map

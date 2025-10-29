// Module update-event-timestamps.js rewrites JSON code-block timestamps keeping the original deltas.

const TIMESTAMP_FIELDS = ['receivedAt', 'sentAt', 'originalTimestamp', 'timestamp'];
const observerRegistry = new WeakMap();

function normalizeSelectors(input) {
  if (typeof input === 'string') {
    return [input];
  }
  if (Array.isArray(input)) {
    return input.filter((value) => typeof value === 'string' && value.trim() !== '');
  }
  return [];
}

function collectRoots(selectors, context = document) {
  if (!context || typeof context.querySelectorAll !== 'function') {
    return [];
  }

  const unique = new Set();

  selectors.forEach((selector) => {
    const trimmed = selector.trim();
    if (!trimmed) {
      return;
    }

    const nodes = context.querySelectorAll(trimmed);
    nodes.forEach((node) => {
      if (node) {
        unique.add(node);
      }
    });
  });

  return Array.from(unique);
}

function collectCodeBlocks(root) {
  if (!root) {
    return [];
  }

  if (root.matches && root.matches('code')) {
    return [root];
  }

  if (typeof root.querySelectorAll !== 'function') {
    return [];
  }

  const candidates = root.querySelectorAll('code');
  const blocks = [];
  candidates.forEach((element) => {
    if (!element) {
      return;
    }
    // Highlight.js wraps the <code> inside a <pre class="language-json">; cover that case.
    const parent = element.parentElement;
    if (
      parent &&
      parent.classList &&
      parent.classList.contains('language-json') &&
      !blocks.includes(element)
    ) {
      blocks.push(element);
      return;
    }
    if (element.classList && element.classList.contains('language-json')) {
      blocks.push(element);
      return;
    }
    const language = element.getAttribute ? element.getAttribute('data-language') : null;
    if (language && language.toLowerCase() === 'json') {
      blocks.push(element);
    }
  });
  return blocks;
}

function readJsonPayload(codeElement) {
  if (!codeElement || typeof codeElement.textContent !== 'string') {
    return null;
  }

  const source = codeElement.textContent.trim();
  if (!source) {
    return null;
  }

  try {
    return JSON.parse(source);
  } catch (error) {
    return null;
  }
}

function takeTimestampSnapshot(payload) {
  const snapshot = new Map();

  TIMESTAMP_FIELDS.forEach((field) => {
    const value = payload[field];
    // Skip fields that are missing or not parseable as ISO strings.
    if (typeof value !== 'string') {
      return;
    }
    const dateValue = new Date(value);
    if (Number.isNaN(dateValue.getTime())) {
      return;
    }
    snapshot.set(field, dateValue);
  });

  return snapshot;
}

function resolveBaseDate(snapshot) {
  if (snapshot.has('receivedAt')) {
    return snapshot.get('receivedAt');
  }

  let baseDate = null;
  snapshot.forEach((date) => {
    if (!baseDate || date > baseDate) {
      baseDate = date;
    }
  });
  return baseDate;
}

function applyTimestampDeltas(payload, snapshot, referenceDate) {
  if (!referenceDate) {
    return false;
  }

  let updated = false;
  const referenceMillis = referenceDate.getTime();
  const baseDate = resolveBaseDate(snapshot);

  if (!baseDate) {
    return false;
  }

  const baseMillis = baseDate.getTime();

  snapshot.forEach((originalDate, field) => {
    if (!Object.prototype.hasOwnProperty.call(payload, field)) {
      return;
    }

    const delta = originalDate.getTime() - baseMillis;
    const nextDate = new Date(referenceMillis + delta);
    payload[field] = nextDate.toISOString();
    updated = true;
  });

  return updated;
}

function renderJsonPayload(codeElement, payload) {
  if (!codeElement) {
    return;
  }
  codeElement.textContent = JSON.stringify(payload, null, 4);
  rehighlight(codeElement);
}

function rehighlight(codeElement) {
  if (typeof window === 'undefined') {
    return;
  }
  const hljs = window.hljs;
  if (!hljs || typeof hljs.highlightElement !== 'function') {
    return;
  }
  if (codeElement.dataset && codeElement.dataset.highlighted) {
    try {
      delete codeElement.dataset.highlighted;
    } catch (error) {
      codeElement.removeAttribute('data-highlighted');
    }
  }
  try {
    hljs.highlightElement(codeElement);
  } catch (error) {
    // Ignore highlight retries
  }
}

function refreshCodeBlock(codeElement) {
  const payload = readJsonPayload(codeElement);
  if (!payload || typeof payload !== 'object') {
    return;
  }

  const snapshot = takeTimestampSnapshot(payload);
  if (!snapshot.size) {
    return;
  }

  // Use the current time as the new reference while preserving spacing between timestamps.
  const now = new Date();
  if (!applyTimestampDeltas(payload, snapshot, now)) {
    return;
  }

  renderJsonPayload(codeElement, payload);
}

export function updateTimestamps(selectors, context = document) {
  const selectorList = normalizeSelectors(selectors);
  if (!selectorList.length) {
    return;
  }

  const attempt = () => {
    const currentRoots = collectRoots(selectorList, context);
    if (!currentRoots.length) {
      return false;
    }
    let touched = false;
    currentRoots.forEach((root) => {
      const codeBlocks = collectCodeBlocks(root);
      if (!codeBlocks.length) {
        return;
      }
      codeBlocks.forEach((codeElement) => {
        refreshCodeBlock(codeElement);
        touched = true;
      });
    });
    return touched;
  };

  if (attempt()) {
    return;
  }

  // The wrapper may be injected later (e.g. by code-expand), so watch and retry once it appears.
  scheduleRetry(context, selectorList, attempt);
}

function scheduleRetry(context, selectors, attempt) {
  if (typeof MutationObserver !== 'function') {
    return;
  }

  const target = resolveObservationTarget(context);
  if (!target) {
    return;
  }

  let registry = observerRegistry.get(target);
  if (!registry) {
    registry = new Map();
    observerRegistry.set(target, registry);
  }

  const key = selectors.join(',');
  if (registry.has(key)) {
    return;
  }

  const observer = new MutationObserver(() => {
    if (!attempt()) {
      return;
    }
    observer.disconnect();
    registry.delete(key);
  });

  const observedNode = resolveObservedNode(target);
  if (!observedNode) {
    return;
  }

  observer.observe(observedNode, { childList: true, subtree: true });
  registry.set(key, observer);
}

function resolveObservationTarget(context) {
  if (context && typeof context.querySelectorAll === 'function') {
    return context;
  }
  if (typeof document !== 'undefined') {
    return document;
  }
  return null;
}

function resolveObservedNode(target) {
  if (!target) {
    return null;
  }
  if (target.nodeType === Node.DOCUMENT_NODE) {
    return target.documentElement || target.body || null;
  }
  return target;
}

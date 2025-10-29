// code-expand.js wires expandable controls to Markdown code blocks that opt-in
// using the <!-- code-expand height:NNNpx --> directive (and optional id=foo).
// The module is idempotent and runs once the DOM is ready.

const CODE_SELECTOR = 'pre > code';
const DEFAULT_MAX_HEIGHT = 200;
const DEFAULT_HEIGHT_VALUE = `${DEFAULT_MAX_HEIGHT}px`;
const DIRECTIVE_PATTERN = /^code-expand(?:\s+(.*))?$/i;
const DATA_ATTRIBUTE = 'codeExpandInitialized';
const CLASS_NAME = 'code-expand';
const TRANSITION_PROPERTY = 'max-height';
const TRANSITION_FALLBACK_TIMEOUT = 450;
const FLOW_ROOT_DISPLAY = 'flow-root';

// buildCodeExpand scans the supplied document for code blocks that request the
// code-expand behaviour. It returns the number of blocks that received expand UI.
export function buildCodeExpand(doc = document) {
  if (!doc || !doc.querySelectorAll) {
    return 0;
  }

  const codeElements = doc.querySelectorAll(CODE_SELECTOR);
  let enhancedCount = 0;

  codeElements.forEach((codeElement) => {
    const preElement = codeElement.parentElement;

    if (!preElement || preElement.tagName !== 'PRE') {
      return;
    }

    if (enhanceCodeBlock(preElement)) {
      enhancedCount += 1;
    }
  });

  return enhancedCount;
}

// enhanceCodeBlock inspects a single <pre> element, returning true when the
// directive is honoured and the expand wrapper is installed.
function enhanceCodeBlock(preElement) {
  const parentElement = preElement.parentElement;

  if (parentElement && parentElement.dataset[DATA_ATTRIBUTE] === 'true') {
    return false;
  }

  const directive = findDirectiveComment(preElement);

  if (!directive) {
    return false;
  }

  const { comment, heightValue, heightPx, id } = directive;
  const naturalHeight = getNaturalHeight(preElement);
  const clampHeight = Number.isFinite(heightPx) ? heightPx : DEFAULT_MAX_HEIGHT;

  if (naturalHeight <= clampHeight) {
    comment.remove();
    return false;
  }

  wrapWithExpandUI(preElement, comment, heightValue, id);
  return true;
}

// findDirectiveComment walks the nearby siblings (searching forward, then
// backward) of the code block to resolve the code-expand directive and its
// configured height, if present. The search tolerates intermediate whitespace
// nodes and climbs ancestors so directives that end up outside nested wrappers
// (e.g., list items) are still honoured.
function findDirectiveComment(preElement) {
  if (!preElement) {
    return null;
  }

  const forwardMatch = findDirectiveCommentInDirection(preElement, 'nextSibling');

  if (forwardMatch) {
    return forwardMatch;
  }

  return findDirectiveCommentInDirection(preElement, 'previousSibling');
}

function findDirectiveCommentInDirection(preElement, siblingKey) {
  let anchor = preElement;

  while (anchor) {
    let sibling = anchor[siblingKey];

    while (sibling) {
      if (sibling.nodeType === Node.TEXT_NODE) {
        if (sibling.textContent.trim() === '') {
          sibling = sibling[siblingKey];
          continue;
        }
        break;
      }

      if (sibling.nodeType === Node.COMMENT_NODE) {
        const commentValue = sibling.textContent.trim();
        const match = DIRECTIVE_PATTERN.exec(commentValue);

        if (match) {
          const parsed = parseDirectiveOptions(match[1], preElement);
          if (!parsed) {
            sibling = sibling[siblingKey];
            continue;
          }
          return {
            comment: sibling,
            heightValue: parsed.heightValue,
            heightPx: parsed.heightPx,
            id: parsed.id,
          };
        }

        sibling = sibling[siblingKey];
        continue;
      }

      break;
    }

    if (sibling) {
      return null;
    }

    anchor = anchor.parentNode;

    if (!anchor || anchor.nodeType === Node.DOCUMENT_NODE) {
      break;
    }
  }

  return null;
}

// getNaturalHeight reads the scrollHeight for the provided element, which
// reflects the natural full height of the code block.
function getNaturalHeight(element) {
  return element.scrollHeight;
}

// normalizeHeightValue cleans the directive token and ensures bare numbers fall back to pixels.
function normalizeHeightValue(rawValue) {
  if (!rawValue) {
    return DEFAULT_HEIGHT_VALUE;
  }

  const trimmed = rawValue.trim();
  if (!trimmed) {
    return DEFAULT_HEIGHT_VALUE;
  }

  if (/^-?\d+(?:\.\d+)?$/.test(trimmed)) {
    return `${trimmed}px`;
  }

  return trimmed;
}

// computeMaxHeightPixels converts a CSS height token into pixels so it can be compared with the block.
function computeMaxHeightPixels(heightValue, preElement) {
  if (!heightValue) {
    return DEFAULT_MAX_HEIGHT;
  }

  const trimmed = heightValue.trim();
  if (!trimmed) {
    return DEFAULT_MAX_HEIGHT;
  }

  const numericValue = Number(trimmed);
  if (!Number.isNaN(numericValue)) {
    return numericValue;
  }

  const pxMatch = trimmed.match(/^(-?\d+(?:\.\d+)?)px$/i);
  if (pxMatch) {
    return parseFloat(pxMatch[1]);
  }

  const doc = (preElement && preElement.ownerDocument) || document;
  const container = doc.body || doc.documentElement;
  if (!container) {
    return DEFAULT_MAX_HEIGHT;
  }

  const probe = doc.createElement('div');
  probe.style.position = 'absolute';
  probe.style.visibility = 'hidden';
  probe.style.height = trimmed;
  probe.style.border = '0';
  probe.style.padding = '0';
  probe.style.boxSizing = 'content-box';
  probe.style.top = '-10000px';
  probe.style.left = '-10000px';

  container.appendChild(probe);
  const rect = probe.getBoundingClientRect();
  container.removeChild(probe);

  if (rect && typeof rect.height === 'number' && !Number.isNaN(rect.height)) {
    return rect.height;
  }

  return DEFAULT_MAX_HEIGHT;
}

// parseDirectiveOptions extracts supported options from the comment payload.
function parseDirectiveOptions(rawOptions, preElement) {
  const source = typeof rawOptions === 'string' ? rawOptions.trim() : '';
  const options = Object.create(null);

  if (source) {
    let cursor = source;
    const optionPattern =
      /^([a-z0-9_-]+)\s*(?::|=)\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s]+)\s*/i;

    while (cursor.length) {
      const match = optionPattern.exec(cursor);
      if (!match) {
        return null;
      }

      const key = match[1].toLowerCase();
      const valueToken = match[2];
      options[key] = stripOptionQuotes(valueToken);

      cursor = cursor.slice(match[0].length);
      cursor = cursor.replace(/^\s+/, '');
    }
  }

  let heightValue = DEFAULT_HEIGHT_VALUE;
  let heightPx = DEFAULT_MAX_HEIGHT;
  if (typeof options.height === 'string' && options.height.trim()) {
    heightValue = normalizeHeightValue(options.height);
    heightPx = computeMaxHeightPixels(heightValue, preElement);
  }

  const rawId = typeof options.id === 'string' ? options.id.trim() : '';
  const sanitizedId = rawId && !/\s/.test(rawId) ? rawId : null;

  return {
    heightValue,
    heightPx,
    id: sanitizedId,
  };
}

// stripOptionQuotes removes surrounding single or double quotes from option values.
function stripOptionQuotes(value) {
  if (typeof value !== 'string') {
    return '';
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }

  const firstChar = trimmed.charAt(0);
  const lastChar = trimmed.charAt(trimmed.length - 1);
  if (
    trimmed.length >= 2 &&
    ((firstChar === '"' && lastChar === '"') || (firstChar === "'" && lastChar === "'"))
  ) {
    const inner = trimmed.slice(1, -1);
    return inner.replace(/\\(['"\\])/g, '$1');
  }

  return trimmed;
}

// setMaxHeightWithAnimation applies a pixel height to the container, using
// requestAnimationFrame when available to avoid skipping the CSS transition.
function setMaxHeightWithAnimation(container, heightPx) {
  const apply = () => {
    container.style.maxHeight = `${heightPx}px`;
  };

  if (typeof requestAnimationFrame === 'function') {
    requestAnimationFrame(apply);
    return;
  }

  apply();
}

// wrapWithExpandUI constructs the wrapper, fade overlay, and expand button for
// a target <pre> element.
function wrapWithExpandUI(preElement, comment, maxHeightValue, containerId) {
  const contextDoc = preElement.ownerDocument || document;
  const container = contextDoc.createElement('div');
  container.className = CLASS_NAME;
  container.dataset[DATA_ATTRIBUTE] = 'true';
  container.style.maxHeight = maxHeightValue || DEFAULT_HEIGHT_VALUE;
  container.style.overflow = 'hidden';
  if (containerId) {
    container.id = containerId;
  }

  const originalParent = preElement.parentNode;

  if (!originalParent) {
    return;
  }

  originalParent.insertBefore(container, preElement);
  container.appendChild(preElement);

  const fade = createFadeElement(contextDoc);
  const button = createExpandButton(contextDoc);

  container.appendChild(fade);
  container.appendChild(button);

  const handleExpand = () => {
    button.setAttribute('aria-expanded', 'true');

    fade.remove();
    button.remove();

    const expandedHeight = preElement.scrollHeight;
    let isFinished = false;
    let fallbackTimer = null;

    const finishExpansion = () => {
      if (isFinished) {
        return;
      }
      isFinished = true;
      container.removeEventListener('transitionend', onTransitionEnd);
      if (fallbackTimer !== null) {
        clearTimeout(fallbackTimer);
        fallbackTimer = null;
      }
      container.style.removeProperty('max-height');
      container.style.display = FLOW_ROOT_DISPLAY;
      container.style.removeProperty('overflow');
      container.classList.remove(CLASS_NAME);
      delete container.dataset[DATA_ATTRIBUTE];
      if (comment.parentNode) {
        comment.remove();
      }
      cleanupStyleAttribute(container);
    };

    const onTransitionEnd = (event) => {
      if (event.target !== container || event.propertyName !== TRANSITION_PROPERTY) {
        return;
      }
      finishExpansion();
    };

    container.addEventListener('transitionend', onTransitionEnd);

    setMaxHeightWithAnimation(container, expandedHeight);
    fallbackTimer = setTimeout(finishExpansion, TRANSITION_FALLBACK_TIMEOUT);
  };

  button.addEventListener('click', handleExpand, { once: true });
}

// createFadeElement returns the gradient overlay that hints at hidden content.
function createFadeElement(contextDoc) {
  const fade = contextDoc.createElement('div');
  fade.className = 'code-expand__fade';
  fade.setAttribute('aria-hidden', 'true');
  return fade;
}

// createExpandButton constructs the expand button with its inline SVG icon.
function createExpandButton(contextDoc) {
  const button = contextDoc.createElement('button');
  button.className = 'code-expand__btn';
  button.type = 'button';
  button.setAttribute('aria-label', 'Expand code');
  button.setAttribute('aria-expanded', 'false');

  button.innerHTML = `
    <svg viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <path fill="currentColor" d="M8 11a.5.5 0 0 1-.35-.15l-4-4a.5.5 0 1 1 .7-.7L8 9.79l3.65-3.64a.5.5 0 1 1 .7.7l-4 4A.5.5 0 0 1 8 11z"></path>
    </svg>
  `.trim();

  return button;
}

// cleanupStyleAttribute removes the style attribute when it no longer carries
// inline declarations.
function cleanupStyleAttribute(element) {
  if (!element.getAttribute('style')) {
    element.removeAttribute('style');
  }
}

if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function handleReady() {
      document.removeEventListener('DOMContentLoaded', handleReady);
      buildCodeExpand();
    });
  } else {
    buildCodeExpand();
  }
}

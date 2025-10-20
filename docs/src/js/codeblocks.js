/* eslint-disable no-console */
const hashHandlerRegistry = new WeakMap();

const LANGUAGE_LABELS = new Map([
  ['csharp', 'C#'],
  ['go', 'Go'],
  ['java', 'Java'],
  ['javascript', 'JavaScript'],
  ['json', 'JSON'],
  ['kotlin', 'Kotlin'],
  ['nodejs', 'Node.js'],
  ['python', 'Python'],
  ['sh', 'Shell'],
  ['sql', 'SQL'],
  ['typescript', 'TypeScript'],
]);

/**
 * Enhance Markdown-authored codeblocks comments into tabbed code examples.
 * @param {Document} [doc=document]
 * @param {Window} [win=window]
 * @returns {Array<{section: HTMLElement, buttons: HTMLButtonElement[], panels: HTMLElement[]}>}
 */
export function buildCodeblocks(doc = document, win = window) {
  if (!doc || !win || !doc.body) {
    return [];
  }

  const environment = resolveEnvironment(win);
  const usedIds = collectExistingIds(doc);
  const tabDataBySection = new WeakMap();
  const idDirectory = new Map();
  const groupRegistry = new Map();
  const COPY_RESET_TIMER =
    typeof Symbol === 'function' ? Symbol('meergoCopyResetTimer') : '__meergoCopyResetTimer';
  const STORAGE_KEY = 'meergo.docs.codeBlockTabs';
  const storage = resolveStorage(win);
  let syncState = storage ? loadSyncState(storage, STORAGE_KEY) : Object.create(null);

  const isPreNode = function (node) {
    return (
      node &&
      typeof node.nodeName === 'string' &&
      node.nodeType === environment.elementNode &&
      node.nodeName.toLowerCase() === 'pre'
    );
  };

  const headingTagPattern = /^h[2-6]$/;

  const isHeadingNode = function (node) {
    return (
      node &&
      typeof node.nodeName === 'string' &&
      node.nodeType === environment.elementNode &&
      headingTagPattern.test(node.nodeName.toLowerCase())
    );
  };

  function extractHeadingLabel(node) {
    if (!node || typeof node.textContent !== 'string') {
      return '';
    }
    return node.textContent.trim();
  }

  function slugify(value, fallback) {
    const base = (value == null ? '' : String(value))
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .replace(/--+/g, '-');
    return base || fallback;
  }

  function uniqueId(base) {
    let candidate = base;
    let index = 2;
    while (usedIds.has(candidate)) {
      candidate = base + '-' + index;
      index += 1;
    }
    usedIds.add(candidate);
    return candidate;
  }

  function extractLanguageToken(element) {
    if (!element || typeof element.getAttribute !== 'function') {
      return null;
    }

    let classAttr = element.getAttribute('class');
    if (!classAttr && typeof element.className === 'string') {
      classAttr = element.className;
    }

    const tokens = classAttr ? classAttr.split(/\s+/) : [];
    for (let index = 0; index < tokens.length; index += 1) {
      const token = tokens[index];
      if (!token) {
        continue;
      }
      const match = token.match(/^language-([a-z0-9+#-]+)$/i);
      if (match) {
        return match[1];
      }
    }

    const dataLanguage = element.getAttribute('data-language');
    if (dataLanguage) {
      return dataLanguage;
    }

    return null;
  }

  function detectCodeLanguage(preElement) {
    if (!preElement || preElement.nodeType !== environment.elementNode) {
      return null;
    }

    const codeElement =
      typeof preElement.querySelector === 'function' ? preElement.querySelector('code') : null;
    const fromCode = extractLanguageToken(codeElement);
    if (fromCode) {
      return fromCode;
    }

    return extractLanguageToken(preElement);
  }

  function resolveLanguageLabel(language, index) {
    if (!language) {
      return 'Code ' + index;
    }

    const canonical = String(language).toLowerCase();
    if (LANGUAGE_LABELS.has(canonical)) {
      return LANGUAGE_LABELS.get(canonical);
    }
    return language;
  }

  function updateHash(panelId) {
    if (!panelId) {
      return;
    }
    const target = '#' + panelId;
    if (win.location.hash === target) {
      return;
    }
    if (typeof win.history.replaceState === 'function') {
      try {
        win.history.replaceState(null, '', target);
      } catch (error) {
        // ignore history failures
      }
    } else {
      win.location.hash = target;
    }
  }

  function focusTab(buttons, index) {
    buttons.forEach(function (button, idx) {
      button.setAttribute('tabindex', idx === index ? '0' : '-1');
    });
    buttons[index].focus();
  }

  function activateTab(section, tabButton, options) {
    const opts = options || {};
    const skipSync = Boolean(opts.skipSync);
    const state = tabDataBySection.get(section);
    if (!state) {
      return;
    }

    const targetPanelId = tabButton.getAttribute('aria-controls');
    if (!targetPanelId) {
      return;
    }

    let targetPanel = null;
    state.panels.forEach(function (panel) {
      if (panel.id === targetPanelId) {
        targetPanel = panel;
      }
    });

    if (!targetPanel) {
      return;
    }

    const value = state.valueByButton ? state.valueByButton.get(tabButton) : null;

    if (state.activeButton === tabButton) {
      if (opts.focus) {
        tabButton.focus();
      }
      if (!opts.skipHash) {
        updateHash(targetPanelId);
      }
      state.activeValue = value;
      state.activePanel = targetPanel;
      return;
    }

    state.buttons.forEach(function (button) {
      const isActive = button === tabButton;
      button.setAttribute('aria-selected', isActive ? 'true' : 'false');
      button.setAttribute('tabindex', isActive ? '0' : '-1');
    });

    state.panels.forEach(function (panel) {
      panel.hidden = panel !== targetPanel;
    });

    state.activeButton = tabButton;
    state.activeValue = value;

    if (opts.focus) {
      tabButton.focus();
    }
    if (!opts.skipHash) {
      updateHash(targetPanelId);
    }
    if (!skipSync && state.syncGroup && value) {
      persistGroupValue(state.syncGroup, value);
      synchronizeGroup(state.syncGroup, value, section);
    }
    state.activePanel = targetPanel;
  }

  function collectCodeblockSets() {
    const blocks = [];
    const walker = doc.createTreeWalker(doc.body, environment.commentFilter);
    let currentComment = walker.nextNode();

    while (currentComment) {
      const raw = (currentComment.nodeValue || '').trim();
      const marker = raw.match(
        /^codeblocks\s+(?:sync:([A-Za-z0-9_-]+)\s+)?([^\s]+)\s*$/i
      );
      if (marker) {
        const syncGroup = marker[1] ? marker[1].trim() : null;
        const title = marker[2] ? marker[2].trim() : '';
        const nodes = [];
        let cursor = currentComment.nextSibling;
        let closingMarker = null;
        while (cursor) {
          if (
            cursor.nodeType === environment.commentNode &&
            isEndMarker(cursor)
          ) {
            closingMarker = cursor;
            break;
          }
          nodes.push(cursor);
          cursor = cursor.nextSibling;
        }

        if (!closingMarker) {
          console.warn(
            'Skipping codeblocks set "%s" because no closing marker was found.',
            title || '(untitled)'
          );
        } else {
          blocks.push({
            title: title || 'Code examples',
            syncGroup: syncGroup || null,
            nodes: nodes.slice(),
            startComment: currentComment,
            endComment: closingMarker
          });
          walker.currentNode = closingMarker;
        }
      }
      currentComment = walker.nextNode();
    }

    const sets = [];
    blocks.forEach(function (block) {
      try {
        const transformed = transformBlock(block);
        if (transformed) {
          sets.push(transformed);
        }
      } catch (error) {
        console.warn(
          'Skipping codeblocks set "%s" due to an unexpected error.',
          block.title
        );
      }
    });

    return sets;
  }

  function transformBlock(block) {
    const tabItems = [];
    let index = 0;
    let pendingHeading = null;
    let headingTagName = null;

    const captureHeading = function (node) {
      if (!isHeadingNode(node)) {
        return false;
      }

      const tagName = node.nodeName.toLowerCase();
      if (headingTagName && tagName !== headingTagName) {
        return false;
      }

      headingTagName = tagName;
      pendingHeading = node;
      return true;
    };

    while (index < block.nodes.length) {
      const node = block.nodes[index];

      if (captureHeading(node)) {
        index += 1;
        continue;
      }

      if (!isPreNode(node)) {
        index += 1;
        continue;
      }

      const codeNode = node;
      const footNodes = [];
      const headingLabel = pendingHeading ? extractHeadingLabel(pendingHeading) : '';
      pendingHeading = null;
      index += 1;

      while (index < block.nodes.length) {
        const candidate = block.nodes[index];
        if (captureHeading(candidate)) {
          break;
        }
        if (isPreNode(candidate)) {
          break;
        }
        if (
          candidate &&
          candidate.nodeType === environment.commentNode
        ) {
          index += 1;
          continue;
        }
        if (
          candidate &&
          candidate.nodeType === environment.textNode &&
          !(candidate.textContent || '').trim()
        ) {
          index += 1;
          continue;
        }
        footNodes.push(candidate);
        index += 1;
      }

      const language = detectCodeLanguage(codeNode);
      const canonicalLanguage = language ? String(language).toLowerCase() : null;
      const fallbackIndex = tabItems.length + 1;
      const label =
        headingLabel !== ''
          ? headingLabel
          : resolveLanguageLabel(canonicalLanguage, fallbackIndex);
      const slugSource =
        headingLabel !== ''
          ? headingLabel
          : canonicalLanguage || 'code-' + fallbackIndex;
      tabItems.push({
        label: label,
        codeNode: codeNode,
        footNodes: footNodes,
        dataset: canonicalLanguage ? { language: canonicalLanguage } : null,
        slugValue: slugSource,
        value: canonicalLanguage || slugify(slugSource, 'code')
      });
    }

    if (tabItems.length < 2) {
      console.warn(
        'Skipping codeblocks set "%s" because it has fewer than two code blocks.',
        block.title
      );
      return null;
    }

    return createTabsSection(block, tabItems);
  }

  function createTabsSection(block, tabItems) {
    const section = doc.createElement('section');
    section.className = 'tabs';
    section.setAttribute('aria-label', block.title);
    section.dataset.tabsType = 'codeblocks';
    if (block.syncGroup) {
      section.dataset.tabsSync = block.syncGroup;
    }

    const tabsNav = doc.createElement('div');
    tabsNav.className = 'tabs-nav';
    tabsNav.setAttribute('role', 'tablist');
    tabsNav.setAttribute('aria-orientation', 'horizontal');

    const header = doc.createElement('div');
    header.className = 'codeblocks-header';
    header.appendChild(tabsNav);

    const copyButton = createCopyButton(doc);
    header.appendChild(copyButton);
    section.appendChild(header);

    const buttons = [];
    const panels = [];
    const valueByButton = new Map();
    const buttonByValue = new Map();
    const titleSlug = slugify(block.title, 'code');

    tabItems.forEach(function (tabInfo, index) {
      const slugSource = tabInfo.slugValue || tabInfo.label || 'code';
      const labelSlug = slugify(slugSource, 'code');
      const tabId = uniqueId('tab-' + titleSlug + '-' + labelSlug);
      const panelId = uniqueId('panel-' + titleSlug + '-' + labelSlug);
      const valueForTab =
        tabInfo.value != null ? String(tabInfo.value) : slugify(slugSource, 'code');

      const button = doc.createElement('button');
      button.type = 'button';
      button.setAttribute('role', 'tab');
      button.id = tabId;
      button.setAttribute('aria-controls', panelId);
      button.setAttribute('aria-selected', 'false');
      button.setAttribute('tabindex', index === 0 ? '0' : '-1');
      button.textContent = tabInfo.label;
      button.dataset.tabValue = valueForTab;

      const panel = doc.createElement('div');
      panel.setAttribute('role', 'tabpanel');
      panel.id = panelId;
      panel.setAttribute('aria-labelledby', tabId);
      panel.hidden = true;
      panel.dataset.tabValue = valueForTab;

      if (tabInfo.codeNode) {
        panel.appendChild(tabInfo.codeNode);
      }

      if (tabInfo.dataset) {
        Object.keys(tabInfo.dataset).forEach(function (key) {
          panel.dataset[key] = tabInfo.dataset[key];
        });
      }

      if (tabInfo.footNodes && tabInfo.footNodes.length) {
        const note = doc.createElement('div');
        note.className = 'codeblocks-note';
        note.setAttribute('role', 'note');
        note.setAttribute('aria-label', tabInfo.label + ' details');

        tabInfo.footNodes.forEach(function (footNode) {
          note.appendChild(footNode);
        });

        panel.appendChild(note);
      }

      tabsNav.appendChild(button);
      section.appendChild(panel);

      buttons.push(button);
      panels.push(panel);
      valueByButton.set(button, valueForTab);
      if (!buttonByValue.has(valueForTab)) {
        buttonByValue.set(valueForTab, button);
      }
      idDirectory.set(tabId, { section: section, button: button, panel: panel });
      idDirectory.set(panelId, { section: section, button: button, panel: panel });
    });

    const parent = block.startComment.parentNode;
    if (parent) {
      parent.insertBefore(section, block.startComment);
      while (
        block.startComment.nextSibling &&
        block.startComment.nextSibling !== block.endComment
      ) {
        parent.removeChild(block.startComment.nextSibling);
      }
      parent.removeChild(block.startComment);
    }

    if (block.endComment.parentNode) {
      block.endComment.parentNode.removeChild(block.endComment);
    }

    return {
      section: section,
      tablist: tabsNav,
      buttons: buttons,
      panels: panels,
      title: block.title,
      syncGroup: block.syncGroup || null,
      valueByButton: valueByButton,
      buttonByValue: buttonByValue,
      copyButton: copyButton
    };
  }

  function createCopyButton(doc) {
    const button = doc.createElement('button');
    button.type = 'button';
    button.className = 'copy-request codeblocks-copy';
    button.setAttribute('aria-label', 'Click to copy');
    button.setAttribute('data-microtip-position', 'left');
    button.setAttribute('role', 'tooltip');
    button.innerHTML = `
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
        <path d="M16.5 8.25V6a2.25 2.25 0 0 0-2.25-2.25H6A2.25 2.25 0 0 0 3.75 6v8.25A2.25 2.25 0 0 0 6 16.5h2.25m8.25-8.25H18a2.25 2.25 0 0 1 2.25 2.25V18A2.25 2.25 0 0 1 18 20.25h-7.5A2.25 2.25 0 0 1 8.25 18v-1.5m8.25-8.25h-6a2.25 2.25 0 0 0-2.25 2.25v6" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    `;
    return button;
  }

  async function handleCopy(section, button) {
    const state = tabDataBySection.get(section);
    if (!state) {
      return;
    }

    const panel = state.activePanel || state.panels[0];
    if (!panel) {
      return;
    }

    const code = extractPanelCode(panel);
    if (!code) {
      return;
    }

    if (
      typeof navigator === 'undefined' ||
      !navigator.clipboard ||
      typeof navigator.clipboard.writeText !== 'function'
    ) {
      provideCopyFeedback(button, false);
      return;
    }

    try {
      await navigator.clipboard.writeText(code);
      provideCopyFeedback(button, true);
    } catch (error) {
      console.error('Cannot copy codeblocks content:', error);
      provideCopyFeedback(button, false);
    }
  }

  function extractPanelCode(panel) {
    if (!panel) {
      return null;
    }
    const codeElement = panel.querySelector('pre code');
    if (codeElement && codeElement.textContent) {
      return normalizeCodeText(codeElement.textContent);
    }
    const preElement = panel.querySelector('pre');
    if (preElement && preElement.textContent) {
      return normalizeCodeText(preElement.textContent);
    }
    return null;
  }

  function normalizeCodeText(value) {
    return String(value).replace(/\u00A0/g, ' ').replace(/\r\n/g, '\n');
  }

  function provideCopyFeedback(button, success) {
    if (!button) {
      return;
    }
    const originalLabel =
      button.dataset.microtipOriginalLabel ||
      button.getAttribute('aria-label') ||
      'Copy code';
    button.dataset.microtipOriginalLabel = originalLabel;
    button.setAttribute('aria-label', success ? 'Copied' : 'Copy failed');

    if (button[COPY_RESET_TIMER] != null && typeof win.clearTimeout === 'function') {
      win.clearTimeout(button[COPY_RESET_TIMER]);
    }

    const resetTooltip = function () {
      if (success && typeof button.blur === 'function') {
        button.blur();
      }
      button.setAttribute('aria-label', originalLabel);
      delete button[COPY_RESET_TIMER];
    };

    if (typeof win.setTimeout !== 'function') {
      resetTooltip();
      return;
    }

    button[COPY_RESET_TIMER] = win.setTimeout(resetTooltip, 3000);
  }

  function getSyncValue(group) {
    if (!group || !syncState) {
      return null;
    }
    if (Object.prototype.hasOwnProperty.call(syncState, group)) {
      return syncState[group];
    }
    return null;
  }

  function persistGroupValue(group, value) {
    if (!storage || !group) {
      return;
    }
    const current = getSyncValue(group);
    if (current === value) {
      return;
    }
    if (value) {
      syncState[group] = value;
    } else {
      delete syncState[group];
    }
    saveSyncState(storage, STORAGE_KEY, syncState);
  }

  function synchronizeGroup(group, value, originSection) {
    if (!group || !value) {
      return;
    }
    const sections = groupRegistry.get(group);
    if (!sections || !sections.length) {
      return;
    }
    sections.forEach(function (section) {
      if (section === originSection) {
        return;
      }
      const state = tabDataBySection.get(section);
      if (!state || !state.buttonByValue) {
        return;
      }
      const targetButton = state.buttonByValue.get(value);
      if (targetButton && state.activeButton !== targetButton) {
        activateTab(section, targetButton, { skipHash: true, skipSync: true });
      }
    });
  }

  function resolveHash(hash) {
    if (!hash) {
      return null;
    }

    const raw = hash.charAt(0) === '#' ? hash.slice(1) : hash;
    if (!raw) {
      return null;
    }

    let decoded;
    try {
      decoded = decodeURIComponent(raw);
    } catch (error) {
      return null;
    }
    return idDirectory.get(decoded) || null;
  }

  const tabSets = collectCodeblockSets();
  if (!tabSets.length) {
    return [];
  }

  tabSets.forEach(function (set) {
    tabDataBySection.set(set.section, {
      buttons: set.buttons,
      panels: set.panels,
      activeButton: null,
      activeValue: null,
      activePanel: null,
      syncGroup: set.syncGroup,
      valueByButton: set.valueByButton,
      buttonByValue: set.buttonByValue,
      copyButton: set.copyButton
    });
    set.section.dataset.tabsInitialized = 'true';

    if (set.syncGroup) {
      if (!groupRegistry.has(set.syncGroup)) {
        groupRegistry.set(set.syncGroup, []);
      }
      groupRegistry.get(set.syncGroup).push(set.section);
    }

    set.buttons.forEach(function (button) {
      button.addEventListener('click', function () {
        activateTab(set.section, button);
      });
    });

    if (set.copyButton) {
      set.copyButton.addEventListener('click', function () {
        handleCopy(set.section, set.copyButton);
      });
    }

    set.tablist.addEventListener('keydown', function (event) {
      if (
        !(event.target instanceof environment.htmlElement) ||
        event.target.getAttribute('role') !== 'tab'
      ) {
        return;
      }

      const state = tabDataBySection.get(set.section);
      if (!state) {
        return;
      }
      const buttons = state.buttons;
      const currentIndex = buttons.indexOf(event.target);
      if (currentIndex === -1) {
        return;
      }

      switch (event.key) {
        case 'ArrowLeft':
        case 'Left':
          event.preventDefault();
          focusTab(buttons, (currentIndex - 1 + buttons.length) % buttons.length);
          break;
        case 'ArrowRight':
        case 'Right':
          event.preventDefault();
          focusTab(buttons, (currentIndex + 1) % buttons.length);
          break;
        case 'Home':
          event.preventDefault();
          focusTab(buttons, 0);
          break;
        case 'End':
          event.preventDefault();
          focusTab(buttons, buttons.length - 1);
          break;
        case 'Enter':
        case ' ':
        case 'Spacebar':
          event.preventDefault();
          activateTab(set.section, event.target, { focus: true });
          break;
        default:
          break;
      }
    });
  });

  const initialTarget = resolveHash(win.location.hash);
  const groupInitialSelections = new Map();

  tabSets.forEach(function (set) {
    const state = tabDataBySection.get(set.section);
    if (!state || !state.buttons.length) {
      return;
    }

    if (initialTarget && initialTarget.section === set.section) {
      return;
    }

    if (state.activeButton) {
      return;
    }

    let targetButton = state.buttons[0];
    let skipSync = false;

    if (state.syncGroup) {
      const storedValue = getSyncValue(state.syncGroup);
      if (storedValue) {
        const storedButton = set.buttonByValue.get(storedValue);
        if (storedButton) {
          targetButton = storedButton;
          skipSync = true;
          groupInitialSelections.set(state.syncGroup, storedValue);
        }
      } else {
        const existingValue = groupInitialSelections.get(state.syncGroup);
        if (existingValue) {
          const matchingButton = set.buttonByValue.get(existingValue);
          if (matchingButton) {
            targetButton = matchingButton;
            skipSync = true;
          }
        }
      }
    }

    if (!targetButton) {
      return;
    }

    activateTab(set.section, targetButton, { skipHash: true, skipSync: skipSync });

    if (state.syncGroup && !skipSync) {
      const currentValue = state.valueByButton
        ? state.valueByButton.get(targetButton)
        : null;
      if (currentValue) {
        groupInitialSelections.set(state.syncGroup, currentValue);
      }
    }
  });

  if (initialTarget) {
    activateTab(initialTarget.section, initialTarget.button, { skipHash: true });
  }

  const hashHandler = function () {
    const hashTarget = resolveHash(win.location.hash);
    if (hashTarget) {
      activateTab(hashTarget.section, hashTarget.button, { skipHash: true, focus: true });
    }
  };

  if (typeof win.removeEventListener === 'function') {
    const existingHandler = hashHandlerRegistry.get(win);
    if (existingHandler) {
      win.removeEventListener('hashchange', existingHandler);
    }
  }

  if (typeof win.addEventListener === 'function') {
    win.addEventListener('hashchange', hashHandler);
    hashHandlerRegistry.set(win, hashHandler);
  }

  return tabSets;
}

if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function handleReady() {
      document.removeEventListener('DOMContentLoaded', handleReady);
      buildCodeblocks();
    });
  } else {
    buildCodeblocks();
  }
}

function resolveEnvironment(win) {
  const fallbackNode = typeof Node !== 'undefined' ? Node : null;
  const nodeApi = win.Node || fallbackNode;
  const elementNode = nodeApi ? nodeApi.ELEMENT_NODE : 1;
  const commentNode = nodeApi ? nodeApi.COMMENT_NODE : 8;
  const textNode = nodeApi ? nodeApi.TEXT_NODE : 3;

  const commentFilter =
    (win.NodeFilter && typeof win.NodeFilter.SHOW_COMMENT === 'number'
      ? win.NodeFilter.SHOW_COMMENT
      : typeof NodeFilter !== 'undefined'
        ? NodeFilter.SHOW_COMMENT
        : 0x80);

  return {
    htmlElement: win.HTMLElement || Object,
    elementNode: elementNode,
    commentNode: commentNode,
    textNode: textNode,
    commentFilter: commentFilter
  };
}

function collectExistingIds(doc) {
  const ids = new Set();
  const elements = doc.querySelectorAll('[id]');
  for (const element of elements) {
    if (element.id) {
      ids.add(element.id);
    }
  }
  return ids;
}

function isEndMarker(commentNode) {
  if (
    !commentNode ||
    commentNode.nodeType !== 8
  ) {
    return false;
  }
  const value = (commentNode.nodeValue || '').trim().toLowerCase();
  return value === 'end codeblocks' || value === 'end';
}

function resolveStorage(win) {
  if (!win) {
    return null;
  }
  try {
    if (!('localStorage' in win)) {
      return null;
    }
    const storage = win.localStorage;
    if (!storage) {
      return null;
    }
    const probeKey = '__codeblocks-sync-test__';
    storage.setItem(probeKey, '1');
    storage.removeItem(probeKey);
    return storage;
  } catch (error) {
    return null;
  }
}

function loadSyncState(storage, key) {
  try {
    const raw = storage.getItem(key);
    if (!raw) {
      return Object.create(null);
    }
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === 'object') {
      return Object.assign(Object.create(null), parsed);
    }
  } catch (error) {
    // ignore storage decoding failures
  }
  return Object.create(null);
}

function saveSyncState(storage, key, value) {
  try {
    storage.setItem(key, JSON.stringify(value));
  } catch (error) {
    // ignore storage write failures
  }
}

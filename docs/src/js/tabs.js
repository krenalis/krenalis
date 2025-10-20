/* eslint-disable no-console */
const hashHandlerRegistry = new WeakMap();

/**
 * Enhance Markdown-authored tab comments into accessible tab widgets.
 * @param {Document} [doc=document]
 * @param {Window} [win=window]
 * @returns {Array<{section: HTMLElement, buttons: HTMLButtonElement[], panels: HTMLElement[]}>}
 */
export function buildTabs(doc = document, win = window) {
  if (!doc || !win || !doc.body) {
    return [];
  }

  const environment = resolveEnvironment(win);
  const usedIds = collectExistingIds(doc);
  const tabDataBySection = new WeakMap();
  const idDirectory = new Map();

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

  function getPlainText(element) {
    return element ? element.textContent.trim() : '';
  }

  const inherentlyContentfulTagNames = new Set([
    'audio',
    'canvas',
    'embed',
    'iframe',
    'img',
    'object',
    'picture',
    'svg',
    'video'
  ]);

  function hasMeaningfulContent(container) {
    if (!container || !container.childNodes || !container.childNodes.length) {
      return false;
    }

    for (const node of container.childNodes) {
      if (nodeHasMeaningfulContent(node)) {
        return true;
      }
    }
    return false;
  }

  function nodeHasMeaningfulContent(node) {
    if (!node) {
      return false;
    }
    if (node.nodeType === environment.elementNode) {
      const tagName = (node.nodeName || '').toLowerCase();
      if (inherentlyContentfulTagNames.has(tagName)) {
        return true;
      }
      if (node.textContent && node.textContent.trim().length > 0) {
        return true;
      }
      return hasMeaningfulContent(node);
    }
    if (node.nodeType === 3) {
      return Boolean(node.textContent && node.textContent.trim().length > 0);
    }
    return false;
  }

  function disableTabButton(button, panel) {
    if (!button) {
      return;
    }
    button.setAttribute('aria-disabled', 'true');
    button.setAttribute('tabindex', '-1');
    if (button.dataset) {
      button.dataset.tabsDisabled = 'true';
    }
    if (panel) {
      if (panel.dataset) {
        panel.dataset.tabsDisabled = 'true';
      }
      panel.hidden = true;
    }
  }

  function isTabDisabled(button) {
    if (!button) {
      return false;
    }
    if (button.dataset && button.dataset.tabsDisabled === 'true') {
      return true;
    }
    return button.getAttribute('aria-disabled') === 'true';
  }

  function updateTabFocusState(buttons, activeButton) {
    const active = isTabDisabled(activeButton) ? null : activeButton;
    buttons.forEach(function (button) {
      const disabled = isTabDisabled(button);
      let tabIndex = '-1';
      if (!disabled && button === active) {
        tabIndex = '0';
      }
      button.setAttribute('tabindex', tabIndex);
    });
  }

  function findFirstEnabledIndex(buttons) {
    for (let index = 0; index < buttons.length; index += 1) {
      if (!isTabDisabled(buttons[index])) {
        return index;
      }
    }
    return -1;
  }

  function findLastEnabledIndex(buttons) {
    for (let index = buttons.length - 1; index >= 0; index -= 1) {
      if (!isTabDisabled(buttons[index])) {
        return index;
      }
    }
    return -1;
  }

  function findNextEnabledIndex(buttons, startIndex, direction) {
    if (!buttons.length) {
      return -1;
    }
    let steps = 0;
    let candidate = startIndex;
    while (steps < buttons.length) {
      candidate = (candidate + direction + buttons.length) % buttons.length;
      if (!isTabDisabled(buttons[candidate])) {
        return candidate;
      }
      steps += 1;
    }
    return -1;
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
    const target = buttons[index];
    if (!target || isTabDisabled(target)) {
      return;
    }
    updateTabFocusState(buttons, target);
    target.focus();
  }

  function activateTab(setEl, tabButton, options) {
    const opts = options || {};
    const tabState = tabDataBySection.get(setEl);
    if (!tabState) {
      return;
    }

    const targetPanelId = tabButton.getAttribute('aria-controls');
    if (!targetPanelId) {
      return;
    }

    if (isTabDisabled(tabButton)) {
      return;
    }

    let targetPanel = null;
    tabState.panels.forEach(function (panel) {
      if (panel.id === targetPanelId) {
        targetPanel = panel;
      }
    });

    if (!targetPanel) {
      return;
    }

    if (tabState.activeButton === tabButton) {
      if (opts.focus) {
        tabButton.focus();
      }
      if (!opts.skipHash) {
        updateHash(targetPanelId);
      }
      return;
    }

    tabState.buttons.forEach(function (button) {
      const isActive = button === tabButton;
      button.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });

    updateTabFocusState(tabState.buttons, tabButton);

    tabState.panels.forEach(function (panel) {
      panel.hidden = panel !== targetPanel;
    });

    tabState.activeButton = tabButton;

    if (opts.focus) {
      tabButton.focus();
    }
    if (!opts.skipHash) {
      updateHash(targetPanelId);
    }
  }

  function collectTabBlocks() {
    const blocks = [];
    const openStack = []; // track nested tab markers so innermost blocks resolve first
    const walker = doc.createTreeWalker(doc.body, environment.commentFilter);
    let currentComment = walker.nextNode();

    while (currentComment) {
      const raw = (currentComment.nodeValue || '').trim();
      const openingMarker = parseTabsMarker(raw);

      if (openingMarker !== null) {
        openStack.push({
          startComment: currentComment,
          rawTitle: openingMarker.title,
          title: openingMarker.title ? openingMarker.title : 'Tabs',
          options: openingMarker.options || Object.create(null)
        });
      } else if (isTabsEndMarker(currentComment)) {
        const activeBlock = openStack.pop();
        if (activeBlock) {
          // closing marker: capture the innermost block before unwinding parents
          blocks.push({
            title: activeBlock.title,
            options: activeBlock.options || Object.create(null),
            startComment: activeBlock.startComment,
            endComment: currentComment
          });
        }
      }

      currentComment = walker.nextNode();
    }

    if (openStack.length) {
      openStack.forEach(function (entry) {
        console.warn(
          'Skipping tabs set "%s" because no closing marker was found.',
          entry.rawTitle || '(untitled)'
        );
      });
    }

    if (!blocks.length) {
      return [];
    }

    const tabSets = [];
    blocks.forEach(function (block) {
      try {
        const result = transformBlock(block);
        if (result) {
          tabSets.push(result);
        }
      } catch (error) {
        console.warn(
          'Skipping tabs set "%s" due to an unexpected error.',
          block.title
        );
      }
    });

    if (tabSets.length > 1) {
      tabSets.sort(function (a, b) {
        return compareNodeOrder(a.section, b.section);
      });
    }

    return tabSets;
  }

  function transformBlock(block) {
    if (
      !block ||
      !block.startComment ||
      !block.endComment ||
      !block.startComment.parentNode ||
      !block.endComment.parentNode
    ) {
      return null;
    }

    const nodes = collectNodesBetween(block.startComment, block.endComment);
    const tabs = [];
    let headingTag = null;
    let currentTab = null;

    nodes.forEach(function (node) {
      if (node.nodeType === environment.elementNode) {
        const tagName = node.nodeName.toLowerCase();
        const isCandidate = tagName === 'h3' || tagName === 'h4';

        if (isCandidate && !headingTag) {
          headingTag = tagName;
        }

        if (isCandidate && headingTag && tagName === headingTag) {
          const labelText = getPlainText(node);
          currentTab = {
            heading: node,
            label: labelText || 'Tab ' + (tabs.length + 1),
            content: []
          };
          tabs.push(currentTab);
          return;
        }
      }

      if (currentTab) {
        currentTab.content.push(node);
      }
    });

    if (!tabs.length) {
      console.warn(
        'Skipping tabs set "%s" because no tab headings were found.',
        block.title
      );
      return null;
    }

    if (tabs.length < 2) {
      console.warn(
        'Skipping tabs set "%s" because it has fewer than two tabs.',
        block.title
      );
      return null;
    }

    const section = doc.createElement('section');
    section.className = 'tabs';
    section.setAttribute('aria-label', block.title);

    const panelBackground =
      block.options && typeof block.options.bg === 'string'
        ? block.options.bg
        : null;
    if (panelBackground) {
      section.dataset.tabsBackground = panelBackground;
      if (section.style && typeof section.style.setProperty === 'function') {
        section.style.setProperty('--tabs-panel-background', panelBackground);
      }
    }

    const tabsNav = doc.createElement('div');
    tabsNav.className = 'tabs-nav';
    tabsNav.setAttribute('role', 'tablist');
    tabsNav.setAttribute('aria-orientation', 'horizontal');
    section.appendChild(tabsNav);

    const buttons = [];
    const panels = [];
    const titleSlug = slugify(block.title, 'tabs');

    tabs.forEach(function (tabInfo, index) {
      const labelSlug = slugify(tabInfo.label, 'tab');
      const tabId = uniqueId('tab-' + titleSlug + '-' + labelSlug);
      const panelId = uniqueId('panel-' + titleSlug + '-' + labelSlug);

      const button = doc.createElement('button');
      button.type = 'button';
      button.setAttribute('role', 'tab');
      button.id = tabId;
      button.setAttribute('aria-controls', panelId);
      button.setAttribute('aria-selected', 'false');
      button.setAttribute('tabindex', '-1');
      button.textContent = tabInfo.label;

      const panel = doc.createElement('div');
      panel.setAttribute('role', 'tabpanel');
      panel.id = panelId;
      panel.setAttribute('aria-labelledby', tabId);
      panel.hidden = true;
      if (panelBackground) {
        panel.dataset.tabsBackground = panelBackground;
        panel.style.background = panelBackground;
        if (typeof panel.style.setProperty === 'function') {
          panel.style.setProperty('background-color', panelBackground);
        } else {
          panel.style.backgroundColor = panelBackground;
        }
      }

      tabInfo.content.forEach(function (node) {
        panel.appendChild(node);
      });

      if (tabInfo.heading && tabInfo.heading.parentNode) {
        tabInfo.heading.parentNode.removeChild(tabInfo.heading);
      }

      if (!hasMeaningfulContent(panel)) {
        disableTabButton(button, panel);
      }

      tabsNav.appendChild(button);
      section.appendChild(panel);

      buttons.push(button);
      panels.push(panel);
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
      title: block.title
    };
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
    const entry = idDirectory.get(decoded) || null;
    if (!entry || !entry.button || isTabDisabled(entry.button)) {
      return null;
    }
    return entry;
  }

  const tabSets = collectTabBlocks();
  if (!tabSets.length) {
    return [];
  }

  tabSets.forEach(function (set) {
    tabDataBySection.set(set.section, {
      buttons: set.buttons,
      panels: set.panels,
      activeButton: null
    });
    set.section.dataset.tabsInitialized = 'true';

    set.buttons.forEach(function (button) {
      button.addEventListener('click', function (event) {
        if (isTabDisabled(button)) {
          if (event && typeof event.preventDefault === 'function') {
            event.preventDefault();
          }
          return;
        }
        activateTab(set.section, button);
      });
    });

    set.tablist.addEventListener('keydown', function (event) {
      if (
        !(event.target instanceof environment.htmlElement) ||
        event.target.getAttribute('role') !== 'tab'
      ) {
        return;
      }

      const buttons = tabDataBySection.get(set.section).buttons;
      const currentIndex = buttons.indexOf(event.target);
      if (currentIndex === -1) {
        return;
      }

      if (isTabDisabled(buttons[currentIndex])) {
        return;
      }

      switch (event.key) {
        case 'ArrowLeft':
        case 'Left':
          event.preventDefault();
          const prevIndex = findNextEnabledIndex(buttons, currentIndex, -1);
          if (prevIndex !== -1) {
            focusTab(buttons, prevIndex);
          }
          break;
        case 'ArrowRight':
        case 'Right':
          event.preventDefault();
          const nextIndex = findNextEnabledIndex(buttons, currentIndex, 1);
          if (nextIndex !== -1) {
            focusTab(buttons, nextIndex);
          }
          break;
        case 'Home':
          event.preventDefault();
          const firstIndex = findFirstEnabledIndex(buttons);
          if (firstIndex !== -1) {
            focusTab(buttons, firstIndex);
          }
          break;
        case 'End':
          event.preventDefault();
          const lastIndex = findLastEnabledIndex(buttons);
          if (lastIndex !== -1) {
            focusTab(buttons, lastIndex);
          }
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
  tabSets.forEach(function (set) {
    if (initialTarget && initialTarget.section === set.section) {
      return;
    }
    const firstEnabledIndex = findFirstEnabledIndex(set.buttons);
    if (firstEnabledIndex !== -1) {
      activateTab(set.section, set.buttons[firstEnabledIndex], { skipHash: true });
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
      buildTabs();
    });
  } else {
    buildTabs();
  }
}

function resolveEnvironment(win) {
  const fallbackNode = typeof Node !== 'undefined' ? Node : null;
  const nodeApi = win.Node || fallbackNode;
  const elementNode = nodeApi ? nodeApi.ELEMENT_NODE : 1;
  const commentNode = nodeApi ? nodeApi.COMMENT_NODE : 8;

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

function collectNodesBetween(startNode, endNode) {
  const nodes = [];
  if (!startNode || !endNode) {
    return nodes;
  }
  let cursor = startNode.nextSibling;
  while (cursor && cursor !== endNode) {
    nodes.push(cursor);
    cursor = cursor.nextSibling;
  }
  return nodes;
}

function parseTabsMarker(rawValue) {
  if (!rawValue) {
    return null;
  }
  const match = rawValue.match(/^tabs(?:\s+(.*))?$/i);
  if (!match) {
    return null;
  }
  const remainder = (match[1] || '').trim();
  if (!remainder) {
    return {
      title: '',
      options: Object.create(null)
    };
  }

  let cursor = remainder;
  const options = Object.create(null);
  const optionPattern = /^([a-z0-9_-]+)\s*:\s*([^\s]+)\s*/i;
  let optionMatch = optionPattern.exec(cursor);
  while (optionMatch) {
    const key = optionMatch[1].toLowerCase();
    const value = optionMatch[2];
    options[key] = value;
    cursor = cursor.slice(optionMatch[0].length);
    cursor = cursor.replace(/^\s+/, '');
    optionMatch = optionPattern.exec(cursor);
  }

  return {
    title: cursor.trim(),
    options: options
  };
}

function isTabsEndMarker(commentNode) {
  if (!commentNode || commentNode.nodeType !== 8) {
    return false;
  }
  const value = (commentNode.nodeValue || '').trim().toLowerCase();
  return value === 'end tabs' || value === 'end';
}

function compareNodeOrder(nodeA, nodeB) {
  if (nodeA === nodeB) {
    return 0;
  }
  if (!nodeA) {
    return 1;
  }
  if (!nodeB) {
    return -1;
  }
  if (typeof nodeA.compareDocumentPosition === 'function') {
    const position = nodeA.compareDocumentPosition(nodeB);
    if (position & 4) {
      return -1;
    }
    if (position & 2) {
      return 1;
    }
  } else if (typeof nodeB.compareDocumentPosition === 'function') {
    const position = nodeB.compareDocumentPosition(nodeA);
    if (position & 4) {
      return 1;
    }
    if (position & 2) {
      return -1;
    }
  }
  return 0;
}

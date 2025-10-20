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
      button.setAttribute('tabindex', isActive ? '0' : '-1');
    });

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
          rawTitle: openingMarker,
          title: openingMarker ? openingMarker : 'Tabs'
        });
      } else if (isTabsEndMarker(currentComment)) {
        const activeBlock = openStack.pop();
        if (activeBlock) {
          // closing marker: capture the innermost block before unwinding parents
          blocks.push({
            title: activeBlock.title,
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
      button.setAttribute('tabindex', index === 0 ? '0' : '-1');
      button.textContent = tabInfo.label;

      const panel = doc.createElement('div');
      panel.setAttribute('role', 'tabpanel');
      panel.id = panelId;
      panel.setAttribute('aria-labelledby', tabId);
      panel.hidden = true;

      tabInfo.content.forEach(function (node) {
        panel.appendChild(node);
      });

      if (tabInfo.heading && tabInfo.heading.parentNode) {
        tabInfo.heading.parentNode.removeChild(tabInfo.heading);
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
    return idDirectory.get(decoded) || null;
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
      button.addEventListener('click', function () {
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
  tabSets.forEach(function (set) {
    if (initialTarget && initialTarget.section === set.section) {
      return;
    }
    activateTab(set.section, set.buttons[0], { skipHash: true });
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
  return (match[1] || '').trim();
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

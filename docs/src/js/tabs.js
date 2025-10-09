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
    const walker = doc.createTreeWalker(doc.body, environment.commentFilter);
    let currentComment = walker.nextNode();

    while (currentComment) {
      const raw = (currentComment.nodeValue || '').trim();
      const marker = raw.match(/^tabs:\s*(.+)$/i);
      if (marker) {
        const title = marker[1].trim();
        const nodes = [];
        let cursor = currentComment.nextSibling;
        let closingMarker = null;
        while (cursor) {
          if (
            cursor.nodeType === environment.commentNode &&
            (cursor.nodeValue || '').trim() === '/tabs'
          ) {
            closingMarker = cursor;
            break;
          }
          nodes.push(cursor);
          cursor = cursor.nextSibling;
        }

        if (!closingMarker) {
          console.warn(
            'Skipping tabs set "%s" because no closing marker was found.',
            title || '(untitled)'
          );
        } else {
          blocks.push({
            title: title || 'Tabs',
            nodes: nodes.slice(),
            startComment: currentComment,
            endComment: closingMarker
          });
          walker.currentNode = closingMarker;
        }
      }
      currentComment = walker.nextNode();
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

    return tabSets;
  }

  function transformBlock(block) {
    const tabs = [];
    let currentTab = null;

    block.nodes.forEach(function (node) {
      if (
        node.nodeType === environment.elementNode &&
        node.nodeName.toLowerCase() === 'h3'
      ) {
        const labelText = getPlainText(node);
        currentTab = {
          heading: node,
          label: labelText || 'Tab ' + (tabs.length + 1),
          content: []
        };
        tabs.push(currentTab);
      } else if (currentTab) {
        currentTab.content.push(node);
      }
    });

    if (tabs.length < 2) {
      console.warn(
        'Skipping tabs set "%s" because it has fewer than two tabs.',
        block.title
      );
      return null;
    }

    const section = doc.createElement('section');
    section.className = 'tabset';
    section.setAttribute('aria-label', block.title);

    const tablist = doc.createElement('div');
    tablist.className = 'tablist';
    tablist.setAttribute('role', 'tablist');
    tablist.setAttribute('aria-orientation', 'horizontal');
    section.appendChild(tablist);

    const buttons = [];
    const panels = [];
    const titleSlug = slugify(block.title, 'tabset');

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

      tablist.appendChild(button);
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
      tablist: tablist,
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
    set.section.dataset.tabsetInitialized = 'true';

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

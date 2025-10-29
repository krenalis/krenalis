// Synchronize any screenshot (or image) with its associated tabs set.
const marker = '.';

function slugify(value) {
  if (value == null) {
    return '';
  }
  let text = String(value);
  if (typeof text.normalize === 'function') {
    text = text.normalize('NFD').replace(/[\u0300-\u036f]/g, '');
  }
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .replace(/--+/g, '-');
}

function partitionsBy(dataTabs) {
  const registry = new Map();
  dataTabs.forEach((image) => {
    const key = image.dataset.tabs;
    if (!key) {
      return;
    }
    if (!registry.has(key)) {
      registry.set(key, []);
    }
    registry.get(key).push(image);
  });
  return registry;
}

function createUpdater(entry) {
  const { tabsSection, images = [], textTargets = [] } = entry;
  const titleSlug = slugify(tabsSection.getAttribute('aria-label') || '');

  function getSrcParts(image) {
    const srcValue = image.getAttribute('src') || '';
    const slashIndex = srcValue.lastIndexOf('/');
    const base = slashIndex === -1 ? srcValue : srcValue.slice(slashIndex + 1);
    const first = base.indexOf(marker);
    const last = base.lastIndexOf(marker);
    if (first === -1 || last === -1 || last <= first) {
      return null;
    }
    const firstIndex = (slashIndex === -1 ? 0 : slashIndex + 1) + first;
    const lastIndex = (slashIndex === -1 ? 0 : slashIndex + 1) + last;
    return {
      prefix: srcValue.slice(0, firstIndex + 1),
      suffix: srcValue.slice(lastIndex)
    };
  }

  const srcPartsByImage = new Map();
  images.forEach((image) => {
    const parts = getSrcParts(image);
    if (parts) {
      srcPartsByImage.set(image, parts);
    }
  });

  function getLabelSlug(button) {
    if (!button || !button.id) {
      return '';
    }
    const idValue = button.id;
    if (titleSlug) {
      const prefix = 'tab-' + titleSlug + '-';
      if (idValue.indexOf(prefix) === 0) {
        return idValue.slice(prefix.length);
      }
    }
    const lastDot = idValue.lastIndexOf(marker);
    if (lastDot !== -1) {
      return idValue.slice(lastDot + 1);
    }
    return slugify(button.textContent || '');
  }

  function updateTargets() {
    const activeButton = tabsSection.querySelector('[role="tab"][aria-selected="true"]');
    if (!activeButton) {
      return;
    }
    const labelSlug = getLabelSlug(activeButton);
    const labelText = activeButton.textContent || '';

    if (labelSlug) {
      images.forEach((image) => {
        const parts = srcPartsByImage.get(image);
        if (!parts) {
          return;
        }
        const nextSrc = parts.prefix + labelSlug + parts.suffix;
        if (image.getAttribute('src') !== nextSrc) {
          image.setAttribute('src', nextSrc);
        }
      });
    }

    if (textTargets.length) {
      const nextText = labelText || '';
      textTargets.forEach((element) => {
        if (element.textContent !== nextText) {
          element.textContent = nextText;
        }
      });
    }
  }

  return updateTargets;
}

function init() {
  const imageCandidates = Array.from(document.querySelectorAll('img[data-tabs]'));
  const textCandidates = Array.from(document.querySelectorAll('div[data-tabs], span[data-tabs]'));
  if (!imageCandidates.length && !textCandidates.length) {
    return;
  }

  const imagesByKey = partitionsBy(imageCandidates);
  const textByKey = partitionsBy(textCandidates);
  const keys = new Set([...imagesByKey.keys(), ...textByKey.keys()]);
  if (!keys.size) {
    return;
  }

  keys.forEach((key) => {
    const tabsSection = document.getElementById(key);
    if (!tabsSection) {
      return;
    }
    const images = imagesByKey.get(key) || [];
    const textTargets = textByKey.get(key) || [];
    if (!images.length && !textTargets.length) {
      return;
    }
    const updateTargets = createUpdater({ tabsSection, images, textTargets });
    const observer = new MutationObserver((mutations) => {
      for (let index = 0; index < mutations.length; index += 1) {
        const mutation = mutations[index];
        if (mutation.type === 'attributes' && mutation.attributeName === 'aria-selected') {
          updateTargets();
          break;
        }
      }
    });

    observer.observe(tabsSection, {
      subtree: true,
      attributes: true,
      attributeFilter: ['aria-selected']
    });

    updateTargets();
  });
}

if (document.readyState === 'complete') {
  init();
} else {
  document.addEventListener('DOMContentLoaded', init, { once: true });
}

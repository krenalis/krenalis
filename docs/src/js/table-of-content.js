// buildTableOfContent renders the documentation table of contents.

const SCROLL_THRESHOLD_PX = 360;
const DEFAULT_TOC_TITLE = "On this page";

// Allow the final heading to activate when we reach within this many pixels of the page bottom.
const PAGE_END_TOLERANCE_PX = 4;

// Keys that indicate the user is manually scrolling and should unlock the active highlight.
const SCROLL_KEYS = new Set([
    "ArrowUp",
    "ArrowDown",
    "PageUp",
    "PageDown",
    "Home",
    "End",
    " ",
    "Spacebar",
]);

const buildTableOfContent = () => {
    const main = document.querySelector("body > article > .wrap > .main");
    // Abort if the documentation layout is not present.
    if (main == null) {
        return;
    }

    const pageHeading = main.querySelector("h1");
    const pageTitleText = pageHeading?.textContent?.trim() ?? "";
    const tocTitleText = pageTitleText === "" ? DEFAULT_TOC_TITLE : pageTitleText;

    let headings = Array.from(main.querySelectorAll("h2, h3"));
    if (pageHeading != null) {
        const firstHeading = headings[0] ?? null;
        if (
            firstHeading != null &&
            firstHeading.tagName.toLowerCase() === "h2" &&
            pageHeading.nextElementSibling === firstHeading
        ) {
            headings = headings.slice(1);
        }
    }
    // Do not render an empty TOC when the page has no headings.
    if (headings.length === 0) {
        return;
    }

    const container = document.createElement("nav");
    container.className = "table-of-content";
    container.setAttribute("aria-label", tocTitleText);

    const title = document.createElement("h2");
    title.className = "table-of-content__title";
    const titleLink = document.createElement("a");
    titleLink.className = "table-of-content__title-link";
    titleLink.href = "#";
    titleLink.textContent = tocTitleText;
    title.appendChild(titleLink);
    container.appendChild(title);

    const list = document.createElement("ol");
    list.className = "table-of-content__list";
    container.appendChild(list);

    const hasH2 = headings.some((heading) => heading.tagName.toLowerCase() === "h2");
    const hasH3 = headings.some((heading) => heading.tagName.toLowerCase() === "h3");
    // Only indent H3 links when both H2 and H3 headings are present.
    const shouldIndentH3 = hasH2 && hasH3;

    let activeId = null;
    let lockedActiveId = null;
    let anchorsById;
    let headingById;

    // Guarantee each heading has a stable id so we can link to it.
    const ensureHeadingId = (heading, index) => {
        if (heading.id === "") {
            heading.id = `toc-${heading.tagName.toLowerCase()}-${index}`;
        }

        return heading.id;
    };

    // Map HTML tag names to the TOC indentation classes.
    const normalizeLevel = (tagName) => {
        const level = tagName.toLowerCase();
        return shouldIndentH3 && level === "h3" ? "h3" : "h2";
    };

    // Build an anchor element for a given heading.
    const createAnchor = (text, level, id) => {
        const anchor = document.createElement("a");
        anchor.textContent = text;
        anchor.classList.add("toc-item", `toc-item--${level}`);
        anchor.dataset.target = id;
        return anchor;
    };

    const createListItem = () => {
        const item = document.createElement("li");
        item.className = "table-of-content__item";
        return item;
    };

    // Retrieve a heading node by id, preferring the cached reference.
    const getTargetHeading = (id) => {
        if (id == null) {
            return null;
        }

        return headingById.get(id) ?? document.getElementById(id);
    };

    // Toggle the active TOC entry, removing any previous highlight.
    const setActiveAnchor = (id) => {
        if (id == null || id === activeId) {
            return;
        }

        if (activeId != null) {
            const previousAnchor = anchorsById.get(activeId);
            if (previousAnchor != null) {
                previousAnchor.classList.remove("is-active");
            }
        }

        const nextAnchor = anchorsById.get(id);
        if (nextAnchor != null) {
            nextAnchor.classList.add("is-active");
            activeId = id;
        }
    };

    const lockActiveAnchor = (id) => {
        lockedActiveId = id;
    };

    const releaseActiveLock = () => {
        lockedActiveId = null;
    };

    // Smoothly scroll the page so the heading lands below the fixed header.
    const scrollToHeading = (heading) => {
        const headingStyles = window.getComputedStyle(heading);
        const scrollMarginTop = parseFloat(headingStyles.scrollMarginTop || "0") || 0;
        const targetTop = heading.getBoundingClientRect().top + window.scrollY - scrollMarginTop;

        const prefersReducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches;
        window.scrollTo({
            top: targetTop,
            behavior: prefersReducedMotion ? "auto" : "smooth",
        });
    };

    if (titleLink != null) {
        titleLink.addEventListener("click", (event) => {
            event.preventDefault();

            if (pageHeading != null) {
                scrollToHeading(pageHeading);
                return;
            }

            const prefersReducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches;
            window.scrollTo({
                top: 0,
                behavior: prefersReducedMotion ? "auto" : "smooth",
            });
        });
    }

    // Click handler factory that keeps the TOC highlight in sync with user intent.
    const handleAnchorClick = (id) => (event) => {
        event.preventDefault();

        const targetHeading = getTargetHeading(id);
        if (targetHeading != null) {
            setActiveAnchor(id);
            lockActiveAnchor(id);
            scrollToHeading(targetHeading);
        }
    };

    const tocItems = headings.map((heading, index) => {
        const headingId = ensureHeadingId(heading, index);
        const normalizedLevel = normalizeLevel(heading.tagName);
        const anchor = createAnchor(heading.textContent, normalizedLevel, headingId);
        const item = createListItem();

        anchor.addEventListener("click", handleAnchorClick(headingId));
        item.appendChild(anchor);
        list.appendChild(item);

        return { id: headingId, heading, anchor };
    });

    anchorsById = new Map(tocItems.map(({ id, anchor }) => [id, anchor]));
    headingById = new Map(tocItems.map(({ id, heading }) => [id, heading]));

    main.appendChild(container);

    // Cache viewport and document measurements used during scroll updates.
    const computeDocumentMetrics = () => {
        const scrollingElement = document.scrollingElement ?? document.documentElement;
        const viewportHeight = window.innerHeight || document.documentElement.clientHeight;
        const scrollTop = window.scrollY || scrollingElement.scrollTop;
        let documentHeight = scrollingElement.scrollHeight;

        if (document.body != null) {
            documentHeight = Math.max(documentHeight, document.body.scrollHeight);
        }

        return {
            scrollBottom: scrollTop + viewportHeight,
            documentHeight,
        };
    };

    const updateActiveHeading = () => {
        if (lockedActiveId != null) {
            const lockedHeading = getTargetHeading(lockedActiveId);
            if (lockedHeading != null) {
                setActiveAnchor(lockedActiveId);
                return;
            }
            releaseActiveLock();
        }

        let currentHeading = tocItems[0]?.heading ?? null;

        for (const { heading } of tocItems) {
            const { top } = heading.getBoundingClientRect();
            if (top <= SCROLL_THRESHOLD_PX) {
                currentHeading = heading;
            } else {
                break;
            }
        }

        const { scrollBottom, documentHeight } = computeDocumentMetrics();
        if (documentHeight - scrollBottom <= PAGE_END_TOLERANCE_PX) {
            currentHeading = tocItems[tocItems.length - 1]?.heading ?? currentHeading;
        }

        if (currentHeading?.id != null) {
            setActiveAnchor(currentHeading.id);
        }
    };

    updateActiveHeading();

    let ticking = false;
    // Batch scroll events to animation frames to avoid redundant work.
    const handleScroll = () => {
        if (ticking) {
            return;
        }

        ticking = true;
        window.requestAnimationFrame(() => {
            updateActiveHeading();
            ticking = false;
        });
    };

    window.addEventListener("scroll", handleScroll, { passive: true });
    window.addEventListener("resize", handleScroll);

    const handleUserInteraction = () => {
        if (lockedActiveId != null) {
            releaseActiveLock();
        }
    };

    // Unlock the active highlight as soon as the user starts scrolling manually.
    window.addEventListener("wheel", handleUserInteraction, { passive: true });
    window.addEventListener("touchstart", handleUserInteraction, { passive: true });
    window.addEventListener("pointerdown", handleUserInteraction, { passive: true });
    window.addEventListener("keydown", (event) => {
        if (SCROLL_KEYS.has(event.key)) {
            handleUserInteraction();
        }
    });
};

export { buildTableOfContent };

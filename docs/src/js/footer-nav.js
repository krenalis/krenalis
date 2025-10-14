// Module footernav wires the bottom navigation on documentation pages to mirror
// sidebar order.

const SIDEBAR_NAV_SELECTOR = "body > article > .wrap > aside > nav";
const CONTENT_SELECTOR = "body > article > .wrap > .main > .content";
const FOOTER_NAV_CLASS = "footer";
const FOOTER_NAV_MODIFIERS = {
	balanced: `${FOOTER_NAV_CLASS}--balanced`,
	alignStart: `${FOOTER_NAV_CLASS}--align-start`,
	alignEnd: `${FOOTER_NAV_CLASS}--align-end`,
};
const INDEX_SUFFIX = /\/index$/;
const TRIMMED_ROOT = "/";

// normalizePath canonicalises a sidebar link or location path so comparisons
// are stable. It handles trailing slashes and index documents, matching
// Hugo-style routing.
const normalizePath = (rawPath) => {
	if (!rawPath) {
		return TRIMMED_ROOT;
	}

	let pathname;
	try {
		const { pathname: parsedPathname = TRIMMED_ROOT } = new URL(rawPath, window.location.origin);
		pathname = parsedPathname;
	} catch (_) {
		pathname = rawPath;
	}

	if (!pathname.startsWith("/")) {
		pathname = `/${pathname}`;
	}
	if (pathname !== TRIMMED_ROOT && pathname.endsWith("/")) {
		pathname = pathname.slice(0, -1);
	}
	if (pathname === "/index" || INDEX_SUFFIX.test(pathname)) {
		const trimmed = pathname.replace(INDEX_SUFFIX, "");
		return trimmed === "" ? TRIMMED_ROOT : trimmed;
	}

	return pathname || TRIMMED_ROOT;
};

// cleanLabel collapses whitespace from a label and trims it.
const cleanLabel = (value = "") => value.replace(/\s+/g, " ").trim();

// createPrevAnchor returns the DOM anchor that points to the previous document.
const createPrevAnchor = (link) => {
	const rawTitle = link.textContent || link.getAttribute("title") || "";
	const title = cleanLabel(rawTitle);
	const anchor = document.createElement("a");
	anchor.className = `${FOOTER_NAV_CLASS}__link ${FOOTER_NAV_CLASS}__link--prev`;
	anchor.href = link.href;
	anchor.rel = "prev";
	if (title) {
		anchor.setAttribute("aria-label", `Previous: ${title}`);
		anchor.title = `Previous: ${title}`;
	}

	const arrow = document.createElement("span");
	arrow.className = `${FOOTER_NAV_CLASS}__arrow`;
	arrow.textContent = "←";

	const label = document.createElement("span");
	label.className = `${FOOTER_NAV_CLASS}__label`;
	label.textContent = "Previous";

	anchor.append(arrow, label);
	return anchor;
};

// createNextAnchor returns the DOM anchor that points to the next document.
const createNextAnchor = (link) => {
	const rawTitle = link.textContent || link.getAttribute("title") || "";
	const trimmedTitle = cleanLabel(rawTitle);
	const anchor = document.createElement("a");
	anchor.className = `${FOOTER_NAV_CLASS}__link ${FOOTER_NAV_CLASS}__link--next`;
	anchor.href = link.href;
	anchor.rel = "next";
	if (trimmedTitle) {
		anchor.setAttribute("aria-label", `Next: ${trimmedTitle}`);
		anchor.title = `Next: ${trimmedTitle}`;
	}

	if (trimmedTitle) {
		const title = document.createElement("span");
		title.className = `${FOOTER_NAV_CLASS}__title`;
		title.textContent = trimmedTitle;
		anchor.appendChild(title);
	}

	const label = document.createElement("span");
	label.className = `${FOOTER_NAV_CLASS}__label`;
	label.textContent = "Next";

	const arrow = document.createElement("span");
	arrow.className = `${FOOTER_NAV_CLASS}__arrow`;
	arrow.textContent = "→";

	anchor.append(label, arrow);
	return anchor;
};

// resolveCurrentLink discovers the sidebar anchor that matches the current
// page. It prefers anchors already tagged with `.selected` and falls back to
// path matching.
const resolveCurrentLink = (links, sidebarNav) => {
	const selected = sidebarNav.querySelector("a.selected");
	if (selected) {
		return {
			current: selected,
			index: links.indexOf(selected),
		};
	}

	const currentPath = normalizePath(window.location.pathname);
	const index = links.findIndex((anchor) => normalizePath(anchor.pathname) === currentPath);

	return {
		current: index >= 0 ? links[index] : null,
		index,
	};
};

// collectSidebarLinks returns the sidebar anchors that contain a real
// destination.
const collectSidebarLinks = (sidebarNav) =>
	Array.from(sidebarNav.querySelectorAll("a[href]")).filter((anchor) => Boolean(anchor.href));

const applyAlignmentModifier = (footerNav, previous, next) => {
	if (previous && next) {
		footerNav.classList.add(FOOTER_NAV_MODIFIERS.balanced);
		return;
	}

	if (previous) {
		footerNav.classList.add(FOOTER_NAV_MODIFIERS.alignStart);
		return;
	}

	if (next) {
		footerNav.classList.add(FOOTER_NAV_MODIFIERS.alignEnd);
	}
};

// buildFooterNav assembles the footer navigation for the active documentation
// page. It mirrors the order from the sidebar to preserve the user’s reading
// flow.
const buildFooterNav = () => {
	const sidebarNav = document.querySelector(SIDEBAR_NAV_SELECTOR);
	const content = document.querySelector(CONTENT_SELECTOR);
	if (!sidebarNav || !content) {
		return;
	}

	const navLinks = collectSidebarLinks(sidebarNav);
	if (!navLinks.length) {
		return;
	}

	const { current, index } = resolveCurrentLink(navLinks, sidebarNav);
	if (!current || index === -1) {
		return;
	}

	const previous = navLinks[index - 1] || null;
	const next = navLinks[index + 1] || null;
	if (!previous && !next) {
		return;
	}

	const existing = content.querySelector(`.${FOOTER_NAV_CLASS}`);
	if (existing) {
		existing.remove();
	}

	const footerNav = document.createElement("nav");
	footerNav.className = FOOTER_NAV_CLASS;
	footerNav.setAttribute("aria-label", "Page navigation");

	applyAlignmentModifier(footerNav, previous, next);

	if (previous) {
		footerNav.appendChild(createPrevAnchor(previous));
	}
	if (next) {
		footerNav.appendChild(createNextAnchor(next));
	}

	content.appendChild(footerNav);
};

export { buildFooterNav };

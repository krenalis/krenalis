const COLLAPSED_CLASS = "sidebar-collapsed";
const EXPANDED_CLASS = "expanded";

const normalizePath = (path) => {
    if (!path) {
        return "/";
    }
    if (path.endsWith("/") && path !== "/") {
        return path.slice(0, -1);
    }
    return path;
};

const handleSidebar = () => {
    setTimeout(() => {
        scrollSidebar();
    }, 100);
    const button = document.querySelector(".sidebar-button");
    button.addEventListener("click", (e) => {
        const parent = e.currentTarget.parentElement;
        parent.classList.toggle("expanded");
    });
    const expandables = document.querySelectorAll(".sidebar-expandable");
    for (const e of expandables) {
        syncSelection(e);
    }
};

const scrollSidebar = () => {
    const sidebar = document.querySelector("body > article > .wrap > aside > nav");
    if (!sidebar) {
        return;
    }

    const selectedItems = sidebar.querySelectorAll("a.selected");
    let target = null;
    if (selectedItems.length === 1) {
        target = selectedItems[0];
    } else if (selectedItems.length > 1) {
        target = Array.from(selectedItems).find((item) => normalizePath(item.pathname) === normalizePath(window.location.pathname));
        if (!target) {
            target = selectedItems[selectedItems.length - 1];
        }
    }

    if (target) {
        target.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
};

const syncSelection = (li) => {
    const anchor = li.querySelector("a");
    if (!anchor) {
        return;
    }

    const childList = Array.from(li.children).find((child) => child.tagName === "UL");
    if (!childList) {
        console.error(`Error: <li.sidebar-expandable> must contain a <ul> child element.`, li);
        return;
    }

    const syncSelectedClass = () => {
        const anchorClasses = anchor.classList;
        const isActive = anchorClasses.contains("selected");
        const selectedDescendant = childList.querySelector("a.selected");
        const isDescendantSelected = selectedDescendant != null;
        if (!anchorClasses.contains("selected")) {
            li.classList.remove(COLLAPSED_CLASS);
        }

        const shouldExpand = (isActive || isDescendantSelected) && !li.classList.contains(COLLAPSED_CLASS);
        if (shouldExpand) {
            li.classList.add(EXPANDED_CLASS);
        } else {
            li.classList.remove(EXPANDED_CLASS);
        }
    };

    anchor.addEventListener("click", (event) => {
        const isCurrentPage = normalizePath(anchor.pathname) === normalizePath(window.location.pathname);
        if (!anchor.classList.contains("selected") || !isCurrentPage) {
            li.classList.remove(COLLAPSED_CLASS);
            return;
        }

        event.preventDefault();
        const isCollapsed = li.classList.toggle(COLLAPSED_CLASS);
        if (isCollapsed) {
            li.classList.remove(EXPANDED_CLASS);
        }
        syncSelectedClass();
    });

    const anchorObserver = new MutationObserver(syncSelectedClass);
    anchorObserver.observe(anchor, {
        attributes: true,
        attributeFilter: ["class"],
    });

    const childObserver = new MutationObserver(syncSelectedClass);
    childObserver.observe(childList, {
        subtree: true,
        attributes: true,
        attributeFilter: ["class"],
    });

    syncSelectedClass();
};

export { handleSidebar };

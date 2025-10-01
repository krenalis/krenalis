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
    const selectedItem = sidebar.querySelector("a.selected");
    if (selectedItem != null) {
        selectedItem.scrollIntoView({ behavior: "smooth", block: "nearest" });
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
        const isActive = anchorClasses.contains("selected") || anchorClasses.contains("expanded");
        const selectedDescendant = childList.querySelector("a.selected");
        const isDescendantSelected = selectedDescendant != null;
        if (isActive || isDescendantSelected) {
            li.classList.add("selected");
        } else {
            li.classList.remove("selected");
        }
    };

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

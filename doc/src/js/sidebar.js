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

    const siblingUL = li.nextElementSibling;
    if (!siblingUL || siblingUL.tagName !== "UL") {
        console.error(`Error: <li.sidebar-expandable> must have a sibling <ul> element.`, li);
        return;
    }

    const syncSelectedClass = () => {
        const isSelected = anchor.classList.contains("selected");
        const selectedSiblingAnchor = siblingUL.querySelector("a.selected");
        const isSiblingSelected = selectedSiblingAnchor != null;
        if (isSelected || isSiblingSelected) {
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

    const siblingObserver = new MutationObserver(syncSelectedClass);
    siblingObserver.observe(siblingUL, {
        subtree: true,
        attributes: true,
        attributeFilter: ["class"],
    });

    syncSelectedClass();
};

export { handleSidebar };

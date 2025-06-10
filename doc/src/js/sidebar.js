const handleSidebar = () => {
    setTimeout(() => {
        scrollSidebar();
    }, 100);
    const button = document.querySelector(".sidebar-button");
    button.addEventListener("click", (e) => {
        const parent = e.currentTarget.parentElement;
        parent.classList.toggle("expanded");
    });
};

const scrollSidebar = () => {
    const sidebar = document.querySelector("body > article > .wrap > aside > nav");
    const selectedItem = sidebar.querySelector("a.selected");
    if (selectedItem != null) {
        selectedItem.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
};

export { handleSidebar };

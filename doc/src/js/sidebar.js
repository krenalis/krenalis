const scrollSidebar = () => {
    const sidebar = document.querySelector("body > article > .wrap > aside > nav");
    const selectedItem = sidebar.querySelector("a.selected");
    selectedItem.scrollIntoView({ behavior: "smooth", block: "nearest" });
};

export { scrollSidebar };

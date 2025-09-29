const buildTableOfContent = () => {
    const headings = document.querySelectorAll("h2");
    if (headings.length === 0) {
        return;
    }

    let html = '<div class="table-of-content">';
    for (const title of headings) {
        html += `<a>${title.textContent}</a>`;
    }
    html += "</div>";

    const main = document.querySelector("body > article > .wrap > .main");
    main.insertAdjacentHTML("beforeend", html);

    const anchors = main.querySelectorAll(".table-of-content > a");
    for (const a of anchors) {
        a.addEventListener("click", (e) => {
            const text = e.currentTarget.textContent;
            const heading = Array.from(headings).find((h) => h.textContent === text);
            if (heading != null) {
                const offset = -50; // Prevent the heading from being covered by the fixed header.
                const top = heading.getBoundingClientRect().top + window.scrollY + offset;
                window.scrollTo({
                    top,
                    behavior: "smooth",
                });
            }
        });
    }
};

export { buildTableOfContent };

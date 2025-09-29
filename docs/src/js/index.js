import { handleSidebar } from "./sidebar.js";
import { buildTableOfContent } from "./table-of-content.js";

const setup = () => {
    handleSidebar();
    buildTableOfContent();
};

setup();

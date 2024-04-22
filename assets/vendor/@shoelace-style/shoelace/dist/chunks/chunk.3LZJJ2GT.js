import {
  menu_label_styles_default
} from "./chunk.DQM2UOFS.js";
import {
  ShoelaceElement
} from "./chunk.PZ6RRH2A.js";

// src/components/menu-label/menu-label.component.ts
import { html } from "lit";
var SlMenuLabel = class extends ShoelaceElement {
  render() {
    return html` <slot part="base" class="menu-label"></slot> `;
  }
};
SlMenuLabel.styles = menu_label_styles_default;

export {
  SlMenuLabel
};

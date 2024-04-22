import {
  carousel_item_styles_default
} from "./chunk.QPEGQM42.js";
import {
  ShoelaceElement
} from "./chunk.PZ6RRH2A.js";

// src/components/carousel-item/carousel-item.component.ts
import { html } from "lit";
var SlCarouselItem = class extends ShoelaceElement {
  static isCarouselItem(node) {
    return node instanceof Element && node.getAttribute("aria-roledescription") === "slide";
  }
  connectedCallback() {
    super.connectedCallback();
    this.setAttribute("role", "group");
  }
  render() {
    return html` <slot></slot> `;
  }
};
SlCarouselItem.styles = carousel_item_styles_default;

export {
  SlCarouselItem
};

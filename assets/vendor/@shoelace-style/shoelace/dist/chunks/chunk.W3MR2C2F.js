import {
  divider_styles_default
} from "./chunk.NCZWQBRI.js";
import {
  watch
} from "./chunk.C7FWPEOY.js";
import {
  ShoelaceElement
} from "./chunk.PZ6RRH2A.js";
import {
  __decorateClass
} from "./chunk.6M63UXML.js";

// src/components/divider/divider.component.ts
import { property } from "lit/decorators.js";
var SlDivider = class extends ShoelaceElement {
  constructor() {
    super(...arguments);
    this.vertical = false;
  }
  connectedCallback() {
    super.connectedCallback();
    this.setAttribute("role", "separator");
  }
  handleVerticalChange() {
    this.setAttribute("aria-orientation", this.vertical ? "vertical" : "horizontal");
  }
};
SlDivider.styles = divider_styles_default;
__decorateClass([
  property({ type: Boolean, reflect: true })
], SlDivider.prototype, "vertical", 2);
__decorateClass([
  watch("vertical")
], SlDivider.prototype, "handleVerticalChange", 1);

export {
  SlDivider
};

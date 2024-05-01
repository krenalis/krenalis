import {
  spinner_styles_default
} from "./chunk.CAHT2E74.js";
import {
  LocalizeController
} from "./chunk.2A3352EO.js";
import {
  ShoelaceElement
} from "./chunk.PZ6RRH2A.js";

// src/components/spinner/spinner.component.ts
import { html } from "lit";
var SlSpinner = class extends ShoelaceElement {
  constructor() {
    super(...arguments);
    this.localize = new LocalizeController(this);
  }
  render() {
    return html`
      <svg part="base" class="spinner" role="progressbar" aria-label=${this.localize.term("loading")}>
        <circle class="spinner__track"></circle>
        <circle class="spinner__indicator"></circle>
      </svg>
    `;
  }
};
SlSpinner.styles = spinner_styles_default;

export {
  SlSpinner
};

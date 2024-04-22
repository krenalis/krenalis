import {
  SlTab
} from "./chunk.XLJJVABO.js";

// src/react/tab/index.ts
import * as React from "react";
import { createComponent } from "@lit-labs/react";
import "@lit-labs/react";
var tagName = "sl-tab";
SlTab.define("sl-tab");
var reactWrapper = createComponent({
  tagName,
  elementClass: SlTab,
  react: React,
  events: {
    onSlClose: "sl-close"
  },
  displayName: "SlTab"
});
var tab_default = reactWrapper;

export {
  tab_default
};

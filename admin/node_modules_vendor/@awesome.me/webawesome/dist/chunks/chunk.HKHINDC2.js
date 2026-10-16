/*! Copyright 2026 Fonticons, Inc. - https://webawesome.com/license */

// src/internal/rendered-watcher.ts
import { isServer } from "lit";
var RenderedWatcher = class {
  constructor(element, callback) {
    this.element = element;
    this.callback = callback;
  }
  /**
   * Starts watching and reports the current state on the next frame. Additional targets also act as change signals,
   * e.g. an internal element that always has a non-zero box when rendered.
   */
  start(...additionalTargets) {
    if (isServer) {
      return;
    }
    this.observer ?? (this.observer = new ResizeObserver(() => this.check()));
    this.observer.observe(this.element);
    for (const target of additionalTargets) {
      this.observer.observe(target);
    }
    this.initialCheckHandle ?? (this.initialCheckHandle = requestAnimationFrame(() => {
      this.initialCheckHandle = void 0;
      this.check();
    }));
  }
  /** Stops watching. */
  stop() {
    if (this.initialCheckHandle !== void 0) {
      cancelAnimationFrame(this.initialCheckHandle);
      this.initialCheckHandle = void 0;
    }
    this.observer?.disconnect();
  }
  check() {
    this.callback(this.element.getClientRects().length > 0);
  }
};

export {
  RenderedWatcher
};

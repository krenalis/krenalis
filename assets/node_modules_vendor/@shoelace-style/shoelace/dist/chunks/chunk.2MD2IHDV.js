import {
  clamp
} from "./chunk.HF7GESMZ.js";
import {
  AutoplayController
} from "./chunk.F4VGSDIW.js";
import {
  carousel_styles_default
} from "./chunk.4DOW4ZG2.js";
import {
  ScrollController
} from "./chunk.VTZVPCX4.js";
import {
  SlCarouselItem
} from "./chunk.RVWTU54Z.js";
import {
  prefersReducedMotion
} from "./chunk.W5IOVX7A.js";
import {
  LocalizeController
} from "./chunk.2A3352EO.js";
import {
  SlIcon
} from "./chunk.NDWJOB5Q.js";
import {
  watch
} from "./chunk.C7FWPEOY.js";
import {
  ShoelaceElement
} from "./chunk.PZ6RRH2A.js";
import {
  __decorateClass
} from "./chunk.6M63UXML.js";

// src/components/carousel/carousel.component.ts
import { classMap } from "lit/directives/class-map.js";
import { html } from "lit";
import { map } from "lit/directives/map.js";
import { property, query, state } from "lit/decorators.js";
import { range } from "lit/directives/range.js";
var SlCarousel = class extends ShoelaceElement {
  constructor() {
    super(...arguments);
    this.loop = false;
    this.navigation = false;
    this.pagination = false;
    this.autoplay = false;
    this.autoplayInterval = 3e3;
    this.slidesPerPage = 1;
    this.slidesPerMove = 1;
    this.orientation = "horizontal";
    this.mouseDragging = false;
    this.activeSlide = 0;
    this.autoplayController = new AutoplayController(this, () => this.next());
    this.scrollController = new ScrollController(this);
    this.slides = this.getElementsByTagName("sl-carousel-item");
    // determines which slide is displayed
    // A map containing the state of all the slides
    this.intersectionObserverEntries = /* @__PURE__ */ new Map();
    this.localize = new LocalizeController(this);
    this.handleSlotChange = (mutations) => {
      const needsInitialization = mutations.some(
        (mutation) => [...mutation.addedNodes, ...mutation.removedNodes].some(
          (node) => SlCarouselItem.isCarouselItem(node) && !node.hasAttribute("data-clone")
        )
      );
      if (needsInitialization) {
        this.initializeSlides();
      }
      this.requestUpdate();
    };
  }
  connectedCallback() {
    super.connectedCallback();
    this.setAttribute("role", "region");
    this.setAttribute("aria-label", this.localize.term("carousel"));
    const intersectionObserver = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          this.intersectionObserverEntries.set(entry.target, entry);
          const slide = entry.target;
          slide.toggleAttribute("inert", !entry.isIntersecting);
          slide.classList.toggle("--in-view", entry.isIntersecting);
          slide.setAttribute("aria-hidden", entry.isIntersecting ? "false" : "true");
        });
      },
      {
        root: this,
        threshold: 0.6
      }
    );
    this.intersectionObserver = intersectionObserver;
    intersectionObserver.takeRecords().forEach((entry) => {
      this.intersectionObserverEntries.set(entry.target, entry);
    });
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.intersectionObserver.disconnect();
    this.mutationObserver.disconnect();
  }
  firstUpdated() {
    this.initializeSlides();
    this.mutationObserver = new MutationObserver(this.handleSlotChange);
    this.mutationObserver.observe(this, { childList: true, subtree: false });
  }
  getPageCount() {
    return Math.ceil(this.getSlides().length / this.slidesPerPage);
  }
  getCurrentPage() {
    return Math.ceil(this.activeSlide / this.slidesPerPage);
  }
  getSlides({ excludeClones = true } = {}) {
    return [...this.slides].filter((slide) => !excludeClones || !slide.hasAttribute("data-clone"));
  }
  handleKeyDown(event) {
    if (["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End"].includes(event.key)) {
      const target = event.target;
      const isRtl = this.localize.dir() === "rtl";
      const isFocusInPagination = target.closest('[part~="pagination-item"]') !== null;
      const isNext = event.key === "ArrowDown" || !isRtl && event.key === "ArrowRight" || isRtl && event.key === "ArrowLeft";
      const isPrevious = event.key === "ArrowUp" || !isRtl && event.key === "ArrowLeft" || isRtl && event.key === "ArrowRight";
      event.preventDefault();
      if (isPrevious) {
        this.previous();
      }
      if (isNext) {
        this.next();
      }
      if (event.key === "Home") {
        this.goToSlide(0);
      }
      if (event.key === "End") {
        this.goToSlide(this.getSlides().length - 1);
      }
      if (isFocusInPagination) {
        this.updateComplete.then(() => {
          var _a;
          const activePaginationItem = (_a = this.shadowRoot) == null ? void 0 : _a.querySelector(
            '[part~="pagination-item--active"]'
          );
          if (activePaginationItem) {
            activePaginationItem.focus();
          }
        });
      }
    }
  }
  handleScrollEnd() {
    const slides = this.getSlides();
    const entries = [...this.intersectionObserverEntries.values()];
    const firstIntersecting = entries.find((entry) => entry.isIntersecting);
    if (this.loop && (firstIntersecting == null ? void 0 : firstIntersecting.target.hasAttribute("data-clone"))) {
      const clonePosition = Number(firstIntersecting.target.getAttribute("data-clone"));
      this.goToSlide(clonePosition, "auto");
      return;
    }
    if (firstIntersecting) {
      this.activeSlide = slides.indexOf(firstIntersecting.target);
    }
  }
  initializeSlides() {
    const slides = this.getSlides();
    const intersectionObserver = this.intersectionObserver;
    this.intersectionObserverEntries.clear();
    this.getSlides({ excludeClones: false }).forEach((slide, index) => {
      intersectionObserver.unobserve(slide);
      slide.classList.remove("--in-view");
      slide.classList.remove("--is-active");
      slide.setAttribute("aria-label", this.localize.term("slideNum", index + 1));
      if (slide.hasAttribute("data-clone")) {
        slide.remove();
      }
    });
    if (this.loop) {
      const slidesPerPage = this.slidesPerPage;
      const lastSlides = slides.slice(-slidesPerPage);
      const firstSlides = slides.slice(0, slidesPerPage);
      lastSlides.reverse().forEach((slide, i) => {
        const clone = slide.cloneNode(true);
        clone.setAttribute("data-clone", String(slides.length - i - 1));
        this.prepend(clone);
      });
      firstSlides.forEach((slide, i) => {
        const clone = slide.cloneNode(true);
        clone.setAttribute("data-clone", String(i));
        this.append(clone);
      });
    }
    this.getSlides({ excludeClones: false }).forEach((slide) => {
      intersectionObserver.observe(slide);
    });
    this.goToSlide(this.activeSlide, "auto");
  }
  handelSlideChange() {
    const slides = this.getSlides();
    slides.forEach((slide, i) => {
      slide.classList.toggle("--is-active", i === this.activeSlide);
    });
    if (this.hasUpdated) {
      this.emit("sl-slide-change", {
        detail: {
          index: this.activeSlide,
          slide: slides[this.activeSlide]
        }
      });
    }
  }
  handleSlidesPerMoveChange() {
    const slides = this.getSlides({ excludeClones: false });
    const slidesPerMove = this.slidesPerMove;
    slides.forEach((slide, i) => {
      const shouldSnap = Math.abs(i - slidesPerMove) % slidesPerMove === 0;
      if (shouldSnap) {
        slide.style.removeProperty("scroll-snap-align");
      } else {
        slide.style.setProperty("scroll-snap-align", "none");
      }
    });
  }
  handleAutoplayChange() {
    this.autoplayController.stop();
    if (this.autoplay) {
      this.autoplayController.start(this.autoplayInterval);
    }
  }
  handleMouseDraggingChange() {
    this.scrollController.mouseDragging = this.mouseDragging;
  }
  /**
   * Move the carousel backward by `slides-per-move` slides.
   *
   * @param behavior - The behavior used for scrolling.
   */
  previous(behavior = "smooth") {
    let previousIndex = this.activeSlide || this.activeSlide - this.slidesPerMove;
    let canSnap = false;
    while (!canSnap && previousIndex > 0) {
      previousIndex -= 1;
      canSnap = Math.abs(previousIndex - this.slidesPerMove) % this.slidesPerMove === 0;
    }
    this.goToSlide(previousIndex, behavior);
  }
  /**
   * Move the carousel forward by `slides-per-move` slides.
   *
   * @param behavior - The behavior used for scrolling.
   */
  next(behavior = "smooth") {
    this.goToSlide(this.activeSlide + this.slidesPerMove, behavior);
  }
  /**
   * Scrolls the carousel to the slide specified by `index`.
   *
   * @param index - The slide index.
   * @param behavior - The behavior used for scrolling.
   */
  goToSlide(index, behavior = "smooth") {
    const { slidesPerPage, loop, scrollContainer } = this;
    const slides = this.getSlides();
    const slidesWithClones = this.getSlides({ excludeClones: false });
    const newActiveSlide = (index + slides.length) % slides.length;
    this.activeSlide = newActiveSlide;
    const nextSlideIndex = clamp(index + (loop ? slidesPerPage : 0), 0, slidesWithClones.length - 1);
    const nextSlide = slidesWithClones[nextSlideIndex];
    const scrollContainerRect = scrollContainer.getBoundingClientRect();
    const nextSlideRect = nextSlide.getBoundingClientRect();
    scrollContainer.scrollTo({
      left: nextSlideRect.left - scrollContainerRect.left + scrollContainer.scrollLeft,
      top: nextSlideRect.top - scrollContainerRect.top + scrollContainer.scrollTop,
      behavior: prefersReducedMotion() ? "auto" : behavior
    });
  }
  render() {
    const { scrollController, slidesPerPage } = this;
    const pagesCount = this.getPageCount();
    const currentPage = this.getCurrentPage();
    const prevEnabled = this.loop || currentPage > 0;
    const nextEnabled = this.loop || currentPage < pagesCount - 1;
    const isLtr = this.localize.dir() === "ltr";
    return html`
      <div part="base" class="carousel">
        <div
          id="scroll-container"
          part="scroll-container"
          class="${classMap({
      carousel__slides: true,
      "carousel__slides--horizontal": this.orientation === "horizontal",
      "carousel__slides--vertical": this.orientation === "vertical"
    })}"
          style="--slides-per-page: ${this.slidesPerPage};"
          aria-busy="${scrollController.scrolling ? "true" : "false"}"
          aria-atomic="true"
          tabindex="0"
          @keydown=${this.handleKeyDown}
          @scrollend=${this.handleScrollEnd}
        >
          <slot></slot>
        </div>

        ${this.navigation ? html`
              <div part="navigation" class="carousel__navigation">
                <button
                  part="navigation-button navigation-button--previous"
                  class="${classMap({
      "carousel__navigation-button": true,
      "carousel__navigation-button--previous": true,
      "carousel__navigation-button--disabled": !prevEnabled
    })}"
                  aria-label="${this.localize.term("previousSlide")}"
                  aria-controls="scroll-container"
                  aria-disabled="${prevEnabled ? "false" : "true"}"
                  @click=${prevEnabled ? () => this.previous() : null}
                >
                  <slot name="previous-icon">
                    <sl-icon library="system" name="${isLtr ? "chevron-left" : "chevron-right"}"></sl-icon>
                  </slot>
                </button>

                <button
                  part="navigation-button navigation-button--next"
                  class=${classMap({
      "carousel__navigation-button": true,
      "carousel__navigation-button--next": true,
      "carousel__navigation-button--disabled": !nextEnabled
    })}
                  aria-label="${this.localize.term("nextSlide")}"
                  aria-controls="scroll-container"
                  aria-disabled="${nextEnabled ? "false" : "true"}"
                  @click=${nextEnabled ? () => this.next() : null}
                >
                  <slot name="next-icon">
                    <sl-icon library="system" name="${isLtr ? "chevron-right" : "chevron-left"}"></sl-icon>
                  </slot>
                </button>
              </div>
            ` : ""}
        ${this.pagination ? html`
              <div part="pagination" role="tablist" class="carousel__pagination" aria-controls="scroll-container">
                ${map(range(pagesCount), (index) => {
      const isActive = index === currentPage;
      return html`
                    <button
                      part="pagination-item ${isActive ? "pagination-item--active" : ""}"
                      class="${classMap({
        "carousel__pagination-item": true,
        "carousel__pagination-item--active": isActive
      })}"
                      role="tab"
                      aria-selected="${isActive ? "true" : "false"}"
                      aria-label="${this.localize.term("goToSlide", index + 1, pagesCount)}"
                      tabindex=${isActive ? "0" : "-1"}
                      @click=${() => this.goToSlide(index * slidesPerPage)}
                      @keydown=${this.handleKeyDown}
                    ></button>
                  `;
    })}
              </div>
            ` : ""}
      </div>
    `;
  }
};
SlCarousel.styles = carousel_styles_default;
SlCarousel.dependencies = { "sl-icon": SlIcon };
__decorateClass([
  property({ type: Boolean, reflect: true })
], SlCarousel.prototype, "loop", 2);
__decorateClass([
  property({ type: Boolean, reflect: true })
], SlCarousel.prototype, "navigation", 2);
__decorateClass([
  property({ type: Boolean, reflect: true })
], SlCarousel.prototype, "pagination", 2);
__decorateClass([
  property({ type: Boolean, reflect: true })
], SlCarousel.prototype, "autoplay", 2);
__decorateClass([
  property({ type: Number, attribute: "autoplay-interval" })
], SlCarousel.prototype, "autoplayInterval", 2);
__decorateClass([
  property({ type: Number, attribute: "slides-per-page" })
], SlCarousel.prototype, "slidesPerPage", 2);
__decorateClass([
  property({ type: Number, attribute: "slides-per-move" })
], SlCarousel.prototype, "slidesPerMove", 2);
__decorateClass([
  property()
], SlCarousel.prototype, "orientation", 2);
__decorateClass([
  property({ type: Boolean, reflect: true, attribute: "mouse-dragging" })
], SlCarousel.prototype, "mouseDragging", 2);
__decorateClass([
  query("slot:not([name])")
], SlCarousel.prototype, "defaultSlot", 2);
__decorateClass([
  query(".carousel__slides")
], SlCarousel.prototype, "scrollContainer", 2);
__decorateClass([
  query(".carousel__pagination")
], SlCarousel.prototype, "paginationContainer", 2);
__decorateClass([
  state()
], SlCarousel.prototype, "activeSlide", 2);
__decorateClass([
  watch("loop", { waitUntilFirstUpdate: true }),
  watch("slidesPerPage", { waitUntilFirstUpdate: true })
], SlCarousel.prototype, "initializeSlides", 1);
__decorateClass([
  watch("activeSlide")
], SlCarousel.prototype, "handelSlideChange", 1);
__decorateClass([
  watch("slidesPerMove")
], SlCarousel.prototype, "handleSlidesPerMoveChange", 1);
__decorateClass([
  watch("autoplay")
], SlCarousel.prototype, "handleAutoplayChange", 1);
__decorateClass([
  watch("mouseDragging")
], SlCarousel.prototype, "handleMouseDraggingChange", 1);

export {
  SlCarousel
};

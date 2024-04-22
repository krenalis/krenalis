/**
 * @license
 * Copyright 2018 Google LLC
 * SPDX-License-Identifier: BSD-3-Clause
 */
const e=new Set(["children","localName","ref","style","className"]),n=new WeakMap,l=(e,l,t,o,a)=>{const i=null==a?void 0:a[l];void 0===i||t===o?(e[l]=t,null==t&&l in HTMLElement.prototype&&e.removeAttribute(l)):((e,l,t)=>{let o=n.get(e);void 0===o&&n.set(e,o=new Map);let a=o.get(l);void 0!==t?void 0===a?(o.set(l,a={handleEvent:t}),e.addEventListener(l,a)):a.handleEvent=t:void 0!==a&&(o.delete(l),e.removeEventListener(l,a))})(e,i,t)},t=({react:n,tagName:t,elementClass:o,events:a,displayName:i})=>{const s=new Set(Object.keys(null!=a?a:{})),c=n.forwardRef(((i,c)=>{const r=n.useRef(null),d=n.useRef(null),u={},v={};for(const[n,l]of Object.entries(i))e.has(n)?u["className"===n?"class":n]=l:s.has(n)||n in o.prototype?v[n]=l:u[n]=l;return n.useLayoutEffect((()=>{if(null!==d.current){for(const e in v)l(d.current,e,i[e],r.current?r.current[e]:void 0,a);r.current=i}})),n.useLayoutEffect((()=>{var e;null===(e=d.current)||void 0===e||e.removeAttribute("defer-hydration")}),[]),u.suppressHydrationWarning=!0,n.createElement(t,{...u,ref:e=>{d.current=e,"function"==typeof c?c(e):null!==c&&(c.current=e)}})}));return c.displayName=null!=i?i:o.name,c};export{t as createComponent};
//# sourceMappingURL=create-component.js.map

import { css, html, LitElement } from 'https://cdn.jsdelivr.net/gh/lit/dist@2.2.5/all/lit-all.min.js';
import {} from 'https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.0.0-beta.74/dist/shoelace.js';

export class TableHead extends LitElement {
	static properties = {
		column: { type: Object },
	};

	static styles = css`
		.head-cell {
			text-align: center;
			padding: 17px 3px;
			background-color: rgb(250, 250, 250);
			display: flex;
			justify-content: center;
			column-gap: 5px;
			align-items: center;
			font-weight: 500;
			border-bottom: 1px solid #cbd5e1;
		}
	`;

	constructor() {
		super();
		this.column = {};
	}

	render() {
		return html`<div class="head-cell" data-column="${this.column.code}">${this.column.title}</div>`;
	}
}

customElements.define('o2b-table-head', TableHead);

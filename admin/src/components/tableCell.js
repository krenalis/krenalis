import { css, html, LitElement } from 'https://cdn.jsdelivr.net/gh/lit/dist@2.2.5/all/lit-all.min.js';
import {} from 'https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.0.0-beta.74/dist/shoelace.js';

export class TableCell extends LitElement {
	static properties = {
		cell: {},
	};

	static styles = css`
		.cell {
			padding: 20px;
			opacity: 0.8;
			text-align: center;
			overflow-wrap: anywhere;
		}
	`;

	constructor() {
		super();
		this.cell = '';
	}

	render() {
		let cellContent;
		switch (typeof this.cell) {
			case 'string':
				cellContent = this.cell === '' ? '-' : this.cell;
				break;
			case 'number':
				cellContent = String(this.cell);
				break;
			case 'boolean':
				cellContent = this.cell
					? html`<sl-icon style="color: var(--sl-color-primary-600)" name="check-circle"></sl-icon>`
					: html`<sl-icon style="color: var(--sl-color-danger-600)" name="x-circle"></sl-icon>`;
				break;
			default:
				console.error(`type ${typeof this.cell} is not supported by the o2b-table`);
		}
		return html` <div class="cell">${cellContent}</div> `;
	}
}

customElements.define('o2b-table-cell', TableCell);

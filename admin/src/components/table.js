import { css, html, LitElement } from 'https://cdn.jsdelivr.net/gh/lit/dist@2.2.5/all/lit-all.min.js';
import {} from 'https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.0.0-beta.74/dist/shoelace.js';
import './tableHead.js';
import './tableCell.js';

export class Table extends LitElement {
	static properties = {
		title: { type: String },
		description: { type: String },
		query: { type: String },
		columns: { type: Array },
		_rows: { state: true },
	};

	static styles = css`
		* {
			box-sizing: border-box;
		}

		.head {
			background-color: transparent;
			padding: 30px;
			text-align: left;
			color: black;
		}

		.head .title {
			font-size: 22px;
			font-weight: 500;
			opacity: 0.9;
		}

		.head .description {
			opacity: 0.8;
		}

		.body {
			background-color: white;
			overflow-x: hidden;
			border: 1px solid rgb(203, 213, 225);
			border-radius: 5px;
			box-shadow: 0px 1px 5px 1px rgb(230 230 230);
		}

		.body::-webkit-scrollbar {
			background-color: transparent;
		}

		.body::-webkit-scrollbar-track {
			background-color: transparent;
		}

		.body::-webkit-scrollbar-thumb {
			background-color: rgb(235, 235, 235);
			border: 1px solid rgb(203, 213, 225);
			border-bottom: 0;
		}

		.body .head-row {
			position: sticky;
			top: 0;
			z-index: 10;
		}

		.body .head-row,
		.body .row {
			display: grid;
		}

		.body .no-data {
			padding: 20px;
			border-top: 0;
		}

		.body .row o2b-table-cell {
			border-top: 1px solid #cbd5e1;
			border-right: 1px solid #cbd5e1;
			display: flex;
			justify-content: center;
			align-items: center;
		}

		.body .row o2b-table-cell:last-child {
			border-right: 0;
		}

		.body .row:first-child o2b-table-cell {
			border-top: 0;
		}

		.body .row o2b-table-cell:first-child {
			font-weight: 500;
		}

		/* spinner */

		.spinner {
			padding: 50px;
			text-align: center;
		}

		.lds-grid {
			display: inline-block;
			position: relative;
			width: 80px;
			height: 80px;
		}

		.lds-grid div {
			position: absolute;
			width: 16px;
			height: 16px;
			border-radius: 50%;
			background: #4f46e5;
			animation: lds-grid 1.2s linear infinite;
		}

		.lds-grid div:nth-child(1) {
			top: 8px;
			left: 8px;
			animation-delay: 0s;
		}

		.lds-grid div:nth-child(2) {
			top: 8px;
			left: 32px;
			animation-delay: -0.4s;
		}

		.lds-grid div:nth-child(3) {
			top: 8px;
			left: 56px;
			animation-delay: -0.8s;
		}

		.lds-grid div:nth-child(4) {
			top: 32px;
			left: 8px;
			animation-delay: -0.4s;
		}

		.lds-grid div:nth-child(5) {
			top: 32px;
			left: 32px;
			animation-delay: -0.8s;
		}

		.lds-grid div:nth-child(6) {
			top: 32px;
			left: 56px;
			animation-delay: -1.2s;
		}

		.lds-grid div:nth-child(7) {
			top: 56px;
			left: 8px;
			animation-delay: -0.8s;
		}

		.lds-grid div:nth-child(8) {
			top: 56px;
			left: 32px;
			animation-delay: -1.2s;
		}

		.lds-grid div:nth-child(9) {
			top: 56px;
			left: 56px;
			animation-delay: -1.6s;
		}

		@keyframes lds-grid {
			0%,
			100% {
				opacity: 1;
			}
			50% {
				opacity: 0.5;
			}
		}
	`;

	constructor() {
		super();
		this.title = '';
		this.description = '';
		this.columns = [];
	}

	async firstUpdated() {
		if (!this.query) return;
		try {
			let res = await fetch('https://localhost:2020/chichi.cgi/run-query', {
				method: 'POST',
				body: JSON.stringify(this.query),
			});
			this._rows = await res.json();
			setTimeout(() => {
				this.renderRoot.querySelector('.spinner').style.display = 'none';
				this.renderRoot.querySelector('.body').style.overflowX = 'scroll';
				this.renderRoot.querySelector('.content').style.display = 'block';
			}, 1000);
		} catch (err) {
			console.error(`error while fetching data for o2b-table "${this.title}": ${err}`);
		}
	}

	render() {
		return html`
			<div class="table">
				<div class="head">
					<div class="title">${this.title}</div>
					<div class="description">${this.description}</div>
				</div>
				<div class="body">
					<div
						class="head-row"
						style="grid-template-columns: ${this.columns.map((column) => column.width + 'px').join(' ')}"
					>
						${this.columns.map((column) => html`<o2b-table-head .column=${column}></o2b-table-head>`)}
					</div>

					<div class="spinner">
						<div class="lds-grid">
							<div></div>
							<div></div>
							<div></div>
							<div></div>
							<div></div>
							<div></div>
							<div></div>
							<div></div>
							<div></div>
						</div>
					</div>

					<div class="content" style="display: none;">
						${this._rows
							? this._rows.map(
									(row) =>
										html`<div
											class="row"
											style="grid-template-columns: ${this.columns.map((column) => column.width + 'px').join(' ')}"
										>
											${row.map((cell) => html`<o2b-table-cell .cell=${cell}></o2b-table-cell>`)}
										</div>`
							  )
							: html`<div class="no-data">No data available</div>`}
					</div>
				</div>
			</div>
		`;
	}
}

customElements.define('o2b-table', Table);

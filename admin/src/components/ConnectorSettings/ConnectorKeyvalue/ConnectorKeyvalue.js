import React from 'react';
import './ConnectorKeyvalue.css';
import { renderConnectorComponent } from '../renderConnectorComponent';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

export default class ConnectorKeyValue extends React.Component {
	constructor(props) {
		super(props);
		let rows = [];
		let keyValues = [];
		if (this.props.value) {
			keyValues = Object.entries(this.props.value);
		}
		if (keyValues.length > 0) {
			let counter = 1;
			for (const [key, value] of keyValues) {
				rows.push({ id: counter, key: key, value: value });
				counter = counter + 1;
			}
		} else {
			rows.push({ id: 1, key: '', value: '' });
		}
		this.state = { rows: rows };
	}

	formatRows = (rows) => {
		let formatted = {};
		for (let r of rows) {
			formatted[r.key] = r.value;
		}
		return formatted;
	};

	onAddRowClick = (e) => {
		let rows = this.state.rows;
		rows.push({ id: this.state.rows[this.state.rows.length - 1].id + 1, key: '', value: '' });
		this.props.onChange(this.props.name, this.formatRows(rows), e);
		this.setState({ rows: rows });
	};

	onRemoveRowClick = (id, e) => {
		let rows = this.state.rows;
		let filtered = rows.filter((row) => row.id !== id);
		this.props.onChange(this.props.name, this.formatRows(filtered), e);
		this.setState({ rows: filtered });
	};

	onKeyChange = async (name, key, e) => {
		let id = Number(e.currentTarget.closest('.row').dataset.id);
		let updated = this.state.rows.map((row) => {
			if (row.id === id) return { ...row, key: key };
			return row;
		});
		this.props.onChange(this.props.name, this.formatRows(updated), e);
		this.setState({ rows: updated });
	};

	onValueChange = (name, value, e) => {
		let id = Number(e.currentTarget.closest('.row').dataset.id);
		let updated = this.state.rows.map((row) => {
			if (row.id === id) return { ...row, value: value };
			return row;
		});
		this.props.onChange(this.props.name, this.formatRows(updated), e);
		this.setState({ rows: updated });
	};

	render() {
		return (
			<div className='ConnectorKeyValue'>
				<div className='label'>{this.props.label}</div>
				<div className='grid'>
					<div className='row labels'>
						<div className='keyLabel'>{this.props.keyLabel}</div>
						<div className='valueLabel'>{this.props.valueLabel}</div>
					</div>
					{this.state.rows.map((row) => {
						return (
							<div className='row' data-id={row.id} key={row.id}>
								<div className='key cell'>
									{renderConnectorComponent(this.props.keyComponent, this.onKeyChange, row.key)}
								</div>
								<div className='value cell'>
									{renderConnectorComponent(this.props.valueComponent, this.onValueChange, row.value)}
								</div>
								{row.id !== 1 && (
									<SlIcon
										className='removeRow'
										name='trash3'
										onClick={() => this.onRemoveRowClick(row.id)}
									/>
								)}
							</div>
						);
					})}
				</div>
				<SlIcon className='addRow' onClick={this.onAddRowClick} name='plus-circle' />
			</div>
		);
	}
}

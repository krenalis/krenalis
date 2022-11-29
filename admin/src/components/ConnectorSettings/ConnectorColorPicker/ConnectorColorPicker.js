import React from 'react';
import './ConnectorColorPicker.css';
import { SlColorPicker } from '@shoelace-style/shoelace/dist/react/index.js';

export default class ConnectorColorPicker extends React.Component {
	state = { value: this.props.value };

	onColorPickerChange = (e) => {
		this.setState({ value: e.currentTarget.value });
		this.props.onChange(this.props.name, e.currentTarget.value, e);
	};

	render() {
		return (
			<div className='ConnectorColorPicker'>
				<SlColorPicker
					value={this.state.value}
					name={this.props.name}
					label={this.props.label}
					onSlChange={this.onColorPickerChange}
				/>
				<div className='label'>{this.props.label}</div>
			</div>
		);
	}
}

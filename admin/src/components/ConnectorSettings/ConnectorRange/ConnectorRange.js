import React from 'react';
import { SlRange } from '@shoelace-style/shoelace/dist/react/index.js';

export default class ConnectorRange extends React.Component {
	state = { value: this.props.value };

	onRangeChange = (e) => {
		this.setState({ value: e.currentTarget.value });
		this.props.onChange(this.props.name, e.currentTarget.value, e);
	};

	render() {
		return (
			<div className='ConnectorRange'>
				<SlRange
					name={this.props.name}
					value={this.state.value}
					label={this.props.label}
					min={this.props.min}
					max={this.props.max}
					step={this.props.step}
					onSlChange={this.onRangeChange}
				/>
			</div>
		);
	}
}

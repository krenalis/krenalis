import React from 'react';
import './ConnectorInput';
import { SlInput } from '@shoelace-style/shoelace/dist/react/index.js';

export default class ConnectorInput extends React.Component {
	state = { value: this.props.value };

	onInputChange = (e) => {
		let value;
		if (this.props.type === 'number') value = Number(e.currentTarget.value);
		else value = e.currentTarget.value;
		this.setState({ value: value });
		this.props.onChange(this.props.name, value, e);
	};

	render() {
		return (
			<div className='ConnectorInput'>
				<SlInput
					name={this.props.name}
					value={this.state.value}
					label={this.props.label}
					placeholder={this.props.placeholder}
					type={this.props.type === '' ? 'text' : this.props.type}
					minlength={this.props.minlength !== 0 && this.props.minlength}
					maxlength={this.props.maxlength !== 0 && this.props.maxlength}
					passwordToggle={this.props.type === 'password'}
					onSlChange={this.onInputChange}
				/>
			</div>
		);
	}
}

import React from 'react';
import './ConnectorSwitch.css';
import { SlSwitch } from '@shoelace-style/shoelace/dist/react';

export default class ConnectorSwitch extends React.Component {
	state = { value: this.props.value };

	onSwitchChange = (e) => {
		this.setState({ value: !this.state.value });
		this.props.onChange(this.props.name, e.currentTarget.checked, e);
	};

	render() {
		return (
			<div className='ConnectorSwitch'>
				<SlSwitch name={this.props.name} onSlChange={this.onSwitchChange} checked={this.state.value}>
					{this.props.label}
				</SlSwitch>
			</div>
		);
	}
}

import React from 'react';
import './ConnectorCheckbox.css';
import { SlCheckbox } from '@shoelace-style/shoelace/dist/react';

export default class ConnectorCheckbox extends React.Component {
	
	state = {value: this.props.value}

	onCheckboxChange = (e) => {
		this.setState({value: !this.state.value});
		this.props.onChange(this.props.name, e.currentTarget.checked, e);
	}
	
	render() {
		return (
			<div className="ConnectorCheckbox">
				<SlCheckbox name={this.props.name} onSlChange={this.onCheckboxChange} checked={this.state.value}>{this.props.label}</SlCheckbox>
			</div>
		)
	}
}

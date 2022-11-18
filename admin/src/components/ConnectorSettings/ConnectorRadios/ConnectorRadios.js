import React from 'react';
import './ConnectorRadios.css';
import { SlRadio, SlRadioGroup } from '@shoelace-style/shoelace/dist/react';

export default class ConnectorRadios extends React.Component {
	state = { value: this.props.value };

	onRadioGroupChange = (e) => {
		this.setState({ value: e.currentTarget.value });
		this.props.onChange(this.props.name, e.currentTarget.value, e);
	};

	render() {
		return (
			<div className='ConnectorRadios'>
				<SlRadioGroup
					value={this.state.value}
					label={this.props.label}
					name={this.props.name}
					onSlChange={this.onRadioGroupChange}
					fieldset
				>
					{this.props.options.map((opt, i) => {
						return <SlRadio value={opt.Value}>{opt.Text}</SlRadio>;
					})}
				</SlRadioGroup>
			</div>
		);
	}
}

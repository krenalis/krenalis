import React from 'react';
import './ConnectorTextarea';
import { SlTextarea } from '@shoelace-style/shoelace/dist/react';

export default class ConnectorTextarea extends React.Component {
	state = { value: this.props.value };

	onTextAreaChange = (e) => {
		this.setState({ value: e.currentTarget.value });
		this.props.onChange(this.props.name, e.currentTarget.value, e);
	};

	render() {
		return (
			<div className='ConnectorTextarea'>
				<SlTextarea
					name={this.props.name}
					value={this.state.value}
					label={this.props.label}
					placeholder={this.props.placeholder}
					rows={this.props.rows}
					minlength={this.props.minlength !== 0 && this.props.minlength}
					maxlength={this.props.maxlength !== 0 && this.props.maxlength}
					onSlChange={this.onTextAreaChange}
				/>
			</div>
		);
	}
}

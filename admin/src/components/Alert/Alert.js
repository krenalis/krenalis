import React from 'react';
import './Alert.css';
import { SlAlert, SlIcon } from '@shoelace-style/shoelace/dist/react';

export default class Alert extends React.Component {
	render() {
		return (
			<SlAlert variant={this.props.status.variant} open>
				<SlIcon slot='icon' name={this.props.status.icon} />
				{this.props.status.text}
			</SlAlert>
		);
	}
}

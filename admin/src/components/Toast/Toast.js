import React from 'react';
import './Toast.css';
import { SlAlert, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

export default class Alert extends React.Component {
	render() {
		return this.props.status == null ? (
			<SlAlert ref={this.props.reactRef} variant='error' closable>
				<SlIcon slot='icon' name='exclamation-octagon' />
				<b>Something went wrong</b>
			</SlAlert>
		) : (
			<SlAlert ref={this.props.reactRef} variant={this.props.status.variant} closable>
				<SlIcon slot='icon' name={this.props.status.icon} />
				<b>{this.props.status.text}</b>
			</SlAlert>
		);
	}
}

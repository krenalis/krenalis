import React, { Component } from 'react'
import { Link } from 'react-router-dom'

import './ConnectorsConfirmation.css'
import '../../../components/Button/Button'

export default class ConnectorsConfirmation extends Component {

	constructor(props) {
		super(props);
		this.state = {
			connector: {}
		}
	}

	async componentDidMount() {
		let connectorID = String(window.location).split('/').pop();
		let connector
		try {
			let res = await fetch('/admin/connectors/get', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(Number(connectorID)),
			});
			connector = await res.json();
		} catch (err) {
			console.error(`error while calling 'connectors/get': ${err.message}`);
		}
		this.setState({ connector: connector });
	}
  
  	render() {
		return (
			<div className="ConnectorsConfirmation">
				<div className="content">
					<div className="logo">
						<img src={this.state.connector.LogoURL} alt={`${this.state.connector.Name}'s logo`} />
					</div>
					<h1>{this.state.connector.Name}</h1>
					<div className="confirmation">
						{this.state.connector.Name}'s connector has been succesfully installed!
					</div>
					<Link className='btn secondary' to='/admin/account/connectors'>See your connnectors</Link>
				</div>
			</div>
		)
  	}
}

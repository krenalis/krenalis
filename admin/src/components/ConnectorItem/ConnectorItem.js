import React, { Component } from 'react'

import './ConnectorItem.css'

export default class ConnectorItem extends Component {
	render() {
		return (
			<div className="ConnectorItem" onClick={this.props.onClick}>
				<div className="entryTitle">
					<div className="logo">
						{this.props.logoUrl === '' ? <div class='unknownLogo'>?</div> : <img alt={`${this.props.name}'s logo`} src={this.props.logoUrl} />}
					</div>
					<div className="name">{this.props.name}</div>
				</div>
			</div>
		)
	}
}

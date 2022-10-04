import React, { Component } from 'react'
import './ConnectorEntry.css'

export default class ConnectorsEntry extends Component {
	render() {
		return (
			<div className="ConnectorEntry" data-id={this.props.id} onClick={this.props.onClick}>
				<div className="entry-title">
					<div className="logo">
						{this.props.logoUrl === '' ? <div class='unknown-logo'>?</div> : <img alt={`${this.props.name}'s logo`} src={this.props.logoUrl} />}
					</div>
					<div className="name">{this.props.name}</div>
				</div>
				{this.props.onRemove != null ? <div onClick={this.props.onRemove} class='remove-btn'>Remove</div> : ''}
			</div>
		)
	}
}

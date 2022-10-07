import React, { Component } from 'react'

import './AccountConnectorItem.css'
import Button from '../Button/Button'

export default class AccountConnectorItem extends Component {
	render() {
		return (
			<div className='AccountConnectorItem'>
				<div className="info">
					<div className="logo">{this.props.logoURL === '' ? <div class='noLogo'>?</div> : <img alt={`${this.props.name}'s logo`} src={this.props.logoURL} />}</div>
					<div className="name">{this.props.name}</div>
				</div>
				<div className="actions">
					<Button theme="primary" onClick={this.props.onImportClick} text="Import" icon="download" />
					<div className="settings" onClick={this.props.onSettingsClick}><i className="material-symbols-outlined">tune</i></div>
					<div className="delete" onClick={this.props.onDeleteClick}><i className="material-symbols-outlined">delete</i></div>
				</div>
			</div>
		)
	}
}

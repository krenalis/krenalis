import React, { Component } from 'react';

import './AccountConnectors.css';
import AccountConnectorItem from '../../../components/AccountConnectorItem/AccountConnectorItem';
import StatusMessage from '../../../components/StatusMessage/StatusMessage'
import Dialog from '../../../components/Dialog/Dialog'
import Overlay from '../../../components/Overlay/Overlay'
import Button from '../../../components/Button/Button'
import call from '../../../utils/call'

import { Navigate } from 'react-router-dom'

export default class AccountConnectors extends Component {

	constructor(props) {
		super(props);
		this.state = {
			'connectors': [],
			'goToSettings': 0,
			'goToImport': 0,
			'askImportConfirmation': 0,
			'resetCursor': false,
			'status': null
		}
	}

	componentDidMount = async () => {
		let [connectors, err] = await call('/admin/connectors/findInstalledConnectors');
		if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
		this.setState({connectors: connectors});
	}

	handleResetCursorChange = (e) => {
		let value = e.currentTarget.value;
		if (value === "true") this.setState({resetCursor: true});
		else if (value === "false") this.setState({resetCursor: false});
	}

	handleImportClick = async (id) => {
		this.setState({askImportConfirmation: id});
	}

	handleImportConfirmation = async () => {
		let id = this.state.askImportConfirmation;
		let resetCursor = this.state.resetCursor;
		this.setState({status: null});
		let [data, err] = await call('/admin/import-raw-user-data-from-connector', {Connector: id, ResetCursor: resetCursor});
		if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
		console.log(data);
	}

	handleDelete = async (id) => {
		this.setState({status: null});
		let [res, err] = await call('/admin/connectors/delete', [id]);
		if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
		let clone = this.state.connectors.slice();
		let connectors = clone.filter((c) => {
			return c.ID !== id;
		});
		this.setState({connectors: connectors})
	}

	render() {
		if (this.state.goToSettings !== 0) {
			return <Navigate to={`${this.state.goToSettings}`} />
		} else if (this.state.goToImport !== 0) {
			return <Navigate to={`${this.state.goToImport}/import`} />
		} else {
			return (
				<div className="AccountConnectors">
					<div class='content'>
						<h1>Data Sources</h1>
						{this.state.status && <StatusMessage onClose={() => {this.setState({status: null})}} message={this.state.status} />}
						<div className="connectors">
							{
								this.state.connectors.length > 0 ? 
									this.state.connectors.map((c) => {
										return <AccountConnectorItem key={c.ID} name={c.Name} logoURL={c.LogoURL} 
											onImportClick={() => {this.handleImportClick(c.ID)}} 
											onSettingsClick={() => {this.setState({goToSettings: c.ID})}}
											onDeleteClick={() => {this.handleDelete(c.ID)}} />
									}) 
								:
									<div className="empty">You don't have any data source installed yet</div>
							}
						</div>
					</div>
					<Dialog 
						isOpen={this.state.askImportConfirmation !== 0}
						onClose={() => {this.setState({askImportConfirmation: 0})}}
						type="question"
						title="Where do you start?"
						description="select the starting point for your new import"
						width="700px"
					>
						<select name="resetCursor" value={this.state.resetCursor ? "true" : "false"} onChange={this.handleResetCursorChange}>
							<option value="true">Start importing all over again</option>
							<option value="false">Pick up the import from where it left off</option>
						</select>
						<div className="buttons">
							<Button text="Cancel" icon="close" onClick={() => {this.setState({askImportConfirmation: 0})}} />
							<Button theme="primary" text="Import" icon="download" onClick={this.handleImportConfirmation} />
						</div>
					</Dialog>
					<Overlay isOpen={this.state.askImportConfirmation !== 0} />
				</div>
			)
		}
	}
}

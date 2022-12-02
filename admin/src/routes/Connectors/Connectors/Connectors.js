import React from 'react';
import './Connectors.css';
import call from '../../../utils/call';
import Navigation from '../../../components/Navigation/Navigation';
import Card from '../../../components/Card/Card';
import Toast from '../../../components/Toast/Toast';
import { Navigate } from 'react-router-dom';

import { SlButton, SlDialog, SlIcon, SlTooltip, SlInput } from '@shoelace-style/shoelace/dist/react/index.js';

export default class Connectors extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.connectionRole = new URL(document.location).searchParams.get('role');
		this.state = {
			connectors: [],
			storageConnections: [],
			connectorToAdd: null,
			goToConnectionAdded: 0,
			showStorage: false,
			askWebsiteInformations: false,
			status: null,
			websitePort: '',
			websiteHost: '',
		};
	}

	componentDidMount = async () => {
		let [connectors, err] = await call('/admin/connectors/find');
		if (err != null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({ connectors: connectors });
	};

	installConnection = async (c, s, host) => {
		let role = this.connectionRole == null || this.connectionRole === '' ? 'Source' : this.connectionRole;
		let body = { Connector: c.ID, Storage: 0, Role: role, Host: '' };
		if (c.OAuth.URL === '') {
			if (c.Type === 'File') body.Storage = s;
			if (c.Type === 'Website') body.Host = host;
			let [, err] = await call('/admin/add-connection', body);
			if (err != null) {
				this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
				this.toast.current.toast();
				return;
			}
			this.setState({ goToConnectionAdded: c.ID });
			return;
		}
		// install with OAuth.
		document.cookie = `add-connection=${c.ID};path=/`;
		document.cookie = `role=${role};path=/`;
		window.location = c.OAuth.URL;
		return;
	};

	addConnection = async (c) => {
		this.setState({ connectorToAdd: c });
		if (c.Type === 'File') {
			let [cns, err] = await call('/admin/connections/find');
			if (err != null) {
				this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
				this.toast.current.toast();
				return;
			}
			let storageConnections = [];
			for (let c of cns) {
				if (c.Type === 'Storage' && c.Role === this.connectionRole) storageConnections.push(c);
			}
			this.setState({ storageConnections: storageConnections, showStorage: true });
			return;
		}
		if (c.Type === 'Website') {
			this.setState({ askWebsiteInformations: true });
			return;
		}
		await this.installConnection(c);
	};

	addFileConnection = async (storageID) => {
		let c = this.state.connectorToAdd;
		await this.installConnection(c, storageID);
	};

	addWebsiteConnection = async () => {
		let c = this.state.connectorToAdd;
		await this.installConnection(c, 0, this.state.websiteHost + ':' + this.state.websitePort);
		this.setState({ askWebsiteInformations: false });
	};

	render() {
		if (this.state.goToConnectionAdded !== 0) {
			return <Navigate to={`added/${this.state.goToConnectionAdded}`} />;
		} else {
			return (
				<div className='Connectors'>
					<Navigation navItems={[{ name: 'Add a connection', link: '/admin/connectors', selected: true }]} />
					<div class='content'>
						<Toast reactRef={this.toast} status={this.state.status} />
						<div className='connectors'>
							{this.state.connectors.map((c) => {
								return (
									<Card key={c.ID} name={c.Name} logoURL={c.LogoURL} type={c.Type}>
										<SlTooltip content={`Add ${c.Name}`}>
											<SlButton
												size='medium'
												variant='primary'
												onClick={async () => {
													await this.addConnection(c);
												}}
												circle
											>
												<SlIcon name='plus' />
											</SlButton>
										</SlTooltip>
									</Card>
								);
							})}
						</div>
					</div>
					<SlDialog
						label='Select a storage'
						open={this.state.showStorage}
						onSlAfterHide={() => {
							this.setState({ showStorage: false, connectorToAdd: null });
						}}
						style={{ '--width': '600px' }}
					>
						{this.state.storageConnections.length === 0 ? (
							<div className='no-storage'>No storage available</div>
						) : (
							this.state.storageConnections.map((s) => {
								return (
									<div className='storage'>
										<div className='name'>{s.Name}</div>
										<SlButton
											variant='primary'
											onClick={async () => {
												await this.addFileConnection(s.ID);
											}}
											className='addStorage'
										>
											<SlIcon name='arrow-right' />
										</SlButton>
									</div>
								);
							})
						)}
					</SlDialog>
					<SlDialog
						label='Website informations'
						open={this.state.askWebsiteInformations}
						onSlAfterHide={() => {
							this.setState({ askWebsiteInformations: false, connectorToAdd: null });
						}}
						style={{ '--width': '600px' }}
					>
						<div className='websiteInfo'>
							<SlInput
								label='Host'
								className='hostInput'
								onSlChange={(e) => {
									this.setState({ websiteHost: e.currentTarget.value });
								}}
								value={this.state.websiteHost}
							/>
							<SlInput
								label='Port'
								className='portInput'
								onSlChange={(e) => {
									this.setState({ websitePort: e.currentTarget.value });
								}}
								value={this.state.websitePort}
							/>
							<SlButton className='addWebsite' variant='primary' onClick={this.addWebsiteConnection}>
								Add website
							</SlButton>
						</div>
					</SlDialog>
				</div>
			);
		}
	}
}

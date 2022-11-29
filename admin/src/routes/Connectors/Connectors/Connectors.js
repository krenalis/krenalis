import React from 'react';
import './Connectors.css';
import call from '../../../utils/call';
import Navigation from '../../../components/Navigation/Navigation';
import Card from '../../../components/Card/Card';
import Toast from '../../../components/Toast/Toast';
import { Navigate } from 'react-router-dom';
import { SlButton, SlDialog, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

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
			status: null,
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

	installConnection = async (c, s) => {
		let role = this.connectionRole == null || this.connectionRole === '' ? 'Source' : this.connectionRole;
		let body = { Type: c.Type, Connector: c.ID, Storage: 0, Role: role };
		if (c.OAuth.URL === '') {
			if (c.Type === 'File') body.Storage = s; // TODO: QUESTO NON È IL CONNETTORE MA UNA CONNECTION CHE ESISTE !! !!
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
		await this.installConnection(c);
	};

	addFileConnection = async (storageID) => {
		let c = this.state.connectorToAdd;
		await this.installConnection(c, storageID);
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
							this.state.storageConnections.map((c) => {
								return (
									<div className='storage'>
										<div className='name'>{c.Name}</div>
										<SlButton
											variant='primary'
											onClick={async () => {
												await this.addFileConnection(c.ID);
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
				</div>
			);
		}
	}
}

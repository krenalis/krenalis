import React from 'react';
import './ConnectorsConnectionAdded.css';
import Toast from '../../../components/Toast/Toast';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import call from '../../../utils/call';
import { NavLink } from 'react-router-dom';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react';

export default class ConnectorsConnectionAdded extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.connectorID = Number(String(window.location).split('/').pop());
		this.state = {
			connector: {},
			status: null,
		};
	}

	async componentDidMount() {
		let [connector, err] = await call('/admin/connectors/get', this.connectorID);
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({ connector: connector });
	}

	render() {
		return (
			<div className='ConnectorsConnectionAdded'>
				<Breadcrumbs
					breadcrumbs={[
						{ Name: 'Add a connection', Link: '/admin/connectors' },
						{ Name: `${this.state.connector.Name}'s connection added` },
					]}
				/>
				<div className='content'>
					<Toast reactRef={this.toast} status={this.state.status} />
					<div className='addedConnection'>
						<div className='logo'>
							{this.state.connector.LogoURL === '' ? (
								<div class='unknownLogo'>?</div>
							) : (
								<img alt={`${this.state.connector.Name}'s logo`} src={this.state.connector.LogoURL} />
							)}
						</div>
						<div className='title'>{this.state.connector.Name} has been added</div>
						<div className='description'>
							You have successfully added a new connection from {this.state.connector.Name}'s connector
						</div>
					</div>
					<SlButton className='link' variant='text' size='medium'>
						<SlIcon slot='suffix' name='arrow-right-circle' />
						See all your connections
						<NavLink to='/admin/account/connections-map'></NavLink>
					</SlButton>
				</div>
			</div>
		);
	}
}

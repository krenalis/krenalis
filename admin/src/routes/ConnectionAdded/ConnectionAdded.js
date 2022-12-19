import { useState, useEffect, useRef } from 'react';
import './ConnectionAdded.css';
import Toast from '../../components/Toast/Toast';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import call from '../../utils/call';
import { NavLink } from 'react-router-dom';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsConnectionAdded = () => {
	let [connector, setConnector] = useState({});
	let [status, setStatus] = useState(null);

	let toastRef = useRef();
	let lastFragment = String(window.location).split('/').pop();
	let connectorID = Number(lastFragment.substring(0, lastFragment.indexOf('?')));
	let connectionRole = new URL(document.location).searchParams.get('role');

	useEffect(() => {
		const fetchConnector = async () => {
			let [connector, err] = await call('/admin/connectors/get', 'POST', connectorID);
			if (err !== null) {
				setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
				toastRef.current.toast();
				return;
			}
			setConnector(connector);
		};
		fetchConnector();
	}, []);

	return (
		<div className='ConnectionAdded'>
			<Breadcrumbs
				breadcrumbs={[
					{ Name: 'Connections map', Link: '/admin/connections-map' },
					{ Name: `Add a new ${connectionRole}`, Link: `/admin/connectors/?role=${connectionRole}` },
					{ Name: `${connector.Name} connection added` },
				]}
			/>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<div className='addedConnection'>
					<div className='logo'>
						{connector.LogoURL === '' ? (
							<div class='unknownLogo'>?</div>
						) : (
							<img alt={`${connector.Name}'s logo`} src={connector.LogoURL} />
						)}
					</div>
					<div className='title'>{connector.Name} connection has been added</div>
					<div className='description'>
						You have successfully added a new connection based on the {connector.Name} connector
					</div>
				</div>
				<SlButton className='link' variant='text' size='medium'>
					<SlIcon slot='suffix' name='arrow-right-circle' />
					See all your connections
					<NavLink to='/admin/connections-map'></NavLink>
				</SlButton>
			</div>
		</div>
	);
};

export default ConnectorsConnectionAdded;

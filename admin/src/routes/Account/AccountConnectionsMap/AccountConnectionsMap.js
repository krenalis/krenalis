import { useState, useEffect, useRef } from 'react';
import './AccountConnectionsMap.css';
import Navigation from '../../../components/Navigation/Navigation';
import Toast from '../../../components/Toast/Toast';
import call from '../../../utils/call';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';
import Xarrow from 'react-xarrows';

const AccountConnectionsMap = () => {
	let [sources, setSources] = useState([]);
	let [destinations, setDestinations] = useState([]);
	let [status, setStatus] = useState(null);

	let toastRef = useRef();

	useEffect(() => {
		const fetchConnections = async () => {
			let [connections, err] = await call('/admin/connections/find', 'GET');
			if (err !== null) {
				setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
				toastRef.current.toast();
				return;
			}
			let sources = [];
			let destinations = [];
			for (let c of connections) {
				if (c.Role === 'Source') sources.push(c);
				if (c.Role === 'Destination') destinations.push(c);
			}
			setSources(sources);
			setDestinations(destinations);
		};
		fetchConnections();
	}, []);

	const renderConnections = (cns) => {
		let connections = [];
		for (let c of cns) {
			if (c.Type === 'Storage') {
				let files = [];
				if (c.Role === 'Source') {
					files = sources.filter((cn) => cn.Storage === c.ID);
				} else if (c.Role === 'Destination') {
					files = destinations.filter((cn) => cn.Storage === c.ID);
				}
				if (files.length > 0) {
					connections.push(
						<div className='connection storage' id={`${c.Role.toLowerCase()}-${c.ID}`}>
							<div className='files'>
								{files.map((f) => {
									return (
										<div className='file' id={`file-${f.ID}`}>
											{f.LogoURL === '' ? (
												<div class='unknownLogo'>?</div>
											) : (
												<div className='littleLogo'>
													<img src={f.LogoURL} rel='noreferrer' alt={`${f.Name}'s logo`} />
												</div>
											)}
											{f.Name}
										</div>
									);
								})}
							</div>
							<div className='storage' id={`storage-${c.ID}`}>
								<div className='littleLogo'>
									<img src={c.LogoURL} rel='noreferrer' alt={`${c.Name}'s logo`} />
								</div>
								{c.Name}
							</div>
							{files.map((f) => {
								return (
									<div key={`arrow-${f.ID}`} className='arrow'>
										<Xarrow
											start={`file-${f.ID}`}
											end={`storage-${c.ID}`}
											startAnchor={c.Role === 'Source' ? 'right' : 'left'}
											endAnchor={c.Role === 'Source' ? 'left' : 'right'}
											showHead={false}
											color='#e4e4e7'
											strokeWidth={2}
										/>
									</div>
								);
							})}
						</div>
					);
					continue;
				}
			}
			if (c.Type !== 'File') {
				connections.push(
					<div className='connection' id={`${c.Role.toLowerCase()}-${c.ID}`}>
						{c.LogoURL === '' ? (
							<div class='unknownLogo'>?</div>
						) : (
							<div className='littleLogo'>
								<img src={c.LogoURL} rel='noreferrer' alt={`${c.Name}'s logo`} />
							</div>
						)}
						{c.Name}
					</div>
				);
			}
		}
		return connections;
	};

	return (
		<div className='AccountConnectionsMap'>
			<Navigation
				navItems={[
					{ name: 'Your connections map', link: '/admin/account/connections-map', selected: true },
					{ name: 'Your connections', link: '/admin/account/connections', selected: false },
					{ name: 'Your schemas', link: '/admin/account/schemas', selected: false },
				]}
			/>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<div className='buttons'>
					<SlButton className='addSource' variant='neutral'>
						<SlIcon slot='suffix' name='plus-circle-dotted' />
						Add a new source
						<NavLink to='/admin/connectors?role=Source'></NavLink>
					</SlButton>
					<SlButton className='addDestination' variant='neutral'>
						<SlIcon slot='suffix' name='plus-circle-dotted' />
						Add a new destination
						<NavLink to='/admin/connectors?role=Destination'></NavLink>
					</SlButton>
				</div>
				<div className='map'>
					<div className='sources'>{renderConnections(sources)}</div>
					<div className='main'>
						<div className='centralLogo' id='centralLogo'>
							Chichi
						</div>
						<div className='databases'>
							<div className='database users' id='usersDatabase'>
								<SlIcon name='database' />
								<div className='name'>Users</div>
							</div>
							<div className='database events' id='eventsDatabase'>
								<SlIcon name='database' />
								<div className='name'>Events</div>
							</div>
						</div>
					</div>
					<div className='destinations'>{renderConnections(destinations)}</div>
				</div>
			</div>
			<div className='arrows'>
				{sources.map((s) => {
					if (s.Type !== 'File') {
						return (
							<div key={`arrow-${s.ID}`} className='arrow'>
								<Xarrow
									start={`source-${s.ID}`}
									end='centralLogo'
									startAnchor='right'
									endAnchor='left'
									showHead={false}
									color='#e4e4e7'
									strokeWidth={2}
								/>
							</div>
						);
					}
					return null;
				})}
				{destinations.map((d) => {
					if (d.Type !== 'File') {
						return (
							<div key={`arrow-${d.ID}`} className='arrow'>
								<Xarrow
									start={`destination-${d.ID}`}
									end='centralLogo'
									startAnchor='left'
									endAnchor='right'
									showHead={false}
									color='#e4e4e7'
									strokeWidth={2}
								/>
							</div>
						);
					}
					return null;
				})}
				<div className='arrow'>
					<Xarrow
						start='centralLogo'
						end='usersDatabase'
						startAnchor='bottom'
						endAnchor='top'
						showHead={false}
						color='#e4e4e7'
						strokeWidth={2}
					/>
				</div>
				<div className='arrow'>
					<Xarrow
						start='centralLogo'
						end='eventsDatabase'
						startAnchor='bottom'
						endAnchor='top'
						showHead={false}
						color='#e4e4e7'
						strokeWidth={2}
					/>
				</div>
			</div>
		</div>
	);
};

export default AccountConnectionsMap;

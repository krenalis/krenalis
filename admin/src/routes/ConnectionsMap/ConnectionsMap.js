import { useState, useEffect, useRef } from 'react';
import './ConnectionsMap.css';
import Navigation from '../../components/Navigation/Navigation';
import ConnectionBlock from '../../components/ConnectionBlock/ConnectionBlock';
import LinkedConnectionBlocks from '../../components/LinkedConnectionBlocks/LinkedConnectionBlocks';
import Arrow from '../../components/Arrow/Arrow';
import Toast from '../../components/Toast/Toast';
import call from '../../utils/call';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';

const ConnectionsMap = () => {
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
				connections.push(
					<LinkedConnectionBlocks
						primaryConnection={c}
						primaryColumn={c.Role === 'Source' ? 'right' : 'left'}
						secondaryConnections={files}
						startAnchor={c.Role === 'Source' ? 'left' : 'right'}
						endAnchor={c.Role === 'Source' ? 'right' : 'left'}
					></LinkedConnectionBlocks>
				);
			} else if (c.Type === 'Stream') {
				let streamed = sources.filter((cn) => cn.Stream === c.ID);
				connections.push(
					<LinkedConnectionBlocks
						primaryConnection={c}
						primaryColumn={c.Role === 'Source' ? 'right' : 'left'}
						secondaryConnections={streamed}
						startAnchor={c.Role === 'Source' ? 'left' : 'right'}
						endAnchor={c.Role === 'Source' ? 'right' : 'left'}
					></LinkedConnectionBlocks>
				);
			} else if (c.Storage === 0 && c.Stream === 0) {
				connections.push(<ConnectionBlock connection={c}></ConnectionBlock>);
			}
		}
		return connections;
	};

	return (
		<div className='ConnectionsMap'>
			<Navigation navItems={[{ name: 'Connections', link: '/admin/connections', selected: true }]} />
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
							C
						</div>
						<div className='databases'>
							<div className='database users' id='usersDatabase'>
								<SlIcon name='database' />
								<div className='name'>Users</div>
								<NavLink to='/admin/users'></NavLink>
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
				{sources.map((c) => {
					if (c.Storage === 0 && c.Stream === 0) {
						return <Arrow start={`${c.ID}`} end='centralLogo' startAnchor='right' endAnchor='left' />;
					}
					return null;
				})}
				{destinations.map((c) => {
					if (c.Storage === 0 && c.Stream === 0) {
						return <Arrow start={`${c.ID}`} end='centralLogo' startAnchor='left' endAnchor='right' />;
					}
					return null;
				})}
				<Arrow start='centralLogo' end='usersDatabase' startAnchor='bottom' endAnchor='top' />
				<Arrow start='centralLogo' end='eventsDatabase' startAnchor='bottom' endAnchor='top' />
			</div>
		</div>
	);
};

export default ConnectionsMap;

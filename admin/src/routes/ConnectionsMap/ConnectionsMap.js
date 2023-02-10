import { useState, useEffect, useContext } from 'react';
import './ConnectionsMap.css';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Header from '../../components/Header/Header';
import ConnectionBlock from '../../components/ConnectionBlock/ConnectionBlock';
import LinkedConnectionBlocks from '../../components/LinkedConnectionBlocks/LinkedConnectionBlocks';
import Arrow from '../../components/Arrow/Arrow';
import { AppContext } from '../../context/AppContext';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';

const ConnectionsMap = () => {
	let [sources, setSources] = useState([]);
	let [destinations, setDestinations] = useState([]);

	let { API, showError } = useContext(AppContext);

	let newConnection = Number(new URL(document.location).searchParams.get('new'));

	useEffect(() => {
		const fetchConnections = async () => {
			let [connections, err] = await API.connections.find();
			if (err) {
				showError(err);
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
						newConnection={newConnection}
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
						newConnection={newConnection}
					></LinkedConnectionBlocks>
				);
			} else if (c.Storage === 0 && c.Stream === 0) {
				connections.push(<ConnectionBlock connection={c} isNew={c.ID === newConnection}></ConnectionBlock>);
			}
		}
		return connections;
	};

	return (
		<div className='ConnectionsMap'>
			<PrimaryBackground height={250} overlap={100}>
				<Header />
			</PrimaryBackground>
			<div className='routeContent'>
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
						return (
							<Arrow
								start={`${c.ID}`}
								end='centralLogo'
								startAnchor='right'
								endAnchor='left'
								isNew={c.ID === newConnection}
							/>
						);
					}
					return null;
				})}
				{destinations.map((c) => {
					if (c.Storage === 0 && c.Stream === 0) {
						return (
							<Arrow
								start={`${c.ID}`}
								end='centralLogo'
								startAnchor='left'
								endAnchor='right'
								isNew={c.ID === newConnection}
							/>
						);
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

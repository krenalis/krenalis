import { useContext, useEffect, useState } from 'react';
import './ConnectionsWrapper.css';
import { NavigationContext } from '../../context/NavigationContext';
import { ConnectionsContext } from '../../context/ConnectionsContext';
import { AppContext } from '../../context/AppContext';
import { Outlet } from 'react-router-dom';

const ConnectionsWrapper = () => {
	let [connections, setConnections] = useState(null);
	let [areConnectionsStale, setAreConnectionsStale] = useState(false);

	let { setCurrentRoute } = useContext(NavigationContext);
	setCurrentRoute('connections');

	let { API, showError } = useContext(AppContext);

	useEffect(() => {
		const fetchConnections = async () => {
			let [connections, err] = await API.connections.find();
			if (err) {
				setConnections([]);
				showError(err);
				return;
			}
			setConnections(connections);
			setAreConnectionsStale(false);
		};
		if (connections == null || areConnectionsStale) {
			fetchConnections();
		}
	}, [areConnectionsStale]);

	// TODO: add a global loading state.
	if (connections == null) {
		return;
	}

	return (
		// This context, which contains the fetched connections, is only used to
		// decrease latency when navigating between the top-level connections
		// routes. When navigating inside a specific connection, additional
		// informations must be fetched via API calls.
		<ConnectionsContext.Provider value={{ connections, setAreConnectionsStale }}>
			<Outlet />
		</ConnectionsContext.Provider>
	);
};

export default ConnectionsWrapper;

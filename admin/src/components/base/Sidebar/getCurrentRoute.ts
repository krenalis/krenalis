import { Location } from 'react-router-dom';
import { UI_BASE_PATH } from '../../../constants/paths';
import TransformedConnection from '../../../lib/core/connection';

const getCurrentRoute = (location: Location, connections: TransformedConnection[]): string => {
	const pathName = location.pathname;

	const fragments = pathName.split('/');
	const isConnectionsRelated =
		fragments.includes('connections') ||
		fragments.includes('connectors') ||
		fragments.includes('oauth') ||
		pathName === UI_BASE_PATH;

	let currentRoute = '';
	if (isConnectionsRelated) {
		currentRoute = 'connections';

		const isConnection = fragments.includes('connections');
		if (isConnection) {
			// Check if it is a source or destination connection.
			const i = fragments.findIndex((s) => s === 'connections');
			if (i !== -1 && fragments.length - 1 > i) {
				const resource = fragments[i + 1];
				if (resource === 'sources') {
					currentRoute = 'connections/sources';
				} else if (resource === 'destinations') {
					currentRoute = 'connections/destinations';
				} else {
					const connectionID = resource;
					const connection = connections.find((c) => c.id === connectionID);
					if (connection != null) {
						if (connection.isSource) {
							currentRoute = 'connections/sources';
						} else {
							currentRoute = 'connections/destinations';
						}
					}
				}
			}
		}

		const isConnector = fragments.includes('connectors');
		if (isConnector) {
			// Check if it is a source or destination connector.
			const params = new URLSearchParams(location.search);
			const role = params.get('role');
			if (role === 'Source') {
				currentRoute = 'connections/sources';
			} else if (role === 'Destination') {
				currentRoute = 'connections/destinations';
			}
		}
	} else if (fragments.includes('users')) {
		currentRoute = 'users';
	} else if (fragments.includes('profile-unification')) {
		currentRoute = 'profile-unification';
		const i = fragments.findIndex((s) => s === 'profile-unification');
		if (i !== -1 && fragments.length - 1 > i) {
			const resource = fragments[i + 1];
			if (resource === 'profiles') {
				currentRoute = 'profile-unification/profiles';
			} else if (resource === 'schema') {
				currentRoute = 'profile-unification/schema';
			} else if (resource === 'rules') {
				currentRoute = 'profile-unification/rules';
			}
		}
	} else if (fragments.includes('settings')) {
		currentRoute = 'settings';
		const lastFragment = fragments[fragments.length - 1];
		if (lastFragment === 'general') {
			currentRoute = 'settings/general';
		} else if (lastFragment === 'data-warehouse') {
			currentRoute = 'settings/dataWarehouse';
		}
	}

	return currentRoute;
};

export default getCurrentRoute;

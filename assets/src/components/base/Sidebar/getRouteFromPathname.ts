import { UI_BASE_PATH } from '../../../constants/paths';
import TransformedConnection from '../../../lib/core/connection';

const getRouteFromPathname = (route: string, connections: TransformedConnection[]): string => {
	const fragments = route.split('/');
	const isConnectionsRelated =
		fragments.includes('connections') ||
		fragments.includes('connectors') ||
		fragments.includes('oauth') ||
		route === UI_BASE_PATH;

	let currentRoute = '';
	if (isConnectionsRelated) {
		currentRoute = 'connections';
		const i = fragments.findIndex((s) => s === 'connections');
		if (i !== -1 && fragments.length - 1 > i) {
			const resource = fragments[i + 1];
			if (resource === 'sources') {
				currentRoute = 'connections/sources';
			} else if (resource === 'destinations') {
				currentRoute = 'connections/destinations';
			} else {
				const connectionID = Number(resource);
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
	} else if (fragments.includes('users')) {
		currentRoute = 'users';
	} else if (fragments.includes('schema')) {
		currentRoute = 'schema';
	} else if (fragments.includes('settings')) {
		currentRoute = 'settings';
		const lastFragment = fragments[fragments.length - 1];
		if (lastFragment === 'general') {
			currentRoute = 'settings/general';
		} else if (lastFragment === 'identity-resolution') {
			currentRoute = 'settings/identityResolution';
		} else if (lastFragment === 'data-warehouse') {
			currentRoute = 'settings/dataWarehouse';
		}
	}

	return currentRoute;
};

export default getRouteFromPathname;

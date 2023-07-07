import { adminBasePath } from '../../../constants/path';

const getRouteFromPathname = (route, connections) => {
	const fragments = route.split('/');
	const isConnectionsRelated =
		fragments.includes('connections') ||
		fragments.includes('connectors') ||
		fragments.includes('oauth') ||
		route === adminBasePath;

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
	} else if (fragments.includes('anonymous-identity')) {
		currentRoute = 'anonymousIdentity';
	}

	return currentRoute;
};

export default getRouteFromPathname;

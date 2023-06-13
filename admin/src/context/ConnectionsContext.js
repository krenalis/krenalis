import { createContext } from 'react';

const defaultConnectionsContext = {
	connections: [],
	setAreConnectionsStale: () => {},
};

export const ConnectionsContext = createContext(defaultConnectionsContext);

import { createContext } from 'react';

const defaultConnectionContext = {
	connection: {},
	setCurrentConnectionSection: () => {},
	setConnection: () => {},
};

export const ConnectionContext = createContext(defaultConnectionContext);

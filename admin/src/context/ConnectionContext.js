import { createContext } from 'react';

const defaultConnectionContext = {
	c: {},
	setCurrentConnectionSection: () => {},
	setConnection: () => {},
};

export const ConnectionContext = createContext(defaultConnectionContext);

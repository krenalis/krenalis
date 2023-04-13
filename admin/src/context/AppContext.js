import { createContext } from 'react';
import API from '../api/api';

const defaultAppContext = {
	API: new API(),
	showStatus: () => {},
	showError: () => {},
	showNotFound: () => {},
	redirect: () => {},
	setIsFullScreen: () => {},
};

export const AppContext = createContext(defaultAppContext);

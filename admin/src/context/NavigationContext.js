import { createContext } from 'react';

const defaultNavigationContext = {
	setCurrentRoute: () => {},
	setCurrentTitle: () => {},
};

export const NavigationContext = createContext(defaultNavigationContext);

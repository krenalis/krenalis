import { createContext } from 'react';

const defaultNavigationContext = {
	setCurrentRoute: () => {},
	setCurrentTitle: () => {},
	setPreviousRoute: () => {},
};

export const NavigationContext = createContext(defaultNavigationContext);

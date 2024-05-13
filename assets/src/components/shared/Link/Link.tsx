import './Link.css';
import React, { ReactNode, useContext } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { uiBasePath } from '../../../constants/path';
import AppContext from '../../../context/AppContext';

interface LinkProps {
	children: ReactNode;
	path: string | null;
	onRedirect?: (...args: any) => void | Promise<void>;
}

const Link = ({ children, path, onRedirect }: LinkProps) => {
	const { setIsLoadingState, toastRef } = useContext(AppContext);

	const onClick = async (e) => {
		if (onRedirect) {
			await onRedirect();
		}
		if (e.ctrlKey || e.metaKey) {
			// do not execute the following logic as the link will be
			// opened in a new tab.
			return;
		}
		if (toastRef?.current) {
			toastRef.current.hide();
		}
		if (`${uiBasePath}${path}` === location.pathname) {
			setIsLoadingState(true);
			e.preventDefault();
			return;
		}
	};

	if (path == null) {
		return children;
	}

	return (
		<RouterLink to={`${uiBasePath}${path}`} onClick={onClick}>
			{children}
		</RouterLink>
	);
};

export { Link };

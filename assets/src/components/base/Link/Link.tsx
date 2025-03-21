import './Link.css';
import React, { ReactNode, useContext } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { UI_BASE_PATH } from '../../../constants/paths';
import AppContext from '../../../context/AppContext';

interface LinkProps {
	children: ReactNode;
	path: string | null;
	className?: string;
	onRedirect?: (...args: any) => void | Promise<void>;
}

const Link = ({ children, path, className, onRedirect }: LinkProps) => {
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
		if (`${UI_BASE_PATH}${path}` === location.pathname) {
			setIsLoadingState(true);
			e.preventDefault();
			return;
		}
	};

	if (path == null) {
		return children;
	}

	return (
		<RouterLink to={`${UI_BASE_PATH}${path}`} onClick={onClick} className={className ? className : ''}>
			{children}
		</RouterLink>
	);
};

export { Link };

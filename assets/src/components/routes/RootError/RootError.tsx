import React from 'react';
import './RootError.css';
import IconWrapper from '../../base/IconWrapper/IconWrapper';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { UI_BASE_PATH } from '../../../constants/paths';
import { storageKeysToBeRemoved } from '../../../constants/storage';

const RootError = () => {
	const onRestartApp = () => {
		sessionStorage.clear();
		for (const key of storageKeysToBeRemoved) {
			localStorage.removeItem(key);
		}
		window.location.href = `${UI_BASE_PATH}`;
	};

	return (
		<div className='root-error'>
			<div className='root-error__message'>
				<IconWrapper name='exclamation-circle' size={40} />
				<div className='root-error__title'>An error occurred in the Admin console</div>
				<div className='root-error__instructions'>If the issue persists, please contact the administrator</div>
				<SlButton variant='primary' className='root-error__action' onClick={onRestartApp}>
					Restart the Admin console
				</SlButton>
			</div>
		</div>
	);
};

export default RootError;

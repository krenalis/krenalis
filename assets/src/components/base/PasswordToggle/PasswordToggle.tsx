import React, { useState } from 'react';
import './PasswordToggle.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';

interface PasswordToggleInterface {
	password: string;
}

const PasswordToggle = ({ password }: PasswordToggleInterface) => {
	const [isVisible, setIsVisible] = useState<boolean>(false);

	const onToggle = () => setIsVisible(!isVisible);

	return (
		<div className='password-toggle'>
			<p className='password-toggle__password'>{isVisible ? password : '●'.repeat(password.length)}</p>
			<div className='password-toggle__toggle-button'>
				<SlButton variant='default' size='small' onClick={onToggle}>
					{isVisible ? 'Hide' : 'Show'}
				</SlButton>
			</div>
		</div>
	);
};

export default PasswordToggle;

import React from 'react';
import './UnknownLogo.css';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

interface UnknownLogoProps {
	size: number;
}

const UnknownLogo = ({ size }: UnknownLogoProps) => {
	return (
		<div className='unknown-logo'>
			<WaIcon name='plug' style={{ fontSize: `${size}px` }}></WaIcon>
		</div>
	);
};

export default UnknownLogo;

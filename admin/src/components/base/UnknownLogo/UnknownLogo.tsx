import React from 'react';
import './UnknownLogo.css';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface UnknownLogoProps {
	size: number;
}

const UnknownLogo = ({ size }: UnknownLogoProps) => {
	return (
		<div className='unknown-logo'>
			<SlIcon name='plug' style={{ fontSize: `${size}px` }}></SlIcon>
		</div>
	);
};

export default UnknownLogo;

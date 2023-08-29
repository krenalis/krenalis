import React from 'react';
import './UnknownLogo.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

interface UnknownLogoProps {
	size: number;
}

const UnknownLogo = ({ size }: UnknownLogoProps) => {
	return (
		<div className='unknownLogo'>
			<SlIcon name='question-lg' style={{ fontSize: `${size}px` }}></SlIcon>
		</div>
	);
};

export default UnknownLogo;

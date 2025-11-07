import React from 'react';
import './LittleLogo.css';
import { ExternalLogo } from '../../routes/ExternalLogo/ExternalLogo';

interface LittleLogoProps {
	code: string;
}

const LittleLogo = ({ code }: LittleLogoProps) => {
	return (
		<div className='little-logo'>
			<ExternalLogo code={code} />
		</div>
	);
};

export default LittleLogo;

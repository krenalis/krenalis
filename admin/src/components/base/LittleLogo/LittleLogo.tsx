import React from 'react';
import './LittleLogo.css';
import { ExternalLogo } from '../../routes/ExternalLogo/ExternalLogo';

interface LittleLogoProps {
	code: string;
	path: string;
}

const LittleLogo = ({ code, path }: LittleLogoProps) => {
	return (
		<div className='little-logo'>
			<ExternalLogo code={code} path={path} />
		</div>
	);
};

export default LittleLogo;

import React from 'react';
import './LittleLogo.css';

interface LittleLogoProps {
	icon: string;
}

const LittleLogo = ({ icon }: LittleLogoProps) => {
	let logo: string;
	if (icon === '') {
		logo = `<div class='unknownLogo'>?</div>`;
	} else {
		logo = icon;
	}
	return <div className='littleLogo' dangerouslySetInnerHTML={{ __html: logo }}></div>;
};

export default LittleLogo;

import { useRef, useEffect } from 'react';
import './LittleLogo.css';

const LittleLogo = ({ icon }) => {
	const logoRef = useRef(null);

	useEffect(() => {
		let logo;
		if (icon === '') {
			logo = `<div class='unknownLogo'>?</div>`;
		} else {
			logo = icon;
		}
		logoRef.current.innerHTML = logo;
	}, []);

	return <div className='littleLogo' ref={logoRef}></div>;
};

export default LittleLogo;

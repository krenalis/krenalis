import { useRef, useEffect } from 'react';
import './LittleLogo.css';

const LittleLogo = ({ icon }) => {
	let logoRef = useRef(null);

	useEffect(() => {
		let logo;
		if (icon === '') {
			logo = `<div class='unknownLogo'>?</div>`;
		} else {
			logo = icon;
		}
		logoRef.current.innerHTML = logo;
	}, []);

	return <div className='LittleLogo' ref={logoRef}></div>;
};

export default LittleLogo;

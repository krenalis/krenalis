import { useRef, useEffect } from 'react';
import './Card.css';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

const Card = ({ icon, name, type, description, children }) => {
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

	return (
		<div className='Card'>
			<div className='top'>
				<div className='logo' ref={logoRef}></div>
				<div className='name'>{name}</div>
				{type && (
					<SlBadge className='type' variant='neutral'>
						{type}
					</SlBadge>
				)}
				<div className='description'>{description}</div>
			</div>
			<div className='body'>{children}</div>
		</div>
	);
};

export default Card;

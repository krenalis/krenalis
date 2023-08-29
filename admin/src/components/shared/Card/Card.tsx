import React, { useRef, useEffect, ReactNode } from 'react';
import './Card.css';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

interface CardProps {
	icon?: string;
	name: string;
	type?: string;
	description?: string;
	children: ReactNode;
}

const Card = ({ icon, name, type, description, children }: CardProps) => {
	const logoRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		let logo: string;
		if (icon == null || icon === '') {
			logo = `<div class='unknownLogo'>?</div>`;
		} else {
			logo = icon;
		}
		if (logoRef.current) {
			logoRef.current.innerHTML = logo;
		}
	}, []);

	return (
		<div className='card'>
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

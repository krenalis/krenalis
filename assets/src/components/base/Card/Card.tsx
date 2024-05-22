import React, { useRef, useEffect, ReactNode } from 'react';
import './Card.css';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';

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
			logo = `<div class='card__unknown-logo'>?</div>`;
		} else {
			logo = icon;
		}
		if (logoRef.current) {
			logoRef.current.innerHTML = logo;
		}
	}, []);

	return (
		<div className='card'>
			<div className='card__top'>
				<div className='card__logo' ref={logoRef}></div>
				<div className='card__name'>{name}</div>
				{type && (
					<SlBadge className='card__type' variant='neutral'>
						{type}
					</SlBadge>
				)}
				<div className='card__description'>{description}</div>
			</div>
			<div className='card__body'>{children}</div>
		</div>
	);
};

export default Card;

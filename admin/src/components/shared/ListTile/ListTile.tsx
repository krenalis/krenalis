import React, { MouseEventHandler, ReactNode } from 'react';
import './ListTile.css';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface ListTileProps {
	icon: ReactNode;
	name: string;
	description?: string;
	disabled?: boolean;
	disablingReason?: string;
	onClick: MouseEventHandler;
	className?: string;
}

const ListTile = ({ icon, name, description, disabled, disablingReason, onClick, className }: ListTileProps) => {
	return (
		<div
			className={`listTile${className ? ' ' + className : ''}${disabled ? ' disabled' : ''}`}
			onClick={disabled ? undefined : onClick}
		>
			<div className='tileIcon'>{icon}</div>
			<div className='tileName'>{name}</div>
			<div className='tileDescription'>
				{description}
				{disablingReason && <div className='disablingReason'>{disablingReason}</div>}
			</div>
			{!disabled && <SlIcon className='tileChevron' name='chevron-right' />}
		</div>
	);
};

export default ListTile;

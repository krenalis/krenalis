import React, { MouseEventHandler, ReactNode } from 'react';
import './ListTile.css';

interface ListTileProps {
	icon: ReactNode;
	name: ReactNode;
	description?: ReactNode;
	disabled?: boolean;
	disablingReason?: string;
	action?: ReactNode;
	onClick?: MouseEventHandler;
	className?: string;
}

const ListTile = ({
	icon,
	name,
	description,
	disabled,
	disablingReason,
	action,
	onClick,
	className,
}: ListTileProps) => {
	return (
		<div
			className={`listTile${className ? ' ' + className : ''}${disabled ? ' disabled' : ''}`}
			onClick={disabled ? null : onClick}
			style={
				onClick && !disabled ? { cursor: 'pointer' } : onClick && disabled ? { cursor: 'not-allowed' } : null
			}
		>
			<div className='tileContent'>
				<div className='tileIcon'>{icon}</div>
				<div className='tileName'>{name}</div>
				<div className='tileDescription'>
					{description}
					{disablingReason && <div className='disablingReason'>{disablingReason}</div>}
				</div>
			</div>
			{!disabled && action}
		</div>
	);
};

export default ListTile;

import React, { MouseEventHandler, ReactNode } from 'react';
import './ListTile.css';

interface ListTileProps {
	icon: ReactNode;
	name: ReactNode;
	id?: string;
	description?: ReactNode;
	showHover?: boolean;
	disabled?: boolean;
	disablingReason?: string;
	action?: ReactNode;
	onClick?: MouseEventHandler;
	className?: string;
}

const ListTile = ({
	icon,
	name,
	id,
	description,
	showHover,
	disabled,
	disablingReason,
	action,
	onClick,
	className,
}: ListTileProps) => {
	return (
		<div
			className={`list-tile${className ? ' ' + className : ''}${showHover ? ' list-tile--show-hover' : ''}${disabled ? ' list-tile--disabled' : ''}`}
			onClick={disabled ? null : onClick}
			data-id={id ? id : ''}
			style={
				onClick && !disabled ? { cursor: 'pointer' } : onClick && disabled ? { cursor: 'not-allowed' } : null
			}
		>
			<div className='list-tile__content'>
				<div className='list-tile__icon'>{icon}</div>
				<div className='list-tile__name'>{name}</div>
				<div className='list-tile__description'>
					{description}
					{disablingReason && <div className='list-tile__disabling-reason'>{disablingReason}</div>}
				</div>
			</div>
			{!disabled && action}
		</div>
	);
};

export default ListTile;

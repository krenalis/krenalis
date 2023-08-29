import React, { MouseEventHandler, ReactNode } from 'react';
import './ListTile.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

interface ListTileProps {
	icon: ReactNode;
	name: string;
	description: string;
	missingSchema: boolean;
	onClick: MouseEventHandler;
}

const ListTile = ({ icon, name, description, missingSchema, onClick }: ListTileProps) => {
	return (
		<div className={`listTile${missingSchema ? ' disabled' : ''}`} onClick={missingSchema ? undefined : onClick}>
			<div className='tileIcon'>{icon}</div>
			<div className='tileName'>{name}</div>
			<div className='tileDescription'>
				{description}
				{missingSchema && <div className='disablingReason'>Missing schema</div>}
			</div>
			{missingSchema ? null : <SlIcon className='tileChevron' name='chevron-right' />}
		</div>
	);
};

export default ListTile;

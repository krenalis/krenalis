import './ListTile.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ListTile = ({ icon, name, description, disabled, disablingReason, onClick }) => {
	return (
		<div className={`listTile${disabled ? ' disabled' : ''}`} onClick={disabled ? null : onClick}>
			<div className='tileIcon'>{icon}</div>
			<div className='tileName'>{name}</div>
			<div className='tileDescription'>
				{description}
				{disabled && disablingReason !== '' && <div className='disablingReason'>{disablingReason}</div>}
			</div>
			{disabled ? null : <SlIcon className='tileChevron' name='chevron-right' />}
		</div>
	);
};

export default ListTile;

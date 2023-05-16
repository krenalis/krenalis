import './ListTile.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ListTile = ({ icon, name, description, missingSchema, onClick }) => {
	return (
		<div className={`listTile${missingSchema ? ' disabled' : ''}`} onClick={missingSchema ? null : onClick}>
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

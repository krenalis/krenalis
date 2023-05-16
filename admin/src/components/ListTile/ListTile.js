import './ListTile.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ListTile = ({ icon, name, description, hasSchema, onClick }) => {
	return (
		<div className={`listTile${hasSchema ? '' : ' disabled'}`} onClick={hasSchema ? onClick : null}>
			<div className='tileIcon'>{icon}</div>
			<div className='tileName'>{name}</div>
			<div className='tileDescription'>
				{description}
				{!hasSchema && <div className='disablingReason'>No schema</div>}
			</div>
			{hasSchema ? <SlIcon className='tileChevron' name='chevron-right' /> : null}
		</div>
	);
};

export default ListTile;

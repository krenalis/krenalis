import './ConnectionProperty.css';
import { SlIconButton } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionProperty = ({
	name,
	label,
	role,
	type,
	isSelected,
	onHandle,
	onRemove,
	disableRemove,
	onConnect,
	connectable,
}) => {
	let hasLabel = label != null && label !== '';
	return (
		<div key={name} className={`ConnectionProperty ${role}${isSelected ? ' selected' : ''}`} id={name}>
			<div className='text'>
				<span className='primary'>{hasLabel ? label : name}</span>
				{hasLabel && <span className='secondary'>{`(${name})`}</span>}
				<div className='type'>{type.name}</div>
			</div>

			<SlIconButton name='dash-circle' label='Remove property' disabled={disableRemove} onClick={onRemove} />
			{connectable && <div className='handle' onClick={onConnect != null ? onConnect : onHandle}></div>}
		</div>
	);
};

export default ConnectionProperty;

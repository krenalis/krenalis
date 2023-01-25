import './ConnectionProperty.css';
import { SlIconButton } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionProperty = ({ name, label, role, isSelected, onHandle, onRemove, disableRemove, onConnect }) => {
	return (
		<div key={name} className={`ConnectionProperty ${role}${isSelected ? ' selected' : ''}`} id={name}>
			<div>{label ? label : name}</div>
			<SlIconButton name='dash-circle' label='Remove property' disabled={disableRemove} onClick={onRemove} />
			<div className='handle' onClick={onConnect != null ? onConnect : onHandle}></div>
		</div>
	);
};

export default ConnectionProperty;

import './SelectedPropertyMessage.css';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const SelectedPropertyMessage = ({ selectedProperty: sp, onClose }) => {
	return (
		<div className='SelectedPropertyMessage'>
			<div>
				Add a mapping
				{sp.role === 'input' ? ' from ' : ' to '}
				<span className='name'>"{sp.label === '' || sp.label == null ? sp.name : sp.label}"</span>
			</div>
			<SlButton className='removeSelectedProperty' variant='neutral' onClick={onClose}>
				Close
			</SlButton>
		</div>
	);
};

export default SelectedPropertyMessage;

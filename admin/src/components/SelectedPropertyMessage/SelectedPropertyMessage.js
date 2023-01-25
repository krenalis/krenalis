import './SelectedPropertyMessage.css';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const SelectedPropertyMessage = ({ selectedProperty: sp, onClose }) => {
	return (
		<div className='SelectedPropertyMessage'>
			<div>
				Add a mapping
				{sp.role === 'input' ? ' from ' : ' to '}
				<span className='name'>"{sp.label === '' || sp.label == null ? sp.name : sp.label}"</span>
			</div>
			<SlButton className='removeSelectedProperty' variant='neutral' onClick={onClose}>
				<SlIcon slot='prefix' name='x-lg' />
				Close
			</SlButton>
		</div>
	);
};

export default SelectedPropertyMessage;

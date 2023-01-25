import './PropertiesDialog.css';
import { SlIcon, SlDialog, SlIconButton, SlInput } from '@shoelace-style/shoelace/dist/react/index.js';

const PropertiesDialog = ({ isOpen, onClose, searchTerm, onSearch, properties, usedProperties, onAddProperty }) => {
	return (
		<SlDialog
			label='Add a property'
			className='PropertiesDialog'
			open={isOpen}
			onSlAfterHide={onClose}
			style={{ '--width': '700px' }}
		>
			<SlInput type='search' clearable placeholder='search' value={searchTerm} onSlInput={onSearch}>
				<SlIcon name='search' slot='prefix'></SlIcon>
			</SlInput>
			{properties.map((p) => {
				let toString = p.label ? p.label : p.name;
				if (
					toString.includes(searchTerm) ||
					toString.includes(searchTerm.charAt(0).toUpperCase() + searchTerm.slice(1)) ||
					toString.includes(searchTerm.toUpperCase) ||
					toString.includes(searchTerm.toLowerCase)
				) {
					return (
						<div
							key={p.name}
							className={`property${
								usedProperties.find((up) => up.name === p.name) != null ? ' used' : ''
							}`}
						>
							<div>{toString}</div>
							<SlIconButton name='plus-circle' label='Add property' onClick={() => onAddProperty(p)} />
						</div>
					);
				}
				return '';
			})}
		</SlDialog>
	);
};

export default PropertiesDialog;

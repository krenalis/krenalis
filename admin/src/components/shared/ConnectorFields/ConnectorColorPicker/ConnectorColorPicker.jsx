import { useState, useEffect } from 'react';
import './ConnectorColorPicker.css';
import { SlColorPicker } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorColorPicker = ({ name, label, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onColorPickerChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connectorColorPicker'>
			<SlColorPicker value={value} name={name} label={label} onSlChange={onColorPickerChange} />
			<div className='label'>{label}</div>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorColorPicker;

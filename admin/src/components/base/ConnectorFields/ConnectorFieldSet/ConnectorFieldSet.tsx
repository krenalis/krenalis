import React, { useState, useEffect, ReactNode } from 'react';
import { FieldSetContext } from '../../../../context/FieldSetContext';
import ConnectorField from '../ConnectorField';
import './ConnectorFieldSet.css';
import ConnectorFieldInterface from '../../../../lib/api/types/ui';

interface ConnectorFieldSetProps {
	name: string;
	fields: ConnectorFieldInterface[];
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorFieldSet = ({ name, fields, val, onChange }: ConnectorFieldSetProps) => {
	const [settings, setSettings] = useState(val);

	useEffect(() => {
		setSettings(val);
	}, [val]);

	const onFieldChange = (fieldName: string, value: any) => {
		setSettings((prevSettings: Record<string, any>) => {
			let vals: Record<string, any>;
			if (prevSettings == null) {
				vals = { [fieldName]: value };
			} else {
				vals = { ...prevSettings, [fieldName]: value };
			}
			onChange(name, vals);
			return vals;
		});
	};

	const fieldSet: ReactNode[] = [];
	for (const [i, f] of fields.entries()) fieldSet.push(<ConnectorField key={i} field={f} />);

	return (
		<div className='connector-fieldsets'>
			<FieldSetContext.Provider value={{ settings: settings, onChange: onFieldChange }}>
				{fieldSet}
			</FieldSetContext.Provider>
		</div>
	);
};

export default ConnectorFieldSet;

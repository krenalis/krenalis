import React, { useState, useEffect, ReactNode } from 'react';
import { FieldSetContext } from '../../../../context/FieldSetContext';
import ConnectorField from '../ConnectorField';
import './ConnectorFieldSet.css';
import ConnectorFieldInterface from '../../../../types/external/ui';

interface ConnectorFieldSetProps {
	name: string;
	fields: ConnectorFieldInterface[];
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorFieldSet = ({ name, fields, val, onChange }: ConnectorFieldSetProps) => {
	const [values, setValues] = useState(val);

	useEffect(() => {
		setValues(val);
	}, [val]);

	const onFieldChange = (fieldName: string, value: any) => {
		setValues((prevValues: Record<string, any>) => {
			let vals: Record<string, any>;
			if (prevValues == null) {
				vals = { [fieldName]: value };
			} else {
				vals = { ...prevValues, [fieldName]: value };
			}
			onChange(name, vals);
			return vals;
		});
	};

	const fieldSet: ReactNode[] = [];
	for (const [i, f] of fields.entries()) fieldSet.push(<ConnectorField key={i} field={f} />);

	return (
		<div className='connectorFieldSet'>
			<FieldSetContext.Provider value={{ values: values, onChange: onFieldChange }}>
				{fieldSet}
			</FieldSetContext.Provider>
		</div>
	);
};

export default ConnectorFieldSet;

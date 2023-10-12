import React, { useState } from 'react';
import './ConnectorAlternativeFieldSets.css';
import ConnectorFieldSet from '../ConnectorFieldSet/ConnectorFieldSet';
import ConnectorField from '../../../../types/external/ui';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';

interface FieldSet {
	Name: string;
	Label: string;
	Fields: ConnectorField[];
}

interface ConnectorAlternativeFieldSetsProps {
	label: string;
	helpText: string;
	sets: FieldSet[];
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorAlternativeFieldSets = ({
	label,
	helpText,
	sets,
	val,
	onChange,
}: ConnectorAlternativeFieldSetsProps) => {
	let initialSet: string = '';
	for (const s of sets) {
		if (val[s.Name] != null) {
			initialSet = s.Name;
			break;
		}
	}

	const [selected, setSelected] = useState(initialSet === '' ? sets[0].Name : initialSet);

	const onSelectChange = (e) => {
		// TODO: get from the server the information required to identify any
		// inputs shared between different sets, then implement the code that
		// automatically fill in those inputs when the user changes the set.
		onChange(selected, null);
		setSelected(e.currentTarget.value);
	};

	const selectedSet = sets.find((set) => set.Name === selected);
	const fieldSet = (
		<ConnectorFieldSet
			name={selectedSet!.Name}
			fields={selectedSet!.Fields}
			val={val[selectedSet!.Name]}
			onChange={onChange}
		></ConnectorFieldSet>
	);

	return (
		<div className='connectorAlternativeFieldSets'>
			<div className='label'>{label}</div>
			<SlSelect value={selected} onSlChange={onSelectChange}>
				{sets.map((s) => (
					<SlOption value={s.Name}>{s.Label}</SlOption>
				))}
			</SlSelect>
			{fieldSet}
			<div className='helpText'>{helpText}</div>
		</div>
	);
};

export default ConnectorAlternativeFieldSets;

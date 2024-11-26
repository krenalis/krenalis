import React, { useState } from 'react';
import './ConnectorAlternativeFieldSets.css';
import ConnectorFieldSet from '../ConnectorFieldSet/ConnectorFieldSet';
import ConnectorField from '../../../../lib/api/types/ui';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';

interface FieldSet {
	name: string;
	label: string;
	fields: ConnectorField[];
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
		if (val[s.name] != null) {
			initialSet = s.name;
			break;
		}
	}

	const [selected, setSelected] = useState(initialSet === '' ? sets[0].name : initialSet);

	const onSelectChange = (e) => {
		// TODO: get from the server the information required to identify any
		// inputs shared between different sets, then implement the code that
		// automatically fill in those inputs when the user changes the set.
		onChange(selected, null);
		setSelected(e.currentTarget.value);
	};

	const selectedSet = sets.find((set) => set.name === selected);
	const fieldSet = (
		<ConnectorFieldSet
			name={selectedSet!.name}
			fields={selectedSet!.fields}
			val={val[selectedSet!.name]}
			onChange={onChange}
		></ConnectorFieldSet>
	);

	return (
		<div className='connector-alternative-fieldsets'>
			<div className='connector-alternative-fieldsets__label'>{label}</div>
			<SlSelect value={selected} onSlChange={onSelectChange}>
				{sets.map((s) => (
					<SlOption key={s.name} value={s.name}>
						{s.label}
					</SlOption>
				))}
			</SlSelect>
			{fieldSet}
			<div className='connector-alternative-fieldsets__help-text'>{helpText}</div>
		</div>
	);
};

export default ConnectorAlternativeFieldSets;

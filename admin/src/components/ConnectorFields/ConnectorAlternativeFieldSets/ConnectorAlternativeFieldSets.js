import { useState } from 'react';
import './ConnectorAlternativeFieldSets.css';
import ConnectorFieldSet from '../ConnectorFieldSet/ConnectorFieldSet';
import { SlSelect, SlOption } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorAlternativeFieldSets = ({ label, helpText, sets, val, onChange }) => {
	let initialSet;
	for (let s of sets) {
		if (val[s.Name] != null) {
			initialSet = s.Name;
			break;
		}
	}

	const [selected, setSelected] = useState(initialSet == null ? sets[0].Name : initialSet);

	const onSelectChange = (e) => {
		// TODO: get from the server the informations required to identify any
		// inputs shared between different sets, then implement the code that
		// automatically fill in those inputs when the user changes the set.
		onChange(selected, null);
		setSelected(e.currentTarget.value);
	};

	let selectedSet = sets.find((set) => set.Name === selected);
	let fieldSet = (
		<ConnectorFieldSet
			name={selectedSet.Name}
			fields={selectedSet.Fields}
			val={val[selectedSet.Name]}
			onChange={onChange}
		></ConnectorFieldSet>
	);

	return (
		<div className='ConnectorAlternativeFieldSets'>
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

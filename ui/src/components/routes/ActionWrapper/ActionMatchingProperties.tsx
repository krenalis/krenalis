import React, { useRef, useContext, useMemo } from 'react';
import Section from '../../shared/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../shared/ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import ActionContext from '../../../context/ActionContext';
import { flattenSchema } from '../../../lib/helpers/transformedAction';

const ActionMatchingProperties = () => {
	const internalMatchingPropertyListRef = useRef(null);
	const externalMatchingPropertyListRef = useRef(null);

	const { connection, action, setAction, actionType } = useContext(ActionContext);

	const flatInputMatchingSchema = useMemo(() => flattenSchema(actionType.InputMatchingSchema), [actionType]);
	const flatOutputMatchingSchema = useMemo(() => flattenSchema(actionType.OutputMatchingSchema), [actionType]);

	const onUpdateMatchingProperties = (e) => {
		const a = { ...action };
		const type = e.target.dataset.type;
		const value = e.target.value;
		if (type === 'Internal') {
			const property = flatInputMatchingSchema[value];
			a.MatchingProperties!.Internal = property ? value : '';
		} else {
			const property = flatOutputMatchingSchema[value];
			a.MatchingProperties!.External = property ? property.full : null;
		}
		setAction(a);
	};

	const onSelectMatchingProperties = (input, value) => {
		const a = { ...action };
		const type = input.dataset.type;
		if (type === 'Internal') {
			const property = flatInputMatchingSchema[value];
			a.MatchingProperties!.Internal = property ? value : '';
		} else {
			const property = flatOutputMatchingSchema[value];
			a.MatchingProperties!.External = property ? property.full : null;
		}
		setAction(a);
	};

	return (
		<Section
			title={`Matching properties`}
			description='The properties used to identify and match the resources'
			padded={true}
		>
			<div className='matchingProperties'>
				<ComboBoxInput
					comboBoxListRef={internalMatchingPropertyListRef}
					onInput={onUpdateMatchingProperties}
					value={action.MatchingProperties!.Internal}
					label='Golden record property'
					data-type='Internal'
					className='inputProperty'
				></ComboBoxInput>
				<ComboBoxList
					ref={internalMatchingPropertyListRef}
					items={getSchemaComboboxItems(actionType.InputMatchingSchema)}
					onSelect={onSelectMatchingProperties}
				/>
				<div className='equal'>=</div>
				<ComboBoxInput
					comboBoxListRef={externalMatchingPropertyListRef}
					onInput={onUpdateMatchingProperties}
					label={`${connection.name}'s property`}
					value={action.MatchingProperties!.External ? action.MatchingProperties!.External.name : ''}
					data-type='External'
				></ComboBoxInput>
				<ComboBoxList
					ref={externalMatchingPropertyListRef}
					items={getSchemaComboboxItems(actionType.OutputMatchingSchema)}
					onSelect={onSelectMatchingProperties}
				/>
			</div>
		</Section>
	);
};

export default ActionMatchingProperties;

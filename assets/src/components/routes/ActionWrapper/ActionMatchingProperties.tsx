import React, { useRef, useContext, useMemo } from 'react';
import Section from '../../shared/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../shared/ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import ActionContext from '../../../context/ActionContext';
import { flattenSchema } from '../../../lib/helpers/transformedAction';
import { checkIfPropertyExists } from './Action.helpers';

const ActionMatchingProperties = () => {
	const { connection, action, setAction, actionType } = useContext(ActionContext);

	const internalMatchingPropertyListRef = useRef(null);
	const externalMatchingPropertyListRef = useRef(null);

	const flatInputMatchingSchema = useMemo(() => flattenSchema(actionType.InputMatchingSchema), [actionType]);
	const flatOutputMatchingSchema = useMemo(() => flattenSchema(actionType.OutputMatchingSchema), [actionType]);

	const internalPropertyError = useMemo<string>(() => {
		return checkIfPropertyExists(action.MatchingProperties.Internal, flatInputMatchingSchema);
	}, [action]);

	const externalPropertyError = useMemo<string>(() => {
		return checkIfPropertyExists(action.MatchingProperties.External, flatOutputMatchingSchema);
	}, [action]);

	const onUpdateMatchingProperties = (e) => {
		const a = { ...action };
		const type = e.target.dataset.type;
		const value = e.target.value;
		if (type === 'Internal') {
			a.MatchingProperties!.Internal = value;
		} else {
			a.MatchingProperties!.External = value;
		}
		setAction(a);
	};

	const onSelectMatchingProperties = (input, value) => {
		const a = { ...action };
		const type = input.dataset.type;
		if (type === 'Internal') {
			a.MatchingProperties!.Internal = value;
		} else {
			a.MatchingProperties!.External = value;
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
					caret={true}
					error={internalPropertyError}
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
					value={action.MatchingProperties!.External}
					data-type='External'
					caret={true}
					error={externalPropertyError}
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

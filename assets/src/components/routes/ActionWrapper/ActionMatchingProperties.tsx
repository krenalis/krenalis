import React, { useRef, useContext, useMemo } from 'react';
import Section from '../../base/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../base/ComboBox/ComboBox';
import { getExternalMatchingPropertiesItems, getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import ActionContext from '../../../context/ActionContext';
import { flattenSchema } from '../../../lib/core/action';
import { checkIfPropertyExists } from './Action.helpers';
import { ComboboxItem } from '../../base/ComboBox/ComboBox.types';

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

	const externalMatchingPropertiesItems = useMemo<ComboboxItem[]>(() => {
		if (action.ExportMode === 'CreateOnly' || action.ExportMode === 'CreateOrUpdate') {
			return getExternalMatchingPropertiesItems(actionType.OutputMatchingSchema, actionType.OutputSchema);
		}
		return getSchemaComboboxItems(actionType.OutputMatchingSchema);
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
			<div className='action__matching-properties'>
				<ComboBoxInput
					comboBoxListRef={internalMatchingPropertyListRef}
					onInput={onUpdateMatchingProperties}
					value={action.MatchingProperties!.Internal}
					label={`User's schema property`}
					data-type='Internal'
					className='action__transformation-input-property'
					caret={true}
					error={internalPropertyError}
				></ComboBoxInput>
				<ComboBoxList
					ref={internalMatchingPropertyListRef}
					items={getSchemaComboboxItems(actionType.InputMatchingSchema)}
					onSelect={onSelectMatchingProperties}
				/>
				<div className='action__matching-properties-equal'>=</div>
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
					items={externalMatchingPropertiesItems}
					onSelect={onSelectMatchingProperties}
				/>
			</div>
		</Section>
	);
};

export default ActionMatchingProperties;

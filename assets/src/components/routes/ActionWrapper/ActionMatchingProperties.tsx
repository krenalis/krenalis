import React, { useRef, useContext, useMemo } from 'react';
import Section from '../../base/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../base/ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import ActionContext from '../../../context/ActionContext';
import { flattenSchema, TransformedMapping } from '../../../lib/core/action';
import { checkIfPropertyExists } from './Action.helpers';

const ActionMatchingProperties = () => {
	const { connection, action, setAction, actionType, mode } = useContext(ActionContext);

	const internalMatchingPropertyListRef = useRef(null);
	const externalMatchingPropertyListRef = useRef(null);

	const flatInputMatchingSchema = useMemo(() => flattenSchema(actionType.InputMatchingSchema), [actionType]);

	const { externalMatchingPropertiesItems, flatExternalMatchingPropertiesSchema } = useMemo(() => {
		if (action.ExportMode === 'CreateOnly' || action.ExportMode === 'CreateOrUpdate') {
			const flatExternalMatchingSchema = flattenSchema(actionType.OutputMatchingSchema);
			const flatOutputSchema = flattenSchema(actionType.OutputSchema);
			const filteredSchema: TransformedMapping = {};
			for (const [k, v] of Object.entries(flatExternalMatchingSchema)) {
				const isInOutputSchema = flatOutputSchema[k] && flatOutputSchema[k].type === v.type;
				if (isInOutputSchema) {
					filteredSchema[k] = v;
				}
			}
			return {
				externalMatchingPropertiesItems: getSchemaComboboxItems(filteredSchema),
				flatExternalMatchingPropertiesSchema: filteredSchema,
			};
		}
		return {
			externalMatchingPropertiesItems: getSchemaComboboxItems(actionType.OutputMatchingSchema),
			flatExternalMatchingPropertiesSchema: flattenSchema(actionType.OutputMatchingSchema),
		};
	}, [action]);

	const internalPropertyError = useMemo<string>(() => {
		return checkIfPropertyExists(action.MatchingProperties.Internal, flatInputMatchingSchema);
	}, [action]);

	const externalPropertyError = useMemo<string>(() => {
		return checkIfPropertyExists(action.MatchingProperties.External, flatExternalMatchingPropertiesSchema);
	}, [action]);

	const onUpdateMatchingProperties = (e) => {
		const a = { ...action };
		const type = e.target.dataset.type;
		const value = e.target.value;
		if (type === 'Internal') {
			a.MatchingProperties!.Internal = value;
		} else {
			a.MatchingProperties!.External = value;
			// The external matching properties cannot be transformed.
			if (mode === 'mappings') {
				a.Transformation.Mapping[value].value = '';
			}
			// TODO(@Andrea): remove the property from the transformation even
			// in case of transformation function (this must be addressed after
			// fixing issue https://github.com/meergo/meergo/issues/507)
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
			// The external matching properties cannot be transformed.
			if (mode === 'mappings') {
				a.Transformation.Mapping[value].value = '';
			}
			// TODO(@Andrea): remove the property from the transformation even
			// in case of transformation function (this must be addressed after
			// fixing issue https://github.com/meergo/meergo/issues/507)
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

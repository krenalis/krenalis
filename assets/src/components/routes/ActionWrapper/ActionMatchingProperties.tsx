import React, { useContext, useMemo } from 'react';
import Section from '../../base/Section/Section';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboboxItems';
import ActionContext from '../../../context/ActionContext';
import { flattenSchema, TransformedMapping } from '../../../lib/core/action';
import { checkIfPropertyExists } from './Action.helpers';
import { Combobox } from '../../base/Combobox/Combobox';

const ActionMatchingProperties = () => {
	const { connection, action, setAction, actionType, mode } = useContext(ActionContext);

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

	const onUpdateMatchingProperties = (name: string, value: string) => {
		const a = { ...action };
		if (name === 'internal') {
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

	const onSelectMatchingProperties = (name: string, value: string) => {
		const a = { ...action };
		if (name === 'internal') {
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
				<Combobox
					onInput={onUpdateMatchingProperties}
					initialValue={action.MatchingProperties!.Internal}
					label={`User's schema property`}
					name='internal'
					className='action__transformation-input-property'
					items={getSchemaComboboxItems(actionType.InputMatchingSchema)}
					onSelect={onSelectMatchingProperties}
					isExpression={false}
					caret={true}
					error={internalPropertyError}
				/>
				<div className='action__matching-properties-equal'>=</div>
				<Combobox
					onInput={onUpdateMatchingProperties}
					label={`${connection.name}'s property`}
					initialValue={action.MatchingProperties!.External}
					name='external'
					isExpression={false}
					items={externalMatchingPropertiesItems}
					onSelect={onSelectMatchingProperties}
					caret={true}
					error={externalPropertyError}
				/>
			</div>
		</Section>
	);
};

export default ActionMatchingProperties;

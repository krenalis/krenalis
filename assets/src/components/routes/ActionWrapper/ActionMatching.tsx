import React, { useContext, useMemo } from 'react';
import Section from '../../base/Section/Section';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboboxItems';
import ActionContext from '../../../context/ActionContext';
import { flattenSchema, outPathsTypesAreEqual, TransformedMapping, validateMatching } from '../../../lib/core/action';
import { checkIfPropertyExists } from './Action.helpers';
import { Combobox } from '../../base/Combobox/Combobox';

const ActionMatching = () => {
	const { connection, action, setAction, actionType, transformationType, selectedOutPaths, setSelectedOutPaths } =
		useContext(ActionContext);

	const flatInMatchingSchema = useMemo(() => flattenSchema(actionType.inputMatchingSchema), [actionType]);

	const { outMatchingItems, flatOutMatchingSchema } = useMemo(() => {
		const flatExternalMatchingSchema = flattenSchema(actionType.outputMatchingSchema);
		const flatOutputSchema = flattenSchema(actionType.outputSchema);

		let filteredSchema: TransformedMapping = {};
		if (action.exportMode === 'CreateOnly' || action.exportMode === 'CreateOrUpdate') {
			for (const [k, v] of Object.entries(flatExternalMatchingSchema)) {
				const a = v.full;
				const b = flatOutputSchema[k]?.full;
				if (b != null && outPathsTypesAreEqual(a.type, b.type) && a.nullable === b.nullable) {
					filteredSchema[k] = v;
				}
			}
		} else {
			filteredSchema = flatExternalMatchingSchema;
		}

		return {
			outMatchingItems: getSchemaComboboxItems(filteredSchema),
			flatOutMatchingSchema: filteredSchema,
		};
	}, [action]);

	const inMatchingError = useMemo<string>(() => {
		const err = checkIfPropertyExists(action.matching.in, flatInMatchingSchema);
		if (err !== '') {
			return err;
		}
		const fullIn = flatInMatchingSchema[action.matching.in]?.full;
		const fullOut = flatOutMatchingSchema[action.matching.out]?.full;
		if (fullIn == null || fullOut == null) {
			return;
		}
		try {
			validateMatching(fullIn, fullOut);
		} catch (err) {
			return err.message;
		}
	}, [action]);

	const outMatchingError = useMemo<string>(() => {
		return checkIfPropertyExists(action.matching.out, flatOutMatchingSchema);
	}, [action]);

	const onUpdateMatching = (name: string, v: string) => {
		const a = { ...action };
		if (name === 'in') {
			a.matching!.in = v;
		} else {
			a.matching!.out = v;
			// The out matching properties cannot be transformed.
			if (transformationType === 'mappings') {
				if (a.transformation.mapping[v] != null) {
					a.transformation.mapping[v].value = '';
				}
			} else {
				const s = selectedOutPaths.filter((p) => p !== v && !p.startsWith(`${v}.`));
				setSelectedOutPaths(s);
			}
		}
		setAction(a);
	};

	const onSelectMatching = (name: string, v: string) => {
		const a = { ...action };
		if (name === 'in') {
			a.matching!.in = v;
		} else {
			a.matching!.out = v;
			// The out matching properties cannot be transformed.
			if (transformationType === 'mappings') {
				if (a.transformation.mapping[v] != null) {
					a.transformation.mapping[v].value = '';
				}
			} else {
				const s = selectedOutPaths.filter((p) => p !== v && !p.startsWith(`${v}.`));
				setSelectedOutPaths(s);
			}
		}
		setAction(a);
	};

	return (
		<Section
			title={`Matching properties`}
			description='The properties used to identify and match the resources'
			padded={true}
			annotated={true}
		>
			<div className='action__matching-properties'>
				<Combobox
					onInput={onUpdateMatching}
					initialValue={action.matching!.in}
					label={`User's schema property`}
					name='in'
					className='action__transformation-input-property'
					items={getSchemaComboboxItems(flatInMatchingSchema)}
					onSelect={onSelectMatching}
					isExpression={false}
					caret={true}
					error={inMatchingError}
				/>
				<div className='action__matching-properties-equal'>=</div>
				<Combobox
					onInput={onUpdateMatching}
					label={`${connection.name}'s property`}
					initialValue={action.matching!.out}
					name='out'
					isExpression={false}
					items={outMatchingItems}
					onSelect={onSelectMatching}
					caret={true}
					error={outMatchingError}
				/>
			</div>
		</Section>
	);
};

export default ActionMatching;

import React, { forwardRef, useContext, useMemo } from 'react';
import { getMatchingComboboxItems } from '../../helpers/getSchemaComboboxItems';
import ActionContext from '../../../context/ActionContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { flattenSchema, propertyTypesAreEqual, TransformedMapping, validateMatching } from '../../../lib/core/action';
import { checkIfPropertyExists } from './Action.helpers';
import { Combobox } from '../../base/Combobox/Combobox';

const ActionMatching = forwardRef<any>((_, ref) => {
	const { connection, action, setAction, actionType, showEmptyMatchingError, transformationType, selectedOutPaths } =
		useContext(ActionContext);

	const flatInMatchingSchema = useMemo(() => flattenSchema(actionType.inputMatchingSchema), [actionType]);
	const { outMatchingItems, flatOutMatchingSchema } = useMemo(() => {
		const flatSourceSchema = flattenSchema(actionType.outputMatchingSchema);
		const flatDestinationSchema = flattenSchema(actionType.outputSchema);

		let filteredSchema: TransformedMapping = {};
		if (action.exportMode === 'CreateOnly' || action.exportMode === 'CreateOrUpdate') {
			for (const [k, v] of Object.entries(flatSourceSchema)) {
				const a = v.full;
				const b = flatDestinationSchema[k]?.full;
				if (b == null || (b != null && propertyTypesAreEqual(a.type, b.type) && a.nullable === b.nullable)) {
					filteredSchema[k] = v;
				}
			}
		} else {
			filteredSchema = flatSourceSchema;
		}

		return {
			outMatchingItems: getMatchingComboboxItems(filteredSchema),
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
		const err = checkIfPropertyExists(action.matching.out, flatOutMatchingSchema);
		if (err != '') {
			return err;
		}
		if (action.matching.out === '') {
			return '';
		}
		let isTransformed = false;
		if (transformationType === 'mappings') {
			const m = action.transformation.mapping[action.matching.out];
			const isInMapping = m != null;
			if (!isInMapping) {
				return '';
			}
			isTransformed = m.value !== '';
		} else {
			isTransformed = selectedOutPaths.includes(action.matching.out);
		}
		if (isTransformed) {
			if (transformationType === 'function') {
				return 'Please ensure that this property is not used in the transformation function, as it is currently selected in the output schema of the transformation';
			} else {
				return 'Please ensure that no value is mapped to this property in the transformation below';
			}
		}
	}, [action, selectedOutPaths, transformationType]);

	const onUpdateMatching = (side: 'in' | 'out', v: string) => {
		const a = { ...action };
		a.matching![side] = v;

		// automatically clear the mapping if the same value of the in
		// matching is already mapped on the out matching.
		if (side === 'out' && v !== '' && a.matching['in'] !== '' && transformationType === 'mappings') {
			const isInMapping = a.transformation.mapping[v] != null;
			if (isInMapping) {
				const isAlreadyMapped = a.transformation.mapping[v].value === a.matching['in'];
				if (isAlreadyMapped) {
					a.transformation.mapping[v].value = '';
				}
			}
		}

		setAction(a);
	};

	const onSelectMatching = (side: 'in' | 'out', v: string) => {
		const a = { ...action };
		a.matching![side] = v;

		// automatically clear the mapping if the same value of the in
		// matching is already mapped on the out matching.
		if (side === 'out' && v !== '' && a.matching['in'] !== '' && transformationType === 'mappings') {
			const isInMapping = a.transformation.mapping[v] != null;
			if (isInMapping) {
				const isAlreadyMapped = a.transformation.mapping[v].value === a.matching['in'];
				if (isAlreadyMapped) {
					a.transformation.mapping[v].value = '';
				}
			}
		}

		setAction(a);
	};

	return (
		<div ref={ref} className='action__matching-section'>
			<div className='action__matching-wrapper'>
				<div className='action__matching-title'>What properties define a match between users?</div>
				<div className='action__matching-properties'>
					<Combobox
						onInput={onUpdateMatching}
						value={action.matching!.in}
						label={`User's schema property`}
						name='in'
						className='action__transformation-input-property'
						items={getMatchingComboboxItems(flatInMatchingSchema)}
						onSelect={onSelectMatching}
						isExpression={false}
						caret={true}
						error={inMatchingError}
					/>
					<div className='action__matching-properties-equal'>=</div>
					<Combobox
						onInput={onUpdateMatching}
						label={`${connection.connector.label}'s property`}
						value={action.matching!.out}
						name='out'
						isExpression={false}
						items={outMatchingItems}
						onSelect={onSelectMatching}
						caret={true}
						error={outMatchingError}
					/>
				</div>
				{showEmptyMatchingError && (action.matching.in === '' || action.matching.out === '') && (
					<div className='action__matching-empty-error'>
						<SlIcon name='exclamation-circle' slot='prefix' />
						Matching properties cannot be empty
					</div>
				)}
			</div>
		</div>
	);
});

export default ActionMatching;

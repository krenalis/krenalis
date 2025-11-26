import React, { forwardRef, useContext, useMemo } from 'react';
import { getMatchingComboboxItems } from '../../helpers/getSchemaComboboxItems';
import PipelineContext from '../../../context/PipelineContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { flattenSchema, propertyTypesAreEqual, TransformedMapping, validateMatching } from '../../../lib/core/pipeline';
import { checkIfPropertyExists } from './Pipeline.helpers';
import { Combobox } from '../../base/Combobox/Combobox';

const PipelineMatching = forwardRef<any>((_, ref) => {
	const {
		connection,
		pipeline,
		setPipeline,
		pipelineType,
		showEmptyMatchingError,
		transformationType,
		selectedOutPaths,
	} = useContext(PipelineContext);

	const flatInMatchingSchema = useMemo(() => flattenSchema(pipelineType.inputMatchingSchema), [pipelineType]);
	const { outMatchingItems, flatOutMatchingSchema } = useMemo(() => {
		const flatSourceSchema = flattenSchema(pipelineType.outputMatchingSchema);
		const flatDestinationSchema = flattenSchema(pipelineType.outputSchema);

		let filteredSchema: TransformedMapping = {};
		if (pipeline.exportMode === 'CreateOnly' || pipeline.exportMode === 'CreateOrUpdate') {
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
	}, [pipeline]);

	const inMatchingError = useMemo<string>(() => {
		const err = checkIfPropertyExists(pipeline.matching.in, flatInMatchingSchema);
		if (err !== '') {
			return err;
		}
		const fullIn = flatInMatchingSchema[pipeline.matching.in]?.full;
		const fullOut = flatOutMatchingSchema[pipeline.matching.out]?.full;
		if (fullIn == null || fullOut == null) {
			return;
		}
		try {
			validateMatching(fullIn, fullOut);
		} catch (err) {
			return err.message;
		}
	}, [pipeline]);

	const outMatchingError = useMemo<string>(() => {
		const err = checkIfPropertyExists(pipeline.matching.out, flatOutMatchingSchema);
		if (err != '') {
			return err;
		}
		if (pipeline.matching.out === '') {
			return '';
		}
		let isTransformed = false;
		if (transformationType === 'mappings') {
			const m = pipeline.transformation.mapping[pipeline.matching.out];
			const isInMapping = m != null;
			if (!isInMapping) {
				return '';
			}
			isTransformed = m.value !== '';
		} else {
			isTransformed = selectedOutPaths.includes(pipeline.matching.out);
		}
		if (isTransformed) {
			if (transformationType === 'function') {
				return 'Please ensure that this property is not used in the transformation function, as it is currently selected in the output schema of the transformation';
			} else {
				return 'Please ensure that no value is mapped to this property in the transformation below';
			}
		}
	}, [pipeline, selectedOutPaths, transformationType]);

	const onUpdateMatching = (side: 'in' | 'out', v: string) => {
		const p = { ...pipeline };
		p.matching![side] = v;

		// automatically clear the mapping if the same value of the in
		// matching is already mapped on the out matching.
		if (side === 'out' && v !== '' && p.matching['in'] !== '' && transformationType === 'mappings') {
			const isInMapping = p.transformation.mapping[v] != null;
			if (isInMapping) {
				const isAlreadyMapped = p.transformation.mapping[v].value === p.matching['in'];
				if (isAlreadyMapped) {
					p.transformation.mapping[v].value = '';
				}
			}
		}

		setPipeline(p);
	};

	const onSelectMatching = (side: 'in' | 'out', v: string) => {
		const p = { ...pipeline };
		p.matching![side] = v;

		// automatically clear the mapping if the same value of the in
		// matching is already mapped on the out matching.
		if (side === 'out' && v !== '' && p.matching['in'] !== '' && transformationType === 'mappings') {
			const isInMapping = p.transformation.mapping[v] != null;
			if (isInMapping) {
				const isAlreadyMapped = p.transformation.mapping[v].value === p.matching['in'];
				if (isAlreadyMapped) {
					p.transformation.mapping[v].value = '';
				}
			}
		}

		setPipeline(p);
	};

	return (
		<div ref={ref} className='pipeline__matching-section'>
			<div className='pipeline__matching-wrapper'>
				<div className='pipeline__matching-title'>What properties define a match between users?</div>
				<div className='pipeline__matching-properties'>
					<Combobox
						onInput={onUpdateMatching}
						value={pipeline.matching!.in}
						label={`User's schema property`}
						name='in'
						className='pipeline__transformation-input-property'
						items={getMatchingComboboxItems(flatInMatchingSchema)}
						onSelect={onSelectMatching}
						isExpression={false}
						caret={true}
						error={inMatchingError}
					/>
					<div className='pipeline__matching-properties-equal'>=</div>
					<Combobox
						onInput={onUpdateMatching}
						label={`${connection.connector.label}'s property`}
						value={pipeline.matching!.out}
						name='out'
						isExpression={false}
						items={outMatchingItems}
						onSelect={onSelectMatching}
						caret={true}
						error={outMatchingError}
					/>
				</div>
				{showEmptyMatchingError && (pipeline.matching.in === '' || pipeline.matching.out === '') && (
					<div className='pipeline__matching-empty-error'>
						<SlIcon name='exclamation-circle' slot='prefix' />
						Matching properties cannot be empty
					</div>
				)}
			</div>
		</div>
	);
});

export default PipelineMatching;

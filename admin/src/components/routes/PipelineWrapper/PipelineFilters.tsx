import React, { useContext, ReactNode, useMemo, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import { getFilterPropertyComboboxItems } from '../../helpers/getSchemaComboboxItems';
import PipelineContext from '../../../context/PipelineContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlButtonGroup from '@shoelace-style/shoelace/dist/react/button-group/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Combobox } from '../../base/Combobox/Combobox';
import {
	FILTER_OPERATORS,
	flattenSchema,
	getCompatibleFilterOperators,
	isBetweenOperator,
	isOneOfOperator,
	isUnaryOperator,
	splitPropertyAndPath,
	TransformedPipeline,
	TransformedProperty,
} from '../../../lib/core/pipeline';
import { FilterLogical, FilterOperator } from '../../../lib/api/types/pipeline';
import { checkIfPropertyExists } from './Pipeline.helpers';
import { StringType } from '../../../lib/api/types/types';

const PipelineFilters = forwardRef<any>((_, ref) => {
	const { pipeline, setPipeline, pipelineType, connection, isTransformationDisabled, isImport } =
		useContext(PipelineContext);
	const targetTerm =
		pipelineType.target === 'Event'
			? 'events'
			: connection.connector.terms.users?.trim()
				? connection.connector.terms.users.toLowerCase()
				: 'contacts';
	const actionVerb = isImport ? 'import' : pipelineType.target === 'Event' ? 'send' : 'export';

	const flatInputSchema = useMemo(() => {
		return flattenSchema(pipelineType.inputSchema);
	}, [pipelineType.inputSchema]);

	const findPropertyInSchema = (propertyName: string): TransformedProperty | undefined => {
		if (propertyName == null || propertyName === '') {
			return undefined;
		}
		return flatInputSchema[propertyName] ?? flatInputSchema[splitPropertyAndPath(propertyName, flatInputSchema)[0]];
	};

	const getPropertyValues = (property: TransformedProperty | undefined): string[] | null => {
		if (property == null || property.type !== 'string') {
			return null;
		}
		const stringType = property.full.type as StringType;
		return stringType.values ?? null;
	};

	const onAddCondition = () => {
		const p = structuredClone(pipeline);
		if (p.filter == null) {
			p.filter = { logical: 'and', conditions: [] };
		}
		p.filter.conditions = [...p.filter.conditions, { property: '', operator: '', values: [''] }];
		setPipeline(p);
	};

	const onRemoveCondition = (id: number) => {
		const p = structuredClone(pipeline);
		p.filter!.conditions.splice(id, 1);
		if (p.filter!.conditions.length === 0) {
			p.filter = null;
		}
		setPipeline(p);
	};

	const onInputPropertyFragment = (name: string, value: string) => {
		updatePropertyFragment(name, value);
	};

	const onSelectPropertyFragment = (name: string, value: string) => {
		const updated = updatePropertyFragment(name, value);

		const id = Number(name.split('-')[1]);
		const hasPath = 'path' in updated.filter!.conditions[id];
		const currentOperator = updated.filter!.conditions[id]['operator'];
		const currentOperatorIndex = FILTER_OPERATORS.findIndex((op) => op === currentOperator);
		const compatibleOperators = getCompatibleFilterOperators(
			flatInputSchema[value],
			hasPath,
			connection.role,
			pipeline.target,
		);
		const isCompatible = compatibleOperators.includes(currentOperatorIndex);
		const isJson = flatInputSchema[value]?.type === 'json';

		if (!isCompatible) {
			// Select the first compatible operator.
			const operator = FILTER_OPERATORS[compatibleOperators[0]];
			changeOperator(id, operator, updated);
			if (!isJson) {
				setTimeout(() => {
					const operatorSelect: any = document
						.querySelector(`[data-id=property-${id}]`)
						.closest('.pipeline__filters-condition')
						.querySelector('.pipeline__filters-operator');
					operatorSelect.show();
				}, 10);
			}
		}

		if (isJson) {
			setTimeout(() => {
				const pathInput: any = document
					.querySelector(`[data-id=property-${id}]`)
					.closest('.pipeline__filters-condition')
					.querySelector('.pipeline__filters-path');
				pathInput.select();
			}, 10);
		}
	};

	const updatePropertyFragment = (name: string, value: string): TransformedPipeline => {
		const p = structuredClone(pipeline);
		const id = Number(name.split('-')[1]);

		const condition = p.filter!.conditions[id];

		const previousPropertyName = condition['property'];
		const previousProperty = findPropertyInSchema(previousPropertyName);
		const previousPropertyValues = getPropertyValues(previousProperty);

		const [_, path] = splitPropertyAndPath(previousPropertyName, flatInputSchema);
		let newPropertyName = '';
		let hasPath = path !== '';
		if (hasPath && flatInputSchema[value]?.type === 'json') {
			newPropertyName = `${value}.${path}`;
		} else {
			newPropertyName = value;
		}

		const compatibleOperators = getCompatibleFilterOperators(
			flatInputSchema[newPropertyName],
			hasPath,
			connection.role,
			pipeline.target,
		);
		const currentOperator = condition['operator'];
		if (currentOperator != null && currentOperator !== '') {
			const index = FILTER_OPERATORS.indexOf(currentOperator);
			if (!compatibleOperators.includes(index)) {
				// The current operator is not compatible with the new property.
				// Reset the operator and the values.
				condition['operator'] = '';
				condition['values'] = [''];
			}
		}

		condition['property'] = newPropertyName;
		const newProperty = findPropertyInSchema(newPropertyName);
		const newPropertyValues = getPropertyValues(newProperty);
		const hasDifferentValues =
			previousPropertyName !== newPropertyName && (previousPropertyValues != null || newPropertyValues != null);
		if (hasDifferentValues) {
			// Reset the old values, because the new property has a different
			// set of enumerated values.
			const op = condition.operator;
			if (isBetweenOperator(op)) {
				condition.values = ['', ''];
			} else {
				condition.values = [''];
			}
		}

		setPipeline(p);
		return p;
	};

	const onInputPathFragment = (e) => {
		const p = structuredClone(pipeline);
		const id = Number(e.target.name.split('-')[1]);
		const propertyName = p.filter!.conditions[id]['property'];
		const [base, _] = splitPropertyAndPath(propertyName, flatInputSchema);
		let newPropertyName = '';
		const newPath = e.target.value;
		if (newPath !== '') {
			newPropertyName = `${base}.${newPath}`;
		} else {
			newPropertyName = base;
		}
		p.filter!.conditions[id]['property'] = newPropertyName;
		setPipeline(p);
	};

	const onChangeOperatorFragment = (e: any) => {
		const id = Number(e.target.name.split('-')[1]);
		const operator = FILTER_OPERATORS[e.target.value];
		changeOperator(id, operator);
	};

	const onOperatorSelectClose = (e: any) => {
		const operator = FILTER_OPERATORS[e.target.value];
		if (!isUnaryOperator(operator)) {
			// Focus the first value input.
			setTimeout(() => {
				const valueInput = e.target
					.closest('.pipeline__filters-condition')
					.querySelector('.pipeline__filters-value-input');
				const isSelect = valueInput.tagName === 'SL-SELECT';
				if (isSelect) {
					valueInput.show();
				} else {
					valueInput.focus();
				}
			}, 50);
		}
	};

	const changeOperator = (conditionID: number, operator: FilterOperator, updatedPipeline?: TransformedPipeline) => {
		const id = conditionID;
		let p: TransformedPipeline;
		if (updatedPipeline != null) {
			p = updatedPipeline;
		} else {
			p = structuredClone(pipeline);
		}
		p.filter!.conditions[id]['operator'] = operator;
		if (isUnaryOperator(operator)) {
			p.filter!.conditions[id]['values'] = null;
		} else {
			const isBetween = isBetweenOperator(operator);
			const isOneOf = isOneOfOperator(operator);

			let values: string[] | null;
			if (isBetween) {
				let v = ['', ''];
				if (p.filter!.conditions[id]['values'] != null) {
					v = p.filter!.conditions[id]['values'].slice(0, 2);
					if (v.length === 1) {
						v.push('');
					}
				}
				values = v;
			} else if (isOneOf) {
				let v = p.filter!.conditions[id]['values'];
				if (v == null) {
					v = [''];
				}
				values = v;
			} else {
				let v = '';
				if (p.filter!.conditions[id]['values'] != null) {
					v = p.filter!.conditions[id]['values'][0];
				}
				values = [v];
			}

			p.filter!.conditions[id]['values'] = values;
		}
		setPipeline(p);
	};

	const onChangeValueFragment = (name: string, value: string) => {
		const p = structuredClone(pipeline);
		const split = name.split('-');
		const id = Number(split[1]);
		const position = Number(split[2]);
		p.filter!.conditions[id]['values'][position] = value;
		setPipeline(p);
	};

	const onLogicalClick = (logical: FilterLogical) => {
		const p = structuredClone(pipeline);
		p.filter!.logical = logical;
		setPipeline(p);
	};

	const onAddValue = (index: number) => {
		const p = structuredClone(pipeline);
		const currentLength = p.filter!.conditions[index]['values'].length;
		p.filter!.conditions[index]['values'] = [...p.filter!.conditions[index]['values'], ''];
		setPipeline(p);
		setTimeout(() => {
			// focus the new input.
			const valueInputs: any = document
				.querySelector(`[data-id=property-${index}]`)
				.closest('.pipeline__filters-condition')
				.querySelectorAll('.pipeline__filters-value-input');
			const newValueInput = valueInputs[currentLength];
			const isSelect = newValueInput.tagName === 'SL-SELECT';
			if (isSelect) {
				newValueInput.show();
			} else {
				newValueInput.focus();
			}
		}, 50);
	};

	const onRemoveValue = (conditionIndex: number, valueIndex: number) => {
		const p = structuredClone(pipeline);
		const values = p.filter!.conditions[conditionIndex]['values'];
		const filtered = [];
		for (const [i, v] of values.entries()) {
			if (i !== valueIndex) {
				filtered.push(v);
			}
		}
		p.filter!.conditions[conditionIndex]['values'] = filtered;
		setPipeline(p);
	};

	const isFileStorageImport = connection.isFileStorage && connection.isSource;

	// For file storage imports, the filter section is displayed
	// together with the transformation section when the file's settings
	// are confirmed. It must be disabled when the settings are changed
	// and not yet re-confirmed.
	const isDisabled = isFileStorageImport && isTransformationDisabled;

	const conditions: ReactNode[] = [];
	if (pipeline.filter != null) {
		for (const [i, condition] of pipeline.filter.conditions.entries()) {
			const [base, path] = splitPropertyAndPath(condition.property, flatInputSchema);

			let property = flatInputSchema?.[base];
			const isUnary = isUnaryOperator(condition.operator);
			const isJSON = property?.type === 'json';
			const isBetween = isBetweenOperator(condition.operator);
			const isOneOf = isOneOfOperator(condition.operator);
			const isInvalidProperty = property == null;

			let logicalElement: ReactNode;
			let propertyInput: ReactNode;
			let pathInput: ReactNode;
			let operatorSelect: ReactNode;
			let valueElements: ReactNode[] = [];

			// Logical
			if (i === 0) {
				if (pipeline.filter.conditions.length > 1) {
					// Add a placeholder to maintain alignment.
					logicalElement = (
						<div className='pipeline__filters-logical pipeline__filters-logical--placeholder'></div>
					);
				}
			} else if (i === 1) {
				logicalElement = (
					<SlButtonGroup className='pipeline__filters-logical'>
						<SlButton
							size='small'
							variant={pipeline.filter!.logical === 'and' ? 'primary' : 'default'}
							onClick={() => onLogicalClick('and')}
							disabled={isDisabled}
						>
							and
						</SlButton>
						<SlButton
							size='small'
							variant={pipeline.filter!.logical === 'or' ? 'primary' : 'default'}
							onClick={() => onLogicalClick('or')}
							disabled={isDisabled}
						>
							or
						</SlButton>
					</SlButtonGroup>
				);
			} else if (i > 1) {
				logicalElement = (
					<div className='pipeline__filters-logical pipeline__filters-logical--text'>
						{pipeline.filter!.logical}
					</div>
				);
			}

			// Property
			const isEventBasedUserImport = connection.isEventBased && connection.isSource && pipeline.target === 'User';
			const isAppEventsExport =
				connection.isApplication && connection.isDestination && pipeline.target === 'Event';
			const isEventImport = connection.isSource && pipeline.target === 'Event';

			let propertiesToHide = [];
			if (isEventBasedUserImport || isAppEventsExport || isEventImport) {
				propertiesToHide = ['kpid'];
			}

			propertyInput = (
				<Combobox
					onInput={onInputPropertyFragment}
					onSelect={onSelectPropertyFragment}
					value={isJSON ? base : condition.property}
					className='pipeline__filters-property'
					size='small'
					name={`property-${i}`}
					items={getFilterPropertyComboboxItems(
						pipelineType.inputSchema,
						connection,
						pipeline.target,
						propertiesToHide,
					)}
					isExpression={false}
					disabled={isDisabled}
					placeholder={'Property'}
					caret={true}
					controlled={true}
					autoResize={true}
					error={
						condition.property !== '' &&
						checkIfPropertyExists(isJSON ? base : condition.property, flatInputSchema, propertiesToHide)
					}
				/>
			);

			if (isJSON) {
				pathInput = (
					<SlInput
						size='small'
						className='pipeline__filters-path'
						value={path}
						onSlInput={onInputPathFragment}
						name={`path-${i}`}
						disabled={isDisabled}
						placeholder='Path (optional)'
					/>
				);
			}

			// Operator
			operatorSelect = (
				<SlSelect
					size='small'
					name={`operator-${i}`}
					className='pipeline__filters-operator'
					value={String(FILTER_OPERATORS.findIndex((op) => op === condition.operator))}
					onSlChange={onChangeOperatorFragment}
					onSlHide={onOperatorSelectClose}
					placeholder='Operator'
					disabled={isInvalidProperty || isDisabled}
				>
					{property != null
						? getCompatibleFilterOperators(property, path !== '', connection.role, pipeline.target).map(
								(i) => (
									<SlOption key={i} value={String(i)}>
										{FILTER_OPERATORS[i]}
									</SlOption>
								),
							)
						: Object.keys(FILTER_OPERATORS).map((i) => (
								<SlOption key={i} value={String(i)}>
									{FILTER_OPERATORS[i]}
								</SlOption>
							))}
				</SlSelect>
			);

			// Values
			if (!isUnary) {
				const id = `value-${i}-0`;

				let propertyValues = [];
				if (!isInvalidProperty && property.type === 'string') {
					const typ = property.full.type as StringType;
					propertyValues = typ.values || [];
				}

				valueElements.push(
					<PipelineFilterValueControl
						key={id}
						name={id}
						value={condition.values != null ? condition.values[0] : ''}
						options={propertyValues}
						disabled={isInvalidProperty || isDisabled}
						onValueChange={onChangeValueFragment}
					/>,
				);

				if (isBetween) {
					valueElements.push(
						<span className='pipeline__filters-value-and' key='and'>
							and
						</span>,
					);
					const id = `value-${i}-1`;
					valueElements.push(
						<PipelineFilterValueControl
							key={id}
							name={id}
							value={condition.values != null ? (condition.values[1] ? condition.values[1] : '') : ''}
							options={propertyValues}
							disabled={isInvalidProperty || isDisabled}
							onValueChange={onChangeValueFragment}
						/>,
					);
				} else if (isOneOf) {
					const additionalValues = condition.values.slice(1);
					let k = 1;
					for (const value of additionalValues) {
						const currentK = k;
						const id = `value-${i}-${currentK}`;
						const input = (
							<PipelineFilterValueControl
								key={id}
								name={id}
								value={value}
								options={propertyValues}
								disabled={isInvalidProperty || isDisabled}
								onValueChange={onChangeValueFragment}
								removable={true}
								onRemove={() => onRemoveValue(i, currentK)}
							/>
						);
						valueElements.push(
							<div className='pipeline__filters-value pipeline__filters-value--additional' key={id}>
								{input}
							</div>,
						);
						k++;
					}
					valueElements.push(
						<SlButton
							className='pipeline__filters-add-value'
							key='add-button'
							variant='default'
							size='small'
							disabled={isDisabled}
							onClick={() => onAddValue(i)}
						>
							Add value
						</SlButton>,
					);
				}
			}

			let values: ReactNode;
			if (isOneOf) {
				values = <div className='pipeline__filters-is-one-of-values'>{valueElements}</div>;
			} else {
				values = valueElements;
			}

			conditions.push(
				<div className='pipeline__filters-filter'>
					{logicalElement}
					<div
						key={i}
						className={`pipeline__filters-condition${isOneOf ? ' pipeline__filters-condition--is-one-of' : ''}`}
					>
						<div className='pipeline__filters-property-and-operator'>
							{propertyInput}
							{pathInput}
							{operatorSelect}
						</div>
						{values}
						<div className='pipeline__filters-remove-condition-wrapper'>
							<SlButton
								className='pipeline__filters-remove-condition'
								size='small'
								onClick={() => onRemoveCondition(i)}
								disabled={isDisabled}
							>
								<SlIcon name='x-circle' slot='prefix' />
							</SlButton>
						</div>
					</div>
				</div>,
			);
		}
	}

	return (
		<Section
			className={`pipeline__filters${isDisabled ? ' pipeline__filters--disabled' : ''}`}
			title='Filters'
			description={
				<>
					<span>{`Choose which ${targetTerm} to ${actionVerb}. Leave empty to ${actionVerb} all ${targetTerm}.`}</span>
					<a href='https://www.krenalis.com/docs/ref/admin/filters' target='_blank' rel='noopener'>
						Learn more about filters
					</a>
				</>
			}
			padded={true}
			ref={ref}
			annotated={true}
		>
			{conditions}
			<SlButton
				className='pipeline__filters-add-condition'
				size='medium'
				variant='text'
				onClick={onAddCondition}
				disabled={isDisabled}
			>
				<SlIcon slot='prefix' name='plus-circle' />
				Add {conditions.length > 0 ? 'new ' : ''}filter
			</SlButton>
		</Section>
	);
});

interface PipelineFilterValueControlProps {
	name: string;
	value: string;
	options: string[] | null;
	disabled: boolean;
	onValueChange: (name: string, value: string) => void;
	removable?: boolean;
	onRemove?: () => void;
}

const PipelineFilterValueControl = ({
	name,
	value,
	options,
	disabled,
	onValueChange,
	removable = false,
	onRemove,
}: PipelineFilterValueControlProps) => {
	const handleInput = (event: any) => {
		onValueChange(name, event.target.value);
	};

	const handleSelect = (event: any) => {
		const target = event.target as any;
		const v = event.detail?.value ?? target.value;
		onValueChange(name, v);
	};

	if (options != null && options.length > 0) {
		return (
			<div
				className={`pipeline__filters-value-control${removable ? ' pipeline__filters-value-control--removable' : ''}`}
			>
				<SlSelect
					size='small'
					className='pipeline__filters-value-input'
					name={name}
					value={value ?? ''}
					onSlChange={handleSelect}
					disabled={disabled}
				>
					{options.map((option, index) => (
						<SlOption key={`${index}-${option}`} value={option}>
							{option === '' ? '\u00A0' : option}
						</SlOption>
					))}
					{removable && (
						<SlButton
							slot='suffix'
							variant='default'
							size='small'
							circle
							className='pipeline__filters-value-remove'
							onClick={onRemove}
							disabled={disabled}
						>
							<SlIcon name='x' />
						</SlButton>
					)}
				</SlSelect>
			</div>
		);
	}

	return (
		<SlInput
			size='small'
			className='pipeline__filters-value-input'
			value={value ?? ''}
			onSlInput={handleInput}
			name={name}
			disabled={disabled}
		>
			{removable && (
				<SlButton
					variant='default'
					size='small'
					circle
					className='pipeline__filters-value-remove'
					onClick={onRemove}
					slot='suffix'
					disabled={disabled}
				>
					<SlIcon name='x' />
				</SlButton>
			)}
		</SlInput>
	);
};

export default PipelineFilters;

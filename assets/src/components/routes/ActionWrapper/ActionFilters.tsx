import React, { useContext, ReactNode, useMemo, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import { getFilterPropertyComboboxItems } from '../../helpers/getSchemaComboboxItems';
import ActionContext from '../../../context/ActionContext';
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
} from '../../../lib/core/action';
import { FilterLogical, FilterOperator } from '../../../lib/api/types/action';
import { checkIfPropertyExists } from './Action.helpers';

const ActionFilters = forwardRef<any>((_, ref) => {
	const { action, setAction, actionType, connection, isTransformationDisabled } = useContext(ActionContext);

	const flatInputSchema = useMemo(() => {
		return flattenSchema(actionType.inputSchema);
	}, [actionType.inputSchema]);

	const onAddCondition = () => {
		const a = { ...action };
		if (a.filter == null) {
			a.filter = { logical: 'and', conditions: [] };
		}
		a.filter.conditions = [...a.filter.conditions, { property: '', operator: '', values: [''] }];
		setAction(a);
	};

	const onRemoveCondition = (id: number) => {
		const a = { ...action };
		a.filter!.conditions.splice(id, 1);
		if (a.filter!.conditions.length === 0) {
			a.filter = null;
		}
		setAction(a);
	};

	const onInputPropertyFragment = (name: string, value: string) => {
		updatePropertyFragment(name, value);
	};

	const onSelectPropertyFragment = (name: string, value: string) => {
		updatePropertyFragment(name, value);

		const a = { ...action };
		const id = Number(name.split('-')[1]);
		const currentOperator = a.filter!.conditions[id]['operator'];
		const currentOperatorIndex = FILTER_OPERATORS.findIndex((op) => op === currentOperator);
		const compatibleOperators = getCompatibleFilterOperators(flatInputSchema[value]);
		const isCompatible = compatibleOperators.includes(currentOperatorIndex);
		const isJson = flatInputSchema[value]?.type === 'json';

		if (!isCompatible) {
			// Select the first compatible operator.
			const operator = FILTER_OPERATORS[compatibleOperators[0]];
			changeOperator(id, operator);
			if (!isJson) {
				setTimeout(() => {
					const operatorSelect: any = document
						.querySelector(`[data-id=property-${id}]`)
						.closest('.action__filters-condition')
						.querySelector('.action__filters-operator');
					operatorSelect.show();
				}, 10);
			}
		}

		if (isJson) {
			setTimeout(() => {
				const pathInput: any = document
					.querySelector(`[data-id=property-${id}]`)
					.closest('.action__filters-condition')
					.querySelector('.action__filters-path');
				pathInput.select();
			}, 10);
		}
	};

	const updatePropertyFragment = (name: string, value: string) => {
		const a = { ...action };
		const id = Number(name.split('-')[1]);
		const propertyName = a.filter!.conditions[id]['property'];
		const [_, path] = splitPropertyAndPath(propertyName, flatInputSchema);
		let newPropertyName = '';
		if (path !== '' && flatInputSchema[value]?.type === 'json') {
			newPropertyName = `${value}.${path}`;
		} else {
			newPropertyName = value;
		}
		const compatibleOperators = getCompatibleFilterOperators(flatInputSchema[newPropertyName]);
		const currentOperator = a.filter!.conditions[id]['operator'];
		if (currentOperator != null && currentOperator !== '') {
			const index = FILTER_OPERATORS.indexOf(currentOperator);
			if (!compatibleOperators.includes(index)) {
				// The current operator is not compatible with the new property.
				// Reset the operator and the values.
				a.filter!.conditions[id]['operator'] = '';
				a.filter!.conditions[id]['values'] = [''];
			}
		}
		a.filter!.conditions[id]['property'] = newPropertyName;
		setAction(a);
	};

	const onInputPathFragment = (e) => {
		const a = { ...action };
		const id = Number(e.target.name.split('-')[1]);
		const propertyName = a.filter!.conditions[id]['property'];
		const [base, _] = splitPropertyAndPath(propertyName, flatInputSchema);
		let newPropertyName = '';
		const newPath = e.target.value;
		if (newPath !== '') {
			newPropertyName = `${base}.${newPath}`;
		} else {
			newPropertyName = base;
		}
		a.filter!.conditions[id]['property'] = newPropertyName;
		setAction(a);
	};

	const onChangeOperatorFragment = (e: any) => {
		const id = Number(e.target.name.split('-')[1]);
		const operator = FILTER_OPERATORS[e.target.value];
		changeOperator(id, operator);
		if (!isUnaryOperator(operator)) {
			// Focus the first value input.
			setTimeout(() => {
				const valueInput = e.target
					.closest('.action__filters-condition')
					.querySelector('.action__filters-value-input');
				valueInput.focus();
			}, 50);
		}
	};

	const changeOperator = (conditionID: number, operator: FilterOperator) => {
		const id = conditionID;
		const a = { ...action };
		a.filter!.conditions[id]['operator'] = operator;
		if (isUnaryOperator(operator)) {
			a.filter!.conditions[id]['values'] = null;
		} else {
			const isBetween = isBetweenOperator(operator);
			const isOneOf = isOneOfOperator(operator);

			let values: string[] | null;
			if (isBetween) {
				let v = ['', ''];
				if (a.filter!.conditions[id]['values'] != null) {
					v = a.filter!.conditions[id]['values'].slice(0, 2);
					if (v.length === 1) {
						v.push('');
					}
				}
				values = v;
			} else if (isOneOf) {
				let v = a.filter!.conditions[id]['values'];
				if (v == null) {
					v = [''];
				}
				values = v;
			} else {
				let v = '';
				if (a.filter!.conditions[id]['values'] != null) {
					v = a.filter!.conditions[id]['values'][0];
				}
				values = [v];
			}

			a.filter!.conditions[id]['values'] = values;
		}
		setAction(a);
	};

	const onInputValueFragment = (e: any) => {
		const a = { ...action };
		const split = e.target.name.split('-');
		const id = Number(split[1]);
		const position = Number(split[2]);
		a.filter!.conditions[id]['values'][position] = e.target.value;
		setAction(a);
	};

	const onLogicalClick = (logical: FilterLogical) => {
		const a = { ...action };
		a.filter!.logical = logical;
		setAction(a);
	};

	const onAddValue = (index: number) => {
		const a = { ...action };
		const currentLength = a.filter!.conditions[index]['values'].length;
		a.filter!.conditions[index]['values'] = [...a.filter!.conditions[index]['values'], ''];
		setAction(a);
		setTimeout(() => {
			// focus the new input.
			const valueInputs: any = document
				.querySelector(`[data-id=property-${index}]`)
				.closest('.action__filters-condition')
				.querySelectorAll('.action__filters-value-input');
			const newValueInput = valueInputs[currentLength];
			newValueInput.focus();
		}, 50);
	};

	const onRemoveValue = (conditionIndex: number, valueIndex: number) => {
		const a = { ...action };
		const values = a.filter!.conditions[conditionIndex]['values'];
		const filtered = [];
		for (const [i, v] of values.entries()) {
			if (i !== valueIndex) {
				filtered.push(v);
			}
		}
		a.filter!.conditions[conditionIndex]['values'] = filtered;
		setAction(a);
	};

	const isFileStorageImport = connection.isFileStorage && connection.isSource;

	// For file storage imports, the filter section is displayed
	// together with the transformation section when the file's settings
	// are confirmed. It must be disabled when the settings are changed
	// and not yet re-confirmed.
	const isDisabled = isFileStorageImport && isTransformationDisabled;

	const conditions: ReactNode[] = [];
	if (action.filter != null) {
		for (const [i, condition] of action.filter.conditions.entries()) {
			const [base, path] = splitPropertyAndPath(condition.property, flatInputSchema);

			let property = flatInputSchema[base];
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

			if (i === 0) {
				if (action.filter.conditions.length > 1) {
					// Add a placeholder to mantain alignment.
					logicalElement = (
						<div className='action__filters-logical action__filters-logical--placeholder'></div>
					);
				}
			} else if (i === 1) {
				logicalElement = (
					<SlButtonGroup className='action__filters-logical'>
						<SlButton
							size='small'
							variant={action.filter!.logical === 'and' ? 'primary' : 'default'}
							onClick={() => onLogicalClick('and')}
							disabled={isDisabled}
						>
							and
						</SlButton>
						<SlButton
							size='small'
							variant={action.filter!.logical === 'or' ? 'primary' : 'default'}
							onClick={() => onLogicalClick('or')}
							disabled={isDisabled}
						>
							or
						</SlButton>
					</SlButtonGroup>
				);
			} else if (i > 1) {
				logicalElement = (
					<div className='action__filters-logical action__filters-logical--text'>
						{action.filter!.logical}
					</div>
				);
			}

			propertyInput = (
				<Combobox
					onInput={onInputPropertyFragment}
					onSelect={onSelectPropertyFragment}
					value={isJSON ? base : condition.property}
					className='action__filters-property'
					size='small'
					name={`property-${i}`}
					items={getFilterPropertyComboboxItems(actionType.inputSchema)}
					isExpression={false}
					disabled={isDisabled}
					placeholder={'Property'}
					caret={true}
					controlled={true}
					autoResize={true}
					error={condition.property !== '' && checkIfPropertyExists(condition.property, flatInputSchema)}
				/>
			);

			if (isJSON) {
				pathInput = (
					<SlInput
						size='small'
						className='action__filters-path'
						value={path}
						onSlInput={onInputPathFragment}
						name={`path-${i}`}
						disabled={isDisabled}
						placeholder='Path'
					/>
				);
			}

			operatorSelect = (
				<SlSelect
					size='small'
					name={`operator-${i}`}
					className='action__filters-operator'
					value={String(FILTER_OPERATORS.findIndex((op) => op === condition.operator))}
					onSlChange={onChangeOperatorFragment}
					placeholder='Operator'
					disabled={isInvalidProperty || isDisabled}
				>
					{property != null
						? getCompatibleFilterOperators(property).map((i) => (
								<SlOption key={i} value={String(i)}>
									{FILTER_OPERATORS[i]}
								</SlOption>
							))
						: Object.keys(FILTER_OPERATORS).map((i) => (
								<SlOption key={i} value={String(i)}>
									{FILTER_OPERATORS[i]}
								</SlOption>
							))}
				</SlSelect>
			);

			if (!isUnary) {
				const id = `value-${i}-0`;
				valueElements.push(
					<SlInput
						key={id}
						size='small'
						className='action__filters-value-input'
						value={condition.values != null ? condition.values[0] : ''}
						onSlInput={onInputValueFragment}
						name={id}
						disabled={isInvalidProperty || isDisabled}
					/>,
				);
				if (isBetween) {
					valueElements.push(
						<span className='action__filters-value-and' key='and'>
							and
						</span>,
					);
					const id = `value-${i}-1`;
					valueElements.push(
						<SlInput
							key={id}
							size='small'
							className='action__filters-value-input'
							value={condition.values != null ? (condition.values[1] ? condition.values[1] : '') : ''}
							onSlInput={onInputValueFragment}
							name={id}
							disabled={isInvalidProperty || isDisabled}
						/>,
					);
				} else if (isOneOf) {
					const additionalValues = condition.values.slice(1);
					let k = 1;
					for (const value of additionalValues) {
						const currentK = k;
						const id = `value-${i}-${currentK}`;
						const input = (
							<SlInput
								size='small'
								className='action__filters-value-input'
								value={value}
								onSlInput={onInputValueFragment}
								name={id}
								disabled={isInvalidProperty || isDisabled}
							>
								<SlButton
									variant='default'
									size='small'
									circle
									className='action__filters-value-remove'
									onClick={() => onRemoveValue(i, currentK)}
									slot='suffix'
									disabled={isDisabled}
								>
									<SlIcon name='x' />
								</SlButton>
							</SlInput>
						);
						valueElements.push(
							<div className='action__filters-value action__filters-value--additional' key={id}>
								{input}
							</div>,
						);
						k++;
					}
					valueElements.push(
						<SlButton
							className='action__filters-add-value'
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
				values = <div className='action__filters-is-one-of-values'>{valueElements}</div>;
			} else {
				values = valueElements;
			}

			conditions.push(
				<div className='action__filters-filter'>
					{logicalElement}
					<div
						key={i}
						className={`action__filters-condition${isOneOf ? ' action__filters-condition--is-one-of' : ''}`}
					>
						<div className='action__filters-property-and-operator'>
							{propertyInput}
							{pathInput}
							{operatorSelect}
						</div>
						{values}
						<div className='action__filters-remove-condition-wrapper'>
							<SlButton
								className='action__filters-remove-condition'
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
			className={`action__filters${isDisabled ? ' action__filters--disabled' : ''}`}
			title='Filter'
			description='The filters that define the action'
			padded={true}
			ref={ref}
			annotated={true}
		>
			{conditions}
			<SlButton
				className='action__filters-add-condition'
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

export default ActionFilters;

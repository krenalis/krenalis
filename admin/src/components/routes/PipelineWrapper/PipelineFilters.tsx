import React, { useContext, ReactNode, useMemo, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import { getFilterPropertyComboboxItems } from '../../helpers/getSchemaComboboxItems';
import PipelineContext from '../../../context/PipelineContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { Combobox } from '../../base/Combobox/Combobox';
import {
	FILTER_OPERATORS,
	MAX_FILTER_DEPTH,
	MAX_FILTER_RULE_COUNT,
	flattenSchema,
	getCompatibleFilterOperators,
	isBetweenOperator,
	isFilterGroup,
	isOneOfOperator,
	isUnaryOperator,
	splitPropertyAndPath,
	TransformedPipeline,
	TransformedProperty,
} from '../../../lib/core/pipeline';
import { Filter, FilterCondition, FilterLogical, FilterOperator } from '../../../lib/api/types/pipeline';
import { checkIfPropertyExists, pipelineObjectLabels } from './Pipeline.helpers';
import { StringType } from '../../../lib/api/types/types';

// countFilterRules returns the number of groups and conditions below filter.
const countFilterRules = (filter: Filter): number =>
	filter.rules.reduce((count, rule) => count + 1 + (isFilterGroup(rule) ? countFilterRules(rule) : 0), 0);

// filterGroupAt returns the group at path in filter.
const filterGroupAt = (filter: Filter, path: number[]): Filter => {
	let group = filter;
	for (const index of path) {
		const rule = group.rules[index];
		if (!isFilterGroup(rule)) {
			throw new Error('Filter path does not identify a group');
		}
		group = rule;
	}
	return group;
};

// filterConditionAt returns the condition at path in filter.
const filterConditionAt = (filter: Filter, path: number[]): FilterCondition => {
	const group = filterGroupAt(filter, path.slice(0, -1));
	const rule = group.rules[path[path.length - 1]];
	if (isFilterGroup(rule)) {
		throw new Error('Filter path does not identify a condition');
	}
	return rule;
};

// filterConditionValues returns the values of condition, initializing them
// when they are absent from the API representation.
const filterConditionValues = (condition: FilterCondition): string[] => (condition.values ??= []);

// newFilterCondition returns an empty condition for the visual editor.
const newFilterCondition = (): FilterCondition => ({ property: '', operator: '', values: [''] });

// pathID returns a stable identifier for a rule path.
const pathID = (path: number[]): string => path.join('-');

const PipelineFilters = forwardRef<any>((_, ref) => {
	const { pipeline, setPipeline, pipelineType, connection, isTransformationDisabled, isImport } =
		useContext(PipelineContext);
	const [filterSubject] = pipelineObjectLabels(connection, pipeline, 'plural');
	const filterSubjectTerm = filterSubject.toLowerCase();
	const actionVerb = isImport ? 'import' : pipelineType.target === 'Event' ? 'send' : 'export';

	const flatInputSchema = useMemo(() => flattenSchema(pipelineType.inputSchema), [pipelineType.inputSchema]);

	const isEventBasedUserImport = connection.isEventBased && connection.isSource && pipeline.target === 'User';
	const isAppEventsExport = connection.isApplication && connection.isDestination && pipeline.target === 'Event';
	const isEventImport = connection.isSource && pipeline.target === 'Event';
	const propertiesToHide = isEventBasedUserImport || isAppEventsExport || isEventImport ? ['kpid'] : [];

	const isFileStorageImport = connection.isFileStorage && connection.isSource;
	const isDisabled = isFileStorageImport && isTransformationDisabled;
	const filterRuleCount = pipeline.filter == null ? 0 : countFilterRules(pipeline.filter);

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
		return (property.full.type as StringType).values ?? null;
	};

	const onAddCondition = (groupPath: number[]) => {
		const p = structuredClone(pipeline);
		if (p.filter == null) {
			p.filter = { operator: 'and', rules: [] };
		}
		filterGroupAt(p.filter, groupPath).rules.push(newFilterCondition());
		setPipeline(p);
	};

	const onAddGroup = (groupPath: number[]) => {
		const p = structuredClone(pipeline);
		const parent = filterGroupAt(p.filter!, groupPath);
		parent.rules.push({
			operator: parent.operator === 'and' ? 'or' : 'and',
			rules: [newFilterCondition()],
		});
		setPipeline(p);
	};

	const onRemoveRule = (path: number[]) => {
		const p = structuredClone(pipeline);
		const filter = p.filter!;
		const groupPath = path.slice(0, -1);
		const group = filterGroupAt(filter, groupPath);
		group.rules.splice(path[path.length - 1], 1);

		if (group.rules.length === 0) {
			if (groupPath.length === 0) {
				p.filter = null;
			} else {
				group.rules.push(newFilterCondition());
			}
		}
		setPipeline(p);
	};

	const onLogicalChange = (groupPath: number[], operator: FilterLogical) => {
		const p = structuredClone(pipeline);
		filterGroupAt(p.filter!, groupPath).operator = operator;
		setPipeline(p);
	};

	const updateProperty = (path: number[], value: string): TransformedPipeline => {
		const p = structuredClone(pipeline);
		const condition = filterConditionAt(p.filter!, path);
		const previousPropertyName = condition.property;
		const previousPropertyValues = getPropertyValues(findPropertyInSchema(previousPropertyName));
		const [, previousPath] = splitPropertyAndPath(previousPropertyName, flatInputSchema);
		const newPropertyName =
			previousPath !== '' && flatInputSchema[value]?.type === 'json' ? `${value}.${previousPath}` : value;
		const newProperty = findPropertyInSchema(newPropertyName);
		const [, newPath] = splitPropertyAndPath(newPropertyName, flatInputSchema);

		const compatibleOperators = getCompatibleFilterOperators(
			newProperty,
			newPath !== '',
			connection.role,
			pipeline.target,
		);
		if (condition.operator !== '' && !compatibleOperators.includes(FILTER_OPERATORS.indexOf(condition.operator))) {
			condition.operator = '';
			condition.values = [''];
		}

		condition.property = newPropertyName;
		const newPropertyValues = getPropertyValues(newProperty);
		if (previousPropertyName !== newPropertyName && (previousPropertyValues != null || newPropertyValues != null)) {
			condition.values = isBetweenOperator(condition.operator) ? ['', ''] : [''];
		}
		setPipeline(p);
		return p;
	};

	const onSelectProperty = (path: number[], value: string) => {
		const p = updateProperty(path, value);
		const condition = filterConditionAt(p.filter!, path);
		const [, propertyPath] = splitPropertyAndPath(condition.property, flatInputSchema);
		const compatibleOperators = getCompatibleFilterOperators(
			flatInputSchema[value],
			propertyPath !== '',
			connection.role,
			pipeline.target,
		);
		const currentOperatorIndex = FILTER_OPERATORS.findIndex((operator) => operator === condition.operator);
		const isJSON = flatInputSchema[value]?.type === 'json';

		if (!compatibleOperators.includes(currentOperatorIndex) && compatibleOperators.length > 0) {
			changeOperator(path, FILTER_OPERATORS[compatibleOperators[0]], p);
			if (!isJSON) {
				setTimeout(() => {
					const property: any = document.querySelector(`[data-id="property-${pathID(path)}"]`);
					property
						?.closest('.pipeline__filters-condition')
						?.querySelector('.pipeline__filters-operator')
						?.show();
				}, 10);
			}
		}

		if (isJSON) {
			setTimeout(() => {
				const property: any = document.querySelector(`[data-id="property-${pathID(path)}"]`);
				property?.closest('.pipeline__filters-condition')?.querySelector('.pipeline__filters-path')?.select();
			}, 10);
		}
	};

	const onInputPath = (path: number[], value: string) => {
		const p = structuredClone(pipeline);
		const condition = filterConditionAt(p.filter!, path);
		const [base] = splitPropertyAndPath(condition.property, flatInputSchema);
		const compatibleOperators = getCompatibleFilterOperators(
			flatInputSchema[base],
			value !== '',
			connection.role,
			pipeline.target,
		);
		if (condition.operator !== '' && !compatibleOperators.includes(FILTER_OPERATORS.indexOf(condition.operator))) {
			condition.operator = '';
			condition.values = [''];
		}
		condition.property = value === '' ? base : `${base}.${value}`;
		setPipeline(p);
	};

	const changeOperator = (path: number[], operator: FilterOperator, updatedPipeline?: TransformedPipeline) => {
		const p = updatedPipeline ?? structuredClone(pipeline);
		const condition = filterConditionAt(p.filter!, path);
		const values = filterConditionValues(condition);
		condition.operator = operator;
		if (isUnaryOperator(operator)) {
			condition.values = [];
		} else if (isBetweenOperator(operator)) {
			condition.values = values.slice(0, 2);
			while (condition.values.length < 2) {
				condition.values.push('');
			}
		} else if (isOneOfOperator(operator)) {
			if (values.length === 0) {
				condition.values = [''];
			}
		} else {
			condition.values = [values[0] ?? ''];
		}
		setPipeline(p);
	};

	const onOperatorSelectClose = (event: any) => {
		const operator = FILTER_OPERATORS[event.target.value];
		if (operator == null || isUnaryOperator(operator)) {
			return;
		}
		setTimeout(() => {
			const valueInput = event.target
				.closest('.pipeline__filters-condition')
				?.querySelector('.pipeline__filters-value-input');
			if (valueInput == null) {
				return;
			}
			if (valueInput.tagName === 'SL-SELECT') {
				valueInput.show();
			} else {
				valueInput.focus();
			}
		}, 50);
	};

	const onChangeValue = (path: number[], position: number, value: string) => {
		const p = structuredClone(pipeline);
		filterConditionValues(filterConditionAt(p.filter!, path))[position] = value;
		setPipeline(p);
	};

	const onAddValue = (path: number[]) => {
		const p = structuredClone(pipeline);
		const condition = filterConditionAt(p.filter!, path);
		const values = filterConditionValues(condition);
		const position = values.length;
		values.push('');
		setPipeline(p);
		setTimeout(() => {
			const property: any = document.querySelector(`[data-id="property-${pathID(path)}"]`);
			const inputs = property
				?.closest('.pipeline__filters-condition')
				?.querySelectorAll('.pipeline__filters-value-input');
			const input = inputs?.[position];
			if (input == null) {
				return;
			}
			if (input.tagName === 'SL-SELECT') {
				input.show();
			} else {
				input.focus();
			}
		}, 50);
	};

	const onRemoveValue = (path: number[], position: number) => {
		const p = structuredClone(pipeline);
		const condition = filterConditionAt(p.filter!, path);
		filterConditionValues(condition).splice(position, 1);
		setPipeline(p);
	};

	const renderCondition = (condition: FilterCondition, path: number[]): ReactNode => {
		const id = pathID(path);
		const groupPath = path.slice(0, -1);
		const isOnlyRuleInNestedGroup =
			groupPath.length > 0 && filterGroupAt(pipeline.filter!, groupPath).rules.length === 1;
		const isRemoveDisabled = isDisabled || isOnlyRuleInNestedGroup;
		const [base, propertyPath] = splitPropertyAndPath(condition.property, flatInputSchema);
		const property = flatInputSchema?.[base];
		const isUnary = isUnaryOperator(condition.operator);
		const isJSON = property?.type === 'json';
		const isBetween = isBetweenOperator(condition.operator);
		const isOneOf = isOneOfOperator(condition.operator);
		const isInvalidProperty = property == null;
		const propertyValues = getPropertyValues(property) ?? [];
		const values = condition.values ?? [];

		const propertyInput = (
			<Combobox
				onInput={(_, value) => updateProperty(path, value)}
				onSelect={(_, value) => onSelectProperty(path, value)}
				value={isJSON ? base : condition.property}
				className='pipeline__filters-property'
				size='small'
				name={`property-${id}`}
				items={getFilterPropertyComboboxItems(
					pipelineType.inputSchema,
					connection,
					pipeline.target,
					propertiesToHide,
				)}
				isExpression={false}
				disabled={isDisabled}
				placeholder='Property'
				caret={true}
				controlled={true}
				autoResize={true}
				error={
					condition.property !== '' &&
					checkIfPropertyExists(isJSON ? base : condition.property, flatInputSchema, propertiesToHide)
				}
			/>
		);
		const pathInput = isJSON ? (
			<SlInput
				size='small'
				className='pipeline__filters-path'
				value={propertyPath}
				onSlInput={(event: any) => onInputPath(path, event.target.value)}
				name={`path-${id}`}
				disabled={isDisabled}
				placeholder='Path (optional)'
			/>
		) : null;
		const operatorSelect = (
			<SlSelect
				size='small'
				name={`operator-${id}`}
				className='pipeline__filters-operator'
				value={String(FILTER_OPERATORS.findIndex((operator) => operator === condition.operator))}
				onSlChange={(event: any) => changeOperator(path, FILTER_OPERATORS[event.target.value])}
				onSlHide={onOperatorSelectClose}
				placeholder='Operator'
				disabled={isInvalidProperty || isDisabled}
			>
				{property != null
					? getCompatibleFilterOperators(property, propertyPath !== '', connection.role, pipeline.target).map(
							(index) => (
								<SlOption key={index} value={String(index)}>
									{FILTER_OPERATORS[index]}
								</SlOption>
							),
						)
					: FILTER_OPERATORS.map((operator, index) => (
							<SlOption key={operator} value={String(index)}>
								{operator}
							</SlOption>
						))}
			</SlSelect>
		);

		const valueElements: ReactNode[] = [];
		if (!isUnary) {
			valueElements.push(
				<PipelineFilterValueControl
					key={`value-${id}-0`}
					name={`value-${id}-0`}
					value={values[0] ?? ''}
					options={propertyValues}
					disabled={isInvalidProperty || isDisabled}
					onValueChange={(value) => onChangeValue(path, 0, value)}
				/>,
			);
			if (isBetween) {
				valueElements.push(
					<span className='pipeline__filters-value-and' key='and'>
						and
					</span>,
					<PipelineFilterValueControl
						key={`value-${id}-1`}
						name={`value-${id}-1`}
						value={values[1] ?? ''}
						options={propertyValues}
						disabled={isInvalidProperty || isDisabled}
						onValueChange={(value) => onChangeValue(path, 1, value)}
					/>,
				);
			} else if (isOneOf) {
				for (const [position, value] of values.slice(1).entries()) {
					const valuePosition = position + 1;
					valueElements.push(
						<div className='pipeline__filters-value' key={`value-${id}-${valuePosition}`}>
							<PipelineFilterValueControl
								name={`value-${id}-${valuePosition}`}
								value={value}
								options={propertyValues}
								disabled={isInvalidProperty || isDisabled}
								onValueChange={(value) => onChangeValue(path, valuePosition, value)}
								removable={true}
								onRemove={() => onRemoveValue(path, valuePosition)}
							/>
						</div>,
					);
				}
				valueElements.push(
					<SlButton
						className='pipeline__filters-add-value'
						key='add-button'
						variant='default'
						size='small'
						disabled={isDisabled}
						onClick={() => onAddValue(path)}
					>
						Add value
					</SlButton>,
				);
			}
		}

		const removeButton = (
			<SlButton
				className='pipeline__filters-remove-condition pipeline__filters-remove-rule'
				size='small'
				variant='text'
				onClick={() => onRemoveRule(path)}
				disabled={isRemoveDisabled}
			>
				<SlIcon name='x-circle' aria-hidden='true' />
				<span className='pipeline__filters-remove-rule-label'>Remove condition</span>
			</SlButton>
		);

		return (
			<div className='pipeline__filters-filter' key={id}>
				<div
					className={`pipeline__filters-condition${isOneOf ? ' pipeline__filters-condition--is-one-of' : ''}`}
				>
					<div className='pipeline__filters-property-and-operator'>
						{propertyInput}
						{pathInput}
						{operatorSelect}
					</div>
					{isOneOf ? (
						<div className='pipeline__filters-is-one-of-values'>{valueElements}</div>
					) : (
						valueElements
					)}
					<div className='pipeline__filters-remove-condition-wrapper'>
						{isRemoveDisabled ? (
							removeButton
						) : (
							<SlTooltip content='Remove condition' hoist={true}>
								{removeButton}
							</SlTooltip>
						)}
					</div>
				</div>
			</div>
		);
	};

	const renderGroup = (group: Filter, groupPath: number[], depth: number): ReactNode => {
		const isRoot = depth === 1;
		const removeGroupTooltip = group.rules.some(isFilterGroup) ? 'Remove groups' : 'Remove group';

		return (
			<div
				className={`pipeline__filters-group${isRoot ? ' pipeline__filters-group--root' : ''}`}
				key={pathID(groupPath) || 'root'}
			>
				<div className='pipeline__filters-group-header'>
					<span className='pipeline__filters-group-subject'>
						{isRoot ? `${filterSubject} matching` : 'matching'}
					</span>
					<SlSelect
						className='pipeline__filters-logical'
						size='small'
						value={group.operator}
						onSlChange={(event: any) => onLogicalChange(groupPath, event.target.value as FilterLogical)}
						disabled={isDisabled}
						aria-label='Match rules'
					>
						<SlOption value='and'>all</SlOption>
						<SlOption value='or'>any</SlOption>
					</SlSelect>
					<span>of the following:</span>
					{!isRoot && (
						<div className='pipeline__filters-remove-group-wrapper'>
							<SlTooltip content={removeGroupTooltip} hoist={true}>
								<SlButton
									className='pipeline__filters-remove-group pipeline__filters-remove-rule'
									size='small'
									variant='text'
									onClick={() => onRemoveRule(groupPath)}
									disabled={isDisabled}
								>
									<SlIcon name='x-circle' aria-hidden='true' />
									<span className='pipeline__filters-remove-rule-label'>Remove group</span>
								</SlButton>
							</SlTooltip>
						</div>
					)}
				</div>
				<div className='pipeline__filters-group-rules'>
					{group.rules.map((rule, index) => {
						const path = [...groupPath, index];
						const isGroup = isFilterGroup(rule);

						return (
							<div
								className={`pipeline__filters-rule${isGroup ? ' pipeline__filters-rule--group' : ''}`}
								key={pathID(path)}
							>
								{index > 0 && (
									<div className='pipeline__filters-connector' aria-hidden='true'>
										{group.operator}
									</div>
								)}
								<div className='pipeline__filters-rule-content'>
									{isGroup ? renderGroup(rule, path, depth + 1) : renderCondition(rule, path)}
								</div>
							</div>
						);
					})}
				</div>
				<div className='pipeline__filters-group-actions'>
					<SlButton
						className='pipeline__filters-add-condition'
						size='medium'
						variant='text'
						onClick={() => onAddCondition(groupPath)}
						disabled={isDisabled || filterRuleCount >= MAX_FILTER_RULE_COUNT}
					>
						<SlIcon slot='prefix' name='plus-circle' />
						Add a condition
					</SlButton>
					<SlButton
						className='pipeline__filters-add-group'
						size='medium'
						variant='text'
						onClick={() => onAddGroup(groupPath)}
						disabled={
							isDisabled || depth >= MAX_FILTER_DEPTH || filterRuleCount > MAX_FILTER_RULE_COUNT - 2
						}
					>
						<SlIcon slot='prefix' name='plus-circle' />
						Add a group
					</SlButton>
				</div>
			</div>
		);
	};

	return (
		<Section
			className={`pipeline__filters${isDisabled ? ' pipeline__filters--disabled' : ''}`}
			title='Filters'
			description={
				<>
					<span>{`Choose which ${filterSubjectTerm} to ${actionVerb}. Leave empty to ${actionVerb} all ${filterSubjectTerm}.`}</span>
					<a href='https://www.krenalis.com/docs/ref/admin/filters' target='_blank' rel='noopener'>
						Learn more about filters
					</a>
				</>
			}
			padded={true}
			ref={ref}
			annotated={true}
		>
			{pipeline.filter == null ? (
				<SlButton
					className='pipeline__filters-add-condition'
					size='medium'
					variant='text'
					onClick={() => onAddCondition([])}
					disabled={isDisabled || filterRuleCount >= MAX_FILTER_RULE_COUNT}
				>
					<SlIcon slot='prefix' name='plus-circle' />
					Add filter
				</SlButton>
			) : (
				renderGroup(pipeline.filter, [], 1)
			)}
		</Section>
	);
});

interface PipelineFilterValueControlProps {
	name: string;
	value: string;
	options: string[];
	disabled: boolean;
	onValueChange: (value: string) => void;
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
		onValueChange(event.target.value);
	};

	const handleSelect = (event: any) => {
		const target = event.target as any;
		const v = event.detail?.value ?? target.value;
		onValueChange(v);
	};

	const removeButton = removable ? (
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
	) : null;

	if (options.length > 0) {
		return (
			<div className={removable ? 'pipeline__filters-value-control--removable' : undefined}>
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
					{removeButton}
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
			{removeButton}
		</SlInput>
	);
};

export default PipelineFilters;

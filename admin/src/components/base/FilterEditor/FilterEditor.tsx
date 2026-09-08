import React, { ReactNode, RefObject, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import './FilterEditor.css';
import { getFilterPropertyComboboxItems } from '../../helpers/getSchemaComboboxItems';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import type SlButtonElement from '@shoelace-style/shoelace/dist/components/button/button.component.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import type SlInputElement from '@shoelace-style/shoelace/dist/components/input/input.component.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { Combobox } from '../Combobox/Combobox';
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
	TransformedProperty,
} from '../../../lib/core/pipeline';
import {
	Filter,
	FilterCondition,
	FilterLogical,
	FilterOperator,
	PipelineTarget,
} from '../../../lib/api/types/pipeline';
import { ObjectType, Role, StringType } from '../../../lib/api/types/types';

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

const NO_HIDDEN_PROPERTIES: string[] = [];
const EMPTY_FILTER: Filter = { operator: 'and', rules: [] };
const EMPTY_FILTER_WITH_CONDITION: Filter = { operator: 'and', rules: [newFilterCondition()] };

// checkIfPropertyExists returns an error when property is not available to the
// filter editor.
const checkIfPropertyExists = (
	property: string,
	schema: ReturnType<typeof flattenSchema>,
	toHide: string[],
): string => {
	if (schema == null || property === '') {
		return '';
	}
	if (schema[property] == null || toHide.includes(property)) {
		return `Property "${property}" does not exist`;
	}
	return '';
};

interface FilterEditorProps {
	disabled?: boolean;
	emptyFilterFocusRef?: RefObject<HTMLElement>;
	filter: Filter | null;
	hoistMenus?: boolean;
	onChange: (filter: Filter | null) => void;
	onCommit?: (filter: Filter | null) => void;
	propertiesToHide?: string[];
	role: Role;
	rootActions?: ReactNode;
	schema: ObjectType;
	showEmptyFilterAsCondition?: boolean;
	subject: string;
	target: PipelineTarget;
	textValueCommitDelayMs?: number;
}

const FilterEditor = ({
	disabled = false,
	emptyFilterFocusRef,
	filter,
	hoistMenus = false,
	onChange,
	onCommit,
	propertiesToHide = NO_HIDDEN_PROPERTIES,
	role,
	rootActions,
	schema,
	showEmptyFilterAsCondition = false,
	subject,
	target,
	textValueCommitDelayMs,
}: FilterEditorProps) => {
	const [conditionRemovalAnnouncement, setConditionRemovalAnnouncement] = useState('');
	const announcementFrameRef = useRef<number>();
	const rootRef = useRef<HTMLDivElement>(null);
	const pendingConditionFocusRef = useRef<number | 'add-condition'>();
	const editableFilter = filter ?? (showEmptyFilterAsCondition ? EMPTY_FILTER_WITH_CONDITION : EMPTY_FILTER);
	const flatInputSchema = useMemo(() => flattenSchema(schema), [schema]);
	const propertyItems = useMemo(
		() => getFilterPropertyComboboxItems(schema, role, target, propertiesToHide),
		[propertiesToHide, role, schema, target],
	);
	const filterRuleCount = countFilterRules(editableFilter);

	useEffect(
		() => () => {
			if (announcementFrameRef.current != null) {
				cancelAnimationFrame(announcementFrameRef.current);
			}
		},
		[],
	);

	useLayoutEffect(() => {
		const pendingFocus = pendingConditionFocusRef.current;
		if (pendingFocus == null) {
			return;
		}
		pendingConditionFocusRef.current = undefined;

		const propertyInputs = rootRef.current?.querySelectorAll<SlInputElement>(
			'.pipeline__filters-property sl-input',
		);
		const target =
			typeof pendingFocus === 'number'
				? propertyInputs?.[pendingFocus]
				: rootRef.current?.querySelector<SlButtonElement>('.pipeline__filters-add-condition');
		if (target != null) {
			// Removing a rule can remount the next input before its internal control is ready.
			let canceled = false;
			void target.updateComplete.then(() => {
				if (!canceled && target.isConnected) {
					target.focus({ preventScroll: true });
				}
			});
			return () => {
				canceled = true;
			};
		}

		// The fallback may be a newly mounted web component whose internal control is not ready yet.
		const animationFrame = requestAnimationFrame(() =>
			emptyFilterFocusRef?.current?.focus({ preventScroll: true }),
		);
		return () => cancelAnimationFrame(animationFrame);
	}, [emptyFilterFocusRef, filter]);

	const announceConditionRemoved = () => {
		setConditionRemovalAnnouncement('');
		if (announcementFrameRef.current != null) {
			cancelAnimationFrame(announcementFrameRef.current);
		}
		announcementFrameRef.current = requestAnimationFrame(() => {
			announcementFrameRef.current = undefined;
			setConditionRemovalAnnouncement('Condition removed');
		});
	};

	const changeFilter = (updatedFilter: Filter | null, commit = false) => {
		onChange(updatedFilter);
		if (commit) {
			onCommit?.(updatedFilter);
		}
	};

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
		const updatedFilter = structuredClone(editableFilter);
		filterGroupAt(updatedFilter, groupPath).rules.push(newFilterCondition());
		changeFilter(updatedFilter);
	};

	const onAddGroup = (groupPath: number[]) => {
		const updatedFilter = structuredClone(editableFilter);
		const parent = filterGroupAt(updatedFilter, groupPath);
		parent.rules.push({
			operator: parent.operator === 'and' ? 'or' : 'and',
			rules: [newFilterCondition()],
		});
		changeFilter(updatedFilter);
	};

	const onRemoveRule = (path: number[]) => {
		const updatedFilter = structuredClone(editableFilter);
		const groupPath = path.slice(0, -1);
		const group = filterGroupAt(updatedFilter, groupPath);
		group.rules.splice(path[path.length - 1], 1);

		if (group.rules.length === 0) {
			if (groupPath.length === 0) {
				changeFilter(null, true);
				return;
			}
			group.rules.push(newFilterCondition());
		}
		changeFilter(updatedFilter, true);
	};

	const onRemoveCondition = (path: number[]) => {
		const propertyInputs = Array.from(
			rootRef.current?.querySelectorAll<HTMLElement>('.pipeline__filters-property sl-input') ?? [],
		);
		const currentProperty = rootRef.current?.querySelector<HTMLElement>(
			`[data-id="property-${pathID(path)}"] sl-input`,
		);
		const currentIndex = currentProperty == null ? -1 : propertyInputs.indexOf(currentProperty);

		if (currentIndex >= 0 && currentIndex < propertyInputs.length - 1) {
			pendingConditionFocusRef.current = currentIndex;
		} else if (currentIndex > 0) {
			pendingConditionFocusRef.current = currentIndex - 1;
		} else {
			pendingConditionFocusRef.current = 'add-condition';
		}

		announceConditionRemoved();
		onRemoveRule(path);
	};

	const onLogicalChange = (groupPath: number[], operator: FilterLogical) => {
		const updatedFilter = structuredClone(editableFilter);
		filterGroupAt(updatedFilter, groupPath).operator = operator;
		changeFilter(updatedFilter, true);
	};

	const updateProperty = (path: number[], value: string): Filter => {
		const updatedFilter = structuredClone(editableFilter);
		const condition = filterConditionAt(updatedFilter, path);
		const previousPropertyName = condition.property;
		const previousPropertyValues = getPropertyValues(findPropertyInSchema(previousPropertyName));
		const [, previousPath] = splitPropertyAndPath(previousPropertyName, flatInputSchema);
		const newPropertyName =
			previousPath !== '' && flatInputSchema[value]?.type === 'json' ? `${value}.${previousPath}` : value;
		const newProperty = findPropertyInSchema(newPropertyName);
		const [, newPath] = splitPropertyAndPath(newPropertyName, flatInputSchema);

		const compatibleOperators = getCompatibleFilterOperators(newProperty, newPath !== '', role, target);
		if (condition.operator !== '' && !compatibleOperators.includes(FILTER_OPERATORS.indexOf(condition.operator))) {
			condition.operator = '';
			condition.values = [''];
		}

		condition.property = newPropertyName;
		const newPropertyValues = getPropertyValues(newProperty);
		if (previousPropertyName !== newPropertyName && (previousPropertyValues != null || newPropertyValues != null)) {
			condition.values = isBetweenOperator(condition.operator) ? ['', ''] : [''];
		}
		return updatedFilter;
	};

	const onSelectProperty = (path: number[], value: string) => {
		const updatedFilter = updateProperty(path, value);
		const condition = filterConditionAt(updatedFilter, path);
		const [, propertyPath] = splitPropertyAndPath(condition.property, flatInputSchema);
		const compatibleOperators = getCompatibleFilterOperators(
			flatInputSchema[value],
			propertyPath !== '',
			role,
			target,
		);
		const currentOperatorIndex = FILTER_OPERATORS.findIndex((operator) => operator === condition.operator);
		const isJSON = flatInputSchema[value]?.type === 'json';

		if (!compatibleOperators.includes(currentOperatorIndex) && compatibleOperators.length > 0) {
			setOperator(updatedFilter, path, FILTER_OPERATORS[compatibleOperators[0]]);
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
		changeFilter(updatedFilter, true);

		if (isJSON) {
			setTimeout(() => {
				const property: any = document.querySelector(`[data-id="property-${pathID(path)}"]`);
				property?.closest('.pipeline__filters-condition')?.querySelector('.pipeline__filters-path')?.select();
			}, 10);
		}
	};

	const updatePath = (path: number[], value: string): Filter => {
		const updatedFilter = structuredClone(editableFilter);
		const condition = filterConditionAt(updatedFilter, path);
		const [base] = splitPropertyAndPath(condition.property, flatInputSchema);
		const compatibleOperators = getCompatibleFilterOperators(flatInputSchema[base], value !== '', role, target);
		if (condition.operator !== '' && !compatibleOperators.includes(FILTER_OPERATORS.indexOf(condition.operator))) {
			condition.operator = '';
			condition.values = [''];
		}
		condition.property = value === '' ? base : `${base}.${value}`;
		return updatedFilter;
	};

	const setOperator = (updatedFilter: Filter, path: number[], operator: FilterOperator) => {
		const condition = filterConditionAt(updatedFilter, path);
		const values = filterConditionValues(condition);
		condition.operator = operator;
		if (isUnaryOperator(operator)) {
			delete condition.values;
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
	};

	const changeOperator = (path: number[], operator: FilterOperator) => {
		const updatedFilter = structuredClone(editableFilter);
		setOperator(updatedFilter, path, operator);
		changeFilter(updatedFilter, true);
	};

	const onOperatorSelectClose = (event: any) => {
		const select = event.target;
		// Hand off focus only after closing, without stealing it from a reopened menu or another control.
		if (select.open || document.activeElement !== select) {
			return;
		}
		const operator = FILTER_OPERATORS[select.value];
		if (operator == null || isUnaryOperator(operator)) {
			return;
		}
		const valueInput = select
			.closest('.pipeline__filters-condition')
			?.querySelector('.pipeline__filters-value-input');
		if (valueInput == null) {
			return;
		}
		if (valueInput.tagName === 'SL-SELECT' && !valueInput.open) {
			valueInput.show();
		} else {
			valueInput.focus();
		}
	};

	const updateValue = (path: number[], position: number, value: string): Filter => {
		const updatedFilter = structuredClone(editableFilter);
		filterConditionValues(filterConditionAt(updatedFilter, path))[position] = value;
		return updatedFilter;
	};

	const onAddValue = (path: number[]) => {
		const updatedFilter = structuredClone(editableFilter);
		const condition = filterConditionAt(updatedFilter, path);
		const values = filterConditionValues(condition);
		const position = values.length;
		values.push('');
		changeFilter(updatedFilter);
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
		const updatedFilter = structuredClone(editableFilter);
		const condition = filterConditionAt(updatedFilter, path);
		filterConditionValues(condition).splice(position, 1);
		changeFilter(updatedFilter, true);
	};

	const renderCondition = (condition: FilterCondition, path: number[], conditionNumber: number): ReactNode => {
		const id = pathID(path);
		const groupPath = path.slice(0, -1);
		const isOnlyRuleInNestedGroup =
			groupPath.length > 0 && filterGroupAt(editableFilter, groupPath).rules.length === 1;
		const isRemoveDisabled = disabled || isOnlyRuleInNestedGroup;
		const hideRemoveButton =
			showEmptyFilterAsCondition &&
			groupPath.length === 0 &&
			editableFilter.rules.length === 1 &&
			condition.property === '' &&
			condition.operator === '' &&
			(condition.values == null || condition.values.every((value) => value === ''));
		const [base, propertyPath] = splitPropertyAndPath(condition.property, flatInputSchema);
		const property = flatInputSchema?.[base];
		const isUnary = isUnaryOperator(condition.operator);
		const isJSON = property?.type === 'json';
		const isBetween = isBetweenOperator(condition.operator);
		const isOneOf = isOneOfOperator(condition.operator);
		const isInvalidProperty = property == null;
		const propertyValues = getPropertyValues(property) ?? [];
		const values = condition.values ?? [];
		const propertyName = isJSON ? base : condition.property;
		const propertyDisplayValue = propertyItems.find((item) => item.term === propertyName)?.displayValue;

		const propertyInput = (
			<Combobox
				onInput={(_, value) => changeFilter(updateProperty(path, value))}
				onSelect={(_, value) => onSelectProperty(path, value)}
				value={propertyName}
				displayValue={propertyDisplayValue}
				className='pipeline__filters-property'
				size='small'
				name={`property-${id}`}
				items={propertyItems}
				isExpression={false}
				disabled={disabled}
				placeholder='Property'
				caret={true}
				controlled={true}
				autoResize={true}
				hoist={hoistMenus}
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
				onSlInput={(event: any) => changeFilter(updatePath(path, event.target.value))}
				onSlChange={(event: any) => changeFilter(updatePath(path, event.target.value), true)}
				name={`path-${id}`}
				disabled={disabled}
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
				onSlAfterHide={onOperatorSelectClose}
				placeholder='Operator'
				disabled={isInvalidProperty || disabled}
				hoist={hoistMenus}
			>
				{property != null
					? getCompatibleFilterOperators(property, propertyPath !== '', role, target).map((index) => (
							<SlOption key={index} value={String(index)}>
								{FILTER_OPERATORS[index]}
							</SlOption>
						))
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
				<FilterValueControl
					key={`value-${id}-0`}
					name={`value-${id}-0`}
					value={values[0] ?? ''}
					options={propertyValues}
					disabled={isInvalidProperty || disabled}
					hoist={hoistMenus}
					onValueChange={(value) => changeFilter(updateValue(path, 0, value))}
					onValueCommit={(value) => changeFilter(updateValue(path, 0, value), true)}
					textValueCommitDelayMs={textValueCommitDelayMs}
				/>,
			);
			if (isBetween) {
				valueElements.push(
					<span className='pipeline__filters-value-and' key='and'>
						and
					</span>,
					<FilterValueControl
						key={`value-${id}-1`}
						name={`value-${id}-1`}
						value={values[1] ?? ''}
						options={propertyValues}
						disabled={isInvalidProperty || disabled}
						hoist={hoistMenus}
						onValueChange={(value) => changeFilter(updateValue(path, 1, value))}
						onValueCommit={(value) => changeFilter(updateValue(path, 1, value), true)}
						textValueCommitDelayMs={textValueCommitDelayMs}
					/>,
				);
			} else if (isOneOf) {
				for (const [position, value] of values.slice(1).entries()) {
					const valuePosition = position + 1;
					valueElements.push(
						<div className='pipeline__filters-value' key={`value-${id}-${valuePosition}`}>
							<FilterValueControl
								name={`value-${id}-${valuePosition}`}
								value={value}
								options={propertyValues}
								disabled={isInvalidProperty || disabled}
								hoist={hoistMenus}
								onValueChange={(value) => changeFilter(updateValue(path, valuePosition, value))}
								onValueCommit={(value) => changeFilter(updateValue(path, valuePosition, value), true)}
								textValueCommitDelayMs={textValueCommitDelayMs}
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
						disabled={disabled}
						onClick={() => onAddValue(path)}
					>
						Add value
					</SlButton>,
				);
			}
		}

		const removeButton = (
			<button
				type='button'
				className='pipeline__filters-remove-condition pipeline__filters-remove-rule'
				aria-label={`Remove condition ${conditionNumber}`}
				onClick={() => onRemoveCondition(path)}
				disabled={isRemoveDisabled}
			>
				<SlIcon name='x' aria-hidden='true' />
			</button>
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
					{!hideRemoveButton && (
						<div className='pipeline__filters-remove-condition-wrapper'>
							{isRemoveDisabled ? (
								removeButton
							) : (
								<SlTooltip
									className='app-tooltip'
									content={`Remove condition ${conditionNumber}`}
									hoist={true}
								>
									{removeButton}
								</SlTooltip>
							)}
						</div>
					)}
				</div>
			</div>
		);
	};

	const renderGroup = (group: Filter, groupPath: number[], depth: number): ReactNode => {
		const isRoot = depth === 1;
		const removeGroupTooltip = group.rules.some(isFilterGroup) ? 'Remove groups' : 'Remove group';

		return (
			<div
				ref={isRoot ? rootRef : undefined}
				className={`pipeline__filters-group${isRoot ? ' pipeline__filters-group--root' : ''}`}
				key={pathID(groupPath) || 'root'}
			>
				<div className='pipeline__filters-group-header'>
					<span className='pipeline__filters-group-subject'>
						{isRoot ? `${subject} matching` : 'matching'}
					</span>
					<SlSelect
						className='pipeline__filters-logical'
						size='small'
						value={group.operator}
						onSlChange={(event: any) => onLogicalChange(groupPath, event.target.value as FilterLogical)}
						disabled={disabled}
						hoist={hoistMenus}
						aria-label='Match rules'
					>
						<SlOption value='and'>all</SlOption>
						<SlOption value='or'>any</SlOption>
					</SlSelect>
					<span>of the following:</span>
					{isRoot && rootActions != null && <div className='filter-editor__root-actions'>{rootActions}</div>}
					{!isRoot && (
						<div className='pipeline__filters-remove-group-wrapper'>
							<SlTooltip className='app-tooltip' content={removeGroupTooltip} hoist={true}>
								<SlButton
									className='pipeline__filters-remove-group pipeline__filters-remove-rule'
									size='small'
									variant='text'
									onClick={() => onRemoveRule(groupPath)}
									disabled={disabled}
								>
									<SlIcon name='x' aria-hidden='true' />
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
									{isGroup
										? renderGroup(rule, path, depth + 1)
										: renderCondition(rule, path, ++conditionNumber)}
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
						disabled={disabled || filterRuleCount >= MAX_FILTER_RULE_COUNT}
					>
						<SlIcon slot='prefix' name='plus-circle' />
						Add a condition
					</SlButton>
					<SlButton
						className='pipeline__filters-add-group'
						size='medium'
						variant='text'
						onClick={() => onAddGroup(groupPath)}
						disabled={disabled || depth >= MAX_FILTER_DEPTH || filterRuleCount > MAX_FILTER_RULE_COUNT - 2}
					>
						<SlIcon slot='prefix' name='plus-circle' />
						Add a group
					</SlButton>
				</div>
			</div>
		);
	};

	let conditionNumber = 0;

	return (
		<>
			{filter == null && !showEmptyFilterAsCondition ? null : renderGroup(editableFilter, [], 1)}
			<span className='filter-editor__announcement' role='status' aria-live='polite' aria-atomic='true'>
				{conditionRemovalAnnouncement}
			</span>
		</>
	);
};

interface FilterValueControlProps {
	name: string;
	value: string;
	options: string[];
	disabled: boolean;
	hoist: boolean;
	onValueChange: (value: string) => void;
	onValueCommit: (value: string) => void;
	textValueCommitDelayMs?: number;
	removable?: boolean;
	onRemove?: () => void;
}

const FilterValueControl = ({
	name,
	value,
	options,
	disabled,
	hoist,
	onValueChange,
	onValueCommit,
	textValueCommitDelayMs,
	removable = false,
	onRemove,
}: FilterValueControlProps) => {
	const idleCommitTimerRef = useRef<ReturnType<typeof setTimeout>>();
	const isComposingRef = useRef(false);
	const onValueCommitRef = useRef(onValueCommit);
	const scheduledValueRef = useRef<string>();
	onValueCommitRef.current = onValueCommit;

	const cancelIdleCommit = useCallback(() => {
		if (idleCommitTimerRef.current != null) {
			clearTimeout(idleCommitTimerRef.current);
			idleCommitTimerRef.current = undefined;
		}
		scheduledValueRef.current = undefined;
	}, []);

	const scheduleIdleCommit = useCallback(
		(nextValue: string) => {
			cancelIdleCommit();
			if (disabled || textValueCommitDelayMs == null) {
				return;
			}
			scheduledValueRef.current = nextValue;
			idleCommitTimerRef.current = setTimeout(() => {
				idleCommitTimerRef.current = undefined;
				scheduledValueRef.current = undefined;
				onValueCommitRef.current(nextValue);
			}, textValueCommitDelayMs);
		},
		[cancelIdleCommit, disabled, textValueCommitDelayMs],
	);

	useEffect(() => () => cancelIdleCommit(), [cancelIdleCommit]);

	useLayoutEffect(() => {
		if (disabled || textValueCommitDelayMs == null) {
			cancelIdleCommit();
		}
	}, [cancelIdleCommit, disabled, textValueCommitDelayMs]);

	useEffect(() => {
		if (scheduledValueRef.current != null && scheduledValueRef.current !== value) {
			cancelIdleCommit();
		}
	}, [cancelIdleCommit, value]);

	const handleInput = (event: any) => {
		const nextValue = event.target.value;
		onValueChange(nextValue);
		if (!isComposingRef.current) {
			scheduleIdleCommit(nextValue);
		}
	};

	const handleSelect = (event: any) => {
		cancelIdleCommit();
		const target = event.target as any;
		const v = event.detail?.value ?? target.value;
		onValueChange(v);
		onValueCommit(v);
	};

	const handleCommit = (event: any) => {
		cancelIdleCommit();
		onValueCommit(event.target.value);
	};

	const handleKeyDown = (event: React.KeyboardEvent) => {
		if (event.key === 'Enter' && !event.nativeEvent.isComposing) {
			cancelIdleCommit();
			onValueCommit((event.currentTarget as any).value);
		}
	};

	const handleCompositionStart = () => {
		isComposingRef.current = true;
		cancelIdleCommit();
	};

	const handleCompositionEnd = (event: React.CompositionEvent) => {
		isComposingRef.current = false;
		scheduleIdleCommit((event.currentTarget as any).value);
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
					hoist={hoist}
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
			onSlChange={handleCommit}
			onKeyDown={handleKeyDown}
			onCompositionStart={handleCompositionStart}
			onCompositionEnd={handleCompositionEnd}
			name={name}
			disabled={disabled}
		>
			{removeButton}
		</SlInput>
	);
};

export { FilterEditor };

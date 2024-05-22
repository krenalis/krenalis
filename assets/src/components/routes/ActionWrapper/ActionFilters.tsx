import React, { useRef, useContext, ReactNode } from 'react';
import Section from '../../base/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../base/ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import ActionContext from '../../../context/ActionContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';

const operatorOptions = {
	1: 'is',
	2: 'is not',
};

const ActionFilters = () => {
	const conditionListRef = useRef(null);

	const { action, setAction, actionType } = useContext(ActionContext);

	const onAddCondition = () => {
		const a = { ...action };
		if (a.Filter == null) {
			a.Filter = { Logical: 'all', Conditions: [] };
		}
		a.Filter.Conditions = [...a.Filter.Conditions, { Property: '', Operator: '', Value: '' }];
		setAction(a);
	};

	const onRemoveCondition = (e) => {
		const a = { ...action };
		const id = e.currentTarget.closest('.action__filters-condition').dataset.id;
		a.Filter!.Conditions.splice(id, 1);
		if (a.Filter!.Conditions.length === 0) {
			a.Filter = null;
		}
		setAction(a);
	};

	const onUpdateConditionFragment = (e) => {
		const a = { ...action };
		const id = e.target.closest('.action__filters-condition').dataset.id;
		const fragment = e.target.dataset.fragment;
		let value: string;
		if (fragment === 'Operator') {
			value = operatorOptions[e.target.value];
		} else {
			value = e.target.value;
		}
		a.Filter!.Conditions[id][fragment] = value;
		setAction(a);
	};

	const onSelectConditionListItem = (input, value) => {
		const a = { ...action };
		const id = input.closest('.action__filters-condition').dataset.id;
		a.Filter!.Conditions[id]['Property'] = value;
		setAction(a);
	};

	const onSwitchFilterLogical = () => {
		const a = { ...action };
		const logical = a.Filter!.Logical;
		if (logical === 'all') {
			a.Filter!.Logical = 'any';
		} else {
			a.Filter!.Logical = 'all';
		}
		setAction(a);
	};

	const conditions: ReactNode[] = [];
	if (action.Filter != null) {
		for (const [i, condition] of action.Filter.Conditions.entries()) {
			let conditionInput: ReactNode, operatorSelect: ReactNode, valueInput: ReactNode;
			conditionInput = (
				<ComboBoxInput
					comboBoxListRef={conditionListRef}
					onInput={onUpdateConditionFragment}
					value={condition.Property}
					className='action__filters-property'
					size='small'
					data-fragment='Property'
				/>
			);
			operatorSelect = (
				<SlSelect
					data-fragment='Operator'
					size='small'
					className='action__filters-operator'
					value={Object.keys(operatorOptions).find((key) => operatorOptions[key] === condition.Operator)}
					onSlChange={onUpdateConditionFragment}
				>
					{Object.keys(operatorOptions).map((k) => (
						<SlOption key={k} value={k}>
							{operatorOptions[k]}
						</SlOption>
					))}
				</SlSelect>
			);
			valueInput = (
				<SlInput
					data-fragment='Value'
					size='small'
					className='action__filters-value'
					value={condition.Value}
					onSlInput={onUpdateConditionFragment}
				/>
			);
			conditions.push(
				<div key={i} className='action__filters-condition' data-id={i}>
					{conditionInput}
					{operatorSelect}
					{valueInput}
					<SlButton
						className='action__filters-remove-condition'
						size='small'
						variant='danger'
						onClick={onRemoveCondition}
					>
						Remove
					</SlButton>
				</div>,
			);
		}
	}

	return (
		<Section
			className='action__filters'
			title='Filter'
			description='The filters that define the action'
			padded={true}
		>
			{conditions.length > 1 && (
				<SlSelect
					className='action__filters-logical'
					size='small'
					value={action.Filter!.Logical}
					onSlChange={onSwitchFilterLogical}
				>
					<SlOption value='all'>All</SlOption>
					<SlOption value='any'>Any</SlOption>
				</SlSelect>
			)}
			{conditions}
			<ComboBoxList
				ref={conditionListRef}
				items={getSchemaComboboxItems(actionType.InputSchema)}
				onSelect={onSelectConditionListItem}
			/>
			<SlButton className='action__filters-add-condition' size='small' variant='neutral' onClick={onAddCondition}>
				Add new condition
			</SlButton>
		</Section>
	);
};

export default ActionFilters;

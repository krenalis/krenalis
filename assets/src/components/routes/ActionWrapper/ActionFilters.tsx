import React, { useContext, ReactNode } from 'react';
import Section from '../../base/Section/Section';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboboxItems';
import ActionContext from '../../../context/ActionContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { Combobox } from '../../base/Combobox/Combobox';

const operatorOptions = {
	1: 'is',
	2: 'is not',
};

const ActionFilters = () => {
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

	const onUpdatePropertyFragment = (name: string, value: string) => {
		const a = { ...action };
		const id = Number(name.split('-')[1]);
		a.Filter!.Conditions[id]['Property'] = value;
		setAction(a);
	};

	const onSelectPropertyFragment = (name: string, value: string) => {
		const a = { ...action };
		const id = Number(name.split('-')[1]);
		a.Filter!.Conditions[id]['Property'] = value;
		setAction(a);
	};

	const onChangeOperatorFragment = (e: any) => {
		const a = { ...action };
		const id = Number(e.target.name.split('-')[1]);
		a.Filter!.Conditions[id]['Operator'] = operatorOptions[e.target.value];
		setAction(a);
	};

	const onInputValueFragment = (e: any) => {
		const a = { ...action };
		const id = Number(e.target.name.split('-')[1]);
		a.Filter!.Conditions[id]['Value'] = e.target.value;
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
				<Combobox
					onInput={onUpdatePropertyFragment}
					onSelect={onSelectPropertyFragment}
					initialValue={condition.Property}
					className='action__filters-property'
					size='small'
					name={`property-${i}`}
					items={getSchemaComboboxItems(actionType.InputSchema)}
					isExpression={false}
				/>
			);
			operatorSelect = (
				<SlSelect
					size='small'
					name={`operator-${i}`}
					className='action__filters-operator'
					value={Object.keys(operatorOptions).find((key) => operatorOptions[key] === condition.Operator)}
					onSlChange={onChangeOperatorFragment}
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
					size='small'
					className='action__filters-value'
					value={condition.Value}
					onSlInput={onInputValueFragment}
					name={`value-${i}`}
				/>
			);
			conditions.push(
				<div key={i} className='action__filters-condition'>
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
			<SlButton className='action__filters-add-condition' size='small' variant='neutral' onClick={onAddCondition}>
				Add new condition
			</SlButton>
		</Section>
	);
};

export default ActionFilters;

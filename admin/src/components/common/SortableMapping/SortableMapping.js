import './SortableMapping.css';
import { useContext, useRef, useMemo } from 'react';
import { ComboBoxInput, ComboBoxList } from '../ComboBox/ComboBox';
import { AppContext } from '../../../providers/AppProvider';
import { getSchemaComboboxItems } from '../../../helpers/getSchemaComboBoxItems';
import { SlButton, SlDropdown, SlMenu, SlMenuItem, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { flattenSchema } from '../../../lib/connections/action';

const SortableMapping = ({ api, mapping, setMapping, inputSchema, outputSchema }) => {
	const { showError } = useContext(AppContext);

	const inputPropertiesListRef = useRef(null);
	const outputPropertiesListRef = useRef(null);

	const flatOutputSchema = useMemo(() => flattenSchema(outputSchema), [outputSchema]);

	const validateExpression = async (expression, schema, destinationProperty) => {
		let errorMessage = '';
		if (expression !== '') {
			let err;
			[errorMessage, err] = await api.validateExpression(
				expression,
				schema,
				destinationProperty.type,
				destinationProperty.nullable
			);
			if (err != null) {
				showError(err);
				return;
			}
		}
		return errorMessage;
	};

	const updateInput = async (pos, value) => {
		const m = [...mapping];
		const relatedExpression = m[pos - 1][1].value;
		const destinationProperty = flatOutputSchema[relatedExpression];
		let errorMessage = '';
		if (destinationProperty) {
			errorMessage = await validateExpression(value, inputSchema, destinationProperty.full);
		}
		m[pos - 1][0].value = value;
		m[pos - 1][0].error = errorMessage;
		setMapping(m);
	};

	const updateOutput = async (pos, value) => {
		const m = [...mapping];
		m[pos - 1][1].value = value;
		setMapping(m);
		const relatedExpression = m[pos - 1][0].value;
		await updateInput(pos, relatedExpression);
	};

	const onUpdateInputProperty = async (e) => {
		const { name, value } = e.currentTarget || e.target;
		const pos = Number(name);
		await updateInput(pos, value);
	};

	const onUpdateOutputProperty = async (e) => {
		const { name, value } = e.currentTarget || e.target;
		const pos = Number(name);
		await updateOutput(pos, value);
	};

	const onSelectInputProperty = async (input, value) => {
		const pos = Number(input.name);
		await updateInput(pos, value);
	};

	const onSelectOutputProperty = async (input, value) => {
		const pos = Number(input.name);
		await updateOutput(pos, value);
	};

	const onMoveCorrelationUp = (position) => {
		const elementIndex = position - 1;
		const element = mapping[elementIndex];
		const previousElementIndex = elementIndex - 1;
		const previousElement = mapping[previousElementIndex];
		const m = [
			...mapping.slice(0, previousElementIndex),
			element,
			previousElement,
			...mapping.slice(elementIndex + 1),
		];
		setMapping(m);
	};

	const onMoveCorrelationDown = (position) => {
		const elementIndex = position - 1;
		const element = mapping[elementIndex];
		const nextElementIndex = elementIndex + 1;
		const nextElement = mapping[nextElementIndex];
		const m = [...mapping.slice(0, elementIndex), nextElement, element, ...mapping.slice(nextElementIndex + 1)];
		setMapping(m);
	};

	const onRemoveCorrelation = (position) => {
		const m = [...mapping];
		if (m.length === 1) {
			showError('You must define at least one mapping');
			return;
		}
		m.splice(position - 1, 1);
		setMapping(m);
	};

	const onAddCorrelation = () => {
		const m = [...mapping];
		m.push([{ value: '', error: '' }, { value: '' }]);
		setMapping(m);
	};

	const alreadyUsedOutputProperties = useMemo(() => {
		const used = [];
		for (const [inputProperty, outputProperty] of mapping) {
			used.push(outputProperty.value);
		}
		return used;
	}, [mapping]);

	return (
		<div className='sortableMapping'>
			{mapping.map(([input, output], i) => {
				const position = i + 1;
				return (
					<div className='correlation'>
						<div className='position'>{position}</div>
						<ComboBoxInput
							comboBoxListRef={inputPropertiesListRef}
							onInput={onUpdateInputProperty}
							value={input.value}
							name={position}
							className='inputProperty'
							size='small'
							error={input.error}
						/>
						<div className='arrow'>
							<SlIcon name='arrow-right' />
						</div>
						<ComboBoxInput
							comboBoxListRef={outputPropertiesListRef}
							onInput={onUpdateOutputProperty}
							value={output.value}
							name={position}
							className='outputProperty'
							size='small'
						/>
						<SlDropdown>
							<SlButton size='small' className='correlationMenu' slot='trigger'>
								<SlIcon slot='prefix' name='three-dots'></SlIcon>
							</SlButton>
							<SlMenu>
								<SlMenuItem onClick={() => onMoveCorrelationUp(position)} disabled={position === 1}>
									<SlIcon slot='prefix' name='arrow-up-circle' />
									Move up
								</SlMenuItem>
								<SlMenuItem
									onClick={() => onMoveCorrelationDown(position)}
									disabled={position === mapping.length}
								>
									<SlIcon slot='prefix' name='arrow-down-circle' />
									Move down
								</SlMenuItem>
								<SlMenuItem className='removeCorrelation' onClick={() => onRemoveCorrelation(position)}>
									<SlIcon slot='prefix' name='trash3' />
									Remove
								</SlMenuItem>
							</SlMenu>
						</SlDropdown>
					</div>
				);
			})}
			<SlButton className='addCorrelation' size='small' variant='neutral' onClick={onAddCorrelation} circle>
				<SlIcon name='plus' />
			</SlButton>
			<ComboBoxList
				ref={inputPropertiesListRef}
				items={getSchemaComboboxItems(inputSchema)}
				onSelect={onSelectInputProperty}
			/>
			<ComboBoxList
				ref={outputPropertiesListRef}
				items={getSchemaComboboxItems(outputSchema, alreadyUsedOutputProperties)}
				onSelect={onSelectOutputProperty}
			/>
		</div>
	);
};

export default SortableMapping;

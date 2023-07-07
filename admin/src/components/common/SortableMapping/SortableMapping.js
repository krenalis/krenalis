import './SortableMapping.css';
import { useContext, useRef, useMemo } from 'react';
import { ComboBoxInput, ComboBoxList } from '../ComboBox/ComboBox';
import { AppContext } from '../../../providers/AppProvider';
import { getSchemaComboboxItems } from '../../../helpers/getSchemaComboBoxItems';
import { SlButton, SlDropdown, SlMenu, SlMenuItem, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const SortableMapping = ({ mapping, setMapping, inputSchema, outputSchema }) => {
	const { showError } = useContext(AppContext);

	const inputPropertiesListRef = useRef(null);
	const outputPropertiesListRef = useRef(null);

	const onUpdateInputProperty = (e) => {
		const m = [...mapping];
		const { name, value } = e.currentTarget || e.target;
		const pos = Number(name);
		m[pos - 1][0] = value;
		setMapping(m);
	};

	const onUpdateOutputProperty = (e) => {
		const m = [...mapping];
		const { name, value } = e.currentTarget || e.target;
		const pos = Number(name);
		m[pos - 1][1] = value;
		setMapping(m);
	};

	const onSelectInputProperty = (input, value) => {
		const m = [...mapping];
		const pos = Number(input.name);
		m[pos - 1][0] = value;

		setMapping(m);
	};

	const onSelectOutputProperty = (input, value) => {
		const m = [...mapping];
		const pos = Number(input.name);
		m[pos - 1][1] = value;
		setMapping(m);
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
		m.push(['', '']);
		setMapping(m);
	};

	const alreadyUsedOutputProperties = useMemo(() => {
		const used = [];
		for (const [inputProperty, outputProperty] of mapping) {
			used.push(outputProperty);
		}
		return used;
	}, [mapping]);

	return (
		<div className='sortableMapping'>
			{mapping.map(([inputProperty, outputProperty], i) => {
				const position = i + 1;
				return (
					<div className='correlation'>
						<div className='position'>{position}</div>
						<ComboBoxInput
							comboBoxListRef={inputPropertiesListRef}
							onInput={onUpdateInputProperty}
							value={inputProperty}
							name={position}
							className='inputProperty'
							size='small'
						/>
						<div className='arrow'>
							<SlIcon name='arrow-right' />
						</div>
						<ComboBoxInput
							comboBoxListRef={outputPropertiesListRef}
							onInput={onUpdateOutputProperty}
							value={outputProperty}
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

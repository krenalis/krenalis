import { useRef, useContext } from 'react';
import { ComboBoxInput, ComboBoxList } from '../../../common/ComboBox/ComboBox';
import { getSchemaComboboxItems } from './Action.helpers';
import { AppContext } from '../../../../providers/AppProvider';
import Section from '../../../common/Section/Section';
import {
	SlButton,
	SlIconButton,
	SlDropdown,
	SlMenu,
	SlMenuItem,
	SlIcon,
} from '@shoelace-style/shoelace/dist/react/index.js';

const ActionIdentifiers = ({ action, setAction, inputSchema, outputSchema }) => {
	const { showError } = useContext(AppContext);

	const inputPropertiesListRef = useRef(null);
	const outputPropertiesListRef = useRef(null);

	const onUpdateInputIdentifier = (e) => {
		const { name, value } = e.currentTarget || e.target;
		const a = { ...action };
		const pos = Number(name);
		a.Identifiers[pos - 1][0] = value;
		setAction(a);
	};

	const onUpdateOutputIdentifier = (e) => {
		const { name, value } = e.currentTarget || e.target;
		const a = { ...action };
		const pos = Number(name);
		a.Identifiers[pos - 1][1] = value;
		setAction(a);
	};

	const onSelectInputIdentifier = (input, value) => {
		const a = { ...action };
		const pos = Number(input.name);
		a.Identifiers[pos - 1][0] = value;
		setAction(a);
	};

	const onSelectOutputIdentifier = (input, value) => {
		const a = { ...action };
		const pos = Number(input.name);
		a.Identifiers[pos - 1][1] = value;
		setAction(a);
	};

	const onAddNewIdentifiers = () => {
		const a = { ...action };
		a.Identifiers.push(['', '']);
		setAction(a);
	};

	const onMoveIdentifiersUp = (position) => {
		const a = { ...action };
		const elementIndex = position - 1;
		const element = a.Identifiers[elementIndex];
		const previousElementIndex = elementIndex - 1;
		const previousElement = a.Identifiers[previousElementIndex];
		a.Identifiers = [
			...a.Identifiers.slice(0, previousElementIndex),
			element,
			previousElement,
			...a.Identifiers.slice(elementIndex + 1),
		];
		console.log(a.Identifiers);
		setAction(a);
	};

	const onMoveIdentifiersDown = (position) => {
		const a = { ...action };
		const elementIndex = position - 1;
		const element = a.Identifiers[elementIndex];
		const nextElementIndex = elementIndex + 1;
		const nextElement = a.Identifiers[nextElementIndex];
		a.Identifiers = [
			...a.Identifiers.slice(0, elementIndex),
			nextElement,
			element,
			...a.Identifiers.slice(nextElementIndex + 1),
		];
		console.log(a.Identifiers);
		setAction(a);
	};

	const onRemoveIdentifiers = (position) => {
		const a = { ...action };
		if (a.Identifiers.length === 1) {
			showError('You must define at least one identifier');
			return;
		}
		a.Identifiers.splice(position - 1, 1);
		setAction(a);
	};

	return (
		<div className='actionIdentifiers'>
			<Section
				title='Identifiers'
				description='The properties used to resolve the identity of the users'
				padded={false}
			>
				{action.Identifiers.map(([inputIdentifier, outputIdentifier], i) => {
					const position = i + 1;
					return (
						<div className='identifiers'>
							<div className='position'>{position}</div>
							<ComboBoxInput
								comboBoxListRef={inputPropertiesListRef}
								onInput={onUpdateInputIdentifier}
								value={inputIdentifier}
								name={position}
								className='inputIdentifier'
								size='small'
							/>
							<div className='arrow'>
								<SlIcon name='arrow-right' />
							</div>
							<ComboBoxInput
								comboBoxListRef={outputPropertiesListRef}
								onInput={onUpdateOutputIdentifier}
								value={outputIdentifier}
								name={position}
								className='outputIdentifier'
								size='small'
							/>
							<SlDropdown>
								<SlButton size='small' className='identifiersMenu' slot='trigger'>
									<SlIcon slot='prefix' name='three-dots'></SlIcon>
								</SlButton>
								<SlMenu>
									<SlMenuItem onClick={() => onMoveIdentifiersUp(position)} disabled={position === 1}>
										<SlIcon slot='prefix' name='arrow-up-circle' />
										Move up
									</SlMenuItem>
									<SlMenuItem
										onClick={() => onMoveIdentifiersDown(position)}
										disabled={position === action.Identifiers.length}
									>
										<SlIcon slot='prefix' name='arrow-down-circle' />
										Move down
									</SlMenuItem>
									<SlMenuItem
										className='removeIdentifiers'
										onClick={() => onRemoveIdentifiers(position)}
									>
										<SlIcon slot='prefix' name='trash3' />
										Remove
									</SlMenuItem>
								</SlMenu>
							</SlDropdown>
						</div>
					);
				})}
				<SlButton
					className='addNewIdentifiers'
					size='small'
					variant='neutral'
					onClick={onAddNewIdentifiers}
					circle
				>
					<SlIcon name='plus' />
				</SlButton>
				<ComboBoxList
					ref={inputPropertiesListRef}
					items={getSchemaComboboxItems(inputSchema)}
					onSelect={onSelectInputIdentifier}
				/>
				<ComboBoxList
					ref={outputPropertiesListRef}
					items={getSchemaComboboxItems(outputSchema)}
					onSelect={onSelectOutputIdentifier}
				/>
			</Section>
		</div>
	);
};

export default ActionIdentifiers;

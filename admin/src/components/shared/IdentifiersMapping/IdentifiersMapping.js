import './IdentifiersMapping.css';
import { useRef } from 'react';
import { ComboBoxInput, ComboBoxList } from '../ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import useIdentifiersMapping from '../../../hooks/useIdentifiersMapping';
import { SlButton, SlDropdown, SlMenu, SlMenuItem, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const IdentifiersMapping = ({ mapping, setMapping, inputSchema, outputSchema }) => {
	const inputPropertiesListRef = useRef(null);
	const outputPropertiesListRef = useRef(null);

	const {
		nonSelectableProperties,
		updateMappedProperty,
		updateIdentifier,
		moveAssociationUp,
		moveAssociationDown,
		removeAssociation,
		addAssociation,
	} = useIdentifiersMapping(mapping, setMapping, inputSchema, outputSchema);

	const onUpdateMappedProperty = async (e) => {
		const { name, value } = e.target;
		const pos = Number(name);
		await updateMappedProperty(pos, value);
	};

	const onUpdateIdentifier = async (e) => {
		const { name, value } = e.target;
		const pos = Number(name);
		await updateIdentifier(pos, value);
	};

	const onSelectMappedProperty = async (input, value) => {
		const pos = Number(input.name);
		await updateMappedProperty(pos, value);
	};

	const onSelectIdentifier = async (input, value) => {
		const pos = Number(input.name);
		await updateIdentifier(pos, value);
	};

	return (
		<div className='identifiers-mapping'>
			{mapping.map(([mapped, identifier], i) => {
				const position = i + 1;
				return (
					<div className='identifiers-mapping__association'>
						<div className='identifiers-mapping__position'>{position}</div>
						<ComboBoxInput
							comboBoxListRef={inputPropertiesListRef}
							onInput={onUpdateMappedProperty}
							value={mapped.value}
							name={position}
							className='identifiers-mapping__mapped-property'
							size='small'
							error={mapped.error}
						/>
						<div className='identifiers-mapping__arrow'>
							<SlIcon name='arrow-right' />
						</div>
						<ComboBoxInput
							comboBoxListRef={outputPropertiesListRef}
							onInput={onUpdateIdentifier}
							value={identifier.value}
							name={position}
							className='identifiers-mapping__identifier'
							size='small'
							error={identifier.error}
						/>
						<SlDropdown>
							<SlButton size='small' className='identifiers-mapping__menu' slot='trigger'>
								<SlIcon slot='prefix' name='three-dots'></SlIcon>
							</SlButton>
							<SlMenu>
								<SlMenuItem onClick={() => moveAssociationUp(position)} disabled={position === 1}>
									<SlIcon slot='prefix' name='arrow-up-circle' />
									Move up
								</SlMenuItem>
								<SlMenuItem
									onClick={() => moveAssociationDown(position)}
									disabled={position === mapping.length}
								>
									<SlIcon slot='prefix' name='arrow-down-circle' />
									Move down
								</SlMenuItem>
								<SlMenuItem
									className='identifiers-mapping__remove'
									onClick={() => removeAssociation(position)}
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
				className='identifiers-mapping__add'
				size='small'
				variant='neutral'
				onClick={addAssociation}
				circle
			>
				<SlIcon name='plus' />
			</SlButton>
			<ComboBoxList
				ref={inputPropertiesListRef}
				items={getSchemaComboboxItems(inputSchema)}
				onSelect={onSelectMappedProperty}
			/>
			<ComboBoxList
				ref={outputPropertiesListRef}
				items={getSchemaComboboxItems(outputSchema, nonSelectableProperties)}
				onSelect={onSelectIdentifier}
			/>
		</div>
	);
};

export default IdentifiersMapping;

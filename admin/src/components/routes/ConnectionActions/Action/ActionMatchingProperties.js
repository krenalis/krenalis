import { useRef } from 'react';
import Section from '../../../common/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../../common/ComboBox/ComboBox';
import { getSchemaComboboxItems } from './Action.helpers';

const ActionMatchingProperties = ({ connection, action, setAction, inputSchema, outputSchema }) => {
	const internalMatchingPropertyListRef = useRef(null);
	const externalMatchingPropertyListRef = useRef(null);

	const onUpdateMatchingProperties = (e) => {
		const a = { ...action };
		a.MatchingProperties[e.target.dataset.type] = e.target.value;
		setAction(a);
	};

	const onSelectMatchingProperties = (input, value) => {
		const a = { ...action };
		a.MatchingProperties[input.dataset.type] = value;
		setAction(a);
	};

	return (
		<Section
			title={`Matching properties`}
			description='The properties used to identify and match the resources'
			padded={true}
		>
			<div className='matchingProperties'>
				<ComboBoxInput
					comboBoxListRef={internalMatchingPropertyListRef}
					onInput={onUpdateMatchingProperties}
					value={action.MatchingProperties.Internal}
					label='Golden record property'
					data-type='Internal'
					className='inputProperty'
				></ComboBoxInput>
				<ComboBoxList
					ref={internalMatchingPropertyListRef}
					items={getSchemaComboboxItems(inputSchema)}
					onSelect={onSelectMatchingProperties}
				/>
				<div className='equal'>=</div>
				<ComboBoxInput
					comboBoxListRef={externalMatchingPropertyListRef}
					onInput={onUpdateMatchingProperties}
					label={`${connection.name}'s property`}
					value={action.MatchingProperties.External}
					data-type='External'
				></ComboBoxInput>
				<ComboBoxList
					ref={externalMatchingPropertyListRef}
					items={getSchemaComboboxItems(outputSchema)}
					onSelect={onSelectMatchingProperties}
				/>
			</div>
		</Section>
	);
};

export default ActionMatchingProperties;

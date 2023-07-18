import { useRef, useContext } from 'react';
import Section from '../../../shared/Section/Section';
import { ComboBoxInput, ComboBoxList } from '../../../shared/ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../../../components/helpers/getSchemaComboBoxItems';
import { ActionContext } from '../../../../context/ActionContext';

const ActionMatchingProperties = () => {
	const internalMatchingPropertyListRef = useRef(null);
	const externalMatchingPropertyListRef = useRef(null);

	const { connection, action, setAction, actionType } = useContext(ActionContext);

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
					items={getSchemaComboboxItems(actionType.InputSchema)}
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
					items={getSchemaComboboxItems(actionType.OutputSchema)}
					onSelect={onSelectMatchingProperties}
				/>
			</div>
		</Section>
	);
};

export default ActionMatchingProperties;

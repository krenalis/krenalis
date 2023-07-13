import { useContext } from 'react';
import SortableMapping from '../../../common/SortableMapping/SortableMapping';
import Section from '../../../common/Section/Section';
import { ActionContext } from '../../../../context/ActionContext';

const ActionIdentifiers = () => {
	const { action, setAction, actionType } = useContext(ActionContext);

	const setIdentifiers = (identifiers) => {
		const a = { ...action };
		a.Identifiers = identifiers;
		setAction(a);
	};

	return (
		<div className='actionIdentifiers'>
			<Section
				title='Identifiers'
				description='The properties used to resolve the identity of the users'
				padded={false}
			>
				<SortableMapping
					mapping={action.Identifiers}
					setMapping={setIdentifiers}
					inputSchema={actionType.InputSchema}
					outputSchema={actionType.OutputSchema}
				/>
			</Section>
		</div>
	);
};

export default ActionIdentifiers;

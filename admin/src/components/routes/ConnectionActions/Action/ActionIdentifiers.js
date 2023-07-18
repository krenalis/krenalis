import { useContext } from 'react';
import SortableMapping from '../../../shared/SortableMapping/SortableMapping';
import Section from '../../../shared/Section/Section';
import { ActionContext } from '../../../../context/ActionContext';
import { AppContext } from '../../../../context/providers/AppProvider';

const ActionIdentifiers = () => {
	const { action, setAction, actionType } = useContext(ActionContext);
	const { api } = useContext(AppContext);

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
					api={api}
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

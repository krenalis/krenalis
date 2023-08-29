import React, { useContext } from 'react';
import IdentifiersMapping from '../../shared/IdentifiersMapping/IdentifiersMapping';
import Section from '../../shared/Section/Section';
import ActionContext from '../../../context/ActionContext';
import { TransformedIdentifiers } from '../../../lib/helpers/transformedIdentifiers';

const ActionIdentifiers = () => {
	const { action, setAction, actionType } = useContext(ActionContext);

	const setIdentifiers = (identifiers: TransformedIdentifiers) => {
		const a = { ...action };
		a.Identifiers = identifiers;
		setAction(a);
	};

	const onRemoveIdentifier = (identifier: string) => {
		if (identifier === '') {
			return;
		}
		const a = { ...action };
		const doesMappingExist = a.Mapping![identifier] != null;
		if (doesMappingExist) {
			a.Mapping![identifier].value = '';
			setAction(a);
		}
	};

	return (
		<div className='actionIdentifiers'>
			<Section
				title='Identifiers'
				description='The properties used to resolve the identity of the users'
				padded={false}
			>
				<IdentifiersMapping
					mapping={action.Identifiers!}
					setMapping={setIdentifiers}
					inputSchema={actionType.InputSchema}
					outputSchema={actionType.OutputSchema}
					onRemoveIdentifier={onRemoveIdentifier}
				/>
			</Section>
		</div>
	);
};

export default ActionIdentifiers;

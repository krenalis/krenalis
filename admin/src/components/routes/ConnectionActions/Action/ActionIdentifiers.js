import SortableMapping from '../../../common/SortableMapping/SortableMapping';
import Section from '../../../common/Section/Section';

const ActionIdentifiers = ({ action, setAction, inputSchema, outputSchema }) => {
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
					inputSchema={inputSchema}
					outputSchema={outputSchema}
				/>
			</Section>
		</div>
	);
};

export default ActionIdentifiers;

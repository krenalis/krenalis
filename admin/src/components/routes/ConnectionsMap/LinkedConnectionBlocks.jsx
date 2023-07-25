import ConnectionBlock from './ConnectionBlock';

const LinkedConnectionBlocks = ({ primaryConnection, primaryColumn, secondaryConnections, newConnection }) => {
	if (primaryColumn !== 'left' && primaryColumn !== 'right') return null;
	const hasSecondaryConnections = secondaryConnections != null && secondaryConnections.length > 0;

	return (
		<div
			className={`linkedConnectionBlocks${` ${primaryColumn}`}${
				hasSecondaryConnections ? ' hasSecondaryConnections' : ''
			}`}
		>
			<div className='primaryConnection'>
				<ConnectionBlock
					connection={primaryConnection}
					isNew={primaryConnection.id === newConnection}
				></ConnectionBlock>
			</div>
			{hasSecondaryConnections && (
				<>
					<div className='secondaryConnections'>
						{secondaryConnections.map((c) => (
							<ConnectionBlock connection={c} isNew={c.id === newConnection}></ConnectionBlock>
						))}
					</div>
				</>
			)}
		</div>
	);
};

export default LinkedConnectionBlocks;

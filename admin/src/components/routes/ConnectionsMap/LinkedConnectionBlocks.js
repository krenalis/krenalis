import ConnectionBlock from './ConnectionBlock';
import Arrow from '../../common/Arrow/Arrow';

const LinkedConnectionBlocks = ({
	primaryConnection,
	primaryColumn,
	secondaryConnections,
	startAnchor,
	endAnchor,
	newConnection,
}) => {
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
					<div className='arrows'>
						{secondaryConnections.map((s) => {
							return (
								<Arrow
									start={`${primaryConnection.id}`}
									end={`${s.id}`}
									startAnchor={startAnchor}
									endAnchor={endAnchor}
									showHead={false}
									color='#e4e4e7'
									strokeWidth={2}
									isNew={s.id === newConnection}
								/>
							);
						})}
					</div>
				</>
			)}
		</div>
	);
};

export default LinkedConnectionBlocks;

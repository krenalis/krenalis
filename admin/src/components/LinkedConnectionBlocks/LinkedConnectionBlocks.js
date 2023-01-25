import './LinkedConnectionBlocks.css';
import ConnectionBlock from '../ConnectionBlock/ConnectionBlock';
import Arrow from '../Arrow/Arrow';

const LinkedConnectionBlocks = ({
	primaryConnection,
	primaryColumn,
	secondaryConnections,
	startAnchor,
	endAnchor,
	newConnection,
}) => {
	if (primaryColumn !== 'left' && primaryColumn !== 'right') return null;
	let hasSecondaryConnections = secondaryConnections != null && secondaryConnections.length > 0;

	return (
		<div className={`LinkedConnectionBlocks${` ${primaryColumn}`}${hasSecondaryConnections ? ' grid' : ''}`}>
			<div className='primaryConnection'>
				<ConnectionBlock
					connection={primaryConnection}
					isNew={primaryConnection.ID === newConnection}
				></ConnectionBlock>
			</div>
			{hasSecondaryConnections && (
				<>
					<div className='secondaryConnections'>
						{secondaryConnections.map((c) => (
							<ConnectionBlock connection={c} isNew={c.ID === newConnection}></ConnectionBlock>
						))}
					</div>
					<div className='arrows'>
						{secondaryConnections.map((s) => {
							return (
								<Arrow
									start={`${primaryConnection.ID}`}
									end={`${s.ID}`}
									startAnchor={startAnchor}
									endAnchor={endAnchor}
									showHead={false}
									color='#e4e4e7'
									strokeWidth={2}
									isNew={s.ID === newConnection}
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

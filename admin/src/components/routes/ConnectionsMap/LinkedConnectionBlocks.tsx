import React from 'react';
import ConnectionBlock from './ConnectionBlock';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import { ArrowAnchor } from '../../../types/internal/app';

interface LinkedConnectionBlocksProps {
	primaryConnection: TransformedConnection;
	primaryColumn: ArrowAnchor;
	secondaryConnections: TransformedConnection[];
	newConnection: number;
}

const LinkedConnectionBlocks = ({
	primaryConnection,
	primaryColumn,
	secondaryConnections,
	newConnection,
}: LinkedConnectionBlocksProps) => {
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
							<ConnectionBlock key={c.id} connection={c} isNew={c.id === newConnection}></ConnectionBlock>
						))}
					</div>
				</>
			)}
		</div>
	);
};

export default LinkedConnectionBlocks;

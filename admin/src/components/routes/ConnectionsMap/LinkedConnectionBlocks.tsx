import React from 'react';
import ConnectionBlock from './ConnectionBlock';
import TransformedConnection from '../../../lib/core/connection';
import { ArrowAnchor } from '../../base/Arrow/Arrow.types';

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
			className={`linked-connection-block linked-connection-block--${primaryColumn}${
				hasSecondaryConnections ? ' linked-connection-block--has-secondary-connections' : ''
			}`}
		>
			<div className='linked-connection-block__primary-connections'>
				<ConnectionBlock
					connection={primaryConnection}
					isNew={primaryConnection.id === newConnection}
				></ConnectionBlock>
			</div>
			{hasSecondaryConnections && (
				<>
					<div className='linked-connection-block__secondary-connections'>
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

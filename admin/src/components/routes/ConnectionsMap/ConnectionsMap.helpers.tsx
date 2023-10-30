import React, { ReactNode } from 'react';
import LinkedConnectionBlocks from './LinkedConnectionBlocks';
import ConnectionBlock from './ConnectionBlock';
import TransformedConnection, { getStorageFileConnections } from '../../../lib/helpers/transformedConnection';

const getConnectionsBlocks = (connections: TransformedConnection[], newConnectionID: number) => {
	const blocks: ReactNode[] = [];
	for (const c of connections) {
		if (c.isFile) {
			continue;
		}
		if (c.isStorage) {
			const linkedFiles = getStorageFileConnections(c.id, connections);
			blocks.push(
				<LinkedConnectionBlocks
					key={c.id}
					primaryConnection={c}
					primaryColumn={c.isSource ? 'right' : 'left'}
					secondaryConnections={linkedFiles}
					newConnection={newConnectionID}
				></LinkedConnectionBlocks>,
			);
			continue;
		}
		blocks.push(<ConnectionBlock key={c.id} connection={c} isNew={c.id === newConnectionID}></ConnectionBlock>);
	}
	return blocks;
};

export { getConnectionsBlocks };

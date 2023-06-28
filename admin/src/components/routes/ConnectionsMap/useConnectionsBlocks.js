import { useMemo } from 'react';
import LinkedConnectionBlocks from './LinkedConnectionBlocks';
import ConnectionBlock from './ConnectionBlock';
import getStorageFileConnections from '../../../helpers/getStorageFileConnections';

const useConnectionsBlocks = (connections, newConnectionID) => {
	return useMemo(() => {
		const blocks = [];
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
						startAnchor={c.isSource ? 'left' : 'right'}
						endAnchor={c.isSource ? 'right' : 'left'}
						newConnection={newConnectionID}
					></LinkedConnectionBlocks>
				);
				continue;
			}
			blocks.push(<ConnectionBlock key={c.id} connection={c} isNew={c.id === newConnectionID}></ConnectionBlock>);
		}
		return blocks;
	}, [connections, newConnectionID]);
};

export default useConnectionsBlocks;

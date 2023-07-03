import LinkedConnectionBlocks from './LinkedConnectionBlocks';
import ConnectionBlock from './ConnectionBlock';
import getStorageFileConnections from '../../../helpers/getStorageFileConnections';

const getConnectionsBlocks = (connections, newConnectionID) => {
	const blocks = [];
	for (const c of connections) {
		if (c.isFile) {
			continue;
		}
		if (c.isStorage) {
			const linkedFiles = getStorageFileConnections(c.id, connections);
			blocks.push(
				<LinkedConnectionBlocks
					primaryConnection={c}
					primaryColumn={c.isSource ? 'right' : 'left'}
					secondaryConnections={linkedFiles}
					newConnection={newConnectionID}
				></LinkedConnectionBlocks>
			);
			continue;
		}
		blocks.push(<ConnectionBlock connection={c} isNew={c.id === newConnectionID}></ConnectionBlock>);
	}
	return blocks;
};

export { getConnectionsBlocks };

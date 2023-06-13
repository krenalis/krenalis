import LinkedConnectionBlocks from '../../components/LinkedConnectionBlocks/LinkedConnectionBlocks';
import ConnectionBlock from '../../components/ConnectionBlock/ConnectionBlock';
import Arrow from '../../components/Arrow/Arrow';

const renderConnectionsBlocks = (connections, newConnectionID) => {
	const blocks = [];
	for (const c of connections) {
		if (c.Type === 'Storage') {
			const files = connections.filter((cn) => cn.Storage === c.ID);
			blocks.push(
				<LinkedConnectionBlocks
					primaryConnection={c}
					primaryColumn={c.Role === 'Source' ? 'right' : 'left'}
					secondaryConnections={files}
					startAnchor={c.Role === 'Source' ? 'left' : 'right'}
					endAnchor={c.Role === 'Source' ? 'right' : 'left'}
					newConnection={newConnectionID}
				></LinkedConnectionBlocks>
			);
			continue;
		}
		blocks.push(<ConnectionBlock connection={c} isNew={c.ID === newConnectionID}></ConnectionBlock>);
	}
	return blocks;
};

const renderConnectionsArrows = (connections, newConnectionID) => {
	const arrows = [];
	for (let c of connections) {
		const isFile = c.Storage !== 0;
		if (isFile) {
			continue;
		}
		arrows.push(
			<Arrow
				start={`${c.ID}`}
				end='centralLogo'
				startAnchor={c.Role === 'Source' ? 'right' : 'left'}
				endAnchor={c.Role === 'Source' ? 'left' : 'right'}
				isNew={c.ID === newConnectionID}
			/>
		);
	}
	return arrows;
};

export { renderConnectionsBlocks, renderConnectionsArrows };

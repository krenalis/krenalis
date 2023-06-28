import { useMemo } from 'react';
import Arrow from '../../common/Arrow/Arrow';

const useConnectionsArrows = (connections, newConnectionID) => {
	return useMemo(() => {
		const arrows = [];
		for (const c of connections) {
			if (c.isFile) {
				continue;
			}
			arrows.push(
				<Arrow
					start={`${c.id}`}
					end='centralLogo'
					startAnchor={c.isSource ? 'right' : 'left'}
					endAnchor={c.isSource ? 'left' : 'right'}
					isNew={c.id === newConnectionID}
				/>
			);
		}
		return arrows;
	}, [connections, newConnectionID]);
};

export default useConnectionsArrows;

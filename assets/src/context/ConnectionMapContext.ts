import { createContext } from 'react';

interface ConnectionMapContext {
	hoveredConnection: number | null;
	setHoveredConnection: React.Dispatch<React.SetStateAction<number>>;
	isEventDbHovered: boolean;
	isUserDbHovered: boolean;
}

const connectionMapContext = createContext<ConnectionMapContext>({} as ConnectionMapContext);

export default connectionMapContext;

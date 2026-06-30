import { createContext } from 'react';

interface ConnectionMapContext {
	hoveredConnection: string | null;
	setHoveredConnection: React.Dispatch<React.SetStateAction<string | null>>;
	isWarehouseHovered: boolean;
}

const connectionMapContext = createContext<ConnectionMapContext>({} as ConnectionMapContext);

export default connectionMapContext;

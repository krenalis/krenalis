import { createContext } from 'react';

interface FullscreenContextType {
	closeFullscreen: () => void;
}

const FullscreenContext = createContext<FullscreenContextType | undefined>(undefined);

export { FullscreenContext };
export type { FullscreenContextType };

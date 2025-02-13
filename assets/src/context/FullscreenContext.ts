import { createContext } from 'react';

interface FullscreenContextType {
	closeFullscreen: (cb?: (...args: any) => void) => void;
}

const FullscreenContext = createContext<FullscreenContextType | undefined>(undefined);

export { FullscreenContext };
export type { FullscreenContextType };

import { createContext } from 'react';

interface KeyContextType {
	value: any;
	onChange: (...args: any) => void;
}

const KeyContext = createContext<KeyContextType | undefined>(undefined);

export { KeyContext };
export type { KeyContextType };

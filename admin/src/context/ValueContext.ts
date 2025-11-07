import { createContext } from 'react';

interface ValueContextType {
	value: any;
	onChange: (...args: any) => void;
}

const ValueContext = createContext<ValueContextType | undefined>(undefined);

export { ValueContext };
export type { ValueContextType };

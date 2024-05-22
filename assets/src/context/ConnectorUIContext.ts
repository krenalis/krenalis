import { createContext } from 'react';
import { ConnectorValues } from '../lib/api/types/responses';

interface ConnectorUIContextType {
	values: ConnectorValues;
	onChange: (name: string, value: any) => void;
}

const ConnectorUIContext = createContext<ConnectorUIContextType | undefined>(undefined);

export { ConnectorUIContext };
export type { ConnectorUIContextType };

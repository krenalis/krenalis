import { createContext } from 'react';
import { ConnectorSettings } from '../lib/api/types/responses';

interface ConnectorUIContextType {
	settings: ConnectorSettings;
	onChange: (name: string, value: any) => void;
}

const ConnectorUIContext = createContext<ConnectorUIContextType | undefined>(undefined);

export { ConnectorUIContext };
export type { ConnectorUIContextType };

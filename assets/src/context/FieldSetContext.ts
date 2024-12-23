import { createContext } from 'react';
import { ConnectorSettings } from '../lib/api/types/responses';

interface FieldSetContextType {
	settings: ConnectorSettings;
	onChange: (name: string, value: any) => void;
}

const FieldSetContext = createContext<FieldSetContextType | undefined>(undefined);

export { FieldSetContext };
export type { FieldSetContextType };

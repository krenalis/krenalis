import { createContext } from 'react';
import { ConnectorValues } from '../types/external/api';

interface FieldSetContextType {
	values: ConnectorValues;
	onChange: (name: string, value: any) => void;
}

const FieldSetContext = createContext<FieldSetContextType | undefined>(undefined);

export { FieldSetContext };
export type { FieldSetContextType };

import { createContext } from 'react';
import { UIValues } from '../types/external/api';

interface FieldSetContextType {
	values: UIValues;
	onChange: (name: string, value: any) => void;
}

const FieldSetContext = createContext<FieldSetContextType | undefined>(undefined);

export { FieldSetContext, FieldSetContextType };

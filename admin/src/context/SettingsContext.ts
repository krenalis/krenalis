import { createContext } from 'react';
import { UIValues } from '../types/external/api';

interface SettingsContextType {
	values: UIValues;
	onChange: (name: string, value: any) => void;
}

const SettingsContext = createContext<SettingsContextType | undefined>(undefined);

export { SettingsContext, SettingsContextType };

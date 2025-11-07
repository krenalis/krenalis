import { createContext } from 'react';
import { ObjectType } from '../lib/api/types/types';

interface SchemaContextType {
	schema: ObjectType;
	isLoadingSchema: boolean;
	setIsLoadingSchema: React.Dispatch<React.SetStateAction<boolean>>;
	latestAlterError: string | null;
	isAltering: boolean;
	setIsAltering: React.Dispatch<React.SetStateAction<boolean>>;
}

const SchemaContext = createContext<SchemaContextType>({} as SchemaContextType);

export { SchemaContext };

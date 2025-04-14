import { createContext } from 'react';
import { ObjectType } from '../lib/api/types/types';

interface SchemaContextType {
	schema: ObjectType;
	isLoadingSchema: boolean;
	setIsLoadingSchema: React.Dispatch<React.SetStateAction<boolean>>;
	latestUpdateError: string | null;
	isUpdating: boolean;
	setIsUpdating: React.Dispatch<React.SetStateAction<boolean>>;
}

const SchemaContext = createContext<SchemaContextType>({} as SchemaContextType);

export { SchemaContext };

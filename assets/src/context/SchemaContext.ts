import { createContext } from 'react';
import { ObjectType } from '../types/external/types';

interface SchemaContextType {
	schema: ObjectType;
	isLoadingSchema: boolean;
	setIsLoadingSchema: React.Dispatch<React.SetStateAction<boolean>>;
}

const SchemaContext = createContext<SchemaContextType>({} as SchemaContextType);

export { SchemaContext };

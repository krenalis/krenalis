import { createContext } from 'react';
import TransformedConnection from '../lib/core/connection';

interface ConnectionContextType {
	connection: TransformedConnection;
	setConnection: React.Dispatch<React.SetStateAction<TransformedConnection>>;
}

const ConnectionContext = createContext<ConnectionContextType>({} as ConnectionContextType);

export default ConnectionContext;

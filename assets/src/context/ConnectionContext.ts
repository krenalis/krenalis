import { createContext } from 'react';
import TransformedConnection from '../lib/helpers/transformedConnection';

interface ConnectionContextType {
	connection: TransformedConnection;
}

const ConnectionContext = createContext<ConnectionContextType>({} as ConnectionContextType);

export default ConnectionContext;

import React, { ReactNode, useContext, useEffect, useState } from 'react';
import TransformedConnection from '../../../lib/core/connection';
import AppContext from '../../../context/AppContext';
import { LinkedConnectionSelector } from '../../base/LinkedConnectionSelector/LinkedConnectionSelector';

interface LinkedConnectionProps {
	connection: TransformedConnection;
	title?: string;
	description?: ReactNode;
}

const LinkedConnections = ({ connection, title, description }: LinkedConnectionProps) => {
	const [linkedConnections, setLinkedConnections] = useState<Number[] | null>();

	useEffect(() => {
		setLinkedConnections(connection.linkedConnections);
	}, [connection]);

	const { connections, setIsLoadingConnections, api, handleError } = useContext(AppContext);

	const onLink = async (id: number) => {
		const [src, dst] = connection.role === 'Source' ? [connection.id, id] : [id, connection.id];
		try {
			await api.workspaces.connections.linkConnection(src, dst);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onUnlink = async (id: number) => {
		const [src, dst] = connection.role === 'Source' ? [connection.id, id] : [id, connection.id];
		try {
			await api.workspaces.connections.unlinkConnection(src, dst);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	return (
		<div className='linked-connections'>
			<LinkedConnectionSelector
				linkedConnections={linkedConnections}
				setLinkedConnections={setLinkedConnections}
				connections={connections}
				role={connection.role}
				onLink={onLink}
				onUnlink={onUnlink}
				isClickable={true}
				title={title}
				description={description != null ? description : null}
			/>
		</div>
	);
};

export { LinkedConnections };

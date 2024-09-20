import React, { useContext, useState } from 'react';
import TransformedConnection from '../../../lib/core/connection';
import AppContext from '../../../context/AppContext';
import { LinkedConnectionSelector } from '../../base/LinkedConnectionSelector/LinkedConnectionSelector';

interface LinkedConnectionProps {
	connection: TransformedConnection;
	isShown: boolean;
}

const LinkedConnections = ({ connection, isShown }: LinkedConnectionProps) => {
	const [linkedConnections, setLinkedConnections] = useState<Number[] | null>(connection.linkedConnections);

	const { connections, setIsLoadingConnections, api, handleError } = useContext(AppContext);

	const onLink = async (id: number) => {
		try {
			await api.workspaces.connections.linkConnection(connection.id, id);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onUnlink = async (id: number) => {
		try {
			await api.workspaces.connections.unlinkConnection(connection.id, id);
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
				isShown={isShown}
				description={
					connection.isSource
						? 'The destinations to which the events are dispatched.'
						: 'The sources whose events are dispatched to the destination.'
				}
			/>
		</div>
	);
};

export { LinkedConnections };

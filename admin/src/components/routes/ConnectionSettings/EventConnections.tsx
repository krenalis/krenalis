import React, { useContext, useState } from 'react';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import AppContext from '../../../context/AppContext';
import { EventConnectionSelector } from '../../shared/EventConnectionSelector/EventConnectionSelector';

interface EventConnectionProps {
	connection: TransformedConnection;
	isShown: boolean;
}

const EventConnections = ({ connection, isShown }: EventConnectionProps) => {
	const [eventConnections, setEventConnections] = useState<Number[] | null>(connection.eventConnections);

	const { connections, setIsLoadingConnections, api, handleError } = useContext(AppContext);

	const onAdd = async (id: number) => {
		try {
			await api.workspaces.connections.addEventConnection(connection.id, id);
		} catch (err) {
			handleError(err);
			throw err;
		}
		setIsLoadingConnections(true);
	};

	const onRemove = async (id: number) => {
		try {
			await api.workspaces.connections.removeEventConnection(connection.id, id);
		} catch (err) {
			handleError(err);
			throw err;
		}
		setIsLoadingConnections(true);
	};

	return (
		<div className='event-connections'>
			<EventConnectionSelector
				eventConnections={eventConnections}
				setEventConnections={setEventConnections}
				connections={connections}
				role={connection.role}
				onAdd={onAdd}
				onRemove={onRemove}
				isClickable={true}
				isShown={isShown}
				description={
					connection.isSource
						? 'The destinations to which to send the received events.'
						: 'The sources from which to receive events to send.'
				}
			/>
		</div>
	);
};

export { EventConnections };

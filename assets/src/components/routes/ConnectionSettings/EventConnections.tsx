import React, { useContext, useState } from 'react';
import TransformedConnection from '../../../lib/core/connection';
import AppContext from '../../../context/AppContext';
import { EventConnectionSelector } from '../../base/EventConnectionSelector/EventConnectionSelector';

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
			return;
		}
		setIsLoadingConnections(true);
	};

	const onRemove = async (id: number) => {
		try {
			await api.workspaces.connections.removeEventConnection(connection.id, id);
		} catch (err) {
			handleError(err);
			return;
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
						? 'The destinations to which the events are dispatched.'
						: 'The sources whose events are dispatched to the destination.'
				}
			/>
		</div>
	);
};

export { EventConnections };

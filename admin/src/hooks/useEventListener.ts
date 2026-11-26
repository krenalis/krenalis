import { useEffect, useContext, useState, useRef } from 'react';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import AppContext from '../context/AppContext';
import { Event, CreateEventListenerResponse, EventListenerEventsResponse } from '../lib/api/types/responses';
import { Filter, EventListenerConsents } from '../lib/api/types/pipeline';

interface EventListenerEvent {
	id: number;
	type: string;
	time: string;
	full: Event;
}

const useEventListener = (
	isListening: boolean,
	setEvents: (events: EventListenerEvent[]) => void,
	setOmitted?: React.Dispatch<React.SetStateAction<number>>,
	connection?: string | null,
	filter?: Filter,
	requiredConsents?: EventListenerConsents | null,
) => {
	const [isListenerNotFound, setIsListenerNotFound] = useState<boolean>(false);

	const { api, handleError, redirect } = useContext(AppContext);

	const lastEventID = useRef<number | null>(0);

	// requiredConsentsKey changes only when the content of requiredConsents
	// changes, so the listener is created again only in that case.
	const requiredConsentsKey = JSON.stringify(requiredConsents ?? null);

	useEffect(() => {
		if (!isListening) {
			return;
		}
		if (isListenerNotFound) {
			setIsListenerNotFound(false);
			return;
		}
		let isCancelled = false;
		let listenerID: string;
		let interval: number;
		const removeListener = (id: string) => api.workspaces.eventListeners.delete(id).catch(handleError);
		const startListener = async () => {
			let listener: CreateEventListenerResponse;
			try {
				listener = await api.workspaces.eventListeners.create(connection, 3, filter, requiredConsents);
			} catch (err) {
				if (isCancelled) {
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'ConnectionNotExist') {
						redirect('connections');
						handleError('The connection does not exist anymore');
						return;
					}
					if (err.code === 'TooManyListeners') {
						handleError('Please note that the number of event listeners allowed has been exceeded');
						return;
					}
				}
				handleError(err);
				return;
			}
			if (isCancelled) {
				// The listener has been stopped, or must be created again, while
				// it was being created.
				removeListener(listener.id);
				return;
			}
			listenerID = listener.id;
			interval = window.setInterval(async () => {
				let res: EventListenerEventsResponse;
				try {
					res = await api.workspaces.eventListeners.events(listenerID);
				} catch (err) {
					if (isCancelled) {
						return;
					}
					if (err instanceof NotFoundError) {
						setIsListenerNotFound(true);
						return;
					}
					handleError(err);
					return;
				}
				if (isCancelled) {
					return;
				}
				const newly: EventListenerEvent[] = [];
				for (const e of res.events) {
					newly.push({
						id: lastEventID.current,
						type: e.type,
						time: e.receivedAt,
						full: e,
					});
					const newID = lastEventID.current + 1;
					lastEventID.current = newID;
				}
				setEvents(newly);
				setOmitted && setOmitted((prevOmitted) => prevOmitted + res.omitted);
			}, 2500);
		};
		startListener();
		return () => {
			isCancelled = true;
			clearInterval(interval);
			if (listenerID != null) {
				removeListener(listenerID);
			}
		};
	}, [isListening, isListenerNotFound, requiredConsentsKey]);
};

export default useEventListener;
export { EventListenerEvent };

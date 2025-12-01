import { useEffect, useContext, useState, useRef } from 'react';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import AppContext from '../context/AppContext';
import { Event, CreateEventListenerResponse, EventListenerEventsResponse } from '../lib/api/types/responses';
import { Filter } from '../lib/api/types/pipeline';

interface EventListenerEvent {
	id: number;
	type: string;
	time: string;
	full: Event;
}

const useEventListener = (
	setEvents: (events: EventListenerEvent[]) => void,
	setOmitted?: React.Dispatch<React.SetStateAction<number>>,
	connection?: number | null,
	filter?: Filter,
) => {
	const [isStarted, setIsStarted] = useState<boolean>(false);
	const [isListenerNotFound, setIsListenerNotFound] = useState<boolean>(false);

	const { api, handleError, redirect } = useContext(AppContext);

	const eventListenerID = useRef<string | null>(null);
	const eventIntervalID = useRef<number | null>(null);
	const lastEventID = useRef<number | null>(0);

	useEffect(() => {
		if (!isStarted) {
			return;
		}
		if (isListenerNotFound) {
			setIsListenerNotFound(false);
			return;
		}
		const startListener = async () => {
			let listener: CreateEventListenerResponse;
			try {
				listener = await api.workspaces.eventListeners.create(connection, 3, filter);
			} catch (err) {
				setIsStarted(false);
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
			const listenerID = listener.id;
			eventListenerID.current = listener.id;
			const interval = window.setInterval(async () => {
				let res: EventListenerEventsResponse;
				try {
					res = await api.workspaces.eventListeners.events(listenerID);
				} catch (err) {
					if (err instanceof NotFoundError) {
						setIsListenerNotFound(true);
						return;
					}
					handleError(err);
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
			eventIntervalID.current = interval;
		};
		startListener();
		return () => {
			const removeListener = async () => {
				try {
					await api.workspaces.eventListeners.delete(eventListenerID.current);
				} catch (err) {
					handleError(err);
					return;
				}
			};
			if (eventListenerID.current == null) {
				return;
			}
			removeListener();
			clearInterval(eventIntervalID.current);
			setIsStarted(false);
		};
	}, [isStarted, isListenerNotFound]);

	const startListening = () => {
		setIsStarted(true);
	};

	const stopListening = async () => {
		if (eventListenerID.current == null) {
			return;
		}
		try {
			await api.workspaces.eventListeners.delete(eventListenerID.current);
		} catch (err) {
			handleError(err);
			return;
		}
		clearInterval(eventIntervalID.current);
		setIsStarted(false);
	};

	return { startListening, stopListening };
};

export default useEventListener;
export { EventListenerEvent };

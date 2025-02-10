import { useEffect, useContext, useState } from 'react';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import AppContext from '../context/AppContext';
import { Event, CreateEventListenerResponse, EventListenerEventsResponse } from '../lib/api/types/responses';
import { Filter } from '../lib/api/types/action';

interface EventListenerEvent {
	id: number;
	type: string;
	time: string;
	full: Event;
}

const useEventListener = (
	setEvents: (events: EventListenerEvent[]) => void,
	setOmitted?: React.Dispatch<React.SetStateAction<number>>,
	filter?: Filter,
) => {
	const [isStarted, setIsStarted] = useState<boolean>(false);
	const [isListenerNotFound, setIsListenerNotFound] = useState<boolean>(false);
	const [eventID, setEventID] = useState<number>(1);

	const { api, handleError, redirect } = useContext(AppContext);

	useEffect(() => {
		if (!isStarted) {
			return;
		}
		if (isListenerNotFound) {
			setIsListenerNotFound(false);
			return;
		}
		let listenerID: string;
		let interval: number;
		let id = eventID;
		const startListener = async () => {
			let listener: CreateEventListenerResponse;
			try {
				listener = await api.workspaces.eventListeners.create(3, filter);
			} catch (err) {
				if (err instanceof UnprocessableError) {
					if (err.code === 'ConnectionNotExists') {
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
			listenerID = listener.id;
			interval = window.setInterval(async () => {
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
						id: id,
						type: e.type,
						time: e.receivedAt,
						full: e,
					});
					const newID = id + 1;
					id = newID;
					setEventID(newID);
				}
				setEvents(newly);
				setOmitted && setOmitted((prevOmitted) => prevOmitted + res.omitted);
			}, 2500);
		};
		startListener();
		return () => {
			clearInterval(interval);
			const removeListener = async () => {
				try {
					await api.workspaces.eventListeners.delete(listenerID);
				} catch (err) {
					handleError(err);
					return;
				}
			};
			removeListener();
		};
	}, [isStarted, isListenerNotFound]);

	const startListening = () => {
		setIsStarted(true);
	};

	return { startListening };
};

export default useEventListener;
export { EventListenerEvent };

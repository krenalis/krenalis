import { useEffect, useContext, useState } from 'react';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import AppContext from '../context/AppContext';
import {
	ObservedEvent,
	AddEventListenerResponse,
	EventListenerEventsResponse,
	Filter,
} from '../lib/api/types/responses';

interface EventListenerEvent {
	id: number;
	err: string;
	type: string;
	time: string;
	source: string;
	full: ObservedEvent;
}

const useEventListener = (
	sources: number[],
	onlyValid: boolean,
	enriched: boolean,
	setEvents: (events: EventListenerEvent[]) => void,
	setDiscarded?: React.Dispatch<React.SetStateAction<number>>,
	filter?: Filter,
	listenedEventTypes?: string[],
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
			let listener: AddEventListenerResponse;
			try {
				if (enriched) {
					listener = await api.workspaces.eventlisteners.addEnriched(3, sources, filter);
				} else {
					listener = await api.workspaces.eventlisteners.addCollected(3, sources, onlyValid);
				}
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
					res = await api.workspaces.eventlisteners.events(listenerID);
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
					const dec = JSON.parse(atob(e.Data));
					if (listenedEventTypes != null) {
						if (!listenedEventTypes.includes(dec.type)) {
							continue;
						}
					}
					newly.push({
						id: id,
						err: e.Err,
						type: dec.type,
						time: e.Header.receivedAt,
						source: JSON.stringify(dec, null, 4),
						full: e,
					});
					const newID = id + 1;
					id = newID;
					setEventID(newID);
				}
				setEvents(newly);
				setDiscarded && setDiscarded((prevDiscarded) => prevDiscarded + res.discarded);
			}, 2500);
		};
		startListener();
		return () => {
			clearInterval(interval);
			const removeListener = async () => {
				try {
					await api.workspaces.eventlisteners.remove(listenerID);
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

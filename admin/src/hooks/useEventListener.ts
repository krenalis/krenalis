import { useEffect, useContext, useState, useRef } from 'react';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import AppContext from '../context/AppContext';
import { Event, CreateEventListenerResponse, EventListenerEventsResponse } from '../lib/api/types/responses';
import { Filter, EventListenerConsents } from '../lib/api/types/pipeline';
import { sleep } from '../utils/sleep';

const MAX_DELETE_RETRY_DELAY = 30000;

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
		let dismissError: (() => void) | undefined;
		const removeListener = async (id: string) => {
			let delay = 1000;
			for (;;) {
				try {
					await api.workspaces.eventListeners.delete(id);
					return;
				} catch (err) {
					// A failed fetch leaves the outcome unknown. Deleting the same listener again is safe.
					if (!(err.cause instanceof TypeError)) {
						handleError(err);
						return;
					}
				}
				if (navigator.onLine) {
					await sleep(delay);
					delay = Math.min(delay * 2, MAX_DELETE_RETRY_DELAY);
				} else {
					await new Promise<void>((resolve) =>
						window.addEventListener('online', () => resolve(), { once: true }),
					);
				}
			}
		};
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
						dismissError = handleError(
							'Please note that the number of event listeners allowed has been exceeded',
						);
						return;
					}
				}
				dismissError = handleError(err);
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
					dismissError = handleError(err);
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
			dismissError?.();
			if (listenerID != null) {
				removeListener(listenerID);
			}
		};
	}, [isListening, isListenerNotFound, requiredConsentsKey]);
};

export default useEventListener;
export { EventListenerEvent };

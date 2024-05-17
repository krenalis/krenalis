import { useEffect, useContext, useState } from 'react';
import { AddEventListenerResponse, EventListenerEventsResponse } from '../types/external/api';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import AppContext from '../context/AppContext';
import { EventListenerEvent } from '../types/internal/app';

const useEventListener = (
	connectionID: number,
	onlyValid: boolean,
	setEvents: (events: EventListenerEvent[]) => void,
	setDiscarded?: React.Dispatch<React.SetStateAction<number>>,
) => {
	const [isListenerNotFound, setIsListenerNotFound] = useState<boolean>(false);
	const [eventID, setEventID] = useState<number>(1);

	const { api, handleError, redirect } = useContext(AppContext);

	useEffect(() => {
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
				listener = await api.workspaces.eventlisteners.add(3, connectionID, onlyValid ? onlyValid : false);
			} catch (err) {
				if (err instanceof UnprocessableError) {
					if (err.code === 'ConnectionNotExists') {
						redirect('connections');
						handleError('The connection does not exist anymore');
					}
					if (err.code === 'TooManyListeners') {
						handleError('Please note that the number of event listeners allowed has been exceeded');
					}
					return;
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
	}, [isListenerNotFound]);

	return;
};

export default useEventListener;

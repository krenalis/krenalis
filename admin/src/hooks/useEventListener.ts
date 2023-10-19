import { useEffect, useContext, useState } from 'react';
import { AddEventListenerResponse, EventListenerEventsResponse } from '../types/external/api';
import { NotFoundError, UnprocessableError } from '../lib/api/errors';
import { AppContext } from '../context/providers/AppProvider';
import statuses from '../constants/statuses';
import { EventListenerEvent } from '../types/internal/app';

const useEventListener = (
	connectionID: number,
	onlyValid: boolean,
	setEvents: (events: EventListenerEvent[]) => void,
	setDiscarded?: React.Dispatch<React.SetStateAction<number>>,
) => {
	const [isListenerNotFound, setIsListenerNotFound] = useState<boolean>(false);
	const [eventID, setEventID] = useState<number>(1);

	const { api, showError, showStatus, redirect } = useContext(AppContext);

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
						showStatus(statuses.connectionDoesNotExistAnymore);
					}
					if (err.code === 'TooManyListeners') {
						showStatus(statuses.tooManyListeners);
					}
					return;
				}
				showError(err);
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
					showError(err);
					return;
				}
				const newly: EventListenerEvent[] = [];
				for (const e of res.events) {
					const dec = JSON.parse(atob(e.Data));
					newly.push({
						id: id,
						err: e.Err,
						type: dec.type,
						path: dec.properties.url,
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
					showError(err);
					return;
				}
			};
			removeListener();
		};
	}, [isListenerNotFound]);

	return;
};

export default useEventListener;

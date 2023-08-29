import React, { useEffect, useState, useContext, ReactNode } from 'react';
import './ConnectionEvents.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { github } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';
import { AddEventListenerResponse, EventListenerEventsResponse } from '../../../types/external/api';
import { EventListenerEvent } from '../../../types/internal/app';

const ConnectionEvents = () => {
	const [events, setEvents] = useState<EventListenerEvent[]>([]);
	const [selectedEvent, setSelectedEvent] = useState<number>();
	const [discarded, setDiscarded] = useState<number>(0);
	const [isListenerNotFound, setIsListenerNotFound] = useState<boolean>(false);
	const [eventID, setEventID] = useState<number>(1);

	const { api, showError, showStatus, redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	useEffect(() => {
		if (isListenerNotFound) {
			setIsListenerNotFound(false);
			return;
		}
		let listenerID: string;
		let interval: number;
		let id = eventID;
		const startListener = async () => {
			let [source, server, stream] = [0, 0, 0];
			switch (c.type) {
				case 'Server':
					server = c.id;
					break;
				case 'Stream':
					stream = c.id;
					break;
				default:
					source = c.id;
			}
			let listener: AddEventListenerResponse;
			try {
				listener = await api.eventlisteners.add(3, source, server, stream);
			} catch (err) {
				if (err instanceof UnprocessableError) {
					if (
						err.code === 'SourceNotExists' ||
						err.code === 'ServerNotExists' ||
						err.code === 'StreamNotExists'
					) {
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
			interval = setInterval(async () => {
				let res: EventListenerEventsResponse;
				try {
					res = await api.eventlisteners.events(listenerID);
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
						type: dec.event,
						path: dec.url,
						time: e.Header.receivedAt,
						full: JSON.stringify(dec, null, 4),
					});
					const newID = id + 1;
					id = newID;
					setEventID(newID);
				}
				setEvents((prevEvents) => [...prevEvents, ...newly]);
				setDiscarded((prevDiscarded) => prevDiscarded + res.discarded);
			}, 2500);
		};
		startListener();
		return () => {
			clearInterval(interval);
			const removeListener = async () => {
				try {
					await api.eventlisteners.remove(listenerID);
				} catch (err) {
					showError(err);
					return;
				}
			};
			removeListener();
		};
	}, [isListenerNotFound]);

	const onSelectEvent = (id: number) => {
		setSelectedEvent(0);
		setTimeout(() => {
			setSelectedEvent(id);
		}, 100);
	};

	let rightPanel: ReactNode;
	if (selectedEvent !== null) {
		if (selectedEvent === 0) {
			// empty panel
		} else {
			const fullEventMessage = events.find((e) => e.id === selectedEvent)?.full!;
			rightPanel = (
				<div className='fullEvent'>
					<SyntaxHighlighter language='javascript' style={github}>
						{fullEventMessage}{' '}
					</SyntaxHighlighter>
				</div>
			);
		}
	} else {
		rightPanel = (
			<div className='selectEventMessage'>
				<IconWrapper size={40} name='cursor'></IconWrapper>
				<div className='title'>Click on one event</div>
				<div className='description'>Select one of the events from the events list to see its full message</div>
			</div>
		);
	}

	const reversedEvents: EventListenerEvent[] = [...events].reverse();

	return (
		<div className='connectionEvents'>
			<div className='events'>
				<div className='eventList'>
					<div className='heading'>
						<div className='title'>
							<IconWrapper name='activity' moat />
							<div className='text'>Live events</div>
						</div>
						<div className='discarded'>
							<span className='count'>{discarded}</span>
							<span className='text'>discarded</span>
						</div>
					</div>
					<div className='body'>
						{events.length === 0 && (
							<div className='noEvents'>
								Listening for new events{' '}
								<span className='loadingEllipsis'>
									<span className='ellipsis1'>.</span>
									<span className='ellipsis2'>.</span>
									<span className='ellipsis3'>.</span>
								</span>
							</div>
						)}
						{reversedEvents.map((e) => {
							return (
								<div
									className={`event${selectedEvent === e.id ? ' selected' : ''}`}
									onClick={() => onSelectEvent(e.id)}
								>
									<div className='name'>{e.type}</div>
									<div className='path'>{e.path}</div>
									<div className='time'>{e.time}</div>
									<div className='error'>
										{e.err !== '' ? (
											<SlTooltip content={e.err} placement='top'>
												<SlIcon className='iconError' name='exclamation-circle-fill'></SlIcon>
											</SlTooltip>
										) : (
											<SlTooltip content='No error' placement='top'>
												<SlIcon className='iconSuccess' name='check-circle-fill'></SlIcon>
											</SlTooltip>
										)}
									</div>
								</div>
							);
						})}
					</div>
				</div>
			</div>
			<div className={`panel${selectedEvent !== null ? ' selected' : ' unselected'}`}>{rightPanel}</div>
		</div>
	);
};

export default ConnectionEvents;

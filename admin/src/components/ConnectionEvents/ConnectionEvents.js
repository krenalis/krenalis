import { useEffect, useState, useContext } from 'react';
import './ConnectionEvents.css';
import IconWrapper from '../IconWrapper/IconWrapper';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import statuses from '../../constants/statuses';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { github } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionEvents = () => {
	let [events, setEvents] = useState([]);
	let [selectedEvent, setSelectedEvent] = useState(null);
	let [discarded, setDiscarded] = useState(0);
	let [isListenerNotFound, setIsListenerNotFound] = useState(false);

	let { API, showError, showStatus, redirect } = useContext(AppContext);
	let { connection: c, setCurrentConnectionSection } = useContext(ConnectionContext);

	setCurrentConnectionSection('events');

	useEffect(() => {
		if (isListenerNotFound) {
			setIsListenerNotFound(false);
			return;
		}
		let listenerID;
		let interval;
		let id = 1;
		const startListener = async () => {
			let [source, server, stream] = [0, 0, 0];
			switch (c.Type) {
				case 'Server':
					server = c.ID;
					break;
				case 'Stream':
					stream = c.ID;
					break;
				default:
					source = c.ID;
			}
			let [listener, err] = await API.eventlisteners.add(3, source, server, stream);
			if (err) {
				if (err instanceof UnprocessableError) {
					if (
						err.code === 'SourceNotExist' ||
						err.code === 'ServerNotExist' ||
						err.code === 'StreamNotExist'
					) {
						redirect('/admin/connections');
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
				let [res, err] = await API.eventlisteners.events(listenerID);
				if (err) {
					if (err instanceof NotFoundError) {
						setIsListenerNotFound(true);
						return;
					}
					showError(err);
					return;
				}
				let newly = [];
				for (let e of res.events) {
					let dec = JSON.parse(atob(e.Data));
					newly.push({
						id: id,
						err: e.Err,
						type: dec.event,
						path: dec.url,
						time: e.Header.receivedAt,
						full: JSON.stringify(dec, null, 4),
					});
					id += 1;
				}
				setEvents((prevEvents) => [...prevEvents, ...newly]);
				setDiscarded((prevDiscarded) => prevDiscarded + res.discarded);
			}, 2500);
		};
		startListener();
		return async () => {
			clearInterval(interval);
			let [, err] = await API.eventlisteners.remove(listenerID);
			if (err) {
				showError(err);
				return;
			}
		};
	}, [isListenerNotFound]);

	const onSelectEvent = (id) => {
		setSelectedEvent(0);
		setTimeout(() => {
			setSelectedEvent(id);
		}, 100);
	};

	let rightPanel;
	if (selectedEvent !== null) {
		if (selectedEvent === 0) {
			// empty panel
		} else {
			let fullEventMessage = events.find((e) => e.id === selectedEvent).full;
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

	return (
		<div className='ConnectionEvents'>
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
						{[...events].reverse().map((e) => {
							return (
								<div
									class={`event${selectedEvent === e.id ? ' selected' : ''}`}
									onClick={() => onSelectEvent(e.id)}
								>
									<div class='name'>{e.type}</div>
									<div class='path'>{e.path}</div>
									<div class='time'>{e.time}</div>
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

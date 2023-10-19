import React, { useState, useContext, ReactNode } from 'react';
import './ConnectionEvents.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import SyntaxHighlight from '../../shared/SyntaxHighlight/SyntaxHighlight';
import useEventListener from '../../../hooks/useEventListener';
import { EventListenerEvent } from '../../../types/internal/app';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

const ConnectionEvents = () => {
	const [events, setEvents] = useState<EventListenerEvent[]>([]);
	const [selectedEvent, setSelectedEvent] = useState<EventListenerEvent>(null);
	const [discarded, setDiscarded] = useState<number>(0);

	const { connection: c } = useContext(ConnectionContext);

	const collectEvents = (newly: EventListenerEvent[]) => {
		setEvents((prevEvents) => [...prevEvents, ...newly]);
	};

	useEventListener(c.id, true, collectEvents, setDiscarded);

	const onEventClick = (event: EventListenerEvent) => {
		setSelectedEvent(null);
		setTimeout(() => {
			setSelectedEvent(event);
		}, 100);
	};

	let rightPanel: ReactNode;
	if (selectedEvent !== null) {
		const fullEventMessage = selectedEvent.source!;
		rightPanel = (
			<div className='fullEvent'>
				<SyntaxHighlight>{fullEventMessage}</SyntaxHighlight>
			</div>
		);
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
				<div className='heading'>
					<div className='title'>
						<IconWrapper name='activity' moat />
						<div className='text'>Live events</div>
					</div>
				</div>
				<div className='eventListener'>
					{discarded > 0 && (
						<div className='discarded'>
							<span className='count'>{discarded}</span>
							<span className='text'>discarded</span>
						</div>
					)}
					<div className='eventList'>
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
										className={`event${
											selectedEvent && selectedEvent.id === e.id ? ' selected' : ''
										}`}
										onClick={() => onEventClick(e)}
									>
										<div className='error'>
											{e.err !== '' ? (
												<SlTooltip content={e.err} placement='top' hoist>
													<SlIcon
														className='iconError'
														name='exclamation-circle-fill'
													></SlIcon>
												</SlTooltip>
											) : (
												<SlTooltip content='No error' placement='top' hoist>
													<SlIcon className='iconSuccess' name='check-circle-fill'></SlIcon>
												</SlTooltip>
											)}
										</div>
										<div className='name'>{e.type}</div>
										<div className='time'>{new Date(e.time).toLocaleString()}</div>
									</div>
								);
							})}
						</div>
					</div>
				</div>
			</div>
			<div className={`panel${selectedEvent !== null ? ' selected' : ' unselected'}`}>{rightPanel}</div>
		</div>
	);
};

export default ConnectionEvents;

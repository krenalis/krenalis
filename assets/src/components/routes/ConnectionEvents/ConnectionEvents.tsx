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
			<div className='connection-events__full-event'>
				<SyntaxHighlight>{fullEventMessage}</SyntaxHighlight>
			</div>
		);
	} else {
		rightPanel = (
			<div className='connection-events__select-event-message'>
				<IconWrapper size={40} name='cursor'></IconWrapper>
				<div className='connection-events__select-event-title'>Click on one event</div>
				<div className='connection-events__select-event-description'>
					Select one of the events from the events list to see its full message
				</div>
			</div>
		);
	}

	const reversedEvents: EventListenerEvent[] = [...events].reverse();

	return (
		<div className='connection-events'>
			<div className='connection-events__events'>
				<div className='connection-events__heading'>
					<div className='connection-events__title'>
						<IconWrapper name='activity' moat />
						<div className='connection-events__text'>Live events</div>
					</div>
				</div>
				<div className='connection-events__event-listener'>
					{discarded > 0 && (
						<div className='connection-events__discarded'>
							<span className='connection-events__discarded-count'>{discarded}</span>
							<span className='connection-events__discarded-text'>discarded</span>
						</div>
					)}
					<div className='connection-events__event-list'>
						<div className='connection-events__event-list-body'>
							{events.length === 0 && (
								<div className='connection-events__no-events'>
									Listening for new events{' '}
									<span className='connection-events__loading-ellipsis'>
										<span className='connection-events__ellipsis1'>.</span>
										<span className='connection-events__ellipsis2'>.</span>
										<span className='connection-events__ellipsis3'>.</span>
									</span>
								</div>
							)}
							{reversedEvents.map((e) => {
								return (
									<div
										key={e.id}
										className={`connection-events__event${
											selectedEvent && selectedEvent.id === e.id
												? ' connection-events__event--selected'
												: ''
										}`}
										onClick={() => onEventClick(e)}
									>
										<div className='connection-events__event-error'>
											{e.err !== '' ? (
												<SlTooltip content={e.err} placement='top' hoist>
													<SlIcon
														className='connection-events__error-icon'
														name='exclamation-circle-fill'
													></SlIcon>
												</SlTooltip>
											) : (
												<SlTooltip content='No error' placement='top' hoist>
													<SlIcon
														className='connection-events__success-icon'
														name='check-circle-fill'
													></SlIcon>
												</SlTooltip>
											)}
										</div>
										<div className='connection-events__event-name'>{e.type}</div>
										<div className='connection-events__event-time'>
											{new Date(e.time).toLocaleString()}
										</div>
									</div>
								);
							})}
						</div>
					</div>
				</div>
			</div>
			<div
				className={`connection-events__panel${selectedEvent !== null ? ' connection-events__panel--selected' : ' connection-events__panel--unselected'}`}
			>
				{rightPanel}
			</div>
		</div>
	);
};

export default ConnectionEvents;

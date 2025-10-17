import React, { useState, useEffect, useContext, ReactNode } from 'react';
import './ConnectionEvents.css';
import IconWrapper from '../../base/IconWrapper/IconWrapper';
import ConnectionContext from '../../../context/ConnectionContext';
import SyntaxHighlight from '../../base/SyntaxHighlight/SyntaxHighlight';
import useEventListener from '../../../hooks/useEventListener';
import { EventListenerEvent } from '../../../hooks/useEventListener';
import JSONbig from 'json-bigint';
import SlRelativeTime from '@shoelace-style/shoelace/dist/react/relative-time/index.js';

const ConnectionEvents = () => {
	const [events, setEvents] = useState<EventListenerEvent[]>([]);
	const [selectedEvent, setSelectedEvent] = useState<EventListenerEvent>(null);
	const [omitted, setOmitted] = useState<number>(0);
	const [hideContent, setHideContent] = useState<boolean>(false);

	const { connection: c } = useContext(ConnectionContext);

	const collectEvents = (newly: EventListenerEvent[]) => {
		for (const e of newly) {
			// Remove the connection of the event, since it should not be displayed in the UI.
			delete e.full['connectionId'];
		}
		setEvents((prevEvents) => [...prevEvents, ...newly]);
	};

	const { startListening } = useEventListener(collectEvents, setOmitted, c.id);

	useEffect(() => {
		startListening();
	}, []);

	const onEventClick = (event: EventListenerEvent) => {
		if (event.id === selectedEvent?.id) {
			// The user has clicked on the same event that is already
			// selected.
			return;
		}
		// Show the event content with a fade-in animation to highlight
		// the transition from one event to the other.
		setHideContent(true);
		setSelectedEvent(event);
		setTimeout(() => {
			setHideContent(false);
		}, 200);
	};

	let rightPanel: ReactNode;
	if (selectedEvent !== null) {
		const fullEventMessage = JSONbig.stringify(selectedEvent.full, null, 4);
		rightPanel = (
			<div
				className={`connection-events__full-event${hideContent ? ' connection-events__full-event--hide' : ''}`}
			>
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
					{omitted > 0 && (
						<div className='connection-events__omitted'>
							<span className='connection-events__omitted-count'>{omitted}</span>
							<span className='connection-events__omitted-text'>omitted</span>
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
										<div className='connection-events__event-type'>
											{e.type}
											{e.type === 'track' && (
												<span className='connection-events__event-name'>{e.full.event}</span>
											)}
										</div>
										<div className='connection-events__event-time'>
											<SlRelativeTime date={e.time} sync lang='en-US' />
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

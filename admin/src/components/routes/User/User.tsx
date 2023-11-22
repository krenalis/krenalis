/* eslint-disable no-restricted-globals */
import React, { useContext, useEffect, useState, useRef, ReactNode, Fragment } from 'react';
import './User.css';
import { adminBasePath } from '../../../constants/path';
import UsersContext from '../../../context/UsersContext';
import AppContext from '../../../context/AppContext';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSkeleton from '@shoelace-style/shoelace/dist/react/skeleton/index.js';
import LittleLogo from '../../shared/LittleLogo/LittleLogo';
import { User as UserInterface, UserEvent } from '../../../types/external/user';
import { UserEventsResponse, userTraitsResponse } from '../../../types/external/api';

const MAX_FETCH_TIME = 200;

const User = () => {
	const [user, setUser] = useState<UserInterface | null>(null);

	const { userIDList } = useContext(UsersContext);
	const { api, showError, showStatus, redirect, setTitle, connections } = useContext(AppContext);

	const fetchTimeoutID = useRef<number | undefined>();

	useEffect(() => {
		if (user == null) {
			setTitle(<SlSkeleton effect='pulse' className='userTitleSkeleton'></SlSkeleton>);
		} else {
			setTitle(
				<div className='userTitleText'>
					<SlIcon name='person-circle' />
					<span className='text'>
						{user.traits.first_name} {user.traits.last_name}
					</span>
				</div>,
			);
		}
	}, [user]);

	useEffect(() => {
		fetchUser();
		return () => {
			clearTimeout(fetchTimeoutID.current);
		};
	}, []);

	const fetchUser = async () => {
		const urlFragments = String(window.location).split('/');
		const fragmentIndex = urlFragments.findIndex((f) => f === 'users');
		const userID = Number(urlFragments[fragmentIndex + 1]);

		// Show the skeletons if the response is slow.
		let isLoading = false;
		fetchTimeoutID.current = window.setTimeout(() => {
			clearTimeout(fetchTimeoutID.current);
			isLoading = true;
			setUser(null);
		}, MAX_FETCH_TIME + 1);

		// Fetch the user's events.
		let eventsResponse: UserEventsResponse;
		try {
			eventsResponse = await api.workspaces.users.events(userID);
		} catch (err) {
			if (err instanceof NotFoundError) {
				showStatus(statuses.usersNotFound);
				redirect('users');
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'NoWarehouse') {
					showStatus(statuses.noWarehouse);
					return;
				}
				if (err.code === 'DataWarehouseFailed') {
					showStatus(statuses.dataWarehouseFailed);
					return;
				}
			}
			showError(err);
			return;
		}

		const enrichedEvents: UserEvent[] = [];
		for (const event of eventsResponse.events) {
			const e = { ...event };
			const conn = connections.find((c) => c.id === event.source);
			if (conn != null) {
				e.logo = getConnectorLogo(conn.connector.icon);
			} else {
				e.logo = <LittleLogo icon='' />;
			}
			enrichedEvents.push(e);
		}

		// Fetch the user's traits.
		let traitsResponse: userTraitsResponse;
		try {
			traitsResponse = await api.workspaces.users.traits(userID);
		} catch (err) {
			if (err instanceof NotFoundError) {
				showStatus(statuses.usersNotFound);
				redirect('users');
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'NoUsersSchema') {
					showStatus(statuses.noUsersSchema);
					return;
				}
				if (err.code === 'NoWarehouse') {
					showStatus(statuses.noWarehouse);
					return;
				}
				if (err.code === 'DataWarehouseFailed') {
					showStatus(statuses.dataWarehouseFailed);
					return;
				}
			}
			showError(err);
			return;
		}

		const u: UserInterface = {
			id: userID,
			events: enrichedEvents,
			traits: { ...traitsResponse.traits },
		};

		clearTimeout(fetchTimeoutID.current);

		if (isLoading) {
			// If the skeletons are showing, delay the rendering to prevent
			// flashes of content.
			setTimeout(() => {
				setUser(u);
			}, 300);
		} else {
			setUser(u);
		}
	};

	const onNavigate = async (direction: 'previous' | 'next') => {
		const urlFragments = String(window.location).split('/');
		const fragmentIndex = urlFragments.findIndex((f) => f === 'users');
		const userID = Number(urlFragments[fragmentIndex + 1]);
		const i = userIDList.findIndex((id) => id === userID);
		let navigationID;
		if (direction === 'previous') {
			if (i - 1 < 0) {
				navigationID = userIDList[userIDList.length - 1];
			} else {
				navigationID = userIDList[i - 1];
			}
		} else if (direction === 'next') {
			if (i + 1 >= userIDList.length) {
				navigationID = userIDList[0];
			} else {
				navigationID = userIDList[i + 1];
			}
		}
		history.pushState({}, '', `${adminBasePath}users/${navigationID}`);
		await fetchUser();
	};

	const traits: ReactNode[] = [];
	if (user != null) {
		for (const trait in user.traits) {
			let value = user.traits[trait];
			if (typeof value === 'object') {
				value = JSON.stringify(value);
			}
			traits.push(
				<Fragment key={trait}>
					<span className='label'>{trait}</span> <span className='value'>{value}</span>
				</Fragment>,
			);
		}
	}

	const events: ReactNode[] = [];
	if (user != null) {
		for (const event of user.events) {
			const conn = connections.find((c) => c.id === event.source);
			let logo: ReactNode;
			if (conn != null) {
				logo = getConnectorLogo(conn.connector.icon);
			} else {
				logo = <LittleLogo icon='' />;
			}
			events.push(
				<Fragment key={event.receivedAt}>
					<div className='eventLogo'>{logo}</div>
					<div className='eventType'>{event.type}</div>
					<div className='eventSentAt'>{event.sentAt}</div>
				</Fragment>,
			);
		}
	}

	const avatarSkeleton = <SlSkeleton effect='pulse' className='avatarSkeleton' />;
	const nameSkeleton = <SlSkeleton effect='pulse' className='nameSkeleton' />;
	const emailSkeleton = <SlSkeleton effect='pulse' className='emailSkeleton' />;
	const otherTraitsSkeleton = (
		<div className='otherTraitsSkeleton'>
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
		</div>
	);
	const eventsSkeleton = (
		<div className='eventsSkeleton'>
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
			<SlSkeleton effect='pulse' />
		</div>
	);

	return (
		<div className='user'>
			<div className='navigation'>
				<SlButton variant='text' onClick={() => onNavigate('previous')}>
					<SlIcon name='chevron-left' slot='prefix' />
				</SlButton>
				<SlButton variant='text' onClick={() => onNavigate('next')}>
					<SlIcon name='chevron-right' slot='suffix' />
				</SlButton>
			</div>
			<div className='traits'>
				<h2>Traits</h2>
				<div className='head'>
					<div className='avatar'>{user == null ? avatarSkeleton : <div className='avatarImage'>?</div>}</div>
					<div className='name'>
						{user == null ? (
							nameSkeleton
						) : (
							<div className='nameText'>
								{user.traits.first_name} {user.traits.last_name}
							</div>
						)}
					</div>
					<div className='email'>
						{user == null ? emailSkeleton : <div className='emailText'>{user.traits.email}</div>}
					</div>
				</div>
				{user == null ? (
					otherTraitsSkeleton
				) : traits.length > 0 ? (
					<div className='otherTraits'>{traits}</div>
				) : (
					<div className='noOtherTraits'>No other traits to show</div>
				)}
			</div>
			<div className='events'>
				<h2>Events</h2>
				{user == null ? (
					eventsSkeleton
				) : events.length > 0 ? (
					<div className='eventsList'>{events}</div>
				) : (
					<div className='noEvents'>No events associated to this user</div>
				)}
			</div>
		</div>
	);
};

export default User;

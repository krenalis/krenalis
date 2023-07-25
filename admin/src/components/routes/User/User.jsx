/* eslint-disable no-restricted-globals */
import { useContext, useEffect, useState, useRef } from 'react';
import './User.css';
import { adminBasePath } from '../../../constants/path';
import { UsersContext } from '../../../context/UsersContext';
import { AppContext } from '../../../context/providers/AppProvider';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import { SlIcon, SlButton, SlSkeleton } from '@shoelace-style/shoelace/dist/react/index.js';

const MAX_FETCH_TIME = 200;

const User = () => {
	const [user, setUser] = useState(null);

	const { userIDList } = useContext(UsersContext);
	const { api, showError, showStatus, redirect, setTitle } = useContext(AppContext);

	const fetchTimeoutID = useRef(null);

	useEffect(() => {
		if (user == null) {
			setTitle(<SlSkeleton effect='pulse' className='userTitleSkeleton'></SlSkeleton>);
		} else {
			setTitle(
				<div className='userTitleText'>
					<SlIcon name='person-circle' />
					<span className='text'>
						{user.traits.FirstName} {user.traits.LastName}
					</span>
				</div>
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
		const u = {
			id: userID,
		};

		// Show the skeletons if the response is slow.
		let isLoading = false;
		fetchTimeoutID.current = setTimeout(() => {
			clearTimeout(fetchTimeoutID.current);
			isLoading = true;
			setUser(null);
		}, MAX_FETCH_TIME + 1);

		let err, res;

		// Fetch the user's events.
		[res, err] = await api.users.events(userID);
		if (err != null) {
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
				if (err.code === 'WarehouseFailed') {
					showStatus(statuses.warehouseConnectionFailed);
					return;
				}
			}
			showError(err);
			return;
		}
		u.events = { ...res.events };

		// Fetch the user's traits.
		[res, err] = await api.users.traits(userID);
		if (err != null) {
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
				if (err.code === 'WarehouseFailed') {
					showStatus(statuses.warehouseConnectionFailed);
					return;
				}
			}
			showError(err);
			return;
		}
		u.traits = { ...res.traits };

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

	const onNavigate = async (direction) => {
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

	const traits = [];
	if (user != null) {
		for (const trait in user.traits) {
			let value = user.traits[trait];
			if (typeof value === 'object') {
				value = JSON.stringify(value);
			}
			traits.push(
				<>
					<span className='label'>{trait}</span> <span className='value'>{value}</span>
				</>
			);
		}
	}

	const events = [];
	if (user != null) {
		for (const event in user.events) {
			let value = user.events[event];
			if (typeof value === 'object') {
				value = JSON.stringify(value);
			}
			events.push(
				<div className='event'>
					{event}: {value}
				</div>
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
								{user.traits.FirstName} {user.traits.LastName}
							</div>
						)}
					</div>
					<div className='email'>
						{user == null ? emailSkeleton : <div className='emailText'>{user.traits.Email}</div>}
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
					<div className='events'>events</div>
				) : (
					<div className='noEvents'>No events associated to this user</div>
				)}
			</div>
		</div>
	);
};

export default User;

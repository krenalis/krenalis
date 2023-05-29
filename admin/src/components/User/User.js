import { useContext, useEffect, useState } from 'react';
import './User.css';
import { NavigationContext } from '../../context/NavigationContext';
import { UsersContext } from '../../context/UsersContext';
import { AppContext } from '../../context/AppContext';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import statuses from '../../constants/statuses';
import { useNavigate } from 'react-router';
import { SlIcon, SlButton, SlSkeleton } from '@shoelace-style/shoelace/dist/react/index.js';

const User = () => {
	let [user, setUser] = useState(null);

	let { usersRows } = useContext(UsersContext);
	let { API, showError, showStatus, redirect } = useContext(AppContext);

	let { setCurrentTitle } = useContext(NavigationContext);

	const navigate = useNavigate();

	useEffect(() => {
		if (user == null) {
			setCurrentTitle(<SlSkeleton effect='pulse' className='userTitleSkeleton'></SlSkeleton>);
		} else {
			setCurrentTitle(
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
		const fetchData = async () => {
			let urlFragments = String(window.location).split('/');
			let fragmentIndex = urlFragments.findIndex((f) => f === 'users');
			let userID = Number(urlFragments[fragmentIndex + 1]);
			let u = {
				id: userID,
			};

			let err, res;

			// Fetch the user's events.
			[res, err] = await API.users.events(userID);
			if (err != null) {
				if (err instanceof NotFoundError) {
					showStatus(statuses.usersNotFound);
					redirect('/admin/users');
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
			[res, err] = await API.users.traits(userID);
			if (err != null) {
				if (err instanceof NotFoundError) {
					showStatus(statuses.usersNotFound);
					redirect('/admin/users');
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
			setTimeout(() => {
				setUser(u);
			}, 200);
		};
		if (user == null) {
			fetchData();
		}
	}, [user]);

	const onNavigate = (direction) => {
		setUser(null);
		let urlFragments = String(window.location).split('/');
		let fragmentIndex = urlFragments.findIndex((f) => f === 'users');
		let userID = Number(urlFragments[fragmentIndex + 1]);
		let i = usersRows.findIndex((row) => row.cells[0] === userID);
		let navigationID;
		if (direction === 'previous') {
			if (i - 1 < 0) {
				navigationID = usersRows[usersRows.length - 1].cells[0];
			} else {
				navigationID = usersRows[i - 1].cells[0];
			}
		} else if (direction === 'next') {
			if (i + 1 >= usersRows.length) {
				navigationID = usersRows[0].cells[0];
			} else {
				navigationID = usersRows[i + 1].cells[0];
			}
		}
		navigate(`/admin/users/${navigationID}`);
	};

	let traits = [];
	if (user != null) {
		for (let trait in user.traits) {
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

	let events = [];
	if (user != null) {
		for (let event in user.events) {
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
				<SlButton variant='default' onClick={() => onNavigate('previous')}>
					<SlIcon name='chevron-left' slot='prefix' />
					Previous
				</SlButton>
				<SlButton variant='default' onClick={() => onNavigate('next')}>
					<SlIcon name='chevron-right' slot='suffix' />
					Next
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

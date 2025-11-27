import React, { ReactNode, useContext, useLayoutEffect } from 'react';
import './ProfileUnification.css';
import ListTile from '../../base/ListTile/ListTile';
import { Link } from '../../base/Link/Link';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Outlet, useLocation } from 'react-router-dom';
import appContext from '../../../context/AppContext';

const ProfileUnification = () => {
	const { setTitle } = useContext(appContext);

	const location = useLocation();

	useLayoutEffect(() => {
		if (location.pathname.endsWith('profile-unification')) {
			setTitle('Profile Unification');
		}
	}, [location.pathname, setTitle]);

	let content: ReactNode;

	if (location.pathname.endsWith('profile-unification')) {
		content = (
			<div className='profile-unification__content'>
				<Link path='profile-unification/profiles'>
					<ListTile
						className='profile-unification__item'
						icon={<SlIcon name='people' />}
						name='Profiles'
						description='Browse and manage unified profiles'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='profile-unification/schema'>
					<ListTile
						className='profile-unification__item'
						icon={<SlIcon name='bookmark-check' />}
						name='Schema'
						description='Explore and alter the profile schema'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='profile-unification/rules'>
					<ListTile
						className='profile-unification__item'
						icon={<SlIcon name='sliders2' />}
						name='Rules'
						description='Configure the rules used to unify identities'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
			</div>
		);
	} else {
		content = <Outlet />;
	}

	return (
		<div className='profile-unification'>
			<div className='route-content'>{content}</div>
		</div>
	);
};

export { ProfileUnification };

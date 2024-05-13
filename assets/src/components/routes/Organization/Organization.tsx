import React, { useContext, useLayoutEffect } from 'react';
import './Organization.css';
import ListTile from '../../shared/ListTile/ListTile';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Link } from '../../shared/Link/Link';

const Organization = () => {
	const { setTitle } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Organization');
	}, []);

	return (
		<div className='organization__content'>
			<div className='organization'>
				<p className='organization__title'>Organization</p>
				<Link path='members'>
					<ListTile
						className='organization__entry'
						icon={<SlIcon name='people' />}
						name={'Members'}
						description='View and modify your organization members'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
			</div>
		</div>
	);
};

export default Organization;

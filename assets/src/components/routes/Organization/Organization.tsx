import React, { useContext, useLayoutEffect } from 'react';
import './Organization.css';
import ListTile from '../../shared/ListTile/ListTile';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

const Organization = () => {
	const { redirect, setTitle } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Organization');
	}, []);

	const onMembersClick = () => {
		redirect('members');
		return;
	};

	return (
		<div className='organization__content'>
			<div className='organization'>
				<p className='organization__title'>Organization</p>
				<ListTile
					className='organization__entry'
					icon={<SlIcon name='people' />}
					name={'Members'}
					description='View and modify your organization members'
					onClick={onMembersClick}
					action={<SlIcon name='chevron-right' />}
				/>
			</div>
		</div>
	);
};

export default Organization;

import { useContext } from 'react';
import { AppContext } from '../../../context/providers/AppProvider';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { useLocation } from 'react-router-dom';

const ConnectionTabs = ({ connection }) => {
	const { redirect } = useContext(AppContext);

	const onActionsClick = () => {
		redirect(`connections/${connection.id}/actions`);
	};

	const onOverviewClick = () => {
		redirect(`connections/${connection.id}/overview`);
	};

	const onEventsClick = () => {
		redirect(`connections/${connection.id}/events`);
	};

	const onSettingsClick = () => {
		redirect(`connections/${connection.id}/settings`);
	};

	const location = useLocation();
	const fragments = location.pathname.split('/');
	const tab = fragments[fragments.length - 1];

	return (
		<div className='links'>
			<div className={`link${tab === 'actions' ? ' selected' : ''}`} onClick={onActionsClick}>
				<SlIcon name='send-exclamation'></SlIcon>
				Actions
			</div>
			<div className={`link${tab === 'overview' ? ' selected' : ''}`} onClick={onOverviewClick}>
				<SlIcon name='activity'></SlIcon>
				Overview
			</div>
			{(connection.isMobile || connection.isWebsite || connection.isServer || connection.isStream) && (
				<div className={`link${tab === 'events' ? ' selected' : ''}`} onClick={onEventsClick}>
					<SlIcon name='play'></SlIcon>
					Live events
				</div>
			)}
			<div className={`link${tab === 'settings' ? ' selected' : ''}`} onClick={onSettingsClick}>
				<SlIcon name='sliders2'></SlIcon>
				Settings
			</div>
		</div>
	);
};

export default ConnectionTabs;

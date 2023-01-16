import { useState } from 'react';
import './Sidebar.css';
import { NavLink, Navigate } from 'react-router-dom';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Sidebar = ({ currentRoute }) => {
	let [isLoggedOut, setIsLoggedOut] = useState(false);

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setIsLoggedOut(true);
	};

	let topItems = [
		{ name: 'connections', label: 'Connections', link: '/admin/connections', icon: 'plugin' },
		{ name: 'users', label: 'Users', link: '/admin/users', icon: 'person-lines-fill' },
	];

	if (isLoggedOut) {
		return <Navigate to='/admin' />;
	}

	return (
		<div className='Sidebar'>
			<div className='Items'>
				<div className='Top'>
					<div className='logo'>
						<div className='image'>C</div>
						<div className='text'>Chichi</div>
					</div>
					{topItems.map((i) => {
						return (
							<div className={`item${i.name === currentRoute ? ' selected' : ''}`}>
								<SlIcon name={i.icon} />
								<div className='text'>{i.label}</div>
								<NavLink to={i.link}></NavLink>
							</div>
						);
					})}
				</div>
				<div className='Bottom'>
					<div className='item' onClick={onLogout}>
						<SlIcon name='box-arrow-left' />
						<div className='text'>Logout</div>
					</div>
				</div>
			</div>
		</div>
	);
};

export default Sidebar;

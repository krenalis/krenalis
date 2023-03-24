import { useState } from 'react';
import './Sidebar.css';
import Flex from '../Flex/Flex';
import { NavLink, Navigate } from 'react-router-dom';
import { SlIcon, SlSelect, SlMenuItem } from '@shoelace-style/shoelace/dist/react/index.js';

const Sidebar = ({ route }) => {
	let [isLoggedOut, setIsLoggedOut] = useState(false);

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setIsLoggedOut(true);
	};

	let topItems = [
		{ name: 'connections', label: 'Connections', link: '/admin/connections', icon: 'plug' },
		{ name: 'users', label: 'Users', link: '/admin/users', icon: 'people' },
		{ name: 'schema', label: 'Schema', link: '/admin/schema', icon: 'database' },
	];

	if (isLoggedOut) {
		return <Navigate to='/admin' />;
	}

	return (
		<div className='Sidebar'>
			<div className='Items'>
				<div className='Top'>
					<Flex className='logoAndWorkspace' justifyContent='left' alignItems='center' gap='10px'>
						<div className='logo'>
							<div className='image'>Logo</div>
						</div>
						<div className='workspace'>
							<SlSelect className='workspaceSelector' label='Workspace' value='1'>
								<SlMenuItem value='1' selected>
									Mock workspace 1
								</SlMenuItem>
								<SlMenuItem value='2'>Mock workspace 2</SlMenuItem>
							</SlSelect>
						</div>
					</Flex>
					<div className='logo'></div>
					{topItems.map((i) => {
						return (
							<div className={`item${i.name === route ? ' selected' : ''}`}>
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

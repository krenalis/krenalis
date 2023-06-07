import { useState } from 'react';
import './Sidebar.css';
import Flex from '../Flex/Flex';
import { NavLink, Navigate } from 'react-router-dom';
import { SlIcon, SlSelect, SlOption } from '@shoelace-style/shoelace/dist/react/index.js';

let sidebarItems = [
	{
		name: 'connections',
		label: 'Connections',
		link: '/admin/connections',
		icon: 'plug',
		subItems: [
			{
				name: 'connections/sources',
				label: 'Sources',
				link: '/admin/connections/sources',
				icon: 'file-arrow-down',
			},
			{
				name: 'connections/destinations',
				label: 'Destinations',
				link: '/admin/connections/destinations',
				icon: 'file-arrow-up',
			},
		],
	},
	{ name: 'schema', label: 'Schema', link: '/admin/schema', icon: 'database' },
	{ name: 'users', label: 'Users', link: '/admin/users', icon: 'people' },
];

const Sidebar = ({ route }) => {
	let [isLoggedOut, setIsLoggedOut] = useState(false);

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setIsLoggedOut(true);
	};

	if (isLoggedOut) {
		return <Navigate to='/admin' />;
	}

	let items = [];
	for (let item of sidebarItems) {
		let isSelected = item.name === route;
		let hasSubItems = item.subItems != null;

		let isChildrenSelected = false;
		if (!isSelected && hasSubItems) {
			for (let subItem of item.subItems) {
				if (subItem.name === route) {
					isChildrenSelected = true;
					break;
				}
			}
		}

		items.push(
			<div
				key={item.name}
				className={`item${isSelected ? ' selected' : isChildrenSelected ? ' isChildrenSelected' : ''}`}
			>
				<SlIcon name={item.icon} />
				<div className='text'>{item.label}</div>
				<NavLink to={item.link}></NavLink>
			</div>
		);

		if (hasSubItems && (isSelected || isChildrenSelected)) {
			for (let subItem of item.subItems) {
				let isSelected = subItem.name === route;
				items.push(
					<div key={subItem.name} className={`subItem${isSelected ? ' selected' : ''}`}>
						<SlIcon name={subItem.icon} />
						<div className='text'>{subItem.label}</div>
						<NavLink to={subItem.link}></NavLink>
					</div>
				);
			}
		}
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
								<SlOption value='1' selected>
									Mock workspace 1
								</SlOption>
								<SlOption value='2'>Mock workspace 2</SlOption>
							</SlSelect>
						</div>
					</Flex>
					<div className='logo'></div>
					{items}
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

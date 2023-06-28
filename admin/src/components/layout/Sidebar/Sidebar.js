import { useContext } from 'react';
import './Sidebar.css';
import Flex from '../../common/Flex/Flex';
import { AppContext } from '../../../providers/AppProvider';
import { SlIcon, SlSelect, SlOption } from '@shoelace-style/shoelace/dist/react/index.js';
import { useLocation } from 'react-router-dom';
import getRouteFromPathname from './getRouteFromPathname';

const sidebarItems = [
	{
		name: 'connections',
		label: 'Connections',
		link: 'connections',
		icon: 'plug',
		subItems: [
			{
				name: 'connections/sources',
				label: 'Sources',
				link: 'connections/sources',
			},
			{
				name: 'connections/destinations',
				label: 'Destinations',
				link: 'connections/destinations',
			},
		],
	},
	{ name: 'schema', label: 'Schema', link: 'schema', icon: 'database' },
	{ name: 'users', label: 'Users', link: 'users', icon: 'people' },
];

const Sidebar = ({ setIsLoggedIn }) => {
	const { redirect, connections } = useContext(AppContext);

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setIsLoggedIn(false);
	};

	const location = useLocation();
	const currentRoute = getRouteFromPathname(location.pathname, connections);

	const items = [];
	for (const item of sidebarItems) {
		const isSelected = item.name === currentRoute;
		const hasSubItems = item.subItems != null;

		let isChildrenSelected = false;
		if (!isSelected && hasSubItems) {
			for (const subItem of item.subItems) {
				if (subItem.name === currentRoute) {
					isChildrenSelected = true;
					break;
				}
			}
		}

		items.push(
			<div
				key={item.name}
				className={`item${isSelected ? ' selected' : isChildrenSelected ? ' isChildrenSelected' : ''}`}
				onClick={() => redirect(`${item.link}`)}
			>
				<SlIcon name={item.icon} />
				<div className='text'>{item.label}</div>
			</div>
		);

		if (hasSubItems && (isSelected || isChildrenSelected)) {
			for (const subItem of item.subItems) {
				const isSelected = subItem.name === currentRoute;
				items.push(
					<div
						key={subItem.name}
						className={`subItem${isSelected ? ' selected' : ''}`}
						onClick={() => redirect(`${subItem.link}`)}
					>
						{subItem.icon && <SlIcon name={subItem.icon} />}
						<div className='text'>{subItem.label}</div>
					</div>
				);
			}
		}
	}

	return (
		<div className='sidebar'>
			<div className='items'>
				<div className='top'>
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
				<div className='bottom'>
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

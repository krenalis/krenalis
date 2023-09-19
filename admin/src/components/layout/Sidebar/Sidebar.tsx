import React, { useContext, ReactNode } from 'react';
import './Sidebar.css';
import Flex from '../../shared/Flex/Flex';
import { AppContext } from '../../../context/providers/AppProvider';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import { useLocation } from 'react-router-dom';
import getRouteFromPathname from './getRouteFromPathname';

interface sidebarItem {
	name: string;
	label: string;
	link: string;
	icon?: string;
	subItems?: sidebarItem[];
}

const sidebarItems: sidebarItem[] = [
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
	{ name: 'schema', label: 'Schema', link: 'schema', icon: 'list-nested' },
	{ name: 'users', label: 'Users', link: 'users', icon: 'people' },
	{ name: 'anonymousIdentity', label: 'Anonymous IDs', link: 'anonymous-identity', icon: 'intersect' },
	{ name: 'dataWarehouse', label: 'Data Warehouse', link: 'data-warehouse', icon: 'database' },
];

interface SidebarProps {
	onLogout: () => void;
}

const Sidebar = ({ onLogout }: SidebarProps) => {
	const { redirect, connections } = useContext(AppContext);

	const location = useLocation();
	const currentRoute = getRouteFromPathname(location.pathname, connections);

	const items = [] as ReactNode[];
	for (const item of sidebarItems) {
		const isSelected = item.name === currentRoute;
		const hasSubItems = item.subItems != null;
		let isChildrenSelected = false;
		if (!isSelected && hasSubItems) {
			for (const subItem of item.subItems!) {
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
			</div>,
		);
		if (hasSubItems && (isSelected || isChildrenSelected)) {
			for (const subItem of item.subItems!) {
				const isSelected = subItem.name === currentRoute;
				items.push(
					<div
						key={subItem.name}
						className={`subItem${isSelected ? ' selected' : ''}`}
						onClick={() => redirect(`${subItem.link}`)}
					>
						{subItem.icon && <SlIcon name={subItem.icon} />}
						<div className='text'>{subItem.label}</div>
					</div>,
				);
			}
		}
	}

	return (
		<div className='sidebar'>
			<div className='items'>
				<div className='top'>
					<Flex className='logoAndWorkspace' justifyContent='left' alignItems='center' gap={10}>
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

import React, { useContext, ReactNode, useState, useEffect } from 'react';
import './Sidebar.css';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { useLocation } from 'react-router-dom';
import getRouteFromPathname from './getRouteFromPathname';
import Workspace from '../../../types/external/workspace';
import { Warehouse } from '../../../types/internal/app';

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
	{
		name: 'settings',
		label: 'Settings',
		link: 'settings',
		icon: 'gear',
		subItems: [
			{ name: 'settings/general', label: 'General', link: 'settings/general' },
			{ name: 'settings/dataWarehouse', label: 'Data Warehouse', link: 'settings/data-warehouse' },
			{
				name: 'settings/identifiers',
				label: 'Identifiers',
				link: 'settings/identifiers',
			},
		],
	},
];

interface SidebarProps {
	setIsLoggedIn: React.Dispatch<React.SetStateAction<boolean>>;
	workspaces: Workspace[];
	warehouse: Warehouse;
	selectedWorkspace: number;
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<number>>;
}

const Sidebar = ({ setIsLoggedIn, workspaces, selectedWorkspace, setSelectedWorkspace }: SidebarProps) => {
	const { redirect, connections, warehouse, setIsLoadingState } = useContext(AppContext);

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setSelectedWorkspace(0);
		setIsLoggedIn(false);
		setIsLoadingState(true);
	};

	const location = useLocation();
	const currentRoute = getRouteFromPathname(location.pathname, connections);

	const items: ReactNode[] = [];
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
		const isDisabled = warehouse == null && (item.name === 'schema' || item.name === 'users');
		items.push(
			<div
				key={item.name}
				className={`item${
					isDisabled
						? ' disabled'
						: isSelected
						? ' selected'
						: isChildrenSelected
						? ' isChildrenSelected'
						: ''
				}`}
				onClick={isDisabled ? null : () => redirect(`${item.link}`)}
			>
				<SlIcon className='itemIcon' name={item.icon} />
				<div className='text'>{item.label}</div>
				{isDisabled && (
					<SlTooltip content='You must first connect a data warehouse'>
						<SlIcon className='disabledItemIcon' name='database-exclamation' />
					</SlTooltip>
				)}
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
					<div className='logo'>
						<div className='image'>Logo</div>
					</div>
					<WorkspaceSelector
						setSelectedWorkspace={setSelectedWorkspace}
						workspaces={workspaces}
						selectedWorkspace={selectedWorkspace}
						setIsLoadingState={setIsLoadingState}
					/>
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

interface WorkspaceSelectorProps {
	selectedWorkspace: number;
	workspaces: Workspace[];
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<number>>;
	setIsLoadingState: React.Dispatch<React.SetStateAction<boolean>>;
}

const WorkspaceSelector = ({
	setSelectedWorkspace,
	selectedWorkspace,
	workspaces,
	setIsLoadingState,
}: WorkspaceSelectorProps) => {
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const [searchTerm, setSearchTerm] = useState<string>('');

	useEffect(() => {
		const handleWorkspaceClick = (e) => {
			const isInWorkspaceDialog = e.target.closest('.workspaceDialog') != null;
			if (!isInWorkspaceDialog) {
				const isInWorkspaceSelector = e.target.closest('.workspaceSelector') != null;
				if (!isInWorkspaceSelector) {
					setIsOpen(false);
				}
			}
		};
		window.addEventListener('click', handleWorkspaceClick);
		() => {
			window.removeEventListener('click', handleWorkspaceClick);
		};
	}, []);

	const onViewAllWorkspaces = () => {
		setSelectedWorkspace(0);
	};

	const onWorkspaceSelectorClick = (e) => {
		const isInWorkspaceDialog = e.target.closest('.workspaceDialog') != null;
		if (!isInWorkspaceDialog) {
			setIsOpen(!isOpen);
		}
	};

	const onWorkspaceChange = (id: number) => {
		setSelectedWorkspace(id);
		setIsLoadingState(true);
		setIsOpen(false);
	};

	const onSearchTermChange = (e) => {
		setSearchTerm(e.target.value);
	};

	const searched: any = [];
	for (const workspace of workspaces) {
		const name = workspace.Name;
		if (
			name.includes(searchTerm) ||
			name.includes(searchTerm.charAt(0).toUpperCase() + searchTerm.slice(1)) ||
			name.includes(searchTerm.toUpperCase()) ||
			name.includes(searchTerm.toLowerCase())
		) {
			searched.push(workspace);
		}
	}
	searched.sort((a: Workspace, b: Workspace) => {
		if (a.Name < b.Name) {
			return -1;
		}
		if (a.Name > b.Name) {
			return 1;
		}
		return 0;
	});
	const options: ReactNode[] = [];
	for (const s of searched) {
		options.push(
			<div
				key={s.ID}
				className={`workspaceDialogOption${s.ID === selectedWorkspace ? ' selected' : ''}`}
				onClick={() => onWorkspaceChange(s.ID)}
			>
				<SlIcon name='check-lg' />
				{s.Name}
			</div>,
		);
	}

	return (
		<div className={`workspaceSelector${isOpen ? ' open' : ''}`} onClick={onWorkspaceSelectorClick}>
			<div className='workspaceSelectorText'>
				<div className='workspaceSelectorLabel'>Workspace</div>
				<div className='workspaceSelectorValue'>{workspaces.find((w) => w.ID === selectedWorkspace).Name}</div>
			</div>
			<SlIcon name='chevron-down' className='workspaceSelectorArrow' />
			<div className='workspaceDialog'>
				<SlInput
					className='workspaceDialogSearch'
					value={searchTerm}
					size='small'
					placeholder='Search workspace'
					onSlInput={onSearchTermChange}
				>
					<SlIcon name='search' slot='prefix' />
				</SlInput>
				{options}
				<div className='workspaceDialogViewAll' onClick={onViewAllWorkspaces}>
					All workspaces
					<SlIcon name='arrow-right-short' />
				</div>
			</div>
		</div>
	);
};

export default Sidebar;

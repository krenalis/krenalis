import React, { useContext, ReactNode, useState, useEffect } from 'react';
import './Sidebar.css';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { useLocation } from 'react-router-dom';
import getRouteFromPathname from './getRouteFromPathname';
import Workspace from '../../../lib/api/types/workspace';
import { Warehouse } from '../../routes/App/App.types';
import { Link } from '../Link/Link';

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
		icon: 'shuffle',
		subItems: [
			{
				name: 'connections/sources',
				label: 'Sources',
				link: 'connections/sources',
				icon: 'arrow-down-right-square',
			},
			{
				name: 'connections/destinations',
				label: 'Destinations',
				link: 'connections/destinations',
				icon: 'arrow-up-right-square',
			},
		],
	},
	{ name: 'users', label: 'User Profiles', link: 'users', icon: 'people' },
	{
		name: 'settings',
		label: 'Customization',
		link: 'settings',
		icon: 'gear',
		subItems: [
			{ name: 'settings/general', label: 'General', link: 'settings/general', icon: 'sliders2' },
			{ name: 'schema', label: 'Customer Model', link: 'schema', icon: 'bookmark-check' },
			{
				name: 'settings/identityResolution',
				label: 'Identity Resolution',
				link: 'settings/identity-resolution',
				icon: 'person-arms-up',
			},
			{
				name: 'settings/dataWarehouse',
				label: 'Data Warehouse',
				link: 'settings/data-warehouse',
				icon: 'database',
			},
		],
	},
];

interface SidebarProps {
	workspaces: Workspace[];
	warehouse: Warehouse;
	selectedWorkspace: number;
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<number>>;
}

const Sidebar = ({ workspaces, selectedWorkspace, setSelectedWorkspace }: SidebarProps) => {
	const { redirect, connections, setIsLoadingState, logout, isPasswordless } = useContext(AppContext);

	const onLogout = () => {
		logout();
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
		items.push(
			<Link path={item.link} key={item.name}>
				<div
					className={`sidebar__item${
						isSelected
							? ' sidebar__item--selected'
							: isChildrenSelected
								? ' sidebar__item--isChildrenSelected'
								: ''
					}`}
				>
					<SlIcon className='sidebar__item-icon' name={item.icon} />
					<div className='sidebar__item-text'>{item.label}</div>
				</div>
			</Link>,
		);
		if (hasSubItems) {
			for (const subItem of item.subItems!) {
				const isSelected = subItem.name === currentRoute;
				items.push(
					<Link path={subItem.link} key={subItem.name}>
						<div className={`sidebar__sub-item${isSelected ? ' sidebar__sub-item--selected' : ''}`}>
							{subItem.icon && <SlIcon name={subItem.icon} />}
							<div className='sidebar__sub-item-text'>{subItem.label}</div>
						</div>
					</Link>,
				);
			}
		}
	}

	return (
		<div className='sidebar'>
			<div className='sidebar__items'>
				<div className='sidebar__top'>
					<div className='sidebar__logo-wrapper'>
						<div className='sidebar__logo'>Meergo</div>
					</div>
					<WorkspaceSelector
						setSelectedWorkspace={setSelectedWorkspace}
						workspaces={workspaces}
						selectedWorkspace={selectedWorkspace}
						setIsLoadingState={setIsLoadingState}
						redirect={redirect}
					/>
					{items}
				</div>
				<div className='sidebar__bottom'>
					<a
						href='https://github.com/meergo/meergo/blob/main/CONTRIBUTING.md#how-to-report-a-bug'
						target='_blank'
						className='sidebar__item'
					>
						<SlIcon name='megaphone' />
						<div className='sidebar__item-text'>Report a bug</div>
					</a>
					{!isPasswordless && (
						<div className='sidebar__item' onClick={onLogout}>
							<SlIcon name='box-arrow-left' />
							<div className='sidebar__item-text sidebar__item-text-logout'>Logout</div>
						</div>
					)}
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
	redirect: (url: string) => void;
}

const WorkspaceSelector = ({
	setSelectedWorkspace,
	selectedWorkspace,
	workspaces,
	setIsLoadingState,
	redirect,
}: WorkspaceSelectorProps) => {
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const [searchTerm, setSearchTerm] = useState<string>('');

	useEffect(() => {
		const handleWorkspaceClick = (e) => {
			const isInWorkspaceDialog = e.target.closest('.workspace-selector__dialog') != null;
			if (!isInWorkspaceDialog) {
				const isInWorkspaceSelector = e.target.closest('.workspace-selector') != null;
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
		redirect('workspaces');
	};

	const onWorkspaceSelectorClick = (e) => {
		const isInWorkspaceDialog = e.target.closest('.workspace-selector__dialog') != null;
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
		const name = workspace.name;
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
		if (a.name < b.name) {
			return -1;
		}
		if (a.name > b.name) {
			return 1;
		}
		return 0;
	});
	const options: ReactNode[] = [];
	for (const s of searched) {
		options.push(
			<div
				key={s.id}
				className={`workspace-selector__dialog-option${s.id === selectedWorkspace ? ' workspace-selector__dialog-option--selected' : ''}`}
				onClick={() => onWorkspaceChange(s.id)}
			>
				<SlIcon name='check-lg' />
				<div className='workspace-selector__dialog-option-name'>{s.name}</div>
			</div>,
		);
	}

	return (
		<div
			className={`workspace-selector${isOpen ? ' workspace-selector--open' : ''}`}
			onClick={onWorkspaceSelectorClick}
		>
			<div className='workspace-selector__text'>
				<div className='workspace-selector__label'>Workspace</div>
				<div className='workspace-selector__value'>
					{workspaces.find((w) => w.id === selectedWorkspace).name}
				</div>
			</div>
			<SlIcon name='chevron-down' className='workspace-selector__arrow' />
			<div className='workspace-selector__dialog'>
				<SlInput
					className='workspace-selector__dialog-search'
					value={searchTerm}
					size='small'
					placeholder='Search workspace'
					onSlInput={onSearchTermChange}
				>
					<SlIcon name='search' slot='prefix' />
				</SlInput>
				{options}
				<div className='workspace-selector__dialog-view-all' onClick={onViewAllWorkspaces}>
					All workspaces
					<SlIcon name='arrow-right-short' />
				</div>
			</div>
		</div>
	);
};

export default Sidebar;

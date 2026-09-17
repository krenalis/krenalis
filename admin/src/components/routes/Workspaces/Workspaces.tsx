import React, { useContext } from 'react';
import './Workspaces.css';
import ListTile from '../../base/ListTile/ListTile';
import Workspace from '../../../lib/api/types/workspace';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import AppContext from '../../../context/AppContext';

const Workspaces = () => {
	const { setSelectedWorkspace, workspaces, redirect, setIsLoadingState } = useContext(AppContext);

	const onWorkspaceClick = (id: string) => {
		setSelectedWorkspace(id);
		setIsLoadingState(true);
		redirect('connections');
	};

	const onCreateNewWorkspace = () => {
		redirect('workspaces/create');
	};

	workspaces.sort((a: Workspace, b: Workspace) => {
		if (a.name < b.name) {
			return -1;
		}
		if (a.name > b.name) {
			return 1;
		}
		return 0;
	});

	return (
		<div className='workspace-list'>
			<div className='workspace-list__content'>
				<div className='workspace-list__title-and-button'>
					<p className='workspace-list__title'>Select a workspace</p>
					{workspaces.length > 0 && (
						<SlButton
							variant='brand'
							size='s'
							className='workspace-list__create-button'
							onClick={onCreateNewWorkspace}
						>
							<WaIcon name='plus' slot='start' />
							Create a new workspace
						</SlButton>
					)}
				</div>
				<div className='workspace-list__workspaces'>
					{workspaces.length === 0 ? (
						<>
							<div className='workspace-list__no-workspace'>
								Currently you don't have any workspace. Create at least one workspace to continue.
							</div>
							<SlButton
								className='workspace-list__no-workspace-action'
								variant='brand'
								onClick={onCreateNewWorkspace}
							>
								<WaIcon name='plus' slot='start' />
								Create your first workspace
							</SlButton>
						</>
					) : (
						workspaces.map((workspace) => {
							return (
								<ListTile
									key={workspace.id}
									className='workspace-list__workspace'
									icon={<SlIcon name='person-workspace' />}
									name={workspace.name}
									showHover={true}
									id={workspace.id}
									onClick={() => onWorkspaceClick(workspace.id)}
									action={<SlIcon name='chevron-right' />}
								/>
							);
						})
					)}
				</div>
			</div>
		</div>
	);
};

export default Workspaces;

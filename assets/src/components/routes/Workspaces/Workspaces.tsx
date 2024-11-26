import React, { useContext } from 'react';
import './Workspaces.css';
import ListTile from '../../base/ListTile/ListTile';
import Workspace from '../../../lib/api/types/workspace';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import AppContext from '../../../context/AppContext';

const Workspaces = () => {
	const { setSelectedWorkspace, workspaces, redirect, setIsLoadingState } = useContext(AppContext);

	const onWorkspaceClick = (id: number) => {
		setSelectedWorkspace(id);
		setIsLoadingState(true);
		redirect('connections');
	};

	const onAddNewWorkspace = () => {
		redirect('workspaces/add');
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
							variant='primary'
							size='small'
							className='workspace-list__add-button'
							onClick={onAddNewWorkspace}
						>
							<SlIcon name='plus' slot='prefix' />
							Add a new workspace
						</SlButton>
					)}
				</div>
				<div className='workspace-list__workspaces'>
					{workspaces.length === 0 ? (
						<>
							<div className='workspace-list__no-workspace'>
								Currently you don't have any workspace. Add at least one workspace to continue.
							</div>
							<SlButton
								className='workspace-list__no-workspace-action'
								variant='primary'
								onClick={onAddNewWorkspace}
							>
								<SlIcon name='plus' slot='prefix' />
								Add your first workspace
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
									id={String(workspace.id)}
									description={workspace.privacyRegion}
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

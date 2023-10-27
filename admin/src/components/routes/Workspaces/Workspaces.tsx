import React, { useState } from 'react';
import './Workspaces.css';
import ListTile from '../../shared/ListTile/ListTile';
import API from '../../../lib/api/api';
import Workspace from '../../../types/external/workspace';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';

interface WorkspacesProps {
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<number>>;
	workspaces: Workspace[];
	api: API;
	showError: (err: Error | string) => void;
	redirect: (url: string) => void;
	setIsWorkspaceStale: React.Dispatch<React.SetStateAction<boolean>>;
}

const Workspaces = ({
	setSelectedWorkspace,
	workspaces,
	api,
	showError,
	redirect,
	setIsWorkspaceStale,
}: WorkspacesProps) => {
	const [isAddWorkspaceDialogOpen, setIsAddWorkspaceDialogOpen] = useState(false);

	const onWorkspaceClick = (id: number) => {
		setSelectedWorkspace(id);
	};

	const onAddNewWorkspace = () => setIsAddWorkspaceDialogOpen(true);

	workspaces.sort((a: Workspace, b: Workspace) => {
		if (a.Name < b.Name) {
			return -1;
		}
		if (a.Name > b.Name) {
			return 1;
		}
		return 0;
	});

	return (
		<div className='workspace-list__content'>
			<div className='workspace-list'>
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
									className='workspace-list__workspace'
									icon={<SlIcon name='person-workspace' />}
									name={workspace.Name}
									description={workspace.PrivacyRegion}
									onClick={() => onWorkspaceClick(workspace.ID)}
									action={<SlIcon name='chevron-right' />}
								/>
							);
						})
					)}
				</div>

				<NewWorkspaceDialog
					setSelectedWorkspace={setSelectedWorkspace}
					isAddWorkspaceDialogOpen={isAddWorkspaceDialogOpen}
					setIsAddWorkspaceDialogOpen={setIsAddWorkspaceDialogOpen}
					api={api}
					showError={showError}
					redirect={redirect}
					setIsWorkspaceStale={setIsWorkspaceStale}
				/>
			</div>
		</div>
	);
};

interface NewWorkspaceDialogProps {
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<number>>;
	isAddWorkspaceDialogOpen: boolean;
	setIsAddWorkspaceDialogOpen: React.Dispatch<React.SetStateAction<boolean>>;
	api: API;
	showError: (err: Error | string) => void;
	redirect: (url: string) => void;
	setIsWorkspaceStale: React.Dispatch<React.SetStateAction<boolean>>;
}

const NewWorkspaceDialog = ({
	setSelectedWorkspace,
	isAddWorkspaceDialogOpen,
	setIsAddWorkspaceDialogOpen,
	api,
	showError,
	redirect,
	setIsWorkspaceStale,
}: NewWorkspaceDialogProps) => {
	const [name, setName] = useState<string>('');
	const [useEuropeRegion, setUseEuropeRegion] = useState<boolean>(false);

	const onNameChange = (e) => setName(e.target.value);

	const onUseEuropeRegionChange = () => setUseEuropeRegion(!useEuropeRegion);

	const onAddWorkspace = async () => {
		const privacyRegion = useEuropeRegion ? 'Europe' : '';
		let id: number;
		try {
			const res = await api.workspaces.add(name, privacyRegion);
			id = res.id;
		} catch (err) {
			showError(err);
		}
		setIsAddWorkspaceDialogOpen(false);
		setName('');
		setUseEuropeRegion(false);
		setSelectedWorkspace(id);
		setIsWorkspaceStale(true);
		redirect('settings');
	};

	return (
		<SlDialog
			className='workspace-list__add-dialog'
			label='Add workspace'
			open={isAddWorkspaceDialogOpen}
			onSlAfterHide={() => setIsAddWorkspaceDialogOpen(false)}
		>
			<SlInput
				className='workspace-list__new-workspace-name'
				maxlength={100}
				label='Name'
				value={name}
				onSlChange={onNameChange}
			/>
			<SlCheckbox
				className='workspace-list__new-workspace-use-europe-region'
				checked={useEuropeRegion}
				onSlChange={onUseEuropeRegionChange}
			>
				Use the European Privacy Region <span>(can be changed later)</span>
			</SlCheckbox>
			<SlButton
				className='workspace-list__new-workspace-add-workspace-button'
				variant='primary'
				onClick={onAddWorkspace}
			>
				Add
			</SlButton>
		</SlDialog>
	);
};

export default Workspaces;

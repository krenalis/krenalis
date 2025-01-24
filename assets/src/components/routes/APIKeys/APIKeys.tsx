import React, { ReactNode, useContext, useEffect, useMemo, useRef, useState } from 'react';
import './APIKeys.css';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import AppContext from '../../../context/AppContext';
import { APIKey, APIKeyResponse, CreateAPIKeyResponse } from '../../../lib/api/types/responses';
import Grid from '../../base/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { NotFoundError } from '../../../lib/api/errors';
import { Link } from '../../base/Link/Link';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';

const GRID_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Workspace' },
	{ name: 'Token' },
	{ name: 'Created' },
	{ name: '' },
];

const APIKeys = () => {
	const [apiKeys, setApiKeys] = useState<APIKey[]>();
	const [isLoadingAPIKeys, setIsLoadingAPIKeys] = useState<boolean>(true);
	const [isCreateAPIKeyDialogOpen, setIsCreateAPIKeyDialogOpen] = useState<boolean>(false);
	const [apiKeyToDelete, setApiKeyToDelete] = useState<number | null>();
	const [isRevokingAPIKey, setIsDeletingAPIKey] = useState<boolean>(false);
	const [apiKeyToEdit, setApiKeyToEdit] = useState<APIKey | null>();

	const { api, handleError, workspaces } = useContext(AppContext);

	useEffect(() => {
		const fetchAPIKeys = async () => {
			let res: APIKeyResponse;
			try {
				res = await api.keys();
			} catch (err) {
				setTimeout(() => {
					handleError(err);
					setIsLoadingAPIKeys(false);
				}, 300);
				return;
			}
			setApiKeys(res.keys);
			setTimeout(() => setIsLoadingAPIKeys(false), 300);
		};

		if (!isLoadingAPIKeys) {
			return;
		}

		fetchAPIKeys();
	}, [isLoadingAPIKeys]);

	const onEditKey = (apiKey: APIKey) => {
		setApiKeyToEdit(apiKey);
	};

	const onDeleteKey = (id: number) => {
		setApiKeyToDelete(id);
	};

	const onCloseDeleteApiKeyDialog = () => {
		setApiKeyToDelete(null);
	};

	const onConfirmDeleteKey = async () => {
		setIsDeletingAPIKey(true);
		try {
			await api.deleteAPIKey(apiKeyToDelete);
		} catch (err) {
			setIsDeletingAPIKey(false);
			if (!(err instanceof NotFoundError)) {
				handleError(err);
				return;
			}
		}
		setIsDeletingAPIKey(false);
		setApiKeyToDelete(null);
		setTimeout(() => {
			setIsLoadingAPIKeys(true);
		}, 300);
	};

	const rows: GridRow[] = useMemo(() => {
		if (apiKeys == null) {
			return [];
		}
		const r: GridRow[] = [];
		for (const apiKey of apiKeys) {
			let workspaceCell: ReactNode;
			if (apiKey.workspace != null) {
				const workspace = workspaces.find((w) => w.id === apiKey.workspace);
				workspaceCell = `${workspace.name}`;
			} else {
				workspaceCell = <span className='api-keys__grid-any-workspace'>Any workspace</span>;
			}

			const tokenCell = <div className='api-keys__token-cell'>{apiKey.token}</div>;

			const createdAtCell = <RelativeTime date={apiKey.createdAt} />;

			const actionsCell = (
				<div className='api-keys__grid-buttons'>
					<SlButton variant='default' size='small' onClick={() => onEditKey(apiKey)}>
						Edit...
					</SlButton>
					<SlButton
						className='connection-actions__delete-action'
						variant='danger'
						size='small'
						onClick={() => onDeleteKey(apiKey.id)}
					>
						Delete
					</SlButton>
				</div>
			);

			r.push({
				cells: [apiKey.name, workspaceCell, tokenCell, createdAtCell, actionsCell],
				key: String(apiKey.id),
			});
		}
		return r;
	}, [apiKeys]);

	return (
		<div className='api-keys'>
			<div className='api-keys__content'>
				<Link path='organization'>
					<SlButton className='api-keys__back-button' variant='text'>
						<SlIcon slot='prefix' name='arrow-left' />
						Organization
					</SlButton>
				</Link>
				<div className='api-keys__title'>
					<p className='api-keys__title-text'>API keys</p>
					<SlButton size='small' variant='primary' onClick={() => setIsCreateAPIKeyDialogOpen(true)}>
						Add a new API key
					</SlButton>
				</div>
				<Grid
					className='api-keys__grid'
					rows={rows}
					columns={GRID_COLUMNS}
					noRowsMessage='No API keys to show'
					isLoading={isLoadingAPIKeys}
				/>
				<AlertDialog
					variant='danger'
					isOpen={apiKeyToDelete != null}
					onClose={onCloseDeleteApiKeyDialog}
					title='Delete the API key?'
					actions={
						<>
							<SlButton onClick={onCloseDeleteApiKeyDialog}>Cancel</SlButton>
							<SlButton variant='danger' onClick={onConfirmDeleteKey} loading={isRevokingAPIKey}>
								Delete
							</SlButton>
						</>
					}
				>
					If you delete the API key the applications that use it will no longer be able to access the API
				</AlertDialog>
				<EditAPIKeyDialog
					apiKeyToEdit={apiKeyToEdit}
					setApiKeyToEdit={setApiKeyToEdit}
					setIsLoadingAPIKeys={setIsLoadingAPIKeys}
				/>
				<CreateAPIKeyDialog
					isOpen={isCreateAPIKeyDialogOpen}
					setIsOpen={setIsCreateAPIKeyDialogOpen}
					setIsLoadingAPIKeys={setIsLoadingAPIKeys}
				/>
			</div>
		</div>
	);
};

interface EditAPIKeyDialogProps {
	apiKeyToEdit: APIKey | null;
	setApiKeyToEdit: React.Dispatch<React.SetStateAction<APIKey | null>>;
	setIsLoadingAPIKeys: React.Dispatch<React.SetStateAction<boolean>>;
}

const EditAPIKeyDialog = ({ apiKeyToEdit, setApiKeyToEdit, setIsLoadingAPIKeys }: EditAPIKeyDialogProps) => {
	const [name, setName] = useState<string>('');
	const [error, setError] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { handleError, api } = useContext(AppContext);

	const inputRef = useRef<any>();

	useEffect(() => {
		if (apiKeyToEdit != null) {
			setName(apiKeyToEdit.name);
			setTimeout(() => {
				inputRef.current.focus();
			}, 100);
		}
	}, [apiKeyToEdit]);

	const onInputName = (e) => {
		const v = e.target.value;
		setName(v);
	};

	const onHide = () => {
		setApiKeyToEdit(null);
		setError('');
		setName('');
	};

	const onSave = async () => {
		setError('');
		setIsSaving(true);

		try {
			validateAPIName(name);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setError(err.message);
			}, 100);
			return;
		}

		try {
			await api.updateAPIKey(apiKeyToEdit.id, name);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setApiKeyToEdit(null);
				setTimeout(() => {
					handleError(err);
				}, 150);
			}, 300);
			return;
		}

		setTimeout(() => {
			setIsSaving(false);
			setApiKeyToEdit(null);
			setTimeout(() => {
				setIsLoadingAPIKeys(true);
			}, 300);
		}, 300);
	};

	return (
		<SlDialog
			className='api-keys__edit-dialog'
			label='Edit the API key'
			open={apiKeyToEdit != null}
			onSlAfterHide={onHide}
		>
			<div className='api-keys__dialog-form'>
				<SlInput
					className='api-keys__dialog-name'
					ref={inputRef}
					label='Name'
					value={name}
					onSlInput={onInputName}
				/>
				{error && (
					<div className='api-keys__dialog-error'>
						<SlIcon slot='icon' name='exclamation-octagon' />
						{error}
					</div>
				)}
				<SlButton loading={isSaving} className='api-keys__dialog-save' variant='primary' onClick={onSave}>
					Save
				</SlButton>
			</div>
		</SlDialog>
	);
};

interface CreateAPIKeyDialogProps {
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	setIsLoadingAPIKeys: React.Dispatch<React.SetStateAction<boolean>>;
}

const CreateAPIKeyDialog = ({ isOpen, setIsOpen, setIsLoadingAPIKeys }: CreateAPIKeyDialogProps) => {
	const [name, setName] = useState<string>('');
	const [workspace, setWorkspace] = useState<number | null>();
	const [error, setError] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);
	const [token, setToken] = useState<string | null>();

	const { handleError, api, workspaces } = useContext(AppContext);

	const inputRef = useRef<any>();

	useEffect(() => {
		if (isOpen) {
			setTimeout(() => {
				inputRef.current.focus();
			}, 100);
		}
	}, [isOpen]);

	const onInputName = (e) => {
		const v = e.target.value;
		setName(v);
	};

	const onChangeWorkspace = (e) => {
		const v = e.target.value;
		if (v === '0') {
			setWorkspace(null);
		} else {
			setWorkspace(Number(v));
		}
	};

	const onHide = (e) => {
		if (e.target.tagName === 'SL-DIALOG') {
			setIsOpen(false);
			setError('');
			setName('');
			setWorkspace(null);
			if (token != null) {
				setToken(null);
				setIsLoadingAPIKeys(true);
			}
		}
	};

	const onDone = () => {
		setIsOpen(false);
		setTimeout(() => {
			setToken(null);
			setIsLoadingAPIKeys(true);
		}, 300);
	};

	const onSave = async () => {
		setError('');
		setIsSaving(true);

		try {
			validateAPIName(name);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setError(err.message);
			}, 100);
			return;
		}

		let res: CreateAPIKeyResponse;
		try {
			res = await api.createAPIKey(name, workspace);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setIsOpen(false);
				setTimeout(() => {
					setName('');
					setWorkspace(null);
					handleError(err);
				}, 150);
			}, 300);
			return;
		}

		setTimeout(() => {
			setIsSaving(false);
			setToken(res.token);
			setTimeout(() => {
				setName('');
				setWorkspace(null);
			}, 300);
		}, 300);
	};

	return (
		<SlDialog
			className={`api-keys__dialog${token != null ? ' api-keys__dialog--copy' : ''}`}
			label={token != null ? 'Your new API key' : 'Create a new API key'}
			open={isOpen}
			onSlAfterHide={onHide}
		>
			{token != null ? (
				<div className='api-keys__dialog-created'>
					<div className='api-keys__dialog-created-title'>Copy the API key</div>
					<div className='api-keys__dialog-created-description'>
						Copy the key and store it in a safe place, like a password manager or secret store, as this will
						be the last time it will be visible to you
					</div>
					<div className='api-keys__key-copy'>
						<SlInput readonly value={token} filled />
						<SlCopyButton value={token} />
					</div>
					<SlButton size='small' className='api-keys__dialog-done' variant='primary' onClick={onDone}>
						Done
					</SlButton>
				</div>
			) : (
				<div className='api-keys__dialog-form'>
					<SlInput
						className='api-keys__dialog-name'
						ref={inputRef}
						label='Name'
						value={name}
						onSlInput={onInputName}
					/>
					{error && (
						<div className='api-keys__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{error}
						</div>
					)}
					<SlSelect
						label='Workspace'
						className='api-keys__dialog-workspaces'
						onSlChange={onChangeWorkspace}
						value={workspace == null ? '0' : String(workspace)}
						hoist={true}
						helpText='Note that you will no longer be able to edit the workspace after the creation of the key'
					>
						<SlOption value='0'>Any workspace</SlOption>
						{workspaces.map((w) => (
							<SlOption value={String(w.id)}>{w.name}</SlOption>
						))}
					</SlSelect>
					<SlButton loading={isSaving} className='api-keys__dialog-save' variant='primary' onClick={onSave}>
						Add
					</SlButton>
				</div>
			)}
		</SlDialog>
	);
};

const validateAPIName = (name: string) => {
	if (name === '') {
		throw new Error('Name is required');
	}
	if (Array.from(name).length > 100) {
		throw new Error('Name must be no longer than 100 characters');
	}
};

export { APIKeys };

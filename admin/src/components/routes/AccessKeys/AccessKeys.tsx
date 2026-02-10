import React, { ReactNode, useContext, useEffect, useMemo, useRef, useState } from 'react';
import './AccessKeys.css';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import AppContext from '../../../context/AppContext';
import { AccessKey, AccessKeyResponse, CreateAccessKeyResponse } from '../../../lib/api/types/responses';
import Grid from '../../base/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { NotFoundError } from '../../../lib/api/errors';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';
import { AccessKeyType } from '../../../lib/api/types/organization';
import { WarehouseResponse } from '../../../lib/api/types/warehouse';

const GRID_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Workspace' },
	{ name: 'Token' },
	{ name: 'Created' },
	{ name: '' },
];

const AccessKeys = () => {
	const [accessKeys, setAccessKeys] = useState<AccessKey[]>();
	const [isLoadingAPIKeys, setIsLoadingAPIKeys] = useState<boolean>(true);
	const [isLoadingMCPKeys, setIsLoadingMCPKeys] = useState<boolean>(true);
	const [isCreatingAPIKey, setIsCreatingAPIKey] = useState<boolean>(false);
	const [isCreatingMCPKey, setIsCreatingMCPKey] = useState<boolean>(false);
	const [accessKeyToDelete, setAccessKeyToDelete] = useState<AccessKey | null>();
	const [isDeletingAccessKey, setIsDeletingAccessKey] = useState<boolean>(false);
	const [accessKeyToEdit, setAccessKeyToEdit] = useState<AccessKey | null>();
	const [accessKeyToDeleteType, setAccessKeyToDeleteType] = useState<AccessKeyType>();
	const [warehouseByWorkspace, setWarehouseByWorkspace] = useState<Record<number, string>>();

	const { api, handleError, workspaces } = useContext(AppContext);

	useEffect(() => {
		const fetchData = async () => {
			let res: AccessKeyResponse;
			try {
				res = await api.keys();
			} catch (err) {
				setTimeout(() => {
					handleError(err);
					if (isLoadingMCPKeys) {
						setIsLoadingMCPKeys(false);
					}
					if (isLoadingAPIKeys) {
						setIsLoadingAPIKeys(false);
					}
				}, 300);
				return;
			}
			setAccessKeys(res.keys);

			let warehouseByWorkspace = {};
			for (const w of workspaces) {
				let res: WarehouseResponse;
				try {
					res = await api.workspaces.warehouse(w.id);
				} catch (err) {
					setTimeout(() => {
						handleError(err);
						if (isLoadingMCPKeys) {
							setIsLoadingMCPKeys(false);
						}
						if (isLoadingAPIKeys) {
							setIsLoadingAPIKeys(false);
						}
					}, 300);
					return;
				}
				warehouseByWorkspace[w.id] = res.platform;
			}

			setTimeout(() => {
				setWarehouseByWorkspace(warehouseByWorkspace);
				if (isLoadingMCPKeys) {
					setIsLoadingMCPKeys(false);
				}
				if (isLoadingAPIKeys) {
					setIsLoadingAPIKeys(false);
				}
			}, 300);
		};

		if (!isLoadingAPIKeys && !isLoadingMCPKeys) {
			return;
		}

		fetchData();
	}, [isLoadingAPIKeys, isLoadingMCPKeys]);

	useEffect(() => {
		if (accessKeyToDelete != null) {
			setAccessKeyToDeleteType(accessKeyToDelete.type);
		}
	}, [accessKeyToDelete]);

	const onEditKey = (accessKey: AccessKey) => {
		setAccessKeyToEdit(accessKey);
	};

	const onDeleteKey = (accessKey: AccessKey) => {
		setAccessKeyToDelete(accessKey);
	};

	const onCloseDeleteAccessKeyDialog = () => {
		setAccessKeyToDelete(null);
	};

	const onConfirmDeleteKey = async () => {
		setIsDeletingAccessKey(true);
		try {
			await api.deleteAccessKey(accessKeyToDelete.id);
		} catch (err) {
			setIsDeletingAccessKey(false);
			if (!(err instanceof NotFoundError)) {
				handleError(err);
				return;
			}
		}
		setIsDeletingAccessKey(false);
		setAccessKeyToDelete(null);
		setTimeout(() => {
			if (accessKeyToDelete.type === 'MCP') {
				setIsLoadingMCPKeys(true);
			} else {
				setIsLoadingAPIKeys(true);
			}
		}, 300);
	};

	const [apiRows, mcpRows]: GridRow[] = useMemo(() => {
		if (accessKeys == null) {
			return [[], []];
		}

		const apiRows: GridRow[] = [];
		const mcpRows: GridRow[] = [];
		for (const k of accessKeys) {
			let workspaceCell: ReactNode;
			if (k.workspace != null) {
				const workspace = workspaces.find((w) => w.id === k.workspace);
				workspaceCell = `${workspace.name}`;
			} else {
				workspaceCell = <span className='access-keys__grid-any-workspace'>Any workspace</span>;
			}

			const tokenCell = <div className='access-keys__token-cell'>{k.token}</div>;

			const createdAtCell = <RelativeTime date={k.createdAt} />;

			const pipelinesCell = (
				<div className='access-keys__grid-buttons'>
					<SlButton variant='default' size='small' onClick={() => onEditKey(k)}>
						Edit...
					</SlButton>
					<SlButton
						className='connection-pipelines__delete-pipeline'
						variant='danger'
						size='small'
						onClick={() => onDeleteKey(k)}
					>
						Delete
					</SlButton>
				</div>
			);

			const row = {
				cells: [k.name, workspaceCell, tokenCell, createdAtCell, pipelinesCell],
				key: String(k.id),
			};

			if (k.type === 'API') {
				apiRows.push(row);
			} else {
				mcpRows.push(row);
			}
		}
		return [apiRows, mcpRows];
	}, [accessKeys]);

	const hasWorkspaceSupportingMCP =
		warehouseByWorkspace != null &&
		Object.keys(warehouseByWorkspace).findIndex((id) => warehouseByWorkspace[id] !== 'Snowflake') !== -1;

	return (
		<div className='access-keys'>
			<div className='access-keys__content'>
				<div className='access-keys__title'>
					<p className='access-keys__title-text'>API keys</p>
					<SlButton size='small' variant='primary' onClick={() => setIsCreatingAPIKey(true)}>
						Add a new API key
					</SlButton>
				</div>
				<Grid
					className='access-keys__grid'
					rows={apiRows}
					columns={GRID_COLUMNS}
					noRowsMessage='No API keys to show'
					isLoading={isLoadingAPIKeys}
				/>
				<div className='access-keys__grid-learn-more'>
					Learn more about{' '}
					<a href='https://www.meergo.com/docs/ref/admin/api-keys' target='_blank'>
						API keys
					</a>
				</div>
				<SlDivider style={{ '--spacing': '30px' } as React.CSSProperties} />
				<div className='access-keys__title access-keys__title--mcp'>
					<p className='access-keys__title-text'>MCP keys</p>
					{hasWorkspaceSupportingMCP && (
						<SlButton size='small' variant='primary' onClick={() => setIsCreatingMCPKey(true)}>
							Add a new MCP key
						</SlButton>
					)}
				</div>
				<Grid
					className='access-keys__grid'
					rows={mcpRows}
					columns={GRID_COLUMNS}
					noRowsMessage={
						hasWorkspaceSupportingMCP ? 'No MCP keys to show' : 'None of your workspaces support MCP'
					}
					isLoading={isLoadingMCPKeys}
				/>
				<div className='access-keys__grid-learn-more'>
					Learn more about{' '}
					<a href='https://www.meergo.com/docs/ref/admin/ai-querying-with-mcp' target='_blank'>
						AI querying with MCP
					</a>
				</div>
				<AlertDialog
					variant='danger'
					isOpen={accessKeyToDelete != null}
					onClose={onCloseDeleteAccessKeyDialog}
					title={`Delete the ${accessKeyToDeleteType} key?`}
					actions={
						<>
							<SlButton onClick={onCloseDeleteAccessKeyDialog}>Cancel</SlButton>
							<SlButton variant='danger' onClick={onConfirmDeleteKey} loading={isDeletingAccessKey}>
								Delete
							</SlButton>
						</>
					}
				>
					If you delete the {accessKeyToDeleteType} key the applications that use it will no longer be able to
					access the {accessKeyToDeleteType === 'API' ? 'API' : 'MCP server'}
				</AlertDialog>
				<EditAccessKeyDialog
					accessKeyToEdit={accessKeyToEdit}
					setAccessKeyToEdit={setAccessKeyToEdit}
					setIsLoadingAccessKeys={accessKeyToEdit?.type === 'MCP' ? setIsLoadingMCPKeys : setIsLoadingAPIKeys}
				/>
				<CreateAccessKeyDialog
					isOpen={isCreatingAPIKey || isCreatingMCPKey}
					isMCP={isCreatingMCPKey}
					setIsOpen={isCreatingMCPKey ? setIsCreatingMCPKey : setIsCreatingAPIKey}
					setIsLoadingAPIKeys={setIsLoadingAPIKeys}
					setIsLoadingMCPKeys={setIsLoadingMCPKeys}
					warehouseByWorkspace={warehouseByWorkspace}
				/>
			</div>
		</div>
	);
};

interface EditAccessKeyDialogProps {
	accessKeyToEdit: AccessKey | null;
	setAccessKeyToEdit: React.Dispatch<React.SetStateAction<AccessKey | null>>;
	setIsLoadingAccessKeys: React.Dispatch<React.SetStateAction<boolean>>;
}

const EditAccessKeyDialog = ({
	accessKeyToEdit,
	setAccessKeyToEdit,
	setIsLoadingAccessKeys,
}: EditAccessKeyDialogProps) => {
	const [name, setName] = useState<string>('');
	const [error, setError] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { handleError, api } = useContext(AppContext);

	const inputRef = useRef<any>();

	useEffect(() => {
		if (accessKeyToEdit != null) {
			setName(accessKeyToEdit.name);
			setTimeout(() => {
				inputRef.current.focus();
			}, 100);
		}
	}, [accessKeyToEdit]);

	const onInputName = (e) => {
		const v = e.target.value;
		setName(v);
	};

	const onHide = () => {
		setAccessKeyToEdit(null);
		setError('');
		setName('');
	};

	const onSave = async () => {
		setError('');
		setIsSaving(true);

		try {
			validateKeyName(name);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setError(err.message);
			}, 100);
			return;
		}

		try {
			await api.updateAccessKey(accessKeyToEdit.id, name);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setAccessKeyToEdit(null);
				setTimeout(() => {
					handleError(err);
				}, 150);
			}, 300);
			return;
		}

		setTimeout(() => {
			setIsSaving(false);
			setAccessKeyToEdit(null);
			setTimeout(() => {
				setIsLoadingAccessKeys(true);
			}, 300);
		}, 300);
	};

	return (
		<SlDialog
			className='access-keys__edit-dialog'
			label={`Edit the ${accessKeyToEdit?.type} key`}
			open={accessKeyToEdit != null}
			onSlAfterHide={onHide}
		>
			<div className='access-keys__dialog-form'>
				<SlInput
					className='access-keys__dialog-name'
					ref={inputRef}
					label='Name'
					value={name}
					onSlInput={onInputName}
				/>
				{error && (
					<div className='access-keys__dialog-error'>
						<SlIcon slot='icon' name='exclamation-octagon' />
						{error}
					</div>
				)}
				<SlButton loading={isSaving} className='access-keys__dialog-save' variant='primary' onClick={onSave}>
					Save
				</SlButton>
			</div>
		</SlDialog>
	);
};

interface CreateAccessKeyDialogProps {
	isOpen: boolean;
	isMCP: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	setIsLoadingAPIKeys: React.Dispatch<React.SetStateAction<boolean>>;
	setIsLoadingMCPKeys: React.Dispatch<React.SetStateAction<boolean>>;
	warehouseByWorkspace: Record<number, string>;
}

const CreateAccessKeyDialog = ({
	isOpen,
	isMCP,
	setIsOpen,
	setIsLoadingAPIKeys,
	setIsLoadingMCPKeys,
	warehouseByWorkspace,
}: CreateAccessKeyDialogProps) => {
	const [name, setName] = useState<string>('');
	const [workspace, setWorkspace] = useState<number | null>();
	const [nameError, setNameError] = useState<string>('');
	const [workspaceError, setWorkspaceError] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);
	const [token, setToken] = useState<string | null>();

	const { handleError, api, workspaces } = useContext(AppContext);

	const inputRef = useRef<any>();
	const keyType = useRef<AccessKeyType>();

	useEffect(() => {
		if (isOpen) {
			keyType.current = isMCP ? 'MCP' : 'API';
		}
	}, [isOpen]);

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
			setNameError('');
			setName('');
			setWorkspace(null);
			setWorkspaceError('');
			if (token != null) {
				setToken(null);
				if (keyType.current === 'MCP') {
					setIsLoadingMCPKeys(true);
				} else {
					setIsLoadingAPIKeys(true);
				}
			}
		}
	};

	const onDone = () => {
		setIsOpen(false);
		setTimeout(() => {
			setToken(null);
			if (keyType.current === 'MCP') {
				setIsLoadingMCPKeys(true);
			} else {
				setIsLoadingAPIKeys(true);
			}
		}, 300);
	};

	const onSave = async () => {
		setNameError('');
		setWorkspaceError('');
		setIsSaving(true);

		try {
			validateKeyName(name);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setNameError(err.message);
			}, 100);
			return;
		}

		if (isMCP && workspace == null) {
			setTimeout(() => {
				setIsSaving(false);
				setWorkspaceError('MCP keys must be linked to a workspace');
			}, 100);
			return;
		}

		let res: CreateAccessKeyResponse;
		try {
			res = await api.createAccessKey(name, workspace, isMCP ? 'MCP' : 'API');
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
			className={`access-keys__dialog${token != null ? ' access-keys__dialog--copy' : ''}`}
			label={
				token != null ? `Your new ${isMCP ? 'MCP' : 'API'} key` : `Create a new ${isMCP ? 'MCP' : 'API'} key`
			}
			open={isOpen}
			onSlAfterHide={onHide}
		>
			{token != null ? (
				<div className='access-keys__dialog-created'>
					<div className='access-keys__dialog-created-title'>Copy the {isMCP ? 'MCP' : 'API'} key</div>
					<div className='access-keys__dialog-created-description'>
						Copy the key and store it in a safe place, like a password manager or secret store, as this will
						be the last time it will be visible to you
					</div>
					<div className='access-keys__key-copy'>
						<SlInput readonly value={token} filled />
						<SlCopyButton value={token} />
					</div>
					<SlButton size='small' className='access-keys__dialog-done' variant='primary' onClick={onDone}>
						Done
					</SlButton>
				</div>
			) : (
				<div className='access-keys__dialog-form'>
					<SlInput
						className='access-keys__dialog-name'
						ref={inputRef}
						label='Name'
						value={name}
						onSlInput={onInputName}
					/>
					{nameError && (
						<div className='access-keys__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{nameError}
						</div>
					)}
					<SlSelect
						label='Workspace'
						className='access-keys__dialog-workspaces'
						onSlChange={onChangeWorkspace}
						value={workspace != null ? String(workspace) : isMCP ? '' : '0'}
						hoist={true}
						helpText='Note that you will no longer be able to edit the workspace after the creation of the key'
					>
						{!isMCP && <SlOption value='0'>Any workspace</SlOption>}
						{warehouseByWorkspace != null &&
							workspaces.map((w) => {
								if (!isMCP || (isMCP && warehouseByWorkspace[w.id] !== 'Snowflake')) {
									return <SlOption value={String(w.id)}>{w.name}</SlOption>;
								}
								return null;
							})}
					</SlSelect>
					{workspaceError && (
						<div className='access-keys__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{workspaceError}
						</div>
					)}
					<SlButton
						loading={isSaving}
						className='access-keys__dialog-save'
						variant='primary'
						onClick={onSave}
					>
						Add
					</SlButton>
				</div>
			)}
		</SlDialog>
	);
};

const validateKeyName = (name: string) => {
	if (name === '') {
		throw new Error('Name is required');
	}
	if (Array.from(name).length > 100) {
		throw new Error('Name must be no longer than 100 characters');
	}
};

export { AccessKeys };

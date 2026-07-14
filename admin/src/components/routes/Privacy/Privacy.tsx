import React, { useContext, useEffect, useMemo, useRef, useState, useLayoutEffect } from 'react';
import './Privacy.css';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ConsentPurposesResponse } from '../../../lib/api/types/responses';
import { UnprocessableError } from '../../../lib/api/errors';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Name' }, { name: 'Code' }, { name: 'Pipelines' }, { name: '' }];

const validatePurposeField = (name: string, value: string) => {
	if (value === '') {
		throw new Error(`${name} is required`);
	}
	if (Array.from(value).length > 100) {
		throw new Error(`${name} must be no longer than 100 characters`);
	}
};

const Privacy = () => {
	const [purposes, setPurposes] = useState<ConsentPurpose[]>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isCreating, setIsCreating] = useState<boolean>(false);
	const [purposeToEdit, setPurposeToEdit] = useState<ConsentPurpose | null>();
	const [purposeToDelete, setPurposeToDelete] = useState<ConsentPurpose | null>();
	const [isDeleting, setIsDeleting] = useState<boolean>(false);

	const { api, handleError, setTitle, redirect } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Settings / Privacy');
	}, [setTitle]);

	useEffect(() => {
		if (!isLoading) {
			return;
		}
		const fetchData = async () => {
			let res: ConsentPurposesResponse;
			try {
				res = await api.workspaces.consentPurposes();
			} catch (err) {
				setTimeout(() => {
					setIsLoading(false);
					handleError(err);
				}, 300);
				return;
			}
			setTimeout(() => {
				setPurposes(res.purposes);
				setIsLoading(false);
			}, 300);
		};
		fetchData();
	}, [isLoading]);

	const onDeletePurpose = (purpose: ConsentPurpose) => {
		setPurposeToDelete(purpose);
	};

	const onCloseDeleteDialog = () => {
		setPurposeToDelete(null);
	};

	const onConfirmDelete = async () => {
		setIsDeleting(true);
		try {
			await api.workspaces.deleteConsentPurpose(purposeToDelete.id);
		} catch (err) {
			setIsDeleting(false);
			if (err instanceof UnprocessableError && err.code === 'ConsentPurposeInUse') {
				setPurposeToDelete(null);
				setTimeout(() => {
					handleError(
						`The "${purposeToDelete.name}" purpose is now required by one or more pipelines. Remove it from those pipelines before you can delete it.`,
					);
					setIsLoading(true);
				}, 150);
				return;
			}
			handleError(err);
			return;
		}
		setIsDeleting(false);
		setPurposeToDelete(null);
		setTimeout(() => {
			setIsLoading(true);
		}, 300);
	};

	const rows: GridRow[] = useMemo(() => {
		if (purposes == null) {
			return [];
		}
		return purposes.map((p) => {
			const codeCell = <span className="privacy__grid-code">{p.code}</span>
			const pipelinesCell =
				p.pipelines.length === 0 ? (
					<span className='privacy__grid-pipelines-empty'>-</span>
				) : (
					<div className='privacy__grid-pipelines'>
						{p.pipelines.map((pl) => (
							<SlTooltip key={pl.id} content={pl.name}>
								<button
									type='button'
									className='privacy__grid-pipeline-logo'
									onClick={() =>
										redirect(`connections/${pl.connection}/pipelines/edit/${pl.id}`)
									}
								>
									<LittleLogo code={pl.connector} path={CONNECTORS_ASSETS_PATH} />
								</button>
							</SlTooltip>
						))}
					</div>
				);
			const actionsCell = (
				<div className='privacy__grid-buttons'>
					<SlButton variant='default' size='small' onClick={() => setPurposeToEdit(p)}>
						Edit...
					</SlButton>
					<SlButton variant='danger' size='small' onClick={() => onDeletePurpose(p)}>
						Delete
					</SlButton>
				</div>
			);
			return {
				cells: [p.name, codeCell, pipelinesCell, actionsCell],
				key: p.id,
			};
		});
	}, [purposes]);

	return (
		<div className='privacy'>
			<div className='privacy__content'>
				<div className='privacy__title'>
					<p className='privacy__title-text'>Purposes</p>
					<SlButton size='small' variant='primary' onClick={() => setIsCreating(true)}>
						Add a new purpose
					</SlButton>
				</div>
				<div className='privacy__description'>
					Pipelines can require a purpose, so they only deliver an event when user consent has been given for it.
				</div>
				<Grid
					className='privacy__grid'
					rows={rows}
					columns={GRID_COLUMNS}
					noRowsMessage='No purposes to show'
					isLoading={isLoading}
				/>
				<AlertDialog
					variant='danger'
					isOpen={purposeToDelete != null}
					onClose={onCloseDeleteDialog}
					title={
						purposeToDelete && purposeToDelete.pipelines.length > 0 ? (
							<span>Unlink the purpose before deleting it</span>
						) : (
							<span>Delete the purpose?</span>
						)
					}
					actions={
						purposeToDelete && purposeToDelete.pipelines.length > 0 ? (
							<SlButton onClick={onCloseDeleteDialog}>Close</SlButton>
						) : (
							<>
								<SlButton onClick={onCloseDeleteDialog}>Cancel</SlButton>
								<SlButton variant='danger' onClick={onConfirmDelete} loading={isDeleting}>
									Delete
								</SlButton>
							</>
						)
					}
				>
					{purposeToDelete && purposeToDelete.pipelines.length > 0
						? `The "${purposeToDelete.name}" purpose is required by one or more pipelines. Remove it from those pipelines before you can delete it.`
						: `Once deleted, no pipeline will be able to require consent for "${purposeToDelete?.name}".`}
				</AlertDialog>
				<PurposeDialog
					isOpen={isCreating}
					purposeToEdit={null}
					onClose={() => setIsCreating(false)}
					onSaved={() => setIsLoading(true)}
				/>
				<PurposeDialog
					isOpen={purposeToEdit != null}
					purposeToEdit={purposeToEdit}
					onClose={() => setPurposeToEdit(null)}
					onSaved={() => setIsLoading(true)}
				/>
			</div>
		</div>
	);
};

interface PurposeDialogProps {
	isOpen: boolean;
	purposeToEdit: ConsentPurpose | null;
	onClose: () => void;
	onSaved: () => void;
}

const PurposeDialog = ({ isOpen, purposeToEdit, onClose, onSaved }: PurposeDialogProps) => {
	const [name, setName] = useState<string>('');
	const [code, setCode] = useState<string>('');
	const [nameError, setNameError] = useState<string>('');
	const [codeError, setCodeError] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { api, handleError } = useContext(AppContext);

	const inputRef = useRef<any>();

	const isEditing = purposeToEdit != null;

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		setName(isEditing ? purposeToEdit.name : '');
		setCode(isEditing ? purposeToEdit.code : '');
		setNameError('');
		setCodeError('');
		setTimeout(() => {
			inputRef.current?.focus();
		}, 100);
	}, [isOpen]);

	const onInputName = (e) => setName(e.target.value);
	const onInputCode = (e) => setCode(e.target.value);

	const onSave = async () => {
		setNameError('');
		setCodeError('');

		try {
			validatePurposeField('Name', name);
		} catch (err) {
			setNameError(err.message);
			return;
		}
		try {
			validatePurposeField('Code', code);
		} catch (err) {
			setCodeError(err.message);
			return;
		}

		setIsSaving(true);
		try {
			if (isEditing) {
				await api.workspaces.updateConsentPurpose(purposeToEdit.id, name, code);
			} else {
				await api.workspaces.createConsentPurpose(name, code);
			}
		} catch (err) {
			setIsSaving(false);
			if (err instanceof UnprocessableError && err.code === 'ConsentPurposeCodeExists') {
				setCodeError('A purpose with this code already exists');
				return;
			}
			onClose();
			setTimeout(() => {
				handleError(err);
			}, 150);
			return;
		}

		setIsSaving(false);
		onClose();
		setTimeout(() => {
			onSaved();
		}, 300);
	};

	return (
		<SlDialog
			className='privacy__dialog'
			label={isEditing ? 'Edit the purpose' : 'Create a new purpose'}
			open={isOpen}
			onSlAfterHide={onClose}
		>
			<div className='privacy__dialog-form'>
				<SlInput
					className='privacy__dialog-name'
					ref={inputRef}
					label='Name'
					value={name}
					onSlInput={onInputName}
					helpText='A recognizable name for this purpose'
				/>
				{nameError && (
					<div className='privacy__dialog-error'>
						<SlIcon slot='icon' name='exclamation-octagon' />
						{nameError}
					</div>
				)}
				<SlInput
					className='privacy__dialog-code'
					label='Code'
					value={code}
					onSlInput={onInputCode}
					helpText="The code of the purpose. It must match the code you use to track consents within your CMP"
				/>
				{codeError && (
					<div className='privacy__dialog-error'>
						<SlIcon slot='icon' name='exclamation-octagon' />
						{codeError}
					</div>
				)}
				<SlButton loading={isSaving} className='privacy__dialog-save' variant='primary' onClick={onSave}>
					Save
				</SlButton>
			</div>
		</SlDialog>
	);
};

export default Privacy;

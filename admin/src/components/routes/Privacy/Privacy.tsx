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
import { isValidPropertyPath } from '../../../utils/filters';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Name' }, { name: 'Code' }, { name: 'Pipelines' }, { name: '' }];

interface PurposePipeline {
	id: string;
	name: string;
	connection: string;
	connector: string;
}

const CODE_FORMAT = /^[A-Za-z_][0-9A-Za-z_]*$/;

const validatePurposeField = (name: string, value: string) => {
	if (value === '') {
		throw new Error(`${name} is required`);
	}
	if (Array.from(value).length > 100) {
		throw new Error(`${name} must be no longer than 100 characters`);
	}
};

const validatePurposeCode = (value: string) => {
	validatePurposeField('Code', value);
	if (!CODE_FORMAT.test(value)) {
		throw new Error(
			'Code must start with a letter or an underscore and can only contain letters, digits and underscores',
		);
	}
};

const validatePurposePath = (name: string, value: string) => {
	if (value === '') {
		return;
	}
	if (Array.from(value).length > 1024) {
		throw new Error(`${name} must be no longer than 1024 characters`);
	}
	if (!isValidPropertyPath(value)) {
		throw new Error(
			`${name} must be property names separated by a dot, each starting with a letter or an underscore and containing only letters, digits and underscores`,
		);
	}
};

const pathToSave = (isCustom: boolean, value: string, defaultValue: string) => {
	if (!isCustom || value === defaultValue) {
		// Leave the path empty when the default is used.
		return '';
	}
	return value;
};

const Privacy = () => {
	const [purposes, setPurposes] = useState<ConsentPurpose[]>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isCreating, setIsCreating] = useState<boolean>(false);
	const [purposeToEdit, setPurposeToEdit] = useState<ConsentPurpose | null>();
	const [purposeToDelete, setPurposeToDelete] = useState<ConsentPurpose | null>();
	const [isDeleting, setIsDeleting] = useState<boolean>(false);

	const { api, connections, handleError, setTitle, redirect } = useContext(AppContext);

	const pipelinesByPurpose = useMemo(() => {
		const result = new Map<string, PurposePipeline[]>();
		for (const connection of connections) {
			for (const pipeline of connection.pipelines) {
				for (const purpose of pipeline.requiredConsents?.purposes ?? []) {
					const pipelines = result.get(purpose) ?? [];
					pipelines.push({
						id: pipeline.id,
						name: pipeline.name,
						connection: connection.id,
						connector: connection.connector.code,
					});
					result.set(purpose, pipelines);
				}
			}
		}
		for (const pipelines of result.values()) {
			pipelines.sort((a, b) => a.name.localeCompare(b.name));
		}
		return result;
	}, [connections]);

	const purposeToDeletePipelines = purposeToDelete == null ? [] : (pipelinesByPurpose.get(purposeToDelete.id) ?? []);

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
			const pipelines = pipelinesByPurpose.get(p.id) ?? [];
			const codeCell = <span className='privacy__grid-code'>{p.code}</span>;
			const pipelinesCell =
				pipelines.length === 0 ? (
					<span className='privacy__grid-pipelines-empty'>-</span>
				) : (
					<div className='privacy__grid-pipelines'>
						{pipelines.map((pl) => (
							<SlTooltip key={pl.id} content={pl.name}>
								<button
									type='button'
									className='privacy__grid-pipeline-logo'
									onClick={() => redirect(`connections/${pl.connection}/pipelines/edit/${pl.id}`)}
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
				key: p.code,
			};
		});
	}, [pipelinesByPurpose, purposes, redirect]);

	return (
		<div className='privacy'>
			<div className='privacy__content'>
				<div className='privacy__title'>
					<p className='privacy__title-text'>Consent purposes</p>
					<SlButton size='small' variant='primary' onClick={() => setIsCreating(true)}>
						Add a new purpose
					</SlButton>
				</div>
				<div className='privacy__description'>
					Pipelines can require a purpose, so they only deliver an event when user consent has been given for
					it.
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
						purposeToDeletePipelines.length > 0 ? (
							<span>Unlink the purpose before deleting it</span>
						) : (
							<span>Delete the purpose?</span>
						)
					}
					actions={
						purposeToDeletePipelines.length > 0 ? (
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
					{purposeToDelete && purposeToDeletePipelines.length > 0
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

interface PathActionProps {
	isCustom: boolean;
	onCustomize: () => void;
	onReset: () => void;
}

// PathAction is the button shown within a path input. It unlocks the path for
// editing or reset it to the default if it already edited.
const PathAction = ({ isCustom, onCustomize, onReset }: PathActionProps) => (
	<SlTooltip slot='suffix' content={isCustom ? 'Reset to the default path' : 'Edit the path'} hoist>
		<SlButton
			className='privacy__dialog-path-action'
			variant='text'
			size='small'
			circle
			onClick={isCustom ? onReset : onCustomize}
		>
			<SlIcon name={isCustom ? 'arrow-counterclockwise' : 'pencil'} />
		</SlButton>
	</SlTooltip>
);

interface PurposeDialogProps {
	isOpen: boolean;
	purposeToEdit: ConsentPurpose | null;
	onClose: () => void;
	onSaved: () => void;
}

const PurposeDialog = ({ isOpen, purposeToEdit, onClose, onSaved }: PurposeDialogProps) => {
	const [name, setName] = useState<string>('');
	const [code, setCode] = useState<string>('');
	const [eventPath, setEventPath] = useState<string>('');
	const [profilePath, setProfilePath] = useState<string>('');
	const [isEventPathCustom, setIsEventPathCustom] = useState<boolean>(false);
	const [isProfilePathCustom, setIsProfilePathCustom] = useState<boolean>(false);
	const [isWarningOpen, setIsWarningOpen] = useState<boolean>(false);
	const [nameError, setNameError] = useState<string>('');
	const [codeError, setCodeError] = useState<string>('');
	const [eventPathError, setEventPathError] = useState<string>('');
	const [profilePathError, setProfilePathError] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { api, handleError } = useContext(AppContext);

	const inputRef = useRef<any>();
	const eventPathInputRef = useRef<any>();
	const profilePathInputRef = useRef<any>();
	const selectEventPathAfterWarning = useRef<boolean>(false);

	const isEditing = purposeToEdit != null;

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		setName(isEditing ? purposeToEdit.name : '');
		setCode(isEditing ? purposeToEdit.code : '');
		setEventPath(isEditing ? purposeToEdit.eventPath : '');
		setProfilePath(isEditing ? purposeToEdit.profilePath : '');
		setIsEventPathCustom(isEditing && purposeToEdit.eventPath !== '');
		setIsProfilePathCustom(isEditing && purposeToEdit.profilePath !== '');
		setIsWarningOpen(false);
		selectEventPathAfterWarning.current = false;
		setNameError('');
		setCodeError('');
		setEventPathError('');
		setProfilePathError('');
		setTimeout(() => {
			inputRef.current?.focus();
		}, 100);
	}, [isOpen]);

	// The tooltips of the path actions bubble their own sl-after-hide up to the
	// dialog, which would close it. Only the event of the dialog itself closes it.
	const onSlAfterHide = (e) => {
		if (e.target !== e.currentTarget) {
			e.stopPropagation();
			return;
		}
		onClose();
	};

	const onInputName = (e) => setName(e.target.value);
	const onInputCode = (e) => setCode(e.target.value);
	const onInputEventPath = (e) => setEventPath(e.target.value);
	const onInputProfilePath = (e) => setProfilePath(e.target.value);

	// The paths default to the consents of the context of an event and to the
	// consents of a profile, both keyed after the code of the purpose. Until
	// the code is written they are shown as a placeholder, since the path they
	// lead to is not decided yet.
	const codeOrPlaceholder = code === '' ? '<code>' : code;
	const defaultEventPath = code === '' ? '' : `context.consents.${code}`;
	const defaultProfilePath = code === '' ? '' : `consents.${code}`;
	const shownEventPath = isEventPathCustom ? eventPath : defaultEventPath;
	const shownProfilePath = isProfilePathCustom ? profilePath : defaultProfilePath;
	const eventPathPlaceholder = isEventPathCustom ? '' : `context.consents.${codeOrPlaceholder}`;
	const profilePathPlaceholder = isProfilePathCustom ? '' : `consents.${codeOrPlaceholder}`;

	const onCustomizeEventPath = () => {
		setEventPath(defaultEventPath);
		setIsEventPathCustom(true);
		selectEventPathAfterWarning.current = true;
		setIsWarningOpen(false);
	};

	const onWarningClosed = () => {
		setIsWarningOpen(false);
		if (selectEventPathAfterWarning.current) {
			selectEventPathAfterWarning.current = false;
			setTimeout(() => {
				eventPathInputRef.current?.select();
			}, 0);
		}
	};

	const onResetEventPath = () => {
		setEventPath('');
		setEventPathError('');
		setIsEventPathCustom(false);
	};

	const onCustomizeProfilePath = () => {
		setProfilePath(defaultProfilePath);
		setIsProfilePathCustom(true);
		setTimeout(() => {
			profilePathInputRef.current?.select();
		}, 0);
	};

	const onResetProfilePath = () => {
		setProfilePath('');
		setProfilePathError('');
		setIsProfilePathCustom(false);
	};

	const onSave = async () => {
		setNameError('');
		setCodeError('');
		setEventPathError('');
		setProfilePathError('');

		const eventPathToSave = pathToSave(isEventPathCustom, eventPath, defaultEventPath);
		const profilePathToSave = pathToSave(isProfilePathCustom, profilePath, defaultProfilePath);

		try {
			validatePurposeField('Name', name);
		} catch (err) {
			setNameError(err.message);
			return;
		}
		try {
			validatePurposeCode(code);
		} catch (err) {
			setCodeError(err.message);
			return;
		}
		try {
			validatePurposePath('Event path', eventPathToSave);
		} catch (err) {
			setEventPathError(err.message);
			return;
		}
		try {
			validatePurposePath('Profile path', profilePathToSave);
		} catch (err) {
			setProfilePathError(err.message);
			return;
		}

		setIsSaving(true);
		try {
			if (isEditing) {
				await api.workspaces.updateConsentPurpose(
					purposeToEdit.id,
					code,
					name,
					eventPathToSave,
					profilePathToSave,
				);
			} else {
				await api.workspaces.addConsentPurpose(code, name, eventPathToSave, profilePathToSave);
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
		<>
			<SlDialog
				className='privacy__dialog'
				label={isEditing ? 'Edit the purpose' : 'Add a new purpose'}
				open={isOpen}
				onSlAfterHide={onSlAfterHide}
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
						helpText='The code of the purpose. It must match the code you use to track consents within your CMP'
					/>
					{codeError && (
						<div className='privacy__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{codeError}
						</div>
					)}

					<SlInput
						className='privacy__dialog-event-path'
						ref={eventPathInputRef}
						label='Event path'
						value={shownEventPath}
						onSlInput={onInputEventPath}
						placeholder={eventPathPlaceholder}
						readonly={!isEventPathCustom}
						helpText='The event property that holds the consent given for this purpose'
					>
						<PathAction
							isCustom={isEventPathCustom}
							onCustomize={() => setIsWarningOpen(true)}
							onReset={onResetEventPath}
						/>
					</SlInput>
					{eventPathError && (
						<div className='privacy__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{eventPathError}
						</div>
					)}

					<SlInput
						className='privacy__dialog-profile-path'
						ref={profilePathInputRef}
						label='Profile path'
						value={shownProfilePath}
						onSlInput={onInputProfilePath}
						placeholder={profilePathPlaceholder}
						readonly={!isProfilePathCustom}
						helpText='The profile property that holds the consent given for this purpose'
					>
						<PathAction
							isCustom={isProfilePathCustom}
							onCustomize={onCustomizeProfilePath}
							onReset={onResetProfilePath}
						/>
					</SlInput>
					{profilePathError && (
						<div className='privacy__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{profilePathError}
						</div>
					)}

					<SlButton loading={isSaving} className='privacy__dialog-save' variant='primary' onClick={onSave}>
						{isEditing ? 'Save' : 'Add'}
					</SlButton>
				</div>
			</SlDialog>
			<AlertDialog
				variant='danger'
				isOpen={isWarningOpen}
				onClose={onWarningClosed}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={() => setIsWarningOpen(false)}>Cancel</SlButton>
						<SlButton
							variant='danger'
							className='privacy__dialog-customize-event-path'
							onClick={onCustomizeEventPath}
						>
							Edit
						</SlButton>
					</>
				}
			>
				Krenalis SDKs integrate with CMPs to send the consents automatically in the context of the event. Change
				this path only if you manually deliver them somewhere else.
			</AlertDialog>
		</>
	);
};

export default Privacy;

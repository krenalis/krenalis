import React, { useContext, useEffect, useMemo, useRef, useState, useLayoutEffect } from 'react';
import { flushSync } from 'react-dom';
import { useSearchParams } from 'react-router-dom';
import './Privacy.css';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose, ProfileConsentLocation } from '../../../lib/api/types/workspace';
import { ConsentPurposesResponse } from '../../../lib/api/types/responses';
import { UnprocessableError } from '../../../lib/api/errors';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import { formatProfileConsentLocation, validateConsentKey } from '../../../utils/consentPurposePaths';
import { ObjectType } from '../../../lib/api/types/types';
import { FlatSchema, flattenSchema } from '../../../lib/core/pipeline';
import { SchemaPropertyInfoTooltip } from '../Schema/SchemaPropertyGrid';

const GRID_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Consent in events' },
	{ name: 'Consent in profiles' },
	{ name: 'Pipelines' },
	{ name: '' },
];

// MAX_EVENT_CONSENT_LOCATIONS is the maximum number of event consent locations a purpose can have.
const MAX_EVENT_CONSENT_LOCATIONS = 5;

// SHOWN_EVENT_CONSENT_LOCATIONS is the number of event consent locations shown
// in the grid before the remaining ones are counted.
const SHOWN_EVENT_CONSENT_LOCATIONS = 2;

interface PurposePipeline {
	id: string;
	name: string;
	connection: string;
	connector: string;
}

const validatePurposeName = (value: string) => {
	if (value === '') {
		throw new Error('Name is required');
	}
	if (Array.from(value).length > 100) {
		throw new Error('Name must be no longer than 100 characters');
	}
};

// checkProfileConsentLocation returns the message to show when location does
// not lead to a profile schema property that can hold a consent, and an empty
// string otherwise.
const checkProfileConsentLocation = (location: ProfileConsentLocation | null, schema: FlatSchema | null): string => {
	if (location == null) {
		return '';
	}
	const propertyPath = location.property;
	if (Array.from(propertyPath).length > 1024) {
		return 'Profile property must be no longer than 1024 characters';
	}
	const key = location.jsonKey || null;
	const path = formatProfileConsentLocation(location);
	const kind = schema?.[propertyPath]?.type;
	if (kind == null) {
		return `Profile path "${path}" does not exist in the profile schema`;
	}
	if (kind === 'json') {
		if (key == null) {
			return `Profile path "${path}" is a JSON property, which holds a consent only in a value inside it`;
		}
	} else if (kind !== 'boolean') {
		return `Profile path "${path}" is a ${kind} property, which cannot hold a consent`;
	} else if (key != null) {
		// The Admin UI never adds a key to a boolean property, but a purpose
		// created or updated through the API can have one.
		return `Profile path "${path}" is a boolean property, which holds a consent itself and has no keys`;
	}
	return '';
};

const Privacy = () => {
	const [purposes, setPurposes] = useState<ConsentPurpose[]>();
	const [profileSchema, setProfileSchema] = useState<ObjectType>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isCreating, setIsCreating] = useState<boolean>(false);
	const [purposeToEdit, setPurposeToEdit] = useState<ConsentPurpose | null>();
	const [purposeToDelete, setPurposeToDelete] = useState<ConsentPurpose | null>();
	const [isDeleting, setIsDeleting] = useState<boolean>(false);

	const [searchParams, setSearchParams] = useSearchParams();

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

	useEffect(() => {
		const purposeID = searchParams.get('purpose');
		if (purposes == null || purposeID == null) {
			return;
		}
		setPurposeToEdit(purposes.find((purpose) => purpose.id === purposeID));

		// Remove the parameter once handled, otherwise the dialog would reopen
		// every time the purposes are reloaded, for example after saving.
		const nextSearchParams = new URLSearchParams(searchParams);
		nextSearchParams.delete('purpose');
		setSearchParams(nextSearchParams, { replace: true });
	}, [purposes, searchParams, setSearchParams]);

	// The profile schema is read to check that the profile path of a purpose
	// leads to a property that can hold the consent given for it.
	useEffect(() => {
		const fetchProfileSchema = async () => {
			let schema: ObjectType;
			try {
				schema = await api.workspaces.profileSchema();
			} catch (err) {
				handleError(err);
				return;
			}
			setProfileSchema(schema);
		};
		fetchProfileSchema();
	}, []);

	const flatProfileSchema = useMemo(() => flattenSchema(profileSchema), [profileSchema]);

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
			const eventConsentLocationsCell =
				p.eventConsentLocations.length === 0 ? (
					<span className='privacy__grid-paths-empty'>-</span>
				) : (
					<div className='privacy__grid-paths'>
						{p.eventConsentLocations.slice(0, SHOWN_EVENT_CONSENT_LOCATIONS).map((location, index) => (
							<span key={index} className='privacy__grid-path'>
								{location.purposeCode}
							</span>
						))}
						{p.eventConsentLocations.length > SHOWN_EVENT_CONSENT_LOCATIONS && (
							<SlTooltip
								content={p.eventConsentLocations
									.slice(SHOWN_EVENT_CONSENT_LOCATIONS)
									.map((location) => location.purposeCode)
									.join(', ')}
							>
								<span className='privacy__grid-paths-more'>{`+${p.eventConsentLocations.length - SHOWN_EVENT_CONSENT_LOCATIONS}`}</span>
							</SlTooltip>
						)}
					</div>
				);
			const profileConsentLocationCell =
				p.profileConsentLocation == null ? (
					<span className='privacy__grid-paths-empty'>-</span>
				) : (
					<span className='privacy__grid-path'>{formatProfileConsentLocation(p.profileConsentLocation)}</span>
				);
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
				cells: [p.name, eventConsentLocationsCell, profileConsentLocationCell, pipelinesCell, actionsCell],
				key: p.id,
			};
		});
	}, [pipelinesByPurpose, purposes, redirect]);

	return (
		<div className='privacy'>
			<div className='privacy__content'>
				<div className='privacy__title'>
					<p className='privacy__title-text'>Consent purposes</p>
					<SlButton size='small' variant='primary' onClick={() => setIsCreating(true)}>
						Add consent purpose
					</SlButton>
				</div>
				<div className='privacy__description'>
					Pipelines can require consent for a purpose, so they only process events or profiles when the user
					has consented to it.
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
					purposes={purposes}
					profileSchema={flatProfileSchema}
					onClose={() => setIsCreating(false)}
					onSaved={() => setIsLoading(true)}
				/>
				<PurposeDialog
					isOpen={purposeToEdit != null}
					purposeToEdit={purposeToEdit}
					purposes={purposes}
					profileSchema={flatProfileSchema}
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
	purposes: ConsentPurpose[] | undefined;
	profileSchema: FlatSchema | null;
	onClose: () => void;
	onSaved: () => void;
}

const getDuplicatePurposeCodeError = (purposeCodes: string[]): string => {
	const seenPurposeCodes = new Set<string>();
	for (const purposeCode of purposeCodes) {
		if (purposeCode === '') {
			continue;
		}
		if (seenPurposeCodes.has(purposeCode)) {
			return `Purpose code "${purposeCode}" is duplicated`;
		}
		seenPurposeCodes.add(purposeCode);
	}
	return '';
};

const PurposeDialog = ({ isOpen, purposeToEdit, purposes, profileSchema, onClose, onSaved }: PurposeDialogProps) => {
	const [name, setName] = useState<string>('');
	const [purposeCodes, setPurposeCodes] = useState<string[]>(['']);
	const [profilePropertyPath, setProfilePropertyPath] = useState<string>('');
	const [profileJSONKey, setProfileJSONKey] = useState<string | null>(null);
	const [nameError, setNameError] = useState<string>('');
	const [purposeCodesError, setPurposeCodesError] = useState<string>('');
	const [duplicatePurposeCodeError, setDuplicatePurposeCodeError] = useState<string>('');
	const [profilePathError, setProfilePathError] = useState<string>('');
	const [hasProfileJSONKeyLostFocus, setHasProfileJSONKeyLostFocus] = useState<boolean>(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);
	const [newPurposeCodeIndex, setNewPurposeCodeIndex] = useState<number | null>(null);
	const [isRemovingPurposeCode, setIsRemovingPurposeCode] = useState<boolean>(false);
	const [validationErrorVersion, setValidationErrorVersion] = useState<number>(0);

	const { api, handleError } = useContext(AppContext);

	const inputRef = useRef<any>();
	const purposeCodeInputRefs = useRef<any[]>([]);
	const purposeCodeRowRefs = useRef<HTMLDivElement[]>([]);
	const isRemovingPurposeCodeRef = useRef<boolean>(false);
	const formRef = useRef<any>();
	const profilePathDropdownRef = useRef<any>();

	const profilePathOptions = useMemo(
		() =>
			Object.entries(profileSchema ?? {}).filter(
				([, property]) => property.type === 'boolean' || property.type === 'json',
			),
		[profileSchema],
	);

	const isEditing = purposeToEdit != null;

	const isProfilePropertyJSON = profileJSONKey !== null;
	const profileConsentLocation: ProfileConsentLocation | null =
		profilePropertyPath === ''
			? null
			: profileJSONKey === null
				? { property: profilePropertyPath }
				: { property: profilePropertyPath, jsonKey: profileJSONKey };

	const isProfileLocationUsedByOtherPurposes =
		profileConsentLocation != null &&
		(!isProfilePropertyJSON || hasProfileJSONKeyLostFocus) &&
		checkProfileConsentLocation(profileConsentLocation, profileSchema) === '' &&
		purposes?.some(
			(purpose) =>
				purpose.id !== purposeToEdit?.id &&
				purpose.profileConsentLocation?.property === profilePropertyPath &&
				(purpose.profileConsentLocation.jsonKey || null) === profileJSONKey,
		) === true;

	const profilePathWarning = !isProfileLocationUsedByOtherPurposes
		? ''
		: isProfilePropertyJSON
			? 'This property and key are also used by other purposes.'
			: 'This property is also used by other purposes.';

	useLayoutEffect(() => {
		if (validationErrorVersion === 0) {
			return;
		}
		formRef.current
			?.querySelector('.privacy__dialog-error')
			?.scrollIntoView({ block: 'center', behavior: 'smooth' });
	}, [validationErrorVersion]);

	// Wait for Shoelace to render the new input before measuring the row or focusing it.
	useLayoutEffect(() => {
		if (newPurposeCodeIndex === null) {
			return;
		}

		const row = purposeCodeRowRefs.current[newPurposeCodeIndex];
		const input = purposeCodeInputRefs.current[newPurposeCodeIndex];
		if (row == null || input == null) {
			return;
		}

		let animation: Animation | undefined;
		let animationFrame: number | undefined;
		let canceled = false;
		input.updateComplete.then(() => {
			if (canceled) {
				return;
			}

			const height = row.getBoundingClientRect().height;
			const marginTop = getComputedStyle(row).marginTop;
			row.style.overflow = 'hidden';
			animation = row.animate(
				[
					{ height: '0', marginTop: '0', opacity: 0, transform: 'translateY(2px)' },
					{ height: `${height}px`, marginTop, opacity: 1, transform: 'translateY(0)' },
				],
				{ duration: 140, easing: 'ease-out' },
			);
			animation.addEventListener('finish', () => setNewPurposeCodeIndex(null), { once: true });
			animationFrame = requestAnimationFrame(() => {
				if (purposeCodeInputRefs.current[newPurposeCodeIndex] === input) {
					input.focus();
				}
			});
		});

		return () => {
			canceled = true;
			if (animationFrame != null) {
				cancelAnimationFrame(animationFrame);
			}
			animation?.cancel();
			row.style.removeProperty('overflow');
		};
	}, [purposeCodes.length, newPurposeCodeIndex]);

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		const originalProfile = purposeToEdit?.profileConsentLocation;
		setName(isEditing ? purposeToEdit.name : '');
		setPurposeCodes(
			isEditing && purposeToEdit.eventConsentLocations.length > 0
				? purposeToEdit.eventConsentLocations.map(({ purposeCode }) => purposeCode)
				: [''],
		);
		setProfilePropertyPath(originalProfile?.property ?? '');
		setProfileJSONKey(originalProfile?.jsonKey || null);
		setNameError('');
		setPurposeCodesError('');
		setDuplicatePurposeCodeError('');
		setProfilePathError('');
		setHasProfileJSONKeyLostFocus((originalProfile?.jsonKey ?? '') !== '');
		setNewPurposeCodeIndex(null);
		setIsRemovingPurposeCode(false);
		isRemovingPurposeCodeRef.current = false;
		if (!isEditing) {
			setTimeout(() => {
				inputRef.current?.focus();
			}, 100);
		}
	}, [isOpen]);

	useEffect(() => {
		if (duplicatePurposeCodeError === '') {
			return;
		}
		const nextError = getDuplicatePurposeCodeError(purposeCodes);
		if (nextError !== duplicatePurposeCodeError) {
			setDuplicatePurposeCodeError(nextError);
		}
	}, [duplicatePurposeCodeError, purposeCodes]);

	// The tooltips of the path actions bubble their own sl-after-hide up to the
	// dialog, which would close it. Only the event of the dialog itself closes it.
	const onSlAfterHide = (e) => {
		if (e.target !== e.currentTarget) {
			e.stopPropagation();
			return;
		}
		onClose();
	};

	const onInputName = (e: any) => setName(e.target.value);

	const showValidationError = (setError: React.Dispatch<React.SetStateAction<string>>, message: string) => {
		setError(message);
		setValidationErrorVersion((version) => version + 1);
	};

	const onInputPurposeCode = (e: any, index: number) => {
		const value = e.target.value;
		const nextPurposeCodes = purposeCodes.map((purposeCode, i) => (i === index ? value : purposeCode));
		setPurposeCodes(nextPurposeCodes);
		if (duplicatePurposeCodeError !== '') {
			setDuplicatePurposeCodeError(getDuplicatePurposeCodeError(nextPurposeCodes));
		}
	};

	const onBlurPurposeCode = () => {
		setDuplicatePurposeCodeError(getDuplicatePurposeCodeError(purposeCodes));
	};

	const onSelectProfilePath = (e: any) => {
		const path = e.detail.item.value;
		setProfilePropertyPath(path);
		setProfileJSONKey(profileSchema?.[path]?.type === 'json' ? '' : null);
		setProfilePathError('');
		setHasProfileJSONKeyLostFocus(false);
	};

	// Capture interactions within the input before the dropdown handles its trigger events.
	const onClearProfilePath = (event: React.MouseEvent) => {
		event.preventDefault();
		event.stopPropagation();
		profilePathDropdownRef.current?.hide();
		setProfilePropertyPath('');
		setProfileJSONKey(null);
		setProfilePathError('');
		setHasProfileJSONKeyLostFocus(false);
	};

	const onInputProfileJSONPath = (e: any) => {
		const value = e.target.value;
		setProfileJSONKey(value);
		setHasProfileJSONKeyLostFocus(false);
		try {
			validateConsentKey(value);
			if (checkProfileConsentLocation({ property: profilePropertyPath, jsonKey: value }, profileSchema) === '') {
				setProfilePathError('');
			}
		} catch {}
	};

	const onBlurProfileJSONPath = () => {
		setHasProfileJSONKeyLostFocus(true);
		try {
			validateConsentKey(profileJSONKey);
			const message = checkProfileConsentLocation(profileConsentLocation, profileSchema);
			if (message !== '') {
				setProfilePathError(message);
				return;
			}
			setProfilePathError('');
		} catch (err) {
			setProfilePathError(err.message);
		}
	};

	const onClickProfileJSONPath = (event: React.MouseEvent) => {
		if (!(event.nativeEvent.composedPath()[0] instanceof HTMLInputElement)) {
			return;
		}
		profilePathDropdownRef.current?.hide();
		event.stopPropagation();
	};

	const onKeyDownProfileJSONPath = (event: React.KeyboardEvent) => {
		if ([' ', 'Enter', 'ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
			event.stopPropagation();
		}
	};

	const onKeyUpProfileJSONPath = (event: React.KeyboardEvent) => {
		if (event.key === ' ') {
			event.stopPropagation();
		}
	};

	const onAddPurposeCode = (index: number) => {
		if (isRemovingPurposeCode) {
			return;
		}
		setNewPurposeCodeIndex(index + 1);
		setPurposeCodes((purposeCodes) => [...purposeCodes.slice(0, index + 1), '', ...purposeCodes.slice(index + 1)]);
	};

	const onRemovePurposeCode = (index: number) => {
		if (isRemovingPurposeCodeRef.current) {
			return;
		}
		isRemovingPurposeCodeRef.current = true;
		setNewPurposeCodeIndex(null);
		setIsRemovingPurposeCode(true);

		const removePurposeCode = () => {
			setPurposeCodes((purposeCodes) => {
				const remaining = purposeCodes.filter((_, i) => i !== index);
				return remaining.length === 0 ? [''] : remaining;
			});
			setIsRemovingPurposeCode(false);
			isRemovingPurposeCodeRef.current = false;
		};
		const row = purposeCodeRowRefs.current[index];
		if (row == null) {
			removePurposeCode();
			return;
		}

		const height = row.getBoundingClientRect().height;
		const marginTop = Number.parseFloat(getComputedStyle(row).marginTop);
		let collapsedHeight = 0;
		// The next row gains the label when the first row is removed. Reserve that space.
		if (index === 0 && purposeCodes.length > 1) {
			const nextRow = purposeCodeRowRefs.current[1];
			if (nextRow != null) {
				const nextMarginTop = Number.parseFloat(getComputedStyle(nextRow).marginTop);
				collapsedHeight = Math.max(0, height - nextRow.getBoundingClientRect().height - nextMarginTop);
			}
		}

		row.style.overflow = 'hidden';
		const animation = row.animate(
			[
				{ height: `${height}px`, marginTop: `${marginTop}px`, opacity: 1, transform: 'translateY(0)' },
				{
					height: `${collapsedHeight}px`,
					marginTop: '0',
					opacity: 0,
					transform: 'translateY(-2px)',
				},
			],
			{ duration: 140, easing: 'ease-in', fill: 'forwards' },
		);
		animation.addEventListener(
			'finish',
			() => {
				// Remove the row in the same frame that restores its natural height to avoid a bounce.
				animation.cancel();
				row.style.removeProperty('overflow');
				flushSync(removePurposeCode);
			},
			{ once: true },
		);
	};

	const onSave = async () => {
		setNameError('');
		setPurposeCodesError('');
		setProfilePathError('');

		const purposeCodesToSave = purposeCodes.filter((purposeCode) => purposeCode !== '');

		try {
			validatePurposeName(name);
		} catch (err) {
			showValidationError(setNameError, err.message);
			return;
		}
		for (const purposeCode of purposeCodesToSave) {
			try {
				validateConsentKey(purposeCode);
			} catch (err) {
				showValidationError(setPurposeCodesError, err.message);
				return;
			}
		}
		if (getDuplicatePurposeCodeError(purposeCodesToSave) !== '') {
			return;
		}
		try {
			if (isProfilePropertyJSON) {
				validateConsentKey(profileJSONKey);
			}
			const message = checkProfileConsentLocation(profileConsentLocation, profileSchema);
			if (message !== '') {
				showValidationError(setProfilePathError, message);
				return;
			}
		} catch (err) {
			showValidationError(setProfilePathError, err.message);
			return;
		}

		const eventConsentLocations = purposeCodesToSave.map((purposeCode) => ({
			purposeCode,
		}));
		setIsSaving(true);
		try {
			if (isEditing) {
				await api.workspaces.updateConsentPurpose(
					purposeToEdit.id,
					name,
					eventConsentLocations,
					profileConsentLocation,
				);
			} else {
				await api.workspaces.addConsentPurpose(name, eventConsentLocations, profileConsentLocation);
			}
		} catch (err) {
			setIsSaving(false);
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

	const nonEmptyPurposeCodes = purposeCodes.filter((purposeCode) => purposeCode !== '').length;

	return (
		<>
			<SlDialog
				className='privacy__dialog'
				label={isEditing ? 'Edit consent purpose' : 'Add consent purpose'}
				open={isOpen}
				onSlRequestClose={(event) => {
					if (event.detail.source === 'overlay') {
						event.preventDefault();
					}
				}}
				onSlInitialFocus={(event) => {
					if (isEditing) {
						event.preventDefault();
					}
				}}
				onSlAfterHide={onSlAfterHide}
			>
				<div className='privacy__dialog-form' ref={formRef}>
					<SlInput
						className='privacy__dialog-name'
						ref={inputRef}
						label='Name'
						value={name}
						onSlInput={onInputName}
					/>
					{nameError && (
						<div className='privacy__dialog-error'>
							<SlIcon slot='icon' name='exclamation-octagon' />
							{nameError}
						</div>
					)}
					<div className='privacy__dialog-paths'>
						{purposeCodes.map((purposeCode, i) => (
							<div
								className={`privacy__dialog-purpose-code-row${i === 0 ? ' privacy__dialog-purpose-code-row--with-label' : ''}`}
								ref={(row) => {
									if (row != null) {
										purposeCodeRowRefs.current[i] = row;
									}
								}}
								key={i}
							>
								<SlInput
									className='privacy__dialog-purpose-code'
									ref={(input) => {
										purposeCodeInputRefs.current[i] = input;
									}}
									value={purposeCode}
									placeholder='purpose code'
									onSlInput={(e) => onInputPurposeCode(e, i)}
									onSlBlur={onBlurPurposeCode}
								>
									{i === 0 && (
										<span
											className='schema-property-grid__label-content privacy__dialog-path-label'
											slot='label'
										>
											Consent in events
											<SchemaPropertyInfoTooltip
												content={
													'In incoming events, consent for this purpose is read from a property in context.consents. Enter the code used for this purpose in the platform you use to manage consent.\n\n' +
													'If you add multiple locations, they are checked from top to bottom. The first location with a consent value is used.'
												}
												label='About consent in events'
											/>
										</span>
									)}
									<span className='privacy__dialog-purpose-code-prefix' slot='prefix'>
										context.consents.
									</span>
								</SlInput>
								<div className='privacy__dialog-purpose-code-actions'>
									<PurposeCodeAction
										className='privacy__dialog-purpose-code-add'
										icon='plus-circle'
										label='Add purpose code'
										isDisabled={purposeCodes.length >= MAX_EVENT_CONSENT_LOCATIONS}
										onClick={() => onAddPurposeCode(i)}
									/>
									{purposeCodes.length > 1 && (purposeCode === '' || nonEmptyPurposeCodes > 1) && (
										<PurposeCodeAction
											className='privacy__dialog-purpose-code-remove'
											icon='x-circle'
											label='Remove purpose code'
											shouldHideTooltipBeforeClick
											onClick={() => onRemovePurposeCode(i)}
										/>
									)}
								</div>
							</div>
						))}
						{(purposeCodesError || duplicatePurposeCodeError) && (
							<div className='privacy__dialog-error'>
								<SlIcon slot='icon' name='exclamation-octagon' />
								{purposeCodesError || duplicatePurposeCodeError}
							</div>
						)}

						<SlDropdown
							className='privacy__dialog-profile-path-dropdown'
							ref={profilePathDropdownRef}
							placement='bottom-start'
							sync='width'
							hoist
						>
							<SlInput
								slot='trigger'
								className={`privacy__dialog-profile-path${isProfilePropertyJSON ? ' privacy__dialog-profile-path--json' : ''}`}
								value={isProfilePropertyJSON ? profileJSONKey : profilePropertyPath}
								onSlInput={isProfilePropertyJSON ? onInputProfileJSONPath : undefined}
								onSlBlur={isProfilePropertyJSON ? onBlurProfileJSONPath : undefined}
								onClickCapture={isProfilePropertyJSON ? onClickProfileJSONPath : undefined}
								onKeyDownCapture={isProfilePropertyJSON ? onKeyDownProfileJSONPath : undefined}
								onKeyUpCapture={isProfilePropertyJSON ? onKeyUpProfileJSONPath : undefined}
								placeholder={isProfilePropertyJSON ? 'key' : 'Select a profile property'}
								readonly={!isProfilePropertyJSON}
							>
								<span
									className='schema-property-grid__label-content privacy__dialog-path-label'
									slot='label'
								>
									Consent in profiles
									<SchemaPropertyInfoTooltip
										content='Consent for this purpose is represented in profiles using a profile property. Select a boolean property, or select a json property and specify the key.'
										label='About profile property'
									/>
								</span>
								{isProfilePropertyJSON && (
									<span className='privacy__dialog-profile-path-prefix' slot='prefix'>
										{profilePropertyPath}.
									</span>
								)}
								{profilePropertyPath !== '' && (
									<SlIconButton
										className='privacy__dialog-profile-path-clear'
										slot='suffix'
										name='x-circle-fill'
										library='system'
										label='Clear profile property'
										onMouseDown={(event) => event.preventDefault()}
										onClickCapture={onClearProfilePath}
									/>
								)}
								<SlIcon slot='suffix' name='chevron-down' />
							</SlInput>
							<SlMenu className='privacy__dialog-profile-path-menu' onSlSelect={onSelectProfilePath}>
								{profilePathOptions.map(([path, property]) => (
									<SlMenuItem key={path} value={path}>
										<span className='privacy__dialog-profile-path-option'>{path}</span>
										<span className='privacy__dialog-profile-path-option-type' slot='suffix'>
											{property.type}
										</span>
									</SlMenuItem>
								))}
							</SlMenu>
						</SlDropdown>
						<div
							className={`privacy__dialog-profile-path-message${profilePathError !== '' ? ' privacy__dialog-error' : profilePathWarning !== '' ? ' privacy__dialog-warning' : ' privacy__dialog-message--hidden'}`}
							aria-hidden={profilePathError !== '' || profilePathWarning !== '' ? undefined : 'true'}
						>
							{profilePathError !== '' ? (
								<>
									<SlIcon slot='icon' name='exclamation-octagon' />
									{profilePathError}
								</>
							) : (
								<>
									<SlIcon name='exclamation-triangle' />
									{profilePathWarning}
								</>
							)}
						</div>
					</div>

					<div className='privacy__dialog-actions'>
						<SlButton onClick={onClose}>Cancel</SlButton>
						<SlButton
							loading={isSaving}
							className='privacy__dialog-save'
							variant='primary'
							onClick={onSave}
						>
							{isEditing ? 'Save' : 'Add'}
						</SlButton>
					</div>
				</div>
			</SlDialog>
		</>
	);
};

interface PurposeCodeActionProps {
	className: string;
	icon: string;
	label: string;
	isDisabled?: boolean;
	shouldHideTooltipBeforeClick?: boolean;
	onClick: () => void;
}

// PurposeCodeAction is a button shown beside a purpose code input, to remove
// that purpose code or add another one.
const PurposeCodeAction = ({
	className,
	icon,
	label,
	isDisabled = false,
	shouldHideTooltipBeforeClick = false,
	onClick,
}: PurposeCodeActionProps) => {
	const tooltipRef = useRef<any>();
	const onActionClick = async () => {
		if (shouldHideTooltipBeforeClick) {
			await tooltipRef.current?.hide();
		}
		onClick();
	};
	return (
		<SlTooltip ref={tooltipRef} className={className} content={label} hoist>
			<SlButton
				className='privacy__dialog-purpose-code-action'
				size='small'
				disabled={isDisabled}
				aria-label={label}
				onClick={onActionClick}
			>
				<SlIcon name={icon} slot='prefix' />
			</SlButton>
		</SlTooltip>
	);
};

export default Privacy;

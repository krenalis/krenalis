import React, { useState, useContext, useEffect, useRef, useMemo, ReactNode } from 'react';
import Section from '../../base/Section/Section';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import Grid from '../../base/Grid/Grid';
import AppContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import { UnprocessableError, NotFoundError } from '../../../lib/api/errors';
import { CONFIRM_ANIMATION_DURATION, ERROR_ANIMATION_DURATION } from './Action.constants';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import {
	AbsolutePathResponse,
	RecordsResponse,
	SheetsResponse,
	ConnectorUIResponse,
} from '../../../lib/api/types/responses';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import TransformedConnector from '../../../lib/core/connector';
import ConnectorFieldInterface from '../../../lib/api/types/ui';
import { redirect } from 'react-router-dom';
import ConnectionContext from '../../../context/ConnectionContext';
import ConnectorField from '../../base/ConnectorFields/ConnectorField';
import ConnectorUI from '../../base/ConnectorUI/ConnectorUI';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import actionContext from '../../../context/ActionContext';
import { flattenSchema } from '../../../lib/core/action';
import { Popover } from '../../base/Popover/Popover';
import {
	filterOrderingPropertySchema,
	getOrderingPropertyPathComboboxItems,
} from '../../helpers/getSchemaComboboxItems';
import { Combobox } from '../../base/Combobox/Combobox';
import { ActionIssues } from './ActionIssues';

const ActionFile = () => {
	const [fileFields, setFileFields] = useState<ConnectorFieldInterface[]>([]);

	const { connectors, api, selectedWorkspace, handleError } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);
	const {
		action,
		setAction,
		setSettings,
		isFormatLoading,
		setIsFormatLoading,
		setIsFormatChanged,
		setIsFileChanged,
		isFormatChanged,
		actionType,
		isEditing,
		setIssues,
	} = useContext(actionContext);

	const formatRef = useRef<string>(action.format);
	const pathInputRef = useRef<any>();

	useEffect(() => {
		if (isFormatChanged && !isFormatLoading) {
			if (pathInputRef.current) {
				setTimeout(() => {
					pathInputRef.current.focus();
				}, 50);
			}
		}
	}, [isFormatChanged, isFormatLoading]);

	useEffect(() => {
		// check if the format has been passed in the query parameters.
		const f = new URL(document.location.href).searchParams.get('format');
		if (f != null) {
			const name = decodeURIComponent(f);
			formatRef.current = name;
			const format = connectors.find((c) => c.name === name);
			const a = { ...action };
			a.format = name;
			a.sheet = format.hasSheets ? '' : null;
			setIsFormatLoading(true);
			setAction(a);
		}
	}, []);

	useEffect(() => {
		const fetchFields = async () => {
			const format = connectors.find((c) => c.name === action.format);
			if (!format.hasSettings(connection.role)) {
				setFileFields([]);
				setTimeout(() => setIsFormatLoading(false), 300);
				return;
			}

			let ui: ConnectorUIResponse;
			if (isEditing && !isFormatChanged) {
				try {
					ui = await api.workspaces.connections.actionUiEvent(action.id, 'load', null);
				} catch (err) {
					setTimeout(() => setIsFormatLoading(false), 300);
					if (err instanceof NotFoundError) {
						redirect('connectors');
						handleError('The format does not exist anymore');
						return;
					}
					if (err instanceof UnprocessableError) {
						if (err.code === 'EventNotExist') {
							handleError(
								'An unexpected error has occurred. Please contact the administrator for more information.',
							);
							return;
						}
					}
					handleError(err);
					return;
				}
			} else {
				try {
					ui = await api.connectors.ui(selectedWorkspace, format.name, connection.role, null);
				} catch (err) {
					setTimeout(() => setIsFormatLoading(false), 300);
					if (err instanceof NotFoundError) {
						redirect('connectors');
						handleError('The format does not exist anymore');
						return;
					}
					if (err instanceof UnprocessableError) {
						if (err.code === 'EventNotExist') {
							console.error(
								`Unprocessable: connector does not implement the 'load' event in its ServeUI method`,
							);
							handleError(
								'An unexpected error has occurred. Please contact the administrator for more information.',
							);
							return;
						}
					}
					handleError(err);
					return;
				}
			}
			setFileFields(ui.fields);
			setSettings(ui.settings);
			setTimeout(() => setIsFormatLoading(false), 300);
		};

		if (action.format == '') {
			return;
		}
		fetchFields();
	}, [formatRef.current]);

	const { hasSheets, icon, fileExtension } = useMemo(() => {
		const format = connectors.find((c) => c.name === action.format);
		return { hasSheets: format?.hasSheets, icon: format?.icon, fileExtension: format?.fileExtension };
	}, [action]);

	const onFormatChange = (e) => {
		const name = e.target.value;
		formatRef.current = name;
		const format = connectors.find((c) => c.name === name);
		const a = { ...action };
		// reset the action.
		a.format = name;
		a.compression = '';
		a.sheet = format.hasSheets ? '' : null;
		a.path = '';
		a.identityColumn = '';
		a.lastChangeTimeColumn = '';
		a.lastChangeTimeFormat = '';
		a.transformation.mapping = flattenSchema(actionType.outputSchema);
		a.transformation.function = null;
		setSettings(null);
		setIsFormatLoading(true);
		setIsFormatChanged(true);
		setIsFileChanged(false);
		setIssues([]);
		setAction(a);
	};

	const formats: TransformedConnector[] = [];
	for (const c of connectors) {
		if (c.isFile) {
			formats.push(c);
		}
	}

	return (
		<Section
			title={`File`}
			className='action__file'
			description='The settings of the file'
			padded={true}
			annotated={true}
		>
			<SlSelect
				label='Format'
				className='action__file-format'
				value={String(action.format)}
				onSlChange={onFormatChange}
			>
				{action.format !== '' && (
					<div className='action__file-format-logo' slot='prefix'>
						<LittleLogo icon={icon} />
					</div>
				)}
				{formats.map((f) => {
					const role = connection.role;
					if (
						(role === 'Source' && f.asSource == null) ||
						(role === 'Destination' && f.asDestination == null)
					) {
						return null;
					}
					return (
						<SlOption key={f.name} value={f.name}>
							<div slot='prefix'>
								<LittleLogo icon={f.icon} />
							</div>
							{f.name}
						</SlOption>
					);
				})}
			</SlSelect>
			{isFormatLoading ? (
				<SlSpinner
					style={
						{
							display: 'block',
							position: 'relative',
							top: '50px',
							margin: 'auto',
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			) : (
				action.format !== '' && (
					<div className='action__file-settings'>
						<FileSettings
							hasSheets={hasSheets}
							fileExtension={fileExtension}
							fileFields={fileFields}
							pathInputRef={pathInputRef}
						/>
					</div>
				)
			)}
		</Section>
	);
};

interface FileSettingsProps {
	hasSheets: boolean;
	fileExtension: string;
	fileFields: ConnectorFieldInterface[];
	pathInputRef: any;
}

const FileSettings = ({ hasSheets, fileExtension, fileFields, pathInputRef }: FileSettingsProps) => {
	const [sheets, setSheets] = useState<Record<string, string>>({});
	const [areSheetsLoading, setAreSheetsLoading] = useState<boolean>(false);
	const [hasSheetsError, setHasSheetsError] = useState<boolean>(false);
	const [absolutePath, setAbsolutePath] = useState<string>('');
	const [absolutePathError, setAbsolutePathError] = useState<string>('');
	const [filePreviewColumns, setFilePreviewColumns] = useState<GridColumn[] | null>(null);
	const [filePreviewRows, setFilePreviewRows] = useState<GridRow[] | null>(null);
	const [filePreviewIssues, setFilePreviewIssues] = useState<string[]>([]);
	const [showFilePreviewContent, setShowFilePreviewContent] = useState<boolean>(false);
	const [isLoadingPreview, setIsLoadingPreview] = useState<boolean>(false);

	const { handleError, api } = useContext(AppContext);
	const {
		connection,
		action,
		setAction,
		settings,
		setSettings,
		actionType,
		setActionType,
		isImport,
		transformationSectionRef,
		setIsFileChanged,
		setIsFormatChanged,
		isFormatChanged,
		isTransformationDisabled,
		isEditing,
		setIssues,
		setShowIssues,
	} = useContext(ActionContext);

	const getAbsolutePathTimeoutID = useRef<number>();
	const sheetsSelectRef = useRef<any>();
	const fileConfirmButtonRef = useRef<any>();

	const pathRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
		lastSheetFetch: '',
	});

	const sheetRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});

	const compressionRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});

	const settingsRef = useRef({
		lastConfirmation: {},
		lastUpdate: {},
	});

	const hasRecordsError = useRef(false);

	const fieldsToRender = useMemo(() => {
		const fields: ReactNode[] = [];
		for (const f of fileFields) {
			fields.push(<ConnectorField key={f.label} field={f} />);
		}
		return fields;
	}, [fileFields]);

	const orderByError = useMemo<string>(() => {
		const filteredSchema = filterOrderingPropertySchema(actionType.inputSchema);
		const property = action.orderBy;
		if (filteredSchema == null || property === '') {
			return '';
		}
		if (filteredSchema[property] == null) {
			return `property "${property}" does not exist in the user schema`;
		}
		return '';
	}, [action, actionType]);

	useEffect(() => {
		pathRef.current = {
			...pathRef.current,
			lastConfirmation: action.path,
			lastUpdate: action.path,
		};
		sheetRef.current = {
			lastConfirmation: action.sheet,
			lastUpdate: action.sheet,
		};
		compressionRef.current = {
			lastConfirmation: action.compression,
			lastUpdate: action.compression,
		};
		settingsRef.current = {
			lastConfirmation: { ...settings },
			lastUpdate: { ...settings },
		};
	}, []);

	useEffect(() => {
		if (isImport || isEditing || actionType.target !== 'Users') {
			return;
		}
		const hasEmailColumn = actionType.inputSchema.properties.find((p) => p.name === 'email');
		if (hasEmailColumn) {
			const a = { ...action };
			a.orderBy = 'email';
			setAction(a);
		}
	}, []);

	useEffect(() => {
		if (isEditing) {
			if (action.path === '') {
				// The user has changed the format of an already saved
				// action, so the action path is now empty.
				return;
			}
			fetchPath(action.path);
		}
	}, []);

	useEffect(() => {
		const fetchSheets = async () => {
			pathRef.current.lastSheetFetch = pathRef.current.lastUpdate;
			let res: SheetsResponse;
			try {
				res = await api.workspaces.connections.sheets(
					connection.id,
					action.path!,
					action.format,
					action.compression,
					settings,
				);
			} catch (err) {
				if (err instanceof UnprocessableError) {
					handleError(err.message);
				} else {
					handleError(err);
				}
				setHasSheetsError(true);
				return;
			}
			const sheets = {};
			let i = 1;
			for (const s of res.sheets) {
				sheets[i] = s;
				i++;
			}
			setSheets(sheets);
		};
		if (!hasSheets || action.path === '') {
			return;
		}
		if (connection.isSource) {
			fetchSheets();
		}
	}, []);

	const checkIsFileChanged = () => {
		if (isFormatChanged) {
			return;
		}
		const isPathChanged = pathRef.current.lastUpdate !== pathRef.current.lastConfirmation;
		const isSheetChanged = sheetRef.current.lastUpdate !== sheetRef.current.lastConfirmation;
		const isCompressionChanged = compressionRef.current.lastUpdate !== compressionRef.current.lastConfirmation;
		let areSettingsChanged = false;
		for (const key in settingsRef.current.lastUpdate) {
			if (settingsRef.current.lastUpdate.hasOwnProperty(key)) {
				if (settingsRef.current.lastUpdate[key] != settingsRef.current.lastConfirmation[key]) {
					areSettingsChanged = true;
					break;
				}
			}
		}
		if (isPathChanged || isSheetChanged || isCompressionChanged || areSettingsChanged || hasRecordsError.current) {
			setIsFileChanged(true);
		} else {
			setIsFileChanged(false);
		}
	};

	const onUpdatePath = async (e) => {
		clearTimeout(getAbsolutePathTimeoutID.current);
		const a = { ...action };
		const path = e.currentTarget.value;
		pathRef.current.lastUpdate = path;
		checkIsFileChanged();
		a.path = path;
		if (a.sheet != null && isImport) {
			a.sheet = '';
		}
		setAction(a);
		setAbsolutePath('');
		setAbsolutePathError('');
		if (path === '') {
			return;
		}
		getAbsolutePathTimeoutID.current = window.setTimeout(async () => {
			await fetchPath(path);
		}, 300);
	};

	const fetchPath = async (path: string) => {
		let res: AbsolutePathResponse;
		try {
			res = await api.workspaces.connections.absolutePath(connection.id, path);
		} catch (err) {
			if (err instanceof UnprocessableError && err.code === 'InvalidPath') {
				setAbsolutePathError(err.message);
				return;
			}
			handleError(err);
			return;
		}
		setAbsolutePath(res.path);
	};

	const onUpdateSheet = (e) => {
		const a = { ...action };
		let sheet: string;
		if (e.type === 'sl-change') {
			sheet = sheets[e.currentTarget.value];
		} else {
			sheet = e.currentTarget.value;
		}
		sheetRef.current.lastUpdate = sheet;
		checkIsFileChanged();
		a.sheet = sheet;
		setAction(a);
	};

	const loadSheets = async () => {
		const a = { ...action };
		a.sheet = '';
		setAction(a);
		sheetsSelectRef.current.classList.add('action__file-sheets--hide-list-box'); // prevent the listbox from flashing.
		setSheets({});
		setAreSheetsLoading(true);
		pathRef.current.lastSheetFetch = pathRef.current.lastUpdate;
		let res: SheetsResponse;
		try {
			res = await api.workspaces.connections.sheets(
				connection.id,
				action.path!,
				action.format,
				action.compression,
				settings,
			);
		} catch (err) {
			setTimeout(() => {
				if (err instanceof UnprocessableError) {
					handleError(err.message);
				} else {
					handleError(err);
				}
				sheetsSelectRef.current.classList.remove('action__file-sheets--hide-list-box');
				setHasSheetsError(true);
				setAreSheetsLoading(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			setHasSheetsError(false);
			setAreSheetsLoading(false);
			const sheets = {};
			let i = 1;
			for (const s of res.sheets) {
				sheets[i] = s;
				i++;
			}
			setSheets(sheets);
			sheetsSelectRef.current.classList.remove('action__file-sheets--hide-list-box');
			setTimeout(() => {
				sheetsSelectRef.current.show();
			});
		}, 300);
	};

	const onSheetsLoad = async () => {
		if (pathRef.current.lastSheetFetch === pathRef.current.lastUpdate) {
			return;
		}
		await loadSheets();
	};

	const onSheetsReload = async () => {
		await loadSheets();
	};

	const onCompressionChange = (e) => {
		const a = { ...action };
		const compression = e.target.value;
		compressionRef.current.lastUpdate = compression;
		checkIsFileChanged();
		a.compression = compression;
		setAction(a);
	};

	const onOrderByChange = (_: string, value: string) => {
		const a = { ...action };
		a.orderBy = value;
		setAction(a);
	};

	const onOrderBySelect = (_: string, value: string) => {
		const a = { ...action };
		a.orderBy = value;
		setAction(a);
	};

	const onFieldChange = (name: string, value: any) => {
		settingsRef.current.lastUpdate[name] = value;
		checkIsFileChanged();
		setSettings({ ...settings, [name]: value });
	};

	const onFilePreview = async () => {
		if (action.path === '') {
			handleError('Please enter a path');
			return;
		}
		if (hasSheets && action.sheet === '') {
			handleError('Please enter a sheet');
			return;
		}
		setFilePreviewIssues([]);
		setIsLoadingPreview(true);
		let res: RecordsResponse;
		try {
			res = await records(20);
		} catch (err) {
			setIsLoadingPreview(false);
			return;
		}
		setShowIssues(false);
		const columns: GridColumn[] = [];
		if (res.schema != null) {
			for (const prop of res.schema.properties!) {
				columns.push({ name: prop.name, type: prop.type.kind });
			}
			const areExcelLike = areColumnsExcelLike(columns);
			if (areExcelLike) {
				for (const column of columns) {
					column.alignment = 'header-center';
				}
			}
		}
		setFilePreviewColumns(columns);
		const rows: GridRow[] = [];
		for (const record of res.records) {
			const row = [];
			for (const property of res.schema.properties) {
				const value = record[property.name];
				row.push(value);
			}
			rows.push({ cells: row });
		}
		setFilePreviewRows(rows);
		setFilePreviewIssues(res.issues);
		setIsLoadingPreview(false);
	};

	const onConfirmFile = async () => {
		if (action.path === '') {
			handleError('Please enter a path');
			return;
		}
		if (hasSheets && action.sheet === '') {
			handleError('Please enter a sheet');
			return;
		}
		setIssues([]);
		fileConfirmButtonRef.current!.load();
		let res: RecordsResponse;
		try {
			res = await records(0, true);
		} catch (err) {
			fileConfirmButtonRef.current!.stop();
			return;
		}
		if (res.schema == null) {
			fileConfirmButtonRef.current.error("This file doesn't have any compatible column");
			setTimeout(() => {
				setIssues(res.issues);
				const actionTyp = { ...actionType };
				actionTyp.inputSchema = null;
				setActionType(actionTyp);
			}, ERROR_ANIMATION_DURATION);
		} else {
			fileConfirmButtonRef.current!.confirm();
			setTimeout(() => {
				setIssues(res.issues);
				const actionTyp = { ...actionType };
				actionTyp.inputSchema = res.schema;
				setActionType(actionTyp);
				setIsFormatChanged(false);
				setTimeout(() => {
					const top = transformationSectionRef.current!.getBoundingClientRect().top;
					transformationSectionRef.current!.closest('.fullscreen').scrollBy({
						top: top - 130,
						left: 0,
						behavior: 'smooth',
					});
				}, 100);
			}, CONFIRM_ANIMATION_DURATION);
		}
	};

	const records = async (limit: number, isConfirmation?: boolean) => {
		let res: RecordsResponse;
		try {
			res = await api.workspaces.connections.records(
				connection.id,
				action.path,
				action.format,
				action.sheet === undefined ? null : action.sheet,
				action.compression,
				settings,
				limit,
			);
		} catch (err) {
			handleError(err);
			const isAlreadyConfirmed = actionType.inputSchema != null;
			if (isAlreadyConfirmed) {
				hasRecordsError.current = true;
				checkIsFileChanged();
			}
			throw err;
		}
		if (isConfirmation) {
			pathRef.current.lastConfirmation = action.path;
			compressionRef.current.lastConfirmation = action.compression;
			settingsRef.current.lastConfirmation = { ...settings };
			if (action.sheet != null) {
				sheetRef.current.lastConfirmation = action.sheet;
			}
			hasRecordsError.current = false;
			setIsFileChanged(false);
		}
		return res;
	};

	return (
		<div>
			<div className='action__file-path-wrapper'>
				<SlInput
					className='action__file-path'
					name='path'
					value={action.path!}
					type='text'
					onSlInput={onUpdatePath}
					placeholder={`${actionType.target.toLowerCase()}.${fileExtension}`}
					ref={pathInputRef}
				>
					<div className='action__file-path-label' slot='label'>
						<div className='action__file-path-text'>Path</div>
						<div className='action__file-path-description'>
							The path of the file.
							{connection.role == 'Destination'
								? ' You can use the ${now}, ${today} and ${unix} placeholders.'
								: ''}
						</div>
					</div>
				</SlInput>
				<div
					className={`action__file-complete-path-error${absolutePathError !== '' ? ' action__file-complete-path-error--visible' : ''}`}
				>
					{absolutePathError}
				</div>
				<div
					className={`action__file-complete-path${absolutePath !== '' ? ' action__file-complete-path--visible' : ''}`}
				>
					{absolutePath}
				</div>
			</div>
			{hasSheets &&
				(connection.role === 'Source' ? (
					<div className='action__file-sheets-wrapper'>
						<SlSelect
							onSlFocus={onSheetsLoad}
							className='action__file-sheets'
							ref={sheetsSelectRef}
							name='sheet'
							value={
								action.sheet == null
									? undefined
									: Object.keys(sheets).find((k) => sheets[k] === action.sheet)
							}
							label='Sheet'
							onSlChange={onUpdateSheet}
							disabled={
								action.path == null ||
								action.path === '' ||
								absolutePathError !== '' ||
								areSheetsLoading ||
								(pathRef.current.lastSheetFetch === pathRef.current.lastUpdate && hasSheetsError)
							}
						>
							{areSheetsLoading && <SlSpinner slot='prefix' />}
							{Object.entries(sheets).map(([i, sheet]) => {
								return (
									<SlOption key={i} value={i}>
										{sheet}
									</SlOption>
								);
							})}
						</SlSelect>
						<SlButton
							onClick={onSheetsReload}
							disabled={action.path == null || action.path === '' || areSheetsLoading}
						>
							<SlIcon slot='prefix' name='arrow-clockwise' />
							Reload
						</SlButton>
					</div>
				) : (
					<SlInput
						className='action__file-sheets-input'
						name='input'
						value={action.sheet!}
						label='Sheet'
						type='text'
						onSlInput={onUpdateSheet}
					/>
				))}
			<SlSelect
				className='action__file-compression'
				name='compression'
				value={action.compression}
				label='Compression'
				onSlChange={onCompressionChange}
			>
				<SlOption value=''>None</SlOption>
				<SlOption value='Zip'>Zip</SlOption>
				<SlOption value='Gzip'>Gzip</SlOption>
				<SlOption value='Snappy'>Snappy</SlOption>
			</SlSelect>
			{actionType.fields.includes('OrderBy') && (
				<div className='action__file-ordering'>
					<Combobox
						value={action.orderBy}
						onInput={onOrderByChange}
						onSelect={onOrderBySelect}
						items={getOrderingPropertyPathComboboxItems(actionType.inputSchema)}
						label='Order users by'
						isExpression={false}
						error={orderByError && orderByError}
						name='ordering'
						caret={true}
					/>
				</div>
			)}
			{fieldsToRender.length > 0 && (
				<ConnectorUI fields={fieldsToRender} settings={settings} onChange={onFieldChange} />
			)}
			{isImport && (
				<div className='action__file-buttons'>
					<SlButton
						className='action__file-preview'
						variant='neutral'
						size='small'
						onClick={onFilePreview}
						loading={isLoadingPreview}
						disabled={isLoadingPreview}
					>
						Preview
					</SlButton>
					<FeedbackButton
						ref={fileConfirmButtonRef}
						className='action__file-confirm'
						variant='success'
						size='small'
						onClick={onConfirmFile}
						animationDuration={CONFIRM_ANIMATION_DURATION}
					>
						Confirm
					</FeedbackButton>
					<Popover
						isOpen={isTransformationDisabled}
						content='Confirm when you have finished editing the file settings.'
					/>
				</div>
			)}
			<SlDrawer
				className='action__file-preview-drawer'
				open={filePreviewColumns != null && filePreviewRows != null}
				onSlAfterShow={() => setShowFilePreviewContent(true)}
				onSlAfterHide={(e: any) => {
					if (e.target.classList.contains('action__issues')) {
						e.stopPropagation();
						return;
					}
					setFilePreviewColumns(null);
					setFilePreviewRows(null);
					setShowFilePreviewContent(false);
					setShowIssues(true);
				}}
				placement='bottom'
				style={{ '--size': '600px' } as React.CSSProperties}
			>
				<div className='action__file-preview-drawer-label' slot='label'>
					<span>File Preview</span>
					{showFilePreviewContent && (
						<ActionIssues
							issues={filePreviewIssues}
							type={connection.connector.type}
							role={connection.role}
						/>
					)}
				</div>
				{showFilePreviewContent ? (
					<Grid
						columns={filePreviewColumns!}
						rows={filePreviewRows!}
						showColumnBorder={true}
						showRowBorder={true}
						noRowsMessage={
							filePreviewColumns.length === 0
								? "This file doesn't have any compatible column"
								: 'This file did not return data'
						}
					/>
				) : (
					<SlSpinner
						style={
							{
								fontSize: '3rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					></SlSpinner>
				)}
			</SlDrawer>
		</div>
	);
};

const areColumnsExcelLike = (columns: GridColumn[]): boolean => {
	let lastColumn = -1;
	for (const column of columns) {
		const current = fromColumnNameToNumber(column.name);
		if (current <= lastColumn) {
			return false;
		}
		lastColumn = current;
	}
	return true;
};

const ALPHABET_LENGTH = 26;
const fromColumnNameToNumber = (columnName: string): number => {
	let num = 0;
	for (let i = 0; i < columnName.length; i++) {
		// get the position of the character in the alphabet.
		const alphabetPosition = columnName.charCodeAt(i) - 'A'.charCodeAt(0) + 1;
		num = num * ALPHABET_LENGTH + alphabetPosition;
	}
	return num;
};

export default ActionFile;

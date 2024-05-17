import React, { useState, useContext, useEffect, useRef, useMemo, ReactNode } from 'react';
import Section from '../../shared/Section/Section';
import FeedbackButton from '../../shared/FeedbackButton/FeedbackButton';
import Grid from '../../shared/Grid/Grid';
import AppContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import { UnprocessableError, NotFoundError, BadRequestError } from '../../../lib/api/errors';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import {
	CompletePathResponse,
	RecordsResponse,
	SheetsResponse,
	ConnectorUIResponse,
} from '../../../types/external/api';
import { GridColumn, GridRow } from '../../shared/Grid/Grid.types';
import TransformedConnector from '../../../lib/helpers/transformedConnector';
import ConnectorFieldInterface from '../../../types/external/ui';
import { redirect } from 'react-router-dom';
import ConnectionContext from '../../../context/ConnectionContext';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import ConnectorUI from '../../shared/ConnectorUI/ConnectorUI';
import LittleLogo from '../../shared/LittleLogo/LittleLogo';
import actionContext from '../../../context/ActionContext';
import { flattenSchema } from '../../../lib/helpers/transformedAction';
import { Popover } from '../../shared/Popover/Popover';

const ActionFile = () => {
	const [fileFields, setFileFields] = useState<ConnectorFieldInterface[]>([]);

	const { connectors, api, selectedWorkspace, handleError } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);
	const {
		action,
		setAction,
		setValues,
		isFileConnectorLoading,
		setIsFileConnectorLoading,
		setIsFileConnectorChanged,
		setIsFileChanged,
		isFileConnectorChanged,
		actionType,
		isEditing,
	} = useContext(actionContext);

	const fileConnectorRef = useRef<string>(action.Connector);
	const pathInputRef = useRef<any>();

	useEffect(() => {
		if (isFileConnectorChanged && !isFileConnectorLoading) {
			if (pathInputRef.current) {
				setTimeout(() => {
					pathInputRef.current.focus();
				}, 50);
			}
		}
	}, [isFileConnectorChanged, isFileConnectorLoading]);

	useEffect(() => {
		// check if the file connector id has been passed in the query
		// parameters.
		const f = new URL(document.location.href).searchParams.get('fileConnector');
		if (f != null) {
			const name = decodeURIComponent(f);
			fileConnectorRef.current = name;
			const connector = connectors.find((c) => c.name === name);
			const a = { ...action };
			a.Connector = name;
			a.Sheet = connector.hasSheets ? '' : null;
			setIsFileConnectorLoading(true);
			setAction(a);
		}
	}, []);

	useEffect(() => {
		const fetchFields = async () => {
			const connector = connectors.find((c) => c.name === action.Connector);
			if (connector.hasUI === false) {
				setFileFields([]);
				setTimeout(() => setIsFileConnectorLoading(false), 300);
				return;
			}

			let ui: ConnectorUIResponse;
			if (isEditing && !isFileConnectorChanged) {
				try {
					ui = await api.workspaces.connections.actionUiEvent(connection.id, action.ID, 'load', null);
				} catch (err) {
					setTimeout(() => setIsFileConnectorLoading(false), 300);
					if (err instanceof NotFoundError) {
						redirect('connectors');
						handleError('The connector does not exist anymore');
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
					ui = await api.connectors.ui(selectedWorkspace, connector.name, connection.role, null);
				} catch (err) {
					setTimeout(() => setIsFileConnectorLoading(false), 300);
					if (err instanceof NotFoundError) {
						redirect('connectors');
						handleError('The connector does not exist anymore');
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
			setFileFields(ui.Fields);
			setValues(ui.Values);
			setTimeout(() => setIsFileConnectorLoading(false), 300);
		};

		if (action.Connector == '') {
			return;
		}
		fetchFields();
	}, [fileConnectorRef.current]);

	const { hasSheets, icon, fileExtension } = useMemo(() => {
		const connector = connectors.find((c) => c.name === action.Connector);
		return { hasSheets: connector?.hasSheets, icon: connector?.icon, fileExtension: connector?.fileExtension };
	}, [action]);

	const onFileConnectorChange = (e) => {
		const name = e.target.value;
		fileConnectorRef.current = name;
		const connector = connectors.find((c) => c.name === name);
		const a = { ...action };
		// reset the action.
		a.Connector = name;
		a.Compression = '';
		a.Sheet = connector.hasSheets ? '' : null;
		a.Path = '';
		a.IdentityProperty = '';
		a.DisplayedProperty = '';
		a.LastChangeTimeProperty = '';
		a.LastChangeTimeFormat = '';
		a.Transformation.Mapping = flattenSchema(actionType.OutputSchema);
		a.Transformation.Function = null;
		setValues(null);
		setIsFileConnectorLoading(true);
		setIsFileConnectorChanged(true);
		setIsFileChanged(false);
		setAction(a);
	};

	const fileConnectors: TransformedConnector[] = [];
	for (const c of connectors) {
		if (c.isFile) {
			fileConnectors.push(c);
		}
	}

	return (
		<Section title={`File`} className='action__file' description='The settings of the file' padded>
			<SlSelect
				label='Type'
				className='action__file-connector'
				value={String(action.Connector)}
				onSlChange={onFileConnectorChange}
			>
				{action.Connector !== '' && (
					<div className='action__file-connector-logo' slot='prefix'>
						<LittleLogo icon={icon} />
					</div>
				)}
				{fileConnectors.map((f) => (
					<SlOption key={f.name} value={f.name}>
						<div slot='prefix'>
							<LittleLogo icon={f.icon} />
						</div>
						{f.name}
					</SlOption>
				))}
			</SlSelect>
			{isFileConnectorLoading ? (
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
				action.Connector !== '' && (
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
	const [completePath, setCompletePath] = useState<string>('');
	const [completePathError, setCompletePathError] = useState<string>('');
	const [filePreviewColumns, setFilePreviewColumns] = useState<GridColumn[] | null>(null);
	const [filePreviewRows, setFilePreviewRows] = useState<GridRow[] | null>(null);
	const [showFilePreviewContent, setShowFilePreviewContent] = useState<boolean>(false);

	const { handleError, api } = useContext(AppContext);
	const {
		connection,
		action,
		setAction,
		values,
		setValues,
		actionType,
		setActionType,
		isImport,
		transformationSectionRef,
		setIsFileChanged,
		setIsFileConnectorChanged,
		isFileConnectorChanged,
		isTransformationDisabled,
	} = useContext(ActionContext);

	const getCompletePathTimeoutID = useRef<number>();
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

	const fieldsToRender = useMemo(() => {
		const fields: ReactNode[] = [];
		for (const f of fileFields) {
			fields.push(<ConnectorField key={f.Label} field={f} />);
		}
		return fields;
	}, [fileFields]);

	useEffect(() => {
		pathRef.current = {
			...pathRef.current,
			lastConfirmation: action.Path,
			lastUpdate: action.Path,
		};
		sheetRef.current = {
			lastConfirmation: action.Sheet,
			lastUpdate: action.Sheet,
		};
		compressionRef.current = {
			lastConfirmation: action.Compression,
			lastUpdate: action.Compression,
		};
		settingsRef.current = {
			lastConfirmation: { ...values },
			lastUpdate: { ...values },
		};
	}, []);

	useEffect(() => {
		const fetchSheets = async () => {
			pathRef.current.lastSheetFetch = pathRef.current.lastUpdate;
			let res: SheetsResponse;
			try {
				res = await api.workspaces.connections.sheets(
					connection.id,
					action.Connector,
					action.Path!,
					action.Compression,
					values,
				);
			} catch (err) {
				if (err instanceof UnprocessableError || err instanceof BadRequestError) {
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
		if (!hasSheets || action.Path === '') {
			return;
		}
		if (connection.isSource) {
			fetchSheets();
		}
	}, []);

	const checkIsFileChanged = () => {
		if (isFileConnectorChanged) {
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
		if (isPathChanged || isSheetChanged || isCompressionChanged || areSettingsChanged) {
			setIsFileChanged(true);
		} else {
			setIsFileChanged(false);
		}
	};

	const onUpdatePath = async (e) => {
		clearTimeout(getCompletePathTimeoutID.current);
		const a = { ...action };
		const path = e.currentTarget.value;
		pathRef.current.lastUpdate = path;
		checkIsFileChanged();
		a.Path = path;
		if (a.Sheet != null) {
			a.Sheet = '';
		}
		setAction(a);
		setCompletePath('');
		setCompletePathError('');
		if (path === '') {
			return;
		}
		getCompletePathTimeoutID.current = window.setTimeout(async () => {
			let res: CompletePathResponse;
			try {
				res = await api.workspaces.connections.completePath(connection.id, path);
			} catch (err) {
				if (err instanceof UnprocessableError && err.code === 'InvalidPath') {
					setCompletePathError(err.message);
					return;
				}
				handleError(err);
				return;
			}
			setCompletePath(res.path);
		}, 300);
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
		a.Sheet = sheet;
		setAction(a);
	};

	const loadSheets = async () => {
		const a = { ...action };
		a.Sheet = '';
		setAction(a);
		sheetsSelectRef.current.classList.add('action__file-sheets--hide-list-box'); // prevent the listbox from flashing.
		setSheets({});
		setAreSheetsLoading(true);
		pathRef.current.lastSheetFetch = pathRef.current.lastUpdate;
		let res: SheetsResponse;
		try {
			res = await api.workspaces.connections.sheets(
				connection.id,
				action.Connector,
				action.Path!,
				action.Compression,
				values,
			);
		} catch (err) {
			setTimeout(() => {
				if (err instanceof UnprocessableError || err instanceof BadRequestError) {
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
		a.Compression = compression;
		setAction(a);
	};

	const onFieldChange = (name: string, value: any) => {
		settingsRef.current.lastUpdate[name] = value;
		checkIsFileChanged();
		setValues({ ...values, [name]: value });
	};

	const onFilePreview = async () => {
		if (action.Path === '') {
			handleError('Please enter a path');
			return;
		}
		if (hasSheets && action.Sheet === '') {
			handleError('Please enter a sheet');
			return;
		}
		const res = await records(20);
		if (res == null) {
			return;
		}
		const columns: GridColumn[] = [];
		for (const prop of res.schema.properties!) {
			let name: string;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		const areExcelLike = areColumnsExcelLike(columns);
		if (areExcelLike) {
			for (const column of columns) {
				column.alignment = 'header-center';
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
	};

	const onConfirmFile = async () => {
		if (action.Path === '') {
			handleError('Please enter a path');
			return;
		}
		if (hasSheets && action.Sheet === '') {
			handleError('Please enter a sheet');
			return;
		}
		fileConfirmButtonRef.current!.load();
		const res = await records(0, true);
		if (res == null) {
			fileConfirmButtonRef.current!.stop();
			return;
		}
		fileConfirmButtonRef.current!.confirm();
		setTimeout(() => {
			const actionTyp = { ...actionType };
			actionTyp.InputSchema = res.schema;
			setActionType(actionTyp);
			setIsFileConnectorChanged(false);
			setTimeout(() => {
				const top = transformationSectionRef.current!.getBoundingClientRect().top;
				transformationSectionRef.current!.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			}, 100);
		}, CONFIRM_ANIMATION_DURATION);
	};

	const records = async (limit: number, isConfirmation?: boolean) => {
		let res: RecordsResponse;
		try {
			res = await api.workspaces.connections.records(
				connection.id,
				action.Connector,
				action.Path,
				action.Sheet === undefined ? null : action.Sheet,
				action.Compression,
				values,
				limit,
			);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ReadFileFailed':
						handleError(err.message);
						break;
					default:
						handleError(err);
				}
				return;
			}
			handleError(err);
			return;
		}
		if (isConfirmation) {
			pathRef.current.lastConfirmation = action.Path;
			compressionRef.current.lastConfirmation = action.Compression;
			settingsRef.current.lastConfirmation = { ...values };
			if (action.Sheet != null) {
				sheetRef.current.lastConfirmation = action.Sheet;
			}
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
					value={action.Path!}
					type='text'
					onSlInput={onUpdatePath}
					placeholder={`${actionType.Target.toLowerCase()}.${fileExtension}`}
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
					className={`action__file-complete-path-error${completePathError !== '' ? ' action__file-complete-path-error--visible' : ''}`}
				>
					{completePathError}
				</div>
				<div
					className={`action__file-complete-path${completePath !== '' ? ' action__file-complete-path--visible' : ''}`}
				>
					{completePath}
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
								action.Sheet == null
									? undefined
									: Object.keys(sheets).find((k) => sheets[k] === action.Sheet)
							}
							label='Sheet'
							onSlChange={onUpdateSheet}
							disabled={
								action.Path == null ||
								action.Path === '' ||
								completePathError !== '' ||
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
							disabled={action.Path == null || action.Path === '' || areSheetsLoading}
						>
							<SlIcon slot='prefix' name='arrow-clockwise' />
							Reload
						</SlButton>
					</div>
				) : (
					<SlInput
						className='action__file-sheets-input'
						name='input'
						value={action.Sheet!}
						label='Sheet'
						type='text'
						onSlInput={onUpdateSheet}
					/>
				))}
			<SlSelect
				className='action__file-compression'
				name='compression'
				value={action.Compression}
				label='Compression'
				onSlChange={onCompressionChange}
			>
				<SlOption value=''>None</SlOption>
				<SlOption value='Zip'>Zip</SlOption>
				<SlOption value='Gzip'>Gzip</SlOption>
				<SlOption value='Snappy'>Snappy</SlOption>
			</SlSelect>
			{fieldsToRender.length > 0 && (
				<ConnectorUI fields={fieldsToRender} values={values} onChange={onFieldChange} />
			)}
			{isImport && (
				<div className='action__file-buttons'>
					<SlButton variant='neutral' size='small' onClick={onFilePreview}>
						Preview
					</SlButton>
					<FeedbackButton
						ref={fileConfirmButtonRef}
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
				label='File Preview'
				open={filePreviewColumns != null && filePreviewRows != null}
				onSlAfterShow={() => setShowFilePreviewContent(true)}
				onSlAfterHide={() => {
					setFilePreviewColumns(null);
					setFilePreviewRows(null);
					setShowFilePreviewContent(false);
				}}
				placement='bottom'
				style={{ '--size': '600px' } as React.CSSProperties}
			>
				{showFilePreviewContent ? (
					<Grid
						columns={filePreviewColumns!}
						rows={filePreviewRows!}
						showColumnBorder={true}
						showRowBorder={true}
						noRowsMessage={'Your file did not return data'}
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

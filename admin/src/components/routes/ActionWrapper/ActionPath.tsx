import React, { useState, useContext, useEffect, useRef } from 'react';
import Section from '../../shared/Section/Section';
import ConfirmationButton from '../../shared/ConfirmationButton/ConfirmationButton';
import Grid from '../../shared/Grid/Grid';
import { AppContext } from '../../../context/providers/AppProvider';
import ActionContext from '../../../context/ActionContext';
import { UnprocessableError, NotFoundError, BadRequestError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import * as variants from '../../../constants/variants';
import * as icons from '../../../constants/icons';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';

import { CompletePathResponse, RecordsResponse, SheetsResponse } from '../../../types/external/api';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';

const ActionPath = () => {
	const [sheets, setSheets] = useState<string[]>([]);
	const [areSheetsLoading, setAreSheetsLoading] = useState<boolean>(false);
	const [hasSheetsError, setHasSheetsError] = useState<boolean>(false);
	const [completePath, setCompletePath] = useState<string>('');
	const [completePathError, setCompletePathError] = useState<string>('');
	const [filePreviewColumns, setFilePreviewColumns] = useState<GridColumn[] | null>(null);
	const [filePreviewRows, setFilePreviewRows] = useState<GridRow[] | null>(null);
	const [showFilePreviewContent, setShowFilePreviewContent] = useState<boolean>(false);

	const { showStatus, showError, api, setAreConnectionsStale } = useContext(AppContext);
	const { connection, action, setAction, actionType, setActionType, isImport, mappingSectionRef, setIsFileChanged } =
		useContext(ActionContext);

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

	useEffect(() => {
		pathRef.current = {
			...pathRef.current,
			lastConfirmation: action.Path!,
			lastUpdate: action.Path!,
		};
		sheetRef.current = {
			lastConfirmation: action.Sheet!,
			lastUpdate: action.Sheet!,
		};
	}, []);

	const onUpdatePath = async (e) => {
		clearTimeout(getCompletePathTimeoutID.current);
		const a = { ...action };
		const path = e.currentTarget.value;
		pathRef.current.lastUpdate = path;
		if (
			pathRef.current.lastUpdate !== pathRef.current.lastConfirmation &&
			pathRef.current.lastConfirmation !== ''
		) {
			setIsFileChanged(true);
		} else {
			setIsFileChanged(false);
		}
		a.Path = path;
		a.Sheet = '';
		setAction(a);
		setCompletePath('');
		setCompletePathError('');
		if (path === '' || connection.storage === 0) {
			return;
		}
		getCompletePathTimeoutID.current = window.setTimeout(async () => {
			let res: CompletePathResponse;
			try {
				res = await api.workspaces.connections.completePath(connection.storage, path);
			} catch (err) {
				if (err instanceof UnprocessableError && err.code === 'InvalidPath') {
					setCompletePathError(err.message);
					return;
				}
				if (err instanceof NotFoundError) {
					showStatus(statuses.linkedStorageDoesNotExistAnymore);
					const cn = { ...connection };
					cn.storage = 0;
					setAreConnectionsStale(true);
					return;
				}
				showError(err);
				return;
			}
			setCompletePath(res.path);
		}, 300);
	};

	const onUpdateSheet = (e) => {
		const a = { ...action };
		const sheet = e.currentTarget.value;
		sheetRef.current.lastUpdate = sheet;
		if (
			sheetRef.current.lastUpdate !== sheetRef.current.lastConfirmation &&
			sheetRef.current.lastConfirmation !== ''
		) {
			setIsFileChanged(true);
		} else {
			setIsFileChanged(false);
		}
		a.Sheet = sheet;
		setAction(a);
	};

	const loadSheets = async () => {
		const a = { ...action };
		a.Sheet = '';
		setAction(a);
		sheetsSelectRef.current.classList.add('hideListbox'); // prevent the listbox from flashing.
		setSheets([]);
		setAreSheetsLoading(true);
		pathRef.current.lastSheetFetch = pathRef.current.lastUpdate;
		let res: SheetsResponse;
		try {
			res = await api.workspaces.connections.sheets(connection.id, action.Path!);
		} catch (err) {
			setTimeout(() => {
				if (err instanceof UnprocessableError || err instanceof BadRequestError) {
					showError(err.message);
				} else {
					showError(err);
				}
				sheetsSelectRef.current.classList.remove('hideListbox');
				setHasSheetsError(true);
				setAreSheetsLoading(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			setHasSheetsError(false);
			setAreSheetsLoading(false);
			setSheets(res.sheets);
			sheetsSelectRef.current.classList.remove('hideListbox');
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

	const onFilePreview = async () => {
		if (actionType.Fields.includes('Path') && action.Path === '') {
			showError('Please enter a path');
			return;
		}
		if (actionType.Fields.includes('Sheet') && action.Sheet === '') {
			showError('Please enter a sheet');
			return;
		}
		const res = await records(action.Path!, action.Sheet, 20);
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
		setFilePreviewColumns(columns);
		const rows: GridRow[] = [];
		for (const row of res.records) {
			rows.push({ cells: row });
		}
		setFilePreviewRows(rows);
	};

	const onConfirmFile = async () => {
		if (actionType.Fields.includes('Path') && action.Path === '') {
			showError('Please enter a path');
			return;
		}
		if (actionType.Fields.includes('Sheet') && action.Sheet === '') {
			showError('Please enter a sheet');
			return;
		}
		fileConfirmButtonRef.current!.load();
		const res = await records(action.Path!, action.Sheet, 0, true);
		if (res == null) {
			fileConfirmButtonRef.current!.stop();
			return;
		}
		fileConfirmButtonRef.current!.confirm();
		setTimeout(() => {
			const actionTyp = { ...actionType };
			actionTyp.InputSchema = res.schema;
			setActionType(actionTyp);
			setTimeout(() => {
				const top = mappingSectionRef.current!.getBoundingClientRect().top;
				mappingSectionRef.current!.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			});
		}, CONFIRM_ANIMATION_DURATION);
	};

	const records = async (path: string, sheet: string | null | undefined, limit: number, isConfirmation?: boolean) => {
		let res: RecordsResponse;
		try {
			res = await api.workspaces.connections.records(
				connection.id,
				path,
				sheet === undefined ? null : sheet,
				limit,
			);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ReadFileFailed':
						showStatus({ variant: variants.DANGER, icon: icons.INVALID_INSERTED_VALUE, text: err.message });
						break;
					case 'NoStorage':
						showStatus(statuses.linkedStorageDoesNotExistAnymore);
						break;
					default:
						break;
				}
				return;
			}
			showError(err);
			return;
		}
		if (isConfirmation) {
			pathRef.current.lastConfirmation = path;
			if (sheet != null) {
				sheetRef.current.lastConfirmation = sheet;
			}
		}
		return res;
	};

	return (
		<Section
			title={`Path${actionType.Fields.includes('Sheet') ? ' and Sheet' : ''}`}
			description={`The path${actionType.Fields.includes('Sheet') ? ' and sheet' : ''} of the file`}
			padded
		>
			<div className='pathInputWrapper'>
				<SlInput
					className='pathInput'
					name='path'
					value={action.Path!}
					label={actionType.Fields.includes('Sheet') ? 'Path' : undefined}
					type='text'
					onSlInput={onUpdatePath}
					placeholder={`${actionType.Target.toLowerCase()}.${connection.connector.fileExtension}`}
				/>
				<div className={`completePathError${completePathError !== '' ? ' visible' : ''}`}>
					{completePathError}
				</div>
				<div className={`completePath${completePath !== '' ? ' visible' : ''}`}>{completePath}</div>
			</div>
			{actionType.Fields.includes('Sheet') && (
				<>
					<div className='sheetsSelectWrapper'>
						<SlSelect
							onSlFocus={onSheetsLoad}
							className='sheetsSelect'
							ref={sheetsSelectRef}
							name='sheet'
							value={action.Sheet == null ? undefined : action.Sheet}
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
							{sheets.map((sheet) => {
								const name = sheet.toLowerCase();
								return (
									<SlOption key={name} value={name}>
										{sheet}
									</SlOption>
								);
							})}
						</SlSelect>
						<SlButton
							onClick={onSheetsReload}
							disabled={action.Path == null || action.Path === '' || areSheetsLoading}
						>
							<SlIcon name='arrow-clockwise' />
						</SlButton>
					</div>
				</>
			)}
			{isImport && (
				<div className='fileButtons'>
					<SlButton variant='neutral' size='small' onClick={onFilePreview}>
						Preview
					</SlButton>
					<ConfirmationButton
						ref={fileConfirmButtonRef}
						variant='success'
						size='small'
						onClick={onConfirmFile}
						animationDuration={CONFIRM_ANIMATION_DURATION}
					>
						Confirm
					</ConfirmationButton>
				</div>
			)}
			<SlDrawer
				className='previewDrawer'
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
		</Section>
	);
};

export default ActionPath;

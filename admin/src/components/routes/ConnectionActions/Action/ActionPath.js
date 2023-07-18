import { useState, useContext, useEffect, useRef } from 'react';
import Section from '../../../shared/Section/Section';
import ConfirmationButton from '../../../shared/ConfirmationButton/ConfirmationButton';
import Grid from '../../../shared/Grid/Grid';
import { AppContext } from '../../../../context/providers/AppProvider';
import { ActionContext } from '../../../../context/ActionContext';
import { UnprocessableError, NotFoundError, BadRequestError } from '../../../../lib/api/errors';
import * as statuses from '../../../../constants/statuses';
import * as variants from '../../../../constants/variants';
import * as icons from '../../../../constants/icons';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import {
	SlButton,
	SlInput,
	SlIcon,
	SlSpinner,
	SlSelect,
	SlOption,
	SlDrawer,
} from '@shoelace-style/shoelace/dist/react/index.js';

const ActionPath = () => {
	const [sheets, setSheets] = useState([]);
	const [areSheetsLoading, setAreSheetsLoading] = useState(false);
	const [hasSheetsError, setHasSheetsError] = useState(false);
	const [completePath, setCompletePath] = useState('');
	const [completePathError, setCompletePathError] = useState('');
	const [filePreviewTable, setFilePreviewTable] = useState(null);
	const [isFilePreviewDrawerOpen, setIsFilePreviewDrawerOpen] = useState(false);

	const { showStatus, showError, api, setAreConnectionsStale } = useContext(AppContext);
	const { connection, action, setAction, actionType, setActionType, isImport, mappingSectionRef, setIsFileChanged } =
		useContext(ActionContext);

	const getCompletePathTimeoutID = useRef(null);
	const sheetsSelectRef = useRef(null);
	const fileConfirmButtonRef = useRef(null);

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
			lastConfirmation: action.Path,
			lastUpdate: action.Path,
		};
		sheetRef.current = {
			lastConfirmation: action.Sheet,
			lastUpdate: action.Sheet,
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
		getCompletePathTimeoutID.current = setTimeout(async () => {
			const [res, err] = await api.connections.completePath(connection.storage, path);
			if (err != null) {
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
		const [res, err] = await api.connections.sheets(connection.id, action.Path);
		if (err != null) {
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
			showError('You must first enter a path');
			return;
		}
		if (actionType.Fields.includes('Sheet') && action.Sheet === '') {
			showError('You must first enter a sheet');
			return;
		}
		const res = await records(action.Path, action.Sheet, 20);
		if (res == null) {
			return;
		}
		const columns = [];
		for (const prop of res.schema.properties) {
			let name;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		const rows = [];
		for (const row of res.records) {
			rows.push({ cells: row });
		}
		const table = { columns, rows };
		setFilePreviewTable(table);
	};

	const onConfirmFile = async () => {
		if (actionType.Fields.includes('Path') && action.Path === '') {
			showError('You must first enter a path');
			return;
		}
		if (actionType.Fields.includes('Sheet') && action.Sheet === '') {
			showError('You must first enter a sheet');
			return;
		}
		fileConfirmButtonRef.current.load();
		const res = await records(action.Path, action.Sheet, 0, true);
		if (res == null) {
			fileConfirmButtonRef.current.stop();
			return;
		}
		fileConfirmButtonRef.current.confirm();
		setTimeout(() => {
			const actionTyp = { ...actionType };
			actionTyp.InputSchema = res.schema;
			setActionType(actionTyp);
			setTimeout(() => {
				const top = mappingSectionRef.current.getBoundingClientRect().top;
				mappingSectionRef.current.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			});
		}, CONFIRM_ANIMATION_DURATION);
	};

	const records = async (path, sheet, limit, isConfirmation) => {
		const [res, err] = await api.connections.records(connection.id, path, sheet, limit);
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ReadFileFailed':
						showStatus([variants.DANGER, icons.INVALID_INSERTED_VALUE, err.message]);
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
			sheetRef.current.lastConfirmation = sheet;
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
					value={action.Path}
					label={actionType.Fields.includes('Sheet') ? 'Path' : null}
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
							value={action.Sheet}
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
				open={filePreviewTable != null}
				onSlAfterShow={() => setIsFilePreviewDrawerOpen(true)}
				onSlAfterHide={() => {
					setFilePreviewTable(null);
					setIsFilePreviewDrawerOpen(false);
				}}
				placement='bottom'
				style={{ '--size': '600px' }}
			>
				{isFilePreviewDrawerOpen ? (
					<Grid
						columns={filePreviewTable.columns}
						rows={filePreviewTable.rows}
						noRowsMessage={'Your file did not return data'}
					/>
				) : (
					<SlSpinner
						style={{
							fontSize: '3rem',
							'--track-width': '6px',
						}}
					></SlSpinner>
				)}
			</SlDrawer>
		</Section>
	);
};

export default ActionPath;

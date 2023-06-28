import { useState, useEffect, useContext, useRef } from 'react';
import './Action.css';
import Section from '../../../common/Section/Section';
import ConfirmationButton from '../../../common/ConfirmationButton/ConfirmationButton';
import statuses from '../../../../constants/statuses';
import * as variants from '../../../../constants/variants';
import * as icons from '../../../../constants/icons';
import Connection from '../../../../lib/connections/connection';
import EditorWrapper from '../../../common/EditorWrapper/EditorWrapper';
import Grid from '../../../common/Grid/Grid';
import ActionMapping from './ActionMapping';
import { ComboBoxList, ComboBoxInput } from '../../../common/ComboBox/ComboBox';
import { getDefaultMappings, getSchemaComboboxItems, getExpressionVariables } from './Action.helpers';
import { AppContext } from '../../../../providers/AppProvider';
import { ConnectionContext } from '../../../../providers/ConnectionProvider';
import { UnprocessableError, NotFoundError, BadRequestError } from '../../../../lib/api/errors';
import {
	SlButton,
	SlInput,
	SlIcon,
	SlSpinner,
	SlSelect,
	SlOption,
	SlIconButton,
	SlDrawer,
} from '@shoelace-style/shoelace/dist/react/index.js';

const queryMaxSize = 16777215;

const exportModeOptions = {
	CreateOnly: 'Create only',
	UpdateOnly: 'Update only',
	CreateOrUpdate: 'Create and update',
};

const operatorOptions = {
	1: 'is',
	2: 'is not',
};

const CONFIRM_ANIMATION_DURATION = 1200;

const Action = ({ actionType: actionTypeProp, action: actionProp, onClose }) => {
	const [action, setAction] = useState(null);
	const [actionType, setActionType] = useState(null);
	const [mode, setMode] = useState('');
	const [fields, setFields] = useState([]);
	const [inputSchema, setInputSchema] = useState(null);
	const [sheets, setSheets] = useState([]);
	const [areSheetsLoading, setAreSheetsLoading] = useState(false);
	const [hasSheetsError, setHasSheetsError] = useState(false);
	const [outputSchema, setOutputSchema] = useState(null);
	const [queryPreviewTable, setQueryPreviewTable] = useState(null);
	const [isQueryPreviewDrawerOpen, setIsQueryPreviewDrawerOpen] = useState(false);
	const [filePreviewTable, setFilePreviewTable] = useState(null);
	const [isFilePreviewDrawerOpen, setIsFilePreviewDrawerOpen] = useState(false);

	const [isNameEditable, setIsNameEditable] = useState(false);
	const [isSaveButtonLoading, setIsSaveButtonLoading] = useState(false);
	const [completePath, setCompletePath] = useState('');
	const [completePathError, setCompletePathError] = useState('');

	const { api, showError, showStatus, redirect, setAreConnectionsStale } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	const queryRef = useRef('');
	const pathRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
		lastSheetFetch: '',
	});
	const sheetRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});
	const sheetsSelectRef = useRef(null);
	const conditionListRef = useRef(null);
	const internalMatchingPropertyListRef = useRef(null);
	const externalMatchingPropertyListRef = useRef(null);
	const mappingSectionRef = useRef(null);
	const queryConfirmButtonRef = useRef(null);
	const fileConfirmButtonRef = useRef(null);
	const getCompletePathTimeoutID = useRef(null);

	const isImport = c.role === 'Source';
	const isEditing = actionProp != null;

	useEffect(() => {
		let actionType;
		const fetchData = async () => {
			const a = isEditing ? { ...actionProp } : null;

			// get the action type.
			if (a != null) {
				const [connection, err] = await api.connections.get(c.id);
				if (err != null) {
					onClose();
					showError(err);
					return;
				}
				const cn = Connection.new(connection);
				if (a.Target === 'Events') {
					actionType = cn.actionTypes.find((t) => t.EventType === a.EventType);
				} else {
					actionType = cn.actionTypes.find((t) => t.Target === a.Target);
				}
			} else {
				actionType = { ...actionTypeProp };
			}
			setActionType(actionType);

			// get the action schemas.
			const [schemas, err] = await api.connections.actionSchemas(c.id, actionType.Target, actionType.EventType);
			if (err != null) {
				onClose();
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'NoUsersSchema':
							showStatus(statuses.noUsersSchema);
							break;
						case 'NoGroupsSchema':
							showStatus(statuses.noGroupsSchema);
							break;
						case 'EventTypeNotExists':
							showStatus([variants.DANGER, icons.NOT_FOUND, err.message]);
							break;
						default:
							break;
					}
					return;
				}
				showError(err);
				return;
			}
			setInputSchema(schemas.In);
			setOutputSchema(schemas.Out);

			// check which fields are supported by the action.
			const fields = [];
			if (c.type === 'App' && c.role === 'Destination' && actionType.Target === 'Events') {
				fields.push('Filter');
			}
			if (
				(c.type === 'App' && schemas.In != null && schemas.Out != null) ||
				(c.type === 'Database' && c.role === 'Source') ||
				(c.type === 'File' && c.role === 'Source')
			) {
				fields.push('Mapping');
			}
			if (
				c.type === 'App' &&
				c.role === 'Destination' &&
				(actionType.Target === 'Users' || actionType.Target === 'Groups')
			) {
				fields.push('MatchingProperties');
				fields.push('ExportMode');
				fields.push('Filter');
			}
			if (c.type === 'Database' && c.role === 'Source') {
				fields.push('Query');
			}
			if (c.type === 'File') {
				if (c.role === 'Destination') {
					fields.push('Filter');
				}
				fields.push('Path');
				if (c.connector.hasSheets) {
					fields.push('Sheet');
				}
			}
			setFields(fields);

			// if the action is on a database or file source, the schema must be
			// fetched from the linked database / file.
			if (fields.includes('Query') && a != null) {
				const res = await query(a.Query, 0);
				if (res != null) {
					setInputSchema(res.Schema);
				}
			} else if (fields.includes('Path') && a != null && isImport) {
				const res = await records(a.Path, a.Sheet, 0);
				if (res != null) {
					setInputSchema(res.schema);
				}
			}

			// generate the action.
			let action;
			if (a != null) {
				if (a.Mapping != null) {
					const mapping = getDefaultMappings(schemas.Out);
					for (const k in mapping) {
						if (a.Mapping[k] != null) {
							mapping[k].value = a.Mapping[k];
							const { root, indentation } = mapping[k];
							for (const k in mapping) {
								if (mapping[k].root === root && mapping[k].indentation !== indentation) {
									mapping[k].disabled = true;
								}
							}
						}
					}
					a.Mapping = mapping;
				}
				action = { ...a };
			} else {
				action = {
					Name: actionType.Name,
					Enabled: false,
					Filter: null,
					Mapping: getDefaultMappings(schemas.Out),
					InSchema: null,
					OutSchema: null,
					PythonSource: null,
					Query: null,
					Path: fields.includes('Path') ? '' : null,
					Sheet: fields.includes('Sheet') ? '' : null,
				};
				if (fields.includes('ExportMode')) {
					action.ExportMode = Object.keys(exportModeOptions)[0];
				}
				if (fields.includes('MatchingProperties')) {
					action.MatchingProperties = { Internal: '', External: '' };
				}
			}
			queryRef.current = action.Query;
			pathRef.current = {
				...pathRef.current,
				lastConfirmation: action.Path,
				lastUpdate: action.Path,
			};
			sheetRef.current = {
				lastConfirmation: action.Sheet,
				lastUpdate: action.Sheet,
			};
			setAction(action);
		};
		fetchData();
	}, []);

	const onAddCondition = () => {
		const a = { ...action };
		if (a.Filter == null) {
			a.Filter = { Logical: 'all', Conditions: [] };
		}
		a.Filter.Conditions = [...a.Filter.Conditions, { Property: '', Operator: '', Value: '' }];
		setAction(a);
	};

	const onRemoveCondition = (e) => {
		const a = { ...action };
		const id = e.currentTarget.closest('.condition').dataset.id;
		a.Filter.Conditions.splice(id, 1);
		if (a.Filter.Conditions.length === 0) {
			a.Filter = null;
		}
		setAction(a);
	};

	const onUpdateConditionFragment = (e) => {
		const a = { ...action };
		const id = e.target.closest('.condition').dataset.id;
		const fragment = e.target.dataset.fragment;
		let value;
		if (fragment === 'Operator') {
			value = operatorOptions[e.target.value];
		} else {
			value = e.target.value;
		}
		a.Filter.Conditions[id][fragment] = value;
		setAction(a);
	};

	const onSelectConditionListItem = (input, value) => {
		const a = { ...action };
		const id = input.closest('.condition').dataset.id;
		a.Filter.Conditions[id]['Property'] = value;
		setAction(a);
	};

	const onSwitchFilterLogical = () => {
		const a = { ...action };
		const logical = a.Filter.Logical;
		if (logical === 'all') {
			a.Filter.Logical = 'any';
		} else {
			a.Filter.Logical = 'all';
		}
		setAction(a);
	};

	const onUpdateName = (e) => {
		const a = { ...action };
		a.Name = e.currentTarget.value;
		setAction(a);
	};

	const onUpdateQuery = async (value) => {
		const a = { ...action };
		a.Query = value;
		setAction(a);
	};

	const query = async (query, limit, isConfirmation) => {
		const a = { ...action };
		const trimmed = query != null ? query : a.Query.trim();
		if (trimmed.length > queryMaxSize) {
			showError('You query is too long');
			return;
		}
		if (!trimmed.includes('$limit')) {
			showError(`Your query does not contain the $limit variable`);
			return;
		}
		const [res, err] = await api.connections.query(c.id, trimmed, limit);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'QueryExecutionFailed') {
					let statusMessage;
					if (err.cause && err.cause !== '') {
						statusMessage = err.cause;
					} else {
						statusMessage = err.message;
					}
					showStatus([variants.DANGER, icons.CODE_ERROR, statusMessage]);
				}
				return;
			}
			showError(err);
			return;
		}
		if (Object.keys(res.Schema.properties).length === 0) {
			showError('Your query did not return any columns');
			return;
		}
		if (isConfirmation) {
			queryRef.current = trimmed;
		}
		return res;
	};

	const onQueryPreview = async () => {
		const res = await query(null, 20);
		if (res == null) {
			return;
		}
		const columns = [];
		for (const prop of res.Schema.properties) {
			let name;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		const rows = [];
		for (const row of res.Rows) {
			rows.push({ cells: row });
		}
		const table = { columns, rows };
		setQueryPreviewTable(table);
	};

	const onConfirmQuery = async () => {
		queryConfirmButtonRef.current.load();
		const res = await query(null, 0, true);
		if (res == null) {
			queryConfirmButtonRef.current.stop();
			return;
		}
		queryConfirmButtonRef.current.confirm();
		setTimeout(() => {
			setInputSchema(res.Schema);
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

	const onChangeExportMode = (e) => {
		const a = { ...action };
		a.ExportMode = e.currentTarget.value;
		setAction(a);
	};

	const onUpdateMatchingProperties = (e) => {
		const a = { ...action };
		a.MatchingProperties[e.target.dataset.type] = e.target.value;
		setAction(a);
	};

	const onSelectMatchingProperties = (input, value) => {
		const a = { ...action };
		a.MatchingProperties[input.dataset.type] = value;
		setAction(a);
	};

	const onUpdatePath = async (e) => {
		clearTimeout(getCompletePathTimeoutID.current);
		const a = { ...action };
		const path = e.currentTarget.value;
		pathRef.current.lastUpdate = path;
		a.Path = path;
		a.Sheet = '';
		setAction(a);
		setCompletePath('');
		setCompletePathError('');
		if (path === '' || c.storage === 0) {
			return;
		}
		getCompletePathTimeoutID.current = setTimeout(async () => {
			const [res, err] = await api.connections.completePath(c.storage, path);
			if (err != null) {
				if (err instanceof UnprocessableError && err.code === 'InvalidPath') {
					setCompletePathError(err.message);
					return;
				}
				if (err instanceof NotFoundError) {
					showStatus(statuses.linkedStorageDoesNotExistAnymore);
					const cn = { ...c };
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
		const [res, err] = await api.connections.sheets(c.id, action.Path);
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

	const records = async (path, sheet, limit, isConfirmation) => {
		const [res, err] = await api.connections.records(c.id, path, sheet, limit);
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

	const onFilePreview = async () => {
		if (fields.includes('Path') && action.Path === '') {
			showError('You must first enter a path');
			return;
		}
		if (fields.includes('Sheet') && action.Sheet === '') {
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
		if (fields.includes('Path') && action.Path === '') {
			showError('You must first enter a path');
			return;
		}
		if (fields.includes('Sheet') && action.Sheet === '') {
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
			setInputSchema(res.schema);
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

	const onSave = async () => {
		const a = { ...action };
		if (a.Mapping != null) {
			const mappingToSave = {};
			const inSchema = { name: 'Object', properties: [] };
			const outSchema = { name: 'Object', properties: [] };
			const flattenedInputSchema = getDefaultMappings(inputSchema);
			const flattenedOutputSchema = getDefaultMappings(outputSchema);
			for (const k in a.Mapping) {
				const v = a.Mapping[k];
				if (v.value === '') {
					continue;
				}
				const variables = getExpressionVariables(v.value);
				for (const variable of variables) {
					const property = flattenedInputSchema[variable];
					if (property == null) {
						showError(`${v.value} does not exist in the schema`);
						return;
					}
					const fullProperty = property.full;
					const isPropertyAlreadyInSchema = inSchema.properties.find((p) => p.name === fullProperty.name);
					if (!isPropertyAlreadyInSchema) {
						inSchema.properties.push(fullProperty);
					}
				}
				mappingToSave[k] = v.value;
				const fullKeyProperty = flattenedOutputSchema[k].full;
				const isKeyPropertyAlreadyInSchema = outSchema.properties.find((p) => p.name === fullKeyProperty.name);
				if (!isKeyPropertyAlreadyInSchema) {
					outSchema.properties.push(fullKeyProperty);
				}
			}
			a.InSchema = inSchema;
			a.OutSchema = outSchema;
			a.Mapping = mappingToSave;
		}
		if (a.PythonSource != null) {
			a.PythonSource = a.PythonSource.trim();
		}
		if (a.Query != null) {
			const trimmed = a.Query.trim();
			a.Query = trimmed;
		}
		let id, err;
		if (actionProp != null) {
			[, err] = await api.connections.setAction(c.id, a.ID, a);
		} else {
			[id, err] = await api.connections.addAction(c.id, {
				Target: actionType.Target,
				EventType: actionType.EventType,
				Action: a,
			});
		}
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'EventTypeNotExists':
					case 'PropertyNotExists':
						showStatus([variants.DANGER, icons.NOT_FOUND, err.message]);
						break;
					default:
						break;
				}
				return;
			}
			showError(err);
			return;
		}
		if (id) {
			sessionStorage.setItem('newAction', id);
		}
		setIsSaveButtonLoading(true);
		setTimeout(() => {
			setIsSaveButtonLoading(false);
			onClose();
		}, 200);
	};

	if (action === null || actionType === null) {
		return;
	}

	const conditions = [];
	if (action.Filter != null) {
		for (const [i, condition] of action.Filter.Conditions.entries()) {
			let conditionInput, operatorSelect, valueInput;
			conditionInput = (
				<ComboBoxInput
					comboBoxListRef={conditionListRef}
					onInput={onUpdateConditionFragment}
					value={condition.Property}
					className='property'
					size='small'
					data-fragment='Property'
				/>
			);
			operatorSelect = (
				<SlSelect
					data-fragment='Operator'
					size='small'
					className='operator'
					value={Object.keys(operatorOptions).find((key) => operatorOptions[key] === condition.Operator)}
					onSlChange={onUpdateConditionFragment}
				>
					{Object.keys(operatorOptions).map((k) => (
						<SlOption value={k}>{operatorOptions[k]}</SlOption>
					))}
				</SlSelect>
			);
			valueInput = (
				<SlInput
					data-fragment='Value'
					size='small'
					className='value'
					value={condition.Value}
					onSlInput={onUpdateConditionFragment}
				/>
			);
			conditions.push(
				<div className='condition' data-id={i}>
					{conditionInput}
					{operatorSelect}
					{valueInput}
					<SlButton className='removeCondition' size='small' variant='danger' onClick={onRemoveCondition}>
						Remove
					</SlButton>
				</div>
			);
		}
	}

	let isQueryChanged = false;
	if (queryRef.current != null) {
		if (queryRef.current.trim() !== action.Query.trim()) {
			isQueryChanged = true;
		}
	} else {
		if (action.Query != null && action.Query.trim() !== '') {
			isQueryChanged = true;
		}
	}

	const isFileChanged =
		pathRef.current.lastUpdate !== pathRef.current.lastConfirmation ||
		sheetRef.current.lastUpdate !== sheetRef.current.lastConfirmation;

	const mustComputeSchema = (c.type === 'Database' || c.type === 'File') && inputSchema == null && !isEditing;
	const hasQueryError = c.type === 'Database' && inputSchema == null && isEditing;
	const hasRecordsError = c.type === 'File' && inputSchema == null && isEditing;
	const isMappingSectionDisabled = hasQueryError || isQueryChanged || hasRecordsError || (isFileChanged && isImport);

	let disabledReason = '';
	if (hasQueryError) {
		disabledReason =
			'Mappings are disabled since the query returned an error. Fix the query before proceding to mappings.';
	} else if (hasRecordsError) {
		disabledReason =
			'Mappings are disabled since the file fetch returned an error. Fix the file informations before proceding to mappings.';
	} else if (c.type === 'Database') {
		disabledReason =
			'Mappings are disabled since you have modified the query. Confirm the query or undo the changes before proceding to mappings';
	} else {
		disabledReason =
			'Mappings are disabled since you have modified the file informations. Confirm the new informations or undo the changes before proceding to mappings';
	}

	return (
		<div className='action'>
			<div className='header'>
				<div className='title'>
					<div className='actionTitle'>
						{c.logo}
						<div className='actionName'>
							{isNameEditable ? (
								<span className='name'>
									<SlInput
										className='nameInput'
										value={action != null ? action.Name : actionType.Name}
										onSlInput={onUpdateName}
									></SlInput>
									<SlIconButton
										name='check-lg'
										label='Confirm'
										onClick={() => setIsNameEditable(false)}
									/>
								</span>
							) : (
								<span className='name'>
									{action != null ? action.Name : actionType.Name}
									<SlIconButton name='pencil' label='Edit' onClick={() => setIsNameEditable(true)} />
								</span>
							)}
						</div>
						{!isNameEditable && <div className='actionTypeDescription'>{actionType.Description}</div>}
					</div>
				</div>
				<div className='headerButtons'>
					<SlButton variant='default' onClick={onClose}>
						Cancel
					</SlButton>
					<SlButton
						className='saveAction'
						variant='primary'
						disabled={(actionType.Schema != null && mode === '') || isMappingSectionDisabled}
						onClick={onSave}
						loading={isSaveButtonLoading}
					>
						{actionProp != null ? 'Save' : 'Add'}
					</SlButton>
				</div>
			</div>
			<div className='body'>
				{fields.includes('Filter') && (
					<Section title='Filter' description='The filters that define the action' padded={true}>
						{conditions.length > 1 && (
							<SlSelect
								className='logical'
								size='small'
								value={action.Filter.Logical}
								onSlChange={onSwitchFilterLogical}
							>
								<SlOption value='all'>All</SlOption>
								<SlOption value='any'>Any</SlOption>
							</SlSelect>
						)}
						{conditions}
						<ComboBoxList
							ref={conditionListRef}
							items={getSchemaComboboxItems(inputSchema)}
							onSelect={onSelectConditionListItem}
						/>
						<SlButton className='addCondition' size='small' variant='neutral' onClick={onAddCondition}>
							Add new condition
						</SlButton>
					</Section>
				)}
				{fields.includes('Query') && (
					<Section title='Query' description='The query used to import the data'>
						<EditorWrapper
							defaultLanguage='sql'
							height={400}
							value={action.Query}
							onChange={onUpdateQuery}
						></EditorWrapper>
						<div className='queryButtons'>
							<SlButton variant='neutral' size='small' onClick={onQueryPreview}>
								Preview
							</SlButton>
							<ConfirmationButton
								ref={queryConfirmButtonRef}
								variant='success'
								size='small'
								onClick={onConfirmQuery}
								animationDuration={CONFIRM_ANIMATION_DURATION}
							>
								Confirm
							</ConfirmationButton>
						</div>
					</Section>
				)}
				{fields.includes('Path') && (
					<Section
						title={`Path${fields.includes('Sheet') ? ' and Sheet' : ''}`}
						description={`The path${fields.includes('Sheet') ? ' and sheet' : ''} of the file`}
						padded
					>
						<div className='pathInputWrapper'>
							<SlInput
								className='pathInput'
								name='path'
								value={action.Path}
								label={fields.includes('Sheet') ? 'Path' : null}
								type='text'
								onSlInput={onUpdatePath}
								placeholder={`${actionType.Target.toLowerCase()}.${c.connector.fileExtension}`}
							/>
							<div className={`completePathError${completePathError !== '' ? ' visible' : ''}`}>
								{completePathError}
							</div>
							<div className={`completePath${completePath !== '' ? ' visible' : ''}`}>{completePath}</div>
						</div>
						{fields.includes('Sheet') && (
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
											(pathRef.current.lastSheetFetch === pathRef.current.lastUpdate &&
												hasSheetsError)
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
					</Section>
				)}
				{fields.includes('ExportMode') && (
					<Section title='Export Mode' description='The mode used to export the data' padded={true}>
						<SlSelect size='medium' value={action.ExportMode} onSlChange={onChangeExportMode}>
							{Object.keys(exportModeOptions).map((k) => (
								<SlOption value={k}>{exportModeOptions[k]}</SlOption>
							))}
						</SlSelect>
					</Section>
				)}
				{fields.includes('MatchingProperties') && (
					<Section
						title={`Matching properties`}
						description='The properties used to identify and match the resources'
						padded={true}
					>
						<div className='matchingProperties'>
							<ComboBoxInput
								comboBoxListRef={internalMatchingPropertyListRef}
								onInput={onUpdateMatchingProperties}
								value={action.MatchingProperties.Internal}
								label='Golden record property'
								data-type='Internal'
								className='inputProperty'
							></ComboBoxInput>
							<ComboBoxList
								ref={internalMatchingPropertyListRef}
								items={getSchemaComboboxItems(inputSchema)}
								onSelect={onSelectMatchingProperties}
							/>
							<div className='equal'>=</div>
							<ComboBoxInput
								comboBoxListRef={externalMatchingPropertyListRef}
								onInput={onUpdateMatchingProperties}
								label={`${c.name}'s property`}
								value={action.MatchingProperties.External}
								data-type='External'
							></ComboBoxInput>
							<ComboBoxList
								ref={externalMatchingPropertyListRef}
								items={getSchemaComboboxItems(outputSchema)}
								onSelect={onSelectMatchingProperties}
							/>
						</div>
					</Section>
				)}
				{fields.includes('Mapping') && !mustComputeSchema && (
					<ActionMapping
						ref={mappingSectionRef}
						disabled={isMappingSectionDisabled}
						disabledReason={disabledReason}
						action={action}
						setAction={setAction}
						inputSchema={inputSchema}
						outputSchema={outputSchema}
						actionType={actionType}
						mode={mode}
						setMode={setMode}
					/>
				)}
				{fields.includes('Query') && (
					<SlDrawer
						className='previewDrawer'
						label='Query Preview'
						open={queryPreviewTable != null}
						onSlAfterShow={() => setIsQueryPreviewDrawerOpen(true)}
						onSlAfterHide={() => {
							setQueryPreviewTable(null);
							setIsQueryPreviewDrawerOpen(false);
						}}
						placement='bottom'
						style={{ '--size': '600px' }}
					>
						{isQueryPreviewDrawerOpen ? (
							<Grid
								columns={queryPreviewTable.columns}
								rows={queryPreviewTable.rows}
								noRowsMessage={'Your query did not return data'}
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
				)}
				{fields.includes('Path') && (
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
				)}
			</div>
		</div>
	);
};

export default Action;

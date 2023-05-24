import { useState, useEffect, useContext, useRef } from 'react';
import { createPortal } from 'react-dom';
import './Action.css';
import Section from '../Section/Section';
import AlertDialog from '../AlertDialog/AlertDialog';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import LittleLogo from '../LittleLogo/LittleLogo';
import EditPage from '../EditPage/EditPage';
import statuses from '../../constants/statuses';
import * as variants from '../../constants/variants';
import * as icons from '../../constants/icons';
import EditorWrapper from '../EditorWrapper/EditorWrapper';
import StyledGrid from '../StyledGrid/StyledGrid';
import { ComboBoxList, ComboBoxInput } from '../ComboBox/ComboBox';
import { UnprocessableError, NotFoundError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import {
	SlButton,
	SlInput,
	SlIcon,
	SlSelect,
	SlOption,
	SlDialog,
	SlIconButton,
	SlAlert,
	SlDrawer,
} from '@shoelace-style/shoelace/dist/react/index.js';

const queryMaxSize = 16777215;

const exportModeOptions = {
	CreateOnly: 'Create only',
	UpdateOnly: 'Update only',
	CreateOrUpdate: 'Create and update',
};

const rawTransformationFunction = `def transform($parameterName: dict) -> dict:
	return {}
;`;

const operatorOptions = {
	1: 'is',
	2: 'is not',
};

const Action = ({ actionType: actionTypeProp, action: actionProp, onClose }) => {
	let [action, setAction] = useState(null);
	let [actionType, setActionType] = useState(null);
	let [propertiesMode, setPropertiesMode] = useState('');
	let [fields, setFields] = useState([]);
	let [inputSchema, setInputSchema] = useState(null);
	let [isInputSchemaDialogOpen, setIsInputSchemaDialogOpen] = useState(false);
	let [outputSchema, setOutputSchema] = useState(null);
	let [isOutputSchemaDialogOpen, setIsOutputSchemaDialogOpen] = useState(false);
	let [queryPreviewTable, setQueryPreviewTable] = useState(null);
	let [filePreviewTable, setFilePreviewTable] = useState(null);
	let [isAlertOpen, setIsAlertOpen] = useState(false);
	let [isNameEditable, setIsNameEditable] = useState(false);

	let { API, showError, showStatus, redirect, connectors } = useContext(AppContext);
	let { connection: c } = useContext(ConnectionContext);

	let queryRef = useRef('');
	let pathRef = useRef('');
	let sheetRef = useRef('');
	let defaultTransformationFunction = useRef('');
	let propertiesListRef = useRef(null);
	let conditionListRef = useRef(null);
	let internalMatchingPropertyListRef = useRef(null);
	let externalMatchingPropertyListRef = useRef(null);

	let isImport = c.Role === 'Source';

	useEffect(() => {
		let actionType;
		const fetchData = async () => {
			let a = actionProp == null ? null : { ...actionProp };

			// get the action type.
			if (a != null) {
				let [actionTypes, err] = await API.connections.actionTypes(c.ID);
				if (err != null) {
					onClose();
					showError(err);
					return;
				}
				if (a.Target === 'Events') {
					actionType = actionTypes.find((t) => t.EventType === a.EventType);
				} else {
					actionType = actionTypes.find((t) => t.Target === a.Target);
				}
			} else {
				actionType = { ...actionTypeProp };
			}
			setActionType(actionType);

			// set the default transformation function.
			let parameterName = actionType.Target.toLowerCase();
			defaultTransformationFunction.current = rawTransformationFunction.replace('$parameterName', parameterName);

			// check which fields are supported by the action.
			let fields = [];
			if (c.Type === 'App' && c.Role === 'Destination' && actionType.Target === 'Events') {
				fields.push('Filter');
			}
			if (
				c.Type === 'App' ||
				(c.Type === 'Database' && c.Role === 'Source') ||
				(c.Type === 'File' && c.Role === 'Source')
			) {
				fields.push('Mapping');
			}
			if (
				c.Type === 'App' &&
				c.Role === 'Destination' &&
				(actionType.Target === 'Users' || actionType.Target === 'Groups')
			) {
				fields.push('MatchingProperties');
				fields.push('ExportMode');
				fields.push('Filter');
			}
			if (c.Type === 'Database' && c.Role === 'Source') {
				fields.push('Query');
			}
			if (c.Type === 'File') {
				if (c.Role === 'Destination') {
					fields.push('Filter');
				}
				fields.push('Path');
				let [connector, err] = await API.connectors.get(c.Connector);
				if (err != null) {
					showError(err);
					return;
				}
				if (connector.HasSheets) {
					fields.push('Sheet');
				}
			}
			setFields(fields);

			// get the action schemas.
			let [schemas, err] = await API.connections.actionSchemas(c.ID, actionType.Target, actionType.EventType);
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

			// if the action is on a database or file source, the schema must be
			// fetched from the linked database / file.
			if (fields.includes('Query') && a != null) {
				let res = await query(a.Query, 0);
				if (res != null) {
					setInputSchema(res.Schema);
				}
			} else if (fields.includes('Path') && a != null && isImport) {
				let res = await records(a.Path, a.Sheet, 0);
				if (res != null) {
					setInputSchema(res.schema);
				}
			}

			// generate the action.
			let action;
			if (a != null) {
				if (a.Mapping != null) {
					setPropertiesMode('mappings');
					let mapping = getDefaultMappings(schemas.Out);
					for (let k in mapping) {
						if (a.Mapping[k] != null) {
							mapping[k].value = a.Mapping[k];
							let { root, indentation } = mapping[k];
							for (let k in mapping) {
								if (mapping[k].root === root && mapping[k].indentation !== indentation) {
									mapping[k].disabled = true;
								}
							}
						}
					}
					a.Mapping = mapping;
				} else {
					setPropertiesMode('transformation');
				}
				action = { ...a };
			} else {
				action = {
					Name: actionType.Name,
					Enabled: false,
					Filter: null,
					Mapping: null,
					Transformation: null,
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
			pathRef.current = action.Path;
			sheetRef.current = action.Sheet;
			setAction(action);
		};
		fetchData();
	}, []);

	const onAddCondition = () => {
		let a = { ...action };
		if (a.Filter == null) {
			a.Filter = { Logical: 'all', Conditions: [] };
		}
		a.Filter.Conditions = [...a.Filter.Conditions, { Property: '', Operator: '', Value: '' }];
		setAction(a);
	};

	const onRemoveCondition = (e) => {
		let a = { ...action };
		let id = e.currentTarget.closest('.condition').dataset.id;
		a.Filter.Conditions.splice(id, 1);
		if (a.Filter.Conditions.length === 0) {
			a.Filter = null;
		}
		setAction(a);
	};

	const onUpdateConditionFragment = (e) => {
		let a = { ...action };
		let id = e.target.closest('.condition').dataset.id;
		let fragment = e.target.dataset.fragment;
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
		let a = { ...action };
		let id = input.closest('.condition').dataset.id;
		a.Filter.Conditions[id]['Property'] = value;
		setAction(a);
	};

	const onSwitchFilterLogical = () => {
		let a = { ...action };
		let logical = a.Filter.Logical;
		if (logical === 'all') {
			a.Filter.Logical = 'any';
		} else {
			a.Filter.Logical = 'all';
		}
		setAction(a);
	};

	const onSwitchPropertiesMode = () => {
		setIsAlertOpen(false);
		setTimeout(() => {
			let a = { ...action };
			if (propertiesMode === 'mappings') {
				a.Mapping = null;
				a.Transformation = getDefaultTransformation();
				setAction(a);
				setPropertiesMode('transformation');
			} else {
				a.Transformation = null;
				a.Mapping = getDefaultMappings(outputSchema);
				setAction(a);
				setPropertiesMode('mappings');
			}
		}, 150);
	};

	const onUpdateName = (e) => {
		let a = { ...action };
		a.Name = e.currentTarget.value;
		setAction(a);
	};

	const onChangeTransformationPythonSource = (value) => {
		let a = { ...action };
		a.Transformation.PythonSource = value;
		setAction(a);
	};

	const onRemoveTransformationProperty = (side, propertyName) => {
		let a = { ...action };
		let properties;
		if (side === 'input') {
			properties = a.Transformation.In.properties;
		} else {
			properties = a.Transformation.Out.properties;
		}
		let filtered = properties.filter((p) => p.name !== propertyName);
		if (side === 'input') {
			a.Transformation.In.properties = filtered;
		} else {
			a.Transformation.Out.properties = filtered;
		}
		setAction(a);
	};

	const onAddTransformationProperty = (side, property) => {
		let a = { ...action };
		if (side === 'input') {
			a.Transformation.In.properties.push(property);
		} else {
			a.Transformation.Out.properties.push(property);
		}
		setAction(a);
	};

	const onSetMappingsMode = () => {
		let a = { ...action };
		a.Mapping = getDefaultMappings(outputSchema);
		setAction(a);
		setPropertiesMode('mappings');
	};

	const onSetTransformationMode = () => {
		let a = { ...action };
		a.Transformation = getDefaultTransformation();
		setAction(a);
		setPropertiesMode('transformation');
	};

	const updateProperty = (name, value) => {
		const getAlternativeProperties = (name, mapping) => {
			let indentation = mapping[name].indentation;
			let parentProperties = [];
			for (let k in mapping) {
				if (mapping[k].indentation < indentation && name.startsWith(k)) {
					parentProperties.push(k);
				}
			}
			let childrenProperties = [];
			for (let k in mapping) {
				if (mapping[k].indentation > indentation && k.startsWith(name)) {
					childrenProperties.push(k);
				}
			}
			return [...parentProperties, ...childrenProperties];
		};

		let a = { ...action };
		if (a.Mapping[name].value === '' && value !== '') {
			let alternativeProperties = getAlternativeProperties(name, a.Mapping);
			// disable
			for (let k in a.Mapping) {
				if (alternativeProperties.includes(k)) {
					a.Mapping[k].disabled = true;
				}
			}
		} else if (value === '') {
			let hasFilledSiblings = false;
			let { root, indentation } = a.Mapping[name];
			for (let k in a.Mapping) {
				if (
					k !== name &&
					a.Mapping[k].root === root &&
					a.Mapping[k].indentation === indentation &&
					a.Mapping[k].value !== ''
				) {
					hasFilledSiblings = true;
				}
			}
			if (!hasFilledSiblings) {
				// enable
				let alternativeProperties = getAlternativeProperties(name, a.Mapping);
				for (let k in a.Mapping) {
					if (alternativeProperties.includes(k)) {
						a.Mapping[k].disabled = false;
					}
				}
			}
		}
		a.Mapping[name].value = value;
		setAction(a);
	};

	const onUpdateProperty = (e) => {
		let { name, value } = e.currentTarget || e.target;
		updateProperty(name, value);
	};

	const onSelectPropertiesListItem = (input, value) => {
		updateProperty(input.name, value);
	};

	const onUpdateQuery = async (value) => {
		let a = { ...action };
		a.Query = value;
		setAction(a);
	};

	const query = async (query, limit, isConfirmation) => {
		let a = { ...action };
		let trimmed = query != null ? query : a.Query.trim();
		if (trimmed.length > queryMaxSize) {
			showError('You query is too long');
			return;
		}
		if (!trimmed.includes('$limit')) {
			showError(`Your query does not contain the $limit variable`);
			return;
		}
		let [res, err] = await API.connections.query(c.ID, trimmed, limit);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'QueryExecutionFailed') {
					showStatus([variants.DANGER, icons.CODE_ERROR, err.cause]);
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
		let res = await query(null, 20);
		if (res == null) {
			return;
		}
		let columns = [];
		for (let prop of res.Schema.properties) {
			let name;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		let table = { columns, rows: res.Rows };
		setQueryPreviewTable(table);
	};

	const onConfirmQuery = async () => {
		let res = await query(null, 0, true);
		if (res == null) {
			return;
		}
		setInputSchema(res.Schema);
		let a = { ...action };
		if (a.Schema != null) {
			a.Schema = res.Schema;
		}
		if (propertiesMode === 'transformation') {
			a.Transformation.In.properties = [];
		}
		setAction(a);
		showStatus(statuses.schemaLoaded);
	};

	const onChangeExportMode = (e) => {
		let a = { ...action };
		a.ExportMode = e.currentTarget.value;
		setAction(a);
	};

	const onUpdateMatchingProperties = (e) => {
		let a = { ...action };
		a.MatchingProperties[e.target.dataset.type] = e.target.value;
		setAction(a);
	};

	const onSelectMatchingProperties = (input, value) => {
		let a = { ...action };
		a.MatchingProperties[input.dataset.type] = value;
		setAction(a);
	};

	const onUpdatePath = (e) => {
		let a = { ...action };
		let path = e.currentTarget.value;
		a.Path = path;
		setAction(a);
	};

	const onUpdateSheet = (e) => {
		let a = { ...action };
		let sheet = e.currentTarget.value;
		a.Sheet = sheet;
		setAction(a);
	};

	const records = async (path, sheet, limit, isConfirmation) => {
		let [res, err] = await API.connections.records(c.ID, path, sheet, limit);
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ReadFileFailed':
						showStatus([variants.DANGER, icons.INVALID_INSERTED_VALUE, err.message]);
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
			pathRef.current = path;
			sheetRef.current = sheet;
		}
		return res;
	};

	const onFilePreview = async () => {
		let res = await records(action.Path, action.Sheet, 20);
		if (res == null) {
			return;
		}
		let columns = [];
		for (let prop of res.schema.properties) {
			let name;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		let table = { columns, rows: res.records };
		setFilePreviewTable(table);
	};

	const onConfirmFile = async () => {
		let res = await records(action.Path, action.Sheet, 0, true);
		if (res == null) {
			return;
		}
		setInputSchema(res.schema);
		let a = { ...action };
		if (a.Schema != null) {
			a.Schema = res.schema;
		}
		if (propertiesMode === 'transformation') {
			a.Transformation.In.properties = [];
		}
		setAction(a);
		showStatus(statuses.schemaLoaded);
	};

	const onSave = async () => {
		let a = { ...action };
		if (a.Mapping != null) {
			let mappingToSave = {};
			for (let k in a.Mapping) {
				let v = a.Mapping[k];
				if (v.value !== '') {
					mappingToSave[k] = v.value;
				}
			}
			a.Mapping = mappingToSave;
		}
		if (a.Transformation != null) {
			let trimmed = a.Transformation.PythonSource.trim();
			a.Transformation.PythonSource = trimmed;
		}
		if (a.Query != null) {
			let trimmed = a.Query.trim();
			a.Query = trimmed;
		}
		let err;
		if (actionProp != null) {
			[, err] = await API.connections.setAction(c.ID, a.ID, a);
		} else {
			[, err] = await API.connections.addAction(c.ID, {
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
		showStatus(statuses.actionSaved);
		onClose();
	};

	const getDefaultMappings = (schema) => {
		if (schema == null) {
			return {};
		}
		const getSubProperties = (parentName, properties, indentation) => {
			let subProperties = {};
			indentation += 1;
			for (let subP of properties) {
				let key = `${parentName}.${subP.name}`;
				subProperties[key] = {
					value: '',
					indentation: indentation,
					root: key.substring(0, key.indexOf('.')),
					disabled: false,
					required: subP.required != null ? subP.required : false,
					type: subP.type.name,
					label: subP.label,
				};
				if (subP.type.name === 'Object') {
					let nestedSubProperties = getSubProperties(key, subP.type.properties, indentation);
					subProperties = { ...subProperties, ...nestedSubProperties };
				}
			}
			return subProperties;
		};
		let defaultMappings = {};
		for (let p of schema.properties) {
			let indentation = 0;
			defaultMappings[p.name] = {
				value: '',
				indentation: indentation,
				root: p.name,
				disabled: false,
				required: p.required != null ? p.required : false,
				type: p.type.name,
				label: p.label,
			};
			if (p.type.name === 'Object') {
				let subProperties = getSubProperties(p.name, p.type.properties, indentation);
				defaultMappings = { ...defaultMappings, ...subProperties };
			}
		}
		return defaultMappings;
	};

	const getDefaultTransformation = () => {
		return {
			In: {
				name: 'Object',
				properties: [],
			},
			Out: {
				name: 'Object',
				properties: [],
			},
			PythonSource: defaultTransformationFunction.current,
		};
	};

	const getSchemaComboboxItems = (side) => {
		let isInput = side === 'input';
		let schema = isInput ? inputSchema : outputSchema;
		if (schema == null) {
			return [];
		}
		let properties = getDefaultMappings(schema);
		let propertiesList = [];
		for (let k in properties) {
			let name;
			if (properties[k].label != null && properties[k].label !== '') {
				name = (
					<div className='propertiesItemName'>
						<div className='label'>{properties[k].label}</div>
						<div className='name'>{k}</div>
					</div>
				);
			} else {
				name = <div className='propertiesItemName'>{k}</div>;
			}
			let content = (
				<>
					{name}
					<div className='propertiesItemType'>{properties[k].type}</div>
				</>
			);
			propertiesList.push({
				content: content,
				searchableTerm: k,
			});
		}
		return propertiesList;
	};

	if (action === null || actionType === null) {
		return;
	}

	let conditions = [];
	if (action.Filter != null) {
		for (let [i, condition] of action.Filter.Conditions.entries()) {
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

	let isFileChanged = false;
	if (pathRef.current != null) {
		if (pathRef.current.trim() !== action.Path.trim()) {
			isFileChanged = true;
		}
	} else {
		if (action.Path != null && action.Path.trim() !== '') {
			isFileChanged = true;
		}
	}
	if (sheetRef.current != null) {
		if (sheetRef.current.trim() !== action.Sheet.trim()) {
			isFileChanged = true;
		}
	} else {
		if (action.Sheet != null && action.Sheet.trim() !== '') {
			isFileChanged = true;
		}
	}

	let hasQueryError = c.Type === 'Database' && inputSchema == null;
	let hasRecordsError = c.Type === 'File' && inputSchema == null;
	let isPropertiesSectionDisabled = hasQueryError || isQueryChanged || hasRecordsError || (isFileChanged && isImport);
	let propertiesSection = null;
	if (fields.includes('Mapping')) {
		let propertiesSectionActions = null;
		if (propertiesMode !== '') {
			let actionText;
			let actionIcon;
			if (propertiesMode === 'mappings') {
				actionText = 'Switch to transformation function';
				actionIcon = <SlIcon name='shuffle' slot='prefix'></SlIcon>;
			} else if (propertiesMode === 'transformation') {
				actionText = 'Switch to mappings';
				actionIcon = <SlIcon name='filetype-py' slot='prefix'></SlIcon>;
			}
			propertiesSectionActions = (
				<SlButton variant='neutral' size='small' onClick={() => setIsAlertOpen(true)}>
					{actionIcon}
					{actionText}
				</SlButton>
			);
		}

		let isSectionPadded = false;
		let propertiesSectionContent = null;
		if (propertiesMode === '') {
			if (fields.includes('Query') && inputSchema == null) {
				propertiesSectionContent = (
					<SlAlert variant='warning' open>
						<SlIcon slot='icon' name='exclamation-triangle' />
						To enable mappings, you should first load a database schema in the "Query" section
					</SlAlert>
				);
			} else if (fields.includes('Path') && inputSchema == null) {
				propertiesSectionContent = (
					<SlAlert variant='warning' open>
						<SlIcon slot='icon' name='exclamation-triangle' />
						To enable mappings, you should first load a file schema by inserting the file path
						{fields.includes('Sheet') && ' and sheet'}
					</SlAlert>
				);
			} else {
				isSectionPadded = true;
				propertiesSectionContent = (
					<div className='propertiesButtons'>
						<SlButton size='small' variant='primary' onClick={onSetMappingsMode}>
							<SlIcon name='shuffle' slot='prefix'></SlIcon>
							Map the properties
						</SlButton>
						<span>or</span>
						<SlButton size='small' variant='primary' onClick={onSetTransformationMode}>
							<SlIcon name='filetype-py' slot='prefix'></SlIcon>
							Write a transformation function
						</SlButton>
					</div>
				);
			}
		} else if (propertiesMode === 'mappings') {
			let mappings = [];
			let defaultMappings = getDefaultMappings(inputSchema);
			for (let k of Object.keys(action.Mapping)) {
				let error;
				let value = action.Mapping[k].value;
				if (!isPropertiesSectionDisabled && value !== '') {
					let doesValueExist = defaultMappings[value] != null;
					if (!doesValueExist) {
						error = `"${value}" does not exist in ${c.Type.toLowerCase()}'s schema`;
					}
				}
				mappings.push(
					<div
						className='mapping'
						data-key={k}
						style={{
							'--mapping-indentation': `${action.Mapping[k].indentation * 30}px`,
						}}
					>
						<ComboBoxInput
							comboBoxListRef={propertiesListRef}
							onInput={onUpdateProperty}
							value={value}
							name={k}
							disabled={isPropertiesSectionDisabled || action.Mapping[k].disabled}
							className='inputProperty'
							size='small'
							error={error}
						>
							{action.Mapping[k].required && <SlIcon name='asterisk' slot='prefix'></SlIcon>}
						</ComboBoxInput>
						<div className='arrow'>
							<SlIcon name='arrow-right' />
						</div>
						<SlInput
							readonly
							disabled
							size='small'
							value={k}
							type='text'
							name={k}
							onSlInput={null}
							className={`outputProperty${action.Mapping[k].indentation > 0 ? ' indented' : ''}`}
						/>
					</div>
				);
			}
			propertiesSectionContent = (
				<div className='mappings'>
					{isPropertiesSectionDisabled && (
						<SlAlert variant='danger' className='mappingsDisabledAlert' open>
							<SlIcon slot='icon' name='exclamation-circle' />
							{hasQueryError
								? 'Mappings are disabled since the query returned an error. Fix the query before proceding to mappings.'
								: hasRecordsError
								? 'Mappings are disabled since the file fetch returned an error. Fix the file informations before proceding to mappings.'
								: c.Type === 'Database'
								? 'Mappings are disabled since you have modified the query. Confirm the query or undo the changes before proceding to mappings'
								: 'Mappings are disabled since you have modified the file informations. Confirm the new informations or undo the changes before proceding to mappings'}
						</SlAlert>
					)}
					{mappings}
					<ComboBoxList
						ref={propertiesListRef}
						items={getSchemaComboboxItems('input')}
						onSelect={onSelectPropertiesListItem}
					/>
				</div>
			);
		} else if (propertiesMode === 'transformation') {
			propertiesSectionContent = (
				<div className='transformation'>
					<div className='inputProperties'>
						{action.Transformation.In.properties.map((p) => {
							return (
								<div className='property'>
									<div className='name'>{p.name}</div>
									<div className='type'>{p.type.name}</div>
									<SlButton
										className='removeProperty'
										size='small'
										variant='danger'
										outline
										onClick={() => onRemoveTransformationProperty('input', p.name)}
									>
										<SlIcon name='trash'></SlIcon>
									</SlButton>
								</div>
							);
						})}
						<SlButton
							className='addProperty'
							size='small'
							variant='default'
							onClick={() => setIsInputSchemaDialogOpen(true)}
						>
							Add new property...
						</SlButton>
					</div>
					<EditorWrapper
						defaultLanguage='python'
						height={400}
						value={action.Transformation.PythonSource}
						onChange={(value) => onChangeTransformationPythonSource(value)}
					/>
					<div className='outputProperties'>
						{action.Transformation.Out.properties.map((p) => {
							return (
								<div className='property'>
									<div className='name'>{p.name}</div>
									<div className='type'>{p.type.name}</div>
									<SlButton
										className='removeProperty'
										size='small'
										variant='danger'
										outline
										onClick={() => onRemoveTransformationProperty('output', p.name)}
									>
										<SlIcon name='trash'></SlIcon>
									</SlButton>
								</div>
							);
						})}
						<SlButton
							className='addProperty'
							size='small'
							variant='default'
							onClick={() => setIsOutputSchemaDialogOpen(true)}
						>
							Add new property...
						</SlButton>
					</div>
				</div>
			);
		}

		propertiesSection = (
			<Section
				title='Properties'
				description='The relation between the event properties and the action type properties'
				actions={propertiesSectionActions}
				padded={isSectionPadded}
			>
				{propertiesSectionContent}
			</Section>
		);
	}

	let connector = connectors.find((connector) => connector.ID === c.Connector);
	let logo;
	if (connector.Icon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connector.Icon} />;
	}

	return (
		<EditPage
			title={
				<div className='actionTitle'>
					{logo}
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
			}
			onCancel={onClose}
			actions={
				<SlButton
					className='saveAction'
					variant='primary'
					disabled={(actionType.Schema != null && propertiesMode === '') || isPropertiesSectionDisabled}
					onClick={onSave}
				>
					{actionProp != null ? 'Save' : 'Add'}
				</SlButton>
			}
		>
			<div className='action'>
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
							items={getSchemaComboboxItems('input')}
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
							<SlButton variant='success' size='small' onClick={onConfirmQuery}>
								Confirm
							</SlButton>
						</div>
					</Section>
				)}
				{fields.includes('Path') && (
					<Section
						title={`Path${fields.includes('Sheet') ? ' and Sheet' : ''}`}
						description={`The path${fields.includes('Sheet') ? ' and sheet' : ''} of the file`}
						padded
					>
						<SlInput
							className='pathInput'
							name='path'
							value={action.Path}
							label={fields.includes('Sheet') ? 'Path' : null}
							type='text'
							onSlInput={onUpdatePath}
							placeholder={`${actionType.Target.toLowerCase()}.${connector.FileExtension}`}
						/>
						{fields.includes('Sheet') && (
							<SlInput
								className='sheetInput'
								name='sheet'
								value={action.Sheet}
								label='Sheet'
								type='text'
								onSlInput={onUpdateSheet}
							/>
						)}
						{isImport && (
							<div className='fileButtons'>
								<SlButton variant='neutral' size='small' onClick={onFilePreview}>
									Preview
								</SlButton>
								<SlButton variant='success' size='small' onClick={onConfirmFile}>
									Confirm
								</SlButton>
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
								items={getSchemaComboboxItems('input')}
								onSelect={onSelectMatchingProperties}
							/>
							<div className='equal'>=</div>
							<ComboBoxInput
								comboBoxListRef={externalMatchingPropertyListRef}
								onInput={onUpdateMatchingProperties}
								label={`${c.Name}'s property`}
								value={action.MatchingProperties.External}
								data-type='External'
							></ComboBoxInput>
							<ComboBoxList
								ref={externalMatchingPropertyListRef}
								items={getSchemaComboboxItems('output')}
								onSelect={onSelectMatchingProperties}
							/>
						</div>
					</Section>
				)}
				{fields.includes('Query') && (
					<SlDrawer
						className='previewDrawer'
						label='Query Preview'
						open={queryPreviewTable != null}
						onSlAfterHide={() => setQueryPreviewTable(null)}
						placement='bottom'
						style={{ '--size': '600px' }}
					>
						{queryPreviewTable != null && (
							<StyledGrid
								columns={queryPreviewTable.columns}
								rows={queryPreviewTable.rows}
								noRowsMessage={'Your query did not return data'}
							/>
						)}
					</SlDrawer>
				)}
				{fields.includes('Path') && (
					<SlDrawer
						className='previewDrawer'
						label='File Preview'
						open={filePreviewTable != null}
						onSlAfterHide={() => setFilePreviewTable(null)}
						placement='bottom'
						style={{ '--size': '600px' }}
					>
						{filePreviewTable != null && (
							<StyledGrid
								columns={filePreviewTable.columns}
								rows={filePreviewTable.rows}
								noRowsMessage={'Your file did not return data'}
							/>
						)}
					</SlDrawer>
				)}
				{propertiesSection}
				{createPortal(
					<AlertDialog
						variant='danger'
						isOpen={isAlertOpen}
						onClose={() => setIsAlertOpen(false)}
						title={'You will lose your work'}
						actions={
							<>
								<SlButton variant='neutral' onClick={() => setIsAlertOpen(false)}>
									Cancel
								</SlButton>
								<SlButton variant='danger' onClick={onSwitchPropertiesMode}>
									Continue
								</SlButton>
							</>
						}
					>
						<div style={{ textAlign: 'center' }}>
							{propertiesMode === 'mappings' ? (
								<p>
									If you switch to the transformation function you will <b>PERMANENTLY</b> lose the
									mappings you have currently created
								</p>
							) : (
								<p>
									If you switch to the mappings you will <b>PERMANENTLY</b> lose the transformation
									code you have currently written
								</p>
							)}
						</div>
					</AlertDialog>,
					document.body
				)}
				{action.Transformation != null &&
					createPortal(
						<SlDialog
							className='inputSchemaDialog'
							label='Input properties'
							open={isInputSchemaDialogOpen}
							onSlRequestClose={() => setIsInputSchemaDialogOpen(false)}
							style={{ '--width': '700px' }}
						>
							{inputSchema.properties.map((p) => {
								let isUsed =
									action.Transformation.In.properties.findIndex((prop) => prop.name === p.name) !==
									-1;
								return (
									<div
										className={`property${isUsed ? ' used' : ''}${
											p.label != null && p.label !== '' ? ' hasLabel' : ''
										}`}
									>
										<div>
											{p.label != null && p.label !== '' && (
												<div className='label'>{p.label}</div>
											)}
											<div className='name'>{p.name}</div>
											<div className='type'>{p.type.name}</div>
										</div>
										{!isUsed && (
											<SlIconButton
												name='plus-circle'
												label='Add property'
												onClick={() => onAddTransformationProperty('input', p)}
											/>
										)}
									</div>
								);
							})}
						</SlDialog>,
						document.body
					)}
				{action.Transformation != null &&
					createPortal(
						<SlDialog
							className='outputSchemaDialog'
							label='Output properties'
							open={isOutputSchemaDialogOpen}
							onSlRequestClose={() => setIsOutputSchemaDialogOpen(false)}
							style={{ '--width': '700px' }}
						>
							{outputSchema.properties.map((p) => {
								let isUsed =
									action.Transformation.Out.properties.findIndex((prop) => prop.name === p.name) !==
									-1;
								return (
									<div
										className={`property${isUsed ? ' used' : ''}${
											p.label != null && p.label !== '' ? ' hasLabel' : ''
										}`}
									>
										<div>
											{p.label != null && p.label !== '' && (
												<div className='label'>{p.label}</div>
											)}
											<div className='name'>{p.name}</div>
											<div className='type'>{p.type.name}</div>
										</div>
										{!isUsed && (
											<SlIconButton
												name='plus-circle'
												label='Add property'
												onClick={() => onAddTransformationProperty('output', p)}
											/>
										)}
									</div>
								);
							})}
						</SlDialog>,
						document.body
					)}
			</div>
		</EditPage>
	);
};

export default Action;

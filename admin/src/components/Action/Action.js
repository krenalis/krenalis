import { useState, useEffect, useContext, useRef } from 'react';
import { createPortal } from 'react-dom';
import './Action.css';
import Section from '../Section/Section';
import AlertDialog from '../AlertDialog/AlertDialog';
import LittleLogo from '../LittleLogo/LittleLogo';
import EditPage from '../EditPage/EditPage';
import statuses from '../../constants/statuses';
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
	SlMenuItem,
	SlDialog,
	SlIconButton,
	SlAlert,
	SlDrawer,
} from '@shoelace-style/shoelace/dist/react/index.js';

const queryMaxSize = 16777215;

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
	let [inputSchema, setInputSchema] = useState(null);
	let [isInputSchemaDialogOpen, setIsInputSchemaDialogOpen] = useState(false);
	let [outputSchema, setOutputSchema] = useState(null);
	let [isOutputSchemaDialogOpen, setIsOutputSchemaDialogOpen] = useState(false);
	let [supports, setSupports] = useState([]);
	let [previewTable, setPreviewTable] = useState(null);
	let [isAlertOpen, setIsAlertOpen] = useState(false);
	let [isNameEditable, setIsNameEditable] = useState(false);
	let [isPropertiesListOpen, setIsPropertiesListOpen] = useState(false);
	let [isConditionListOpen, setIsConditionListOpen] = useState(false);
	let [focusedProperty, setFocusedProperty] = useState(null);
	let [focusedCondition, setFocusedCondition] = useState(null);
	let [propertySearchTerm, setPropertySearchTerm] = useState('');
	let [conditionSearchTerm, setConditionSearchTerm] = useState('');

	let { API, showError, showStatus, redirect } = useContext(AppContext);
	let { connection: c } = useContext(ConnectionContext);

	let initialQuery = useRef('');
	let defaultTransformationFunction = useRef('');
	let propertiesListRef = useRef(null);
	let conditionListRef = useRef(null);

	useEffect(() => {
		let actionType;
		const fetchData = async () => {
			let a = actionProp == null ? null : { ...actionProp };
			// get the action type
			if (a != null) {
				let [actionTypes, err] = await API.connections.actionTypes(c.ID);
				if (err != null) {
					onClose();
					showError(err);
					return;
				}
				for (let t of actionTypes) {
					if (a.Target === 'Users' || a.Target === 'Groups') {
						if (a.Target === t.Target) actionType = t;
						continue;
					}
					if (a.EventType === t.EventType) actionType = t;
				}
			} else {
				actionType = { ...actionTypeProp };
			}
			let actionTypeInfos, err;
			if (actionType.Target === 'Users') {
				[actionTypeInfos, err] = await API.connections.usersAction(c.ID);
			} else if (actionType.Target === 'Groups') {
				[actionTypeInfos, err] = await API.connections.groupsAction(c.ID);
			} else {
				if (actionType.EventType == null) {
					[actionTypeInfos, err] = await API.connections.eventsAction(c.ID);
				} else {
					[actionTypeInfos, err] = await API.connections.eventAction(c.ID, actionType.EventType);
				}
			}
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
							showStatus(statuses.genericNotFound(err.message));
							break;
						default:
							break;
					}
					return;
				}
				showError(err);
				return;
			}
			setActionType(actionType);
			setInputSchema(actionTypeInfos.InputSchema);
			setOutputSchema(actionTypeInfos.OutputSchema);
			setSupports(actionTypeInfos.Supports);

			if (actionTypeInfos.Supports.includes('Query') && a != null) {
				let res = await query(a.Query, 0);
				if (res == null) {
					a.Query = null;
				} else {
					setInputSchema(res.Schema);
				}
			}

			// get the action
			let action;
			if (a != null) {
				if (a.Mapping != null) {
					setPropertiesMode('mappings');
					let mapping = getDefaultMappings(actionTypeInfos.OutputSchema);
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
					Enabled: true,
					Filter: null,
					Mapping: null,
					Transformation: null,
					Query: null,
				};
			}
			initialQuery.current = action.Query;
			setAction(action);

			// set the default transformation function
			let parameterName;
			if (actionType.Target === 'Users') {
				parameterName = 'user';
			} else if (actionType.Target === 'Groups') {
				parameterName = 'group';
			} else {
				parameterName = 'event';
			}
			defaultTransformationFunction.current = rawTransformationFunction.replace('$parameterName', parameterName);
		};
		fetchData();
	}, []);

	useEffect(() => {
		if (focusedProperty == null) {
			return;
		}
		setPropertySearchTerm(focusedProperty.value);
	}, [focusedProperty]);

	useEffect(() => {
		if (focusedCondition == null) {
			return;
		}
		setConditionSearchTerm(focusedCondition.value);
	}, [focusedCondition]);

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
		setAction(a);
	};

	const onUpdateConditionFragment = (e) => {
		let a = { ...action };
		let id = e.currentTarget.closest('.condition').dataset.id;
		let fragment = e.currentTarget.dataset.fragment;
		let value;
		if (fragment === 'Operator') {
			value = operatorOptions[e.currentTarget.value];
		} else {
			value = e.currentTarget.value;
		}
		a.Filter.Conditions[id][fragment] = value;
		if (fragment === 'Property') {
			setConditionSearchTerm(value);
		}
		setAction(a);
	};

	const onSelectConditionListItem = (value) => {
		let a = { ...action };
		let id = focusedCondition.closest('.condition').dataset.id;
		a.Filter.Conditions[id]['Property'] = value;
		setAction(a);
		setConditionSearchTerm(value);
		setIsConditionListOpen(false);
		focusedCondition.focus();
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

	const onPropertyChange = (e) => {
		let { name, value } = e.currentTarget;
		updateProperty(name, value);
		setPropertySearchTerm(e.currentTarget.value);
	};

	const onSelectPropertiesListItem = (value) => {
		updateProperty(focusedProperty.name, value);
		setPropertySearchTerm(value);
		setIsPropertiesListOpen(false);
		focusedProperty.focus();
	};

	const onUpdateQuery = async (value) => {
		let a = { ...action };
		a.Query = value;
		setAction(a);
	};

	const query = async (query, limit) => {
		let a = { ...action };
		let trimmed = query != null ? query : a.Query.trim();
		if (trimmed.length > queryMaxSize) {
			showError('You query is too long');
			return;
		}
		if (!trimmed.includes(':limit')) {
			showError(`Your query does not contain the ':limit' placeholder`);
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
					showStatus(statuses.queryExecutionFailed);
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
		return res;
	};

	const onQueryPreview = async () => {
		let res = await query(null, 20);
		if (res == null) {
			return;
		}
		let columns = [];
		for (let k in res.Schema.properties) {
			columns.push({ Name: res.Schema.properties[k].name });
		}
		let table = { columns, rows: res.Rows };
		setPreviewTable(table);
	};

	const onConfirmSchema = async () => {
		let res = await query(null, 0);
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
						showStatus(statuses.genericNotFound(err.message));
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

	const getPropertiesList = (onSelect) => {
		if (inputSchema == null) {
			return;
		}
		let properties = getDefaultMappings(inputSchema);
		let propertiesList = [];
		for (let k in properties) {
			propertiesList.push({
				content: (
					<SlMenuItem className='propertiesItem' onClick={() => onSelect(k)}>
						<div className='propertiesItemName'>{k}</div>
						<div className='propertiesItemType'>{properties[k].type}</div>
					</SlMenuItem>
				),
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
					openComboBoxList={() => setIsConditionListOpen(true)}
					closeComboBoxList={() => setIsConditionListOpen(false)}
					setFocused={setFocusedCondition}
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

	let propertiesSection = null;
	if (supports.includes('Mapping')) {
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
			if (c.Type === 'Database' && inputSchema == null) {
				propertiesSectionContent = (
					<SlAlert variant='warning' open>
						<SlIcon slot='icon' name='exclamation-triangle' />
						To enable mappings, you should first load a database schema in the "Query" section
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
			propertiesSectionContent = (
				<div className='mappings'>
					{Object.keys(action.Mapping).map((k) => {
						return (
							<div
								className='mapping'
								data-key={k}
								style={{
									'--mapping-indentation': `${action.Mapping[k].indentation * 30}px`,
								}}
							>
								<ComboBoxInput
									comboBoxListRef={propertiesListRef}
									onInput={onPropertyChange}
									openComboBoxList={() => setIsPropertiesListOpen(true)}
									closeComboBoxList={() => setIsPropertiesListOpen(false)}
									setFocused={setFocusedProperty}
									value={action.Mapping[k].value}
									name={k}
									disabled={action.Mapping[k].disabled}
									className='inputProperty'
									size='small'
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
					})}
					<ComboBoxList
						ref={propertiesListRef}
						isOpen={isPropertiesListOpen}
						searchTerm={propertySearchTerm}
						comboBoxItems={getPropertiesList(onSelectPropertiesListItem)}
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

	let isConfirmSchemaButtonDisabled = false;
	if (initialQuery.current != null) {
		if (action.Query.trim() === initialQuery.current.trim()) {
			isConfirmSchemaButtonDisabled = true;
		}
	} else {
		if (action.Query == null || action.Query.trim() === '') {
			isConfirmSchemaButtonDisabled = true;
		}
	}

	return (
		<EditPage
			title={
				<div className='actionTitle'>
					<LittleLogo url={c.LogoURL} alternativeText={`${c.Name}'s logo`} />
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
					disabled={actionType.Schema != null && propertiesMode === ''}
					onClick={onSave}
				>
					{actionProp != null ? 'Save' : 'Add'}
				</SlButton>
			}
		>
			<div className='action'>
				{supports.includes('Filter') && (
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
							isOpen={isConditionListOpen}
							searchTerm={conditionSearchTerm}
							comboBoxItems={getPropertiesList(onSelectConditionListItem)}
						/>
						<SlButton className='addCondition' size='small' variant='neutral' onClick={onAddCondition}>
							Add new condition
						</SlButton>
					</Section>
				)}
				{supports.includes('Query') && (
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
							<SlButton
								variant='success'
								size='small'
								onClick={onConfirmSchema}
								disabled={isConfirmSchemaButtonDisabled}
							>
								Confirm
							</SlButton>
						</div>
					</Section>
				)}
				{propertiesSection}
				{supports.includes('Query') && (
					<SlDrawer
						className='previewDrawer'
						label='Preview'
						open={previewTable != null}
						onSlAfterHide={() => setPreviewTable(null)}
						placement='bottom'
						style={{ '--size': '600px' }}
					>
						{previewTable != null && (
							<StyledGrid
								columns={previewTable.columns}
								rows={previewTable.rows}
								noRowsMessage={'Your query did not return data'}
							/>
						)}
					</SlDrawer>
				)}
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
									<div className={`property${isUsed ? ' used' : ''}`}>
										<div>
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
									<div className={`property${isUsed ? ' used' : ''}`}>
										<div>
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

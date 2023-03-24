import { useState, useEffect, useContext } from 'react';
import { createPortal } from 'react-dom';
import './Action.css';
import Section from '../Section/Section';
import AlertDialog from '../AlertDialog/AlertDialog';
import statuses from '../../constants/statuses';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import {
	SlButton,
	SlInput,
	SlIcon,
	SlSelect,
	SlMenuItem,
	SlDialog,
	SlIconButton,
} from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const defaultTransformationFunction = `def transform(event: dict) -> dict:
	return event
`;

const Action = ({ actionTypeProp, actionProp, onClose }) => {
	let [action, setAction] = useState(null);
	let [actionType, setActionType] = useState(null);
	let [propertiesMode, setPropertiesMode] = useState('');
	let [eventsSchema, setEventsSchema] = useState([]);
	let [isEventSchemaDialogOpen, setIsEventSchemaDialogOpen] = useState(false);
	let [actionTypeSchema, setActionTypeSchema] = useState([]);
	let [isActionTypeSchemaDialogOpen, setIsActionTypeSchemaDialogOpen] = useState(false);
	let [isAlertOpen, setIsAlertOpen] = useState(false);

	let { API, showError, showStatus } = useContext(AppContext);
	let { connection: c } = useContext(ConnectionContext);

	useEffect(() => {
		const fetchData = async () => {
			let [eventsSchema, err] = await API.eventsSchema();
			if (err != null) {
				onClose();
				showError(err);
				return;
			}
			let action, actionType;
			if (actionProp != null) {
				action = { ...actionProp };
				if (action.Mapping != null) setPropertiesMode('mappings');
				else setPropertiesMode('transformation');
				let [actionTypes, err] = await API.connections.actionTypes(c.ID);
				if (err != null) {
					onClose();
					showError(err);
					return;
				}
				actionType = actionTypes.find((a) => a.ID === action.ActionType);
				if (action.Mapping != null) {
					let mapping = getDefaultMappings(actionType);
					for (let k in mapping) {
						if (action.Mapping[k] != null) {
							mapping[k].value = action.Mapping[k];
							let { root, indentation } = mapping[k];
							for (let k in mapping) {
								if (mapping[k].root === root && mapping[k].indentation !== indentation) {
									mapping[k].disabled = true;
								}
							}
						}
					}
					action.Mapping = mapping;
				}
			} else {
				actionType = actionTypeProp;
				let endpoint = 0;
				if (actionType.Endpoints != null) {
					endpoint = Number(Object.keys(actionType.Endpoints)[0]);
				}
				action = {
					ID: 0,
					Connection: c.ID,
					ActionType: actionType.ID,
					Name: actionType.Name,
					Endpoint: endpoint,
					Filter: { Logical: 'all', Conditions: [] },
					Enabled: true,
					Mapping: null,
					Transformation: null,
				};
			}
			setAction(action);
			setActionType(actionType);
			setEventsSchema(eventsSchema);
			setActionTypeSchema({ ...actionType.Schema });
		};
		fetchData();
	}, []);

	const onAddCondition = () => {
		let a = { ...action };
		a.Filter.Conditions = [...action.Filter.Conditions, { Property: '', Operator: '', Value: '' }];
		setAction(a);
	};

	const onRemoveCondition = (e) => {
		let a = { ...action };
		let id = e.currentTarget.parentElement.dataset.id;
		a.Filter.Conditions.splice(id, 1);
		setAction(a);
	};

	const onUpdateConditionFragment = (e) => {
		let a = { ...action };
		let id = e.currentTarget.parentElement.dataset.id;
		let fragment = e.currentTarget.dataset.fragment;
		a.Filter.Conditions[id][fragment] = e.currentTarget.value;
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
				a.Mapping = getDefaultMappings(actionType);
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

	const onChangeEndpoint = (e) => {
		let a = { ...action };
		a.Endpoint = Number(e.currentTarget.value);
		setAction(a);
	};

	const onSetMappingsMode = () => {
		let a = { ...action };
		a.Mapping = getDefaultMappings(actionType);
		setAction(a);
		setPropertiesMode('mappings');
	};

	const onSetTransformationMode = () => {
		let a = { ...action };
		a.Transformation = getDefaultTransformation();
		setAction(a);
		setPropertiesMode('transformation');
	};

	const onMappingUpdate = (e) => {
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
		let { name, value } = e.currentTarget;
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
		let err;
		if (actionProp != null) {
			[, err] = await API.connections.setAction(c.ID, a.ID, a);
		} else {
			[, err] = await API.connections.addAction(c.ID, a);
		}
		if (err != null) {
			showError(err);
			return;
		}
		showStatus(statuses.actionSaved);
		onClose();
	};

	const getDefaultMappings = (actionType) => {
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
				};
				if (subP.type.name === 'Object') {
					let nestedSubProperties = getSubProperties(key, subP.type.properties, indentation);
					subProperties = { ...subProperties, ...nestedSubProperties };
				}
			}
			return subProperties;
		};
		let defaultMappings = {};
		for (let p of actionType.Schema.properties) {
			let indentation = 0;
			defaultMappings[p.name] = { value: '', indentation: indentation, root: p.name, disabled: false };
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
			PythonSource: defaultTransformationFunction,
		};
	};

	if (action === null || actionType === null) {
		return;
	}

	let conditions = [];
	for (let [i, condition] of action.Filter.Conditions.entries()) {
		conditions.push(
			<div className='condition' data-id={i}>
				<SlInput
					data-fragment='Property'
					size='small'
					className='property'
					value={condition.Property}
					onSlInput={onUpdateConditionFragment}
				/>
				<SlInput
					data-fragment='Operator'
					size='small'
					className='operator'
					value={condition.Operator}
					onSlInput={onUpdateConditionFragment}
				/>
				<SlInput
					data-fragment='Value'
					size='small'
					className='value'
					value={condition.Value}
					onSlInput={onUpdateConditionFragment}
				/>
				<SlButton className='removeCondition' size='small' variant='danger' onClick={onRemoveCondition}>
					<SlIcon name='trash' slot='prefix'></SlIcon>
					Remove
				</SlButton>
			</div>
		);
	}

	return (
		<div className='action'>
			<div className='actionType'>
				<img src={c.LogoURL} alt={`${c.Name}'s logo`} className='actionTypeLogo' />
				<div className='actionTypeName'>{actionProp != null ? actionProp.Name : actionType.Name}</div>
				<div className='actionTypeDescription'>{actionType.Description}</div>
			</div>
			<Section title='Name' description='The name that will be associated to the action'>
				<SlInput className='nameInput' value={action.Name} onSlInput={onUpdateName} />
			</Section>
			<Section title='Filter' description='The filters that define the action'>
				{action.Filter.Conditions.length > 1 && (
					<SlSelect
						className='logical'
						size='small'
						value={action.Filter.Logical}
						onSlChange={onSwitchFilterLogical}
					>
						<SlMenuItem value='all'>All</SlMenuItem>
						<SlMenuItem value='any'>Any</SlMenuItem>
					</SlSelect>
				)}
				{conditions}
				<SlButton className='addCondition' size='small' variant='default' onClick={onAddCondition}>
					<SlIcon name='plus' slot='prefix'></SlIcon>
					Add new condition
				</SlButton>
			</Section>
			{actionType.Schema != null &&
				<Section
					title='Properties'
					description='The relation between the event properties and the action type properties'
					actions={
						propertiesMode === '' ? null : propertiesMode === 'mappings' ? (
							<SlButton variant='neutral' size='small' onClick={() => setIsAlertOpen(true)}>
								Switch to transformation function
							</SlButton>
						) : (
							<SlButton variant='neutral' size='small' onClick={() => setIsAlertOpen(true)}>
								Switch to mappings
							</SlButton>
						)
					}
				>
					{propertiesMode === '' ? (
						<div className='propertiesButtons'>
							<SlButton variant='default' onClick={onSetMappingsMode}>
								<SlIcon name='shuffle' slot='prefix'></SlIcon>
								Map the properties
							</SlButton>
							<span>or</span>
							<SlButton variant='default' onClick={onSetTransformationMode}>
								<SlIcon name='filetype-py' slot='prefix'></SlIcon>
								Write a transformation function
							</SlButton>
						</div>
					) : propertiesMode === 'mappings' ? (
						<div className='mappings'>
							{Object.keys(action.Mapping).map((k) => {
								return (
									<div
										className='mapping'
										style={{ '--mapping-indentation': `${action.Mapping[k].indentation * 30}px` }}
									>
										<SlInput
											size='small'
											value={action.Mapping[k].value}
											type='text'
											name={k}
											onSlInput={onMappingUpdate}
											disabled={action.Mapping[k].disabled}
											className='inputProperty'
										/>
										<div className='arrow'>
											<SlIcon name='arrow-right' />
										</div>
										<SlInput
											readonly
											size='small'
											value={k}
											type='text'
											name={k}
											onSlInput={null}
											className={`outputProperty${
												action.Mapping[k].indentation > 0 ? ' indented' : ''
											}`}
										/>
									</div>
								);
							})}
						</div>
					) : (
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
									onClick={() => setIsEventSchemaDialogOpen(true)}
								>
									<SlIcon name='plus' slot='prefix'></SlIcon>
									Add new property
								</SlButton>
							</div>
							<div className='editorWrapper'>
								<Editor
									onChange={(value) => onChangeTransformationPythonSource(value)}
									defaultLanguage='python'
									value={action.Transformation.PythonSource}
									theme='vs-primary'
								/>
							</div>
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
									onClick={() => setIsActionTypeSchemaDialogOpen(true)}
								>
									<SlIcon name='plus' slot='prefix'></SlIcon>
									Add new property
								</SlButton>
							</div>
						</div>
					)}
				</Section>
			}
			{actionType.Endpoints != null && Object.keys(actionType.Endpoints).length > 1 && (
				<Section
					title='Endpoint'
					description='The location of the server to which the action events will be sent'
				>
					<SlSelect
						className='endpoint'
						size='small'
						value={String(action.Endpoint)}
						onSlChange={onChangeEndpoint}
					>
						{Object.entries(actionType.Endpoints).map(([key, endpoint]) => {
							return <SlMenuItem value={key}>{endpoint}</SlMenuItem>;
						})}
					</SlSelect>
				</Section>
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
								If you switch to the mappings you will <b>PERMANENTLY</b> lose the transformation code
								you have currently written
							</p>
						)}
					</div>
				</AlertDialog>,
				document.body
			)}
			{action.Transformation != null &&
				createPortal(
					<SlDialog
						className='eventSchemaDialog'
						label='Event properties'
						open={isEventSchemaDialogOpen}
						onSlRequestClose={() => setIsEventSchemaDialogOpen(false)}
						style={{ '--width': '700px' }}
					>
						{eventsSchema.properties.map((p) => {
							let isUsed =
								action.Transformation.In.properties.findIndex((prop) => prop.name === p.name) !== -1;
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
						className='actionTypeSchemaDialog'
						label='Action type properties'
						open={isActionTypeSchemaDialogOpen}
						onSlRequestClose={() => setIsActionTypeSchemaDialogOpen(false)}
						style={{ '--width': '700px' }}
					>
						{actionTypeSchema.properties.map((p) => {
							let isUsed =
								action.Transformation.Out.properties.findIndex((prop) => prop.name === p.name) !== -1;
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
			<div className='saveWrapper'>
				<SlButton variant='primary' disabled={(actionType.Schema != null) && propertiesMode === ''} onClick={onSave}>
					Save
				</SlButton>
			</div>
		</div>
	);
};

export default Action;

import { useState, useEffect, useContext, useRef } from 'react';
import './Action.css';
import * as variants from '../../../../constants/variants';
import * as icons from '../../../../constants/icons';
import ActionMapping from './ActionMapping';
import ActionPath from './ActionPath';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionMatchingProperties from './ActionMatchingProperties';
import ActionIdentifiers from './ActionIdentifiers';
import {
	convertActionMapping,
	convertActionIdentifiers,
	computeDefaultAction,
	computeActionFields,
	flattenSchema,
	getExpressionVariables,
} from '../../../../lib/connections/action';
import { AppContext } from '../../../../providers/AppProvider';
import { ConnectionContext } from '../../../../providers/ConnectionProvider';
import { UnprocessableError, NotFoundError } from '../../../../lib/api/errors';
import { SlButton, SlInput, SlIconButton } from '@shoelace-style/shoelace/dist/react/index.js';

const Action = ({ actionType: providedActionType, action: providedAction, onClose }) => {
	const [action, setAction] = useState(null);
	const [actionType, setActionType] = useState(null);
	const [mode, setMode] = useState('');
	const [fields, setFields] = useState([]);
	const [inputSchema, setInputSchema] = useState(null);
	const [outputSchema, setOutputSchema] = useState(null);
	const [isNameEditable, setIsNameEditable] = useState(false);
	const [isSaveButtonLoading, setIsSaveButtonLoading] = useState(false);
	const [isFileChanged, setIsFileChanged] = useState(false);
	const [isQueryChanged, setIsQueryChanged] = useState(false);

	const { api, showError, showStatus, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const mappingSectionRef = useRef(null);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		const fetchData = async () => {
			// Get the action type.
			let actionType;
			if (isEditing) {
				actionType = connection.actionTypeByAction(providedAction);
			} else {
				actionType = { ...providedActionType };
			}
			setActionType(actionType);

			// Get the action type schemas.
			const [schemas, err] = await api.connections.actionSchemas(
				connection.id,
				actionType.Target,
				actionType.EventType
			);
			if (err != null) {
				onClose();
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'NoUsersSchema':
							showStatus([variants.DANGER, icons.NOT_FOUND, 'The user schema is not currently defined']);
							break;
						case 'NoGroupsSchema':
							showStatus([
								variants.DANGER,
								icons.NOT_FOUND,
								'The groups schema is not currently defined',
							]);
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

			// Compute which fields are supported by the action.
			const fields = computeActionFields(connection, actionType, schemas);
			setFields(fields);

			// If the action is on a database or file source, the schema must be
			// fetched from the linked database / file.
			if (fields.includes('Query') && providedAction != null) {
				const [res, err] = await api.connections.query(connection.id, providedAction.Query, 0);
				if (err !== null) {
					if (err instanceof NotFoundError) {
						redirect('connections');
						showStatus([variants.DANGER, icons.NOT_FOUND, 'The connection does not exist anymore']);
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
				setInputSchema(res.Schema);
			} else if (fields.includes('Path') && providedAction != null && isImport) {
				const [res, err] = await api.connections.records(
					connection.id,
					providedAction.Path,
					providedAction.Sheet,
					0
				);
				if (err != null) {
					if (err instanceof UnprocessableError) {
						switch (err.code) {
							case 'ReadFileFailed':
								showStatus([variants.DANGER, icons.INVALID_INSERTED_VALUE, err.message]);
								break;
							case 'NoStorage':
								showStatus([
									variants.DANGER,
									icons.NOT_FOUND,
									'The storage of this file connection does not exist anymore',
								]);
								break;
							default:
								break;
						}
						return;
					}
					showError(err);
					return;
				}
				setInputSchema(res.schema);
			}

			// Compute the action in a UI-friendly format.
			let action;
			if (isEditing) {
				// TODO: merge all this conversion inside a `convertToUIFormat`
				// method in a new `Action` class and also define a
				// `convertToServerFormat` method.
				if (providedAction.Mapping != null) {
					providedAction.Mapping = convertActionMapping(providedAction.Mapping, schemas.Out);
				}
				if (providedAction.Identifiers != null) {
					providedAction.Identifiers = convertActionIdentifiers(
						providedAction.Identifiers,
						providedAction.Mapping
					);
				}
				action = { ...providedAction };
			} else {
				action = computeDefaultAction(actionType, schemas.Out, fields);
			}
			setAction(action);
		};
		fetchData();
	}, []);

	const onUpdateName = (e) => {
		const a = { ...action };
		a.Name = e.currentTarget.value;
		setAction(a);
	};

	const onSave = async () => {
		const actionToSet = { ...action };
		const flattenedInputSchema = flattenSchema(inputSchema);
		const flattenedOutputSchema = flattenSchema(outputSchema);
		// TODO: extract validation and data transformation / conversion in
		// lib/action.js.
		if (fields.includes('Identifiers')) {
			if (actionToSet.Identifiers.length === 0) {
				showError(`You must define at least one identifier`);
				return;
			} else {
				if (actionToSet.Mapping == null) {
					actionToSet.Mapping = flattenedOutputSchema;
				}
				for (let i = 0; i < actionToSet.Identifiers.length; i++) {
					const [inputIdentifier, outputIdentifier] = actionToSet.Identifiers[i];
					if (inputIdentifier === '' || outputIdentifier === '') {
						showError(`You cannot use an empty value in the identifiers`);
						return;
					}
					const variables = getExpressionVariables(inputIdentifier);
					for (const variable of variables) {
						if (!(variable in flattenedInputSchema)) {
							showError(`Property ${variable} used in identifiers doesn't exist`);
							return;
						}
					}
					if (!(outputIdentifier in flattenedOutputSchema)) {
						showError(`Property ${outputIdentifier} used as identifier doesn't exist`);
						return;
					}
					const otherIdentifiers = [
						...actionToSet.Identifiers.slice(0, i),
						...actionToSet.Identifiers.slice(i + 1),
					];
					for (const [otherInputIdentifier, otherOutputIdenifier] of otherIdentifiers) {
						if (outputIdentifier === otherOutputIdenifier) {
							showError(`Property ${outputIdentifier} is used more than once in the identifiers`);
							return;
						}
					}
					actionToSet.Mapping[outputIdentifier].value = inputIdentifier;
				}
				actionToSet.Identifiers = actionToSet.Identifiers.map(
					([inputIdentifier, outputIdentifier]) => outputIdentifier
				);
			}
		}

		const inSchema = { name: 'Object', properties: [] };
		const outSchema = { name: 'Object', properties: [] };

		if (actionToSet.Mapping != null) {
			const mappingToSave = {};
			for (const k in actionToSet.Mapping) {
				const v = actionToSet.Mapping[k];
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
			actionToSet.Mapping = mappingToSave;
		}

		if (actionToSet.Transformation != null) {
			for (const propertyName of actionToSet.Transformation.In) {
				const isPropertyAlreadyInSchema = inSchema.properties.find((p) => p.name === propertyName);
				if (!isPropertyAlreadyInSchema) {
					const fullProperty = flattenedInputSchema[propertyName].full;
					inSchema.properties.push(fullProperty);
				}
			}
			for (const propertyName of actionToSet.Transformation.Out) {
				const isPropertyAlreadyInSchema = inSchema.properties.find((p) => p.name === propertyName);
				if (!isPropertyAlreadyInSchema) {
					const fullProperty = flattenedOutputSchema[propertyName].full;
					outSchema.properties.push(fullProperty);
				}
			}
			actionToSet.Transformation.Func = actionToSet.Transformation.Func.trim();
		}

		if (inSchema.properties.length > 0 && outSchema.properties.length > 0) {
			actionToSet.InSchema = inSchema;
			actionToSet.OutSchema = outSchema;
		} else {
			actionToSet.InSchema = null;
			actionToSet.OutSchema = null;
		}

		if (actionToSet.Query != null) {
			actionToSet.Query = actionToSet.Query.trim();
		}

		let id, err;
		if (isEditing) {
			[, err] = await api.connections.setAction(connection.id, actionToSet.ID, actionToSet);
		} else {
			[id, err] = await api.connections.addAction(connection.id, {
				Target: actionType.Target,
				EventType: actionType.EventType,
				Action: actionToSet,
			});
		}
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'EventTypeNotExists':
					case 'PropertyNotExists':
						showStatus([variants.DANGER, icons.NOT_FOUND, err.message]);
						break;
					case 'TargetAlreadyExists':
						showStatus([variants.DANGER, icons.FORBIDDEN, err.message]);
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

	const mustComputeSchema =
		(connection.type === 'Database' || connection.type === 'File') && inputSchema == null && !isEditing;
	const hasQueryError = connection.type === 'Database' && inputSchema == null && isEditing;
	const hasRecordsError = connection.type === 'File' && inputSchema == null && isEditing;
	const isMappingSectionDisabled = hasQueryError || isQueryChanged || hasRecordsError || (isFileChanged && isImport);

	let disabledReason = '';
	if (hasQueryError) {
		disabledReason =
			'Mappings are disabled since the query returned an error. Fix the query before proceding to mappings.';
	} else if (hasRecordsError) {
		disabledReason =
			'Mappings are disabled since the file fetch returned an error. Fix the file informations before proceding to mappings.';
	} else if (connection.type === 'Database') {
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
						{connection.logo}
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
						{providedAction != null ? 'Save' : 'Add'}
					</SlButton>
				</div>
			</div>
			<div className='body'>
				{fields.includes('Filter') && (
					<ActionFilters action={action} setAction={setAction} inputSchema={inputSchema} />
				)}
				{fields.includes('Query') && (
					<ActionQuery
						connection={connection}
						action={action}
						setInputSchema={setInputSchema}
						mappingSectionRef={mappingSectionRef}
						setAction={setAction}
						setIsQueryChanged={setIsQueryChanged}
					/>
				)}
				{fields.includes('Path') && (
					<ActionPath
						fields={fields}
						connection={connection}
						action={action}
						setAction={setAction}
						actionType={actionType}
						setInputSchema={setInputSchema}
						isImport={isImport}
						mappingSectionRef={mappingSectionRef}
						setIsFileChanged={setIsFileChanged}
					/>
				)}
				{fields.includes('ExportMode') && <ActionExportMode action={action} setAction={setAction} />}
				{fields.includes('MatchingProperties') && (
					<ActionMatchingProperties
						connection={connection}
						action={action}
						setAction={setAction}
						inputSchema={inputSchema}
						outputSchema={outputSchema}
					/>
				)}
				{fields.includes('Identifiers') && (
					<ActionIdentifiers
						action={action}
						setAction={setAction}
						inputSchema={inputSchema}
						outputSchema={outputSchema}
					/>
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
						fields={fields}
					/>
				)}
			</div>
		</div>
	);
};

export default Action;

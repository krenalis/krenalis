import { useEffect, useState, useContext, useMemo } from 'react';
import {
	flattenSchema,
	transformActionMapping,
	computeDefaultAction,
	computeActionTypeFields,
	TransformedActionType,
	TransformedAction,
	TransformedMapping,
} from '../lib/helpers/transformedAction';
import { AppContext } from '../context/providers/AppProvider';
import * as variants from '../constants/variants';
import * as icons from '../constants/icons';
import TransformedConnection, { getActionTypeFromConnection } from '../lib/helpers/transformedConnection';
import {
	TransformedIdentifiers,
	transformActionIdentifiers,
	untransformActionIdentifiers,
	validateIdentifiersMapping,
} from '../lib/helpers/transformedIdentifiers';
import { UnprocessableError, NotFoundError } from '../lib/api/errors';
import { Action, ActionToSet, ActionType, Mapping, MappingExpression, Transformation } from '../types/external/action';
import { ActionSchemasResponse, ExecQueryResponse, RecordsResponse } from '../types/external/api';
import { ObjectType } from '../types/external/types';

const useActionData = (
	onClose: () => void,
	connection: TransformedConnection,
	providedActionType: ActionType,
	providedAction: Action,
	setIsSaveButtonLoading: React.Dispatch<React.SetStateAction<boolean>>
) => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [action, setAction] = useState<TransformedAction>();
	const [actionType, setActionType] = useState<TransformedActionType>();

	const { api, showError, showStatus, redirect } = useContext(AppContext);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		const fetchData = async () => {
			// Get the action type.
			let actionType: ActionType | undefined;
			if (isEditing) {
				actionType = getActionTypeFromConnection(connection, providedAction.Target, providedAction.EventType);
				if (actionType == null) {
					console.error(
						`Action type with target ${providedAction.Target}${
							providedAction.EventType ? ' and event type ' + providedAction.EventType : ''
						} does not exists anymore`
					);
					return;
				}
			} else {
				actionType = { ...providedActionType };
			}

			// Get the action type schemas.
			let schemas: ActionSchemasResponse;
			try {
				schemas = await api.connections.actionSchemas(connection.id, actionType.Target, actionType.EventType);
			} catch (err) {
				onClose();
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'NoUsersSchema':
							showStatus({
								variant: variants.DANGER,
								icon: icons.NOT_FOUND,
								text: 'The user schema is not currently defined',
							});
							break;
						case 'NoGroupsSchema':
							showStatus({
								variant: variants.DANGER,
								icon: icons.NOT_FOUND,
								text: 'The groups schema is not currently defined',
							});
							break;
						case 'EventTypeNotExists':
							showStatus({ variant: variants.DANGER, icon: icons.NOT_FOUND, text: err.message });
							break;
						default:
							break;
					}
					return;
				}
				showError(err);
				return;
			}
			let inputSchema = schemas.In;
			let outputSchema = schemas.Out;

			// Compute which fields are supported by the action type.
			const fields = computeActionTypeFields(connection, actionType, schemas);

			// If the action type is an import from a database source, the input
			// schema is the schema of the database table itself.
			if (fields.includes('Query') && isEditing) {
				let res: ExecQueryResponse;
				try {
					res = await api.connections.query(connection.id, providedAction.Query!, 0);
				} catch (err) {
					if (err instanceof NotFoundError) {
						redirect('connections');
						showStatus({
							variant: variants.DANGER,
							icon: icons.NOT_FOUND,
							text: 'The connection does not exist anymore',
						});
						return;
					}
					if (err instanceof UnprocessableError) {
						if (err.code === 'QueryExecutionFailed') {
							let statusMessage: string;
							if (err.cause && err.cause !== '') {
								statusMessage = err.cause;
							} else {
								statusMessage = err.message;
							}
							showStatus({ variant: variants.DANGER, icon: icons.CODE_ERROR, text: statusMessage });
						}
						return;
					}
					showError(err);
					return;
				}
				inputSchema = res.Schema;
			}

			// If the action type is an import from a file source, the input
			// schema is the schema of the file itself.
			if (fields.includes('Path') && isEditing && isImport) {
				let res: RecordsResponse;
				try {
					res = await api.connections.records(connection.id, providedAction.Path!, providedAction.Sheet, 0);
				} catch (err) {
					if (err instanceof UnprocessableError) {
						switch (err.code) {
							case 'ReadFileFailed':
								showStatus({
									variant: variants.DANGER,
									icon: icons.INVALID_INSERTED_VALUE,
									text: err.message,
								});
								break;
							case 'NoStorage':
								showStatus({
									variant: variants.DANGER,
									icon: icons.NOT_FOUND,
									text: 'The storage of this file connection does not exist anymore',
								});
								break;
							default:
								break;
						}
						return;
					}
					showError(err);
					return;
				}
				inputSchema = res.schema;
			}

			const transformedActionType: TransformedActionType = {
				Name: actionType.Name,
				Description: actionType.Description,
				Target: actionType.Target,
				EventType: actionType.EventType,
				MissingSchema: actionType.MissingSchema,
				InputSchema: inputSchema,
				OutputSchema: outputSchema,
				Fields: fields,
			};
			setActionType(transformedActionType);

			// Compute the action in a UI-friendly format.
			let transformedAction: TransformedAction;
			if (isEditing) {
				// TODO: merge this conversions inside a single transformation
				// function.
				let transformedMapping: TransformedMapping | null = null;
				let transformedIdentifiers: TransformedIdentifiers | null = null;
				if (providedAction.Mapping != null) {
					transformedMapping = transformActionMapping(providedAction.Mapping, schemas.Out);
					if (providedAction.Identifiers != null) {
						transformedIdentifiers = transformActionIdentifiers(
							providedAction.Identifiers,
							transformedMapping!
						);
					}
				}
				transformedAction = {
					ID: providedAction.ID,
					Connection: providedAction.Connection,
					Target: providedAction.Target,
					Name: providedAction.Name,
					Enabled: providedAction.Enabled,
					EventType: providedAction.EventType,
					Running: providedAction.Running,
					ScheduleStart: providedAction.ScheduleStart,
					SchedulePeriod: providedAction.SchedulePeriod,
					InSchema: providedAction.InSchema,
					OutSchema: providedAction.OutSchema,
					Filter: providedAction.Filter,
					Mapping: transformedMapping,
					Transformation: providedAction.Transformation,
					Identifiers: transformedIdentifiers,
					Query: providedAction.Query,
					Path: providedAction.Path,
					Table: providedAction.Table,
					Sheet: providedAction.Sheet,
					ExportMode: providedAction.ExportMode,
					MatchingProperties: providedAction.MatchingProperties,
				};
			} else {
				transformedAction = computeDefaultAction(actionType, schemas.Out, fields);
			}
			setAction(transformedAction);
			setIsLoading(false);
		};
		fetchData();
	}, [providedActionType, providedAction]);

	const isTransformationAllowed = useMemo(
		() => connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server',
		[connection, providedActionType, providedAction]
	);

	const saveAction = async () => {
		if (action == null || actionType == null) {
			return;
		}

		let identifiers: string[];
		let mapping: Mapping;
		let inSchema: ObjectType;
		let outSchema: ObjectType;
		let transformation: Transformation;
		let query: string;

		// TODO: extract validation and data transformation / conversion in
		// lib/action.js.
		const flattenedInputSchema = flattenSchema(actionType.InputSchema);
		const flattenedOutputSchema = flattenSchema(actionType.OutputSchema);
		if (actionType.Fields.includes('Identifiers')) {
			if (action.Identifiers!.length === 0) {
				showError(`You must define at least one identifier`);
				return;
			} else {
				if (action.Mapping == null) {
					action.Mapping = flattenedOutputSchema;
				}
				const errorMessage = validateIdentifiersMapping(action.Identifiers!);
				if (errorMessage) {
					showError(errorMessage);
					return;
				}
				for (const [mapped, identifier] of action.Identifiers!) {
					action.Mapping![identifier.value].value = mapped.value;
				}
				identifiers = untransformActionIdentifiers(action.Identifiers!);
			}
		}

		if (action.Mapping != null) {
			const inputSchema: ObjectType = { name: 'Object', properties: [] };
			const outputSchema: ObjectType = { name: 'Object', properties: [] };
			const mappingToSave = {};
			const expressions: MappingExpression[] = [];
			for (const k in action.Mapping) {
				const v = action.Mapping[k];
				if (v.value === '') {
					continue;
				}
				if (v.error && v.error !== '') {
					showError(`You must fix the errors on the mapping before saving`);
					return;
				}
				const property = flattenedOutputSchema![k];
				const fullProperty = property.full;
				const parentProperty = flattenedOutputSchema![property.root!].full;
				expressions.push({
					value: v.value,
					type: fullProperty!.type,
					nullable: fullProperty!.nullable,
				});
				mappingToSave[k] = v.value;
				const isKeyPropertyAlreadyInSchema = outputSchema.properties!.find(
					(p) => p.name === parentProperty!.name
				);
				if (!isKeyPropertyAlreadyInSchema) {
					outputSchema.properties!.push(parentProperty);
				}
			}
			let inputProperties: string[];
			try {
				inputProperties = await api.expressionsProperties(expressions, actionType.InputSchema);
			} catch (err) {
				showError(err);
				return;
			}
			for (const prop of inputProperties) {
				const parentName = prop.split('.')[0];
				const isPropertyAlreadyInSchema = inputSchema.properties!.find((p) => p.name === parentName);
				if (!isPropertyAlreadyInSchema) {
					const fullProperty = flattenedInputSchema![parentName].full;
					inputSchema.properties!.push(fullProperty);
				}
			}
			mapping = mappingToSave;
			inSchema = inputSchema;
			outSchema = outputSchema;
		}

		if (action.Transformation != null) {
			inSchema = actionType.InputSchema;
			outSchema = actionType.OutputSchema;
			transformation = {
				In: actionType.InputSchema.properties!.map((p) => p.name),
				Out: actionType.OutputSchema.properties!.map((p) => p.name),
				Func: action.Transformation.Func.trim(),
			};
		}

		if (action.Query != null) {
			query = action.Query.trim();
		}

		let actionToSet: ActionToSet = {
			name: action.Name,
			enabled: action.Enabled,
			filter: action.Filter,
			inSchema: inSchema!,
			outSchema: outSchema!,
			identifiers: identifiers!,
			mapping: mapping!,
			transformation: transformation!,
			query: query!,
			path: action.Path,
			tableName: action.Table,
			sheet: action.Sheet,
			exportMode: action.ExportMode,
			matchingProperties: action.MatchingProperties,
		};

		let id: number = 0;
		try {
			if (isEditing) {
				await api.connections.setAction(connection.id, action.ID!, actionToSet);
			} else {
				id = await api.connections.addAction(
					connection.id,
					actionType.Target,
					actionType.EventType,
					actionToSet
				);
			}
		} catch (err) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'EventTypeNotExists':
					case 'PropertyNotExists':
						showStatus({ variant: variants.DANGER, icon: icons.NOT_FOUND, text: err.message });
						break;
					case 'TargetAlreadyExists':
						showStatus({ variant: variants.DANGER, icon: icons.FORBIDDEN, text: err.message });
						break;
					default:
						break;
				}
				return;
			}
			showError(err);
			return;
		}

		sessionStorage.setItem('newActionID', String(id));
		setIsSaveButtonLoading(true);
		setTimeout(() => {
			setIsSaveButtonLoading(false);
			onClose();
		}, 200);
	};

	return {
		isEditing,
		isImport,
		isTransformationAllowed,
		action,
		isLoading,
		actionType,
		setActionType,
		setAction,
		saveAction,
	};
};

export default useActionData;

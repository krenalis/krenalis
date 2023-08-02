import { useEffect, useState, useContext, useMemo } from 'react';
import {
	flattenSchema,
	convertActionMapping,
	computeDefaultAction,
	computeActionTypeFields,
} from '../lib/helpers/action';
import { AppContext } from '../context/providers/AppProvider';
import * as variants from '../constants/variants';
import * as icons from '../constants/icons';
import { getActionTypeFromConnection } from '../lib/helpers/connection';
import {
	transformActionIdentifiers,
	untransformActionIdentifiers,
	validateIdentifiersMapping,
} from '../lib/helpers/identifiers';
import { UnprocessableError, NotFoundError } from '../lib/api/errors';

const useActionData = (onClose, connection, providedActionType, providedAction, setIsSaveButtonLoading) => {
	const [isLoading, setIsLoading] = useState(true);
	const [action, setAction] = useState(null);
	const [actionType, setActionType] = useState(null);

	const { api, showError, showStatus, redirect } = useContext(AppContext);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		const fetchData = async () => {
			// Get the action type.
			let actionType;
			if (isEditing) {
				actionType = getActionTypeFromConnection(connection, providedAction.Target, providedAction.EventType);
			} else {
				actionType = { ...providedActionType };
			}

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
			actionType.InputSchema = schemas.In;
			actionType.OutputSchema = schemas.Out;

			// Compute which fields are supported by the action type.
			const fields = computeActionTypeFields(connection, actionType, schemas);
			actionType.Fields = fields;

			// If the action type is an import from a database source, the input
			// schema is the schema of the database table itself.
			if (fields.includes('Query') && isEditing) {
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
				actionType.InputSchema = res.Schema;
			}

			// If the action type is an import from a file source, the input
			// schema is the schema of the file itself.
			if (fields.includes('Path') && isEditing && isImport) {
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
				actionType.InputSchema = res.schema;
			}

			setActionType(actionType);

			// Compute the action in a UI-friendly format.
			let action;
			if (isEditing) {
				// TODO: merge all this conversion inside a single
				// transformation function.
				if (providedAction.Mapping != null) {
					providedAction.Mapping = convertActionMapping(providedAction.Mapping, schemas.Out);
				}
				if (providedAction.Identifiers != null) {
					providedAction.Identifiers = transformActionIdentifiers(
						providedAction.Identifiers,
						providedAction.Mapping
					);
				}
				action = { ...providedAction };
			} else {
				action = computeDefaultAction(actionType, schemas.Out, fields);
			}
			setAction(action);
			setIsLoading(false);
		};
		fetchData();
	}, [providedActionType, providedAction]);

	const isTransformationAllowed = useMemo(
		() => connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server',
		[connection, providedActionType, providedAction]
	);

	const saveAction = async () => {
		const actionToSet = { ...action };
		const flattenedInputSchema = flattenSchema(actionType.InputSchema);
		const flattenedOutputSchema = flattenSchema(actionType.OutputSchema);
		// TODO: extract validation and data transformation / conversion in
		// lib/action.js.
		if (actionType.Fields.includes('Identifiers')) {
			if (actionToSet.Identifiers.length === 0) {
				showError(`You must define at least one identifier`);
				return;
			} else {
				if (actionToSet.Mapping == null) {
					actionToSet.Mapping = flattenedOutputSchema;
				}
				const errorMessage = validateIdentifiersMapping(actionToSet.Identifiers);
				if (errorMessage) {
					showError(errorMessage);
					return;
				}
				for (const [mapped, identifier] of actionToSet.Identifiers) {
					actionToSet.Mapping[identifier.value].value = mapped.value;
				}
				actionToSet.Identifiers = untransformActionIdentifiers(actionToSet.Identifiers);
			}
		}

		if (actionToSet.Mapping != null) {
			const inSchema = { name: 'Object', properties: [] };
			const outSchema = { name: 'Object', properties: [] };
			const mappingToSave = {};
			const expressions = [];
			for (const k in actionToSet.Mapping) {
				const v = actionToSet.Mapping[k];
				if (v.value === '') {
					continue;
				}
				if (v.error && v.error !== '') {
					showError(`You must fix the errors on the mapping before saving`);
					return;
				}
				const fullKeyProperty = flattenedOutputSchema[k].full;
				expressions.push({
					value: v.value,
					type: fullKeyProperty.type,
					nullable: fullKeyProperty.nullable,
				});
				mappingToSave[k] = v.value;
				const isKeyPropertyAlreadyInSchema = outSchema.properties.find((p) => p.name === fullKeyProperty.name);
				if (!isKeyPropertyAlreadyInSchema) {
					outSchema.properties.push(fullKeyProperty);
				}
			}
			const [inputProperties, err] = await api.expressionsProperties(expressions, actionType.InputSchema);
			if (err) {
				showError(err);
				return;
			}
			for (const prop of inputProperties) {
				const parentName = prop.split('.')[0];
				const isPropertyAlreadyInSchema = inSchema.properties.find((p) => p.name === parentName);
				if (!isPropertyAlreadyInSchema) {
					const fullProperty = flattenedInputSchema[parentName].full;
					inSchema.properties.push(fullProperty);
				}
			}
			actionToSet.Mapping = mappingToSave;
			actionToSet.InSchema = inSchema;
			actionToSet.OutSchema = outSchema;
		}

		if (actionToSet.Transformation != null) {
			actionToSet.InSchema = actionType.InputSchema;
			actionToSet.OutSchema = actionType.OutputSchema;
			actionToSet.Transformation.In = actionType.InputSchema.properties.map((p) => p.name);
			actionToSet.Transformation.Out = actionType.OutputSchema.properties.map((p) => p.name);
			actionToSet.Transformation.Func = actionToSet.Transformation.Func.trim();
		}

		if (actionToSet.Query != null) {
			actionToSet.Query = actionToSet.Query.trim();
		}

		let id, err;
		if (isEditing) {
			[, err] = await api.connections.setAction(connection.id, actionToSet.ID, actionToSet);
		} else {
			[id, err] = await api.connections.addAction(
				connection.id,
				actionType.Target,
				actionType.EventType,
				actionToSet
			);
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

		sessionStorage.setItem('newActionID', id);
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

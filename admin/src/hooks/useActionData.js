import { useEffect, useState, useContext } from 'react';
import { computeActionTypeFields } from '../lib/connections/action';
import { convertActionMapping } from '../lib/connections/action';
import { computeDefaultAction } from '../lib/connections/action';
import { convertActionIdentifiers } from '../lib/connections/action';
import { AppContext } from '../providers/AppProvider';
import * as variants from '../constants/variants';
import * as icons from '../constants/icons';
import { getActionTypeFromConnection } from '../lib/connections/connection';
import { flattenSchema, getExpressionVariables } from '../lib/connections/action';
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
			setIsLoading(false);
		};
		fetchData();
	}, [providedActionType, providedAction]);

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

	return {
		isEditing,
		isImport,
		action,
		isLoading,
		actionType,
		setActionType,
		setAction,
		saveAction,
	};
};

export default useActionData;

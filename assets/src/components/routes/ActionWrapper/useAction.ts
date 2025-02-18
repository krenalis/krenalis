import { useEffect, useState, useContext, useMemo } from 'react';
import {
	computeDefaultAction,
	computeActionTypeFields,
	TransformedActionType,
	TransformedAction,
	transformActionType,
	transformAction,
	transformInActionToSet,
	flattenSchema,
} from '../../../lib/core/action';
import AppContext from '../../../context/AppContext';
import TransformedConnection, { getActionTypeFromConnection } from '../../../lib/core/connection';
import { UnavailableError, UnprocessableError } from '../../../lib/api/errors';
import { Action, ActionToSet, ActionType } from '../../../lib/api/types/action';
import {
	ActionSchemasResponse,
	ExecQueryResponse,
	RecordsResponse,
	ConnectorSettings,
} from '../../../lib/api/types/responses';
import { ObjectType } from '../../../lib/api/types/types';
import { FullscreenContext } from '../../../context/FullscreenContext';

const useAction = (connection: TransformedConnection, providedActionType: ActionType, providedAction: Action) => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [action, setAction] = useState<TransformedAction>();
	const [settings, setSettings] = useState<ConnectorSettings>();
	const [actionType, setActionType] = useState<TransformedActionType>();
	const [isSaveHidden, setIsSaveHidden] = useState<boolean>(false);
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isFileConnectorLoading, setIsFileConnectorLoading] = useState<boolean>(
		providedAction !== null && connection.isFile && connection.isSource,
	);
	const [isFileConnectorChanged, setIsFileConnectorChanged] = useState<boolean>(false);
	const [isTableChanged, setIsTableChanged] = useState<boolean>(false);
	const [selectedInPaths, setSelectedInPaths] = useState<string[]>([]);
	const [selectedOutPaths, setSelectedOutPaths] = useState<string[]>([]);

	const { api, handleError, redirect, connectors } = useContext(AppContext);
	const { closeFullscreen } = useContext(FullscreenContext);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		// Filter out the selected properties that are no longer in the
		// schemas.
		if (isLoading) {
			return;
		}
		if (actionType.inputSchema) {
			const flatIn = flattenSchema(actionType.inputSchema);
			const inPaths = [];
			for (const p of selectedInPaths) {
				if (flatIn[p]) {
					inPaths.push(p);
				}
			}
			setSelectedInPaths(inPaths);
		}
		if (actionType.outputSchema) {
			const flatOut = flattenSchema(actionType.outputSchema);
			const outPaths = [];
			for (const p of selectedOutPaths) {
				if (flatOut[p]) {
					outPaths.push(p);
				}
			}
			setSelectedOutPaths(outPaths);
		}
	}, [actionType]);

	useEffect(() => {
		const handleException = (err: Error | string) => {
			setTimeout(() => {
				setIsLoading(false);
				closeFullscreen();
				redirect(`connections/${connection.id}/actions`);
				handleError(err);
			}, 300);
		};

		const setupAction = async () => {
			// Get the action type.
			let actionType: ActionType;
			if (isEditing) {
				const typ = getActionTypeFromConnection(connection, providedAction.target, providedAction.eventType);
				if (typ == null) {
					console.error(
						`Action type with target ${providedAction.target}${
							providedAction.eventType ? ' and event type ' + providedAction.eventType : ''
						} does not exists anymore`,
					);
					return;
				} else {
					actionType = typ;
				}
			} else {
				actionType = { ...providedActionType };
			}

			// Fetch the action schemas.
			let inputSchema: ObjectType;
			let outputSchema: ObjectType;
			let inputMatchingSchema: ObjectType;
			let outputMatchingSchema: ObjectType;
			try {
				let schemas: ActionSchemasResponse;
				schemas = await api.workspaces.connections.actionSchemas(
					connection.id,
					actionType.target,
					actionType.eventType,
				);

				inputSchema = schemas.in;
				outputSchema = schemas.out;
				inputMatchingSchema = schemas.matchings ? schemas.matchings.internal : null;
				outputMatchingSchema = schemas.matchings ? schemas.matchings.external : null;
			} catch (err) {
				handleException(err);
				return;
			}

			// Compute which fields are supported by the action type.
			const fields = computeActionTypeFields(connection, actionType, outputSchema);

			try {
				// Handle cases that requires additional steps to
				// retrieve the schemas.

				// If the action type is an import from a database
				// source, the input schema is the schema of the
				// database table itself.
				if (fields.includes('Query') && isEditing) {
					let res: ExecQueryResponse;
					try {
						res = await api.workspaces.connections.execQuery(connection.id, providedAction.query!, 0);
						inputSchema = res.schema;
					} catch (err) {
						if (
							err instanceof UnavailableError ||
							(err instanceof UnprocessableError &&
								(err.code === 'InvalidPlaceholder' || err.code === 'UnsupportedColumnType'))
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the action (or at least can see its state
							// in order to debug the problem).
						} else {
							throw err;
						}
					}
				}

				// If the action type is an import from a file source,
				// the input schema is the schema of the file itself.
				if (fields.includes('File') && isEditing && isImport) {
					let s: ConnectorSettings = null;
					const connector = connectors.find((c) => c.name === providedAction.format);
					if (connector.hasSettings(connection.role)) {
						// get the settings of the file.
						let ui = await api.workspaces.connections.actionUiEvent(providedAction.id, 'load', null);
						s = ui.settings;
						setSettings(ui.settings);
					}
					let res: RecordsResponse;
					try {
						res = await api.workspaces.connections.records(
							connection.id,
							providedAction.path!,
							providedAction.format,
							providedAction.sheet,
							providedAction.compression,
							s,
							0,
						);
						inputSchema = res.schema;
					} catch (err) {
						if (
							err instanceof UnavailableError ||
							(err instanceof UnprocessableError &&
								(err.code === 'NoColumnsFound' ||
									err.code === 'SheetNotExist' ||
									err.code === 'UnsupportedColumnType'))
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the action (or at least can see its state
							// in order to debug the problem).
						} else {
							throw err;
						}
					}
				}

				// If the action type is an export to a database
				// destination, the output schema is the schema of the
				// database table itself.
				if (fields.includes('TableName') && isEditing) {
					let schema: ObjectType;
					try {
						schema = await api.workspaces.connections.tableSchema(connection.id, providedAction.tableName);
						outputSchema = schema;
					} catch (err) {
						if (
							err instanceof UnavailableError ||
							(err instanceof UnprocessableError && err.code === 'UnsupportedColumnType')
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the action (or at least can see its state
							// in order to debug the problem).
						} else {
							throw err;
						}
					}
				}
			} catch (err) {
				handleException(err);
				return;
			}

			const transformedActionType = transformActionType(
				actionType,
				fields,
				inputSchema,
				outputSchema,
				inputMatchingSchema,
				outputMatchingSchema,
			);
			setActionType(transformedActionType);

			let transformedAction: TransformedAction;
			if (isEditing) {
				transformedAction = transformAction(providedAction, outputSchema);
				if (transformedAction.transformation.function != null) {
					// Set the initial value of the selected properties
					// of the function.
					const func = transformedAction.transformation.function;
					setSelectedInPaths(func.inPaths);
					setSelectedOutPaths(func.outPaths);
				}
			} else {
				transformedAction = computeDefaultAction(actionType, connection, outputSchema, fields);
			}
			setAction(transformedAction);
			setIsLoading(false);
		};
		setupAction();
	}, [providedActionType, providedAction]);

	const saveAction = async () => {
		if (action == null || actionType == null) {
			return 'Invalid action or action type';
		}

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(
				action,
				settings,
				actionType,
				api,
				connection,
				true,
				selectedInPaths,
				selectedOutPaths,
			);
		} catch (err) {
			return err;
		}

		let id: number = 0;
		try {
			if (isEditing) {
				await api.workspaces.connections.updateAction(action.id!, actionToSet);
			} else {
				id = await api.workspaces.connections.createAction(
					connection.id,
					actionType.target,
					actionType.eventType,
					actionToSet,
				);
			}
		} catch (err) {
			return err;
		}

		sessionStorage.setItem('newActionID', String(id));
		return null;
	};

	const isTransformationFunctionSupported = useMemo(() => {
		if (isLoading) return false;
		if (actionType.target === 'Users' || actionType.target === 'Groups') {
			if (connection.isSource) {
				return connection.isApp || connection.isDatabase || connection.isFileStorage || connection.isEventBased;
			} else {
				return connection.isApp || connection.isDatabase;
			}
		}
		if (actionType.target == 'Events' && connection.isApp && connection.isDestination) {
			return true;
		}
		return false;
	}, [isLoading, actionType, connection]);

	const { isTransformationHidden, isTransformationDisabled } = useMemo(() => {
		if (isLoading) return { isTransformationHidden: false, isTransformationDisabled: false };
		let isTransformationHidden: boolean = false;
		let isTransformationDisabled: boolean = false;

		const inputSchemaIsNotDefined = actionType.inputSchema == null;
		const outputSchemaIsNotDefined = actionType.outputSchema == null;

		if (connection.isDatabase) {
			if (isQueryChanged || isTableChanged) {
				isTransformationDisabled = true;
			}
			if (isEditing) {
				if (connection.isSource && inputSchemaIsNotDefined) {
					// the execution of the query returned an error.
					isTransformationDisabled = true;
				}
				if (connection.isDestination && outputSchemaIsNotDefined) {
					// reading the table returned an error.
					isTransformationDisabled = true;
				}
			} else {
				if (connection.isSource && inputSchemaIsNotDefined) {
					// a valid query has not been confirmed yet.
					isTransformationHidden = true;
				}
				if (connection.isDestination && outputSchemaIsNotDefined) {
					// a valid table has not been confirmed yet.
					isTransformationHidden = true;
				}
			}
		}

		if (connection.isFileStorage) {
			if (connection.isSource && isFileChanged) {
				isTransformationDisabled = true;
			}
			if (connection.isSource && (isFileConnectorLoading || isFileConnectorChanged)) {
				isTransformationHidden = true;
			}
			if (isEditing) {
				if (connection.isSource && inputSchemaIsNotDefined) {
					// reading the file returned an error.
					isTransformationDisabled = true;
				}
			} else {
				if (connection.isSource && inputSchemaIsNotDefined) {
					// a valid file has not been confirmed yet.
					isTransformationHidden = true;
				}
			}
		}

		return {
			isTransformationHidden,
			isTransformationDisabled,
		};
	}, [
		isLoading,
		connection,
		actionType,
		isQueryChanged,
		isTableChanged,
		isEditing,
		isFileChanged,
		isFileConnectorLoading,
		isFileConnectorChanged,
	]);

	return {
		isEditing,
		isImport,
		isTransformationFunctionSupported,
		action,
		settings,
		setSettings,
		isLoading,
		actionType,
		setActionType,
		setAction,
		saveAction,
		isSaveHidden,
		setIsSaveHidden,
		setIsFileChanged,
		isFileConnectorLoading,
		setIsFileConnectorLoading,
		isFileConnectorChanged,
		setIsFileConnectorChanged,
		setIsTableChanged,
		setIsQueryChanged,
		isTransformationHidden,
		isTransformationDisabled,
		selectedInPaths,
		setSelectedInPaths,
		selectedOutPaths,
		setSelectedOutPaths,
	};
};

export { useAction };

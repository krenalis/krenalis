import { useEffect, useState, useContext, useMemo } from 'react';
import {
	computeDefaultAction,
	computeActionTypeFields,
	TransformedActionType,
	TransformedAction,
	transformActionType,
	transformAction,
	transformInActionToSet,
} from '../../../lib/core/action';
import AppContext from '../../../context/AppContext';
import TransformedConnection, { getActionTypeFromConnection } from '../../../lib/core/connection';
import { UnprocessableError } from '../../../lib/api/errors';
import { Action, ActionToSet, ActionType } from '../../../lib/api/types/action';
import {
	ActionSchemasResponse,
	ExecQueryResponse,
	RecordsResponse,
	ConnectorValues,
} from '../../../lib/api/types/responses';
import { ObjectType } from '../../../lib/api/types/types';
import { sleep } from '../../../utils/sleep';
import { FullscreenContext } from '../../../context/FullscreenContext';

const useAction = (
	connection: TransformedConnection,
	providedActionType: ActionType,
	providedAction: Action,
	setIsSaveButtonLoading: React.Dispatch<React.SetStateAction<boolean>>,
) => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [action, setAction] = useState<TransformedAction>();
	const [values, setValues] = useState<ConnectorValues>();
	const [actionType, setActionType] = useState<TransformedActionType>();
	const [isSaveHidden, setIsSaveHidden] = useState<boolean>(false);
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isFileConnectorLoading, setIsFileConnectorLoading] = useState<boolean>(
		providedAction !== null && connection.isFile && connection.isSource,
	);
	const [isFileConnectorChanged, setIsFileConnectorChanged] = useState<boolean>(false);
	const [isTableChanged, setIsTableChanged] = useState<boolean>(false);

	const { api, handleError, redirect, connectors } = useContext(AppContext);
	const { closeFullscreen } = useContext(FullscreenContext);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
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

			// Compute which fields are supported by the action type.
			const fields = computeActionTypeFields(connection, actionType);

			// Compute the action schemas.
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

				// If the action type is an import from a database
				// source, the input schema is the schema of the
				// database table itself.
				if (fields.includes('Query') && isEditing) {
					let res: ExecQueryResponse;
					try {
						res = await api.workspaces.connections.query(connection.id, providedAction.query!, 0);
						inputSchema = res.schema;
					} catch (err) {
						if (
							err instanceof UnprocessableError &&
							(err.code === 'InvalidPlaceholder' || err.code === 'UnsupportedColumnType')
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the action.
						} else {
							throw err;
						}
					}
				}

				// If the action type is an import from a file source,
				// the input schema is the schema of the file itself.
				if (fields.includes('File') && isEditing && isImport) {
					let values: ConnectorValues = null;
					const connector = connectors.find((c) => c.name === providedAction.connector);
					if (connector.hasUI) {
						// get the values of the file settings.
						let ui = await api.workspaces.connections.actionUiEvent(
							connection.id,
							providedAction.id,
							'load',
							null,
						);
						values = ui.values;
						setValues(ui.values);
					}
					let res: RecordsResponse;
					try {
						res = await api.workspaces.connections.records(
							connection.id,
							providedAction.connector,
							providedAction.path!,
							providedAction.sheet,
							providedAction.compression,
							values,
							0,
						);
						inputSchema = res.schema;
					} catch (err) {
						if (
							err instanceof UnprocessableError &&
							(err.code === 'NoColumnsFound' ||
								err.code === 'SheetNotExist' ||
								err.code === 'UnsupportedColumnType')
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the action.
						} else {
							throw err;
						}
					}
				}

				// If the action type is an export to a database
				// destination, the output schema is the schema of the
				// database table itself.
				if (fields.includes('Table') && isEditing) {
					let schema: ObjectType;
					try {
						schema = await api.workspaces.connections.tableSchema(connection.id, providedAction.table);
						outputSchema = schema;
					} catch (err) {
						if (err instanceof UnprocessableError && err.code === 'UnsupportedColumnType') {
							handleError(err.message);
							// continue execution so that user can fix
							// the action.
						} else {
							throw err;
						}
					}
				}
			} catch (err) {
				setTimeout(() => {
					setIsLoading(false);
					closeFullscreen();
					redirect(`connections/${connection.id}/actions`);
					handleError(err);
				}, 300);
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
			actionToSet = await transformInActionToSet(action, values, actionType, api, connection, true);
		} catch (err) {
			return err;
		}

		let id: number = 0;
		try {
			if (isEditing) {
				await api.workspaces.connections.setAction(connection.id, action.id!, actionToSet);
			} else {
				id = await api.workspaces.connections.addAction(
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
		setIsSaveButtonLoading(true);
		await sleep(200);
		setIsSaveButtonLoading(false);
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
					// reading the table returned an erro.
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
		values,
		setValues,
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
	};
};

export { useAction };

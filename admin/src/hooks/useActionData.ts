import { useEffect, useState, useContext } from 'react';
import {
	computeDefaultAction,
	computeActionTypeFields,
	TransformedActionType,
	TransformedAction,
	transformActionType,
	transformAction,
	transformInActionToSet,
} from '../lib/helpers/transformedAction';
import AppContext from '../context/AppContext';
import TransformedConnection, { getActionTypeFromConnection } from '../lib/helpers/transformedConnection';
import { UnprocessableError, NotFoundError } from '../lib/api/errors';
import { Action, ActionToSet, ActionType } from '../types/external/action';
import { ActionSchemasResponse, ExecQueryResponse, RecordsResponse } from '../types/external/api';
import { ObjectType } from '../types/external/types';
import { sleep } from '../lib/utils/sleep';
import { FullscreenContext } from '../context/FullscreenContext';

const useAction = (
	connection: TransformedConnection,
	providedActionType: ActionType,
	providedAction: Action,
	setIsSaveButtonLoading: React.Dispatch<React.SetStateAction<boolean>>,
) => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [action, setAction] = useState<TransformedAction>();
	const [actionType, setActionType] = useState<TransformedActionType>();
	const [isSaveHidden, setIsSaveHidden] = useState<boolean>(false);
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isFileConnectorLoading, setIsFileConnectorLoading] = useState<boolean>(
		providedAction !== null && connection.isFile && connection.isSource,
	);
	const [isFileConnectorChanged, setIsFileConnectorChanged] = useState<boolean>(false);
	const [isTableChanged, setIsTableChanged] = useState<boolean>(false);

	const { api, handleError, setIsLoadingState } = useContext(AppContext);
	const { closeFullscreen } = useContext(FullscreenContext);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		const setupAction = async () => {
			// Get the action type.
			let actionType: ActionType;
			if (isEditing) {
				const typ = getActionTypeFromConnection(connection, providedAction.Target, providedAction.EventType);
				if (typ == null) {
					console.error(
						`Action type with target ${providedAction.Target}${
							providedAction.EventType ? ' and event type ' + providedAction.EventType : ''
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
					actionType.Target,
					actionType.EventType,
				);

				inputSchema = schemas.In;
				outputSchema = schemas.Out;
				inputMatchingSchema = schemas.Matchings ? schemas.Matchings.Internal : null;
				outputMatchingSchema = schemas.Matchings ? schemas.Matchings.External : null;

				// If the action type is an import from a database source, the
				// input schema is the schema of the database table itself.
				if (fields.includes('Query') && isEditing) {
					let res: ExecQueryResponse;
					res = await api.workspaces.connections.query(connection.id, providedAction.Query!, 0);
					inputSchema = res.Schema;
				}

				// If the action type is an import from a file source,
				// the input schema is the schema of the file itself.
				if (fields.includes('File') && isEditing && isImport) {
					let res: RecordsResponse;
					res = await api.workspaces.connections.records(
						connection.id,
						providedAction.Connector,
						providedAction.Path!,
						providedAction.Sheet,
						providedAction.Compression,
						providedAction.Settings,
						0,
					);
					inputSchema = res.schema;
				}

				// If the action type is an export to a database
				// destination, the output schema is the schema of the
				// database table itself.
				if (fields.includes('Table') && isEditing) {
					let schema: ObjectType;
					schema = await api.workspaces.connections.tableSchema(connection.id, providedAction.Table);
					outputSchema = schema;
				}
			} catch (err) {
				if (err instanceof UnprocessableError) {
					let message: string;
					if (err.code === 'DatabaseFailed') {
						let errorMessage: string;
						if (err.cause && err.cause !== '') {
							errorMessage = err.cause;
						} else {
							errorMessage = err.message;
						}
						message = errorMessage;
					} else {
						message = err.message;
					}
					handleError(message);
					// continue execution so that user can fix the action.
				} else if (err instanceof NotFoundError) {
					setIsLoading(false);
					closeFullscreen();
					setIsLoadingState(true);
					// exit action route and reload the state.
					return;
				} else {
					setIsLoading(false);
					closeFullscreen();
					handleError(err);
					// something unexpected happened, user cannot fix the
					// action.
					return;
				}
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
			actionToSet = await transformInActionToSet(action, actionType, api, connection);
		} catch (err) {
			return err;
		}

		let id: number = 0;
		try {
			if (isEditing) {
				await api.workspaces.connections.setAction(connection.id, action.ID!, actionToSet);
			} else {
				id = await api.workspaces.connections.addAction(
					connection.id,
					actionType.Target,
					actionType.EventType,
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

	const isTransformationAllowed =
		connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server';

	let isMappingHidden = false;
	let isMappingDisabled = false;
	let mappingDisabledReason = '';

	if (!isLoading) {
		isMappingHidden =
			isFileConnectorLoading ||
			isFileConnectorChanged ||
			((connection.type === 'Database' || connection.type === 'Storage') &&
				actionType!.InputSchema == null &&
				!isEditing) ||
			(connection.type === 'Database' &&
				connection.role === 'Destination' &&
				actionType!.OutputSchema == null &&
				!isEditing);

		const hasQueryError =
			connection.type === 'Database' &&
			connection.role === 'Source' &&
			actionType!.InputSchema == null &&
			isEditing;
		const hasRecordsError = connection.type === 'Storage' && actionType!.InputSchema == null && isEditing;
		const hasTableError = connection.type === 'Database' && actionType!.OutputSchema == null && isEditing;

		isMappingDisabled =
			hasQueryError ||
			isQueryChanged ||
			hasRecordsError ||
			(isFileChanged && connection.role === 'Source') ||
			hasTableError ||
			isTableChanged;

		if (hasQueryError) {
			mappingDisabledReason =
				'Mapping is disabled since the query execution returned an error. Please fix the query before proceeding to mapping.';
		} else if (hasRecordsError) {
			mappingDisabledReason =
				'Mapping is disabled due to an error in the file information. Please fix the file information before proceeding to mapping.';
		} else if (hasTableError) {
			mappingDisabledReason = `Mapping is disabled because the provided table could not be retrieved. Please fix the table name before proceeding to mapping.`;
		} else if (connection.type === 'Database' && connection.role === 'Source') {
			mappingDisabledReason =
				'Mapping is disabled since the query has been modified. Please confirm the query or revert the changes before proceeding to mapping.';
		} else if (connection.type === 'Database' && connection.role === 'Destination') {
			mappingDisabledReason =
				'Mapping is disabled since the table name has been modified. Please confirm the table name or revert the changes before proceeding to mapping.';
		} else if (connection.type === 'Storage') {
			mappingDisabledReason =
				'Mapping is disabled since the file information has been modified. Please confirm the new file information or revert the changes before proceeding to mapping.';
		}
	}

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
		isSaveHidden,
		setIsSaveHidden,
		setIsFileChanged,
		isFileConnectorLoading,
		setIsFileConnectorLoading,
		setIsFileConnectorChanged,
		setIsTableChanged,
		setIsQueryChanged,
		isMappingHidden,
		isMappingDisabled,
		mappingDisabledReason,
	};
};

export { useAction };

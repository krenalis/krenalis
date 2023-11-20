import { useEffect, useState, useContext, useMemo } from 'react';
import {
	computeDefaultAction,
	computeActionTypeFields,
	TransformedActionType,
	TransformedAction,
	transformActionType,
	transformAction,
	transformInActionToSet,
} from '../lib/helpers/transformedAction';
import { AppContext } from '../context/providers/AppProvider';
import * as variants from '../constants/variants';
import * as icons from '../constants/icons';
import TransformedConnection, { getActionTypeFromConnection } from '../lib/helpers/transformedConnection';
import { UnprocessableError, NotFoundError } from '../lib/api/errors';
import { Action, ActionToSet, ActionType } from '../types/external/action';
import { ActionSchemasResponse, ExecQueryResponse, RecordsResponse } from '../types/external/api';
import { ObjectType } from '../types/external/types';
import Workspace from '../types/external/workspace';
import { sleep } from '../lib/utils/sleep';

const useActionData = (
	connection: TransformedConnection,
	providedActionType: ActionType,
	providedAction: Action,
	setIsSaveButtonLoading: React.Dispatch<React.SetStateAction<boolean>>,
	workspace: Workspace,
) => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [action, setAction] = useState<TransformedAction>();
	const [actionType, setActionType] = useState<TransformedActionType>();
	const [isSaveHidden, setIsSaveHidden] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isTableChanged, setIsTableChanged] = useState<boolean>(false);
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);

	const { api, showError, showStatus, redirect } = useContext(AppContext);

	const isEditing = providedAction != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		const fetchData = async () => {
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

			// Get the action type schemas.
			let schemas: ActionSchemasResponse;
			try {
				schemas = await api.workspaces.connections.actionSchemas(
					connection.id,
					actionType.Target,
					actionType.EventType,
				);
			} catch (err) {
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
						case 'EventTypeNotExist':
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
			let inputMatchingSchema = schemas.Matchings ? schemas.Matchings.Internal : null;
			let outputMatchingSchema = schemas.Matchings ? schemas.Matchings.External : null;

			// Compute which fields are supported by the action type.
			const fields = computeActionTypeFields(connection, actionType, schemas);

			// If the action type is an import from a database source, the input
			// schema is the schema of the database table itself.
			if (fields.includes('Query') && isEditing) {
				let res: ExecQueryResponse;
				try {
					res = await api.workspaces.connections.query(connection.id, providedAction.Query!, 0);
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
						if (err.code === 'DatabaseFailed') {
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
					res = await api.workspaces.connections.records(
						connection.id,
						providedAction.Path!,
						providedAction.Sheet,
						0,
					);
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

			// If the action type is an exrpot to a database destination, the
			// output schema is the schema of the database table itself.
			if (fields.includes('Table') && isEditing) {
				let schema: ObjectType;
				try {
					schema = await api.workspaces.connections.tableSchema(connection.id, providedAction.Table);
				} catch (err) {
					showError(err);
					return;
				}
				outputSchema = schema;
			}

			const transformedActionType = transformActionType(
				actionType,
				inputSchema,
				outputSchema,
				inputMatchingSchema,
				outputMatchingSchema,
				fields,
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
		fetchData();
	}, [providedActionType, providedAction]);

	const saveAction = async () => {
		if (action == null || actionType == null) {
			return 'Invalid action or action type';
		}

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(action, actionType, api, workspace.AnonymousIdentifiers);
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

	const isTransformationAllowed: boolean = useMemo(
		() => connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server',
		[connection, providedActionType, providedAction],
	);

	const { mustComputeSchema, isMappingSectionDisabled, disabledReason } = useMemo(() => {
		let mustComputeSchema = false;
		let isMappingSectionDisabled = false;
		let disabledReason = '';

		if (isLoading) {
			return { mustComputeSchema, isMappingSectionDisabled, disabledReason };
		}

		mustComputeSchema =
			((connection.type === 'Database' || connection.type === 'File') &&
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

		const hasRecordsError =
			connection.type === 'File' &&
			connection.role === 'Destination' &&
			actionType!.InputSchema == null &&
			isEditing;

		const hasTableError = connection.type === 'Database' && actionType!.OutputSchema == null && isEditing;

		isMappingSectionDisabled =
			hasQueryError ||
			isQueryChanged ||
			hasRecordsError ||
			(isFileChanged && isImport) ||
			hasTableError ||
			isTableChanged ||
			mustComputeSchema;

		disabledReason = '';
		if (hasQueryError) {
			disabledReason =
				'Mappings are disabled since the query returned an error. Fix the query before proceeding to mappings.';
		} else if (hasRecordsError) {
			disabledReason =
				'Mappings are disabled due to an error in the file fetch. Please resolve the file information issue before proceeding with the mappings.';
		} else if (hasTableError) {
			disabledReason = `Mappings are disabled because the table couldn't be retrieved. Please resolve this issue before proceeding with the mappings.`;
		} else if (connection.type === 'Database' && connection.role === 'Source') {
			disabledReason =
				'Mappings are disabled since the query has been modified. Please confirm the query or revert the changes before proceeding with mappings.';
		} else if (connection.type === 'Database' && connection.role === 'Destination') {
			disabledReason =
				'Mappings are disabled since the table name has been modified. Please confirm the table name or revert the changes before proceeding with mappings.';
		} else {
			disabledReason =
				'Mappings are disabled since the file information has been modified . Please confirm the new information or revert the changes before proceeding with mappings.';
		}

		return { mustComputeSchema, isMappingSectionDisabled, disabledReason };
	}, [isLoading, isQueryChanged, isFileChanged, isTableChanged, connection, actionType, isEditing, isImport]);

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
		setIsTableChanged,
		setIsQueryChanged,
		isMappingSectionDisabled,
		disabledReason,
		mustComputeSchema,
	};
};

export default useActionData;

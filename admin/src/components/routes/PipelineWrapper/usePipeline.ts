import { useEffect, useState, useContext, useMemo } from 'react';
import {
	computeDefaultPipeline,
	computePipelineTypeFields,
	TransformedPipelineType,
	TransformedPipeline,
	transformPipelineType,
	transformPipeline,
	transformInPipelineToSet,
	flattenSchema,
} from '../../../lib/core/pipeline';
import AppContext from '../../../context/AppContext';
import TransformedConnection, { getPipelineTypeFromConnection } from '../../../lib/core/connection';
import { UnavailableError, UnprocessableError } from '../../../lib/api/errors';
import { Pipeline, PipelineToSet, PipelineType } from '../../../lib/api/types/pipeline';
import {
	PipelineSchemasResponse,
	ExecQueryResponse,
	RecordsResponse,
	ConnectorSettings,
} from '../../../lib/api/types/responses';
import { ObjectType } from '../../../lib/api/types/types';
import { FullscreenContext } from '../../../context/FullscreenContext';

const usePipeline = (
	connection: TransformedConnection,
	providedPipelineType: PipelineType,
	providedPipeline: Pipeline,
) => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [pipeline, setPipeline] = useState<TransformedPipeline>();
	const [settings, setSettings] = useState<ConnectorSettings>();
	const [pipelineType, setPipelineType] = useState<TransformedPipelineType>();
	const [transformationType, setTransformationType] = useState<'mappings' | 'function' | ''>('');
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isFileConnectorLoading, setIsFileConnectorLoading] = useState<boolean>(
		providedPipeline !== null && connection.isFile && connection.isSource,
	);
	const [isFileConnectorChanged, setIsFileConnectorChanged] = useState<boolean>(false);
	const [isTableChanged, setIsTableChanged] = useState<boolean>(false);
	const [selectedInPaths, setSelectedInPaths] = useState<string[]>([]);
	const [selectedOutPaths, setSelectedOutPaths] = useState<string[]>([]);
	const [issues, setIssues] = useState<string[]>([]);
	const [showIssues, setShowIssues] = useState<boolean>(true);
	const [autoSelectedPaths, setAutoSelectedPaths] = useState<string[]>([]);

	const { api, handleError, redirect, connectors } = useContext(AppContext);
	const { closeFullscreen } = useContext(FullscreenContext);

	const isEditing = providedPipeline != null;
	const isImport = connection.role === 'Source';

	useEffect(() => {
		// Filter out the selected properties that are no longer in the
		// schemas, when they change.
		if (isLoading) {
			return;
		}
		if (pipelineType.inputSchema) {
			const flatIn = flattenSchema(pipelineType.inputSchema);
			const inPaths = [];
			for (const p of selectedInPaths) {
				if (flatIn[p]) {
					inPaths.push(p);
				}
			}
			setSelectedInPaths(inPaths);
		}
		if (pipelineType.outputSchema) {
			const flatOut = flattenSchema(pipelineType.outputSchema);
			const outPaths = [];
			for (const p of selectedOutPaths) {
				if (flatOut[p]) {
					outPaths.push(p);
				}
			}
			setSelectedOutPaths(outPaths);
		}
	}, [pipelineType?.inputSchema, pipelineType?.outputSchema]);

	useEffect(() => {
		if (isLoading || pipelineType.outputSchema == null) {
			return;
		}
		computeAutoSelectedPaths('', !isEditing);
	}, [pipelineType?.outputSchema]);

	const computeAutoSelectedPaths = (prefix: string, updateCheckboxes?: boolean) => {
		const flatOut = flattenSchema(pipelineType.outputSchema);
		let paths = Object.keys(flatOut);
		if (prefix !== '') {
			paths = paths.filter((pa) => pa === prefix || pa.startsWith(`${prefix}.`));
		}

		const isEventSend = connection.isDestination && pipelineType.target.includes('Event');
		if (isEventSend) {
			// Automatically compute the selected out paths based on the
			// required properties. These properties will also be pre-populated
			// in the function editor.
			let selected = [];
			for (const path of paths) {
				const p = flatOut[path];
				let isAutoSelected = false;
				if (p.createRequired) {
					const firstPathFragment = path.split('.')[0];
					const hasRequiredFirstLevelParent =
						paths.findIndex((pa) => pa === firstPathFragment && flatOut[pa].createRequired) !== -1;
					const hasRequiredChild =
						paths.findIndex((pa) => pa.startsWith(`${path}.`) && flatOut[pa].createRequired) !== -1;
					const isFirstLevel = !path.includes('.');
					isAutoSelected = !hasRequiredChild && (isFirstLevel || hasRequiredFirstLevelParent);
				}
				if (isAutoSelected) {
					selected.push(path);
				}
			}

			if (prefix !== '') {
				const filtered = [...selectedOutPaths].filter((pa) => pa !== prefix && !pa.startsWith(`${prefix}.`));
				const merged = [...filtered, ...selected];
				selected = merged;
			}

			setAutoSelectedPaths(selected);
			if (updateCheckboxes) {
				setSelectedOutPaths(selected);
			}
		}
	};

	useEffect(() => {
		const handleException = (err: Error | string) => {
			setTimeout(() => {
				setIsLoading(false);
				closeFullscreen();
				redirect(`connections/${connection.id}/pipelines`);
				handleError(err);
			}, 300);
		};

		const setupPipeline = async () => {
			// Get the pipeline type.
			let pipelineType: PipelineType;
			if (isEditing) {
				const typ = getPipelineTypeFromConnection(
					connection,
					providedPipeline.target,
					providedPipeline.eventType,
				);
				if (typ == null) {
					console.error(
						`Pipeline type with target ${providedPipeline.target}${
							providedPipeline.eventType ? ' and event type ' + providedPipeline.eventType : ''
						} does not exists anymore`,
					);
					return;
				} else {
					pipelineType = typ;
				}
			} else {
				pipelineType = { ...providedPipelineType };
			}

			// Fetch the pipeline schemas.
			let inputSchema: ObjectType;
			let outputSchema: ObjectType;
			let inputMatchingSchema: ObjectType;
			let outputMatchingSchema: ObjectType;
			try {
				let schemas: PipelineSchemasResponse;
				schemas = await api.workspaces.connections.pipelineSchemas(
					connection.id,
					pipelineType.target,
					pipelineType.eventType,
				);

				inputSchema = schemas.in;
				outputSchema = schemas.out;
				inputMatchingSchema = schemas.matchings ? schemas.matchings.internal : null;
				outputMatchingSchema = schemas.matchings ? schemas.matchings.external : null;
			} catch (err) {
				handleException(err);
				return;
			}

			// Compute which fields are supported by the pipeline type.
			const fields = computePipelineTypeFields(connection, pipelineType);

			try {
				// Handle cases that requires additional steps to
				// retrieve the schemas.

				// If the pipeline type is an import from a database
				// source, the input schema is the schema of the
				// database table itself.
				if (fields.includes('Query') && isEditing) {
					let res: ExecQueryResponse;
					try {
						res = await api.workspaces.connections.execQuery(connection.id, providedPipeline.query!, 0);
						inputSchema = res.schema;
						setIssues(res.issues);
					} catch (err) {
						if (
							err instanceof UnavailableError ||
							(err instanceof UnprocessableError &&
								(err.code === 'InvalidPlaceholder' || err.code === 'UnsupportedColumnType'))
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the pipeline (or at least can see its state
							// in order to debug the problem).
						} else {
							throw err;
						}
					}
				}

				// If the pipeline type is an import from a file source,
				// the input schema is the schema of the file itself.
				if (fields.includes('File') && isEditing && isImport) {
					let s: ConnectorSettings | null = null;
					const connector = connectors.find((c) => c.code === providedPipeline.format);
					if (connector.hasSettings(connection.role)) {
						// get the settings of the file.
						let ui = await api.workspaces.connections.pipelineUiEvent(providedPipeline.id, 'load', null);
						s = ui.settings;
						setSettings(ui.settings);
					}
					let res: RecordsResponse;
					try {
						res = await api.workspaces.connections.records(
							connection.id,
							providedPipeline.path!,
							providedPipeline.format,
							providedPipeline.sheet,
							providedPipeline.compression,
							s,
							0,
						);
						inputSchema = res.schema;
						setIssues(res.issues);
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
							// the pipeline (or at least can see its state
							// in order to debug the problem).
						} else {
							throw err;
						}
					}
				}

				// If the pipeline type is an export to a database
				// destination, the output schema is the schema of the
				// database table itself.
				if (fields.includes('TableName') && isEditing) {
					try {
						const res = await api.workspaces.connections.tableSchema(
							connection.id,
							providedPipeline.tableName,
						);
						outputSchema = res.schema;
						setIssues(res.issues);
					} catch (err) {
						if (
							err instanceof UnavailableError ||
							(err instanceof UnprocessableError && err.code === 'UnsupportedColumnType')
						) {
							handleError(err.message);
							// continue execution so that user can fix
							// the pipeline (or at least can see its state
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

			const transformedPipelineType = transformPipelineType(
				pipelineType,
				fields,
				inputSchema,
				outputSchema,
				inputMatchingSchema,
				outputMatchingSchema,
			);
			setPipelineType(transformedPipelineType);

			let transformedPipeline: TransformedPipeline;
			if (isEditing) {
				transformedPipeline = transformPipeline(
					providedPipeline,
					outputSchema,
					fields.includes('Transformation'),
				);
				if (transformedPipeline.transformation.function != null) {
					// Set the initial value of the selected properties
					// of the function.
					const func = transformedPipeline.transformation.function;
					setSelectedInPaths(func.inPaths);
					setSelectedOutPaths(func.outPaths);
				}
			} else {
				transformedPipeline = computeDefaultPipeline(pipelineType, connection, outputSchema, fields);
			}
			setPipeline(transformedPipeline);
			setIsLoading(false);
		};
		setupPipeline();
	}, [providedPipelineType, providedPipeline]);

	const savePipeline = async () => {
		if (pipeline == null || pipelineType == null) {
			return 'Invalid pipeline or pipeline type';
		}

		let pipelineToSet: PipelineToSet;
		try {
			pipelineToSet = await transformInPipelineToSet(
				pipeline,
				settings,
				pipelineType,
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
				await api.workspaces.connections.updatePipeline(pipeline.id!, pipelineToSet);
			} else {
				id = await api.workspaces.connections.createPipeline(
					connection.id,
					pipelineType.target,
					pipelineType.eventType,
					pipelineToSet,
				);
			}
		} catch (err) {
			return err;
		}

		sessionStorage.setItem('newPipelineID', String(id));
		return null;
	};

	const { isTransformationHidden, isTransformationDisabled } = useMemo(() => {
		if (isLoading) return { isTransformationHidden: false, isTransformationDisabled: false };
		let isTransformationHidden: boolean = false;
		let isTransformationDisabled: boolean = false;

		const inputSchemaIsNotDefined = pipelineType.inputSchema == null;
		const outputSchemaIsNotDefined = pipelineType.outputSchema == null;

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
		pipelineType,
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
		pipeline,
		settings,
		setSettings,
		isLoading,
		pipelineType,
		setPipelineType,
		transformationType,
		setTransformationType,
		setPipeline,
		savePipeline,
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
		autoSelectedPaths,
		computeAutoSelectedPaths,
		issues,
		setIssues,
		showIssues,
		setShowIssues,
	};
};

export { usePipeline };

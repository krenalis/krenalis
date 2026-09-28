import { useEffect, useState, useContext, useMemo, useRef } from 'react';
import {
	computeDefaultPipeline,
	computePipelineTypeFields,
	TransformedPipelineType,
	TransformedPipeline,
	transformPipelineType,
	transformPipeline,
	transformInPipelineToSet,
	flattenSchema,
	checkFunctionPath,
	getConsentPropertyPaths,
	validateRequiredConsents,
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
import { ConsentPurpose } from '../../../lib/api/types/workspace';
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
	const [consentPurposes, setConsentPurposes] = useState<ConsentPurpose[] | null>(null);

	const consentPurposesRequestRef = useRef<number>(0);
	const consentPurposeNamesRef = useRef(new Map<string, string>());

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

	// refreshConsentPurposes reads the consent purposes again and returns them.
	// It returns null if they cannot be read, or if they have been requested
	// again in the meantime.
	const refreshConsentPurposes = async (): Promise<ConsentPurpose[] | null> => {
		const request = ++consentPurposesRequestRef.current;
		let purposes: ConsentPurpose[];
		try {
			const res = await api.workspaces.consentPurposes();
			purposes = res.purposes;
		} catch (err) {
			handleError(err);
			return null;
		}
		// Discard the purposes of an earlier request.
		if (request !== consentPurposesRequestRef.current) {
			return null;
		}
		for (const p of purposes) {
			consentPurposeNamesRef.current.set(p.id, p.name);
		}
		setConsentPurposes(purposes);
		return purposes;
	};

	useEffect(() => {
		if (pipelineType == null || !pipelineType.fields.includes('Consents')) {
			return;
		}
		refreshConsentPurposes();
		// Reload the purposes when the user comes back from configuring them in
		// another tab.
		const onFocus = () => {
			refreshConsentPurposes();
		};
		window.addEventListener('focus', onFocus);
		return () => {
			window.removeEventListener('focus', onFocus);
		};
	}, [pipelineType?.fields]);

	useEffect(() => {
		if (isLoading || pipelineType.outputSchema == null || isEditing) {
			return;
		}
		computeAutoSelectedPaths();
	}, [pipelineType?.outputSchema]);

	const computeAutoSelectedPaths = () => {
		// Compute the automatically selected out paths. These are required
		// paths that must be used in the transformation function. They are
		// flagged automatically, and pre-populated with zero values in the
		// transformation function.
		const flatOut = flattenSchema(pipelineType.outputSchema);
		let paths = Object.keys(flatOut);

		const isEventSend = connection.isDestination && pipelineType.target.includes('Event');
		if (isEventSend) {
			let autoSelected = [];
			for (const path of paths) {
				const { isRequired } = checkFunctionPath(path, pipeline, pipelineType, 'output', selectedOutPaths);
				const hasRequiredChild =
					paths.findIndex((pa) => {
						const isChild = pa.startsWith(`${path}.`);
						if (!isChild) {
							return false;
						}
						const { isRequired } = checkFunctionPath(
							pa,
							pipeline,
							pipelineType,
							'output',
							selectedOutPaths,
						);
						return isRequired;
					}) !== -1;
				if (isRequired && !hasRequiredChild) {
					// The path is automatically selected.
					const property = flatOut[path];
					const isObject = property.type === 'object';
					if (isObject) {
						// Auto select all children leafs.
						const childrenLeafs = paths.filter((p) => {
							const isChildren = p.startsWith(`${path}.`);
							const isLeaf = flatOut[p].type !== 'object';
							return isChildren && isLeaf;
						});
						for (const c of childrenLeafs) {
							autoSelected.push(c);
						}
					} else {
						// Auto select the property.
						autoSelected.push(path);
					}
				}
			}
			setSelectedOutPaths(autoSelected);
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
			if (consentPurposes != null) {
				validateRequiredConsents(pipeline, pipelineType, connection, consentPurposes, selectedOutPaths);
			}
		} catch (err) {
			return err;
		}

		let id: string = '';
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
			const isConsentsErr =
				err instanceof UnprocessableError &&
				(err.code === 'ConsentPurposeNotExist' || err.code === 'ConsentPurposeLocationNotSet');
			if (isConsentsErr) {
				// A required purpose has been deleted, or its consent location
				// has been removed, after the purposes have been read. Read
				// them again to show which one.
				const purposes = await refreshConsentPurposes();
				if (purposes != null) {
					try {
						validateRequiredConsents(pipeline, pipelineType, connection, purposes, selectedOutPaths);
					} catch (consentsErr) {
						return consentsErr;
					}
				}
			}
			return err;
		}

		sessionStorage.setItem('newPipelineID', id);
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

	const consentPropertyPaths = useMemo(() => {
		if (pipeline == null || pipelineType == null) {
			return new Map<string, ConsentPurpose[]>();
		}
		return getConsentPropertyPaths(pipeline, pipelineType, connection, consentPurposes ?? []);
	}, [pipeline, pipelineType, connection, consentPurposes]);

	// deletedConsentPurposes are the required purposes deleted after they have
	// been read, kept so that the user can see and remove them.
	const deletedConsentPurposes = useMemo(() => {
		const deleted: ConsentPurpose[] = [];
		if (consentPurposes == null || pipeline?.requiredConsents == null) {
			return deleted;
		}
		for (const id of pipeline.requiredConsents.purposes) {
			if (consentPurposes.some((p) => p.id === id)) {
				continue;
			}
			deleted.push({
				id: id,
				name: consentPurposeNamesRef.current.get(id) ?? id,
				eventConsentLocations: [],
				profileConsentLocation: null,
			});
		}
		return deleted;
	}, [consentPurposes, pipeline?.requiredConsents]);

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
		computeAutoSelectedPaths,
		issues,
		setIssues,
		showIssues,
		setShowIssues,
		consentPurposes: consentPurposes ?? [],
		deletedConsentPurposes,
		consentPropertyPaths,
	};
};

export { usePipeline };

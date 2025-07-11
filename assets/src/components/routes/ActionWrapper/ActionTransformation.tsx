import React, { useState, useRef, useContext, useEffect, forwardRef, useMemo, ReactNode } from 'react';
import {
	checkIfPropertyExists,
	updateMappingProperty,
	getSampleIdentifiers,
	updateMappingPropertyError,
} from './Action.helpers';
import {
	getSchemaComboboxItems,
	getIdentityColumnComboboxItems,
	getLastChangeTimeComboboxItems,
} from '../../helpers/getSchemaComboboxItems';
import {
	TransformedAction,
	TransformedActionType,
	TransformedMapping,
	TransformedProperty,
	doesLastChangeTimeColumnNeedFormat,
	flattenSchema,
	getTransformationFunctionParameterName,
	isRecursiveType,
	parseMapString,
	propertyTypesAreEqual,
	stringifyMapPairs,
	transformInActionToSet,
} from '../../../lib/core/action';
import { RAW_TRANSFORMATION_FUNCTIONS } from './Action.constants';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import Section from '../../base/Section/Section';
import EditorWrapper from '../../base/EditorWrapper/EditorWrapper';
import Accordion from '../../base/Accordion/Accordion';
import useEventListener from '../../../hooks/useEventListener';
import AppContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';
import SlButtonGroup from '@shoelace-style/shoelace/dist/react/button-group/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import SlSplitPanel from '@shoelace-style/shoelace/dist/react/split-panel/index.js';
import SlAlert from '@shoelace-style/shoelace/dist/react/alert/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SyntaxHighlight from '../../base/SyntaxHighlight/SyntaxHighlight';
import SlRelativeTime from '@shoelace-style/shoelace/dist/react/relative-time/index.js';
import {
	AppUsersResponse,
	ExecQueryResponse,
	FindUsersResponse,
	PreviewSendEventResponse,
	RecordsResponse,
	TransformationLanguagesResponse,
	TransformDataResponse,
} from '../../../lib/api/types/responses';
import getLanguageLogo from '../../helpers/getLanguageLogo';
import Type, { ArrayType, MapType, ObjectType, Property, TextType } from '../../../lib/api/types/types';
import { EventListenerEvent } from '../../../hooks/useEventListener';
import { Sample } from './Action.types';
import { UnprocessableError } from '../../../lib/api/errors';
import ConnectionContext from '../../../context/ConnectionContext';
import Workspace from '../../../lib/api/types/workspace';
import { ActionToSet, ExportMode, TransformationFunction, TransformationPurpose } from '../../../lib/api/types/action';
import TransformedConnector from '../../../lib/core/connector';
import { Combobox } from '../../base/Combobox/Combobox';
import { ComboboxItem } from '../../base/Combobox/Combobox.types';
import JSONbig from 'json-bigint';
import actionContext from '../../../context/ActionContext';
import TransformedConnection from '../../../lib/core/connection';
import appContext from '../../../context/AppContext';

const lastChangeTimeFormats = {
	iso8601: 'ISO8601',
	excel: 'Excel',
};

const ActionTransformation = forwardRef<any>((_, ref) => {
	const [transformationLanguages, setTransformationLanguages] = useState<string[]>();
	const [selectedLanguage, setSelectedLanguage] = useState<string>('');
	const [isFullscreenTransformationOpen, setIsFullscreenTransformationOpen] = useState<boolean>(false);
	const [isCustomLastChangeTimeFormatSelected, setIsCustomLastChangeTimeFormatSelected] = useState<boolean>(false);

	const { api, handleError, workspaces, selectedWorkspace, connectors } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);
	const {
		isTransformationDisabled,
		action,
		setAction,
		actionType,
		transformationType,
		setTransformationType,
		setIsSaveHidden,
		isFormatChanged,
		isEditing,
		handleEmptyMatchingError,
	} = useContext(ActionContext);

	const isFirstCompilation = useRef(true);
	const lastChangeTimeFormatRef = useRef(null);
	const lastChangeTimeCustomFormatInputRef = useRef(null);

	const hasIdentityColumns = useMemo(() => {
		return (
			connection.isSource && (connection.isDatabase || connection.isFileStorage) && actionType.target === 'User'
		);
	}, [connection, actionType]);

	useEffect(() => {
		// when a new file is confirmed the UI should behave as if it is
		// the first the user is compiling the action's transformation.
		if (!isFormatChanged) {
			isFirstCompilation.current = true;
		}
	}, [isFormatChanged]);

	useEffect(() => {
		const fetchTransformationLanguages = async () => {
			let response: TransformationLanguagesResponse;
			try {
				response = await api.transformationLanguages();
			} catch (err) {
				handleError(err);
				return;
			}
			const languages = response.languages;
			setTransformationLanguages(languages);
		};
		fetchTransformationLanguages();
	}, []);

	useEffect(() => {
		if (action.transformation.function != null) {
			setTransformationType('function');
			setSelectedLanguage(action.transformation.function.language);
		} else {
			setTransformationType('mappings');
		}
	}, []);

	useEffect(() => {
		if (!hasIdentityColumns || !action.lastChangeTimeColumn) {
			return;
		}
		// check if the last change time format is custom.
		const formats = Object.values(lastChangeTimeFormats);
		if (!formats.includes(action.lastChangeTimeFormat)) {
			setIsCustomLastChangeTimeFormatSelected(true);
		}
	}, []);

	useEffect(() => {
		if (hasIdentityColumns && isFirstCompilation.current && !isEditing) {
			// precompile the 'IdentityColumn' and 'lastChangeTimeColumn'
			// fields, if possible.
			const a = { ...action };
			const hasIdColumn = actionType.inputSchema.properties.findIndex((prop) => prop.name === 'id') !== -1;
			if (hasIdColumn) {
				a.identityColumn = 'id';
				isFirstCompilation.current = false;
			}
			const hasLastChangeTimeColumn =
				actionType.inputSchema.properties.findIndex((prop) => prop.name === 'timestamp') !== -1;
			if (hasLastChangeTimeColumn) {
				a.lastChangeTimeColumn = 'timestamp';
				if (doesLastChangeTimeColumnNeedFormat(a.lastChangeTimeColumn, actionType.inputSchema)) {
					a.lastChangeTimeFormat = '';
				}
			}
			setAction(a);
		}
	}, [isFirstCompilation.current]);

	useEffect(() => {
		const body = document.querySelector('.fullscreen') as HTMLDivElement;
		if (isFullscreenTransformationOpen) {
			// hide the fullScreen scrollbar.
			body.style.overflow = 'hidden';
			setIsSaveHidden(true);
		} else {
			body.style.overflow = 'auto';
			setIsSaveHidden(false);
		}
	}, [isFullscreenTransformationOpen]);

	useEffect(() => {
		if (selectedLanguage == '') {
			return;
		}
		const a = { ...action };
		const isTransformationUndefined = a.transformation.function == null;
		const isLanguageChanged = !isTransformationUndefined && a.transformation.function.language !== selectedLanguage;
		if (isTransformationUndefined || isLanguageChanged) {
			a.transformation.function = {
				source: RAW_TRANSFORMATION_FUNCTIONS[selectedLanguage].replace(
					'$parameterName',
					getTransformationFunctionParameterName(connection, actionType),
				),
				language: selectedLanguage,
				preserveJSON: false,
				inPaths: [],
				outPaths: [],
			};
			setAction(a);
		}
	}, [selectedLanguage]);

	const flatInputSchema = useMemo<TransformedMapping>(() => {
		return flattenSchema(actionType.inputSchema);
	}, [actionType.inputSchema]);

	useEffect(() => {
		// validate mapping expressions when the action is opened and
		// revalidate them when the schema changes.
		const validateExpressions = async () => {
			let a = action;
			const keys = Object.keys(action.transformation.mapping);
			for (const k of keys) {
				const property = action.transformation.mapping[k];
				const value = property.value;
				if (value === '') {
					continue;
				}
				let errorMessage = '';
				try {
					errorMessage = await api.validateExpression(
						value,
						actionType.inputSchema.properties,
						property.full.type,
					);
				} catch (err) {
					handleError(err);
					return;
				}
				if (errorMessage !== '') {
					a = updateMappingProperty(a, k, value, errorMessage);
				}
			}
			setAction(a);
		};
		if (flatInputSchema != null && transformationType === 'mappings') {
			validateExpressions();
		}
	}, [flatInputSchema, transformationType]);

	const needFormat: boolean = useMemo(() => {
		if (
			(connection.isFileStorage || connection.isDatabase) &&
			action.lastChangeTimeColumn &&
			!isTransformationDisabled
		) {
			return doesLastChangeTimeColumnNeedFormat(action.lastChangeTimeColumn, actionType.inputSchema);
		}
		return false;
	}, [action, actionType, isTransformationDisabled]);

	const format: TransformedConnector | null = useMemo(() => {
		if (action.format) {
			return connectors.find((c) => c.name === action.format);
		}
		return null;
	}, [action]);

	const identityColumnError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			if (action.identityColumn === '' && !isFirstCompilation.current) {
				return 'The user identifier cannot be empty';
			}
			return checkIfPropertyExists(action.identityColumn, flatInputSchema);
		}
	}, [action, flatInputSchema]);

	const lastChangeTimeColumnError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			return checkIfPropertyExists(action.lastChangeTimeColumn, flatInputSchema);
		}
	}, [action, flatInputSchema]);

	const { identityColumnList, lastChangeTimeList, mappingList } = useMemo(() => {
		return {
			identityColumnList: getIdentityColumnComboboxItems(actionType.inputSchema),
			lastChangeTimeList: getLastChangeTimeComboboxItems(actionType.inputSchema),
			mappingList: getSchemaComboboxItems(actionType.inputSchema),
		};
	}, [actionType]);

	const onChangeTransformationFunction = (source: string) => {
		const a = { ...action };
		a.transformation.function = {
			source: source,
			language: selectedLanguage,
			preserveJSON: a.transformation.function.preserveJSON,
			inPaths: [],
			outPaths: [],
		};
		setAction(a);
	};

	const updateMapping = (path: string, value: string) => {
		const updatedAction = updateMappingProperty(action, path, value, '');
		setAction(updatedAction);
	};

	const updateMappingError = (path: string, errorMessage: string) => {
		const updatedAction = updateMappingPropertyError(action, path, errorMessage);
		setAction(updatedAction);
	};

	const onSelectProperty = (path: string, value: string) => {
		if (path === 'identityColumn') {
			const a = { ...action };
			a.identityColumn = value;
			setAction(a);
			if (isFirstCompilation.current) {
				isFirstCompilation.current = false;
			}
			return;
		} else if (path === 'lastChangeTimeColumn') {
			const a = { ...action };
			a.lastChangeTimeColumn = value;
			if (value === '' || !doesLastChangeTimeColumnNeedFormat(value, actionType.inputSchema)) {
				a.lastChangeTimeFormat = '';
			}
			setAction(a);
			return;
		}
		updateMapping(path, value);
	};

	const onUpdateIdentityColumn = (_: string, value: string) => {
		const a = { ...action };
		a.identityColumn = value;
		setAction(a);
		if (isFirstCompilation.current) {
			isFirstCompilation.current = false;
		}
	};

	const onUpdateLastChangeTimeColumn = (_: string, value: string) => {
		const a = { ...action };
		const oldValue = a.lastChangeTimeColumn;
		a.lastChangeTimeColumn = value;
		const needFormat = doesLastChangeTimeColumnNeedFormat(value, actionType.inputSchema);
		if (value === '' || !needFormat) {
			setIsCustomLastChangeTimeFormatSelected(false);
			a.lastChangeTimeFormat = '';
		} else {
			const neededFormat = doesLastChangeTimeColumnNeedFormat(oldValue, actionType.inputSchema);
			if (!neededFormat) {
				setTimeout(() => {
					lastChangeTimeFormatRef.current.show();
				}, 50);
			}
		}
		setAction(a);
	};

	const onChangeLastChangeTimeFormat = (e) => {
		const a = { ...action };
		const v = e.target.value;
		if (v === 'custom') {
			setIsCustomLastChangeTimeFormatSelected(true);
			a.lastChangeTimeFormat = '';
			setTimeout(() => {
				if (lastChangeTimeCustomFormatInputRef.current) {
					lastChangeTimeCustomFormatInputRef.current.focus();
				}
			}, 50);
		} else {
			setIsCustomLastChangeTimeFormatSelected(false);
			a.lastChangeTimeFormat = lastChangeTimeFormats[e.target.value];
		}
		setAction(a);
	};

	const onInputLastChangeTimeCustomFormat = (e) => {
		const a = { ...action };
		a.lastChangeTimeFormat = e.target.value;
		setAction(a);
	};

	const onChangeIncremental = () => {
		const a = { ...action };
		a.incremental = !a.incremental;
		setAction(a);
	};

	const onOpenFullscreenTransformation = () => {
		if (actionType.fields.includes('Matching')) {
			// If the matching properties are not defined, prevent the opening
			// of testing mode and show an error. Displaying the same error
			// during action testing in testing mode would be less clear.
			const inMatching = action.matching.in;
			const outMatching = action.matching.out;
			if (inMatching === '' || outMatching === '') {
				handleEmptyMatchingError();
				return;
			}
		}
		setIsFullscreenTransformationOpen(true);
	};

	const onCloseFullscreenTransformation = () => {
		setIsFullscreenTransformationOpen(false);
	};

	if (transformationLanguages == null) {
		return null;
	}

	const box = (
		<TransformationBox
			transformationType={transformationType}
			setTransformationType={setTransformationType}
			workspaces={workspaces}
			selectedWorkspace={selectedWorkspace}
			action={action}
			setAction={setAction}
			updateMapping={updateMapping}
			updateMappingError={updateMappingError}
			mappingItems={mappingList}
			onSelectMappingItem={onSelectProperty}
			isTransformationDisabled={isTransformationDisabled}
			transformationLanguages={transformationLanguages}
			selectedLanguage={selectedLanguage}
			setSelectedLanguage={setSelectedLanguage}
			onOpenFullscreenTransformation={onOpenFullscreenTransformation}
			onChangeTransformationFunction={onChangeTransformationFunction}
			isFullscreenTransformationOpen={isFullscreenTransformationOpen}
			onCloseFullscreenTransformation={onCloseFullscreenTransformation}
			actionType={actionType}
			hasSchema={actionType.outputSchema != null}
			flatInputSchema={flatInputSchema}
		/>
	);

	return (
		<div
			className={`action__transformation${isTransformationDisabled ? ' action__transformation--disabled' : ''}`}
			ref={ref}
		>
			{hasIdentityColumns ? (
				<Section
					title='Identity columns'
					description='The columns from which to import the value to uniquely identify a user identity, and possibly the time of its last modification.'
					padded={true}
					annotated={true}
				>
					<div className='action__transformation-identity-columns'>
						<div className='action__transformation-identity-column'>
							<Combobox
								onInput={onUpdateIdentityColumn}
								onSelect={onUpdateIdentityColumn}
								name='identityColumn'
								value={identityColumnList.length === 0 ? '' : action.identityColumn!}
								disabled={isTransformationDisabled || identityColumnList.length === 0}
								className='action__transformation-input-property'
								isExpression={false}
								items={identityColumnList}
								label='Identity'
								required
								caret={true}
								clearable={action.identityColumn?.length > 0}
								error={
									identityColumnList.length === 0
										? `No column ${
												connection.isFileStorage ? 'in the file' : 'returned by the query'
											} can be used as user identifier`
										: identityColumnError
								}
								size='small'
								helpText='A column that uniquely identifies a user identity'
							/>
						</div>
						<div className='action__transformation-last-change-time-column'>
							<div className='action__transformation-last-change-time'>
								<Combobox
									onInput={onUpdateLastChangeTimeColumn}
									onSelect={onUpdateLastChangeTimeColumn}
									value={action.lastChangeTimeColumn!}
									name='lastChangeTimeColumn'
									disabled={isTransformationDisabled}
									className='action__transformation-input-property'
									isExpression={false}
									label='Last change time'
									caret={true}
									items={lastChangeTimeList}
									clearable={action.lastChangeTimeColumn?.length > 0}
									error={lastChangeTimeColumnError}
									size='small'
									helpText='A column with the time of the last modification of a user identity'
								/>
							</div>
							{needFormat && (
								<div className='action__transformation-last-change-format-property'>
									<div className='action__transformation-last-change-format'>
										<SlSelect
											onSlChange={onChangeLastChangeTimeFormat}
											value={
												isCustomLastChangeTimeFormatSelected
													? 'custom'
													: action.lastChangeTimeColumn
														? Object.keys(lastChangeTimeFormats).find(
																(key) =>
																	lastChangeTimeFormats[key] ===
																	action.lastChangeTimeFormat,
															)
														: ''
											}
											name='lastChangeTimeFormat'
											label='Format'
											size='small'
											ref={lastChangeTimeFormatRef}
										>
											<SlOption value='iso8601'>ISO 8601</SlOption>
											{format?.name === 'Excel' && <SlOption value='excel'>Excel</SlOption>}
											<SlOption value='custom'>Custom...</SlOption>
										</SlSelect>
									</div>
									{isCustomLastChangeTimeFormatSelected && (
										<div className='action__transformation-last-change-custom-format'>
											<SlInput
												onSlInput={onInputLastChangeTimeCustomFormat}
												value={action.lastChangeTimeFormat}
												name='lastChangeTimeCustomFormat'
												placeholder='%Y-%m-%d'
												helpText='C89 "strftime" format string'
												size='small'
												ref={lastChangeTimeCustomFormatInputRef}
											></SlInput>
										</div>
									)}
								</div>
							)}
						</div>
						{actionType.fields.includes('Incremental') && (
							<div className='action__transformation-incremental'>
								<SlCheckbox
									checked={action.incremental}
									onSlChange={onChangeIncremental}
									disabled={action.lastChangeTimeColumn === ''}
									helpText='Only imports users whose last change time is subsequent to the last import'
								>
									Run incremental import
								</SlCheckbox>
							</div>
						)}
					</div>
				</Section>
			) : (
				actionType.fields.includes('Incremental') && (
					<Section
						title='Incremental import'
						description='Only imports users that have been updated since the last import'
						padded={true}
						annotated={true}
					>
						<SlCheckbox checked={action.incremental} onSlChange={onChangeIncremental}>
							Run incremental import
						</SlCheckbox>
					</Section>
				)
			)}
			<Section
				title='Transformation'
				description='The relation between the event properties and the action type properties'
				padded={false}
				annotated={true}
				className='action__transformation-section'
			>
				{box}
				<FullscreenTransformation
					isFullscreenTransformationOpen={isFullscreenTransformationOpen}
					selectedLanguage={selectedLanguage}
					body={box}
					flatInputSchema={flatInputSchema}
					inputSchema={actionType.inputSchema}
					outputSchema={actionType.outputSchema}
				/>
			</Section>
		</div>
	);
});

interface TransformationBoxProps {
	transformationType: 'mappings' | 'function' | '';
	setTransformationType: React.Dispatch<React.SetStateAction<'mappings' | 'function' | ''>>;
	workspaces: Workspace[];
	selectedWorkspace: number;
	action: TransformedAction;
	setAction: React.Dispatch<React.SetStateAction<TransformedAction>>;
	updateMapping: (path: string, value: string) => void;
	updateMappingError: (path: string, errorMessage: string) => void;
	mappingItems: ComboboxItem[];
	onSelectMappingItem: (path: string, value: string) => void;
	isTransformationDisabled: boolean;
	transformationLanguages: string[];
	selectedLanguage: string;
	setSelectedLanguage: React.Dispatch<React.SetStateAction<string>>;
	onOpenFullscreenTransformation: () => void;
	onChangeTransformationFunction: (source: string) => void;
	isFullscreenTransformationOpen: boolean;
	onCloseFullscreenTransformation: () => void;
	actionType: TransformedActionType;
	hasSchema: boolean;
	flatInputSchema: TransformedMapping;
}

const isMappingChanged = (oldMapping: TransformedMapping, newMapping: TransformedMapping): boolean => {
	let isChanged = false;
	for (const key in oldMapping) {
		if (oldMapping[key].value !== newMapping[key].value) {
			isChanged = true;
			break;
		}
	}
	return isChanged;
};

const isTransformationChanged = (
	oldTransformation: TransformationFunction,
	newTransformation: TransformationFunction,
): boolean => {
	return oldTransformation.source.trim() !== newTransformation.source.trim();
};

const isMappingModified = (
	transformationType: '' | 'mappings' | 'function',
	oldValue: TransformedMapping | TransformationFunction,
	newValue: TransformedMapping | TransformationFunction,
) => {
	if (transformationType === '') {
		return;
	}
	if (transformationType === 'mappings') {
		const oldV = oldValue as TransformedMapping;
		const newV = newValue as TransformedMapping;
		return isMappingChanged(oldV, newV);
	} else {
		const oldV = oldValue as TransformationFunction;
		const newV = newValue as TransformationFunction;
		return isTransformationChanged(oldV, newV);
	}
};

const TransformationBox = ({
	transformationType,
	setTransformationType,
	workspaces,
	selectedWorkspace,
	action,
	setAction,
	mappingItems,
	onSelectMappingItem,
	updateMapping,
	updateMappingError,
	isTransformationDisabled,
	transformationLanguages,
	selectedLanguage,
	setSelectedLanguage,
	onOpenFullscreenTransformation,
	onChangeTransformationFunction,
	isFullscreenTransformationOpen,
	onCloseFullscreenTransformation,
	actionType,
	hasSchema,
	flatInputSchema,
}: TransformationBoxProps) => {
	const [isAlertOpen, setIsAlertOpen] = useState<boolean>(false);
	const [isCompletelyOpen, setIsCompletelyOpen] = useState<boolean>(false);
	const [isFullscreenAnimating, setIsFullscreenAnimating] = useState<boolean>(false);
	const [isEditTooltipOpen, setIsEditTooltipOpen] = useState<boolean>();

	const pendingTransformationType = useRef<string>();
	const firstValue = useRef<TransformedMapping | TransformationFunction>();
	const hasNeverChangedTransformationType = useRef<boolean>(true);

	const { connection } = useContext(ConnectionContext);
	const { setSelectedInPaths, setSelectedOutPaths, isEditing, isImport } = useContext(actionContext);

	useEffect(() => {
		if (transformationType === 'mappings') {
			firstValue.current = structuredClone(action.transformation.mapping);
		} else {
			firstValue.current = structuredClone(action.transformation.function);
		}
	}, [transformationType, selectedLanguage]);

	useEffect(() => {
		if (isFullscreenTransformationOpen) {
			setTimeout(() => {
				setIsFullscreenAnimating(true);
			}, 100);
			setTimeout(() => {
				setIsFullscreenAnimating(false);
				setIsCompletelyOpen(true);
			}, 300);
		} else {
			setIsCompletelyOpen(false);
		}
	}, [isFullscreenTransformationOpen]);

	const onEditorMount = (editor: any) => {
		editor.onDidAttemptReadOnlyEdit(() => {
			setIsEditTooltipOpen(true);
		});
	};

	const onChangeTransformationType = (delay: number) => {
		hasNeverChangedTransformationType.current = false;
		const a = { ...action };
		a.inSchema = null;
		a.outSchema = null;
		setIsAlertOpen(false);
		setTimeout(() => {
			if (pendingTransformationType.current == 'mappings') {
				a.transformation.mapping = flattenSchema(actionType.outputSchema);
				a.transformation.function = null;
				setSelectedLanguage('');
				setTransformationType('mappings');
			} else {
				a.transformation.mapping = null;
				a.transformation.function = {
					source: RAW_TRANSFORMATION_FUNCTIONS[pendingTransformationType.current].replace(
						'$parameterName',
						getTransformationFunctionParameterName(connection, actionType),
					),
					language: pendingTransformationType.current,
					preserveJSON: false,
					inPaths: [],
					outPaths: [],
				};
				setSelectedLanguage(pendingTransformationType.current);
				setTransformationType('function');
			}
			setAction(a);
			setSelectedInPaths([]);
			setSelectedOutPaths([]);
		}, delay);
	};

	const onTransformationTypeClick = (newTransformationType: string) => {
		if (newTransformationType === transformationType) {
			return;
		}
		setIsEditTooltipOpen(false);
		pendingTransformationType.current = newTransformationType;
		if (
			isMappingModified(
				transformationType,
				firstValue.current,
				transformationType === 'mappings' ? action.transformation.mapping : action.transformation.function,
			) ||
			(isEditing && hasNeverChangedTransformationType.current)
		) {
			setIsAlertOpen(true);
		} else {
			onChangeTransformationType(0);
		}
	};

	const onFunctionPreserveJSONSwitch = (e) => {
		e.preventDefault();
		const a = { ...action };
		a.transformation.function.preserveJSON = !a.transformation.function.preserveJSON;
		setAction(a);
	};

	const onOpenTransformation = () => {
		setIsEditTooltipOpen(false);
		onOpenFullscreenTransformation();
	};

	let body: ReactNode;
	if (transformationType === 'mappings') {
		const workspace = workspaces.find((w) => w.id === selectedWorkspace);
		const mappings: ReactNode[] = [];
		for (const path in action.transformation.mapping) {
			const property = action.transformation.mapping[path];

			const isIdentifier = isImport && workspace.identifiers.includes(path);
			const isOutMatchingProperty = !!action.matching?.out && action.matching.out === path;
			const showMatchingIn = isOutMatchingProperty && property.value === '';
			const isTableKey = !!action.tableKey && action.tableKey === path;
			const isDisabled =
				isTransformationDisabled ||
				property.disabled === true ||
				(isOutMatchingProperty && property.value === '');

			const keys = Object.keys(action.transformation.mapping);

			const parents: string[] = [];
			for (const key of keys) {
				if (path.startsWith(`${key}.`)) {
					parents.push(key);
				}
			}

			let closestMappedParent: string;
			for (const parent of [...parents].reverse()) {
				const isMapped =
					action.transformation.mapping[parent].value !== '' &&
					action.transformation.mapping[parent].error === '';
				if (isMapped) {
					closestMappedParent = parent;
					break;
				}
			}

			let automaticMapping: string | undefined;
			if (closestMappedParent != null) {
				// If there is a mapped parent, and the property that is
				// mapped on it has a sub-property that has the same
				// name and type of the current property, show the
				// automatic mapping between the two.
				const mapping = action.transformation.mapping[closestMappedParent];
				const indentationDifference = property.indentation - mapping.indentation;
				const mappingProperty = flatInputSchema[mapping.value];
				if (mappingProperty.full.type.kind === 'object') {
					const flat = flattenSchema(mappingProperty.full.type as ObjectType);
					let key = Object.keys(flat).find(
						(k) =>
							flat[k].full.name === property.full.name &&
							flat[k].indentation === indentationDifference - 1 &&
							propertyTypesAreEqual(flat[k].full.type, property.full.type),
					);
					if (key != null) {
						automaticMapping = `${mapping.value}.${key}`;
					}
				}
			}

			const hasRequired =
				isTableKey ||
				(actionType.target === 'Event' && (property.createRequired || property.updateRequired)) ||
				(action.exportMode != null &&
					((property.createRequired && action.exportMode.includes('Create')) ||
						(property.updateRequired && action.exportMode.includes('Update'))));

			let showRequired = false;
			if (hasRequired) {
				const isFirstLevel = property.indentation === 0;
				if (isFirstLevel || isTableKey) {
					showRequired = true;
				} else {
					if (property.value !== '') {
						showRequired = true;
					} else {
						const hasMappedParent = closestMappedParent != null;
						if (hasMappedParent) {
							showRequired = true;
						} else {
							const siblings: string[] = [];
							for (const key of keys) {
								const prop = action.transformation.mapping[key];
								if (
									prop.root === property.root &&
									prop.indentation === property.indentation &&
									key !== path
								) {
									siblings.push(key);
								}
							}
							const hasMappedSiblings =
								siblings.findIndex((k) => action.transformation.mapping[k].value !== '') !== -1;
							if (hasMappedSiblings) {
								showRequired = true;
							}
						}
					}
				}
			}

			const typ = property.full.type;
			const isEnum = typ.kind === 'text' && (typ as TextType).values != null;
			const isBool = typ.kind === 'boolean';

			let enumValues: string[] = [];
			if (isEnum) {
				const values = (typ as TextType).values;
				for (const v of values) {
					enumValues.push(`"${v}"`);
				}
			} else if (isBool) {
				enumValues = ['true', 'false'];
			}

			const typeName = toMeergoStringType(property.full.type);

			if (property.type === 'map') {
				mappings.push(
					<MapMapping
						property={property}
						propertyPath={path}
						mappingItems={mappingItems}
						updateMapping={updateMapping}
						onSelect={onSelectMappingItem}
						updateMappingError={updateMappingError}
						automaticMapping={automaticMapping}
						isFullscreenTransformationOpen={isFullscreenTransformationOpen}
						isDisabled={isDisabled}
						indentation={action.transformation.mapping![path].indentation!}
						showRequired={showRequired}
					/>,
				);
			} else {
				mappings.push(
					<React.Fragment key={path}>
						<Combobox
							onInput={updateMapping}
							value={
								showMatchingIn
									? action.matching.in
									: automaticMapping != null
										? automaticMapping
										: property.value
							}
							controlled={true}
							name={path}
							disabled={isDisabled}
							className='action__transformation-input-property'
							size='small'
							error={
								isOutMatchingProperty && property.value !== ''
									? 'Please leave this input empty, as the mapping is automatic in this case'
									: property.error
							}
							autocompleteExpressions={true}
							updateError={updateMappingError}
							type={property.full.type}
							isExpression={true}
							enumValues={enumValues.length > 0 ? enumValues : undefined}
							items={mappingItems}
							onSelect={onSelectMappingItem}
						>
							{isIdentifier && (
								<div className='action__transformation-property-icon' slot='prefix'>
									<SlTooltip content='Used as identifier' hoist>
										<SlIcon
											name='person-check'
											className='action__transformation-property-identifier'
										/>
									</SlTooltip>
								</div>
							)}
						</Combobox>
						<div className='action__transformation-mapping-arrow'>
							<SlIcon name='arrow-right' />
						</div>
						<div
							className={`action__transformation-output-property${
								property?.indentation! > 0 ? ' action__transformation-output-property--indented' : ''
							}`}
							style={
								{
									'--mapping-indentation': `${action.transformation.mapping![path].indentation! * 30}px`,
								} as React.CSSProperties
							}
						>
							<PropertyTooltip propertyName={path} typeName={typeName} type={property.full.type}>
								<span className='action__transformation-output-property-key'>{property.full.name}</span>
								<span className='action__transformation-output-property-type'>{typeName}</span>
							</PropertyTooltip>
							{showRequired && (
								<span className='action__transformation-output-property-required'>required</span>
							)}
						</div>
					</React.Fragment>,
				);
			}
		}
		let [leftHeader, rightHeader] = transformationHeaders(connection, action);
		body = (
			<div className='action__transformation-mappings'>
				{!isCompletelyOpen && (
					<>
						<div className='action__mapping-left-header'>
							{leftHeader[0]} {leftHeader[1]}
						</div>
						<div></div>
						<div className='action__mapping-right-header'>
							{rightHeader[0]} {rightHeader[1]}
						</div>
					</>
				)}
				{mappings}
			</div>
		);
	} else {
		const isTransformationLanguageDeprecated = !transformationLanguages.includes(selectedLanguage);
		body = (
			<div className='action__transformation-function'>
				<EditorWrapper
					language={selectedLanguage}
					height={400}
					name='actionTransformationEditor'
					value={action.transformation!.function.source}
					onChange={(source) => onChangeTransformationFunction(source!)}
					className={!isFullscreenTransformationOpen ? 'action__transformation-function-minimized' : ''}
					isReadOnly={isFullscreenTransformationOpen ? false : true}
					onMount={onEditorMount}
					sync={!isFullscreenTransformationOpen}
				/>
				{isTransformationLanguageDeprecated && (
					<SlAlert variant='danger' className='action__transformation-language-deprecated' open>
						<SlIcon slot='icon' name='exclamation-circle' />
						{selectedLanguage} is not supported anymore
					</SlAlert>
				)}
			</div>
		);
	}

	return (
		<div
			className={`transformation-box${' transformation-box--' + transformationType}${
				isFullscreenAnimating ? ' transformation-box--is-fullscreen-animating' : ''
			}`}
		>
			<div className='transformation-box__header'>
				{hasSchema && (
					<div className='transformation-box__header-title'>
						{isCompletelyOpen ? (
							<>
								<span className='transformation-box__header-icon'>
									{transformationType === 'mappings' ? (
										<SlIcon name='shuffle' />
									) : (
										getLanguageLogo(selectedLanguage)
									)}
								</span>
								<div className='transformation-box__header-text'>
									{transformationType === 'mappings' ? 'Mappings' : selectedLanguage}
								</div>
							</>
						) : (
							<SlButtonGroup className='transformation-box__header-buttons'>
								<SlButton
									className='transformation-box__mappings-button'
									variant={transformationType === 'mappings' ? 'primary' : 'default'}
									onClick={() => onTransformationTypeClick('mappings')}
									disabled={isTransformationDisabled}
								>
									Mappings
								</SlButton>
								{['JavaScript', 'Python'].map((language) => {
									const isConfigured = transformationLanguages.includes(language);
									const isDisabled = isTransformationDisabled || !isConfigured;
									const tab = (
										<SlButton
											key={language}
											variant={
												transformationType === 'function' && selectedLanguage === language
													? 'primary'
													: 'default'
											}
											onClick={isDisabled ? null : () => onTransformationTypeClick(language)}
											disabled={isDisabled}
										>
											{language}
										</SlButton>
									);
									if (isConfigured) {
										return tab;
									} else {
										return (
											<SlTooltip
												content={`It is not possible to use ${language} for the transformation because it needs to be configured first.`}
												className='transformation-box__not-configured-tooltip'
											>
												{tab}
											</SlTooltip>
										);
									}
								})}
							</SlButtonGroup>
						)}
					</div>
				)}
				<div className='transformation-box__header-right-buttons'>
					{transformationType === 'function' && (
						<SlDropdown
							className={`transformation-box__function-settings${isFullscreenTransformationOpen ? ' transformation-box__function-settings--visible' : ''}`}
						>
							<SlButton slot='trigger' circle>
								<SlIcon className='transformation-box__function-settings-icon' name='gear' />
								<SlIcon
									className={`transformation-box__function-settings-icon-dot${action.transformation.function.preserveJSON ? ' transformation-box__function-settings-icon-dot--active' : ''}`}
									name='circle-fill'
								></SlIcon>
							</SlButton>
							<SlMenu className='transformation-box__function-settings-menu'>
								<SlSwitch
									size='small'
									checked={action.transformation.function.preserveJSON}
									onClick={onFunctionPreserveJSONSwitch}
								>
									Preserve JSON
									<span className='transformation-box__preserve-json-description'>
										Pass and receive JSON values as strings in their original format, without
										decoding or encoding them.
									</span>
								</SlSwitch>
							</SlMenu>
						</SlDropdown>
					)}
					<SlTooltip
						className='transformation-box__edit-tooltip'
						trigger='manual'
						open={isEditTooltipOpen}
						placement='bottom'
					>
						<div className='transformation-box__fullscreen-tooltip' slot='content'>
							<span>Open the testing mode to edit the function</span>
						</div>
						<SlButton
							className='transformation-box__fullscreen-button'
							variant='primary'
							onClick={
								isFullscreenTransformationOpen ? onCloseFullscreenTransformation : onOpenTransformation
							}
							disabled={isTransformationDisabled}
						>
							{isCompletelyOpen ? (
								<SlIcon name='arrows-angle-contract' />
							) : (
								<SlIcon name='arrows-angle-expand' />
							)}
							{isCompletelyOpen ? 'Exit testing mode' : 'Testing mode'}
						</SlButton>
					</SlTooltip>
				</div>
			</div>
			<div className='transformation-box__body'>
				{hasSchema ? (
					body
				) : (
					<div className='transformation-box__no-transformation-text'>
						Sending these events does not require an explicit mapping or function transformation
					</div>
				)}
			</div>
			<AlertDialog
				variant='danger'
				isOpen={isAlertOpen}
				onClose={() => setIsAlertOpen(false)}
				title={'You will lose your work'}
				actions={
					<>
						<SlButton onClick={() => setIsAlertOpen(false)}>Cancel</SlButton>
						<SlButton variant='danger' onClick={() => onChangeTransformationType(200)}>
							Continue
						</SlButton>
					</>
				}
			>
				<div style={{ textAlign: 'center' }}>
					<p>If you switch the transformation type you will permanently lose all the work you have done</p>
				</div>
			</AlertDialog>
		</div>
	);
};

interface FullscreenTransformationProps {
	isFullscreenTransformationOpen: boolean;
	selectedLanguage: string;
	body: ReactNode;
	flatInputSchema: TransformedMapping;
	inputSchema: ObjectType;
	outputSchema: ObjectType;
}

const FullscreenTransformation = ({
	isFullscreenTransformationOpen,
	selectedLanguage,
	body,
	flatInputSchema,
	inputSchema,
	outputSchema,
}: FullscreenTransformationProps) => {
	const [isInputSchemaSelected, setIsInputSchemaSelected] = useState<boolean>(true);
	const [inSearchTerm, setInSearchTerm] = useState<string>('');
	const [showOnlyInSelected, setShowOnlyInSelected] = useState<boolean>();
	const [isOutputSchemaSelected, setIsOutputSchemaSelected] = useState<boolean>(true);
	const [outSearchTerm, setOutSearchTerm] = useState<string>('');
	const [showOnlyOutSelected, setShowOnlyOutSelected] = useState<boolean>();
	const [samples, setSamples] = useState<Sample[]>(null);
	const [isFetchingSamples, setIsFetchingSamples] = useState<boolean>(false);
	const [selectedSample, setSelectedSample] = useState<Sample>(null);
	const [events, setEvents] = useState<EventListenerEvent[]>([]);
	const [selectedEvent, setSelectedEvent] = useState<EventListenerEvent>(null);
	const [output, setOutput] = useState<string>('');
	const [outputError, setOutputError] = useState<string>('');
	const [isExecuting, setIsExecuting] = useState<boolean>(false);
	const [isBodyRendered, setIsBodyRendered] = useState<boolean>(false);
	const [isBodyShown, setIsBodyShown] = useState<boolean>(false);

	const { handleError, api } = useContext(AppContext);
	const {
		action,
		settings,
		actionType,
		connection,
		transformationType,
		selectedInPaths,
		setSelectedInPaths,
		selectedOutPaths,
		setSelectedOutPaths,
	} = useContext(ActionContext);

	const firstNameIdentifier = useRef<string>('');
	const lastNameIdentifier = useRef<string>('');
	const emailIdentifier = useRef<string>('');
	const idIdentifier = useRef<string>('');
	const lastExecutedSample = useRef<Sample>(null);
	const lastExecutedEvent = useRef<EventListenerEvent>(null);
	const eventSchema = useRef<ObjectType>(null);
	const hasAlreadyFetchedSamples = useRef<boolean>(false);
	const selectedInProperties = useRef<string[]>();
	const selectedOutProperties = useRef<string[]>();

	const collectEvents = (newly: EventListenerEvent[]) => {
		setEvents((prevEvents) => [...prevEvents, ...newly]);
	};

	const { isEventBasedUserImport, isAppEventsExport } = useMemo(() => {
		return {
			isEventBasedUserImport: connection.isEventBased && connection.isSource && actionType.target === 'User',
			isAppEventsExport: connection.isApp && connection.isDestination && actionType.target === 'Event',
		};
	}, [connection, actionType]);

	const { flatOutputSchema } = useMemo(() => {
		return {
			flatOutputSchema: flattenSchema(outputSchema),
		};
	}, [outputSchema]);

	let eventListenerFilter = null;
	if (isEventBasedUserImport || isAppEventsExport) {
		let filter = {
			logical: action.filter != null ? action.filter.logical : 'and',
			conditions: action.filter != null ? [...action.filter.conditions] : [],
		};
		if (isAppEventsExport && connection.linkedConnections == null) {
			filter = null;
		} else {
			filter.conditions.push({
				property: 'connection',
				operator: 'is one of',
				values: isEventBasedUserImport
					? [String(connection.id)]
					: connection.linkedConnections.map((id) => String(id)),
			});
			if (isEventBasedUserImport) {
				filter.conditions.push({
					property: 'traits',
					operator: 'is not',
					values: ['null'],
				});
			}
		}
	}

	const { startListening } = useEventListener(collectEvents, null, eventListenerFilter);

	useEffect(() => {
		if (isEventBasedUserImport || isAppEventsExport) {
			startListening();
		}
	}, []);

	useEffect(() => {
		// Reset the output of the transformation tests when the user
		// switches the language or the type of the transformation.
		setOutput('');
		setOutputError('');
	}, [transformationType, selectedLanguage]);

	useEffect(() => {
		setShowOnlyInSelected(false);
		setShowOnlyOutSelected(false);
		selectedInProperties.current = null;
		selectedOutProperties.current = null;
		setInSearchTerm('');
		setOutSearchTerm('');
	}, [transformationType]);

	useEffect(() => {
		if (actionType.target === 'Event' && outputSchema == null) {
			// The action doesn't have a transformation. The fullscreen
			// is shown only to allow testing of event dispatching, so
			// we preselect the samples and preview panels.
			setIsInputSchemaSelected(false);
			setIsOutputSchemaSelected(false);
		}
	}, []);

	useEffect(() => {
		if (isFullscreenTransformationOpen) {
			// Delay the rendering of the transformation box to avoid
			// content flashes and to ensure that the code editor is
			// rendered only when the screen is fully open, avoiding
			// unexpected behaviors such as text selection issues.
			setTimeout(() => {
				setIsBodyRendered(true);
				setTimeout(() => {
					setIsBodyShown(true);
				}, 300);
			}, 150);
		} else {
			setIsBodyRendered(false);
			setIsBodyShown(false);
		}
	}, [isFullscreenTransformationOpen]);

	useEffect(() => {
		const fetchSamples = async () => {
			if (actionType.target !== 'User') {
				return;
			}
			if (!isFullscreenTransformationOpen || hasAlreadyFetchedSamples.current) {
				return;
			}
			setIsFetchingSamples(true);
			let samples: Sample[];
			if (connection.isFileStorage && connection.isSource) {
				let res: RecordsResponse;
				try {
					res = await api.workspaces.connections.records(
						connection.id,
						action.path,
						action.format,
						action.sheet,
						action.compression,
						settings,
						20,
					);
				} catch (err) {
					setIsFetchingSamples(false);
					handleError(err);
					return;
				}
				samples = res.records;
			} else if (connection.isDatabase && connection.isSource) {
				// Will show a button to execute the query and retrieve the
				// samples (as the query can be potentially destructive).
				setIsFetchingSamples(false);
				return;
			} else if (connection.isApp && connection.isSource) {
				let res: AppUsersResponse;
				try {
					res = await api.workspaces.connections.appUsers(connection.id, inputSchema);
				} catch (err) {
					setIsFetchingSamples(false);
					handleError(err);
					return;
				}
				samples = res.users;
			} else if ((connection.isApp || connection.isDatabase) && connection.isDestination) {
				const properties: string[] = [];
				for (const prop of inputSchema.properties) {
					properties.push(prop.name);
				}
				let res: FindUsersResponse;
				try {
					res = await api.workspaces.users.find(properties, null, '', true, 0, 20);
				} catch (err) {
					setIsFetchingSamples(false);
					handleError(err);
					return;
				}
				if (res.users.length === 0) {
					return;
				}
				const s: Sample[] = [];
				for (const user of res.users) {
					s.push(user.traits);
				}
				samples = s;
			} else {
				setIsFetchingSamples(false);
				return;
			}
			setIsFetchingSamples(false);
			hasAlreadyFetchedSamples.current = true;
			const idents = getSampleIdentifiers(samples[0]);
			if (idents != null) {
				firstNameIdentifier.current = idents.firstNameIdentifier;
				lastNameIdentifier.current = idents.lastNameIdentifier;
				emailIdentifier.current = idents.emailIdentifier;
				idIdentifier.current = idents.idIdentifier;
			}
			setSamples(samples);
		};
		fetchSamples();
	}, [isFullscreenTransformationOpen]);

	useEffect(() => {
		let el: Element;
		if (selectedSample == null) {
			// sample has been closed.
			el = document.querySelector('.fullscreen-transformation__sample--last-executed');
			if (el == null) {
				// sample has been closed directly by a click on its heading and
				// not because another sample has been executed.
				return;
			}
		} else {
			// sample has been closed because another sample has been opened.
			el = document.querySelector('.fullscreen-transformation__sample--open');
		}
		const accordion = el.closest('.accordion');
		const panel = document.querySelector(
			'.fullscreen-transformation__input-panel .fullscreen-transformation__panel-content',
		);

		setTimeout(() => {
			const isVisible = isElementVisibleInLeftPanel(accordion, panel);
			if (!isVisible) {
				el.scrollIntoView({ behavior: 'smooth', block: 'start', inline: 'nearest' });
			}
		}, 250);
	}, [selectedSample]);

	useEffect(() => {
		let el: Element;
		if (selectedEvent == null) {
			// event has been closed.
			el = document.querySelector('.fullscreen-transformation__event--last-executed');
			if (el == null) {
				// event has been closed directly by a click on its heading and
				// not because another event has been executed.
				return;
			}
		} else {
			// event has been closed because another event has been opened.
			el = document.querySelector('.fullscreen-transformation__event--open');
		}
		const accordion = el.closest('.accordion');
		const panel = document.querySelector(
			'.fullscreen-transformation__input-panel .fullscreen-transformation__panel-content',
		);

		setTimeout(() => {
			const isVisible = isElementVisibleInLeftPanel(accordion, panel);
			if (!isVisible) {
				el.scrollIntoView({ behavior: 'smooth', block: 'start', inline: 'nearest' });
			}
		}, 250);
	}, [selectedEvent]);

	const onSelectInputSamples = () => {
		setIsInputSchemaSelected(false);
	};

	const onSelectInputSchema = () => {
		setIsInputSchemaSelected(true);
	};

	const onSelectOutputResult = () => {
		setIsOutputSchemaSelected(false);
	};

	const onSelectOutputSchema = () => {
		setIsOutputSchemaSelected(true);
	};

	const onInputInSearchTerm = (e) => {
		setInSearchTerm(e.target.value);
	};

	const onInputOutSearchTerm = (e) => {
		setOutSearchTerm(e.target.value);
	};

	const onChangeShowOnlySelected = (side: 'in' | 'out') => {
		let selectedProperties: React.MutableRefObject<string[]>;
		let setterFunc: React.Dispatch<React.SetStateAction<boolean>>;
		if (side === 'in') {
			selectedProperties = selectedInProperties;
			setterFunc = setShowOnlyInSelected;
		} else {
			selectedProperties = selectedOutProperties;
			setterFunc = setShowOnlyOutSelected;
		}
		const isShowingOnlySelected = selectedProperties.current != null;
		if (isShowingOnlySelected) {
			selectedProperties.current = null;
			setterFunc(false);
		} else {
			const toShow: string[] = [];
			let properties: Property[];
			let paths: string[];
			if (side === 'in') {
				properties = inputSchema.properties;
				paths = selectedInPaths;
			} else {
				properties = outputSchema.properties;
				paths = selectedOutPaths;
			}
			for (const p of properties) {
				const isSelected = paths.includes(p.name);
				const hasSelectedChildren = paths.findIndex((prop) => prop.startsWith(`${p.name}.`)) !== -1;
				if (!isSelected && !hasSelectedChildren) {
					continue;
				}
				toShow.push(p.name);
			}
			selectedProperties.current = toShow;
			setterFunc(true);
		}
	};

	const onChangeSelectedPath = (side: 'in' | 'out', path: string) => {
		let paths: string[];
		let schema: TransformedMapping;
		if (side === 'in') {
			paths = selectedInPaths;
			schema = flatInputSchema;
		} else {
			paths = selectedOutPaths;
			schema = flatOutputSchema;
		}

		const keys = Object.keys(schema);
		const children = keys.filter((k) => k.startsWith(`${path}.`));

		const isSelected = paths.includes(path);
		let p: string[] = [];
		if (isSelected) {
			// Remove the property from the selected list.
			for (const s of paths) {
				if (s !== path) {
					p.push(s);
				}
			}
		} else {
			p = [];
			p.push(path);

			// Remove any child properties that were previously selected
			// since only the parent property will be sent to the
			// server.
			for (const s of paths) {
				if (!children.includes(s)) {
					p.push(s);
				}
			}
		}

		if (side == 'in') {
			setSelectedInPaths(p);
		} else {
			setSelectedOutPaths(p);
		}
	};

	const onSampleClick = (e: any, sample: Sample) => {
		const isOnExecuteButton = e.target === 'SL-ICON-BUTTON';
		if (isOnExecuteButton) {
			return;
		}
		const isOpen = JSON.stringify(sample) === JSON.stringify(selectedSample);
		if (isOpen) {
			setSelectedSample(null);
		} else {
			setSelectedSample(sample);
		}
	};

	const onEventClick = (evt: any, event: EventListenerEvent) => {
		const isOnExecuteButton = evt.target === 'SL-ICON-BUTTON';
		if (isOnExecuteButton) {
			return;
		}
		const isOpen = JSON.stringify(event) === JSON.stringify(selectedEvent);
		if (isOpen) {
			setSelectedEvent(null);
		} else {
			setSelectedEvent(event);
		}
	};

	const onTransformSample = async (sample: Record<string, any>) => {
		lastExecutedSample.current = sample;
		if (JSON.stringify(sample) !== JSON.stringify(selectedSample)) {
			setSelectedSample(null);
		}
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(
				action,
				settings,
				actionType,
				api,
				connection,
				false,
				selectedInPaths,
				selectedOutPaths,
			);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsExecuting(false);
			}, 300);
			return;
		}

		let inSchema = actionToSet.inSchema;

		// Only send the sample's properties that are actually present
		// in the input schema of the "ActionToSet".
		let s = buildFilteredSample(flattenSchema(actionToSet.inSchema), sample);

		let purpose: TransformationPurpose =
			action.exportMode != null && action.exportMode === 'UpdateOnly' ? 'Update' : 'Create';
		let res: TransformDataResponse;
		try {
			res = await api.transformData(s, inSchema, actionToSet.outSchema, actionToSet.transformation, purpose);
		} catch (err) {
			setOutput('');
			if (err instanceof UnprocessableError && err.code === 'TransformationFailed') {
				setTimeout(() => {
					setOutputError(err.message);
					setIsExecuting(false);
				}, 300);
			} else {
				setTimeout(() => {
					handleError(err);
					setIsExecuting(false);
				}, 300);
			}
			return;
		}
		setOutput(JSONbig.stringify(res.data, null, 4));
		setTimeout(() => setIsExecuting(false), 300);
	};

	const onTransformUserEvent = async (event: EventListenerEvent) => {
		lastExecutedEvent.current = event;
		if (selectedEvent && event.id !== selectedEvent.id) {
			setSelectedEvent(null);
		}
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);

		if (transformationType === 'mappings') {
			let hasMappedProperty = false;
			for (const k in action.transformation.mapping) {
				if (action.transformation.mapping[k].value !== '') {
					hasMappedProperty = true;
					break;
				}
			}
			if (!hasMappedProperty) {
				setTimeout(() => {
					// Since having no transformation is allowed in the actions
					// that import users from events, simply display an empty
					// JSON object.
					setOutput(JSONbig.stringify({}, null, 4));
					setIsExecuting(false);
				}, 300);
				return;
			}
		}

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(
				action,
				settings,
				actionType,
				api,
				connection,
				false,
				selectedInPaths,
				selectedOutPaths,
			);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsExecuting(false);
			}, 300);
			return;
		}

		let inSchema: ObjectType;
		if (eventSchema.current != null) {
			inSchema = eventSchema.current;
		} else {
			try {
				inSchema = await api.eventsSchema();
			} catch (err) {
				setTimeout(() => {
					handleError(err);
					setIsExecuting(false);
				}, 300);
				return;
			}
			eventSchema.current = { ...inSchema };
		}

		let purpose: TransformationPurpose =
			action.exportMode != null && action.exportMode === 'UpdateOnly' ? 'Update' : 'Create';
		let res: TransformDataResponse;
		try {
			const data = event.full;
			res = await api.transformData(data, inSchema, actionToSet.outSchema, actionToSet.transformation, purpose);
		} catch (err) {
			setOutput('');
			if (err instanceof UnprocessableError && err.code === 'TransformationFailed') {
				setTimeout(() => {
					setOutputError(err.message);
					setIsExecuting(false);
				}, 300);
			} else {
				setTimeout(() => {
					handleError(err);
					setIsExecuting(false);
				}, 300);
			}
			return;
		}
		setOutput(JSONbig.stringify(res.data, null, 4));
		setTimeout(() => setIsExecuting(false), 300);
	};

	const onTransformEvent = async (event: EventListenerEvent) => {
		lastExecutedEvent.current = event;
		if (selectedEvent && event.id !== selectedEvent.id) {
			setSelectedEvent(null);
		}
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(
				action,
				settings,
				actionType,
				api,
				connection,
				false,
				selectedInPaths,
				selectedOutPaths,
			);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsExecuting(false);
			}, 300);
			return;
		}

		let res: PreviewSendEventResponse;
		try {
			res = await api.workspaces.connections.previewSendEvent(
				connection.id,
				actionType.eventType,
				event.full,
				actionToSet.outSchema,
				actionToSet.transformation,
			);
		} catch (err) {
			setOutput('');
			if (err instanceof UnprocessableError && err.code === 'TransformationFailed') {
				setTimeout(() => {
					setOutputError(err.message);
					setIsExecuting(false);
				}, 300);
			} else {
				setTimeout(() => {
					handleError(err);
					setIsExecuting(false);
				}, 300);
			}
			return;
		}
		setOutput(res.preview);
		setTimeout(() => setIsExecuting(false), 300);
	};

	const onQuery = async () => {
		let res: ExecQueryResponse;
		try {
			res = await api.workspaces.connections.execQuery(connection.id, action.query, 20);
		} catch (err) {
			handleError(err);
			return;
		}
		const idents = getSampleIdentifiers(res.rows[0]);
		if (idents != null) {
			firstNameIdentifier.current = idents.firstNameIdentifier;
			lastNameIdentifier.current = idents.lastNameIdentifier;
			emailIdentifier.current = idents.emailIdentifier;
			idIdentifier.current = idents.idIdentifier;
		}
		setSamples(res.rows);
	};

	const onClear = () => {
		lastExecutedSample.current = null;
		lastExecutedEvent.current = null;
		setOutput('');
		setOutputError('');
	};

	let inputPanelContent: ReactNode = null;
	if (isInputSchemaSelected) {
		inputPanelContent = (
			<div className='fullscreen-transformation__panel-schema'>
				<SlInput
					className='fullscreen-transformation__panel-schema-search'
					onSlInput={onInputInSearchTerm}
					value={inSearchTerm}
					placeholder='Search a property...'
					size='small'
					clearable
				>
					<SlIcon name='search' slot='prefix' />
				</SlInput>
				{transformationType === 'function' && (
					<SlSwitch
						className='fullscreen-transformation__panel-schema-show-only-selected'
						size='small'
						onSlChange={() => onChangeShowOnlySelected('in')}
					>
						Show only selected properties
					</SlSwitch>
				)}
				{inputSchema?.properties.map((p) => {
					if (transformationType === 'function') {
						if (showOnlyInSelected) {
							const isSelected = selectedInProperties.current?.includes(p.name);
							if (!isSelected) {
								return null;
							}
						}
					}
					if (isRecursiveType(p.type)) {
						return (
							<TransformationNestedProperties
								key={p.name}
								property={p}
								language={selectedLanguage}
								nesting={1}
								side='input'
								transformationType={transformationType}
								exportMode={action.exportMode}
								searchTerm={inSearchTerm}
								flatSchema={flatInputSchema}
								selectedPaths={selectedInPaths}
								onChangeSelectedPath={(path) => onChangeSelectedPath('in', path)}
								tableKey={action.tableKey}
							/>
						);
					} else {
						return (
							<TransformationProperty
								key={p.name}
								language={selectedLanguage}
								property={p}
								side='input'
								transformationType={transformationType}
								exportMode={action.exportMode}
								searchTerm={inSearchTerm}
								selectedPaths={selectedInPaths}
								onChangeSelectedPath={(path) => onChangeSelectedPath('in', path)}
								tableKey={action.tableKey}
							/>
						);
					}
				})}
			</div>
		);
	} else if (isFetchingSamples) {
		inputPanelContent = (
			<div className='fullscreen-transformation__samples-loading'>
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				/>
			</div>
		);
	} else if (samples) {
		inputPanelContent = (
			<div className='fullscreen-transformation__samples'>
				{Array.from(samples.entries()).map(([i, s]) => {
					const isOpen = JSON.stringify(s) === JSON.stringify(selectedSample);
					const isLastExecuted =
						lastExecutedSample.current && JSON.stringify(lastExecutedSample.current) === JSON.stringify(s);

					let highlightedLines: boolean[] = [true]; // First curly brace is highlighted.

					if (transformationType === 'function') {
						// Highlight the selected properties.
						for (const k in s) {
							const v = s[k];
							if (typeof v === 'object') {
								const children = getSelectedChildrenProperties(k, selectedInPaths, v);
								const keys = Object.keys(children);

								let isSelected = false;
								if (selectedInPaths.includes(k)) {
									isSelected = true;
									highlightedLines.push(true);
									for (const _ of keys) {
										highlightedLines.push(true);
									}
								} else {
									const hasSelectedChildren = keys.findIndex((key) => children[key] === true) !== -1;
									isSelected = hasSelectedChildren;
									highlightedLines.push(hasSelectedChildren);
									for (const key of keys) {
										highlightedLines.push(children[key]);
									}
								}
								highlightedLines.push(isSelected); // Final curly brace is highlighted.
								continue;
							} else {
								highlightedLines.push(selectedInPaths.includes(k));
								continue;
							}
						}
						highlightedLines.push(true); // Final curly brace is highlighted.
					}

					return (
						<Accordion
							key={i}
							isOpen={isOpen}
							summary={
								<div
									className={`fullscreen-transformation__sample${isOpen ? ' fullscreen-transformation__sample--open' : ''}${isLastExecuted ? ' fullscreen-transformation__sample--last-executed' : ''}`}
									onClick={(e) => onSampleClick(e, s)}
								>
									<div className='fullscreen-transformation__sample-name'>
										{actionType.target === 'User' ? (
											<>
												{idIdentifier.current && (
													<div className='fullscreen-transformation__sample-id'>
														{removeQuotes(s[idIdentifier.current])}
													</div>
												)}
												<div>
													<div className='fullscreen-transformation__sample-full-name'>
														{firstNameIdentifier.current && lastNameIdentifier.current
															? removeQuotes(s[firstNameIdentifier.current]) +
																' ' +
																removeQuotes(s[lastNameIdentifier.current])
															: `Sample ${i}`}
													</div>
													{emailIdentifier.current && (
														<div className='fullscreen-transformation__sample-email'>
															{removeQuotes(s[emailIdentifier.current])}
														</div>
													)}
												</div>
											</>
										) : (
											''
										)}
									</div>
									<div className='fullscreen-transformation__execute-button'>
										<SlIconButton
											disabled={isExecuting}
											name='play-circle'
											onClick={(e) => {
												e.stopPropagation();
												onTransformSample(s);
											}}
										/>
									</div>
								</div>
							}
							details={
								<div className='fullscreen-transformation__sample-source'>
									<SyntaxHighlight
										language='json'
										showLineNumbers={true}
										wrapLines={true}
										lineNumberStyle={{ display: 'none' }}
										lineProps={(n) => {
											if (highlightedLines[n - 1] === false) {
												return { 'data-excluded': '' };
											}
											return {};
										}}
									>
										{JSONbig.stringify(s, null, 4)}
									</SyntaxHighlight>
								</div>
							}
						/>
					);
				})}
			</div>
		);
	} else if (connection.isDatabase && connection.isSource) {
		inputPanelContent = (
			<div className='fullscreen-transformation__query-execution'>
				<SlIcon name='database-down' />
				<p className='fullscreen-transformation__query-execution-text'>
					Execute the query to retrieve the samples
				</p>
				<SlButton
					className='fullscreen-transformation__query-execution-button'
					variant='primary'
					onClick={onQuery}
				>
					Execute the query
				</SlButton>
			</div>
		);
	} else if (isEventBasedUserImport || isAppEventsExport) {
		if (isAppEventsExport && (connection.linkedConnections == null || connection.linkedConnections.length === 0)) {
			inputPanelContent = (
				<div className='fullscreen-transformation__no-sample'>
					<p className='fullscreen-transformation__no-sample-text'>
						<h3>There are no events</h3>
						<div>Please link an event source to this connection to start collecting events.</div>
					</p>
				</div>
			);
		} else {
			const reversedEvents: EventListenerEvent[] = [...events].reverse();
			inputPanelContent = (
				<div className='fullscreen-transformation__event-listener'>
					<div className='fullscreen-transformation__event-listener-list'>
						<div className='fullscreen-transformation__event-listener-body'>
							{events.length === 0 && (
								<div className='fullscreen-transformation__event-listener-no-event'>
									Listening for new events{' '}
									<span className='fullscreen-transformation__event-listener-loading-ellipsis'>
										<span className='fullscreen-transformation__event-listener-ellipsis1'>.</span>
										<span className='fullscreen-transformation__event-listener-ellipsis2'>.</span>
										<span className='fullscreen-transformation__event-listener-ellipsis3'>.</span>
									</span>
								</div>
							)}
							{reversedEvents.map((e) => {
								const isOpen = selectedEvent && selectedEvent.id === e.id;
								const isLastExecuted =
									lastExecutedEvent.current &&
									JSON.stringify(lastExecutedEvent.current) === JSON.stringify(e);
								return (
									<Accordion
										key={e.id}
										isOpen={JSON.stringify(e) === JSON.stringify(selectedEvent)}
										summary={
											<div
												className={`fullscreen-transformation__event${isOpen ? ' fullscreen-transformation__event--open' : ''}${
													isLastExecuted
														? ' fullscreen-transformation__event--last-executed'
														: ''
												}`}
												onClick={(evt) => onEventClick(evt, e)}
											>
												<div className='fullscreen-transformation__event-type'>
													{e.type}{' '}
													{e.type === 'track' && (
														<span className='fullscreen-transformation__event-name'>
															{e.full.event}
														</span>
													)}
												</div>
												<div className='fullscreen-transformation__event-time'>
													<SlRelativeTime date={e.time} sync lang='en-US' />
												</div>
												<SlIconButton
													className='fullscreen-transformation__event-run'
													name='play-circle'
													onClick={(evt) => {
														if (isAppEventsExport) {
															onTransformEvent(e);
														} else {
															onTransformUserEvent(e);
														}
														evt.stopPropagation();
													}}
												/>
											</div>
										}
										details={
											<div className='fullscreen-transformation__event-source'>
												<SyntaxHighlight>{JSONbig.stringify(e.full, null, 4)}</SyntaxHighlight>
											</div>
										}
									/>
								);
							})}
						</div>
					</div>
				</div>
			);
		}
	} else if (connection.isDestination && actionType.target === 'User') {
		inputPanelContent = (
			<div className='fullscreen-transformation__no-sample'>
				<p className='fullscreen-transformation__no-sample-text'>
					<h3>There are no users</h3>
					<div>No users have been imported into the warehouse yet.</div>
				</p>
			</div>
		);
	} else {
		inputPanelContent = (
			<div className='fullscreen-transformation__no-sample'>
				<p className='fullscreen-transformation__no-sample-text'>
					<h3>There are no samples</h3>
					<div>No samples can be retrieved to test the transformation.</div>
				</p>
			</div>
		);
	}

	let [leftHeader, rightHeader] = transformationHeaders(connection, action);
	return (
		<div
			className={`fullscreen-transformation${isFullscreenTransformationOpen ? ' fullscreen-transformation--open' : ''}`}
		>
			<SlSplitPanel style={{ '--min': '70%', '--max': 'calc(100% - 10px)' } as React.CSSProperties}>
				<div className='fullscreen-transformation__left-panel' slot='start'>
					<SlSplitPanel style={{ '--min': '10px', '--max': '42%' } as React.CSSProperties}>
						<div
							className={`fullscreen-transformation__input-panel${isInputSchemaSelected ? ' fullscreen-transformation__input-panel--schema' : ''}`}
							slot='start'
						>
							<div className='fullscreen-transformation__panel-title-wrapper'>
								<div className='fullscreen-transformation__panel-title'>
									<div className='fullscreen-transformation__panel-title-text'>{leftHeader[0]}</div>
									<div className='fullscreen-transformation__panel-sub-title'>{leftHeader[1]}</div>
								</div>
								<SlButtonGroup>
									<SlButton
										size='small'
										variant={isInputSchemaSelected ? 'primary' : 'default'}
										onClick={onSelectInputSchema}
									>
										Schema
									</SlButton>
									<SlButton
										size='small'
										variant={isInputSchemaSelected ? 'default' : 'primary'}
										onClick={onSelectInputSamples}
									>
										Samples
									</SlButton>
								</SlButtonGroup>
							</div>
							<div className='fullscreen-transformation__panel-content'>{inputPanelContent}</div>
						</div>
						<div className='fullscreen-transformation__right-panel' slot='end'>
							<div
								slot='start'
								className={`fullscreen-transformation__editor-panel${!isBodyShown ? ' fullscreen-transformation__editor-panel--hide' : ''}`}
							>
								{!isBodyShown && (
									<SlSpinner
										className='fullscreen-transformation__editor-panel-spinner'
										style={
											{
												fontSize: '3rem',
												'--track-width': '6px',
											} as React.CSSProperties
										}
									></SlSpinner>
								)}
								{isBodyRendered && body}
							</div>
						</div>
					</SlSplitPanel>
				</div>
				<div
					className={`fullscreen-transformation__output-panel${isOutputSchemaSelected ? ' fullscreen-transformation__output-panel--schema' : ''}`}
					slot='end'
				>
					<div className='fullscreen-transformation__panel-title-wrapper'>
						<div className='fullscreen-transformation__panel-title'>
							<div className='fullscreen-transformation__panel-title-text'>{rightHeader[0]}</div>
							<div className='fullscreen-transformation__panel-sub-title'>{rightHeader[1]}</div>
						</div>
						<SlButtonGroup>
							<SlButton
								size='small'
								variant={isOutputSchemaSelected ? 'primary' : 'default'}
								onClick={onSelectOutputSchema}
								disabled={isExecuting}
							>
								Schema
							</SlButton>
							<SlButton
								size='small'
								variant={isOutputSchemaSelected ? 'default' : 'primary'}
								onClick={onSelectOutputResult}
								disabled={isExecuting}
							>
								{connection.isDestination && actionType.target === 'Event' ? 'Preview' : 'Result'}
							</SlButton>
						</SlButtonGroup>
					</div>
					<div className='fullscreen-transformation__panel-content'>
						{isOutputSchemaSelected ? (
							outputSchema == null ? (
								<h3 className='fullscreen-transformation__panel-schema--no-schema'>
									There is no output schema
								</h3>
							) : (
								<div className='fullscreen-transformation__panel-schema'>
									<SlInput
										className='fullscreen-transformation__panel-schema-search'
										onSlInput={onInputOutSearchTerm}
										value={outSearchTerm}
										placeholder='Search a property...'
										size='small'
										clearable
									>
										<SlIcon name='search' slot='prefix' />
									</SlInput>
									{transformationType === 'function' && (
										<SlSwitch
											className='fullscreen-transformation__panel-schema-show-only-selected'
											size='small'
											onSlChange={() => onChangeShowOnlySelected('out')}
										>
											Show only selected properties
										</SlSwitch>
									)}
									{outputSchema?.properties.map((p) => {
										if (transformationType === 'function') {
											if (showOnlyOutSelected) {
												const isSelected = selectedOutProperties.current?.includes(p.name);
												if (!isSelected) {
													return null;
												}
											}
										}
										if (isRecursiveType(p.type)) {
											return (
												<TransformationNestedProperties
													key={p.name}
													property={p}
													language={selectedLanguage}
													nesting={1}
													side='output'
													transformationType={transformationType}
													exportMode={action.exportMode}
													searchTerm={outSearchTerm}
													flatSchema={flatOutputSchema}
													selectedPaths={selectedOutPaths}
													onChangeSelectedPath={(path) => onChangeSelectedPath('out', path)}
													tableKey={action.tableKey}
												/>
											);
										} else {
											return (
												<TransformationProperty
													key={p.name}
													property={p}
													language={selectedLanguage}
													side='output'
													transformationType={transformationType}
													exportMode={action.exportMode}
													searchTerm={outSearchTerm}
													selectedPaths={selectedOutPaths}
													onChangeSelectedPath={(path) => onChangeSelectedPath('out', path)}
													isOutMatchingProperty={
														action.matching?.out && action.matching.out === p.name
													}
													tableKey={action.tableKey}
												/>
											);
										}
									})}
								</div>
							)
						) : isExecuting ? (
							<SlSpinner
								style={
									{
										fontSize: '3rem',
										'--track-width': '6px',
									} as React.CSSProperties
								}
							></SlSpinner>
						) : output !== '' || outputError !== '' ? (
							<div className='fullscreen-transformation__output-code'>
								<SlTooltip content='Clear' placement='left' onClick={onClear}>
									<SlIconButton className='fullscreen-transformation__output-clear' name='x-lg' />
								</SlTooltip>
								{outputError !== '' ? (
									<div className='fullscreen-transformation__output-error'>{outputError}</div>
								) : (
									<div className='fullscreen-transformation__output-success'>
										{connection.isApp &&
										connection.isDestination &&
										actionType.target === 'Event' ? (
											output
										) : (
											<SyntaxHighlight>{output}</SyntaxHighlight>
										)}
									</div>
								)}
							</div>
						) : (
							<div className='fullscreen-transformation__output-placeholder'>
								<SlIcon name='play-circle' />
								<p className='fullscreen-transformation__output-placeholder-text'>
									Run the transformation on a sample to see the resulting output
								</p>
							</div>
						)}
					</div>
				</div>
			</SlSplitPanel>
		</div>
	);
};

interface MapMappingProps {
	property: TransformedProperty;
	propertyPath: string;
	mappingItems: ComboboxItem[];
	updateMapping: (path: string, value: string) => void;
	updateMappingError: (patah: string, errorMessage: string) => void;
	onSelect: (path: string, value: string) => void;
	automaticMapping: string | undefined;
	isFullscreenTransformationOpen: boolean;
	isDisabled: boolean;
	indentation: number;
	showRequired: boolean;
}

const MapMapping = ({
	property,
	propertyPath,
	mappingItems,
	updateMapping,
	updateMappingError,
	onSelect,
	automaticMapping,
	isFullscreenTransformationOpen,
	isDisabled,
	indentation,
	showRequired,
}: MapMappingProps) => {
	const [pairs, setPairs] = useState<Array<[string, string]>>([['', '']]);
	const [logicalErrors, setLogicalErrors] = useState<Record<number, string>>({});
	const [validationErrors, setValidationErrors] = useState<Record<number, string>>({});
	const [reloadLogicalErrors, setReloadLogicalErrors] = useState<boolean>(false);
	const [isResetting, setIsResetting] = useState<boolean>(false);

	const { api, handleError } = useContext(appContext);
	const { actionType } = useContext(actionContext);

	const hasFilledPairs = pairs.findIndex((p) => p[0] !== '' || p[1] !== '') > -1;
	const hasMultiplePairs = pairs.length > 1;
	const isParentMappingDisabled = isDisabled || hasFilledPairs || hasMultiplePairs;
	const areChildrenMappingDisabled =
		isDisabled ||
		automaticMapping != null ||
		(property.value !== '' && !hasFilledPairs && !hasMultiplePairs && !isResetting);

	useEffect(() => {
		// At first render, compute the pairs based on the property
		// value. Automatically convert filled out "map()" expressions
		// into corresponding pairs. Also reload the pairs when the
		// testing mode is open/closed to trigger the re-render of the
		// component and synchronize the changes between the two modes.
		const isMapExpression = property.value.startsWith('map(') && property.value !== 'map()';
		if (isMapExpression) {
			const p = parseMapString(property.value);
			setPairs(p);
			// Validate expressions when the action is opened.
			validatePairExpressions(p);
			setReloadLogicalErrors(true);
		}
	}, [isFullscreenTransformationOpen]);

	useEffect(() => {
		// Revalidate expressions when the schema changes.
		if (actionType.inputSchema != null) {
			validatePairExpressions(pairs);
		}
	}, [actionType.inputSchema]);

	useEffect(() => {
		if (property.value === '' && isResetting) {
			// The property value has been succesfully emptied in the
			// action.
			setIsResetting(false);
		}
	}, [property.value]);

	useEffect(() => {
		if (!hasFilledPairs) {
			return;
		}
		if (Object.keys(logicalErrors).length > 0 || Object.keys(validationErrors).length > 0) {
			// Propagate the error in the mapping property so that it's
			// not possible to save the action.
			updateMappingError(propertyPath, "There are errors in the map's mapping");
		} else {
			updateMappingError(propertyPath, '');
		}
	}, [property.value, logicalErrors, validationErrors]);

	useEffect(() => {
		const mappingContainers = document.querySelectorAll('.action__transformation-mappings');

		const onFocusOut = () => {
			// Add a short delay to reload the logical errors only after
			// the focus has fully shifted (The computation of logical
			// errors checks the focus to decide whether to show or hide
			// certain messages).
			setTimeout(() => {
				setReloadLogicalErrors(true);
			}, 50);
		};

		for (const container of Array.from(mappingContainers)) {
			container.addEventListener('focusout', onFocusOut);
		}

		return () => {
			for (const container of Array.from(mappingContainers)) {
				container.removeEventListener('focusout', onFocusOut);
			}
		};
	}, []);

	useEffect(() => {
		if (!reloadLogicalErrors) {
			return;
		}
		// Check if the pairs have logical errors (e.g. non-mapped or
		// duplicated keys).
		let err = {};
		let index = 0;
		for (const pair of pairs) {
			let combobox: HTMLElement;
			if (isFullscreenTransformationOpen) {
				combobox = document.querySelector(
					`.action__transformation-input-property[data-id="${propertyPath}"] ~ .action__transformation-input-property[data-id="${index}"]`,
				) as HTMLElement;
			} else {
				combobox = document.querySelector(
					`.action__body .action__transformation-section > .section__content .action__transformation-mappings .action__transformation-input-property[data-id="${propertyPath}"] ~ .action__transformation-input-property[data-id="${index}"]`,
				);
			}
			const comboboxInput = combobox.querySelector('sl-input');
			const keyInput = combobox.nextElementSibling.nextElementSibling.querySelector('sl-input') as HTMLElement;
			const hasAlreadyError = logicalErrors[index] != null;
			const hasFocus = comboboxInput.shadowRoot.activeElement !== null || document.activeElement === keyInput;
			if (hasFocus && !hasAlreadyError) {
				continue;
			}
			let [key, value] = pair;
			if (key !== '' && value === '') {
				err[index] = 'Key must be mapped';
			}
			const sameKeyIndex = pairs.findIndex((p, i) => i !== index && p[0] === key);
			if (sameKeyIndex > -1) {
				if (key === '') {
					const hasMappedEmptyKey = pairs.findIndex((p) => p[0] === '' && p[1] !== '') > -1;
					if (hasMappedEmptyKey) {
						err[index] = `Key "${key}" is duplicated`;
					}
				} else {
					err[index] = `Key "${key}" is duplicated`;
				}
			}
			index++;
		}
		setReloadLogicalErrors(false);
		setLogicalErrors(err);
	}, [reloadLogicalErrors]);

	const validatePairExpressions = async (pairs: Array<[string, string]>) => {
		let i = 0;
		for (const pair of pairs) {
			const value = pair[1];
			if (value === '') {
				i++;
				continue;
			}
			let errorMessage = '';
			try {
				errorMessage = await api.validateExpression(
					value,
					actionType.inputSchema.properties,
					(property.full.type as MapType).elementType,
				);
			} catch (err) {
				handleError(err);
				return;
			}
			if (errorMessage !== '') {
				updatePairValidationError(String(i), errorMessage);
			}
			i++;
		}
	};

	const updatePairValidationError = (i: string, errorMessage: string) => {
		const index = Number(i);
		setValidationErrors((prev) => {
			const err = structuredClone(prev);
			if (errorMessage === '') {
				delete err[index];
			} else {
				err[index] = errorMessage;
			}
			return err;
		});
	};

	const onUpdatePair = (index: number, part: 'key' | 'value', val: string) => {
		let newPairs = [
			...pairs.slice(0, index),
			[part === 'key' ? val : pairs[index][0], part === 'value' ? val : pairs[index][1]] as ['', ''],
			...pairs.slice(index + 1, pairs.length),
		];
		setPairs(newPairs);
		let value = stringifyMapPairs(newPairs);
		if (value === 'map()') {
			// The user has emptied all inputs so we must empty the
			// value of the mapping without automatically setting the
			// "map()" value.
			value = '';
			setIsResetting(true);
		}
		updateMapping(propertyPath, value);
	};

	const onSelectPairValue = (index: number, val: string) => {
		let newPairs = [
			...pairs.slice(0, index),
			[pairs[index][0], val] as ['', ''],
			...pairs.slice(index + 1, pairs.length),
		];
		setPairs(newPairs);
		let value = stringifyMapPairs(newPairs);
		updateMapping(propertyPath, value);
		setReloadLogicalErrors(true);
	};

	const onAddPair = (index: number) => {
		let newPairs = [...pairs.slice(0, index + 1), ['', ''] as ['', ''], ...pairs.slice(index + 1, pairs.length)];
		setPairs(newPairs);
		let value = stringifyMapPairs(newPairs);
		if (value === 'map()') {
			// Avoid automatically setting the "map()"" value in the
			// action if all the inputs are empty.
			value = '';
		}

		// Shift the errors.
		setLogicalErrors(shiftErrors(logicalErrors, index));
		setValidationErrors(shiftErrors(validationErrors, index));

		updateMapping(propertyPath, value);
		setReloadLogicalErrors(true);

		setTimeout(() => {
			let panel: HTMLElement;
			let newPairCombobox: HTMLElement;
			if (isFullscreenTransformationOpen) {
				panel = document.querySelector('.fullscreen-transformation__right-panel') as HTMLElement;
				newPairCombobox = panel.querySelector(
					`.action__transformation-input-property[data-id="${propertyPath}"] ~ .action__transformation-input-property[data-id="${index + 1}"]`,
				) as HTMLElement;
			} else {
				panel = document.querySelector('.fullscreen-action') as HTMLElement;
				newPairCombobox = document.querySelector(
					`.action__body .action__transformation-section > .section__content .action__transformation-mappings .action__transformation-input-property[data-id="${propertyPath}"] ~ .action__transformation-input-property[data-id="${index + 1}"]`,
				);
			}
			const focusKey = () => {
				const keyContainer = newPairCombobox.nextElementSibling.nextElementSibling;
				const keyInput = keyContainer.querySelector('sl-input') as HTMLElement;
				keyInput.focus();
			};
			const panelBottom = panel.scrollTop + panel.clientHeight;
			const elementBottom = newPairCombobox.offsetTop + newPairCombobox.offsetHeight;
			if (elementBottom > panelBottom) {
				const scrollAmount = elementBottom - panel.clientHeight;
				panel.scrollTo({
					top: scrollAmount + 20,
					behavior: 'smooth',
				});
				setTimeout(focusKey, 200);
			} else {
				focusKey();
			}
		}, 300);
	};

	const onRemovePair = (index: number) => {
		if (pairs.length === 1) {
			// Prevent the last pair from being removed.
			return;
		}
		let newPairs = [...pairs.slice(0, index), ...pairs.slice(index + 1, pairs.length)];
		setPairs(newPairs);
		let value = stringifyMapPairs(newPairs);
		if (value === 'map()') {
			const isAlreadyEmpty = property.value === '';
			if (isAlreadyEmpty) {
				return;
			}
			// The user has emptied all inputs so we must empty the
			// value of the mapping without automatically setting the
			// "map()" value.
			value = '';
			setIsResetting(true);
		}

		// Remove the errors related to the pair.
		const logicalErr = { ...logicalErrors };
		delete logicalErr[index];
		const validationErr = { ...validationErrors };
		delete validationErr[index];

		// Unshift the errors.
		setLogicalErrors(unshiftErrors(logicalErr, index));
		setValidationErrors(unshiftErrors(validationErr, index));

		updateMapping(propertyPath, value);
		setReloadLogicalErrors(true);
	};

	const onClearPair = () => {
		let newPairs = [['', '']] as Array<[string, string]>;
		setPairs(newPairs);
		setLogicalErrors({});
		setValidationErrors({});
		updateMapping(propertyPath, '');
		setIsResetting(true);
	};

	const typeName = toMeergoStringType(property.full.type);

	return (
		<>
			<Combobox
				onInput={updateMapping}
				value={
					automaticMapping != null ? automaticMapping : hasFilledPairs || isResetting ? '' : property.value
				}
				controlled={true}
				name={propertyPath}
				disabled={isParentMappingDisabled}
				className='action__transformation-input-property'
				size='small'
				error={hasFilledPairs || isResetting ? '' : property.error}
				autocompleteExpressions={true}
				updateError={updateMappingError}
				type={property.full.type}
				isExpression={true}
				items={mappingItems}
				onSelect={onSelect}
			/>
			<div className='action__transformation-mapping-arrow'>
				<SlIcon name='arrow-right' />
			</div>
			<div
				className={`action__transformation-output-property${
					property?.indentation! > 0 ? ' action__transformation-output-property--indented' : ''
				}`}
				style={
					{
						'--mapping-indentation': `${indentation * 30}px`,
					} as React.CSSProperties
				}
			>
				<PropertyTooltip propertyName={propertyPath} typeName={typeName} type={property.full.type}>
					<span className='action__transformation-output-property-key'>{property.full.name}</span>
					<span className='action__transformation-output-property-type'>{typeName}</span>
				</PropertyTooltip>
				{showRequired && <span className='action__transformation-output-property-required'>required</span>}
			</div>
			{pairs.map(([key, value], i) => {
				const elementType = (property.full.type as MapType).elementType;
				const hasDuplicatedKey = logicalErrors[i] != null && logicalErrors[i].endsWith('duplicated');
				return (
					<React.Fragment key={i}>
						<Combobox
							onInput={(_: string, value: string) => {
								onUpdatePair(i, 'value', value);
							}}
							value={value}
							controlled={true}
							name={String(i)}
							disabled={areChildrenMappingDisabled}
							className='action__transformation-input-property'
							size='small'
							error={
								logicalErrors[i] != null && !hasDuplicatedKey
									? logicalErrors[i]
									: validationErrors[i] != null
										? validationErrors[i]
										: ''
							}
							autocompleteExpressions={true}
							updateError={updatePairValidationError}
							type={(property.full.type as MapType).elementType}
							isExpression={true}
							items={mappingItems}
							onSelect={(_: string, value: string) => {
								onSelectPairValue(i, value);
							}}
						/>
						<div className='action__transformation-mapping-arrow'>
							<SlIcon name='arrow-right' />
						</div>
						<div
							className='action__transformation-output-property action__transformation-output-property--indented action__transformation-output-property--map'
							style={
								{
									'--mapping-indentation': `${(indentation + 1) * 30}px`,
								} as React.CSSProperties
							}
						>
							"{' '}
							<SlInput
								size='small'
								value={key}
								disabled={areChildrenMappingDisabled}
								onSlInput={(e: any) => {
									onUpdatePair(i, 'key', e.target.value);
								}}
							/>{' '}
							"
							<PropertyTooltip propertyName={''} typeName={elementType.kind} type={elementType}>
								<span className='action__transformation-output-property-type'>{elementType.kind}</span>
							</PropertyTooltip>
							<SlTooltip content='Add key'>
								<SlButton
									className='action__transformation-output-property-add'
									size='small'
									onClick={areChildrenMappingDisabled ? null : () => onAddPair(i)}
									disabled={areChildrenMappingDisabled}
								>
									<SlIcon name='plus-circle' slot='prefix' />
								</SlButton>
							</SlTooltip>
							<SlTooltip content={pairs.length === 1 ? 'Clear key' : 'Remove key'}>
								<SlButton
									className='action__transformation-output-property-remove'
									size='small'
									onClick={
										areChildrenMappingDisabled
											? null
											: pairs.length > 1
												? () => onRemovePair(i)
												: hasFilledPairs
													? () => onClearPair()
													: null
									}
									disabled={areChildrenMappingDisabled || (pairs.length === 1 && !hasFilledPairs)}
								>
									<SlIcon name='x-circle' slot='prefix' />
								</SlButton>
							</SlTooltip>
							{hasDuplicatedKey && (
								<div className='action__transformation-output-property-error'>{logicalErrors[i]}</div>
							)}
						</div>
					</React.Fragment>
				);
			})}
		</>
	);
};

const shiftErrors = (errors: Record<number, string>, afterIndex: number): Record<number, string> => {
	const err = {};
	const keys = Object.keys(errors);
	for (const k of keys) {
		const index = Number(k);
		if (index <= afterIndex) {
			err[index] = errors[index];
		} else {
			// Shift the error.
			err[index + 1] = errors[index];
		}
	}
	return err;
};

const unshiftErrors = (errors: Record<number, string>, afterIndex: number): Record<number, string> => {
	const err = {};
	const keys = Object.keys(errors);
	for (const k of keys) {
		const index = Number(k);
		if (index < afterIndex) {
			err[index] = errors[index];
		} else {
			// Unshift the error.
			err[index - 1] = errors[index];
		}
	}
	return err;
};

interface TransformationNestedPropertiesProps {
	property: Property;
	language: string;
	nesting: number;
	parentName?: string;
	side: 'input' | 'output';
	transformationType: 'mappings' | 'function' | '';
	exportMode: ExportMode;
	searchTerm: string;
	flatSchema: TransformedMapping;
	selectedPaths: string[];
	onChangeSelectedPath: (path: string) => void;
	tableKey: string | null;
}

const TransformationNestedProperties = ({
	property,
	language,
	nesting,
	parentName,
	side,
	transformationType,
	exportMode,
	searchTerm,
	flatSchema,
	selectedPaths,
	onChangeSelectedPath,
	tableKey,
}: TransformationNestedPropertiesProps) => {
	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	let path = property.name;
	let parentProperty: Property;
	const isFirstLevel = parentName == null;
	if (!isFirstLevel) {
		path = parentName + '.' + property.name;
		parentProperty = flatSchema[parentName]?.full;
	}

	let isSearched = false;
	if (searchTerm === '') {
		isSearched = true;
	} else {
		isSearched = property.name.toLowerCase().includes(searchTerm.toLowerCase());
	}

	let hasSearchedChildren = false;
	if (searchTerm === '') {
		hasSearchedChildren = true;
	} else {
		if (property.type.kind === 'object') {
			for (const key in flatSchema) {
				const isChildren = key.startsWith(`${path}.`);
				if (!isChildren) {
					continue;
				}
				const name = flatSchema[key].full.name;
				const isSearched = name.toLowerCase().includes(searchTerm.toLowerCase());
				if (isSearched) {
					hasSearchedChildren = true;
					break;
				}
			}
		} else {
			// compute the sub-properties on the fly since array and map
			// sub-properties are not already included inside the
			// schemas of the action.
			const s = flattenSchema(property.type as ArrayType | MapType);
			for (const key in s) {
				const name = s[key].full.name;
				const isSearched = name.toLowerCase().includes(searchTerm.toLowerCase());
				if (isSearched) {
					hasSearchedChildren = true;
					break;
				}
			}
		}
	}

	if (!isSearched && !hasSearchedChildren) {
		return null;
	}

	const showSearchedChildren = searchTerm !== '' && hasSearchedChildren;

	let properties: Property[] = [];
	if (property.type.kind === 'object') {
		properties = property.type.properties;
	} else {
		const t = property.type as ArrayType | MapType;
		const elementTyp = t.elementType as ObjectType;
		properties = elementTyp.properties;
	}

	let hideCheckbox = false;
	if (parentProperty != null && (parentProperty.type.kind === 'array' || parentProperty.type.kind === 'map')) {
		// direct children of an array or map property.
		hideCheckbox = true;
	} else if (
		(property.type.kind === 'array' || property.type.kind === 'map') &&
		parentName != null &&
		parentProperty == null
	) {
		// descendant of an array or map property.
		hideCheckbox = true;
	}

	return (
		<div
			className={`fullscreen-transformation__nested${isExpanded || showSearchedChildren ? ' fullscreen-transformation__nested--expand' : ''}`}
		>
			<TransformationProperty
				property={property}
				language={language}
				isParent={true}
				parentName={parentName}
				side={side}
				transformationType={transformationType}
				exportMode={exportMode}
				selectedPaths={selectedPaths}
				showCaret={hasSearchedChildren}
				onChangeSelectedPath={onChangeSelectedPath}
				isExpanded={isExpanded || showSearchedChildren}
				setIsExpanded={setIsExpanded}
				tableKey={tableKey}
				hideCheckbox={hideCheckbox}
			/>
			<div
				className='fullscreen-transformation__sub-properties'
				style={{ '--property-indentation': `${nesting * 20}px` } as React.CSSProperties}
			>
				{(isExpanded || showSearchedChildren) &&
					properties.map((p) => {
						if (isRecursiveType(p.type)) {
							return (
								<TransformationNestedProperties
									key={p.name}
									property={p}
									language={language}
									nesting={nesting + 1}
									parentName={path}
									side={side}
									transformationType={transformationType}
									exportMode={exportMode}
									searchTerm={searchTerm}
									flatSchema={flatSchema}
									selectedPaths={selectedPaths}
									onChangeSelectedPath={onChangeSelectedPath}
									tableKey={tableKey}
								/>
							);
						} else {
							return (
								<TransformationProperty
									key={p.name}
									property={p}
									language={language}
									parentName={path}
									side={side}
									transformationType={transformationType}
									exportMode={exportMode}
									searchTerm={searchTerm}
									selectedPaths={selectedPaths}
									onChangeSelectedPath={onChangeSelectedPath}
									tableKey={tableKey}
									hideCheckbox={
										property.type.kind === 'array' || property.type.kind === 'map' || hideCheckbox
									}
								/>
							);
						}
					})}
			</div>
		</div>
	);
};

interface TransformationPropertyProps {
	property: Property;
	language: string;
	isParent?: boolean;
	parentName?: string;
	side: 'input' | 'output';
	transformationType: 'mappings' | 'function' | '';
	exportMode: ExportMode;
	searchTerm?: string;
	showCaret?: boolean;
	selectedPaths: string[];
	onChangeSelectedPath: (path: string) => void;
	isExpanded?: boolean;
	setIsExpanded?: React.Dispatch<React.SetStateAction<boolean>>;
	isOutMatchingProperty?: boolean;
	tableKey: string | null;
	hideCheckbox?: boolean;
}

const TransformationProperty = ({
	property,
	language,
	isParent,
	parentName,
	side,
	transformationType,
	exportMode,
	searchTerm,
	showCaret = true,
	selectedPaths,
	onChangeSelectedPath,
	isExpanded,
	setIsExpanded,
	isOutMatchingProperty,
	tableKey,
	hideCheckbox = false,
}: TransformationPropertyProps) => {
	const { workspaces, selectedWorkspace } = useContext(AppContext);
	const { isImport, actionType, action } = useContext(ActionContext);

	let path = property.name;
	if (parentName) {
		path = parentName + '.' + property.name;
	}

	const workspace = workspaces.find((w) => w.id === selectedWorkspace);
	const isIdentifier = isImport && workspace.identifiers.includes(path) && side === 'output';
	const isSelected = selectedPaths.includes(path);
	const hasSelectedChildren = selectedPaths.findIndex((p) => p.startsWith(`${path}.`)) !== -1;
	const hasSelectedParent = selectedPaths.findIndex((p) => path.startsWith(`${p}.`)) !== -1;
	const isTableKey = !!tableKey && tableKey === path;
	const isSelectDisabled =
		transformationType === 'function' && ((isOutMatchingProperty && !isSelected) || hasSelectedParent);

	const onWrapperClick = (e: any) => {
		if (isSelectDisabled) {
			return;
		}
		const isCopy = e.target.closest('.fullscreen-transformation__property-copy') != null;
		const isCaret = e.target.closest('.fullscreen-transformation__property-caret') != null;
		const isCheckbox = e.target.closest('.fullscreen-transformation__property-check') != null;
		if (isCopy || isCaret || isCheckbox) {
			e.stopPropagation();
			return;
		}
		onChangeSelectedPath(path);
	};

	let isSearched = true;
	if (searchTerm != null && searchTerm !== '') {
		isSearched = property.name.toLowerCase().includes(searchTerm.toLowerCase());
	}

	if (!isSearched) {
		return null;
	}

	const hasRequired =
		isTableKey ||
		(actionType.target === 'Event' && (property.createRequired || property.updateRequired)) ||
		(exportMode != null &&
			((property.createRequired && exportMode.includes('Create')) ||
				(property.updateRequired && exportMode.includes('Update'))));

	let showRequired = false;
	if (hasRequired) {
		const isFirstLevel = parentName == null;
		if (isFirstLevel || isTableKey) {
			showRequired = true;
		} else {
			if (isSelected) {
				showRequired = true;
			} else {
				if (hasSelectedParent) {
					showRequired = true;
				} else {
					const selectedSiblings: string[] = [];
					for (const path of selectedPaths) {
						const hasSameParent = path.startsWith(`${parentName}.`);
						if (hasSameParent) {
							const suffix = path.slice(`${parentName}.`.length);
							const isLowerLevel = suffix.includes('.');
							if (!isLowerLevel) {
								selectedSiblings.push(path);
							}
						}
					}
					if (selectedSiblings.length > 0) {
						showRequired = true;
					}
				}
			}
		}
	}

	let typeName = '';
	if (language === '') {
		typeName = toMeergoStringType(property.type);
	} else if (language === 'Python') {
		typeName = toPythonType(property.type, action.transformation.function.preserveJSON, property.nullable);
	} else {
		typeName = toJavascriptType(property.type, action.transformation.function.preserveJSON, property.nullable);
	}

	return (
		<div
			className={`fullscreen-transformation__property-wrapper${isParent ? ' fullscreen-transformation__property-wrapper--parent' : ''}${isSelected ? ' fullscreen-transformation__property-wrapper--selected' : ''}${isOutMatchingProperty && transformationType === 'function' ? ' fullscreen-transformation__property-wrapper--is-out-matching' : ''}`}
			style={{ cursor: transformationType === 'function' ? 'pointer' : 'default' }}
			onClick={transformationType === 'function' ? onWrapperClick : null}
		>
			<div className='fullscreen-transformation__property-padding'>
				{isParent && showCaret && (
					<SlIcon
						className='fullscreen-transformation__property-caret'
						name='caret-right-fill'
						onClick={() => {
							setIsExpanded(!isExpanded);
						}}
					/>
				)}
			</div>
			{transformationType === 'function' &&
				(hideCheckbox ? (
					<div className='fullscreen-transformation__property-check-empty' />
				) : (
					<SlCheckbox
						className='fullscreen-transformation__property-check'
						checked={isSelected || hasSelectedParent}
						indeterminate={hasSelectedChildren && !isSelected}
						disabled={isSelectDisabled}
						onSlChange={() => onChangeSelectedPath(path)}
						size='small'
					/>
				))}
			<div className='fullscreen-transformation__property'>
				<div className='fullscreen-transformation__property-name'>
					{parentName != null && <span className='fullscreen-transformation__property-nested-icon' />}
					{isIdentifier && (
						<SlTooltip content='Used as identifier'>
							<SlIcon
								className='fullscreen-transformation__property-identifier-icon'
								name='person-check'
							/>
						</SlTooltip>
					)}
					<PropertyTooltip propertyName={path} typeName={typeName} type={property.type}>
						<span className='fullscreen-transformation__property-name-text'>{property.name}</span>
						<span className='fullscreen-transformation__property-type'>
							<span>{typeName}</span>
							{side === 'input' && property.readOptional && <span>- optional</span>}
							{showRequired && (
								<span className='fullscreen-transformation__property-required'>required</span>
							)}
						</span>
					</PropertyTooltip>
					{!isOutMatchingProperty && (
						<SlCopyButton
							className='fullscreen-transformation__property-copy'
							value={parentName ? `${parentName}.${property.name}` : property.name}
							copyLabel='Click to copy'
							successLabel='✓ Copied'
							errorLabel='Copying to clipboard is not supported by your browser'
						/>
					)}
					{transformationType === 'function' && isOutMatchingProperty && !isSelected && (
						<SlTooltip content='You cannot select this property since it is already used as matching property'>
							<SlIcon className='fullscreen-transformation__property-disabled-info' name='info-circle' />
						</SlTooltip>
					)}
					{transformationType === 'function' && isOutMatchingProperty && isSelected && (
						<div className='fullscreen-transformation__property-error'>
							Ensure that this property is not returned by the transformation function, and then deselect
							this
						</div>
					)}
				</div>
			</div>
		</div>
	);
};

interface TypeTooltipProps {
	propertyName: string;
	typeName: string;
	type: Type;
	children: ReactNode;
}

const PropertyTooltip = ({ propertyName, typeName, type, children }: TypeTooltipProps) => {
	return (
		<SlTooltip className='type-tooltip' placement='top-start' distance={5}>
			<div slot='content'>
				<div className='type-tooltip__title'>
					<span className='type-tooltip__property-name'>{propertyName}</span>{' '}
					<span className='type-tooltip__type-name'>{typeName}</span>
				</div>
				{typeDescription(type)}
			</div>
			{children}
		</SlTooltip>
	);
};

function buildFilteredSample(flatSchema: TransformedMapping, sample: Sample): Record<string, any> {
	const result: Record<string, any> = {};
	const keys = Object.keys(flatSchema);
	for (const key of keys) {
		if (flatSchema[key].type === 'object') {
			continue;
		}
		const value = getValueFromPath(sample, key);
		if (value !== undefined) {
			setValueAtPath(result, key, value);
		}
	}
	return result;
}

function getValueFromPath(obj: any, path: string): any {
	const keys = path.split('.');
	let current = obj;
	for (const key of keys) {
		if (current == null || typeof current !== 'object') {
			return undefined;
		}
		current = current[key];
	}
	return current;
}

function setValueAtPath(obj: any, path: string, value: any): void {
	const keys = path.split('.');
	let current = obj;
	let i = 0;
	for (const k of keys) {
		if (i === keys.length - 1) {
			current[k] = value;
		} else {
			current[k] = current[k] || {};
			current = current[k];
		}
		i++;
	}
}

function getSelectedChildrenProperties(
	parentPath: string,
	selectedPaths: string[],
	value: Record<string, any>,
): Record<string, any> {
	let props: Record<string, any> = {};
	for (const k in value) {
		props[k] = false;
		const v = value[k];
		const path = `${parentPath}.${k}`;
		if (typeof v === 'object') {
			if (selectedPaths.includes(path)) {
				props[k] = true;
			} else {
				const p = getSelectedChildrenProperties(path, selectedPaths, v);
				if (Object.keys(p).length > 0) {
					props[k] = true;
				}
			}
		} else {
			if (selectedPaths.includes(path)) {
				props[k] = true;
			}
		}
	}
	return props;
}

function isElementVisibleInLeftPanel(element: Element, container: Element) {
	const elementRect = element.getBoundingClientRect();
	const containerRect = container.getBoundingClientRect();

	const elementTop = elementRect.top;
	const elementBottom = elementRect.bottom;
	const containerTop = containerRect.top + container.scrollTop;
	const containerBottom = containerTop + container.clientHeight;

	const isVerticallyVisible = elementTop >= containerTop && elementBottom <= containerBottom;
	return isVerticallyVisible;
}

function typeDescription(type: Type): ReactNode[] {
	let elements: ReactNode[] = [
		<div>
			A Meergo {meergoTypeDescription(type)}
			{type.kind !== 'decimal' && type.kind !== 'object' && type.kind !== 'array' && type.kind !== 'map'
				? ' ' + 'type'
				: ''}
			.
		</div>,
	];
	if (type.kind === 'int' || type.kind === 'uint' || type.kind === 'float') {
		if (type.minimum != null) {
			elements.push(<div>Minimum: {type.minimum}</div>);
		}
		if (type.maximum != null) {
			elements.push(<div>Maximum: {type.maximum}</div>);
		}
		if (type.kind === 'float' && type.real != null) {
			elements.push(<div>Real: {type.real}</div>);
		}
	} else if (type.kind === 'decimal') {
		if (type.minimum != null) {
			elements.push(<div>Minimum: {type.minimum}</div>);
		}
		if (type.maximum != null) {
			elements.push(<div>Maximum: {type.maximum}</div>);
		}
	} else if (type.kind === 'year') {
		elements.push(<div>Minimum: 1</div>);
		elements.push(<div>Maximum: 9999</div>);
	} else if (type.kind === 'text') {
		if (type.values != null) {
			elements.push(<div>Values: {type.values.join(', ')}</div>);
		}
		if (type.regexp != null) {
			elements.push(<div>Regular expression: {type.regexp}</div>);
		}
		if (type.byteLen != null) {
			elements.push(<div>Max bytes: {type.byteLen}</div>);
		}
		if (type.charLen != null) {
			elements.push(<div>Max characters: {type.charLen}</div>);
		}
	} else if (type.kind === 'array' || type.kind === 'map') {
		const elementTypeDescription = typeDescription(type.elementType);
		elements = [...elements, ...elementTypeDescription.slice(1)];
	}
	return elements;
}

function meergoTypeDescription(type: Type): ReactNode {
	let description = <code>{type.kind}</code>;
	if (type.kind === 'int' || type.kind === 'uint' || type.kind === 'float') {
		description = (
			<>
				{type.bitSize}-bit <code>{type.kind}</code>
			</>
		);
	} else if (type.kind === 'decimal') {
		description = (
			<>
				<code>{type.kind}</code> type with precision {type.precision} and scale {type.scale}
			</>
		);
	} else if (type.kind === 'array' || type.kind === 'map') {
		description = (
			<code>
				{type.kind} of {meergoTypeDescription(type.elementType)}
			</code>
		);
	}
	return description;
}

function toMeergoStringType(type: Type) {
	if (type.kind === 'int' || type.kind === 'uint' || type.kind === 'float') {
		return `${type.kind}(${type.bitSize})`;
	} else if (type.kind === 'decimal') {
		return `decimal(${type.precision}, ${type.scale})`;
	} else if (type.kind === 'array' || type.kind === 'map') {
		return `${type.kind} of ${toMeergoStringType(type.elementType)}`;
	}
	return type.kind;
}

function toJavascriptType(type: Type, preserveJSON: boolean, nullable?: boolean) {
	let t: string;

	const kind = type.kind;
	switch (kind) {
		case 'boolean':
			t = 'boolean';
			break;
		case 'int':
		case 'uint':
			if (type.bitSize === 64) {
				t = 'bigint';
			} else {
				t = `number (${kind})`;
			}
			break;
		case 'float':
			t = 'number';
			break;
		case 'decimal':
			t = 'string';
			break;
		case 'datetime':
		case 'date':
		case 'time':
			t = 'Date';
			break;
		case 'year':
			t = 'number';
			break;
		case 'uuid':
			t = 'string';
			break;
		case 'json':
			if (preserveJSON) {
				t = 'string (JSON)';
			} else {
				t = 'any';
			}
			break;
		case 'inet':
			t = 'string';
			break;
		case 'text':
			t = 'string';
			break;
		case 'array':
			const arrayElementType = toJavascriptType(type.elementType, preserveJSON);
			t = `${arrayElementType}[]`;
			break;
		case 'object':
			t = 'object';
			break;
		case 'map':
			const mapElementType = toJavascriptType(type.elementType, preserveJSON);
			t = `object with ${mapElementType} values`;
			break;
		default:
			throw new Error(`schema contains unknown property kind ${kind}`);
	}

	if (nullable) {
		t += ' | null';
	}

	return t;
}

function toPythonType(type: Type, preserveJSON: boolean, nullable?: boolean) {
	let t: string;

	const kind = type.kind;
	switch (kind) {
		case 'boolean':
			t = 'bool';
			break;
		case 'int':
		case 'uint':
			t = 'int';
			break;
		case 'float':
			t = 'float';
			break;
		case 'decimal':
			t = 'decimal.Decimal';
			break;
		case 'datetime':
			t = 'datetime.datetime';
			break;
		case 'date':
			t = 'datetime.date';
			break;
		case 'time':
			t = 'datetime.time';
			break;
		case 'year':
			t = 'int';
			break;
		case 'uuid':
			t = 'uuid.UUID';
			break;
		case 'json':
			if (preserveJSON) {
				t = 'str (JSON)';
			} else {
				t = 'Any';
			}
			break;
		case 'inet':
			t = 'str';
			break;
		case 'text':
			t = 'str';
			break;
		case 'array':
			const arrayElementType = toPythonType(type.elementType, preserveJSON);
			t = `list[${arrayElementType}]`;
			break;
		case 'object':
			t = 'dict';
			break;
		case 'map':
			const mapElementType = toPythonType(type.elementType, preserveJSON);
			t = `dict[str, ${mapElementType}]`;
			break;
		default:
			throw new Error(`schema contains unknown property kind ${kind}`);
	}

	if (nullable) {
		t += ' | None';
	}

	return t;
}

function removeQuotes(v: any | null) {
	if (v == null) {
		return null;
	}
	if (typeof v !== 'string') {
		return v;
	}
	return v.replace(/^"|"$/g, '');
}

// transformationHeaders returns two pairs of values, the two values ​​to use
// for the left transformation header, and the two values ​​for the right.
//
// The two values ​​in each pair form a meaningful header and can, for example,
// be displayed on two separate lines or concatenated with a space.
function transformationHeaders(
	connection: TransformedConnection,
	action: TransformedAction,
): [Array<string>, Array<string>] {
	let leftHeader: Array<string>;
	let rightHeader: Array<string>;
	let terms = connection.connector.terms;
	if (connection.isSource) {
		if (connection.isEventBased) {
			leftHeader = ['Input event', `from ${connection.connector.name}`];
		} else if (connection.isFileStorage) {
			leftHeader = [`Input ${terms.user}`, `from ${action.format}`];
		} else {
			leftHeader = [`Input ${terms.user}`, `from ${connection.connector.name}`];
		}
		rightHeader = ['Output user', 'to warehouse'];
	} else {
		if (action.target == 'Event') {
			leftHeader = ['Input event', 'from source connections'];
			rightHeader = ['Output event', `to ${connection.connector.name}`];
		} else {
			leftHeader = ['Input user', 'from warehouse'];
			rightHeader = [`Output ${terms.user}`, `to ${connection.connector.name}`];
		}
	}
	return [leftHeader, rightHeader];
}

export default ActionTransformation;

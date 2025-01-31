import React, { useState, useRef, useContext, useEffect, forwardRef, useMemo, ReactNode } from 'react';
import { checkIfPropertyExists, updateMappingProperty, getSampleIdentifiers } from './Action.helpers';
import {
	getSchemaComboboxItems,
	getIdentityColumnComboboxItems,
	getLastChangeTimeComboboxItems,
} from '../../helpers/getSchemaComboboxItems';
import {
	TransformedAction,
	TransformedActionType,
	TransformedMapping,
	doesLastChangeTimeColumnNeedFormat,
	flattenSchema,
	getTransformationFunctionParameterName,
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
import Type, { ObjectType, Property } from '../../../lib/api/types/types';
import { EventListenerEvent } from '../../../hooks/useEventListener';
import { Sample } from './Action.types';
import { UnprocessableError } from '../../../lib/api/errors';
import ConnectionContext from '../../../context/ConnectionContext';
import Workspace from '../../../lib/api/types/workspace';
import { ActionToSet, ExportMode, TransformationFunction, TransformationPurpose } from '../../../lib/api/types/action';
import { debounceWithAbort } from '../../../utils/debounce';
import TransformedConnector from '../../../lib/core/connector';
import { Combobox } from '../../base/Combobox/Combobox';
import { ComboboxItem } from '../../base/Combobox/Combobox.types';
import JSONbig from 'json-bigint';
import actionContext from '../../../context/ActionContext';

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
		isTransformationFunctionSupported,
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
	const lastChangeTimeCustomFormatInputRef = useRef(null);
	const sharedMapping = useRef<TransformedMapping>();

	const hasIdentityColumns = useMemo(() => {
		return (
			connection.isSource && (connection.isDatabase || connection.isFileStorage) && actionType.target === 'Users'
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
			sharedMapping.current = action.transformation.mapping;
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

	useEffect(() => {
		sharedMapping.current = { ...action.transformation.mapping };
	}, [action.transformation.mapping]);

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

	const flatSchema = useMemo<TransformedMapping>(() => {
		return flattenSchema(actionType.inputSchema);
	}, [actionType]);

	const identityColumnError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			if (action.identityColumn === '' && !isFirstCompilation.current) {
				return 'The user identifier cannot be empty';
			}
			return checkIfPropertyExists(action.identityColumn, flatSchema);
		}
	}, [action, flatSchema]);

	const lastChangeTimeColumnError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			return checkIfPropertyExists(action.lastChangeTimeColumn, flatSchema);
		}
	}, [action, flatSchema]);

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

	const updateMapping = async (name: string, value: string, signal?: AbortSignal) => {
		let errorMessage = '';
		if (value !== '') {
			try {
				errorMessage = await api.validateExpression(
					value,
					actionType.inputSchema.properties,
					action.transformation.mapping![name].full.type,
					signal,
				);
			} catch (err) {
				if (err.name === 'AbortError') {
					return;
				}
				handleError(err);
				return;
			}
		}
		let doesNotExist = errorMessage.endsWith('does not exist');
		const isEventBasedUserImport = connection.isEventBased && connection.isSource && actionType.target === 'Users';
		const isAppEventsExport = connection.isApp && connection.isDestination && actionType.target === 'Events';
		if (doesNotExist) {
			if (isEventBasedUserImport) {
				errorMessage += `, perhaps you meant "traits.${value}"?`;
			} else if (isAppEventsExport) {
				errorMessage += `, perhaps you meant "properties.${value}" or "traits.${value}"?`;
			}
		}
		const updatedAction = updateMappingProperty(action, name, value, errorMessage);
		setAction(updatedAction);
	};

	const debouncedUpdateMapping = useMemo(() => debounceWithAbort(updateMapping, 500), [actionType, action]);

	const onUpdateMapping = async (name: string, value: string) => {
		debouncedUpdateMapping(name, value);
	};

	const onSelectProperty = async (name: string, value: string) => {
		if (name === 'identityColumn') {
			const a = { ...action };
			a.identityColumn = value;
			setAction(a);
			if (isFirstCompilation.current) {
				isFirstCompilation.current = false;
			}
			return;
		} else if (name === 'lastChangeTimeColumn') {
			const a = { ...action };
			a.lastChangeTimeColumn = value;
			if (value === '' || !doesLastChangeTimeColumnNeedFormat(value, actionType.inputSchema)) {
				a.lastChangeTimeFormat = '';
			}
			setAction(a);
			return;
		}
		await updateMapping(name, value);
	};

	const onUpdateIdentityColumn = async (_: string, value: string) => {
		const a = { ...action };
		a.identityColumn = value;
		setAction(a);
		if (isFirstCompilation.current) {
			isFirstCompilation.current = false;
		}
	};

	const onUpdateLastChangeTimeColumn = async (_: string, value: string) => {
		const a = { ...action };
		a.lastChangeTimeColumn = value;
		if (value === '' || !doesLastChangeTimeColumnNeedFormat(value, actionType.inputSchema)) {
			setIsCustomLastChangeTimeFormatSelected(false);
			a.lastChangeTimeFormat = '';
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
			sharedMapping={sharedMapping}
			transformationType={transformationType}
			setTransformationType={setTransformationType}
			workspaces={workspaces}
			selectedWorkspace={selectedWorkspace}
			action={action}
			setAction={setAction}
			onUpdateMapping={onUpdateMapping}
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
			isTransformationFunctionSupported={isTransformationFunctionSupported}
		/>
	);

	return (
		<div
			className={`action__transformation${isTransformationDisabled ? ' action__transformation--disabled' : ''}`}
			ref={ref}
		>
			{hasIdentityColumns && (
				<Section
					title='Identity columns'
					description='The columns from which to import the value to uniquely identify a user identity, and possibly the time of its last modification.'
					padded={true}
					annotated={true}
				>
					<div className='action__transformation-identity-columns'>
						<div className='action__transformation-identity-column'>
							<div className='action__transformation-identity-columns-label'>
								Identity
								<span className='action__transformation-identity-columns-asterisk'>*</span>:
							</div>
							<Combobox
								onInput={onUpdateIdentityColumn}
								onSelect={onUpdateIdentityColumn}
								name='identityColumn'
								value={identityColumnList.length === 0 ? '' : action.identityColumn!}
								disabled={isTransformationDisabled || identityColumnList.length === 0}
								className='action__transformation-input-property'
								isExpression={false}
								items={identityColumnList}
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
								<div className='action__transformation-identity-columns-label'>Last change time:</div>
								<Combobox
									onInput={onUpdateLastChangeTimeColumn}
									onSelect={onUpdateLastChangeTimeColumn}
									value={action.lastChangeTimeColumn!}
									name='lastChangeTimeColumn'
									disabled={isTransformationDisabled}
									className='action__transformation-input-property'
									isExpression={false}
									caret={true}
									items={lastChangeTimeList}
									clearable={action.lastChangeTimeColumn?.length > 0}
									error={lastChangeTimeColumnError}
									size='small'
									helpText='A column with the time of the last modification of a user identity'
								/>
							</div>
							<div className='action__transformation-last-change-format-property'>
								<div className='action__transformation-last-change-format'>
									<div className='action__transformation-identity-columns-label'>Format:</div>
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
										disabled={!needFormat}
										size='small'
									>
										<SlOption value='iso8601'>ISO 8601</SlOption>
										{format?.name === 'Excel' && <SlOption value='excel'>Excel</SlOption>}
										<SlOption value='custom'>Custom...</SlOption>
									</SlSelect>
								</div>
								{needFormat && isCustomLastChangeTimeFormatSelected && (
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
						</div>
					</div>
				</Section>
			)}
			<Section
				title='Transformation'
				description='The relation between the event properties and the action type properties'
				padded={false}
				annotated={true}
			>
				{box}
				<FullscreenTransformation
					isFullscreenTransformationOpen={isFullscreenTransformationOpen}
					selectedLanguage={selectedLanguage}
					body={box}
					inputSchema={actionType.inputSchema}
					outputSchema={actionType.outputSchema}
				/>
			</Section>
		</div>
	);
});

interface TransformationBoxProps {
	sharedMapping: React.MutableRefObject<TransformedMapping>;
	transformationType: 'mappings' | 'function' | '';
	setTransformationType: React.Dispatch<React.SetStateAction<'mappings' | 'function' | ''>>;
	workspaces: Workspace[];
	selectedWorkspace: number;
	action: TransformedAction;
	setAction: React.Dispatch<React.SetStateAction<TransformedAction>>;
	onUpdateMapping: (name: string, value: string) => void;
	mappingItems: ComboboxItem[];
	onSelectMappingItem: (name: string, value: string) => void;
	isTransformationDisabled: boolean;
	transformationLanguages: string[];
	selectedLanguage: string;
	setSelectedLanguage: React.Dispatch<React.SetStateAction<string>>;
	onOpenFullscreenTransformation: () => void;
	onChangeTransformationFunction: (source: string) => void;
	isFullscreenTransformationOpen: boolean;
	onCloseFullscreenTransformation: () => void;
	actionType: TransformedActionType;
	isTransformationFunctionSupported: boolean;
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
	sharedMapping,
	transformationType,
	setTransformationType,
	workspaces,
	selectedWorkspace,
	action,
	setAction,
	mappingItems,
	onSelectMappingItem,
	onUpdateMapping,
	isTransformationDisabled,
	transformationLanguages,
	selectedLanguage,
	setSelectedLanguage,
	onOpenFullscreenTransformation,
	onChangeTransformationFunction,
	isFullscreenTransformationOpen,
	onCloseFullscreenTransformation,
	actionType,
	isTransformationFunctionSupported,
}: TransformationBoxProps) => {
	const [isAlertOpen, setIsAlertOpen] = useState<boolean>(false);
	const [isCompletelyOpen, setIsCompletelyOpen] = useState<boolean>(false);
	const [isFullscreenAnimating, setIsFullscreenAnimating] = useState<boolean>(false);
	const [isEditTooltipOpen, setIsEditTooltipOpen] = useState<boolean>();

	const pendingTransformationType = useRef<string>();
	const firstValue = useRef<TransformedMapping | TransformationFunction>();
	const hasNeverChangedTransformationType = useRef<boolean>(true);

	const { connection } = useContext(ConnectionContext);
	const { setSelectedInPaths, setSelectedOutPaths, isEditing } = useContext(actionContext);

	useEffect(() => {
		if (transformationType === 'mappings') {
			firstValue.current = JSON.parse(JSON.stringify(action.transformation.mapping));
		} else {
			firstValue.current = JSON.parse(JSON.stringify(action.transformation.function));
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

	const onEditorMount = (editor) => {
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
				sharedMapping.current = { ...a.transformation.mapping };
				a.transformation.function = null;
				setSelectedLanguage('');
				setSelectedInPaths([]);
				setSelectedOutPaths([]);
				setAction(a);
				setTransformationType('mappings');
			} else {
				a.transformation.mapping = null;
				sharedMapping.current = null;
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
				setAction(a);
				setTransformationType('function');
			}
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
		for (const k in action.transformation.mapping) {
			const isOutMatchingProperty = action.matching?.out && action.matching.out === k;

			const property = action.transformation.mapping[k];

			const hasRequired =
				action.exportMode != null &&
				((property.createRequired && action.exportMode.includes('Create')) ||
					(property.updateRequired && action.exportMode.includes('Update')));

			let showRequired = false;
			if (hasRequired) {
				const isFirstLevel = property.indentation === 0;
				if (isFirstLevel) {
					showRequired = true;
				} else {
					if (property.value !== '') {
						showRequired = true;
					} else {
						const keys = Object.keys(action.transformation.mapping);

						const parents: string[] = [];
						for (const key of keys) {
							if (k.startsWith(`${key}.`)) {
								parents.push(key);
							}
						}

						const hasMappedParent =
							parents.findIndex(
								(k) =>
									action.transformation.mapping[k].value !== '' &&
									action.transformation.mapping[k].error === '',
							) !== -1;
						if (hasMappedParent) {
							showRequired = true;
						} else {
							const siblings: string[] = [];
							for (const key of keys) {
								const prop = action.transformation.mapping[key];
								if (
									prop.root === property.root &&
									prop.indentation === property.indentation &&
									key !== k
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

			const showMatchingIn = isOutMatchingProperty && property.value === '';
			mappings.push(
				<div
					key={k}
					className='action__transformation-mapping'
					data-key={k}
					style={
						{
							'--mapping-indentation': `${action.transformation.mapping![k].indentation! * 30}px`,
						} as React.CSSProperties
					}
				>
					<Combobox
						onInput={onUpdateMapping}
						value={showMatchingIn ? action.matching.in : property.value}
						sharedMapping={showMatchingIn ? null : sharedMapping}
						controlled={showMatchingIn}
						name={k}
						disabled={
							isTransformationDisabled ||
							property.disabled === true ||
							(isOutMatchingProperty && property.value === '')
						}
						className='action__transformation-input-property'
						size='small'
						error={
							isOutMatchingProperty && property.value !== ''
								? 'Please leave this input empty, as the mapping is automatic in this case'
								: property.error
						}
						autocompleteExpressions={true}
						isExpression={true}
						items={mappingItems}
						onSelect={onSelectMappingItem}
					>
						{workspace.identifiers.includes(k) && (
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
					>
						<span className='action__transformation-output-property-key'>{k}</span>
						<span className='action__transformation-output-property-type'>{property.type}</span>
						{showRequired && (
							<span className='action__transformation-output-property-required'>required</span>
						)}
					</div>
				</div>,
			);
		}
		body = <div className='action__transformation-mappings'>{mappings}</div>;
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
				<div className='transformation-box__header-title'>
					{isCompletelyOpen || !isTransformationFunctionSupported || transformationLanguages.length == 0 ? (
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
							{transformationLanguages.map((language) => {
								return (
									<SlButton
										key={language}
										variant={
											transformationType === 'function' && selectedLanguage === language
												? 'primary'
												: 'default'
										}
										onClick={() => onTransformationTypeClick(language)}
										disabled={isTransformationDisabled}
									>
										{language}
									</SlButton>
								);
							})}
						</SlButtonGroup>
					)}
				</div>
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
			<div className='transformation-box__body'>{body}</div>
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
	inputSchema: ObjectType;
	outputSchema: ObjectType;
}

const FullscreenTransformation = ({
	isFullscreenTransformationOpen,
	selectedLanguage,
	body,
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

	const collectEvents = (newly: EventListenerEvent[]) => {
		setEvents((prevEvents) => [...prevEvents, ...newly]);
	};

	const { isEventBasedUserImport, isAppEventsExport } = useMemo(() => {
		return {
			isEventBasedUserImport: connection.isEventBased && connection.isSource && actionType.target === 'Users',
			isAppEventsExport: connection.isApp && connection.isDestination && actionType.target === 'Events',
		};
	}, [connection, actionType]);

	const { flatInputSchema, flatOutputSchema } = useMemo(() => {
		return {
			flatInputSchema: flattenSchema(inputSchema),
			flatOutputSchema: flattenSchema(outputSchema),
		};
	}, [inputSchema, outputSchema]);

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
		setInSearchTerm('');
		setOutSearchTerm('');
	}, [transformationType]);

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
			if (actionType.target !== 'Users') {
				return;
			}
			if (!isFullscreenTransformationOpen || hasAlreadyFetchedSamples.current) {
				return;
			}
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
					handleError(err);
					return;
				}
				samples = res.records;
			} else if (connection.isDatabase && connection.isSource) {
				// Will show a button to execute the query and retrieve the
				// samples (as the query can be potentially destructive).
				return;
			} else if (connection.isApp && connection.isSource) {
				let res: AppUsersResponse;
				try {
					res = await api.workspaces.connections.appUsers(connection.id, inputSchema);
				} catch (err) {
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
				return;
			}
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

	const onChangeShowOnlyInSelected = () => {
		setShowOnlyInSelected(!showOnlyInSelected);
	};

	const onChangeShowOnlyOutSelected = () => {
		setShowOnlyOutSelected(!showOnlyOutSelected);
	};

	const onChangeSelectedPath = (side: 'in' | 'out', path: string) => {
		let paths;
		let schema;
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
				setOutputError(err.message);
				setIsExecuting(false);
			}, 300);
			return;
		}

		let inSchema = actionToSet.inSchema;

		// Only send the sample's properties that are actually present in the
		// input schema of the "ActionToSet".
		let s = {};
		for (const k in sample) {
			const isInSchema = actionToSet.inSchema.properties.findIndex((prop) => prop.name === k) !== -1;
			if (isInSchema) {
				s[k] = sample[k];
			}
		}

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
				setOutputError(err.message);
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
				setOutputError(err.message);
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
						onSlChange={onChangeShowOnlyInSelected}
					>
						Show only selected properties
					</SlSwitch>
				)}
				{inputSchema.properties.map((p) => {
					if (transformationType === 'function') {
						const isSelected = selectedInPaths.includes(p.name);
						const hasSelectedChildren =
							selectedInPaths.findIndex((prop) => prop.startsWith(`${p.name}.`)) !== -1;
						if (showOnlyInSelected && !isSelected && !hasSelectedChildren) {
							return null;
						}
					}

					if (p.type.name === 'Object') {
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
							/>
						);
					}
				})}
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
										{actionType.target === 'Users' ? (
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
					<SlIcon name='x-lg' />
					<p className='fullscreen-transformation__no-sample-text'>
						This connection cannot retrieve events for testing the transformation because no event source
						has been added. Please add an event source in the connection settings to start collecting events
						and enable transformation testing.
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
												<div className='fullscreen-transformation__event-name'>{e.type}</div>
												<div className='fullscreen-transformation__event-time'>
													{new Date(e.time).toLocaleString()}
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
	} else if (connection.isDestination && actionType.target === 'Users') {
		inputPanelContent = (
			<div className='fullscreen-transformation__no-sample'>
				<SlIcon name='x-lg' />
				<p className='fullscreen-transformation__no-sample-text'>
					Users cannot be retrieved to test the transformation because no users have been imported into the
					warehouse yet. Please import some users into the warehouse to enable transformation testing.
				</p>
			</div>
		);
	} else {
		inputPanelContent = (
			<div className='fullscreen-transformation__no-sample'>
				<SlIcon name='x-lg' />
				<p className='fullscreen-transformation__no-sample-text'>
					This connection cannot retrieve samples to test the transformation
				</p>
			</div>
		);
	}

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
									<div className='fullscreen-transformation__panel-title-text'>Input</div>
									<div className='fullscreen-transformation__panel-sub-title'>{`from ${connection.isSource ? connection.connector.name : 'warehouse'}`}</div>
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
							<div className='fullscreen-transformation__panel-title-text'>Output</div>
							<div className='fullscreen-transformation__panel-sub-title'>{`to ${connection.isDestination ? connection.connector.name : 'warehouse'}`}</div>
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
								{connection.isDestination && actionType.target === 'Events' ? 'Preview' : 'Result'}
							</SlButton>
						</SlButtonGroup>
					</div>
					<div className='fullscreen-transformation__panel-content'>
						{isOutputSchemaSelected ? (
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
										onSlChange={onChangeShowOnlyOutSelected}
									>
										Show only selected properties
									</SlSwitch>
								)}
								{outputSchema.properties.map((p) => {
									if (transformationType === 'function') {
										const isSelected = selectedOutPaths.includes(p.name);
										const hasSelectedChildren =
											selectedOutPaths.findIndex((prop) => prop.startsWith(`${p.name}.`)) !== -1;
										if (showOnlyOutSelected && !isSelected && !hasSelectedChildren) {
											return null;
										}
									}

									if (p.type.name === 'Object') {
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
											/>
										);
									}
								})}
							</div>
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
										actionType.target === 'Events' ? (
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
}: TransformationNestedPropertiesProps) => {
	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	const typ = property.type as ObjectType;

	let path = property.name;
	if (parentName) {
		path = parentName + '.' + property.name;
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
	}

	if (!isSearched && !hasSearchedChildren) {
		return null;
	}

	const showSearchedChildren = searchTerm !== '' && hasSearchedChildren;

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
			/>
			<div
				className='fullscreen-transformation__sub-properties'
				style={{ '--property-indentation': `${nesting * 20}px` } as React.CSSProperties}
			>
				{(isExpanded || showSearchedChildren) &&
					typ.properties.map((p) => {
						if (p.type.name === 'Object') {
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
}: TransformationPropertyProps) => {
	const { workspaces, selectedWorkspace } = useContext(AppContext);

	let path = property.name;
	if (parentName) {
		path = parentName + '.' + property.name;
	}

	const workspace = workspaces.find((w) => w.id === selectedWorkspace);
	const isIdentifier = workspace.identifiers.includes(path);
	const isSelected = selectedPaths.includes(path);
	const hasSelectedChildren = selectedPaths.findIndex((p) => p.startsWith(`${path}.`)) !== -1;
	const hasSelectedParent = selectedPaths.findIndex((p) => path.startsWith(`${p}.`)) !== -1;

	let isSearched = true;
	if (searchTerm != null && searchTerm !== '') {
		isSearched = property.name.toLowerCase().includes(searchTerm.toLowerCase());
	}

	if (!isSearched) {
		return null;
	}

	const hasRequired =
		exportMode != null &&
		((property.createRequired && exportMode.includes('Create')) ||
			(property.updateRequired && exportMode.includes('Update')));

	let showRequired = false;
	if (hasRequired) {
		const isFirstLevel = parentName == null;
		if (isFirstLevel) {
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

	return (
		<div
			className={`fullscreen-transformation__property-wrapper${isParent ? ' fullscreen-transformation__property-wrapper--parent' : ''}${isSelected ? ' fullscreen-transformation__property-wrapper--selected' : ''}${isOutMatchingProperty && transformationType === 'function' ? ' fullscreen-transformation__property-wrapper--is-out-matching' : ''}`}
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
			{transformationType === 'function' && (
				<SlCheckbox
					className='fullscreen-transformation__property-check'
					checked={isSelected || hasSelectedParent}
					indeterminate={hasSelectedChildren && !isSelected}
					disabled={(isOutMatchingProperty && !isSelected) || hasSelectedParent}
					onSlChange={() => onChangeSelectedPath(path)}
					size='small'
				/>
			)}
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
					<span
						className='fullscreen-transformation__property-name-text'
						style={{ cursor: transformationType === 'function' ? 'pointer' : 'default' }}
						onClick={transformationType === 'function' ? () => onChangeSelectedPath(path) : null}
					>
						{property.name}
					</span>
					<span className='fullscreen-transformation__property-type'>
						<span>
							{language === ''
								? property.type.name
								: language === 'Python'
									? toPythonType(property.type, property.nullable)
									: toJavascriptType(property.type, property.nullable)}
						</span>
						{side === 'input' && property.readOptional && <span>- optional</span>}
						{showRequired && <span className='fullscreen-transformation__property-required'>required</span>}
					</span>
					{!isParent && (transformationType === 'mappings' || !isOutMatchingProperty) && (
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

function toJavascriptType(type: Type, nullable: boolean) {
	// TODO: add additional information (property values, property
	// length).
	const name = type.name;

	let t: string;
	switch (name) {
		case 'Boolean':
			t = 'boolean';
			break;
		case 'Int':
		case 'Uint':
			if (type.bitSize === 8 || type.bitSize === 16 || type.bitSize === 24 || type.bitSize === 32) {
				t = 'number';
			} else {
				t = 'bigint';
			}
			break;
		case 'Float':
			t = 'number';
			break;
		case 'Decimal':
			t = 'string';
			break;
		case 'DateTime':
		case 'Date':
		case 'Time':
		case 'Year':
			t = 'Date';
			break;
		case 'UUID':
			t = 'string';
			break;
		case 'JSON':
			t = 'string';
			break;
		case 'Inet':
			t = 'string';
			break;
		case 'Text':
			t = 'string';
			break;
		case 'Array':
			t = 'Array';
			break;
		case 'Object':
			t = 'Object';
			break;
		case 'Map':
			t = 'Map';
			break;
		default:
			console.error(`schema contains unknow property type ${name}`);
			'unknown property type';
	}

	if (nullable) {
		t += ' | null';
	}

	return t;
}

function toPythonType(type: Type, nullable: boolean) {
	// TODO: add additional information (property values, property
	// length).

	let t: string;
	switch (type.name) {
		case 'Boolean':
			t = 'bool';
			break;
		case 'Int':
		case 'Uint':
			t = 'int';
			break;
		case 'Float':
			t = 'float';
			break;
		case 'Decimal':
			t = 'decimal.Decimal';
			break;
		case 'DateTime':
			t = 'datetime.datetime';
			break;
		case 'Date':
			t = 'datetime.date';
			break;
		case 'Time':
			t = 'datetime.time';
			break;
		case 'Year':
			t = 'int';
			break;
		case 'UUID':
			t = 'uuid.UUID';
			break;
		case 'JSON':
			t = 'str';
			break;
		case 'Inet':
			t = 'str';
			break;
		case 'Text':
			t = 'str';
			break;
		case 'Array':
			t = 'list';
			break;
		case 'Object':
			t = 'dict';
			break;
		case 'Map':
			t = 'dict';
			break;
		default:
			console.error(`schema contains unknow property type ${type}`);
			return 'unknown property type';
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

export default ActionTransformation;

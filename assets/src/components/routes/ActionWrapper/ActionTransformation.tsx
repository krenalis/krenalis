import React, { useState, useRef, useContext, useEffect, forwardRef, useMemo, ReactNode } from 'react';
import { checkIfPropertyExists, updateMappingProperty, getSampleIdentifiers } from './Action.helpers';
import {
	getSchemaComboboxItems,
	getIdentityPropertyComboboxItems,
	getLastChangeTimeComboboxItems,
} from '../../helpers/getSchemaComboboxItems';
import {
	TransformedAction,
	TransformedActionType,
	TransformedMapping,
	doesLastChangeTimePropertyNeedFormat,
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
	EventPreviewResponse,
	ExecQueryResponse,
	FindUsersResponse,
	RecordsResponse,
	TransformationLanguagesResponse,
	TransformDataResponse,
} from '../../../lib/api/types/responses';
import getLanguageLogo from '../../helpers/getLanguageLogo';
import Type, { ObjectType, Property, typeNameToIconName } from '../../../lib/api/types/types';
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
		mode,
		setMode,
		setIsSaveHidden,
		isFormatChanged,
		isEditing,
	} = useContext(ActionContext);

	const isFirstCompilation = useRef(true);
	const lastChangeTimeCustomFormatInputRef = useRef(null);

	const hasIdentityAndTimestamp = useMemo(() => {
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
			setMode('transformation');
			setSelectedLanguage(action.transformation.function.language);
		} else {
			setMode('mappings');
		}
	}, []);

	useEffect(() => {
		if (!hasIdentityAndTimestamp || !action.lastChangeTimeProperty) {
			return;
		}
		// check if the last change time format is custom.
		const formats = Object.values(lastChangeTimeFormats);
		if (!formats.includes(action.lastChangeTimeFormat)) {
			setIsCustomLastChangeTimeFormatSelected(true);
		}
	}, []);

	useEffect(() => {
		if (hasIdentityAndTimestamp && isFirstCompilation.current && !isEditing) {
			// precompile the 'IdentityProperty' and 'lastChangeTimeProperty'
			// fields, if possible.
			const a = { ...action };
			const hasIdColumn = actionType.inputSchema.properties.findIndex((prop) => prop.name === 'id') !== -1;
			if (hasIdColumn) {
				a.identityProperty = 'id';
				isFirstCompilation.current = false;
			}
			const hasLastChangeTimeProperty =
				actionType.inputSchema.properties.findIndex((prop) => prop.name === 'timestamp') !== -1;
			if (hasLastChangeTimeProperty) {
				a.lastChangeTimeProperty = 'timestamp';
				if (doesLastChangeTimePropertyNeedFormat(a.lastChangeTimeProperty, actionType.inputSchema)) {
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
				inProperties: [],
				outProperties: [],
			};
			setAction(a);
		}
	}, [selectedLanguage]);

	const needFormat: boolean = useMemo(() => {
		if (
			(connection.isFileStorage || connection.isDatabase) &&
			action.lastChangeTimeProperty &&
			!isTransformationDisabled
		) {
			return doesLastChangeTimePropertyNeedFormat(action.lastChangeTimeProperty, actionType.inputSchema);
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

	const identityPropertyError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			if (action.identityProperty === '' && !isFirstCompilation.current) {
				return 'The user identifier cannot be empty';
			}
			return checkIfPropertyExists(action.identityProperty, flatSchema);
		}
	}, [action, flatSchema]);

	const lastChangeTimePropertyError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			return checkIfPropertyExists(action.lastChangeTimeProperty, flatSchema);
		}
	}, [action, flatSchema]);

	const { identityPropertyList, lastChangeTimeList, mappingList } = useMemo(() => {
		return {
			identityPropertyList: getIdentityPropertyComboboxItems(actionType.inputSchema),
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
			inProperties: [],
			outProperties: [],
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
		if (name === 'identityProperty') {
			const a = { ...action };
			a.identityProperty = value;
			setAction(a);
			if (isFirstCompilation.current) {
				isFirstCompilation.current = false;
			}
			return;
		} else if (name === 'lastChangeTimeProperty') {
			const a = { ...action };
			a.lastChangeTimeProperty = value;
			if (value === '' || !doesLastChangeTimePropertyNeedFormat(value, actionType.inputSchema)) {
				a.lastChangeTimeFormat = '';
			}
			setAction(a);
			return;
		}
		await updateMapping(name, value);
	};

	const onUpdateIdentityProperty = async (_: string, value: string) => {
		const a = { ...action };
		a.identityProperty = value;
		setAction(a);
		if (isFirstCompilation.current) {
			isFirstCompilation.current = false;
		}
	};

	const onUpdateLastChangeTimeProperty = async (_: string, value: string) => {
		const a = { ...action };
		a.lastChangeTimeProperty = value;
		if (value === '' || !doesLastChangeTimePropertyNeedFormat(value, actionType.inputSchema)) {
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
				handleError('Matching properties cannot be empty');
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
			mode={mode}
			setMode={setMode}
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
			{hasIdentityAndTimestamp && (
				<Section title='Special properties' padded={true} annotated={true}>
					<div className='action__transformation-special-properties'>
						<div className='action__transformation-identity-property'>
							<div className='action__transformation-special-properties-label'>
								Identity
								<span className='action__transformation-special-properties-asterisk'>*</span>:
							</div>
							<Combobox
								onInput={onUpdateIdentityProperty}
								onSelect={onUpdateIdentityProperty}
								name='identityProperty'
								initialValue={identityPropertyList.length === 0 ? '' : action.identityProperty!}
								disabled={isTransformationDisabled || identityPropertyList.length === 0}
								className='action__transformation-input-property'
								isExpression={false}
								items={identityPropertyList}
								caret={true}
								clearable={action.identityProperty?.length > 0}
								error={
									identityPropertyList.length === 0
										? `No column ${
												connection.isFileStorage ? 'in the file' : 'returned by the query'
											} can be used as user identifier`
										: identityPropertyError
								}
								size='small'
								helpText='A property that uniquely identifies a user'
							/>
						</div>
						<div className='action__transformation-last-change-time-property'>
							<div className='action__transformation-last-change-time'>
								<div className='action__transformation-special-properties-label'>Last change time:</div>
								<Combobox
									onInput={onUpdateLastChangeTimeProperty}
									onSelect={onUpdateLastChangeTimeProperty}
									initialValue={action.lastChangeTimeProperty!}
									name='lastChangeTimeProperty'
									disabled={isTransformationDisabled}
									className='action__transformation-input-property'
									isExpression={false}
									caret={true}
									items={lastChangeTimeList}
									clearable={action.lastChangeTimeProperty?.length > 0}
									error={lastChangeTimePropertyError}
									size='small'
									helpText='A property with the time of the last modification of a user'
								/>
							</div>
							<div className='action__transformation-last-change-format-property'>
								<div className='action__transformation-last-change-format'>
									<div className='action__transformation-special-properties-label'>Format:</div>
									<SlSelect
										onSlChange={onChangeLastChangeTimeFormat}
										value={
											isCustomLastChangeTimeFormatSelected
												? 'custom'
												: action.lastChangeTimeProperty
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
	mode: 'mappings' | 'transformation' | '';
	setMode: React.Dispatch<React.SetStateAction<'mappings' | 'transformation' | ''>>;
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
	mode: '' | 'mappings' | 'transformation',
	oldValue: TransformedMapping | TransformationFunction,
	newValue: TransformedMapping | TransformationFunction,
) => {
	if (mode === '') {
		return;
	}
	if (mode === 'mappings') {
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
	mode,
	setMode,
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

	const pendingMode = useRef<string>();
	const firstValue = useRef<TransformedMapping | TransformationFunction>();
	const hasNeverChangedMode = useRef<boolean>(true);

	const { connection } = useContext(ConnectionContext);
	const { setSelectedInProperties, setSelectedOutProperties, isEditing } = useContext(actionContext);

	useEffect(() => {
		if (mode === 'mappings') {
			firstValue.current = JSON.parse(JSON.stringify(action.transformation.mapping));
		} else {
			firstValue.current = JSON.parse(JSON.stringify(action.transformation.function));
		}
	}, [mode, selectedLanguage]);

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

	const onChangeMode = (delay: number) => {
		hasNeverChangedMode.current = false;
		const a = { ...action };
		a.inSchema = null;
		a.outSchema = null;
		setIsAlertOpen(false);
		setTimeout(() => {
			if (pendingMode.current == 'mappings') {
				a.transformation.mapping = flattenSchema(actionType.outputSchema);
				a.transformation.function = null;
				setSelectedLanguage('');
				setSelectedInProperties([]);
				setSelectedOutProperties([]);
				setAction(a);
				setMode('mappings');
			} else {
				a.transformation.mapping = null;
				a.transformation.function = {
					source: RAW_TRANSFORMATION_FUNCTIONS[pendingMode.current].replace(
						'$parameterName',
						getTransformationFunctionParameterName(connection, actionType),
					),
					language: pendingMode.current,
					preserveJSON: false,
					inProperties: [],
					outProperties: [],
				};
				setSelectedLanguage(pendingMode.current);
				setAction(a);
				setMode('transformation');
			}
		}, delay);
	};

	const onModeClick = (newMode: string) => {
		if (newMode === mode) {
			return;
		}
		setIsEditTooltipOpen(false);
		pendingMode.current = newMode;
		if (
			isMappingModified(
				mode,
				firstValue.current,
				mode === 'mappings' ? action.transformation.mapping : action.transformation.function,
			) ||
			(isEditing && hasNeverChangedMode.current)
		) {
			setIsAlertOpen(true);
		} else {
			onChangeMode(0);
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
	if (mode === 'mappings') {
		const workspace = workspaces.find((w) => w.id === selectedWorkspace);
		const mappings: ReactNode[] = [];
		for (const k in action.transformation.mapping) {
			const isOutMatchingProperty = action.matching?.out && action.matching.out === k;
			if (isOutMatchingProperty) {
				continue;
			}
			const isRequired =
				action.transformation.mapping[k].createRequired || action.transformation.mapping[k].updateRequired;
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
						initialValue={action.transformation.mapping[k].value}
						name={k}
						disabled={isTransformationDisabled || action.transformation.mapping[k].disabled === true}
						className='action__transformation-input-property'
						size='small'
						error={action.transformation.mapping[k].error}
						autocompleteExpressions={true}
						isExpression={true}
						items={mappingItems}
						onSelect={onSelectMappingItem}
					>
						{isRequired && (
							<div className='action__transformation-property-icon' slot='prefix'>
								<SlTooltip content='Required' hoist>
									<SlIcon name='asterisk' className='action__transformation-property-icon-required' />
								</SlTooltip>
							</div>
						)}
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
					<SlInput
						readonly
						disabled
						size='small'
						value={k}
						type='text'
						name={k}
						className={`action__transformation-output-property${
							action.transformation.mapping![k].indentation! > 0
								? ' action__transformation-output-property--indented'
								: ''
						}`}
					>
						<SlIcon slot='suffix' name={typeNameToIconName[action.transformation.mapping[k].type]} />
					</SlInput>
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
			className={`transformation-box${' transformation-box--' + mode}${
				isFullscreenAnimating ? ' transformation-box--is-fullscreen-animating' : ''
			}`}
		>
			<div className='transformation-box__header'>
				<div className='transformation-box__header-title'>
					{isCompletelyOpen || !isTransformationFunctionSupported || transformationLanguages.length == 0 ? (
						<>
							<span className='transformation-box__header-icon'>
								{mode === 'mappings' ? <SlIcon name='shuffle' /> : getLanguageLogo(selectedLanguage)}
							</span>
							<div className='transformation-box__header-text'>
								{mode === 'mappings' ? 'Mappings' : selectedLanguage}
							</div>
						</>
					) : (
						<SlButtonGroup className='transformation-box__header-buttons'>
							<SlButton
								className='transformation-box__mappings-button'
								variant={mode === 'mappings' ? 'primary' : 'default'}
								onClick={() => onModeClick('mappings')}
								disabled={isTransformationDisabled}
							>
								Mappings
							</SlButton>
							{transformationLanguages.map((language) => {
								return (
									<SlButton
										key={language}
										variant={
											mode === 'transformation' && selectedLanguage === language
												? 'primary'
												: 'default'
										}
										onClick={() => onModeClick(language)}
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
					{mode === 'transformation' && (
						<SlDropdown className='transformation-box__function-settings'>
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
						<SlButton variant='danger' onClick={() => onChangeMode(200)}>
							Continue
						</SlButton>
					</>
				}
			>
				<div style={{ textAlign: 'center' }}>
					<p>If you switch the mapping mode you will permanently lose all the work you have done</p>
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
		values,
		actionType,
		connection,
		mode,
		selectedInProperties,
		setSelectedInProperties,
		selectedOutProperties,
		setSelectedOutProperties,
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
		// Reset the output of the transformation tests when the user switches
		// the language or the mode of the transformation.
		setOutput('');
		setOutputError('');
	}, [mode, selectedLanguage]);

	useEffect(() => {
		setShowOnlyInSelected(false);
		setShowOnlyOutSelected(false);
		setInSearchTerm('');
		setOutSearchTerm('');
	}, [mode]);

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
						action.format,
						action.path,
						action.sheet,
						action.compression,
						values,
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
					s.push(user.properties);
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

	const onChangeSelectedProperty = (side: 'in' | 'out', key: string) => {
		let properties;
		let schema;
		if (side === 'in') {
			properties = selectedInProperties;
			schema = flatInputSchema;
		} else {
			properties = selectedOutProperties;
			schema = flatOutputSchema;
		}

		const keys = Object.keys(schema);
		const children = keys.filter((k) => k.startsWith(`${key}.`));

		const isSelected = properties.includes(key);
		let props: string[] = [];
		if (isSelected) {
			// Remove the property from the selected list.
			for (const s of properties) {
				if (s !== key) {
					props.push(s);
				}
			}
		} else {
			props = [];
			props.push(key);

			// Remove any child properties that were previously selected
			// since only the parent property will be sent to the
			// server.
			for (const s of properties) {
				if (!children.includes(s)) {
					props.push(s);
				}
			}
		}

		if (side == 'in') {
			setSelectedInProperties(props);
		} else {
			setSelectedOutProperties(props);
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
				values,
				actionType,
				api,
				connection,
				false,
				selectedInProperties,
				selectedOutProperties,
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

		if (mode === 'mappings') {
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
				values,
				actionType,
				api,
				connection,
				false,
				selectedInProperties,
				selectedOutProperties,
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
				values,
				actionType,
				api,
				connection,
				false,
				selectedInProperties,
				selectedOutProperties,
			);
		} catch (err) {
			setTimeout(() => {
				setOutputError(err.message);
				setIsExecuting(false);
			}, 300);
			return;
		}

		let res: EventPreviewResponse;
		try {
			res = await api.workspaces.connections.eventPreview(
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
			res = await api.workspaces.connections.query(connection.id, action.query, 20);
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

	let InputPanelTitle = '';
	let OutputPanelTitle = '';
	if (connection.isSource) {
		if (actionType.target === 'Users') {
			const term = connection.connector.termForUsers;
			if (isEventBasedUserImport) {
				InputPanelTitle = 'Events';
			} else {
				InputPanelTitle = term[0].toUpperCase() + term.slice(1, term.length);
			}
			OutputPanelTitle = 'Resulting user';
		} else if (actionType.target === 'Groups') {
			const term = connection.connector.termForGroups;
			InputPanelTitle = term[0].toUpperCase() + term.slice(1, term.length);
			OutputPanelTitle = 'Resulting group';
		}
	} else {
		if (actionType.target === 'Events') {
			InputPanelTitle = 'Events';
			OutputPanelTitle = 'Request';
		} else if (actionType.target === 'Users') {
			InputPanelTitle = 'Users';
			const term = removeTrailingS(connection.connector.termForUsers);
			OutputPanelTitle = 'Resulting ' + term;
		} else if (actionType.target === 'Groups') {
			InputPanelTitle = 'Groups';
			const term = removeTrailingS(connection.connector.termForGroups);
			OutputPanelTitle = 'Resulting ' + term;
		}
	}

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
				{mode === 'transformation' && (
					<SlSwitch
						className='fullscreen-transformation__panel-schema-show-only-selected'
						size='small'
						onSlChange={onChangeShowOnlyInSelected}
					>
						Show only selected properties
					</SlSwitch>
				)}
				{inputSchema.properties.map((p) => {
					if (inSearchTerm !== '') {
						const name = p.name;
						const isSearched = name.toLowerCase().includes(inSearchTerm.toLowerCase());
						if (!isSearched) {
							return null;
						}
					}

					if (mode === 'transformation') {
						const isSelected = selectedInProperties.includes(p.name);
						const hasSelectedChildren =
							selectedInProperties.findIndex((prop) => prop.startsWith(`${p.name}.`)) !== -1;
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
								mode={mode}
								exportMode={action.exportMode}
								selectedProperties={selectedInProperties}
								onChangeSelectedProperty={(key) => onChangeSelectedProperty('in', key)}
							/>
						);
					} else {
						return (
							<TransformationProperty
								key={p.name}
								language={selectedLanguage}
								property={p}
								side='input'
								mode={mode}
								exportMode={action.exportMode}
								selectedProperties={selectedInProperties}
								onChangeSelectedProperty={(key) => onChangeSelectedProperty('in', key)}
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
					let sampleToShow = s;
					if (mode === 'transformation') {
						// Show only the checked properties.
						const filtered = {};
						for (const k in s) {
							if (selectedInProperties.includes(k)) {
								filtered[k] = s[k];
							}
						}
						sampleToShow = filtered;
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
									<SyntaxHighlight>{JSONbig.stringify(sampleToShow, null, 4)}</SyntaxHighlight>
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
			<SlSplitPanel style={{ '--min': '10px', '--max': '800px' } as React.CSSProperties}>
				<div className='fullscreen-transformation__left-panel' slot='start'>
					<SlSplitPanel style={{ '--min': '10px', '--max': 'calc(100% - 10px)' } as React.CSSProperties}>
						<div className='fullscreen-transformation__input-panel' slot='start'>
							<div className='fullscreen-transformation__panel-title-wrapper'>
								<div className='fullscreen-transformation__panel-title'>{InputPanelTitle}</div>
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
						<div className='fullscreen-transformation__output-panel' slot='end'>
							<div className='fullscreen-transformation__panel-title-wrapper'>
								<div className='fullscreen-transformation__panel-title'>{OutputPanelTitle}</div>
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
										{OutputPanelTitle === 'Request' ? 'Preview' : 'Result'}
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
										{mode === 'transformation' && (
											<SlSwitch
												className='fullscreen-transformation__panel-schema-show-only-selected'
												size='small'
												onSlChange={onChangeShowOnlyOutSelected}
											>
												Show only selected properties
											</SlSwitch>
										)}
										{outputSchema.properties.map((p) => {
											if (outSearchTerm !== '') {
												const name = p.name;
												const isSearched = name
													.toLowerCase()
													.includes(outSearchTerm.toLowerCase());
												if (!isSearched) {
													return null;
												}
											}

											if (action.matching?.out && action.matching.out === p.name) {
												// Do not show the property used
												//  as external matching
												//  property as it must not be
												//  transformed.
												return null;
											}

											if (mode === 'transformation') {
												const isSelected = selectedOutProperties.includes(p.name);
												const hasSelectedChildren =
													selectedOutProperties.findIndex((prop) =>
														prop.startsWith(`${p.name}.`),
													) !== -1;
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
														mode={mode}
														exportMode={action.exportMode}
														selectedProperties={selectedOutProperties}
														onChangeSelectedProperty={(key) =>
															onChangeSelectedProperty('out', key)
														}
													/>
												);
											} else {
												return (
													<TransformationProperty
														key={p.name}
														property={p}
														language={selectedLanguage}
														side='output'
														mode={mode}
														exportMode={action.exportMode}
														selectedProperties={selectedOutProperties}
														onChangeSelectedProperty={(key) =>
															onChangeSelectedProperty('out', key)
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
											<SlIconButton
												className='fullscreen-transformation__output-clear'
												name='x-lg'
											/>
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
	);
};

interface TransformationNestedPropertiesProps {
	property: Property;
	language: string;
	nesting: number;
	parentName?: string;
	side: 'input' | 'output';
	mode: 'mappings' | 'transformation' | '';
	exportMode: ExportMode;
	selectedProperties: string[];
	onChangeSelectedProperty: (key: string) => void;
}

const TransformationNestedProperties = ({
	property,
	language,
	nesting,
	parentName,
	side,
	mode,
	exportMode,
	selectedProperties,
	onChangeSelectedProperty,
}: TransformationNestedPropertiesProps) => {
	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	const typ = property.type as ObjectType;

	return (
		<div
			className={`fullscreen-transformation__nested${isExpanded ? ' fullscreen-transformation__nested--expand' : ''}`}
		>
			<TransformationProperty
				property={property}
				language={language}
				isParent={true}
				parentName={parentName}
				side={side}
				mode={mode}
				exportMode={exportMode}
				selectedProperties={selectedProperties}
				onChangeSelectedProperty={onChangeSelectedProperty}
				isExpanded={isExpanded}
				setIsExpanded={setIsExpanded}
			/>
			<div
				className='fullscreen-transformation__sub-properties'
				style={{ '--property-indentation': `${nesting * 20}px` } as React.CSSProperties}
			>
				{isExpanded &&
					typ.properties.map((p) => {
						if (p.type.name === 'Object') {
							return (
								<TransformationNestedProperties
									key={p.name}
									property={p}
									language={language}
									nesting={nesting + 1}
									parentName={parentName ? parentName + '.' + property.name : property.name}
									side={side}
									mode={mode}
									exportMode={exportMode}
									selectedProperties={selectedProperties}
									onChangeSelectedProperty={onChangeSelectedProperty}
								/>
							);
						} else {
							return (
								<TransformationProperty
									key={p.name}
									property={p}
									language={language}
									parentName={parentName ? parentName + '.' + property.name : property.name}
									side={side}
									mode={mode}
									exportMode={exportMode}
									selectedProperties={selectedProperties}
									onChangeSelectedProperty={onChangeSelectedProperty}
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
	mode: 'mappings' | 'transformation' | '';
	exportMode: ExportMode;
	selectedProperties: string[];
	onChangeSelectedProperty: (key: string) => void;
	isExpanded?: boolean;
	setIsExpanded?: React.Dispatch<React.SetStateAction<boolean>>;
}

const TransformationProperty = ({
	property,
	language,
	isParent,
	parentName,
	side,
	mode,
	exportMode,
	selectedProperties,
	onChangeSelectedProperty,
	isExpanded,
	setIsExpanded,
}: TransformationPropertyProps) => {
	const { workspaces, selectedWorkspace } = useContext(AppContext);

	let key = property.name;
	if (parentName) {
		key = parentName + '.' + property.name;
	}

	const workspace = workspaces.find((w) => w.id === selectedWorkspace);
	const isIdentifier = workspace.identifiers.includes(key);
	const isSelected = selectedProperties.includes(key);
	const hasSelectedChildren = selectedProperties.findIndex((p) => p.startsWith(`${key}.`)) !== -1;
	const hasSelectedParent = selectedProperties.findIndex((p) => key.startsWith(`${p}.`)) !== -1;

	return (
		<div
			className={`fullscreen-transformation__property-wrapper${isParent ? ' fullscreen-transformation__property-wrapper--parent' : ''}`}
		>
			{mode === 'transformation' && (
				<SlCheckbox
					className='fullscreen-transformation__property-check'
					checked={isSelected || hasSelectedParent}
					indeterminate={hasSelectedChildren && !isSelected}
					disabled={hasSelectedParent}
					onSlChange={() => onChangeSelectedProperty(key)}
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
					<span className='fullscreen-transformation__property-name-text'>{property.name}</span>
					<span className='fullscreen-transformation__property-type'>
						{side === 'input' && property.readOptional && <span>optional</span>}
						{side === 'output' &&
							exportMode != null &&
							((property.createRequired && exportMode.includes('Create')) ||
								(property.updateRequired && exportMode.includes('Update'))) && <span>required</span>}
						<span>
							{language === ''
								? property.type.name
								: language === 'Python'
									? fromKindToPythonType(property.type)
									: fromKindToJavascriptType(property.type)}
						</span>
					</span>
					{!isParent && (
						<SlCopyButton
							className='fullscreen-transformation__property-copy'
							value={parentName ? `${parentName}.${property.name}` : property.name}
							copyLabel='Click to copy'
							successLabel='✓ Copied'
							errorLabel='Copying to clipboard is not supported by your browser'
						/>
					)}
				</div>
			</div>
			{isParent && (
				<SlIcon
					className='fullscreen-transformation__property-caret'
					name='caret-right-fill'
					onClick={() => {
						setIsExpanded(!isExpanded);
					}}
				/>
			)}
		</div>
	);
};

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

function fromKindToJavascriptType(type: Type) {
	// TODO: add additional information (property is nullable, property values,
	//  property length). This needs the full type definition and not the
	// type name only.
	const name = type.name;
	switch (name) {
		case 'Boolean':
			return 'Boolean';
		case 'Int':
		case 'Uint':
			if (type.bitSize === 8 || type.bitSize === 16 || type.bitSize === 24 || type.bitSize === 32) {
				return 'Number';
			} else {
				return 'BigInt';
			}
		case 'Float':
			return 'Number';
		case 'Decimal':
			return 'String';
		case 'DateTime':
		case 'Date':
		case 'Time':
		case 'Year':
			return 'Date';
		case 'UUID':
			return 'String';
		case 'JSON':
			return 'String';
		case 'Inet':
			return 'String';
		case 'Text':
			return 'String';
		case 'Array':
			return 'Array';
		case 'Object':
			return 'Object';
		case 'Map':
			return 'Map';
		default:
			console.error(`schema contains unknow property type ${name}`);
			return 'unknown property type';
	}
}

function fromKindToPythonType(type: Type) {
	// TODO: add additional information (property is nullable, property values,
	// property length). This needs the full type definition and not the
	// type name only.
	switch (type.name) {
		case 'Boolean':
			return 'bool';
		case 'Int':
		case 'Uint':
			return 'int';
		case 'Float':
			return 'float';
		case 'Decimal':
			return 'decimal.Decimal';
		case 'DateTime':
			return 'datetime.datetime';
		case 'Date':
			return 'datetime.date';
		case 'Time':
			return 'datetime.time';
		case 'Year':
			return 'int';
		case 'UUID':
			return 'uuid.UUID';
		case 'JSON':
			return 'str';
		case 'Inet':
			return 'str';
		case 'Text':
			return 'str';
		case 'Array':
			return 'list';
		case 'Object':
			return 'dict';
		case 'Map':
			return 'dict';
		default:
			console.error(`schema contains unknow property type ${type}`);
			return 'unknown property type';
	}
}

function removeTrailingS(str: string) {
	if (str.endsWith('s')) {
		return str.slice(0, -1);
	}
	return str;
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

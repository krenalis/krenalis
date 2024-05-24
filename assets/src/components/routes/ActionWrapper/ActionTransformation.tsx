import React, { useState, useRef, useContext, useEffect, forwardRef, useMemo, ReactNode } from 'react';
import { checkIfPropertyExists, updateMappingProperty, extractSpecialProperties } from './Action.helpers';
import {
	getDisplayedPropertyComboboxItems,
	getSchemaComboboxItems,
	getIdentityPropertyComboboxItems,
	getLastChangeTimeComboboxItems,
} from '../../helpers/getSchemaComboBoxItems';
import {
	TransformedAction,
	TransformedActionType,
	TransformedMapping,
	doesLastChangeTimePropertyNeedFormat,
	flattenSchema,
	transformInActionToSet,
} from '../../../lib/core/action';
import { RAW_TRANSFORMATION_FUNCTIONS } from './Action.constants';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { ComboBoxInput, ComboBoxList } from '../../base/ComboBox/ComboBox';
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
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import SlSplitPanel from '@shoelace-style/shoelace/dist/react/split-panel/index.js';
import SlAlert from '@shoelace-style/shoelace/dist/react/alert/index.js';
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
import Type, { ObjectType, Property } from '../../../lib/api/types/types';
import { EventListenerEvent } from '../../../hooks/useEventListener';
import { Sample } from './Action.types';
import { UnprocessableError } from '../../../lib/api/errors';
import ConnectionContext from '../../../context/ConnectionContext';
import Workspace from '../../../lib/api/types/workspace';
import { ActionToSet, TransformationFunction } from '../../../lib/api/types/action';
import { debounceWithAbort } from '../../../utils/debounce';
import TransformedConnector from '../../../lib/core/connector';

const defaultTransformationParameterByTarget = {
	Users: 'user',
	Groups: 'group',
	Events: 'event',
};

const lastChangeTimeFormats = {
	dateTime: 'DateTime',
	dateOnly: 'DateOnly',
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
		isFileConnectorChanged,
	} = useContext(ActionContext);

	const mappingListRef = useRef(null);
	const identityPropertyListRef = useRef(null);
	const displayedPropertyListRef = useRef(null);
	const lastChangeTimePropertyListRef = useRef(null);
	const isFirstCompilation = useRef(true);
	const lastChangeTimeCustomFormatInputRef = useRef(null);

	const hasSpecialProperties = useMemo(() => {
		return (
			connection.isSource &&
			(connection.isApp || connection.isDatabase || connection.isFileStorage || connection.isEventBased) &&
			(actionType.Target === 'Users' || actionType.Target === 'Events')
		);
	}, [connection, actionType]);

	useEffect(() => {
		// when a new file is confirmed the UI should behave as if it is
		// the first the user is compiling the action's transformation.
		if (!isFileConnectorChanged) {
			isFirstCompilation.current = true;
		}
	}, [isFileConnectorChanged]);

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
		if (action.Transformation.Function != null) {
			setMode('transformation');
			setSelectedLanguage(action.Transformation.Function.Language);
		} else {
			setMode('mappings');
		}
	}, []);

	useEffect(() => {
		if (!hasSpecialProperties || !action.LastChangeTimeProperty) {
			return;
		}
		// check if the last change time format is custom.
		const formats = Object.values(lastChangeTimeFormats);
		if (!formats.includes(action.LastChangeTimeFormat)) {
			setIsCustomLastChangeTimeFormatSelected(true);
		}
	}, []);

	useEffect(() => {
		if (hasSpecialProperties && isFirstCompilation.current) {
			// precompile the 'IdentityProperty', 'displayedProperty' and
			// 'lastChangeTimeProperty' fields, if possible.
			const a = { ...action };
			if (connection.isApp) {
				const suggestedDisplayedProperty = connection.connector.suggestedDisplayedProperty;
				if (suggestedDisplayedProperty !== '') {
					// check if the suggested displayed property exists in the
					// input schema.
					const flatSchema = flattenSchema(actionType.InputSchema);
					if (flatSchema[suggestedDisplayedProperty]) {
						a.DisplayedProperty = suggestedDisplayedProperty;
					}
				}
			} else if (connection.isDatabase || connection.isFileStorage) {
				const hasIdColumn = actionType.InputSchema.properties.findIndex((prop) => prop.name === 'id') !== -1;
				if (hasIdColumn) {
					a.IdentityProperty = 'id';
					a.DisplayedProperty = 'id';
					isFirstCompilation.current = false;
				}
				const hasLastChangeTimeProperty =
					actionType.InputSchema.properties.findIndex((prop) => prop.name === 'timestamp') !== -1;
				if (hasLastChangeTimeProperty) {
					a.LastChangeTimeProperty = 'timestamp';
					if (doesLastChangeTimePropertyNeedFormat(a.LastChangeTimeProperty, actionType.InputSchema)) {
						a.LastChangeTimeFormat = lastChangeTimeFormats['dateTime'];
					}
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
		const isTransformationUndefined = a.Transformation.Function == null;
		const isLanguageChanged = !isTransformationUndefined && a.Transformation.Function.Language !== selectedLanguage;
		if (isTransformationUndefined || isLanguageChanged) {
			a.Transformation.Function = {
				Source: RAW_TRANSFORMATION_FUNCTIONS[selectedLanguage].replace(
					'$parameterName',
					defaultTransformationParameterByTarget[actionType.Target],
				),
				Language: selectedLanguage,
			};
			setAction(a);
		}
	}, [selectedLanguage]);

	const needFormat: boolean = useMemo(() => {
		if (
			(connection.isFileStorage || connection.isDatabase) &&
			action.LastChangeTimeProperty &&
			!isTransformationDisabled
		) {
			return doesLastChangeTimePropertyNeedFormat(action.LastChangeTimeProperty, actionType.InputSchema);
		}
		return false;
	}, [action, actionType, isTransformationDisabled]);

	const fileConnector: TransformedConnector | null = useMemo(() => {
		if (action.Connector) {
			return connectors.find((c) => c.name === action.Connector);
		}
		return null;
	}, [action]);

	const flatSchema = useMemo<TransformedMapping>(() => {
		return flattenSchema(actionType.InputSchema);
	}, [actionType]);

	const identityPropertyError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			if (action.IdentityProperty === '' && !isFirstCompilation.current) {
				return 'The user identifier cannot be empty';
			}
			return checkIfPropertyExists(action.IdentityProperty, flatSchema);
		}
	}, [action, flatSchema]);

	const displayedProperty = useMemo<string>(() => {
		if (!hasSpecialProperties || connection.isEventBased) {
			return;
		}
		return checkIfPropertyExists(action.DisplayedProperty, flatSchema);
	}, [action, flatSchema]);

	const lastChangeTimePropertyError = useMemo<string>(() => {
		if (connection.isFileStorage || connection.isDatabase) {
			return checkIfPropertyExists(action.LastChangeTimeProperty, flatSchema);
		}
	}, [action, flatSchema]);

	const { identityPropertyList, displayedPropertyList, lastChangeTimeList, mappingList } = useMemo(() => {
		return {
			identityPropertyList: getIdentityPropertyComboboxItems(actionType.InputSchema),
			displayedPropertyList: getDisplayedPropertyComboboxItems(actionType.InputSchema),
			lastChangeTimeList: getLastChangeTimeComboboxItems(actionType.InputSchema),
			mappingList: getSchemaComboboxItems(actionType.InputSchema),
		};
	}, [actionType]);

	const onChangeTransformationFunction = (source: string) => {
		const a = { ...action };
		a.Transformation.Function = {
			Source: source,
			Language: selectedLanguage,
		};
		setAction(a);
	};

	const updateMapping = async (name: string, value: string, signal?: AbortSignal) => {
		let errorMessage = '';
		if (value !== '') {
			try {
				errorMessage = await api.validateExpression(
					value,
					actionType.InputSchema.properties,
					action.Transformation.Mapping![name].full.type,
					action.Transformation.Mapping![name].full.required,
					action.Transformation.Mapping![name].full.nullable,
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
		const updatedAction = updateMappingProperty(action, name, value, errorMessage);
		setAction(updatedAction);
	};

	const debouncedUpdateMapping = useMemo(() => debounceWithAbort(updateMapping, 500), [actionType, action]);

	const onUpdateMapping = async (e: any) => {
		const target = e.target;
		let { name, value } = target;
		debouncedUpdateMapping(name, value);
	};

	const onSelectProperty = async (input, value) => {
		if (input.name === 'identityProperty') {
			const a = { ...action };
			a.IdentityProperty = value;
			if (isFirstCompilation.current && a.DisplayedProperty === '') {
				a.DisplayedProperty = value;
			}
			setAction(a);
			if (isFirstCompilation.current) {
				isFirstCompilation.current = false;
			}
			return;
		} else if (input.name === 'displayedProperty') {
			const a = { ...action };
			a.DisplayedProperty = value;
			setAction(a);
			return;
		} else if (input.name === 'lastChangeTimeProperty') {
			const a = { ...action };
			a.LastChangeTimeProperty = value;
			if (value === '' || !doesLastChangeTimePropertyNeedFormat(value, actionType.InputSchema)) {
				a.LastChangeTimeFormat = '';
			}
			setAction(a);
			return;
		}
		await updateMapping(input.name, value);
	};

	const onUpdateIdentityProperty = async (e) => {
		const a = { ...action };
		const value = e.target.value;
		a.IdentityProperty = value;
		if (isFirstCompilation.current && a.DisplayedProperty === '') {
			a.DisplayedProperty = value;
		}
		setAction(a);
		if (isFirstCompilation.current) {
			isFirstCompilation.current = false;
		}
	};

	const onUpdateDisplayedProperty = (e) => {
		const a = { ...action };
		a.DisplayedProperty = e.target.value;
		setAction(a);
	};

	const onUpdateLastChangeTimeProperty = async (e) => {
		const target = e.target;
		let { value } = target;
		const a = { ...action };
		a.LastChangeTimeProperty = value;
		if (value === '' || !doesLastChangeTimePropertyNeedFormat(value, actionType.InputSchema)) {
			setIsCustomLastChangeTimeFormatSelected(false);
			a.LastChangeTimeFormat = '';
		}
		setAction(a);
	};

	const onChangeLastChangeTimeFormat = (e) => {
		const a = { ...action };
		const v = e.target.value;
		if (v === 'custom') {
			setIsCustomLastChangeTimeFormatSelected(true);
			a.LastChangeTimeFormat = '';
			setTimeout(() => {
				if (lastChangeTimeCustomFormatInputRef.current) {
					lastChangeTimeCustomFormatInputRef.current.focus();
				}
			}, 50);
		} else {
			setIsCustomLastChangeTimeFormatSelected(false);
			a.LastChangeTimeFormat = lastChangeTimeFormats[e.target.value];
		}
		setAction(a);
	};

	const onInputLastChangeTimeCustomFormat = (e) => {
		const a = { ...action };
		a.LastChangeTimeFormat = e.target.value;
		setAction(a);
	};

	const onOpenFullscreenTransformation = () => {
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
			mappingListRef={mappingListRef}
			onUpdateMapping={onUpdateMapping}
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
			{hasSpecialProperties && (
				<Section title='Special properties' padded={true}>
					<div className='action__transformation-special-properties'>
						{(connection.isFileStorage || connection.isDatabase) && (
							<div className='action__transformation-identity-property'>
								<div className='action__transformation-special-properties-label'>
									Identity
									<span className='action__transformation-special-properties-asterisk'>*</span>:
								</div>
								<ComboBoxInput
									comboBoxListRef={identityPropertyListRef}
									onInput={onUpdateIdentityProperty}
									value={identityPropertyList.length === 0 ? '' : action.IdentityProperty!}
									name='identityProperty'
									disabled={isTransformationDisabled || identityPropertyList.length === 0}
									className='action__transformation-input-property'
									caret={true}
									clearable={action.IdentityProperty?.length > 0}
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
						)}
						<div className='action__transformation-displayed-property'>
							<div className='action__transformation-special-properties-label'>Displayed:</div>
							{connection.isEventBased ? (
								<SlInput
									onSlInput={onUpdateDisplayedProperty}
									value={action.DisplayedProperty}
									disabled={isTransformationDisabled}
									size='small'
								/>
							) : (
								<ComboBoxInput
									comboBoxListRef={displayedPropertyListRef}
									onInput={onUpdateDisplayedProperty}
									value={action.DisplayedProperty}
									name='displayedProperty'
									disabled={isTransformationDisabled}
									className='action__transformation-input-property'
									caret={true}
									clearable={action.DisplayedProperty?.length > 0}
									error={displayedProperty}
									size='small'
									helpText='An additional property displayed to distinguish a user'
								/>
							)}
						</div>
						{(connection.isFileStorage || connection.isDatabase) && (
							<div className='action__transformation-last-change-time-property'>
								<div className='action__transformation-last-change-time'>
									<div className='action__transformation-special-properties-label'>
										Last change time:
									</div>
									<ComboBoxInput
										comboBoxListRef={lastChangeTimePropertyListRef}
										onInput={onUpdateLastChangeTimeProperty}
										value={action.LastChangeTimeProperty!}
										name='lastChangeTimeProperty'
										disabled={isTransformationDisabled}
										className='action__transformation-input-property'
										caret={true}
										clearable={action.LastChangeTimeProperty?.length > 0}
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
													: action.LastChangeTimeProperty
														? Object.keys(lastChangeTimeFormats).find(
																(key) =>
																	lastChangeTimeFormats[key] ===
																	action.LastChangeTimeFormat,
															)
														: ''
											}
											name='lastChangeTimeFormat'
											disabled={!needFormat}
											size='small'
										>
											<SlOption value='dateTime'>2006-01-02 15:04:05</SlOption>
											<SlOption value='dateOnly'>2006-01-02</SlOption>
											<SlOption value='iso8601'>ISO 8601</SlOption>
											{fileConnector?.name === 'Excel' && (
												<SlOption value='excel'>Excel</SlOption>
											)}
											<SlOption value='custom'>Custom...</SlOption>
										</SlSelect>
									</div>
									{needFormat && isCustomLastChangeTimeFormatSelected && (
										<div className='action__transformation-last-change-custom-format'>
											<SlInput
												onSlInput={onInputLastChangeTimeCustomFormat}
												value={action.LastChangeTimeFormat}
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
						)}
					</div>
				</Section>
			)}
			<Section
				title='Transformation'
				description='The relation between the event properties and the action type properties'
				padded={false}
			>
				{box}
				<FullscreenTransformation
					isFullscreenTransformationOpen={isFullscreenTransformationOpen}
					selectedLanguage={selectedLanguage}
					body={box}
					inputSchema={actionType.InputSchema}
					outputSchema={actionType.OutputSchema}
				/>
				<ComboBoxList ref={identityPropertyListRef} items={identityPropertyList} onSelect={onSelectProperty} />
				<ComboBoxList
					ref={displayedPropertyListRef}
					items={displayedPropertyList}
					onSelect={onSelectProperty}
				/>
				<ComboBoxList
					ref={lastChangeTimePropertyListRef}
					items={lastChangeTimeList}
					onSelect={onSelectProperty}
				/>
				<ComboBoxList ref={mappingListRef} items={mappingList} onSelect={onSelectProperty} />
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
	mappingListRef: React.MutableRefObject<any>;
	onUpdateMapping: (...args: any) => void;
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
	return oldTransformation.Source.trim() !== newTransformation.Source.trim();
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
	mappingListRef,
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
	const [hasFullscreenText, setHasFullscreenText] = useState<boolean>();
	const [isFullscreenAnimating, setIsFullscreenAnimating] = useState<boolean>(false);

	const pendingMode = useRef<string>();
	const firstValue = useRef<TransformedMapping | TransformationFunction>();

	useEffect(() => {
		if (mode === 'mappings') {
			firstValue.current = JSON.parse(JSON.stringify(action.Transformation.Mapping));
		} else {
			firstValue.current = JSON.parse(JSON.stringify(action.Transformation.Function));
		}
	}, [mode, selectedLanguage]);

	useEffect(() => {
		if (isFullscreenTransformationOpen) {
			setTimeout(() => {
				setIsFullscreenAnimating(true);
			}, 100);
			setTimeout(() => {
				setIsFullscreenAnimating(false);
				setHasFullscreenText(true);
			}, 250);
		} else {
			setHasFullscreenText(false);
		}
	}, [isFullscreenTransformationOpen]);

	const onChangeMode = (delay: number) => {
		const a = { ...action };
		a.InSchema = null;
		a.OutSchema = null;
		setIsAlertOpen(false);
		setTimeout(() => {
			if (pendingMode.current == 'mappings') {
				a.Transformation.Mapping = flattenSchema(actionType.OutputSchema);
				a.Transformation.Function = null;
				setSelectedLanguage('');
				setAction(a);
				setMode('mappings');
			} else {
				a.Transformation.Mapping = null;
				a.Transformation.Function = {
					Source: RAW_TRANSFORMATION_FUNCTIONS[pendingMode.current].replace(
						'$parameterName',
						defaultTransformationParameterByTarget[actionType.Target],
					),
					Language: pendingMode.current,
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
		pendingMode.current = newMode;
		if (
			isMappingModified(
				mode,
				firstValue.current,
				mode === 'mappings' ? action.Transformation.Mapping : action.Transformation.Function,
			)
		) {
			setIsAlertOpen(true);
		} else {
			onChangeMode(0);
		}
	};

	let body: ReactNode;
	if (mode === 'mappings') {
		const workspace = workspaces.find((w) => w.ID === selectedWorkspace);
		const mappings: ReactNode[] = [];
		for (const k in action.Transformation.Mapping) {
			mappings.push(
				<div
					key={k}
					className='action__transformation-mapping'
					data-key={k}
					style={
						{
							'--mapping-indentation': `${action.Transformation.Mapping![k].indentation! * 30}px`,
						} as React.CSSProperties
					}
				>
					<ComboBoxInput
						comboBoxListRef={mappingListRef}
						onInput={onUpdateMapping}
						value={action.Transformation.Mapping[k].value}
						name={k}
						disabled={isTransformationDisabled || action.Transformation.Mapping[k].disabled === true}
						className='action__transformation-input-property'
						size='small'
						error={action.Transformation.Mapping[k].error}
						autocompleteExpressions={true}
					>
						{action.Transformation.Mapping[k].required && (
							<div className='action__transformation-property-icon' slot='prefix'>
								<SlTooltip content='Required' hoist>
									<SlIcon name='asterisk' className='action__transformation-property-icon-required' />
								</SlTooltip>
							</div>
						)}
						{workspace.Identifiers.includes(k) && (
							<div className='action__transformation-property-icon' slot='prefix'>
								<SlTooltip content='Used as identifier' hoist>
									<SlIcon
										name='person-check'
										className='action__transformation-property-identifier'
									/>
								</SlTooltip>
							</div>
						)}
					</ComboBoxInput>
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
							action.Transformation.Mapping![k].indentation! > 0
								? ' action__transformation-output-property--indented'
								: ''
						}`}
					/>
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
					value={action.Transformation!.Function.Source}
					onChange={(source) => onChangeTransformationFunction(source!)}
					className='action__transformation-function-minimized'
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
					{hasFullscreenText || !isTransformationFunctionSupported || transformationLanguages.length == 0 ? (
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
				<SlButton
					className='transformation-box__fullscreen-button'
					variant='primary'
					onClick={
						isFullscreenTransformationOpen
							? onCloseFullscreenTransformation
							: onOpenFullscreenTransformation
					}
					disabled={isTransformationDisabled}
				>
					{hasFullscreenText ? (
						<SlIcon name='arrows-angle-contract' />
					) : (
						<SlIcon name='arrows-angle-expand' />
					)}
					{hasFullscreenText ? 'Exit testing mode' : 'Testing mode'}
				</SlButton>
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
	const [isInputSchemaSelected, setIsInputSchemaSelected] = useState<boolean>(false);
	const [isOutputSchemaSelected, setIsOutputSchemaSelected] = useState<boolean>(false);
	const [samples, setSamples] = useState<Sample[]>(null);
	const [selectedSample, setSelectedSample] = useState<Sample>(null);
	const [events, setEvents] = useState<EventListenerEvent[]>([]);
	const [selectedEvent, setSelectedEvent] = useState<EventListenerEvent>(null);
	const [output, setOutput] = useState<string>('');
	const [outputError, setOutputError] = useState<string>('');
	const [isExecuting, setIsExecuting] = useState<boolean>(false);

	const { handleError, api } = useContext(AppContext);
	const { action, values, actionType, connection } = useContext(ActionContext);

	const firstNameIdentifier = useRef<string>('');
	const lastNameIdentifier = useRef<string>('');
	const emailIdentifier = useRef<string>('');
	const idIdentifier = useRef<string>('');
	const lastExecutedSample = useRef<Sample>(null);
	const lastExecutedEvent = useRef<EventListenerEvent>(null);

	const collectEvents = (newly: EventListenerEvent[]) => {
		setEvents((prevEvents) => [...prevEvents, ...newly]);
	};

	const { startListening } = useEventListener(0, true, collectEvents);

	useEffect(() => {
		if (connection.isApp && connection.isDestination && actionType.Target === 'Events') {
			startListening();
		}
	}, []);

	useEffect(() => {
		const fetchSamples = async () => {
			let samples: Sample[];
			if (actionType.Target === 'Users') {
				if (connection.isFile && connection.isSource) {
					let res: RecordsResponse;
					try {
						// res = await api.workspaces.connections.records(connection.id, action.Path, action.Sheet, 20);
					} catch (err) {
						handleError(err);
						return;
					}
					const smpls: Sample[] = [];
					for (const r of res.records) {
						const sample = {};
						for (let i = 0; i < res.schema.properties.length; i++) {
							const propertyName = res.schema.properties[i].name;
							sample[propertyName] = {
								value: r[propertyName],
								property: res.schema.properties[i],
							};
						}
						smpls.push(sample);
					}
					samples = smpls;
				} else if (connection.isDatabase && connection.isSource) {
					// Will show a button to execute the query and retrieve the
					// samples (as the query can be destructive).
					return;
				} else if (connection.isApp && connection.isSource) {
					let res: AppUsersResponse;
					try {
						res = await api.workspaces.connections.appUsers(connection.id, inputSchema);
					} catch (err) {
						handleError(err);
						return;
					}
					const smpls: Sample[] = [];
					for (const u of res.users) {
						const sample = {};
						for (let i = 0; i < inputSchema.properties.length; i++) {
							const propertyName = inputSchema.properties[i].name;
							sample[propertyName] = {
								value: u[propertyName],
								property: inputSchema.properties[i],
							};
						}
						smpls.push(sample);
					}
					samples = smpls;
				} else if ((connection.isApp || connection.isDatabase) && connection.isDestination) {
					const properties: string[] = [];
					for (const prop of inputSchema.properties) {
						properties.push(prop.name);
					}
					let res: FindUsersResponse;
					try {
						res = await api.workspaces.users.find(properties, null, 0, 20);
					} catch (err) {
						handleError(err);
						return;
					}
					if (res.users.length === 0) {
						return;
					}
					const smpls: Sample[] = [];
					for (const u of res.users) {
						const sample = {};
						for (let i = 0; i < res.schema.properties.length; i++) {
							const propertyName = res.schema.properties[i].name;
							sample[propertyName] = {
								value: u[propertyName],
								property: res.schema.properties[i],
							};
						}
						smpls.push(sample);
					}
					samples = smpls;
				}
			}
			const { firstNameID, lastNameID, emailID, idID } = extractSpecialProperties(samples);
			firstNameIdentifier.current = firstNameID;
			lastNameIdentifier.current = lastNameID;
			emailIdentifier.current = emailID;
			idIdentifier.current = idID;
			setSamples(samples);
		};
		fetchSamples();
	}, []);

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

	const onExecuteSample = async (sample: Record<string, any>) => {
		lastExecutedSample.current = sample;
		if (JSON.stringify(sample) !== JSON.stringify(selectedSample)) {
			setSelectedSample(null);
		}
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(action, values, actionType, api, connection);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsExecuting(false);
			}, 300);
			return;
		}

		const normalized = normalizeSample(sample);
		let s = {};
		for (const k in normalized) {
			const isInSchema = actionToSet.inSchema.properties.findIndex((prop) => prop.name === k) !== -1;
			if (isInSchema) {
				s[k] = normalized[k];
			}
		}

		let res: TransformDataResponse;
		try {
			res = await api.transformData(s, actionToSet.inSchema, actionToSet.outSchema, actionToSet.transformation);
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
		setOutput(JSON.stringify(res.data, null, 4));
		setTimeout(() => setIsExecuting(false), 300);
	};

	const onExecuteEvent = async (event: EventListenerEvent) => {
		lastExecutedEvent.current = event;
		if (selectedEvent && event.id !== selectedEvent.id) {
			setSelectedEvent(null);
		}
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);

		let actionToSet: ActionToSet;
		try {
			actionToSet = await transformInActionToSet(action, values, actionType, api, connection);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsExecuting(false);
			}, 300);
			return;
		}

		let res: EventPreviewResponse;
		try {
			res = await api.workspaces.connections.eventPreview(
				connection.id,
				actionType.EventType,
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
			res = await api.workspaces.connections.query(connection.id, action.Query, 20);
		} catch (err) {
			handleError(err);
			return;
		}
		const smpls: Sample[] = [];
		for (const r of res.Rows) {
			const sample = {};
			for (let i = 0; i < res.Schema.properties.length; i++) {
				const propertyName = res.Schema.properties[i].name;
				sample[propertyName] = {
					value: r[propertyName],
					property: res.Schema.properties[i],
				};
			}
			smpls.push(sample);
		}
		const { firstNameID, lastNameID, emailID, idID } = extractSpecialProperties(smpls);
		firstNameIdentifier.current = firstNameID;
		lastNameIdentifier.current = lastNameID;
		emailIdentifier.current = emailID;
		idIdentifier.current = idID;
		setSamples(smpls);
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
		if (actionType.Target === 'Users') {
			const term = connection.connector.termForUsers;
			InputPanelTitle = term[0].toUpperCase() + term.slice(1, term.length);
			OutputPanelTitle = 'Resulting user';
		} else if (actionType.Target === 'Groups') {
			const term = connection.connector.termForGroups;
			InputPanelTitle = term[0].toUpperCase() + term.slice(1, term.length);
			OutputPanelTitle = 'Resulting group';
		}
	} else {
		if (actionType.Target === 'Events') {
			InputPanelTitle = 'Events';
			OutputPanelTitle = 'Request';
		} else if (actionType.Target === 'Users') {
			InputPanelTitle = 'Users';
			const term = removeTrailingS(connection.connector.termForUsers);
			OutputPanelTitle = 'Resulting ' + term;
		} else if (actionType.Target === 'Groups') {
			InputPanelTitle = 'Groups';
			const term = removeTrailingS(connection.connector.termForGroups);
			OutputPanelTitle = 'Resulting ' + term;
		}
	}

	let inputPanelContent: ReactNode = null;
	if (isInputSchemaSelected) {
		inputPanelContent = (
			<div className='fullscreen-transformation__panel-schema'>
				{inputSchema.properties.map((p) => {
					if (p.type.name === 'Object') {
						return (
							<TransformationNestedProperties
								key={p.name}
								property={p}
								language={selectedLanguage}
								nesting={0}
							/>
						);
					} else {
						return <TransformationProperty key={p.name} language={selectedLanguage} property={p} />;
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
										{actionType.Target === 'Users' ? (
											<>
												{idIdentifier.current && (
													<div className='fullscreen-transformation__sample-id'>
														{s[idIdentifier.current].value}
													</div>
												)}
												<div>
													<div className='fullscreen-transformation__sample-full-name'>
														{firstNameIdentifier.current && lastNameIdentifier.current
															? s[firstNameIdentifier.current].value +
																' ' +
																s[lastNameIdentifier.current].value
															: `Sample ${i}`}
													</div>
													{emailIdentifier.current && (
														<div className='fullscreen-transformation__sample-email'>
															{s[emailIdentifier.current].value}
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
												onExecuteSample(s);
											}}
										/>
									</div>
								</div>
							}
							details={
								<div className='fullscreen-transformation__sample-source'>
									<SyntaxHighlight>{JSON.stringify(normalizeSample(s), null, 4)}</SyntaxHighlight>
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
	} else if (connection.isApp && connection.isDestination && actionType.Target === 'Events') {
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
												isLastExecuted ? ' fullscreen-transformation__event--last-executed' : ''
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
													onExecuteEvent(e);
													evt.stopPropagation();
												}}
											/>
										</div>
									}
									details={
										<div className='fullscreen-transformation__event-source'>
											<SyntaxHighlight>{e.source}</SyntaxHighlight>
										</div>
									}
								/>
							);
						})}
					</div>
				</div>
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
										variant={isInputSchemaSelected ? 'default' : 'primary'}
										onClick={onSelectInputSamples}
									>
										Samples
									</SlButton>
									<SlButton
										size='small'
										variant={isInputSchemaSelected ? 'primary' : 'default'}
										onClick={onSelectInputSchema}
									>
										Schema
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
										variant={isOutputSchemaSelected ? 'default' : 'primary'}
										onClick={onSelectOutputResult}
										disabled={isExecuting}
									>
										{OutputPanelTitle === 'Request' ? 'Preview' : 'Result'}
									</SlButton>
									<SlButton
										size='small'
										variant={isOutputSchemaSelected ? 'primary' : 'default'}
										onClick={onSelectOutputSchema}
										disabled={isExecuting}
									>
										Schema
									</SlButton>
								</SlButtonGroup>
							</div>
							<div className='fullscreen-transformation__panel-content'>
								{isOutputSchemaSelected ? (
									<div className='fullscreen-transformation__panel-schema'>
										{outputSchema.properties.map((p) => {
											if (p.type.name === 'Object') {
												return (
													<TransformationNestedProperties
														key={p.name}
														property={p}
														language={selectedLanguage}
														nesting={0}
													/>
												);
											} else {
												return (
													<TransformationProperty
														key={p.name}
														property={p}
														language={selectedLanguage}
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
												actionType.Target === 'Events' ? (
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
					<div slot='start' className='fullscreen-transformation__editor-panel'>
						{body}
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
}

const TransformationNestedProperties = ({
	property,
	language,
	nesting,
	parentName,
}: TransformationNestedPropertiesProps) => {
	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	const typ = property.type as ObjectType;
	return (
		<div
			className={`fullscreen-transformation__property${isExpanded ? ' fullscreen-transformation__property--expand' : ''}${
				property.label != null && property.label !== '' ? ' fullscreen-transformation__property--has-label' : ''
			}`}
		>
			<div className='fullscreen-transformation__parent-property'>
				<SlIcon
					name='caret-right-fill'
					onClick={() => {
						setIsExpanded(!isExpanded);
					}}
				/>
				<TransformationProperty property={property} language={language} isParent={true} />
			</div>
			<div className='fullscreen-transformation__sub-properties'>
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
								/>
							);
						} else {
							return (
								<TransformationProperty
									key={p.name}
									property={p}
									language={language}
									parentName={parentName ? parentName + '.' + property.name : property.name}
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
}

const TransformationProperty = ({ property, language, isParent, parentName }: TransformationPropertyProps) => {
	const { workspaces, selectedWorkspace } = useContext(AppContext);

	const workspace = workspaces.find((w) => w.ID === selectedWorkspace);
	let isIdentifier = false;
	if (parentName) {
		isIdentifier = workspace.Identifiers.includes(parentName + '.' + property.name);
	} else {
		isIdentifier = workspace.Identifiers.includes(property.name);
	}

	return (
		<div className={isParent ? '' : 'fullscreen-transformation__property'}>
			<div className='fullscreen-transformation__property-name'>
				{isIdentifier && (
					<SlTooltip content='Used as identifier'>
						<SlIcon className='fullscreen-transformation__property-identifier-icon' name='person-check' />
					</SlTooltip>
				)}
				{property.required && (
					<SlTooltip content='Required'>
						<SlIcon className='fullscreen-transformation__property-required-icon' name='asterisk' />
					</SlTooltip>
				)}
				{property.name}
				<SlCopyButton
					className='fullscreen-transformation__property-copy'
					value={property.name}
					copyLabel='Click to copy'
					successLabel='✓ Copied'
					errorLabel='Copying to clipboard is not supported by your browser'
				/>
			</div>
			{property.label != null && property.label !== '' && (
				<span className='fullscreen-transformation__property-label'>{property.label}</span>
			)}
			<div className='fullscreen-transformation__property-type'>
				{language === ''
					? property.type.name
					: language === 'Python'
						? fromKindToPythonType(property.type)
						: fromKindToJavascriptType(property.type)}
			</div>
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

const normalizeSample = (sample: Sample): Record<string, any> => {
	const normalized = {};
	for (const k in sample) {
		normalized[k] = sample[k].value;
	}
	return normalized;
};

function removeTrailingS(str: string) {
	if (str.endsWith('s')) {
		return str.slice(0, -1);
	}
	return str;
}

export default ActionTransformation;

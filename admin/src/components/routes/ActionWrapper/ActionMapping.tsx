import React, { useState, useRef, useContext, useEffect, forwardRef, ReactNode } from 'react';
import { updateMappingProperty, autocompleteExpression } from './Action.helpers';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import { flattenSchema, isIdentifierProperty } from '../../../lib/helpers/transformedAction';
import { rawTransformationFunctions } from './Action.constants';
import AlertDialog from '../../shared/AlertDialog/AlertDialog';
import { ComboBoxInput, ComboBoxList } from '../../shared/ComboBox/ComboBox';
import Section from '../../shared/Section/Section';
import EditorWrapper from '../../shared/EditorWrapper/EditorWrapper';
import Accordion from '../../shared/Accordion/Accordion';
import useEventListener from '../../../hooks/useEventListener';
import { AppContext } from '../../../context/providers/AppProvider';
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
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlSplitPanel from '@shoelace-style/shoelace/dist/react/split-panel/index.js';
import SlAlert from '@shoelace-style/shoelace/dist/react/alert/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SyntaxHighlight from '../../shared/SyntaxHighlight/SyntaxHighlight';
import {
	AppUsersResponse,
	EventPreviewResponse,
	ExecQueryResponse,
	FindUsersResponse,
	RecordsResponse,
	TransformationLanguagesResponse,
	TransformationPreviewResponse,
} from '../../../types/external/api';
import getLanguageLogo from '../../helpers/getLanguageLogo';
import { ObjectType, Property } from '../../../types/external/types';
import actionContext from '../../../context/ActionContext';
import extractSpecialProperties from '../../../lib/utils/extractSpecialProperties';
import { EventListenerEvent, Sample } from '../../../types/internal/app';
import { UnprocessableError } from '../../../lib/api/errors';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';

const defaultTransformationParameterByTarget = {
	Users: 'user',
	Groups: 'group',
	Events: 'event',
};

const timestampFormats = {
	standard: '2006-01-02 15:04:05',
	rfc3339: '2006-01-02T15:04:05Z07:00',
	rfc3339WithNanoseconds: '2006-01-02T15:04:05.999999999Z07:00',
	dateOnly: '2006-01-02',
};

const ActionMapping = forwardRef<any>((_, ref) => {
	const [isAlertOpen, setIsAlertOpen] = useState<boolean>(false);
	const [transformationLanguages, setTransformationLanguages] = useState<string[]>();
	const [selectedLanguage, setSelectedLanguage] = useState<string>();
	const [isFullscreenTransformationOpen, setIsFullscreenTransformationOpen] = useState<boolean>(false);

	const { api, showError, workspaces, selectedWorkspace } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);
	const {
		isMappingSectionDisabled,
		disabledReason,
		isTransformationAllowed,
		action,
		setAction,
		actionType,
		mode,
		setMode,
		setIsSaveHidden,
	} = useContext(ActionContext);

	const propertiesListRef = useRef(null);
	const minimizedEditorRef = useRef(null);
	const fullscreenEditorRef = useRef(null);

	useEffect(() => {
		const fetchTransformationLanguages = async () => {
			let response: TransformationLanguagesResponse;
			try {
				response = await api.transformationLanguages();
			} catch (err) {
				showError(err);
				return;
			}
			const languages = response.languages;
			setTransformationLanguages(languages);
		};
		if (action.Transformation) {
			setSelectedLanguage(action.Transformation.Language);
		}
		fetchTransformationLanguages();
	}, []);

	useEffect(() => {
		if (action.Transformation != null) {
			setMode('transformation');
		} else {
			setMode('mappings');
		}
	}, []);

	useEffect(() => {
		if (connection.isFile && connection.isSource) {
			// precompile the 'IdentityProperty' and 'TimestampProperty' fields,
			// if possible.
			const a = { ...action };
			if (action.IdentityProperty === '') {
				const hasIdProperty = actionType.InputSchema.properties.findIndex((prop) => prop.name === 'id') !== -1;
				if (hasIdProperty) {
					a.IdentityProperty = 'id';
				}
			}
			if (action.TimestampProperty === '') {
				const hasTimestampProperty =
					actionType.InputSchema.properties.findIndex((prop) => prop.name === 'timestamp') !== -1;
				if (hasTimestampProperty) {
					a.TimestampProperty = 'timestamp';
				}
			}
			setAction(a);
		}
	}, []);

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
		if (selectedLanguage == null) {
			return;
		}
		const a = { ...action };
		const isTransformationUndefined = a.Transformation == null;
		const isLanguageChanged = a.Transformation != null && a.Transformation.Language !== selectedLanguage;
		if (isTransformationUndefined || isLanguageChanged) {
			a.Transformation = {
				Source: rawTransformationFunctions[selectedLanguage].replace(
					'$parameterName',
					defaultTransformationParameterByTarget[actionType.Target],
				),
				Language: selectedLanguage,
			};
			setAction(a);
		}
	}, [selectedLanguage]);

	const onMinimizedEditorMount = (editor) => {
		minimizedEditorRef.current = editor;
	};

	const onFullscreenEditorMount = (editor) => {
		fullscreenEditorRef.current = editor;
	};

	const onSwitchPropertiesMode = () => {
		setIsAlertOpen(false);
		setTimeout(() => {
			const a = { ...action };
			a.InSchema = null;
			a.OutSchema = null;
			if (mode === 'mappings') {
				a.Mapping = null;
				a.Transformation = {
					Source: rawTransformationFunctions[selectedLanguage].replace(
						'$parameterName',
						defaultTransformationParameterByTarget[actionType.Target],
					),
					Language: selectedLanguage,
				};
				setAction(a);
				setMode('transformation');
			} else {
				a.Mapping = flattenSchema(actionType.OutputSchema);
				a.Transformation = null;
				setAction(a);
				setMode('mappings');
			}
		}, 150);
	};

	const onChangeTransformationFunction = (source: string) => {
		const a = { ...action };
		a.Transformation = {
			Source: source,
			Language: selectedLanguage,
		};
		setAction(a);
	};

	const onTransformationLanguageSelect = (e) => {
		setSelectedLanguage(e.detail.item.value);
		setIsAlertOpen(true);
	};

	const onLanguageChange = (e) => {
		const language = e.detail.item.value;
		setSelectedLanguage(language);
	};

	const updateProperty = async (name, value) => {
		let errorMessage = '';
		if (value !== '') {
			try {
				errorMessage = await api.validateExpression(
					value,
					actionType.InputSchema,
					action.Mapping![name].full.type,
					action.Mapping![name].full.nullable,
				);
			} catch (err) {
				showError(err);
				return;
			}
		}
		const updatedAction = updateMappingProperty(action, name, value, errorMessage);
		setAction(updatedAction);
	};

	const onUpdateProperty = async (e) => {
		const target = e.target;
		let { name, value } = target;
		const oldValue = action.Mapping![name].value;
		const isPasted = Math.abs(oldValue.length - value.length) > 1;
		const isBackspaced = oldValue.length > value.length;
		const isEqual = oldValue.length === value.length;
		let newCursorPosition: number | undefined;
		if (!isPasted && !isBackspaced && !isEqual) {
			const currentCursorPosition = target.shadowRoot.querySelector('input').selectionStart;
			const { autocompleted, cursorPosition } = autocompleteExpression(value, currentCursorPosition);
			value = autocompleted;
			newCursorPosition = cursorPosition;
		}
		await updateProperty(name, value);
		if (newCursorPosition) {
			setTimeout(() => {
				target.setSelectionRange(newCursorPosition, newCursorPosition);
			});
		}
	};

	const onSelectProperty = async (input, value) => {
		if (input.name === 'identityProperty') {
			const a = { ...action };
			a.IdentityProperty = value;
			setAction(a);
			return;
		} else if (input.name === 'timestampProperty') {
			const a = { ...action };
			a.TimestampProperty = value;
			if (value === '') {
				a.TimestampFormat = '';
			}
			setAction(a);
			return;
		}
		await updateProperty(input.name, value);
	};

	const onUpdateIdentityProperty = async (e) => {
		const target = e.target;
		let { value } = target;
		const a = { ...action };
		a.IdentityProperty = value;
		setAction(a);
	};

	const onUpdateTimestampProperty = async (e) => {
		const target = e.target;
		let { value } = target;
		const a = { ...action };
		a.TimestampProperty = value;
		if (value === '') {
			a.TimestampFormat = '';
		}
		setAction(a);
	};

	const onChangeTimestampFormat = (e) => {
		const a = { ...action };
		a.TimestampFormat = timestampFormats[e.target.value];
		setAction(a);
	};

	const onOpenFullscreenTransformation = () => {
		setIsFullscreenTransformationOpen(true);
		// wait while the full screen transformation becomes visible.
		setTimeout(() => {
			fullscreenEditorRef.current.focus();
			fullscreenEditorRef.current.setPosition(minimizedEditorRef.current.getPosition());
		}, 100);
	};

	const onCloseFullscreen = () => {
		setIsFullscreenTransformationOpen(false);
	};

	if (transformationLanguages == null) {
		return null;
	}

	let content: ReactNode | null = null;
	if (mode === 'mappings') {
		const workspace = workspaces.find((w) => w.ID === selectedWorkspace);
		const mappings: ReactNode[] = [];
		for (const k in action.Mapping) {
			// hide anonymous identifiers and their parent properties.
			const isIdentifier = isIdentifierProperty(k, workspace.AnonymousIdentifiers.Priority);
			if (isIdentifier) {
				continue;
			}
			mappings.push(
				<div
					key={k}
					className='mapping'
					data-key={k}
					style={
						{
							'--mapping-indentation': `${action.Mapping![k].indentation! * 30}px`,
						} as React.CSSProperties
					}
				>
					<ComboBoxInput
						comboBoxListRef={propertiesListRef}
						onInput={onUpdateProperty}
						value={action.Mapping[k].value}
						name={k}
						disabled={isMappingSectionDisabled || action.Mapping[k].disabled === true}
						className='inputProperty'
						size='small'
						error={action.Mapping[k].error}
					>
						{action.Mapping[k].required && (
							<div className='propertyIcon' slot='prefix'>
								<SlTooltip content='Required' hoist>
									<SlIcon name='asterisk' className='isRequiredIcon' />
								</SlTooltip>
							</div>
						)}
						{workspace.Identifiers.includes(k) && (
							<div className='propertyIcon' slot='prefix'>
								<SlTooltip content='Used as identifier' hoist>
									<SlIcon name='person-check' className='isIdentifierIcon' />
								</SlTooltip>
							</div>
						)}
					</ComboBoxInput>
					<div className='arrow'>
						<SlIcon name='arrow-right' />
					</div>
					<SlInput
						readonly
						disabled
						size='small'
						value={k}
						type='text'
						name={k}
						className={`outputProperty${action.Mapping![k].indentation! > 0 ? ' indented' : ''}`}
					/>
				</div>,
			);
		}
		content = (
			<div className='mappings'>
				{isMappingSectionDisabled && (
					<SlAlert variant='danger' className='mappingsDisabledAlert' open>
						<SlIcon slot='icon' name='exclamation-circle' />
						{disabledReason}
					</SlAlert>
				)}
				<div>{mappings}</div>
			</div>
		);
	} else if (mode === 'transformation') {
		const isTransformationLanguageDeprecated = !transformationLanguages.includes(selectedLanguage);
		const fullscreenButton = (
			<SlButton onClick={onOpenFullscreenTransformation} variant='primary'>
				Edit...
			</SlButton>
		);
		content = (
			<div className='transformation'>
				<EditorWrapper
					language={selectedLanguage}
					languageChoices={transformationLanguages}
					actions={fullscreenButton}
					onLanguageChange={onLanguageChange}
					height={400}
					name='actionTransformationEditor'
					value={action.Transformation!.Source}
					onChange={(source) => onChangeTransformationFunction(source!)}
					isReadOnly={true}
					onClick={onOpenFullscreenTransformation}
					onMount={onMinimizedEditorMount}
				/>
				{isTransformationLanguageDeprecated && (
					<SlAlert variant='danger' className='languageDeprecatedAlert' open>
						<SlIcon slot='icon' name='exclamation-circle' />
						{selectedLanguage} is not supported anymore
					</SlAlert>
				)}
				<FullscreenTransformation
					isFullscreenTransformationOpen={isFullscreenTransformationOpen}
					onCloseFullscreen={onCloseFullscreen}
					selectedLanguage={selectedLanguage}
					transformationLanguages={transformationLanguages}
					onLanguageChange={onLanguageChange}
					value={action.Transformation!.Source}
					onChangeTransformationFunction={onChangeTransformationFunction}
					inputSchema={actionType.InputSchema}
					outputSchema={actionType.OutputSchema}
					onMount={onFullscreenEditorMount}
				/>
			</div>
		);
	}

	let actionButton: ReactNode;
	if (mode === 'transformation') {
		actionButton = (
			<SlButton variant='neutral' size='small' onClick={() => setIsAlertOpen(true)}>
				<SlIcon name='shuffle' slot='prefix'></SlIcon>
				Switch to mappings
			</SlButton>
		);
	} else {
		const isTransformationSupported = transformationLanguages.length > 0;
		if (isTransformationAllowed && isTransformationSupported) {
			actionButton = (
				<SlDropdown className='switchToTransformationDropdown'>
					<SlButton slot='trigger' variant='neutral' size='small' caret>
						<SlIcon name='file-earmark-code' slot='prefix'></SlIcon>
						Switch to transformation function
					</SlButton>
					<SlMenu onSlSelect={onTransformationLanguageSelect}>
						{transformationLanguages.map((language) => (
							<SlMenuItem key={language} value={language}>
								{getLanguageLogo(language)}
								{language}
							</SlMenuItem>
						))}
					</SlMenu>
				</SlDropdown>
			);
		}
	}

	return (
		<>
			<Section
				ref={ref}
				title='Properties'
				description='The relation between the event properties and the action type properties'
				actions={actionButton}
				padded={false}
				className={mode}
			>
				{connection.isFile && connection.isSource && (
					<div className='specialProperties'>
						<div className='identityProperty'>
							<div className='label'>
								Identity<span className='asterisk'>*</span>:
							</div>
							<ComboBoxInput
								comboBoxListRef={propertiesListRef}
								onInput={onUpdateIdentityProperty}
								value={action.IdentityProperty!}
								name='identityProperty'
								disabled={isMappingSectionDisabled}
								className='inputProperty'
								size='small'
							/>
						</div>
						<div className='timestampProperty'>
							<div className='timestamp'>
								<div className='label'>Timestamp:</div>
								<ComboBoxInput
									comboBoxListRef={propertiesListRef}
									onInput={onUpdateTimestampProperty}
									value={action.TimestampProperty!}
									name='timestampProperty'
									disabled={isMappingSectionDisabled}
									className='inputProperty'
									size='small'
								/>
							</div>
							<div className='format'>
								<div className='label'>with format:</div>
								<SlSelect
									onSlChange={onChangeTimestampFormat}
									value={
										action.TimestampProperty
											? Object.keys(timestampFormats).find(
													(key) => timestampFormats[key] === action.TimestampFormat,
											  )
											: ''
									}
									name='timestampFormat'
									disabled={action.TimestampProperty == null || action.TimestampProperty === ''}
									size='small'
								>
									<SlOption value='standard'>2006-01-02 15:04:05</SlOption>
									<SlOption value='rfc3339'>2006-01-02T15:04:05Z07:00</SlOption>
									<SlOption value='rfc3339WithNanoseconds'>
										2006-01-02T15:04:05.999999999Z07:00
									</SlOption>
									<SlOption value='dateOnly'>2006-01-02</SlOption>
								</SlSelect>
							</div>
						</div>
					</div>
				)}
				{content}
				<ComboBoxList
					ref={propertiesListRef}
					items={getSchemaComboboxItems(actionType.InputSchema)}
					onSelect={onSelectProperty}
				/>
			</Section>
			<AlertDialog
				variant='danger'
				isOpen={isAlertOpen}
				onClose={() => setIsAlertOpen(false)}
				title={'You will lose your work'}
				actions={
					<>
						<SlButton variant='neutral' onClick={() => setIsAlertOpen(false)}>
							Cancel
						</SlButton>
						<SlButton variant='danger' onClick={onSwitchPropertiesMode}>
							Continue
						</SlButton>
					</>
				}
			>
				<div style={{ textAlign: 'center' }}>
					{mode === 'mappings' ? (
						<p>
							If you switch to the transformation function you will <b>PERMANENTLY</b> lose the mappings
							you have currently created
						</p>
					) : (
						<p>
							If you switch to the mappings you will <b>PERMANENTLY</b> lose the transformation code you
							have currently written
						</p>
					)}
				</div>
			</AlertDialog>
		</>
	);
});

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

interface FullscreenTransformationProps {
	isFullscreenTransformationOpen: boolean;
	onCloseFullscreen: () => void;
	selectedLanguage: string;
	transformationLanguages: string[];
	value: string;
	onLanguageChange: (e: any) => void;
	onChangeTransformationFunction: (source: string) => void;
	inputSchema: ObjectType;
	outputSchema: ObjectType;
	onMount?: (editor: any) => void;
}

const FullscreenTransformation = ({
	onCloseFullscreen,
	isFullscreenTransformationOpen,
	selectedLanguage,
	transformationLanguages,
	onLanguageChange,
	value,
	onChangeTransformationFunction,
	inputSchema,
	outputSchema,
	onMount,
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

	const { connection } = useContext(actionContext);
	const { showError, api } = useContext(AppContext);
	const { action, actionType } = useContext(ActionContext);

	const firstNameIdentifier = useRef<string>('');
	const lastNameIdentifier = useRef<string>('');
	const emailIdentifier = useRef<string>('');
	const idIdentifier = useRef<string>('');
	const lastExecutedSample = useRef<Sample>(null);
	const lastExecutedEvent = useRef<EventListenerEvent>(null);

	const collectEvents = (newly: EventListenerEvent[]) => {
		setEvents((prevEvents) => [...prevEvents, ...newly]);
	};

	useEventListener(0, true, collectEvents);

	useEffect(() => {
		const fetchSamples = async () => {
			let samples: Sample[];
			if (actionType.Target === 'Users') {
				if (connection.isFile && connection.isSource) {
					let res: RecordsResponse;
					try {
						res = await api.workspaces.connections.records(connection.id, action.Path, action.Sheet, 20);
					} catch (err) {
						showError(err);
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
						showError(err);
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
				} else if (connection.isApp && connection.isDestination) {
					const properties: string[] = [];
					for (const prop of inputSchema.properties) {
						properties.push(prop.name);
					}
					let res: FindUsersResponse;
					try {
						res = await api.workspaces.users.find(null, properties, 0, 20);
					} catch (err) {
						showError(err);
						return;
					}
					if (res.count === 0) {
						return;
					}
					const smpls: Sample[] = [];
					for (const u of res.users) {
						const sample = {};
						for (let i = 0; i < res.schema.properties.length; i++) {
							const propertyName = res.schema.properties[i].name;
							sample[propertyName] = {
								value: u[i],
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
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);
		let res: TransformationPreviewResponse;
		try {
			res = await api.transformationPreview(
				normalizeSample(sample),
				actionType.InputSchema,
				actionType.OutputSchema,
				null,
				action.Transformation,
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
					showError(err);
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
		setOutputError('');
		setIsOutputSchemaSelected(false);
		setIsExecuting(true);
		let res: EventPreviewResponse;
		try {
			res = await api.workspaces.connections.eventPreview(
				connection.id,
				actionType.EventType,
				event.full,
				null,
				action.Transformation,
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
					showError(err);
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
			showError(err);
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
			<div className='panelSchema'>
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
			<div className='samples'>
				{Array.from(samples.entries()).map(([i, s]) => {
					return (
						<Accordion
							key={i}
							isOpen={JSON.stringify(s) === JSON.stringify(selectedSample)}
							summary={
								<div
									className={`sample${
										lastExecutedSample.current &&
										JSON.stringify(lastExecutedSample.current) === JSON.stringify(s)
											? ' lastExecuted'
											: ''
									}`}
									onClick={(e) => onSampleClick(e, s)}
								>
									<div className='sampleName'>
										{actionType.Target === 'Users' ? (
											<>
												{idIdentifier.current && (
													<div className='sampleID'>{s[idIdentifier.current].value}</div>
												)}
												<div>
													<div className='sampleFullName'>
														{firstNameIdentifier.current && lastNameIdentifier.current
															? s[firstNameIdentifier.current].value +
															  ' ' +
															  s[lastNameIdentifier.current].value
															: `Sample ${i}`}
													</div>
													{emailIdentifier.current && (
														<div className='sampleEmail'>
															{s[emailIdentifier.current].value}
														</div>
													)}
												</div>
											</>
										) : (
											''
										)}
									</div>
									<div className='executeButton'>
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
								<div className='sampleSource'>
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
			<div className='queryExecution'>
				<SlIcon name='database-down' />
				<p className='queryExecutionText'>Execute the query to retrieve the samples</p>
				<SlButton className='queryExecutionButton' variant='primary' onClick={onQuery}>
					Execute the query
				</SlButton>
			</div>
		);
	} else if (connection.isApp && connection.isDestination && actionType.Target === 'Events') {
		const reversedEvents: EventListenerEvent[] = [...events].reverse();
		inputPanelContent = (
			<div className='eventListener'>
				<div className='eventList'>
					<div className='body'>
						{events.length === 0 && (
							<div className='noEvents'>
								Listening for new events{' '}
								<span className='loadingEllipsis'>
									<span className='ellipsis1'>.</span>
									<span className='ellipsis2'>.</span>
									<span className='ellipsis3'>.</span>
								</span>
							</div>
						)}
						{reversedEvents.map((e) => {
							return (
								<Accordion
									key={e.id}
									isOpen={JSON.stringify(e) === JSON.stringify(selectedEvent)}
									summary={
										<div
											className={`event${
												selectedEvent && selectedEvent.id === e.id ? ' selected' : ''
											}`}
											onClick={(evt) => onEventClick(evt, e)}
										>
											<div className='name'>{e.type}</div>
											<div className='time'>{new Date(e.time).toLocaleString()}</div>
											<SlIconButton
												className='runButton'
												name='play-circle'
												onClick={(evt) => {
													onExecuteEvent(e);
													evt.stopPropagation();
												}}
											/>
										</div>
									}
									details={
										<div className='eventSource'>
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
			<div className='noSample'>
				<SlIcon name='x-lg' />
				<p className='noSampleText'>This connection cannot retrieve samples to test the transformation</p>
			</div>
		);
	}

	return (
		<div className={`fullscreenTransformation${isFullscreenTransformationOpen ? ' isOpen' : ''}`}>
			<SlSplitPanel style={{ '--min': '0%', '--max': '800px' } as React.CSSProperties}>
				<div className='leftPanel' slot='start'>
					<SlSplitPanel>
						<div className='inputPanel' slot='start'>
							<div className='panelTitleWrapper'>
								<div className='panelTitle'>{InputPanelTitle}</div>
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
							<div className='panelContent'>{inputPanelContent}</div>
						</div>
						<div className='outputPanel' slot='end'>
							<div className='panelTitleWrapper'>
								<div className='panelTitle'>{OutputPanelTitle}</div>
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
							<div className='panelContent'>
								{isOutputSchemaSelected ? (
									<div className='panelSchema'>
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
									<div className='outputCode'>
										<SlTooltip content='Clear' placement='left' onClick={onClear}>
											<SlIconButton className='clearButton' name='x-lg' />
										</SlTooltip>
										{outputError !== '' ? (
											<div className='outputError'>{outputError}</div>
										) : (
											<div className='outputSuccess'>
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
									<div className='outputPlaceholder'>
										<SlIcon name='play-circle' />
										<p className='outputPlaceholderText'>
											Run the transformation on a sample to see the resulting output
										</p>
									</div>
								)}
							</div>
						</div>
					</SlSplitPanel>
				</div>
				<div className='rightPanel' slot='end'>
					<div slot='start' className='editorPanel'>
						<EditorWrapper
							language={selectedLanguage}
							languageChoices={transformationLanguages}
							onLanguageChange={onLanguageChange}
							name='fullscreenActionTransformationEditor'
							value={value}
							onChange={(source) => onChangeTransformationFunction(source!)}
							onMount={onMount}
						/>
						<div className='closeButtonWrapper'>
							<SlButton onClick={onCloseFullscreen} variant='primary'>
								Exit editor
							</SlButton>
						</div>
					</div>
				</div>
			</SlSplitPanel>
		</div>
	);
};

function fromPhysicalTypeToJavascriptType(typeName: string) {
	// TODO: add additional information (property is nullable, property values,
	//  property length). This needs the full type definition and not the
	// type name only.
	switch (typeName) {
		case 'Boolean':
			return 'Boolean';
		case 'Int':
		case 'Int8':
		case 'Int16':
		case 'Int24':
		case 'UInt':
		case 'UInt8':
		case 'UInt16':
		case 'UInt24':
		case 'Float':
		case 'Float32':
			return 'Number';
		case 'Int64':
		case 'UInt64':
			return 'BigInt';
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
			console.error(`schema contains unknow property type ${typeName}`);
			return 'unknown property type';
	}
}

function fromPhysicalTypeToPythonType(typeName: string) {
	// TODO: add additional information (property is nullable, property values,
	// property length). This needs the full type definition and not the
	// type name only.
	switch (typeName) {
		case 'Boolean':
			return 'bool';
		case 'Int':
		case 'Int8':
		case 'Int16':
		case 'Int24':
		case 'Int64':
		case 'UInt':
		case 'UInt8':
		case 'UInt16':
		case 'UInt24':
		case 'UInt64':
			return 'int';
		case 'Float':
		case 'Float32':
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
			console.error(`schema contains unknow property type ${typeName}`);
			return 'unknown property type';
	}
}

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
			className={`property${isExpanded ? ' expand' : ''}${
				property.label != null && property.label !== '' ? ' hasLabel' : ''
			}`}
		>
			<div className='parentProperty'>
				<SlIcon
					name='caret-right-fill'
					onClick={() => {
						setIsExpanded(!isExpanded);
					}}
				/>
				<TransformationProperty property={property} language={language} isParent={true} />
			</div>
			<div className='subProperties'>
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
		<div className={isParent ? '' : 'property'}>
			<div className='name'>
				{isIdentifier && (
					<SlTooltip content='Used as identifier'>
						<SlIcon className='identifierIcon' name='person-check' />
					</SlTooltip>
				)}
				{property.required && (
					<SlTooltip content='Required'>
						<SlIcon className='requiredIcon' name='asterisk' />
					</SlTooltip>
				)}
				{property.name}
				{property.label != null && property.label !== '' && <span className='label'>({property.label})</span>}
				<SlCopyButton
					className='copyProperty'
					value={property.name}
					copyLabel='Click to copy'
					successLabel='✓ Copied'
					errorLabel='Copying to clipboard is not supported by your browser'
				/>
			</div>
			<div className='type'>
				{language === 'Python'
					? fromPhysicalTypeToPythonType(property.type.name)
					: fromPhysicalTypeToJavascriptType(property.type.name)}
			</div>
		</div>
	);
};

export default ActionMapping;

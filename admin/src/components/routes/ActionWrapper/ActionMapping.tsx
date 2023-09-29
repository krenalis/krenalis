import React, { useState, useRef, useContext, useEffect, forwardRef, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { updateMappingProperty, autocompleteExpression } from './Action.helpers';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import { flattenSchema, isIdentifierProperty } from '../../../lib/helpers/transformedAction';
import { rawTransformationFunctions } from './Action.constants';
import AlertDialog from '../../shared/AlertDialog/AlertDialog';
import { ComboBoxInput, ComboBoxList } from '../../shared/ComboBox/ComboBox';
import Section from '../../shared/Section/Section';
import EditorWrapper from '../../shared/EditorWrapper/EditorWrapper';
import { AppContext } from '../../../context/providers/AppProvider';
import ActionContext from '../../../context/ActionContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlAlert from '@shoelace-style/shoelace/dist/react/alert/index.js';
import { TransformationLanguagesResponse } from '../../../types/external/api';
import getLanguageLogo from '../../helpers/getLanguageLogo';

const defaultTransformationParameterByTarget = {
	Users: 'user',
	Groups: 'group',
	Events: 'event',
};

const ActionMapping = forwardRef<any>((props, ref) => {
	const [isAlertOpen, setIsAlertOpen] = useState<boolean>(false);
	const [transformationLanguages, setTransformationLanguages] = useState<string[]>();
	const [selectedLanguage, setSelectedLanguage] = useState<string>();

	const { api, showError, workspaces, selectedWorkspace } = useContext(AppContext);
	const {
		isMappingSectionDisabled,
		disabledReason,
		isTransformationAllowed,
		action,
		setAction,
		actionType,
		mode,
		setMode,
	} = useContext(ActionContext);

	const propertiesListRef = useRef(null);

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
		if (selectedLanguage == null) {
			return;
		}
		const a = { ...action };
		a.Transformation = {
			Source: rawTransformationFunctions[selectedLanguage].replace(
				'$parameterName',
				defaultTransformationParameterByTarget[actionType.Target],
			),
			Language: selectedLanguage,
		};
		setAction(a);
	}, [selectedLanguage]);

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
		await updateProperty(input.name, value);
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

			// hide properties already mapped in the identifiers section.
			let isAlreadyMappedInIdentity = false;
			if (actionType.Fields.includes('Identifiers')) {
				for (const [, identifier] of action.Identifiers!) {
					if (identifier.value === k) {
						isAlreadyMappedInIdentity = true;
						break;
					}
				}
			}
			if (isAlreadyMappedInIdentity) continue;

			mappings.push(
				<div
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
						{action.Mapping[k].required && <SlIcon name='asterisk' slot='prefix'></SlIcon>}
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
				<ComboBoxList
					ref={propertiesListRef}
					items={getSchemaComboboxItems(actionType.InputSchema)}
					onSelect={onSelectProperty}
				/>
			</div>
		);
	} else if (mode === 'transformation') {
		const isTransformationLanguageDeprecated = !transformationLanguages.includes(selectedLanguage);
		content = (
			<div className='transformation'>
				<EditorWrapper
					language={selectedLanguage}
					languageChoices={transformationLanguages}
					onLanguageChange={onLanguageChange}
					height={400}
					value={action.Transformation!.Source}
					onChange={(source) => onChangeTransformationFunction(source!)}
				/>
				{isTransformationLanguageDeprecated && (
					<SlAlert variant='danger' className='languageDeprecatedAlert' open>
						<SlIcon slot='icon' name='exclamation-circle' />
						{selectedLanguage} is not supported anymore
					</SlAlert>
				)}
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
							<SlMenuItem value={language}>
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
			>
				{content}
			</Section>
			{createPortal(
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
								If you switch to the transformation function you will <b>PERMANENTLY</b> lose the
								mappings you have currently created
							</p>
						) : (
							<p>
								If you switch to the mappings you will <b>PERMANENTLY</b> lose the transformation code
								you have currently written
							</p>
						)}
					</div>
				</AlertDialog>,
				document.body,
			)}
		</>
	);
});

export default ActionMapping;

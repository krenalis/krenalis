import React, { useState, useRef, useContext, useEffect, forwardRef, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { updateMappingProperty, autocompleteExpression } from './Action.helpers';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import { flattenSchema, isIdentifierProperty } from '../../../lib/helpers/transformedAction';
import { rawTransformationFunction } from './Action.constants';
import AlertDialog from '../../shared/AlertDialog/AlertDialog';
import { ComboBoxInput, ComboBoxList } from '../../shared/ComboBox/ComboBox';
import Section from '../../shared/Section/Section';
import EditorWrapper from '../../shared/EditorWrapper/EditorWrapper';
import { AppContext } from '../../../context/providers/AppProvider';
import ActionContext from '../../../context/ActionContext';
import { SlButton, SlIcon, SlInput, SlAlert } from '@shoelace-style/shoelace/dist/react/index.js';

const defaultTransformationParameterByTarget = {
	Users: 'user',
	Groups: 'group',
	Events: 'event',
};

const ActionMapping = forwardRef<any>((props, ref) => {
	const [isAlertOpen, setIsAlertOpen] = useState<boolean>(false);

	const { api, showError, workspace } = useContext(AppContext);
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

	const defaultTransformationFunction = useRef('');
	const propertiesListRef = useRef(null);

	useEffect(() => {
		if (action.Transformation != null) {
			setMode('transformation');
		} else {
			setMode('mappings');
		}
		defaultTransformationFunction.current = rawTransformationFunction.replace(
			'$parameterName',
			defaultTransformationParameterByTarget[actionType.Target]
		);
	}, []);

	const onSwitchPropertiesMode = () => {
		setIsAlertOpen(false);
		setTimeout(() => {
			const a = { ...action };
			a.InSchema = null;
			a.OutSchema = null;
			if (mode === 'mappings') {
				a.Mapping = null;
				a.Transformation = { Func: defaultTransformationFunction.current, In: [], Out: [] };
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

	const onChangeTransformationFunction = (value: string) => {
		const a = { ...action };
		a.Transformation!.Func = value;
		setAction(a);
	};

	const updateProperty = async (name, value) => {
		let errorMessage = '';
		if (value !== '') {
			try {
				errorMessage = await api.validateExpression(
					value,
					actionType.InputSchema,
					action.Mapping![name].full.type,
					action.Mapping![name].full.nullable
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

	let content: ReactNode | null = null;
	if (mode === 'mappings') {
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
				</div>
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
		content = (
			<div className='transformation'>
				<EditorWrapper
					defaultLanguage='python'
					height={400}
					value={action.Transformation!.Func}
					onChange={(value) => onChangeTransformationFunction(value!)}
				/>
			</div>
		);
	}

	return (
		<>
			<Section
				ref={ref}
				title='Properties'
				description='The relation between the event properties and the action type properties'
				actions={
					isTransformationAllowed && (
						<SlButton variant='neutral' size='small' onClick={() => setIsAlertOpen(true)}>
							{mode === 'mappings' ? (
								<SlIcon name='filetype-py' slot='prefix'></SlIcon>
							) : (
								<SlIcon name='shuffle' slot='prefix'></SlIcon>
							)}
							{mode === 'mappings' ? 'Switch to transformation function' : 'Switch to mappings'}
						</SlButton>
					)
				}
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
				document.body
			)}
		</>
	);
});

export default ActionMapping;

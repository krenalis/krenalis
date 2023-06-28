import { useState, useRef, useContext, useEffect, forwardRef } from 'react';
import { createPortal } from 'react-dom';
import {
	getDefaultMappings,
	rawTransformationFunction,
	updateMappingProperty,
	getSchemaComboboxItems,
	addPropertyToActionSchema,
	removePropertyFromActionSchema,
	getExpressionVariables,
} from './Action.helpers';
import AlertDialog from '../../../common/AlertDialog/AlertDialog';
import { ComboBoxInput, ComboBoxList } from '../../../common/ComboBox/ComboBox';
import Section from '../../../common/Section/Section';
import EditorWrapper from '../../../common/EditorWrapper/EditorWrapper';
import { ConnectionContext } from '../../../../providers/ConnectionProvider';
import {
	SlButton,
	SlIcon,
	SlDialog,
	SlInput,
	SlIconButton,
	SlAlert,
} from '@shoelace-style/shoelace/dist/react/index.js';

const defaultTransformationParameterByTarget = {
	Users: 'user',
	Groups: 'group',
	Events: 'event',
};

const ActionMapping = forwardRef(
	({ disabled, disabledReason, action, setAction, inputSchema, outputSchema, actionType, mode, setMode }, ref) => {
		const [isAlertOpen, setIsAlertOpen] = useState(false);
		const [isInputSchemaDialogOpen, setIsInputSchemaDialogOpen] = useState(false);
		const [isOutputSchemaDialogOpen, setIsOutputSchemaDialogOpen] = useState(false);

		const { connection } = useContext(ConnectionContext);

		const defaultTransformationFunction = useRef('');
		const propertiesListRef = useRef(null);

		useEffect(() => {
			if (action.Mapping != null) {
				setMode('mappings');
			} else {
				setMode('transformation');
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
					a.PythonSource = defaultTransformationFunction.current;
					setAction(a);
					setMode('transformation');
				} else {
					a.Mapping = getDefaultMappings(outputSchema);
					a.PythonSource = null;
					setAction(a);
					setMode('mappings');
				}
			}, 150);
		};

		const onChangeTransformationPythonSource = (value) => {
			const a = { ...action };
			a.PythonSource = value;
			setAction(a);
		};

		const onRemoveTransformationProperty = (side, propertyName) => {
			const updatedAction = removePropertyFromActionSchema(action, side, propertyName);
			setAction(updatedAction);
		};

		const onAddTransformationProperty = (side, property) => {
			const updatedAction = addPropertyToActionSchema(action, side, property);
			setAction(updatedAction);
		};

		const onUpdateProperty = (e) => {
			const { name, value } = e.currentTarget || e.target;
			const updatedAction = updateMappingProperty(action, name, value);
			setAction(updatedAction);
		};

		const onSelectPropertiesListItem = (input, value) => {
			const updatedAction = updateMappingProperty(action, input.name, value);
			setAction(updatedAction);
		};

		let mappingContent = null;
		if (mode === 'mappings') {
			const mappings = [];
			const defaultMappings = getDefaultMappings(inputSchema);
			for (const k of Object.keys(action.Mapping)) {
				let error;
				const value = action.Mapping[k].value;
				if (!disabled && value !== '') {
					const variables = getExpressionVariables(value);
					for (const variable of variables) {
						const doesValueExist = defaultMappings[variable] != null;
						if (!doesValueExist) {
							error = `"${variable}" does not exist in ${connection.type.toLowerCase()}'s schema`;
							break;
						}
					}
				}
				mappings.push(
					<div
						className='mapping'
						data-key={k}
						style={{
							'--mapping-indentation': `${action.Mapping[k].indentation * 30}px`,
						}}
					>
						<ComboBoxInput
							comboBoxListRef={propertiesListRef}
							onInput={onUpdateProperty}
							value={value}
							name={k}
							disabled={disabled || action.Mapping[k].disabled}
							className='inputProperty'
							size='small'
							error={error}
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
							onSlInput={null}
							className={`outputProperty${action.Mapping[k].indentation > 0 ? ' indented' : ''}`}
						/>
					</div>
				);
			}
			mappingContent = (
				<div className='mappings'>
					{disabled && (
						<SlAlert variant='danger' className='mappingsDisabledAlert' open>
							<SlIcon slot='icon' name='exclamation-circle' />
							{disabledReason}
						</SlAlert>
					)}
					{mappings}
					<ComboBoxList
						ref={propertiesListRef}
						items={getSchemaComboboxItems(inputSchema)}
						onSelect={onSelectPropertiesListItem}
					/>
				</div>
			);
		} else {
			mappingContent = (
				<div className='transformation'>
					<div className='inputProperties'>
						{action.InSchema &&
							action.InSchema.properties.map((p) => {
								return (
									<div className='property'>
										<div className='name'>{p.name}</div>
										<div className='type'>{p.type.name}</div>
										<SlButton
											className='removeProperty'
											size='small'
											variant='danger'
											outline
											onClick={() => onRemoveTransformationProperty('input', p.name)}
										>
											<SlIcon name='trash'></SlIcon>
										</SlButton>
									</div>
								);
							})}
						<SlButton
							className='addProperty'
							size='small'
							variant='default'
							onClick={() => setIsInputSchemaDialogOpen(true)}
						>
							Add new property...
						</SlButton>
					</div>
					<EditorWrapper
						defaultLanguage='python'
						height={400}
						value={action.PythonSource}
						onChange={(value) => onChangeTransformationPythonSource(value)}
					/>
					<div className='outputProperties'>
						{action.OutSchema &&
							action.OutSchema.properties.map((p) => {
								return (
									<div className='property'>
										<div className='name'>{p.name}</div>
										<div className='type'>{p.type.name}</div>
										<SlButton
											className='removeProperty'
											size='small'
											variant='danger'
											outline
											onClick={() => onRemoveTransformationProperty('output', p.name)}
										>
											<SlIcon name='trash'></SlIcon>
										</SlButton>
									</div>
								);
							})}
						<SlButton
							className='addProperty'
							size='small'
							variant='default'
							onClick={() => setIsOutputSchemaDialogOpen(true)}
						>
							Add new property...
						</SlButton>
					</div>
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
						<SlButton variant='neutral' size='small' onClick={() => setIsAlertOpen(true)}>
							{mode === 'mappings' ? (
								<SlIcon name='filetype-py' slot='prefix'></SlIcon>
							) : (
								<SlIcon name='shuffle' slot='prefix'></SlIcon>
							)}
							{mode === 'mappings' ? 'Switch to transformation function' : 'Switch to mappings'}
						</SlButton>
					}
					padded={false}
				>
					{mappingContent}
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
									If you switch to the mappings you will <b>PERMANENTLY</b> lose the transformation
									code you have currently written
								</p>
							)}
						</div>
					</AlertDialog>,
					document.body
				)}
				{action.PythonSource != null &&
					createPortal(
						<SlDialog
							className='inputSchemaDialog'
							label='Input properties'
							open={isInputSchemaDialogOpen}
							onSlRequestClose={() => setIsInputSchemaDialogOpen(false)}
							style={{ '--width': '700px' }}
						>
							{inputSchema.properties.map((p) => {
								const isUsed =
									action.InSchema &&
									action.InSchema.properties.findIndex((prop) => prop.name === p.name) !== -1;
								return (
									<div
										className={`property${isUsed ? ' used' : ''}${
											p.label != null && p.label !== '' ? ' hasLabel' : ''
										}`}
									>
										<div>
											{p.label != null && p.label !== '' && (
												<div className='label'>{p.label}</div>
											)}
											<div className='name'>{p.name}</div>
											<div className='type'>{p.type.name}</div>
										</div>
										{!isUsed && (
											<SlIconButton
												name='plus-circle'
												label='Add property'
												onClick={() => onAddTransformationProperty('input', p)}
											/>
										)}
									</div>
								);
							})}
						</SlDialog>,
						document.body
					)}
				{action.PythonSource != null &&
					createPortal(
						<SlDialog
							className='outputSchemaDialog'
							label='Output properties'
							open={isOutputSchemaDialogOpen}
							onSlRequestClose={() => setIsOutputSchemaDialogOpen(false)}
							style={{ '--width': '700px' }}
						>
							{outputSchema.properties.map((p) => {
								const isUsed =
									action.OutSchema &&
									action.OutSchema.properties.findIndex((prop) => prop.name === p.name) !== -1;
								return (
									<div
										className={`property${isUsed ? ' used' : ''}${
											p.label != null && p.label !== '' ? ' hasLabel' : ''
										}`}
									>
										<div>
											{p.label != null && p.label !== '' && (
												<div className='label'>{p.label}</div>
											)}
											<div className='name'>{p.name}</div>
											<div className='type'>{p.type.name}</div>
										</div>
										{!isUsed && (
											<SlIconButton
												name='plus-circle'
												label='Add property'
												onClick={() => onAddTransformationProperty('output', p)}
											/>
										)}
									</div>
								);
							})}
						</SlDialog>,
						document.body
					)}
			</>
		);
	}
);

export default ActionMapping;

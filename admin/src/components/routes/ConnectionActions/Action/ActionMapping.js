import { useState, useRef, useContext, useEffect, forwardRef } from 'react';
import { createPortal } from 'react-dom';
import { updateMappingProperty, getSchemaComboboxItems } from './Action.helpers';
import { flattenSchema, getExpressionVariables } from '../../../../lib/connections/action';
import { rawTransformationFunction } from './Action.constants';
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
	(
		{ disabled, disabledReason, action, setAction, inputSchema, outputSchema, actionType, mode, setMode, fields },
		ref
	) => {
		const [isAlertOpen, setIsAlertOpen] = useState(false);
		const [isInputSchemaDialogOpen, setIsInputSchemaDialogOpen] = useState(false);
		const [isOutputSchemaDialogOpen, setIsOutputSchemaDialogOpen] = useState(false);

		const { connection } = useContext(ConnectionContext);

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
					a.Mapping = flattenSchema(outputSchema);
					a.Transformation = null;
					setAction(a);
					setMode('mappings');
				}
			}, 150);
		};

		const onChangeTransformationFunction = (value) => {
			const a = { ...action };
			a.Transformation.Func = value;
			setAction(a);
		};

		const onRemoveTransformationProperty = (side, propertyName) => {
			const a = { ...action };
			if (side === 'input') {
				const inputProperties = a.Transformation.In;
				a.Transformation.In = inputProperties.filter((p) => p !== propertyName);
			} else {
				const outputProperties = a.Transformation.Out;
				a.Transformation.Out = outputProperties.filter((p) => p !== propertyName);
			}
			setAction(a);
		};

		const onAddTransformationProperty = (side, propertyName) => {
			const a = { ...action };
			if (side === 'input') {
				a.Transformation.In.push(propertyName);
			} else {
				a.Transformation.Out.push(propertyName);
			}
			setAction(a);
		};

		const onUpdateProperty = (e) => {
			const { name, value } = e.currentTarget || e.target;
			const updatedAction = updateMappingProperty(action, name, value);
			setAction(updatedAction);
		};

		const onSelectProperty = (input, value) => {
			const updatedAction = updateMappingProperty(action, input.name, value);
			setAction(updatedAction);
		};

		let content = null;
		if (mode === 'mappings') {
			const mappings = [];
			const defaultMappings = flattenSchema(inputSchema);
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
				let isAlreadyMappedInIdentity = false;
				if (fields.includes('Identifiers')) {
					for (const [inputIdentifiers, outputIdentifiers] of action.Identifiers) {
						if (outputIdentifiers === k) {
							isAlreadyMappedInIdentity = true;
						}
					}
				}
				if (isAlreadyMappedInIdentity) continue;
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
							disabled={disabled || action.Mapping[k].disabled === true}
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
			content = (
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
						onSelect={onSelectProperty}
					/>
				</div>
			);
		} else if (mode === 'transformation') {
			content = (
				<div className='transformation'>
					<div className='inputProperties'>
						{action.Transformation.In.map((propertyName) => {
							const fullProperty = flattenSchema(inputSchema)[propertyName];
							return (
								<div className='property'>
									<div className='name'>{fullProperty.full.name}</div>
									<div className='type'>{fullProperty.full.type.name}</div>
									<SlButton
										className='removeProperty'
										size='small'
										variant='danger'
										outline
										onClick={() => onRemoveTransformationProperty('input', fullProperty.full.name)}
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
						value={action.Transformation.Func}
						onChange={(value) => onChangeTransformationFunction(value)}
					/>
					<div className='outputProperties'>
						{action.Transformation.Out.map((propertyName) => {
							const fullProperty = flattenSchema(outputSchema)[propertyName];
							return (
								<div className='property'>
									<div className='name'>{fullProperty.full.name}</div>
									<div className='type'>{fullProperty.full.type.name}</div>
									<SlButton
										className='removeProperty'
										size='small'
										variant='danger'
										outline
										onClick={() => onRemoveTransformationProperty('output', fullProperty.full.name)}
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
									If you switch to the mappings you will <b>PERMANENTLY</b> lose the transformation
									code you have currently written
								</p>
							)}
						</div>
					</AlertDialog>,
					document.body
				)}
				{mode === 'transformation' &&
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
									action.Transformation.In.findIndex((propertyName) => propertyName === p.name) !==
									-1;
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
												onClick={() => onAddTransformationProperty('input', p.name)}
											/>
										)}
									</div>
								);
							})}
						</SlDialog>,
						document.body
					)}
				{mode === 'transformation' &&
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
									action.Transformation.Out.findIndex((propertyName) => propertyName === p.name) !==
									-1;
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
												onClick={() => onAddTransformationProperty('output', p.name)}
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

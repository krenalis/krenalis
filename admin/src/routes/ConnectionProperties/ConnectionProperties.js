import { useState, useEffect } from 'react';
import './ConnectionProperties.css';
import call from '../../utils/call';
import ConnectionProperty from '../../components/ConnectionProperty/ConnectionProperty';
import TransformationNode from '../../components/TrasformationNode/TransformationNode';
import TransformationDialog from '../../components/TransformationDialog/TransformationDialog';
import { getTransformationType } from '../../utils/getTransformationType';
import { transformationFunction } from '../../assets/docs/transformationFunction';
import {
	SlButton,
	SlIcon,
	SlDialog,
	SlTooltip,
	SlIconButton,
	SlInput,
} from '@shoelace-style/shoelace/dist/react/index.js';
import Xarrow from 'react-xarrows';

const ConnectionProperties = ({ connection: c, onError, onStatuChange, isSelected }) => {
	let [inputProperties, setInputProperties] = useState([]);
	let [outputProperties, setOutputProperties] = useState([]);
	let [usedInputProperties, setUsedInputProperties] = useState([]);
	let [usedOutputProperties, setUsedOutputProperties] = useState([]);
	let [transformations, setTransformations] = useState([]);
	let [lastTransformationPosition, setLastTransformationPosition] = useState(1);
	let [inputSearchTerm, setInputSearchTerm] = useState('');
	let [outputSearchTerm, setOutputSearchTerm] = useState('');
	let [isInputDialogOpen, setIsInputDialogOpen] = useState(false);
	let [isOutputDialogOpen, setIsOutputDialogOpen] = useState(false);
	let [selectedProperty, setSelectedProperty] = useState(null);
	let [selectedTransformation, setSelectedTransformation] = useState(null);
	let [selectedPredefinedTransformation, setSelectedPredefinedTransformation] = useState(0);
	let [predefinedTransformations, setPredefinedTransformations] = useState([]);
	let [showPredefinedTransformations, setShowPredefinedTransformations] = useState(false);

	useEffect(() => {
		const fetchState = async () => {
			let err;

			// get the connection properties and the user properties.
			let connectionSchema;
			[connectionSchema, err] = await call(`/api/connections/${c.ID}/schema`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			if (connectionSchema == null) return;
			let connectionProperties = [];
			for (let p of connectionSchema.properties) {
				connectionProperties.push(p);
			}
			let userSchema;
			[userSchema, err] = await call('/admin/user-schema-properties', 'GET');
			if (err) {
				onError(err);
				return;
			}
			if (userSchema == null) return;
			let userProperties = [];
			for (let p of userSchema.properties) {
				userProperties.push(p);
			}

			// place the properties in the proper column.
			let inputProperties, outputProperties;
			if (c.Role === 'Source') {
				inputProperties = connectionProperties;
				outputProperties = userProperties;
			} else if (c.Role === 'Destination') {
				inputProperties = userProperties;
				outputProperties = connectionProperties;
			}
			setInputProperties(inputProperties);
			setOutputProperties(outputProperties);

			// get the predefined transformations.
			let predefinedTransformations;
			[predefinedTransformations, err] = await call('/admin/predefined-transformations', 'GET');
			if (err) {
				onError(err);
				return;
			}
			setPredefinedTransformations(predefinedTransformations);

			// get the transformations.
			let transformations;
			[transformations, err] = await call(`/api/connections/${c.ID}/mappings`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			if (transformations == null) return;

			// replace the predefined transformations IDs with the full
			// predefined transformations.
			for (let t of transformations) {
				if (t.PredefinedFunc != null) {
					let predefinedTransformation = predefinedTransformations.find((pt) => pt.ID === t.PredefinedFunc);
					t.PredefinedFunc = predefinedTransformation;
				}
			}

			// get the input properties and the output properties used by the
			// transformations.
			let usedInputProperties = [];
			let usedOutputProperties = [];
			for (let t of transformations) {
				for (let input of t.InProperties) {
					let isDuplicate = false;
					for (let p of usedInputProperties) {
						if (input === p.name) {
							isDuplicate = true;
							break;
						}
					}
					if (!isDuplicate) {
						let fullProperty = inputProperties.find((p) => p.name === input);
						usedInputProperties.push(fullProperty);
					}
				}
				for (let output of t.OutProperties) {
					let isDuplicate = false;
					for (let p of usedOutputProperties) {
						if (output === p.name) {
							isDuplicate = true;
							break;
						}
					}
					if (!isDuplicate) {
						let fullProperty = outputProperties.find((p) => p.name === output);
						usedOutputProperties.push(fullProperty);
					}
				}
			}
			setUsedInputProperties(usedInputProperties);
			setUsedOutputProperties(usedOutputProperties);

			// compute the positions of the transformations.
			let pos = lastTransformationPosition;
			for (let t of transformations) {
				t.Position = pos;
				pos += 1;
			}
			setTransformations(transformations);
			setLastTransformationPosition(pos);
		};

		fetchState();
	}, []);

	const onAddUsedProperty = (p, type) => {
		if (type === 'input') {
			setUsedInputProperties([...usedInputProperties, p]);
		} else {
			setUsedOutputProperties([...usedOutputProperties, p]);
		}
	};

	const onRemoveUsedProperty = (e, removedName, type) => {
		e.stopPropagation();
		if (type === 'input') {
			setUsedInputProperties(usedInputProperties.filter((p) => p.name !== removedName));
		} else {
			setUsedOutputProperties(usedOutputProperties.filter((p) => p.name !== removedName));
		}
		let trs = [];
		for (let t of transformations) {
			let transformationProperties = type === 'input' ? t.InProperties : t.OutProperties;
			let doesContainRemovedProperty =
				transformationProperties.findIndex((p) => p != null && p === removedName) !== -1;
			if (doesContainRemovedProperty) {
				if (getTransformationType(t) === 'one-to-one') continue; // remove the transformation.
				let filtered = [];
				if (t.PredefinedFunc != null) {
					// replace the removed property with 'undefined' to preserve order.
					for (let p of transformationProperties) {
						if (p != null && p === removedName) {
							filtered.push(undefined);
						} else {
							filtered.push(p);
						}
					}
				} else {
					for (let p of transformationProperties) {
						if (p !== removedName) {
							filtered.push(p);
						}
					}
				}
				if (type === 'input') {
					t.InProperties = filtered;
				} else {
					t.OutProperties = filtered;
				}
			}
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onAddOneToOneTransformation = (name, type) => {
		if (type === sp.type) {
			onError(`cannot connect two ${type} properties`);
			return;
		}
		let inputProperty, outputProperty;
		if (sp.type === 'input') {
			inputProperty = inputProperties.find((p) => p.name === sp.name);
			outputProperty = outputProperties.find((p) => p.name === name);
		} else {
			inputProperty = inputProperties.find((p) => p.name === name);
			outputProperty = outputProperties.find((p) => p.name === sp.name);
		}
		let t = {
			Position: lastTransformationPosition,
			InProperties: [inputProperty.name],
			OutProperties: [outputProperty.name],
		};
		t.CustomFunc = null;
		t.PredefinedFunc = null;
		setTransformations([...transformations, t]);
		setLastTransformationPosition(lastTransformationPosition + 1);
		setSelectedProperty(null);
	};

	const onAddPredefinedTransformation = () => {
		let t = {
			Position: lastTransformationPosition,
			InProperties: [],
			OutProperties: [],
		};
		t.CustomFunc = null;
		let pt = predefinedTransformations.find((t) => t.ID === selectedPredefinedTransformation);
		setShowPredefinedTransformations(false);
		t.PredefinedFunc = pt;
		setTransformations([...transformations, t]);
		setLastTransformationPosition(lastTransformationPosition + 1);
	};

	const onAddCustomTransformation = () => {
		let t = {
			Position: lastTransformationPosition,
			InProperties: [],
			OutProperties: [],
		};
		t.SourceCode = t.CustomFunc = { InTypes: [], OutTypes: [], Source: transformationFunction };
		t.PredefinedFunc = null;
		setTransformations([...transformations, t]);
		setLastTransformationPosition(lastTransformationPosition + 1);
	};

	const onChangeTransformation = (position, value) => {
		let trs = [...transformations];
		let i = trs.findIndex((t) => t.Position === position);
		trs[i].CustomFunc.Source = value === '' ? transformationFunction : value;
		setTransformations(trs);
	};

	const onRemoveTransformation = (position) => {
		let trs = transformations.filter((t) => t.Position !== position);
		setTransformations(trs);
		setSelectedTransformation('');
	};

	const onCustomTransformationConnect = (transformationPosition) => {
		let sp = selectedProperty;
		let trs = [];
		for (let t of transformations) {
			if (t.Position === transformationPosition) {
				if (sp.type === 'input') {
					if (t.InProperties.findIndex((property) => property === sp.name) === -1) {
						t.InProperties.push(sp.name);
					}
				}
				if (sp.type === 'output') {
					if (t.OutProperties.findIndex((property) => property === sp.name) === -1) {
						t.OutProperties.push(sp.name);
					}
				}
			}
			trs.push(t);
		}
		setTransformations(trs);
		setSelectedProperty(null);
	};

	const onPredefinedTransformationConnect = (transformationPosition, parameter) => {
		let sp = selectedProperty;
		let trs = [];
		for (let t of transformations) {
			if (t.Position === transformationPosition) {
				if (sp.type === 'input') {
					if (t.InProperties.findIndex((property) => property != null && property === sp.name) === -1) {
						let parameterIndex = t.PredefinedFunc.In.properties.findIndex((p) => p.label === parameter);
						if (t.InProperties.length === 0) {
							let parametersCount = t.PredefinedFunc.In.properties.length;
							t.InProperties = Array(parametersCount);
							t.InProperties[parameterIndex] = sp.name;
						} else {
							t.InProperties[parameterIndex] = sp.name;
						}
					}
				}
				if (sp.type === 'output') {
					if (t.OutProperties.findIndex((property) => property != null && property === sp.name) === -1) {
						let parameterIndex = t.PredefinedFunc.Out.properties.findIndex((p) => p.label === parameter);
						let parametersCount = t.PredefinedFunc.Out.properties.length;
						if (parametersCount === 1) {
							// it's possible to connect an arbitrary number of
							// output properties
							t.OutProperties.push(sp.name);
						} else if (t.OutProperties.length === 0) {
							t.OutProperties = Array(parametersCount);
							t.OutProperties[parameterIndex] = sp.name;
						} else {
							t.OutProperties[parameterIndex] = sp.name;
						}
					}
				}
			}
			trs.push(t);
		}
		setTransformations(trs);
		setSelectedProperty(null);
	};

	const onRemoveConnection = (transformationPosition, propertyName, propertyType, e) => {
		if (e.target.previousSibling == null || e.target.previousSibling.tagName !== 'svg') return; // the click is not on the label of the arrow.
		let trs = [];
		for (let t of transformations) {
			if (t.Position === transformationPosition) {
				if (getTransformationType(t) === 'one-to-one') continue;
				let properties = propertyType === 'input' ? t.InProperties : t.OutProperties;
				let filtered = [];
				if (t.PredefinedFunc !== null) {
					// replace the removed property with 'undefined' to preserve order.
					for (let p of properties) {
						if (p != null && p === propertyName) {
							filtered.push(undefined);
						} else {
							filtered.push(p);
						}
					}
				} else {
					for (let p of properties) {
						if (p !== propertyName) {
							filtered.push(p);
						}
					}
				}
				if (propertyType === 'input') {
					t.InProperties = filtered;
				} else {
					t.OutProperties = filtered;
				}
			}
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onSave = async () => {
		let trs = [];
		for (let t of transformations) {
			let toSave = { ...t };
			delete toSave.Position;
			if (t.PredefinedFunc !== null) {
				// validate the predefined function connections.
				for (let [i, p] of t.PredefinedFunc.In.properties.entries()) {
					if (t.InProperties[i] == null) {
						onError(
							`The input parameter "${p.label}" of the predefined transformation "${t.PredefinedFunc.Name}" is not linked to any input property`
						);
						return;
					}
				}
				for (let [i, p] of t.PredefinedFunc.Out.properties.entries()) {
					if (t.OutProperties[i] == null) {
						onError(
							`The output parameter "${p.label}" of the predefined transformation "${t.PredefinedFunc.Name}" is not linked to any output property`
						);
						return;
					}
				}
				toSave.PredefinedFunc = t.PredefinedFunc.ID;
			}
			// TODO: VALIDATE THE CUSTOMFUNC TOO...
			trs.push(toSave);
		}
		let [, err] = await call(`/api/connections/${c.ID}/mappings`, 'PUT', trs);
		if (err) {
			onError(err);
			return;
		}
		onStatuChange({
			variant: 'success',
			icon: 'check2-circle',
			text: 'Your transformations have been successfully saved',
		});
	};

	const isSelectedProperty = (name, type) => {
		let sp = selectedProperty;
		return sp && sp.name === name && sp.type === type;
	};

	let sp = selectedProperty;
	let st = selectedTransformation;
	return (
		<div className={`ConnectionProperties${sp ? ' selectedProperty' : ''}`}>
			{sp && (
				<div className='selectedPropertyMessage'>
					<div>
						Add a mapping
						{sp.type === 'input' ? ' from ' : ' to '}
						<span className='name'>"{sp.label === '' ? sp.name : sp.label}"</span>
					</div>
					<SlButton
						className='removeSelectedProperty'
						variant='neutral'
						onClick={() => {
							setSelectedProperty(null);
						}}
					>
						<SlIcon slot='prefix' name='x-lg' />
						Close
					</SlButton>
				</div>
			)}
			<div className='main'>
				<div className='properties usedInputProperties'>
					<div className='title'>{c.Role === 'Source' ? `${c.Name} properties` : 'Golden record'}</div>
					<SlButton
						className='addUsedProperty'
						variant='neutral'
						disabled={sp != null}
						onClick={() => setIsInputDialogOpen(true)}
					>
						Add property
					</SlButton>
					{usedInputProperties.map(({ name, label }) => {
						let type = 'input';
						let isSelected = isSelectedProperty(name, type);
						return (
							<ConnectionProperty
								name={name}
								label={label}
								type={type}
								isSelected={isSelected}
								onHandle={() => setSelectedProperty({ name: name, label: label, type: type })}
								onRemove={(e) => onRemoveUsedProperty(e, name, type)}
								disableRemove={sp != null}
								onConnect={sp && !isSelected ? () => onAddOneToOneTransformation(name, type) : null}
							/>
						);
					})}
				</div>
				<div className='transformations'>
					<SlButton
						className='saveButton'
						variant='primary'
						size='large'
						disabled={sp != null}
						onClick={onSave}
					>
						<SlIcon slot='prefix' name='save' />
						Save
					</SlButton>
					{transformations.map((t) => {
						return (
							<div key={t.Position} className='transformation' id={`transformation-${t.Position}`}>
								<TransformationNode
									transformation={t}
									onSelect={sp ? null : () => setSelectedTransformation(t)}
									onRemove={() => onRemoveTransformation(t.Position)}
									onCustomTransformationConnect={
										sp ? () => onCustomTransformationConnect(t.Position) : null
									}
									onPredefinedTransformationConnect={
										sp
											? (handleID) => onPredefinedTransformationConnect(t.Position, handleID)
											: null
									}
								/>
								{st && st.Position === t.Position && (
									<TransformationDialog
										transformation={t}
										onClose={() => setSelectedTransformation(null)}
										onEditorChange={(value) => onChangeTransformation(t.Position, value)}
										onRemove={() => onRemoveTransformation(t.Position)}
									/>
								)}
							</div>
						);
					})}
					<div className='addTransformationButtons'>
						<SlTooltip content='Write a custom transformation' disabled={sp != null}>
							<SlButton
								className='addTransformation'
								variant='default'
								disabled={sp != null}
								onClick={onAddCustomTransformation}
							>
								<SlIcon name='plus-lg'></SlIcon>
							</SlButton>
						</SlTooltip>
						<SlTooltip content='Choose a predefined transformation' disabled={sp != null}>
							<SlButton
								className='addTransformation'
								variant='default'
								disabled={sp != null}
								onClick={() => setShowPredefinedTransformations(true)}
							>
								<SlIcon name='list'></SlIcon>
							</SlButton>
						</SlTooltip>
					</div>
					{showPredefinedTransformations && (
						<SlDialog
							label='Select a predefined transformation'
							className='predefinedTransformationsDialog'
							open={true}
							onSlAfterHide={() => setShowPredefinedTransformations(false)}
							style={{ '--width': '700px' }}
						>
							<div className='predefinedTransformations'>
								{predefinedTransformations.map((t) => {
									return (
										<div
											className={`predefinedTransformation${
												t.ID === selectedPredefinedTransformation ? ' selected' : ''
											}`}
											onClick={() => setSelectedPredefinedTransformation(t.ID)}
										>
											<SlIcon name={t.Icon}></SlIcon>
											<div className='name'>{t.Name}</div>
											<div className='description'>{t.Description}</div>
										</div>
									);
								})}
							</div>
							<SlButton
								disabled={selectedPredefinedTransformation === 0}
								slot='footer'
								variant='primary'
								onClick={onAddPredefinedTransformation}
							>
								Add
							</SlButton>
						</SlDialog>
					)}
				</div>
				<div className='properties usedOutputProperties'>
					<div className='title'>{c.Role === 'Source' ? `Golden record` : `${c.Name} properties`}</div>
					<SlButton
						className='addUsedProperty'
						variant='neutral'
						disabled={sp != null}
						onClick={() => setIsOutputDialogOpen(true)}
					>
						Add property
					</SlButton>
					{usedOutputProperties.map(({ name, label }) => {
						let type = 'output';
						let isSelected = isSelectedProperty(name, type);
						return (
							<ConnectionProperty
								name={name}
								label={label}
								type={type}
								isSelected={isSelected}
								onHandle={() => setSelectedProperty({ name: name, label: label, type: type })}
								onRemove={(e) => onRemoveUsedProperty(e, name, type)}
								disableRemove={sp != null}
								onConnect={sp && !isSelected ? () => onAddOneToOneTransformation(name, type) : null}
							/>
						);
					})}
				</div>
			</div>
			<div className='arrows'>
				{isSelected &&
					transformations.map((t) => {
						let inputArrows = [];
						for (let [i, p] of t.InProperties.entries()) {
							if (p != null) {
								inputArrows.push(
									<div
										className={`arrow${isSelectedProperty(p, 'input') ? ' selected' : ''}`}
										onClick={
											isSelectedProperty(p, 'input')
												? (e) => {
														onRemoveConnection(t.Position, p, 'input', e);
												  }
												: null
										}
									>
										<Xarrow
											start={p}
											end={
												t.PredefinedFunc !== null
													? `transformation-${
															t.Position
													  }-input-${t.PredefinedFunc.In.properties[i].label.replace(
															/\s/g,
															''
													  )}`
													: `transformation-${t.Position}`
											}
											startAnchor='right'
											endAnchor='left'
											showHead={false}
											color='#cacad6'
											strokeWidth={1}
											labels={isSelectedProperty(p, 'input') && '-'}
										/>
									</div>
								);
							}
						}
						let outputArrows = [];
						for (let [i, p] of t.OutProperties.entries()) {
							if (p != null) {
								outputArrows.push(
									<div
										className={`arrow${isSelectedProperty(p, 'output') ? ' selected' : ''}`}
										onClick={
											isSelectedProperty(p, 'output')
												? (e) => {
														onRemoveConnection(t.Position, p, 'output', e);
												  }
												: null
										}
									>
										<Xarrow
											start={
												t.PredefinedFunc !== null &&
												t.PredefinedFunc.Out.properties.length === 1
													? `transformation-${
															t.Position
													  }-output-${t.PredefinedFunc.Out.properties[0].label.replace(
															/\s/g,
															''
													  )}`
													: t.PredefinedFunc !== null
													? `transformation-${
															t.Position
													  }-output-${t.PredefinedFunc.Out.properties[i].label.replace(
															/\s/g,
															''
													  )}`
													: `transformation-${t.Position}`
											}
											end={p}
											startAnchor='right'
											endAnchor='left'
											showHead={false}
											color='#cacad6'
											strokeWidth={1}
											labels={isSelectedProperty(p, 'output') && '-'}
										/>
									</div>
								);
							}
						}
						return [...inputArrows, ...outputArrows];
					})}
			</div>
			<SlDialog
				label='Add a property'
				open={isInputDialogOpen}
				onSlAfterHide={() => setIsInputDialogOpen(false)}
				style={{ '--width': '700px' }}
			>
				<SlInput
					type='search'
					clearable
					placeholder='search'
					value={inputSearchTerm}
					onSlInput={(e) => setInputSearchTerm(e.currentTarget.value)}
				>
					<SlIcon name='search' slot='prefix'></SlIcon>
				</SlInput>
				<div className='dialogProperties'>
					{inputProperties.map((p) => {
						let toString = p.label ? p.label : p.name;
						if (
							toString.includes(inputSearchTerm) ||
							toString.includes(inputSearchTerm.charAt(0).toUpperCase() + inputSearchTerm.slice(1)) ||
							toString.includes(inputSearchTerm.toUpperCase) ||
							toString.includes(inputSearchTerm.toLowerCase)
						) {
							return (
								<div
									key={p.name}
									className={`property${
										usedInputProperties.find((up) => up.name === p.name) != null ? ' used' : ''
									}`}
								>
									<div>{toString}</div>
									<SlIconButton
										name='plus-circle'
										label='Add property'
										onClick={(e) => onAddUsedProperty(p, 'input')}
									/>
								</div>
							);
						}
						return '';
					})}
				</div>
			</SlDialog>
			<SlDialog
				label='Add a property'
				open={isOutputDialogOpen}
				onSlAfterHide={() => setIsOutputDialogOpen(false)}
				style={{ '--width': '700px' }}
			>
				<SlInput
					type='search'
					clearable
					placeholder='search'
					value={outputSearchTerm}
					onSlInput={(e) => setOutputSearchTerm(e.currentTarget.value)}
				>
					<SlIcon name='search' slot='prefix'></SlIcon>
				</SlInput>
				<div className='dialogProperties'>
					{outputProperties.map((p) => {
						let toString = p.label ? p.label : p.name;
						if (
							toString.includes(outputSearchTerm) ||
							toString.includes(outputSearchTerm.charAt(0).toUpperCase() + outputSearchTerm.slice(1)) ||
							toString.includes(outputSearchTerm.toUpperCase) ||
							toString.includes(outputSearchTerm.toLowerCase)
						) {
							return (
								<div
									key={p.name}
									className={`property${
										usedOutputProperties.find((up) => up.name === p.name) != null ? ' used' : ''
									}`}
								>
									<div>{toString}</div>
									<SlIconButton
										name='plus-circle'
										label='Add property'
										onClick={(e) => onAddUsedProperty(p, 'output')}
									/>
								</div>
							);
						}
						return '';
					})}
				</div>
			</SlDialog>
		</div>
	);
};

export default ConnectionProperties;

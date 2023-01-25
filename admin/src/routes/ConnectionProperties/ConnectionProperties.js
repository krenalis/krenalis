import { useState, useEffect } from 'react';
import './ConnectionProperties.css';
import call from '../../utils/call';
import ConnectionProperty from '../../components/ConnectionProperty/ConnectionProperty';
import TransformationNode from '../../components/TrasformationNode/TransformationNode';
import TransformationDialog from '../../components/TransformationDialog/TransformationDialog';
import SelectedPropertyMessage from '../../components/SelectedPropertyMessage/SelectedPropertyMessage';
import PropertiesDialog from '../../components/PropertiesDialog/PropertiesDialog';
import { Transformation } from '../../utils/transformations';
import { SlButton, SlIcon, SlDialog, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';
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

	const hooksByRole = {
		input: {
			properties: inputProperties,
			setProperties: setInputProperties,
			usedProperties: usedInputProperties,
			setUsedProperties: setUsedInputProperties,
			searchTerm: inputSearchTerm,
			setSearchTerm: setInputSearchTerm,
			isDialogOpen: isInputDialogOpen,
			setIsDialogOpen: setIsInputDialogOpen,
		},
		output: {
			properties: outputProperties,
			setProperties: setOutputProperties,
			usedProperties: usedOutputProperties,
			setUsedProperties: setUsedOutputProperties,
			searchTerm: outputSearchTerm,
			setSearchTerm: setOutputSearchTerm,
			isDialogOpen: isOutputDialogOpen,
			setIsDialogOpen: setIsOutputDialogOpen,
		},
	};

	useEffect(() => {
		const fetchState = async () => {
			let err;

			// get the connection properties and the warehouse user properties.
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

			// turn the transformations into "Transformation" objects.
			let transformationObjects = [];
			for (let t of transformations) {
				transformationObjects.push(new Transformation(t));
			}

			setTransformations(transformationObjects);
			setLastTransformationPosition(pos);
		};
		fetchState();
	}, []);

	const incrementTransformationPosition = () => {
		setLastTransformationPosition(lastTransformationPosition + 1);
	};

	const onAddUsedProperty = (role, p) => {
		let { usedProperties, setUsedProperties } = hooksByRole[role];
		setUsedProperties([...usedProperties, p]);
	};

	const onRemoveUsedProperty = (e, role, name) => {
		e.stopPropagation();
		let { usedProperties, setUsedProperties } = hooksByRole[role];
		setUsedProperties(usedProperties.filter((p) => p.name !== name));
		// remove the property from the transformations that use it.
		let trs = [];
		for (let t of transformations) {
			if (t.containsProperty(role, name)) {
				if (t.Type === 'one-to-one') continue; // remove the transformation.
				t.removeProperty(role, name);
			}
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onAddCustomTransformation = () => {
		setTransformations([...transformations, Transformation.createCustomTransformation(lastTransformationPosition)]);
		incrementTransformationPosition();
	};

	const onAddPredefinedTransformation = () => {
		let predefined = predefinedTransformations.find((t) => t.ID === selectedPredefinedTransformation);
		setTransformations([
			...transformations,
			Transformation.createPredefinedTransformation(predefined, lastTransformationPosition),
		]);
		setShowPredefinedTransformations(false);
		incrementTransformationPosition();
	};

	const onOneToOneConnect = (name, role) => {
		let sp = selectedProperty;
		if (role === sp.role) {
			onError(`cannot connect two ${role} properties`);
			return;
		}
		let input = sp.role === 'input' ? sp.name : name;
		let output = sp.role === 'output' ? sp.name : name;
		setTransformations([
			...transformations,
			Transformation.createOneToOneTransformation(input, output, lastTransformationPosition),
		]);
		setSelectedProperty(null);
		incrementTransformationPosition();
	};

	const onTransformationConnect = (position, parameter) => {
		let sp = selectedProperty;
		let trs = [];
		for (let t of transformations) {
			if (t.Position === position && !t.containsProperty(sp.role, sp.name)) {
				t.addProperty(sp.role, sp.name, parameter);
			}
			trs.push(t);
		}
		setTransformations(trs);
		setSelectedProperty(null);
	};

	const onRemoveConnection = (e, role, position, name) => {
		if (e.target.previousSibling == null || e.target.previousSibling.tagName !== 'svg') return; // the click is not on the label of the arrow.
		let trs = [];
		for (let t of transformations) {
			if (t.Position === position) {
				if (t.Type === 'one-to-one') continue; // remove the transformation.
				t.removeProperty(role, name);
			}
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onChangeTransformation = (position, value) => {
		let trs = [];
		for (let t of transformations) {
			if (t.Position === position) t.updateSource(value);
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onRemoveTransformation = (position) => {
		let trs = [];
		for (let t of transformations) {
			if (t.Position !== position) trs.push(t);
		}
		setTransformations(trs);
		setSelectedTransformation(null);
	};

	const onSave = async () => {
		let trs = [];
		for (let t of transformations) {
			let err = t.validateProperties();
			if (err != null) {
				onError(err);
				return;
			}
			let toSave = t.toServerFormat();
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

	const isSelectedProperty = (name, role) => {
		let sp = selectedProperty;
		return sp && sp.name === name && sp.role === role;
	};

	let sp = selectedProperty;
	let st = selectedTransformation;
	return (
		<div className={`ConnectionProperties${sp ? ' selectedProperty' : ''}`}>
			{sp && (
				<SelectedPropertyMessage
					selectedProperty={sp}
					onClose={() => {
						setSelectedProperty(null);
					}}
				/>
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
					{usedInputProperties.map(({ name, label, type }) => {
						let role = 'input';
						let isSelected = isSelectedProperty(name, role);
						return (
							<ConnectionProperty
								name={name}
								label={label}
								role={role}
								isSelected={isSelected}
								onHandle={() =>
									setSelectedProperty({ name: name, label: label, type: type, role: role })
								}
								onRemove={(e) => onRemoveUsedProperty(e, role, name)}
								disableRemove={sp != null}
								onConnect={sp && !isSelected ? () => onOneToOneConnect(name, role) : null}
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
									onConnect={sp ? (handleID) => onTransformationConnect(t.Position, handleID) : null}
									onRemove={() => onRemoveTransformation(t.Position)}
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
					{usedOutputProperties.map(({ name, label, type }) => {
						let role = 'output';
						let isSelected = isSelectedProperty(name, role);
						return (
							<ConnectionProperty
								name={name}
								label={label}
								role={role}
								isSelected={isSelected}
								onHandle={() =>
									setSelectedProperty({ name: name, label: label, type: type, role: role })
								}
								onRemove={(e) => onRemoveUsedProperty(e, role, name)}
								disableRemove={sp != null}
								onConnect={sp && !isSelected ? () => onOneToOneConnect(name, role) : null}
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
							if (p !== undefined) {
								inputArrows.push(
									<div
										className={`arrow${isSelectedProperty(p, 'input') ? ' selected' : ''}`}
										onClick={
											isSelectedProperty(p, 'input')
												? (e) => {
														onRemoveConnection(e, 'input', t.Position, p);
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
							if (p !== undefined) {
								outputArrows.push(
									<div
										className={`arrow${isSelectedProperty(p, 'output') ? ' selected' : ''}`}
										onClick={
											isSelectedProperty(p, 'output')
												? (e) => {
														onRemoveConnection(e, 'output', t.Position, p);
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
			{Object.keys(hooksByRole).map((role) => {
				let { isDialogOpen, setIsDialogOpen, searchTerm, setSearchTerm, properties, usedProperties } =
					hooksByRole[role];
				return (
					<PropertiesDialog
						isOpen={isDialogOpen}
						onClose={() => setIsDialogOpen(false)}
						searchTerm={searchTerm}
						onSearch={(e) => setSearchTerm(e.currentTarget.value)}
						properties={properties}
						usedProperties={usedProperties}
						onAddProperty={(p) => onAddUsedProperty(role, p)}
					/>
				);
			})}
		</div>
	);
};

export default ConnectionProperties;

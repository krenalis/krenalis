import { useState, useEffect, useContext } from 'react';
import './ConnectionMappings.css';
import ConnectionProperty from '../../components/ConnectionProperty/ConnectionProperty';
import MappingNode from '../../components/MappingNode/MappingNode';
import SelectedPropertyMessage from '../../components/SelectedPropertyMessage/SelectedPropertyMessage';
import PropertiesDialog from '../../components/PropertiesDialog/PropertiesDialog';
import { Mapping } from '../../utils/mappings';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { useNavigate } from 'react-router';
import { SlButton, SlIcon, SlDialog, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';
import Xarrow from 'react-xarrows';
import { NotFoundError, UnprocessableError } from '../../api/errors';

const ConnectionMappings = ({ connection: c, onConnectionChange, isSelected }) => {
	let [inputProperties, setInputProperties] = useState([]);
	let [outputProperties, setOutputProperties] = useState([]);
	let [usedInputProperties, setUsedInputProperties] = useState([]);
	let [usedOutputProperties, setUsedOutputProperties] = useState([]);
	let [mappings, setMappings] = useState([]);
	let [lastMappingPosition, setLastMappingPosition] = useState(1);
	let [inputSearchTerm, setInputSearchTerm] = useState('');
	let [outputSearchTerm, setOutputSearchTerm] = useState('');
	let [isInputDialogOpen, setIsInputDialogOpen] = useState(false);
	let [isOutputDialogOpen, setIsOutputDialogOpen] = useState(false);
	let [selectedProperty, setSelectedProperty] = useState(null);
	let [selectedPredefinedMapping, setSelectedPredefinedMapping] = useState(0);
	let [predefinedMappings, setPredefinedMappings] = useState([]);
	let [showPredefinedMappings, setShowPredefinedMappings] = useState(false);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

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

	const navigate = useNavigate();

	useEffect(() => {
		const fetchState = async () => {
			let err;

			// get the connection properties and the warehouse user properties.
			let connectionSchema;
			[connectionSchema, err] = await API.connections.schema(c.ID);
			if (err) {
				showError(err);
				return;
			}
			if (connectionSchema == null) return;
			let connectionProperties = [];
			for (let p of connectionSchema.properties) {
				connectionProperties.push(p);
			}
			let userSchema;
			[userSchema, err] = await API.workspace.userSchema();
			if (err) {
				showError(err);
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

			// get the predefined mappings.
			let predefinedMappings;
			[predefinedMappings, err] = await API.predefinedMappings();
			if (err) {
				showError(err);
				return;
			}
			setPredefinedMappings(predefinedMappings);

			// get the mappings.
			let mappings;
			[mappings, err] = await API.connections.mappings(c.ID);
			if (err) {
				showError(err);
				return;
			}
			if (mappings == null) return;

			// replace the predefined mappings IDs with the full predefined
			// mappings.
			for (let m of mappings) {
				if (m.PredefinedFunc != null) {
					let predefinedMapping = predefinedMappings.find((pt) => pt.ID === m.PredefinedFunc);
					m.PredefinedFunc = predefinedMapping;
				}
			}

			// get the input properties and the output properties used by the
			// mappings.
			let usedInputProperties = [];
			let usedOutputProperties = [];
			for (let m of mappings) {
				for (let input of m.InProperties) {
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
				for (let output of m.OutProperties) {
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

			// compute the positions of the mappings.
			let pos = lastMappingPosition;
			for (let m of mappings) {
				m.Position = pos;
				pos += 1;
			}

			// turn the mappings into "Mapping" objects.
			let mappingObjects = [];
			for (let m of mappings) {
				mappingObjects.push(new Mapping(m));
			}

			setMappings(mappingObjects);
			setLastMappingPosition(pos);
		};
		fetchState();
	}, []);

	const incrementMappingPosition = () => {
		setLastMappingPosition(lastMappingPosition + 1);
	};

	const onAddUsedProperty = (role, p) => {
		let { usedProperties, setUsedProperties } = hooksByRole[role];
		setUsedProperties([...usedProperties, p]);
	};

	const onRemoveUsedProperty = (e, role, name) => {
		e.stopPropagation();
		let { usedProperties, setUsedProperties } = hooksByRole[role];
		setUsedProperties(usedProperties.filter((p) => p.name !== name));
		// remove the property from the mappings that use it.
		let mps = [];
		for (let m of mappings) {
			if (m.containsProperty(role, name)) {
				if (m.Type === 'one-to-one') continue; // remove the mapping.
				m.removeProperty(role, name);
			}
			mps.push(m);
		}
		setMappings(mps);
	};

	const onAddPredefinedMapping = () => {
		let predefined = predefinedMappings.find((t) => t.ID === selectedPredefinedMapping);
		setMappings([...mappings, Mapping.createPredefinedMapping(predefined, lastMappingPosition)]);
		setShowPredefinedMappings(false);
		incrementMappingPosition();
	};

	const onOneToOneConnect = (name, role) => {
		let sp = selectedProperty;
		if (role === sp.role) {
			showError(`cannot connect two ${role} properties`);
			return;
		}
		let input = sp.role === 'input' ? sp.name : name;
		let output = sp.role === 'output' ? sp.name : name;
		setMappings([...mappings, Mapping.createOneToOneMapping(input, output, lastMappingPosition)]);
		setSelectedProperty(null);
		incrementMappingPosition();
	};

	const onMappingConnect = (position, parameter) => {
		let sp = selectedProperty;
		let mps = [];
		for (let m of mappings) {
			if (m.Position === position && !m.containsProperty(sp.role, sp.name)) {
				m.addProperty(sp.role, sp.name, parameter);
			}
			mps.push(m);
		}
		setMappings(mps);
		setSelectedProperty(null);
	};

	const onRemoveConnection = (e, role, position, name) => {
		if (e.target.previousSibling == null || e.target.previousSibling.tagName !== 'svg') return; // the click is not on the label of the arrow.
		let mps = [];
		for (let m of mappings) {
			if (m.Position === position) {
				if (m.Type === 'one-to-one') continue; // remove the mapping.
				m.removeProperty(role, name);
			}
			mps.push(m);
		}
		setMappings(mps);
	};

	const onRemoveMapping = (position) => {
		let mps = [];
		for (let m of mappings) {
			if (m.Position !== position) mps.push(m);
		}
		setMappings(mps);
	};

	const onSave = async () => {
		let mps = [];
		for (let m of mappings) {
			let err = m.validateProperties();
			if (err != null) {
				showError(err);
				return;
			}
			let toSave = m.toServerFormat();
			mps.push(toSave);
		}
		let [, err] = await API.connections.setMappings(c.ID, mps);
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'AlreadyHasTransformation') {
					showStatus(statuses.alreadyHasTransformation);
				}
				return;
			}
			showError(err);
			return;
		}
		showStatus(statuses.mappingsSaved);
		c.Mappings = mps;
		onConnectionChange(c);
	};

	const onAlertDialogCloseRequest = (e) => {
		e.preventDefault();
	};

	const isSelectedProperty = (name, role) => {
		let sp = selectedProperty;
		return sp && sp.name === name && sp.role === role;
	};

	let sp = selectedProperty;
	return (
		<div className={`ConnectionMappings${sp ? ' selectedProperty' : ''}`}>
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
								type={type}
								isSelected={isSelected}
								onHandle={() =>
									setSelectedProperty({ name: name, label: label, type: type, role: role })
								}
								onRemove={(e) => onRemoveUsedProperty(e, role, name)}
								disableRemove={sp != null}
								onConnect={sp && !isSelected ? () => onOneToOneConnect(name, role) : null}
								connectable
							/>
						);
					})}
				</div>
				<div className='mappings'>
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
					{mappings.map((m) => {
						return (
							<div key={m.Position} className='mapping' id={`mapping-${m.Position}`}>
								<MappingNode
									mapping={m}
									onConnect={sp ? (handleID) => onMappingConnect(m.Position, handleID) : null}
									onRemove={() => onRemoveMapping(m.Position)}
								/>
							</div>
						);
					})}
					<div className='addMappingButtons'>
						<SlTooltip content='Choose a predefined mapping' disabled={sp != null}>
							<SlButton
								className='addMapping'
								variant='default'
								disabled={sp != null}
								onClick={() => setShowPredefinedMappings(true)}
							>
								<SlIcon name='list'></SlIcon>
							</SlButton>
						</SlTooltip>
					</div>
					{showPredefinedMappings && (
						<SlDialog
							label='Select a predefined mapping'
							className='predefinedMappingsDialog'
							open={true}
							onSlAfterHide={() => setShowPredefinedMappings(false)}
							style={{ '--width': '700px' }}
						>
							<div className='predefinedMappings'>
								{predefinedMappings.map((m) => {
									return (
										<div
											className={`predefinedMapping${
												m.ID === selectedPredefinedMapping ? ' selected' : ''
											}`}
											onClick={() => setSelectedPredefinedMapping(m.ID)}
										>
											<SlIcon name={m.Icon}></SlIcon>
											<div className='name'>{m.Name}</div>
											<div className='description'>{m.Description}</div>
										</div>
									);
								})}
							</div>
							<SlButton
								disabled={selectedPredefinedMapping === 0}
								slot='footer'
								variant='primary'
								onClick={onAddPredefinedMapping}
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
								type={type}
								isSelected={isSelected}
								onHandle={() =>
									setSelectedProperty({ name: name, label: label, type: type, role: role })
								}
								onRemove={(e) => onRemoveUsedProperty(e, role, name)}
								disableRemove={sp != null}
								onConnect={sp && !isSelected ? () => onOneToOneConnect(name, role) : null}
								connectable
							/>
						);
					})}
				</div>
			</div>
			<div className='arrows'>
				{isSelected &&
					mappings.map((m) => {
						let inputArrows = [];
						for (let [i, p] of m.InProperties.entries()) {
							if (p !== undefined) {
								inputArrows.push(
									<div
										className={`arrow${isSelectedProperty(p, 'input') ? ' selected' : ''}`}
										onClick={
											isSelectedProperty(p, 'input')
												? (e) => {
														onRemoveConnection(e, 'input', m.Position, p);
												  }
												: null
										}
									>
										<Xarrow
											start={p}
											end={
												m.PredefinedFunc !== null
													? `mapping-${m.Position}-input-${m.PredefinedFunc.In.properties[
															i
													  ].label.replace(/\s/g, '')}`
													: `mapping-${m.Position}`
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
						for (let [i, p] of m.OutProperties.entries()) {
							if (p !== undefined) {
								outputArrows.push(
									<div
										className={`arrow${isSelectedProperty(p, 'output') ? ' selected' : ''}`}
										onClick={
											isSelectedProperty(p, 'output')
												? (e) => {
														onRemoveConnection(e, 'output', m.Position, p);
												  }
												: null
										}
									>
										<Xarrow
											start={
												m.PredefinedFunc !== null &&
												m.PredefinedFunc.Out.properties.length === 1
													? `mapping-${
															m.Position
													  }-output-${m.PredefinedFunc.Out.properties[0].label.replace(
															/\s/g,
															''
													  )}`
													: m.PredefinedFunc !== null
													? `mapping-${m.Position}-output-${m.PredefinedFunc.Out.properties[
															i
													  ].label.replace(/\s/g, '')}`
													: `mapping-${m.Position}`
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
			<SlDialog
				label='Transformation already configured'
				className='AlertDialog'
				open={c.Transformation != null}
				onSlRequestClose={onAlertDialogCloseRequest}
				style={{ '--width': '700px' }}
			>
				<div className='message'>
					This connection already has a transformation configured. To set the mappings make sure you delete it
					first.
				</div>
				<SlButton variant='primary' className='backToOverview' onClick={() => navigate(0)}>
					Return to overview
				</SlButton>
			</SlDialog>
		</div>
	);
};

export default ConnectionMappings;

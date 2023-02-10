import { useState, useEffect, useContext } from 'react';
import './ConnectionTransformation.css';
import call from '../../utils/call';
import ConnectionProperty from '../../components/ConnectionProperty/ConnectionProperty';
import PropertiesDialog from '../../components/PropertiesDialog/PropertiesDialog';
import { Transformation } from '../../utils/transformations';
import statuses from '../../constants/statuses';
import { AppContext } from '../../context/AppContext';
import { useNavigate } from 'react-router';
import Editor from '@monaco-editor/react';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';
import { NotFoundError, UnprocessableError } from '../../api/errors';

const ConnectionTransformation = ({ connection: c, onConnectionChange, isSelected }) => {
	let [inputProperties, setInputProperties] = useState([]);
	let [outputProperties, setOutputProperties] = useState([]);
	let [usedInputProperties, setUsedInputProperties] = useState([]);
	let [usedOutputProperties, setUsedOutputProperties] = useState([]);
	let [inputSearchTerm, setInputSearchTerm] = useState('');
	let [outputSearchTerm, setOutputSearchTerm] = useState('');
	let [isInputDialogOpen, setIsInputDialogOpen] = useState(false);
	let [isOutputDialogOpen, setIsOutputDialogOpen] = useState(false);
	let [transformation, setTransformation] = useState(null);

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

			// get the transformation.
			let transformation;
			[transformation, err] = await API.connections.transformation(c.ID);
			if (err) {
				showError(err);
				return;
			}
			let transformationObject = new Transformation(transformation);
			setTransformation(transformationObject);

			// get the input properties and the output properties used by the
			// transformation.
			let usedInputProperties = [];
			let usedOutputProperties = [];
			if (transformation != null) {
				usedInputProperties = transformation.In.properties;
				usedOutputProperties = transformation.Out.properties;
			}
			setUsedInputProperties(usedInputProperties);
			setUsedOutputProperties(usedOutputProperties);
		};
		fetchState();
	}, []);

	const onRemoveUsedProperty = (e, role, name) => {
		e.stopPropagation();
		let { usedProperties, setUsedProperties } = hooksByRole[role];
		setUsedProperties(usedProperties.filter((p) => p.name !== name));
		let t = new Transformation({ ...transformation });
		t.removeProperty(role, name);
		setTransformation(t);
	};

	const onAddUsedProperty = (role, p) => {
		let { usedProperties, setUsedProperties } = hooksByRole[role];
		setUsedProperties([...usedProperties, p]);
		let t = new Transformation({ ...transformation });
		t.addProperty(role, p);
		setTransformation(t);
	};

	const onTransformationChange = (value) => {
		let t = new Transformation({ ...transformation });
		t.PythonSource = value;
		setTransformation(t);
	};

	const onSave = async () => {
		let t = new Transformation({ ...transformation });
		let toSave = {
			In: t.In,
			Out: t.Out,
			PythonSource: t.PythonSource,
		};
		let [, err] = await API.connections.setTransformation(c.ID, toSave);
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'AlreadyHasMappings') {
					showStatus(statuses.alreadyHasMappings);
				}
				return;
			}
			showError(err);
			return;
		}
		showStatus(statuses.transformationSaved);
		c.Transformation = toSave;
		onConnectionChange(c);
	};

	const onClear = async () => {
		let [, err] = await API.connections.setTransformation(c.ID, null);
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		let t = new Transformation();
		setTransformation(t);
		setUsedInputProperties([]);
		setUsedOutputProperties([]);
		showStatus(statuses.transformationCleanedUp);
		c.Transformation = null;
		onConnectionChange(c);
	};

	const onAlertDialogCloseRequest = (e) => {
		e.preventDefault();
	};

	if (transformation == null) {
		return;
	}

	return (
		<div className={'ConnectionTransformation'}>
			<div className='main'>
				<div className='properties usedInputProperties'>
					<div className='title'>{c.Role === 'Source' ? `${c.Name} properties` : 'Golden record'}</div>
					<SlButton className='addUsedProperty' variant='neutral' onClick={() => setIsInputDialogOpen(true)}>
						Add property
					</SlButton>
					{usedInputProperties.map(({ name, label, type }) => {
						let role = 'input';
						return (
							<ConnectionProperty
								name={name}
								label={label}
								role={role}
								type={type}
								onRemove={(e) => onRemoveUsedProperty(e, role, name)}
							/>
						);
					})}
				</div>
				<div className='transformation'>
					<div className='buttons'>
						<SlButton
							className='saveButton'
							variant='primary'
							disabled={usedInputProperties.length === 0 && usedOutputProperties.length === 0}
							onClick={onSave}
						>
							<SlIcon slot='prefix' name='save' />
							Save
						</SlButton>
						<SlButton className='clearButton' variant='danger' onClick={onClear}>
							<SlIcon slot='prefix' name='x-lg' />
							Clear
						</SlButton>
					</div>
					<div className='editorWrapper'>
						<Editor
							onChange={(value) => onTransformationChange(value)}
							defaultLanguage='python'
							value={transformation.PythonSource}
							theme='vs-primary'
						/>
					</div>
				</div>
				<div className='properties usedOutputProperties'>
					<div className='title'>{c.Role === 'Source' ? `Golden record` : `${c.Name} properties`}</div>
					<SlButton className='addUsedProperty' variant='neutral' onClick={() => setIsOutputDialogOpen(true)}>
						Add property
					</SlButton>
					{usedOutputProperties.map(({ name, label, type }) => {
						let role = 'output';
						return (
							<ConnectionProperty
								name={name}
								label={label}
								role={role}
								type={type}
								onRemove={(e) => onRemoveUsedProperty(e, role, name)}
							/>
						);
					})}
				</div>
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
				label='Mappings already configured'
				className='AlertDialog'
				open={c.Mappings.length > 0}
				onSlRequestClose={onAlertDialogCloseRequest}
				style={{ '--width': '700px' }}
			>
				<div className='message'>
					This connection already has mappings configured. To set the transformation make sure you delete them
					first.
				</div>
				<SlButton variant='primary' className='backToOverview' onClick={() => navigate(0)}>
					Return to overview
				</SlButton>
			</SlDialog>
		</div>
	);
};

export default ConnectionTransformation;

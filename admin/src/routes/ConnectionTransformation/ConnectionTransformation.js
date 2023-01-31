import { useState, useEffect } from 'react';
import './ConnectionTransformation.css';
import call from '../../utils/call';
import ConnectionProperty from '../../components/ConnectionProperty/ConnectionProperty';
import PropertiesDialog from '../../components/PropertiesDialog/PropertiesDialog';
import { Transformation } from '../../utils/transformations';
import Editor from '@monaco-editor/react';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionTransformation = ({ connection: c, onError, onStatuChange, isSelected }) => {
	let [inputProperties, setInputProperties] = useState([]);
	let [outputProperties, setOutputProperties] = useState([]);
	let [usedInputProperties, setUsedInputProperties] = useState([]);
	let [usedOutputProperties, setUsedOutputProperties] = useState([]);
	let [inputSearchTerm, setInputSearchTerm] = useState('');
	let [outputSearchTerm, setOutputSearchTerm] = useState('');
	let [isInputDialogOpen, setIsInputDialogOpen] = useState(false);
	let [isOutputDialogOpen, setIsOutputDialogOpen] = useState(false);
	let [transformation, setTransformation] = useState(null);

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

			// get the transformation.
			let transformation;
			[transformation, err] = await call(`/api/connections/${c.ID}/transformation`);
			if (err) {
				onError(err);
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
		let [, err] = await call(`/api/connections/${c.ID}/transformation`, 'PUT', toSave);
		if (err) {
			onError(err);
			return;
		}
		onStatuChange({
			variant: 'success',
			icon: 'check2-circle',
			text: 'Your transformation has been successfully saved',
		});
	};

	const onReset = async () => {
		let [, err] = await call(`/api/connections/${c.ID}/transformation`, 'PUT', null);
		if (err) {
			onError(err);
			return;
		}
		let t = new Transformation();
		setTransformation(t);
		setUsedInputProperties([]);
		setUsedOutputProperties([]);
		onStatuChange({
			variant: 'success',
			icon: 'check2-circle',
			text: 'Your transformation has been successfully reset',
		});
	};

	if (transformation == null) return;

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
						<SlButton className='saveButton' variant='primary' onClick={onSave}>
							<SlIcon slot='prefix' name='save' />
							Save
						</SlButton>
						<SlButton className='resetButton' variant='danger' onClick={onReset}>
							<SlIcon slot='prefix' name='x-lg' />
							Reset
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
		</div>
	);
};

export default ConnectionTransformation;

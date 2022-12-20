import { useState, useEffect, useRef } from 'react';
import './ConnectionProperties.css';
import NotFound from '../NotFound/NotFound';
import Toast from '../../components/Toast/Toast';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import call from '../../utils/call';
import { transformationFunction } from '../../assets/docs/transformationFunction';
import {
	SlButton,
	SlIcon,
	SlDialog,
	SlTooltip,
	SlIconButton,
	SlInput,
	SlBadge,
} from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';
import Xarrow from 'react-xarrows';

const ConnectionProperties = () => {
	let [connection, setConnection] = useState({});
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
	let [status, setStatus] = useState(null);
	let [notFound, setNotFound] = useState(false);

	const toastRef = useRef();
	const connectionID = Number(String(window.location).split('/').at(-2));

	const onError = (err) => {
		setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
		toastRef.current.toast();
		return;
	};

	useEffect(() => {
		const fetchState = async () => {
			// get the connection.
			let [connection, err] = await call('/admin/connections/get', 'POST', connectionID);
			if (err) {
				onError(err);
				return;
			}
			if (connection == null) {
				setNotFound(true);
				return;
			}
			setConnection(connection);

			// get the connection properties and the user properties.
			let connectionSchema;
			[connectionSchema, err] = await call(`/api/connections/${connectionID}/schema`, 'GET');
			if (err) {
				onError(err);
				return;
			}
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
			let userProperties = [];
			for (let p of userSchema.properties) {
				userProperties.push(p);
			}

			// place the properties in the proper column.
			let inputProperties, outputProperties;
			if (connection.Role === 'Source') {
				inputProperties = connectionProperties;
				outputProperties = userProperties;
			} else if (connection.Role === 'Destination') {
				inputProperties = userProperties;
				outputProperties = connectionProperties;
			}
			setInputProperties(inputProperties);
			setOutputProperties(outputProperties);

			// get the transformations.
			let transformations;
			[transformations, err] = await call(`/api/connections/${connectionID}/transformations`, 'GET');
			if (err) {
				onError(err);
				return;
			}

			// get the input properties and the output properties used by the
			// transformations.
			let usedInputProperties = [];
			for (let t of transformations) {
				for (let input of t.In.properties) {
					let isDuplicate = false;
					for (let p of usedInputProperties) {
						if (input.name === p.name) isDuplicate = true;
					}
					if (!isDuplicate) usedInputProperties.push(input);
				}
			}
			setUsedInputProperties(usedInputProperties);
			let usedOutputProperties = [];
			for (let t of transformations) {
				let output = outputProperties.find((p) => t.Out === p.name);
				usedOutputProperties.push(output);
			}
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
		} else if (type === 'output') {
			setUsedOutputProperties([...usedOutputProperties, p]);
		}
	};

	const onRemoveUsedProperty = (e, name, type) => {
		e.stopPropagation();
		if (type === 'input') {
			let properties = usedInputProperties.filter((p) => p.name !== name);
			setUsedInputProperties(properties);
			let trs = [];
			for (let t of transformations) {
				if (t.In.properties.findIndex((p) => p.name === name) !== -1) {
					let oldDefaultTransformation = computeDefaultTransformationFunction(t);
					t.In.properties = t.In.properties.filter((p) => p.name !== name);
					if (t.SourceCode === '' || t.SourceCode === oldDefaultTransformation)
						t.SourceCode = computeDefaultTransformationFunction(t);
				}
				trs.push(t);
			}
			setTransformations(trs);
		} else if (type === 'output') {
			let properties = usedOutputProperties.filter((p) => p.name !== name);
			setUsedOutputProperties(properties);
			let trs = [];
			for (let t of transformations) {
				if (t.Out === name) t.Out = '';
				trs.push(t);
			}
			setTransformations(trs);
		}
	};

	const onAddTransformation = () => {
		let t = { Position: lastTransformationPosition, In: { properties: [] }, Out: '' };
		t.SourceCode = computeDefaultTransformationFunction(t);
		setTransformations([...transformations, t]);
		setLastTransformationPosition(lastTransformationPosition + 1);
	};

	const onChangeTransformation = (position, value) => {
		let trs = [...transformations];
		let i = trs.findIndex((t) => t.Position === position);
		trs[i].SourceCode = value === '' ? computeDefaultTransformationFunction(trs[i]) : value;
		setTransformations(trs);
	};

	const onRemoveTransformation = (position) => {
		let trs = transformations.filter((t) => t.Position !== position);
		setTransformations(trs);
		setSelectedTransformation('');
	};

	const onAddArrow = (transformationPosition) => {
		let sp = selectedProperty;
		let trs = [];
		for (let t of transformations) {
			if (t.Position === transformationPosition) {
				if (sp.type === 'input') {
					if (t.In.properties.findIndex((property) => property.name === sp.name) === -1) {
						let oldDefaultTransformation = computeDefaultTransformationFunction(t);
						let p = inputProperties.find((p) => p.name === sp.name);
						t.In.properties.push(p);
						if (t.SourceCode === '' || t.SourceCode === oldDefaultTransformation)
							t.SourceCode = computeDefaultTransformationFunction(t);
					}
				}
				if (sp.type === 'output') {
					let alreadyUsed = false;
					for (let t of transformations) {
						if (t.Out === sp.name) {
							alreadyUsed = true;
							break;
						}
					}
					if (alreadyUsed) {
						onError('output properties can be linked to only one transformation');
						return;
					} else {
						t.Out = sp.name;
					}
				}
			}
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onRemoveArrow = (transformationPosition, propertyName, propertyType, e) => {
		if (e.target.previousSibling == null || e.target.previousSibling.tagName !== 'svg') return; // the click is not on the label of the arrow.
		let trs = [];
		for (let t of transformations) {
			if (t.Position === transformationPosition) {
				if (propertyType === 'input') {
					let oldDefaultTransformation = computeDefaultTransformationFunction(t);
					let properties = t.In.properties.filter((p) => p.name !== propertyName);
					t.In.properties = properties;
					if (t.SourceCode === '' || t.SourceCode === oldDefaultTransformation)
						t.SourceCode = computeDefaultTransformationFunction(t);
				}
				if (propertyType === 'output') {
					t.Out = '';
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
			trs.push(toSave);
		}
		let [, err] = await call(`/api/connections/${connectionID}/transformations`, 'PUT', trs);
		if (err) {
			onError(err);
			return;
		}
		setStatus({
			variant: 'success',
			icon: 'check2-circle',
			text: 'Your transformations have been successfully saved',
		});
		toastRef.current.toast();
	};

	const computeDefaultTransformationFunction = (t) => {
		let f = transformationFunction;
		if (t.In.properties.length > 0) {
			let properties = '';
			t.In.properties.forEach((p, i) => {
				if (i === 0) properties += `user["${p.name}"]`;
				else properties += ` + user["${p.name}"]`;
			});
			let i = f.indexOf('return');
			f = f.substring(0, i + 7) + properties;
		}
		return f;
	};

	const isSelectedProperty = (name, type) => {
		let sp = selectedProperty;
		return sp && sp.name === name && sp.type === type;
	};

	if (notFound) {
		return <NotFound />;
	}

	let sp = selectedProperty;
	let st = selectedTransformation;
	let cn = connection;

	return (
		<div className={`ConnectionProperties${sp ? ' selectedProperty' : ''}`}>
			{sp && (
				<div className='selectedPropertyMessage'>
					<div>
						Modify the links
						{sp.type === 'input' ? ' from ' : ' to '}
						<span className='name'>"{sp.name}"</span>
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
			<Breadcrumbs
				breadcrumbs={[
					{ Name: 'Connections list', Link: '/admin/connections' },
					{ Name: `${cn.Name} properties` },
				]}
			/>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<div className='head'>
					<div className='title'>
						{cn.LogoURL !== '' && <img className='littleLogo' src={cn.LogoURL} alt={`${cn.Name}'s logo`} />}
						<div className='text'>
							{cn.Role === 'Source'
								? `Map ${cn.Name} properties to your golden record`
								: `Map your golden record to ${cn.Name} properties`}
						</div>
					</div>
					<div className='badges'>
						<SlBadge className='type' variant='neutral'>
							{cn.Type}
						</SlBadge>
						<SlBadge className='role' variant='neutral'>
							{cn.Role}
						</SlBadge>
					</div>
					<SlTooltip content='Save properties'>
						<SlButton className='saveButton' variant='primary' size='large' onClick={onSave}>
							<SlIcon slot='prefix' name='save' />
							Save
						</SlButton>
					</SlTooltip>
				</div>
				<div className='properties usedInputProperties'>
					<div className='title'>{cn.Role === 'Source' ? `${cn.Name} properties` : 'Golden record'}</div>
					<SlButton className='addUsedProperty' variant='neutral' onClick={() => setIsInputDialogOpen(true)}>
						Add property
					</SlButton>
					{usedInputProperties.map((p) => {
						return (
							<div
								key={p.name}
								className={`property${isSelectedProperty(p.name, 'input') ? ' selected' : ''}`}
								id={p.name}
								onClick={() => setSelectedProperty({ name: p.name, type: 'input' })}
							>
								<div>{p.label ? p.label : p.name}</div>
								<SlIconButton
									name='dash-circle'
									label='Remove property'
									onClick={(e) => onRemoveUsedProperty(e, p.name, 'input')}
								/>
							</div>
						);
					})}
				</div>
				<div className='transformations'>
					{transformations.map((t) => {
						return (
							<div
								key={t.Position}
								className='transformation'
								id={`transformation-${t.Position}`}
								onClick={sp ? () => onAddArrow(t.Position) : null}
							>
								<SlIconButton
									className='addTransformationFunction'
									name='braces'
									label='Add transformation'
									onClick={sp ? null : () => setSelectedTransformation(t)}
								/>
								{st && t.Position === st.Position && (
									<SlDialog
										label='Modify the transformation'
										open={true}
										onSlAfterHide={() => setSelectedTransformation(null)}
										style={{ '--width': '700px' }}
									>
										<div className='editorWrapper'>
											<Editor
												onChange={(value) => onChangeTransformation(t.Position, value)}
												defaultLanguage='python'
												value={t.SourceCode}
												theme='vs-light'
											/>
										</div>
										<SlButton
											className='removeTransformation'
											slot='footer'
											variant='danger'
											onClick={() => onRemoveTransformation(t.Position)}
										>
											Remove
										</SlButton>
									</SlDialog>
								)}
							</div>
						);
					})}
					<SlTooltip content='Add a transformation'>
						<SlButton className='addTransformation' variant='default' onClick={onAddTransformation}>
							<SlIcon name='plus'></SlIcon>
						</SlButton>
					</SlTooltip>
				</div>
				<div className='properties usedOutputProperties'>
					<div className='title'>{cn.Role === 'Source' ? `Golden record` : `${cn.Name} properties`}</div>
					<SlButton className='addUsedProperty' variant='neutral' onClick={() => setIsOutputDialogOpen(true)}>
						Add property
					</SlButton>
					{usedOutputProperties.map((p) => {
						return (
							<div
								key={p.name}
								className={`property${isSelectedProperty(p.name, 'output') ? ' selected' : ''}`}
								id={p.name}
								onClick={() => setSelectedProperty({ name: p.name, type: 'output' })}
							>
								<div>{p.label ? p.label : p.name}</div>
								<SlIconButton
									name='dash-circle'
									label='Remove property'
									onClick={(e) => onRemoveUsedProperty(e, p.name, 'output')}
								/>
							</div>
						);
					})}
				</div>
			</div>
			<div className='arrows'>
				{transformations.map((t) => {
					let arrows = t.In.properties.map((p) => {
						return (
							<div
								className={`arrow${isSelectedProperty(p.name, 'input') ? ' selected' : ''}`}
								onClick={
									isSelectedProperty(p.name, 'input')
										? (e) => {
												onRemoveArrow(t.Position, p.name, 'input', e);
										  }
										: null
								}
							>
								<Xarrow
									start={p.name}
									end={`transformation-${t.Position}`}
									startAnchor='right'
									endAnchor='left'
									showHead={false}
									color='#818cf8'
									strokeWidth={2}
									labels={isSelectedProperty(p.name, 'input') ? '-' : ''}
								/>
							</div>
						);
					});
					let out = t.Out;
					if (out === '') return arrows;
					arrows.push(
						<div
							className={`arrow${isSelectedProperty(out, 'output') ? ' selected' : ''}`}
							onClick={
								isSelectedProperty(out, 'output')
									? (e) => {
											onRemoveArrow(t.Position, out, 'output', e);
									  }
									: null
							}
						>
							<Xarrow
								start={`transformation-${t.Position}`}
								end={out}
								startAnchor='right'
								endAnchor='left'
								showHead={false}
								color='#818cf8'
								strokeWidth={2}
								labels={isSelectedProperty(out, 'output') && '-'}
							/>
						</div>
					);
					return arrows;
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

import { useState, useEffect, useRef } from 'react';
import './AccountConnectionProperties.css';
import NotFound from '../../NotFound/NotFound';
import Toast from '../../../components/Toast/Toast';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import call from '../../../utils/call';
import { defaultTransformationFunction } from '../../../utils/docs/defaultTransformationFunction';
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

const AccountConnectionProperties = () => {
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

			// get the connection properties and the golden record properties.
			let connectionProperties, goldenRecordProperties;
			[connectionProperties, err] = await call(`/api/connections/${connectionID}/properties`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			[goldenRecordProperties, err] = await call('/admin/user-schema-properties', 'GET');
			if (err) {
				onError(err);
				return;
			}

			// place the properties in the proper column.
			let inputProperties, outputProperties;
			if (connection.Role === 'Source') {
				inputProperties = connectionProperties;
				outputProperties = goldenRecordProperties;
			} else if (connection.Role === 'Destination') {
				inputProperties = goldenRecordProperties;
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
				for (let input of t.In) {
					let isDuplicate = false;
					for (let p of usedInputProperties) {
						if (input.Name === p.Name) isDuplicate = true;
					}
					if (!isDuplicate) usedInputProperties.push(input);
				}
			}
			setUsedInputProperties(usedInputProperties);
			let usedOutputProperties = [];
			for (let t of transformations) {
				let output = outputProperties.find((p) => t.Out === p.Name);
				usedOutputProperties.push(output);
			}
			setUsedOutputProperties(usedOutputProperties);

			// compute the positions of the transformations and replace their
			// 'Out' string properties with full object properties.
			let pos = lastTransformationPosition;
			for (let t of transformations) {
				let output = outputProperties.find((p) => t.Out === p.Name);
				t.Out = output;
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
			let properties = usedInputProperties.filter((p) => p.Name !== name);
			setUsedInputProperties(properties);
			let trs = [];
			for (let t of transformations) {
				if (t.In.findIndex((p) => p.Name === name) !== -1) {
					let oldDefaultTransformation = computeDefaultTransformationFunction(t);
					t.In = t.In.filter((p) => p.Name !== name);
					if (t.SourceCode === '' || t.SourceCode === oldDefaultTransformation)
						t.SourceCode = computeDefaultTransformationFunction(t);
				}
				trs.push(t);
			}
			setTransformations(trs);
		} else if (type === 'output') {
			let properties = usedOutputProperties.filter((p) => p.Name !== name);
			setUsedOutputProperties(properties);
			let trs = [];
			for (let t of transformations) {
				if (t.Out.Name === name) t.Out = null;
				trs.push(t);
			}
			setTransformations(trs);
		}
	};

	const onAddTransformation = () => {
		let t = { Position: lastTransformationPosition, In: [], Out: null };
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
		let p;
		if (sp.type === 'input') {
			p = inputProperties.find((p) => p.Name === sp.name);
		} else if (sp.type === 'output') {
			p = outputProperties.find((p) => p.Name === sp.name);
		}
		let trs = [];
		for (let t of transformations) {
			if (t.Position === transformationPosition) {
				if (sp.type === 'input') {
					if (t.In.findIndex((property) => property.Name === sp.name) === -1) {
						let oldDefaultTransformation = computeDefaultTransformationFunction(t);
						t.In.push(p);
						if (t.SourceCode === '' || t.SourceCode === oldDefaultTransformation)
							t.SourceCode = computeDefaultTransformationFunction(t);
					}
				}
				if (sp.type === 'output') {
					let alreadyUsed = false;
					for (let t of trs) {
						if (t.Out != null && t.Out.Name === sp.name) {
							alreadyUsed = true;
							break;
						}
					}
					if (alreadyUsed) {
						onError('output properties can be linked to only one transformation');
						return;
					} else {
						t.Out = p;
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
					let properties = t.In.filter((p) => p.Name !== propertyName);
					t.In = properties;
					if (t.SourceCode === '' || t.SourceCode === oldDefaultTransformation)
						t.SourceCode = computeDefaultTransformationFunction(t);
				}
				if (propertyType === 'output') {
					t.Out = null;
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
			toSave.Out = toSave.Out == null ? '' : toSave.Out.Name;
			toSave.Connection = connectionID;
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
		let f = defaultTransformationFunction;
		if (t.In.length > 0) {
			let properties = '';
			t.In.forEach((p, i) => {
				if (i === 0) properties += `user["${p.Name}"]`;
				else properties += ` + user["${p.Name}"]`;
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
		<div className={`AccountConnectionProperties${sp ? ' selectedProperty' : ''}`}>
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
					{ Name: 'Your connections', Link: '/admin/account/connections' },
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
								key={p.Name}
								className={`property${isSelectedProperty(p.Name, 'input') ? ' selected' : ''}`}
								id={p.Name}
								onClick={() => setSelectedProperty({ name: p.Name, type: 'input' })}
							>
								<div>{p.Label === '' ? p.Name : p.Label}</div>
								<SlIconButton
									name='dash-circle'
									label='Remove property'
									onClick={(e) => onRemoveUsedProperty(e, p.Name, 'input')}
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
								key={p.Name}
								className={`property${isSelectedProperty(p.Name, 'output') ? ' selected' : ''}`}
								id={p.Name}
								onClick={() => setSelectedProperty({ name: p.Name, type: 'output' })}
							>
								<div>{p.Label === '' ? p.Name : p.Label}</div>
								<SlIconButton
									name='dash-circle'
									label='Remove property'
									onClick={(e) => onRemoveUsedProperty(e, p.Name, 'output')}
								/>
							</div>
						);
					})}
				</div>
			</div>
			<div className='arrows'>
				{transformations.map((t) => {
					let arrows = t.In.map((p) => {
						return (
							<div
								className={`arrow${isSelectedProperty(p.Name, 'input') ? ' selected' : ''}`}
								onClick={
									isSelectedProperty(p.Name, 'input')
										? (e) => {
												onRemoveArrow(t.Position, p.Name, 'input', e);
										  }
										: null
								}
							>
								<Xarrow
									start={p.Name}
									end={`transformation-${t.Position}`}
									startAnchor='right'
									endAnchor='left'
									showHead={false}
									color='#818cf8'
									strokeWidth={2}
									labels={isSelectedProperty(p.Name, 'input') ? '-' : ''}
								/>
							</div>
						);
					});
					let out = t.Out;
					if (out == null) return arrows;
					arrows.push(
						<div
							className={`arrow${isSelectedProperty(out.Name, 'output') ? ' selected' : ''}`}
							onClick={
								isSelectedProperty(out.Name, 'output')
									? (e) => {
											onRemoveArrow(t.Position, out.Name, 'output', e);
									  }
									: null
							}
						>
							<Xarrow
								start={`transformation-${t.Position}`}
								end={out.Name}
								startAnchor='right'
								endAnchor='left'
								showHead={false}
								color='#818cf8'
								strokeWidth={2}
								labels={isSelectedProperty(out.Name, 'output') && '-'}
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
						let toString = p.Label === '' ? p.Name : p.Label;
						if (
							toString.includes(inputSearchTerm) ||
							toString.includes(inputSearchTerm.charAt(0).toUpperCase() + inputSearchTerm.slice(1)) ||
							toString.includes(inputSearchTerm.toUpperCase) ||
							toString.includes(inputSearchTerm.toLowerCase)
						) {
							return (
								<div
									key={p.Name}
									className={`property${
										usedInputProperties.find((up) => up.Name === p.Name) != null ? ' used' : ''
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
						let toString = p.Label === '' ? p.Name : p.Label;
						if (
							toString.includes(outputSearchTerm) ||
							toString.includes(outputSearchTerm.charAt(0).toUpperCase() + outputSearchTerm.slice(1)) ||
							toString.includes(outputSearchTerm.toUpperCase) ||
							toString.includes(outputSearchTerm.toLowerCase)
						) {
							return (
								<div
									key={p.Name}
									className={`property${
										usedOutputProperties.find((up) => up.Name === p.Name) != null ? ' used' : ''
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

export default AccountConnectionProperties;

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
	let [newTransformationID, setNewTransformationID] = useState(1);
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

	// get the connection.
	useEffect(() => {
		const fetchConnection = async () => {
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
		};
		fetchConnection();
	}, []);

	// get the properties.
	useEffect(() => {
		const fetchProperties = async () => {
			let connProperties, goldenRecordProperties, err;
			// get the connection properties.
			[connProperties, err] = await call(`/api/connections/${connectionID}/properties`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			let connectionProperties = [];
			for (let p of connProperties) connectionProperties.push(p.Name);
			// get the golden record properties.
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
		};
		// fetch the properties only if the connection is already fetched.
		if (Object.keys(connection).length === 0) return;
		fetchProperties();
	}, [connection]);

	// get the transformations with their linked used properties.
	useEffect(() => {
		const fetchTransformations = async () => {
			let [transformations, err] = await call(`/api/connections/${connectionID}/transformations`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			setTransformations(transformations);
			let counter = 1;
			for (let t of transformations) {
				t.ID = counter;
				counter += 1;
			}
			setNewTransformationID(counter);
			let usedInputProperties = [];
			for (let t of transformations) {
				for (let input of t.Inputs) {
					let isDuplicate = false;
					for (let p of usedInputProperties) {
						if (input === p) isDuplicate = true;
					}
					if (!isDuplicate) usedInputProperties.push(input);
				}
			}
			setUsedInputProperties(usedInputProperties);
			let usedOutputProperties = [];
			for (let t of transformations) {
				usedOutputProperties.push(t.Output);
			}
			setUsedOutputProperties(usedOutputProperties);
		};
		fetchTransformations();
	}, []);

	const onAddUsedProperty = (name, type) => {
		if (type === 'input') {
			let properties = [...usedInputProperties, name];
			setUsedInputProperties(properties);
		} else if (type === 'output') {
			let properties = [...usedOutputProperties, name];
			setUsedOutputProperties(properties);
		}
	};

	const onRemoveUsedProperty = (e, name, type) => {
		e.stopPropagation();
		if (type === 'input') {
			let properties = usedInputProperties.filter((p) => p !== name);
			setUsedInputProperties(properties);
			let trs = [];
			for (let t of transformations) {
				if (t.Inputs.findIndex((p) => p === name) !== -1) {
					let oldDefaultTransformation = computeDefaultTransformationFunction(t);
					t.Inputs = t.Inputs.filter((p) => p !== name);
					if (t.Source === '' || t.Source === oldDefaultTransformation)
						t.Source = computeDefaultTransformationFunction(t);
				}
				trs.push(t);
			}
			setTransformations(trs);
		} else if (type === 'output') {
			let properties = usedOutputProperties.filter((p) => p !== name);
			setUsedOutputProperties(properties);
			let trs = [];
			for (let t of transformations) {
				if (t.Output === name) t.Output = '';
				trs.push(t);
			}
			setTransformations(trs);
		}
	};

	const onAddTransformation = () => {
		let t = { ID: newTransformationID, Inputs: [], Output: '' };
		t.Source = computeDefaultTransformationFunction(t);
		let trs = [...transformations, t];
		setTransformations(trs);
		setNewTransformationID(newTransformationID + 1);
	};

	const onChangeTransformation = (id, value) => {
		let trs = [...transformations];
		let i = trs.findIndex((t) => t.ID === id);
		trs[i].Source = value === '' ? computeDefaultTransformationFunction(trs[i]) : value;
		setTransformations(trs);
	};

	const onRemoveTransformation = (id) => {
		let trs = transformations.filter((t) => t.ID !== id);
		setTransformations(trs);
		setSelectedTransformation('');
	};

	const onAddArrow = (transformationID) => {
		let p = selectedProperty;
		let trs = [];
		for (let t of transformations) {
			if (t.ID === transformationID) {
				if (p.type === 'input') {
					if (t.Inputs.findIndex((property) => property === p.name) === -1) {
						let oldDefaultTransformation = computeDefaultTransformationFunction(t);
						t.Inputs.push(p.name);
						if (t.Source === '' || t.Source === oldDefaultTransformation)
							t.Source = computeDefaultTransformationFunction(t);
					}
				}
				if (p.type === 'output') {
					let alreadyUsed = false;
					for (let t of trs) {
						if (t.Output === p.name) {
							alreadyUsed = true;
							break;
						}
					}
					if (alreadyUsed) {
						onError('output properties can be linked to only one transformation');
						return;
					} else {
						t.Output = p.name;
					}
				}
			}
			trs.push(t);
		}
		setTransformations(trs);
	};

	const onRemoveArrow = (transformationID, property, type, e) => {
		if (e.target.previousSibling == null || e.target.previousSibling.tagName !== 'svg') return; // the click is not on the label of the arrow.
		let trs = [];
		for (let t of transformations) {
			if (t.ID === transformationID) {
				if (type === 'input') {
					let properties = t.Inputs.filter((p) => p !== property);
					t.Inputs = properties;
				}
				if (type === 'output') {
					t.Output = '';
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
			delete toSave.ID;
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
		if (t.Inputs.length > 0) {
			let properties = '';
			t.Inputs.forEach((p, i) => {
				if (i === 0) properties += `user['${p}']`;
				else properties += ` + user['${p}']`;
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
			<div className='content'>
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
								key={p}
								className={`property${isSelectedProperty(p, 'input') ? ' selected' : ''}`}
								id={p}
								onClick={() => setSelectedProperty({ name: p, type: 'input' })}
							>
								<div>{p}</div>
								<SlIconButton
									name='dash-circle'
									label='Remove property'
									onClick={(e) => onRemoveUsedProperty(e, p, 'input')}
								/>
							</div>
						);
					})}
				</div>
				<div className='transformations'>
					{transformations.map((t) => {
						return (
							<div
								key={t.ID}
								className='transformation'
								id={`transformation-${t.ID}`}
								onClick={sp ? () => onAddArrow(t.ID) : null}
							>
								<SlIconButton
									className='addTransformationFunction'
									name='braces'
									label='Add transformation'
									onClick={sp ? null : () => setSelectedTransformation(t)}
								/>
								{st && t.ID === st.ID && (
									<SlDialog
										label='Modify the transformation'
										open={true}
										onSlAfterHide={() => setSelectedTransformation(null)}
										style={{ '--width': '700px' }}
									>
										<div className='editorWrapper'>
											<Editor
												onChange={(value) => onChangeTransformation(t.ID, value)}
												defaultLanguage='python'
												value={t.Source}
												theme='vs-light'
											/>
										</div>
										<SlButton
											className='removeTransformation'
											slot='footer'
											variant='danger'
											onClick={() => onRemoveTransformation(t.ID)}
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
								key={p}
								className={`property${isSelectedProperty(p, 'output') ? ' selected' : ''}`}
								id={p}
								onClick={() => setSelectedProperty({ name: p, type: 'output' })}
							>
								<div>{p}</div>
								<SlIconButton
									name='dash-circle'
									label='Remove property'
									onClick={(e) => onRemoveUsedProperty(e, p, 'output')}
								/>
							</div>
						);
					})}
				</div>
			</div>
			<div className='arrows'>
				{transformations.map((t) => {
					let arrows = t.Inputs.map((p) => {
						return (
							<div
								className={`arrow${isSelectedProperty(p, 'input') ? ' selected' : ''}`}
								onClick={
									isSelectedProperty(p, 'input')
										? (e) => {
												onRemoveArrow(t.ID, p, 'input', e);
										  }
										: null
								}
							>
								<Xarrow
									start={p}
									end={`transformation-${t.ID}`}
									startAnchor='right'
									endAnchor='left'
									showHead={false}
									color='#818cf8'
									strokeWidth={2}
									labels={isSelectedProperty(p, 'input') ? '-' : ''}
								/>
							</div>
						);
					});
					let grp = t.Output;
					if (grp === '') return arrows;
					arrows.push(
						<div
							className={`arrow${isSelectedProperty(grp, 'output') ? ' selected' : ''}`}
							onClick={
								isSelectedProperty(grp, 'output')
									? (e) => {
											onRemoveArrow(t.ID, grp, 'output', e);
									  }
									: null
							}
						>
							<Xarrow
								start={`transformation-${t.ID}`}
								end={grp}
								startAnchor='right'
								endAnchor='left'
								showHead={false}
								color='#818cf8'
								strokeWidth={2}
								labels={isSelectedProperty(grp, 'output') && '-'}
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
						if (
							p.includes(inputSearchTerm) ||
							p.includes(inputSearchTerm.charAt(0).toUpperCase() + inputSearchTerm.slice(1)) ||
							p.includes(inputSearchTerm.toUpperCase) ||
							p.includes(inputSearchTerm.toLowerCase)
						) {
							return (
								<div
									key={p}
									className={`property${
										usedInputProperties.find((up) => up === p) != null ? ' used' : ''
									}`}
								>
									<div>{p}</div>
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
						if (
							p.includes(outputSearchTerm) ||
							p.includes(outputSearchTerm.charAt(0).toUpperCase() + outputSearchTerm.slice(1)) ||
							p.includes(outputSearchTerm.toUpperCase) ||
							p.includes(outputSearchTerm.toLowerCase)
						) {
							return (
								<div
									key={p}
									className={`property${
										usedOutputProperties.find((up) => up === p) != null ? ' used' : ''
									}`}
								>
									<div>{p}</div>
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

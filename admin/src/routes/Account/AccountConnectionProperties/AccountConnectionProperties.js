import React from 'react';
import './AccountConnectionProperties.css';
import NotFound from '../../NotFound/NotFound';
import Toast from '../../../components/Toast/Toast';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import call from '../../../utils/call';
import { showError } from '../../../utils/status';
import { SlButton, SlIcon, SlDialog, SlTooltip, SlIconButton, SlInput } from '@shoelace-style/shoelace/dist/react';
import Editor from '@monaco-editor/react';
import Xarrow from 'react-xarrows';

export default class AccountConnectionProperties extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.connectionID = Number(String(window.location).split('/').at(-2));
		this.state = {
			connection: {},
			connectionProperties: [],
			leftProperties: [],
			rightProperties: [],
			transformations: [],
			newTransformationID: 0,
			searchTerm: '',
			isDialogOpen: false,
			selectedProperty: null,
			selectedTransformation: null,
			status: null,
			notFound: false,
		};
	}

	async componentDidMount() {
		let err, connection, connectionProperties, leftProperties, rightProperties, transformations;

		[connection, err] = await call('/admin/connections/get', this.connectionID);
		if (err) {
			showError.call(this, err);
			return;
		}
		if (connection == null) {
			this.setState({ notFound: true });
			return;
		}

		[connectionProperties, err] = await call('/admin/connectors-properties', {
			Connector: this.connectionID,
		});
		if (err) {
			showError.call(this, err);
			return;
		}

		[leftProperties, err] = await call('/admin/connections/get-used-properties', this.connectionID);
		if (err) {
			showError.call(this, err);
			return;
		}

		[rightProperties, err] = await call('/admin/user-schema-properties');
		if (err) {
			showError.call(this, err);
			return;
		}

		[transformations, err] = await call(`/api/connections/${this.connectionID}/transformations`);
		if (err) {
			showError.call(this, err);
			return;
		}

		let maxID = 0;
		for (let t of transformations) {
			if (t.ID > maxID) maxID = t.ID;
		}

		this.setState({
			connection: connection,
			connectionProperties: connectionProperties.Properties,
			leftProperties: leftProperties,
			rightProperties: rightProperties,
			transformations: transformations,
			newTransformationID: maxID + 1,
		});
	}

	onAddProperty = (name) => {
		let leftProperties = this.state.leftProperties;
		let property = this.state.connectionProperties.find((p) => p.Name === name);
		leftProperties.push(property);
		this.setState({ leftProperties: leftProperties });
	};

	onRemoveProperty = (name, e) => {
		e.stopPropagation();
		let leftProperties = this.state.leftProperties.filter((p) => p.Name !== name);
		let trs = this.state.transformations;
		let transformations = [];
		for (let t of trs) {
			t.InputProperties = t.InputProperties.filter((p) => p.Name !== name);
			transformations.push(t);
		}
		this.setState({ leftProperties: leftProperties, transformations: transformations });
	};

	onAddTransformation = () => {
		let transformations = this.state.transformations;
		transformations.push({
			ID: this.state.newTransformationID,
			SourceCode: '',
			InputProperties: [],
			GRProperty: '',
		});
		this.setState({ transformations: transformations, newTransformationID: this.state.newTransformationID + 1 });
	};

	onChangeTransformation = (id, value) => {
		let transformations = this.state.transformations;
		let i = transformations.findIndex((t) => t.ID === id);
		transformations[i].SourceCode = value;
		this.setState({ transformations: transformations });
	};

	onRemoveTransformation = (id) => {
		let transformations = this.state.transformations.filter((t) => t.ID !== id);
		this.setState({ selectedTransformation: '', transformations: transformations });
	};

	onAddArrow = (transformationID) => {
		let prop = this.state.selectedProperty;
		let trs = this.state.transformations;
		let transformations = [];
		for (let t of trs) {
			if (t.ID === transformationID) {
				if (prop.column === 'left') {
					if (t.InputProperties.find((p) => p.Name === prop.name) == null) {
						t.InputProperties.push({ Connection: this.connectionID, Name: prop.name });
					}
				}
				if (prop.column === 'right') {
					// check if GRProperty is already used.
					let alreadyUsed = false;
					for (let t of trs) {
						if (t.GRProperty === prop.name) {
							alreadyUsed = true;
							break;
						}
					}
					if (alreadyUsed) {
						showError.call(this, 'golden record properties can be linked to only one transformation');
						return;
					} else {
						t.GRProperty = prop.name;
					}
				}
			}
			transformations.push(t);
		}
		this.setState({ transformations: transformations });
	};

	onRemoveArrow = (transformationID, property, column, e) => {
		if (e.target.previousSibling == null || e.target.previousSibling.tagName !== 'svg') return; // the click is not on the label of the arrow.
		let trs = this.state.transformations;
		let transformations = [];
		for (let t of trs) {
			if (t.ID === transformationID) {
				if (column === 'left') {
					let properties = t.InputProperties.filter((p) => p.Name !== property);
					t.InputProperties = properties;
				}
				if (column === 'right') {
					t.GRProperty = '';
				}
			}
			transformations.push(t);
		}
		this.setState({ transformations: transformations });
	};

	onSave = async () => {
		let [, err] = await call('/admin/transformations/save', {
			connection: this.state.connection.ID,
			transformations: this.state.transformations,
		});
		if (err != null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({
			status: {
				variant: 'success',
				icon: 'check2-circle',
				text: 'Your transformations have been successfully saved',
			},
		});
		this.toast.current.toast();
	};

	isSelectedProperty = (name, column) => {
		let sp = this.state.selectedProperty;
		return sp && sp.name === name && sp.column === column;
	};

	render() {
		if (this.state.notFound) return <NotFound />;

		let sp = this.state.selectedProperty;
		let st = this.state.selectedTransformation;
		let connection = this.state.connection;
		let term = this.state.searchTerm;
		return (
			<div className={`AccountConnectionProperties${sp ? ' selectedProperty' : ''}`}>
				{sp && (
					<div className='selectedPropertyMessage'>
						<div>
							Modify the links
							{sp.column === 'left' ? ' from ' : ' to '}
							<span className='name'>"{sp.name}"</span>
						</div>
						<SlButton
							className='removeSelectedProperty'
							variant='neutral'
							onClick={() => {
								this.setState({ selectedProperty: null });
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
						{ Name: `${connection.Name} properties` },
					]}
				/>
				<div className='content'>
					<Toast reactRef={this.toast} status={this.state.status} />
					<div className='head'>
						<div className='title'>
							{connection.LogoURL !== '' && (
								<img
									className='littleLogo'
									src={connection.LogoURL}
									alt={`${connection.Name}'s logo`}
								/>
							)}
							<div className='text'>Map {connection.Name} properties to your golden record</div>
						</div>
						<SlTooltip content='Save properties'>
							<SlButton className='saveButton' variant='primary' size='large' onClick={this.onSave}>
								<SlIcon slot='prefix' name='save' />
								Save
							</SlButton>
						</SlTooltip>
					</div>
					<div className='properties leftProperties'>
						<SlButton
							className='addProperty'
							variant='neutral'
							onClick={() => {
								this.setState({ isDialogOpen: true });
							}}
						>
							Add
						</SlButton>
						{this.state.leftProperties.map((p) => {
							return (
								<div
									key={p.Name}
									className={`property${this.isSelectedProperty(p.Name, 'left') ? ' selected' : ''}`}
									id={p.Name}
									onClick={() =>
										this.setState({
											selectedProperty: { name: p.Name, column: 'left' },
										})
									}
								>
									<div>{p.Name}</div>
									<SlIconButton
										name='dash-circle'
										label='Remove property'
										onClick={(e) => this.onRemoveProperty(p.Name, e)}
									/>
								</div>
							);
						})}
					</div>
					<div className='transformations'>
						{this.state.transformations.map((t) => {
							return (
								<div
									key={t.ID}
									className='transformation'
									id={`transformation-${t.ID}`}
									onClick={sp ? () => this.onAddArrow(t.ID) : null}
								>
									<SlIconButton
										className='addTransformationFunction'
										name='braces'
										label='Add transformation'
										onClick={sp ? null : () => this.setState({ selectedTransformation: t })}
									/>
									{st && t.ID === st.ID && (
										<SlDialog
											label='Modify the transformation'
											open={true}
											onSlAfterHide={() => this.setState({ selectedTransformation: null })}
											style={{ '--width': '700px' }}
										>
											<div className='editorWrapper'>
												<Editor
													onChange={(value) => {
														this.onChangeTransformation(t.ID, value);
													}}
													defaultLanguage='python'
													value={t.SourceCode}
													theme='vs-light'
												/>
											</div>
											<SlButton
												className='removeTransformation'
												slot='footer'
												variant='danger'
												onClick={() => {
													this.onRemoveTransformation(t.ID);
												}}
											>
												Remove
											</SlButton>
										</SlDialog>
									)}
								</div>
							);
						})}
						<SlTooltip content='Add a transformation'>
							<SlButton
								className='addTransformation'
								variant='primary'
								onClick={this.onAddTransformation}
							>
								<SlIcon name='plus'></SlIcon>
							</SlButton>
						</SlTooltip>
					</div>
					<div className='properties rightProperties'>
						{this.state.rightProperties.map((p) => {
							return (
								<div
									key={p}
									id={p}
									className={`property${this.isSelectedProperty(p, 'right') ? ' selected' : ''}`}
									onClick={() =>
										this.setState({
											selectedProperty: { name: p, column: 'right' },
										})
									}
								>
									{p}
								</div>
							);
						})}
					</div>
				</div>
				<div className='arrows'>
					{this.state.transformations.map((t) => {
						let arrows = t.InputProperties.map((p) => {
							return (
								<div
									className={`arrow${this.isSelectedProperty(p.Name, 'left') ? ' selected' : ''}`}
									onClick={
										this.isSelectedProperty(p.Name, 'left')
											? (e) => {
													this.onRemoveArrow(t.ID, p.Name, 'left', e);
											  }
											: null
									}
								>
									<Xarrow
										start={p.Name}
										end={`transformation-${t.ID}`}
										startAnchor='right'
										endAnchor='left'
										showHead={false}
										color='#818cf8'
										strokeWidth={2}
										labels={this.isSelectedProperty(p.Name, 'left') ? '-' : ''}
									/>
								</div>
							);
						});
						let grp = t.GRProperty;
						if (grp === '') return arrows;
						arrows.push(
							<div
								className={`arrow${this.isSelectedProperty(grp, 'right') ? ' selected' : ''}`}
								onClick={
									this.isSelectedProperty(grp, 'right')
										? (e) => {
												this.onRemoveArrow(t.ID, grp, 'right', e);
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
									labels={this.isSelectedProperty(grp, 'right') && '-'}
								/>
							</div>
						);
						return arrows;
					})}
				</div>
				<SlDialog
					label='Add a property'
					open={this.state.isDialogOpen}
					onSlAfterHide={() => this.setState({ isDialogOpen: false })}
					style={{ '--width': '700px' }}
				>
					<SlInput
						type='search'
						clearable
						placeholder='search'
						value={term}
						onSlInput={(e) => {
							this.setState({ searchTerm: e.currentTarget.value });
						}}
					>
						<SlIcon name='search' slot='prefix'></SlIcon>
					</SlInput>
					<div className='properties'>
						{this.state.connectionProperties.map((p) => {
							if (
								p.Name.includes(term) ||
								p.Name.includes(term.charAt(0).toUpperCase() + term.slice(1)) ||
								p.Name.includes(term.toUpperCase) ||
								p.Name.includes(term.toLowerCase)
							) {
								return (
									<div
										key={p.Name}
										className={`property${
											this.state.leftProperties.find((lp) => lp.Name === p.Name) != null
												? ' used'
												: ''
										}`}
									>
										<div>{p.Name}</div>
										<SlIconButton
											name='plus-circle'
											label='Add property'
											onClick={(e) => this.onAddProperty(p.Name)}
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
	}
}

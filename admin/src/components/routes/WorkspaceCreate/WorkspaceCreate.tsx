import React, { useState, useContext, useRef, useEffect } from 'react';
import './WorkspaceCreate.css';
import { ObjectType } from '../../../lib/api/types/types';
import { UIPreferences } from '../../../lib/api/types/workspace';
import API from '../../../lib/api/api';
import appContext from '../../../context/AppContext';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import { PostgreSQLSettings } from '../../base/PostgreSQLSettings/PostgreSQLSettings';
import { SnowflakeSettings } from '../../base/SnowflakeSettings/SnowflakeSettings';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import InitialSchema from './InitialSchema.json';
import * as icons from '../../../constants/icons';
import { IS_DOCKER_KEY } from '../../../constants/storage';
import { ExternalLogo } from '../ExternalLogo/ExternalLogo';
import { WAREHOUSES_ASSETS_PATH } from '../../../constants/paths';
import Section from '../../base/Section/Section';

const postgresqlIcon = <ExternalLogo slot='prefix' code='postgresql' path={WAREHOUSES_ASSETS_PATH} />;

const snowflakeIcon = <ExternalLogo slot='prefix' code='snowflake' path={WAREHOUSES_ASSETS_PATH} />;

const WorkspaceCreate = () => {
	const [name, setName] = useState<string>('');
	const [selectedWarehouse, setSelectedWarehouse] = useState<string>(
		localStorage.getItem(IS_DOCKER_KEY) != null ? 'PostgreSQL-Docker' : 'Snowflake',
	);
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [isCheckingWarehouse, setIsCheckingWarehouse] = useState<boolean>(false);
	const [isCreatingWorkspace, setIsCreatingWorkspace] = useState<boolean>(false);

	const nameInputRef = useRef<any>();

	const { handleError, api, setSelectedWorkspace, setIsLoadingState, redirect, showStatus, workspaces } =
		useContext(appContext);

	useEffect(() => {
		// automatically focus the name input.
		setTimeout(() => {
			nameInputRef.current?.focus();
		}, 50);
	}, []);

	const onNameInput = (e) => setName(e.target.value);

	const onChangeWarehouse = (e) => {
		setSelectedWarehouse(e.target.value);
		// reset the settings.
		setWarehouseSettings(undefined);
	};

	const onCancel = () => {
		redirect('workspaces');
	};

	const onWarehouseAction = async (action: 'test' | 'create') => {
		try {
			validateWorkspaceName(name);
		} catch (err) {
			handleError(err);
			return;
		}

		if (action === 'test') {
			setIsCheckingWarehouse(true);
		} else {
			setIsCreatingWorkspace(true);
		}

		let warehouse = selectedWarehouse;
		let settings = warehouseSettings;
		let mcpSettings = null;
		if (selectedWarehouse === 'PostgreSQL-Docker') {
			warehouse = 'PostgreSQL';
			settings = {
				host: 'warehouse',
				port: 5432,
				username: 'warehouse',
				password: 'warehouse',
				database: 'warehouse',
				schema: 'public',
			};
			mcpSettings = {
				host: 'warehouse',
				port: 5432,
				username: 'warehouse_ro',
				password: 'warehouse_ro',
				database: 'warehouse',
				schema: 'public',
			};
		}

		let uiProperties: UIPreferences = {
			profile: {
				image: '',
				firstName: 'first_name',
				lastName: 'last_name',
				extra: 'email',
			},
		};

		if (action == 'test') {
			try {
				await api.workspaces.testCreation(
					name,
					InitialSchema as ObjectType,
					warehouse,
					'Normal',
					settings,
					uiProperties,
				);
			} catch (err) {
				setTimeout(() => {
					setIsCheckingWarehouse(false);
					handleError(err);
				}, 300);
				return;
			}
			setTimeout(() => {
				setIsCheckingWarehouse(false);
				showStatus({
					variant: 'success',
					icon: icons.OK,
					text: `${selectedWarehouse} responded successfully`,
				});
			}, 300);
		} else {
			let id: number;
			try {
				const res = await api.workspaces.create(
					name,
					InitialSchema as ObjectType,
					warehouse,
					'Normal',
					settings,
					uiProperties,
				);
				id = res.id;
			} catch (err) {
				setIsCreatingWorkspace(false);
				handleError(err);
				return;
			}
			// TODO(Gianluca): this call was written just to work and is just a
			// prototype, which needs to be reviewed.
			try {
				const newApi = new API(window.location.origin, id);
				await newApi.workspaces.updateWarehouse(name, 'Normal', settings, mcpSettings, false);
			} catch (err) {
				handleError(err);
				return;
			}
			setIsCreatingWorkspace(false);
			setSelectedWorkspace(id);
			setIsLoadingState(true);
			redirect('connections');
			if (localStorage.getItem(IS_DOCKER_KEY) != null) {
				localStorage.removeItem(IS_DOCKER_KEY);
			}
		}
	};

	const hasWorkspaces = workspaces.length > 0;
	const isDocker = localStorage.getItem(IS_DOCKER_KEY) != null;

	return (
		<div className='workspace-create'>
			<div className='workspace-create__title'>
				{!hasWorkspaces ? 'Create your first workspace' : 'Create a new workspace'}
			</div>
			<Section
				title='Name'
				description={'Choose a name for the workspace. It can be changed later.'}
				padded={true}
				annotated={true}
			>
				<SlInput
					className='workspace-create__name'
					maxlength={100}
					value={name}
					onSlInput={onNameInput}
					ref={nameInputRef}
					placeholder='My workspace'
				/>
			</Section>
			<Section
				title='Data warehouse'
				description={
					<p className='workspace-create__data-warehouse-description'>
						This is the data warehouse where your user data and events will be stored.
						<br />
						<div className='workspace-create__data-warehouse-description-second-p'>
							It must be an empty database. Meergo will create views and tables.
						</div>
					</p>
				}
				padded={true}
				annotated={true}
			>
				<SlSelect
					className='workspace-create__warehouse-list'
					value={selectedWarehouse}
					onSlChange={onChangeWarehouse}
				>
					{selectedWarehouse === 'PostgreSQL' || selectedWarehouse === 'PostgreSQL-Docker'
						? postgresqlIcon
						: snowflakeIcon}
					<SlOption value='PostgreSQL'>
						{postgresqlIcon}
						PostgreSQL
					</SlOption>
					<SlOption value='Snowflake'>
						{snowflakeIcon}
						Snowflake
					</SlOption>
					{isDocker && (
						<SlOption value='PostgreSQL-Docker'>
							{postgresqlIcon}
							PostgreSQL via Docker
						</SlOption>
					)}
				</SlSelect>
				{selectedWarehouse === 'PostgreSQL-Docker' ? (
					<div className='workspace-create__docker-description'>
						Since you are using Meergo with Docker you can easily create a new workspace by connecting it to
						the PostgreSQL warehouse provided directly by our image.
					</div>
				) : (
					<div className='workspace-create__warehouse-settings'>
						{selectedWarehouse === 'PostgreSQL' ? (
							<PostgreSQLSettings
								settings={warehouseSettings}
								setSettings={setWarehouseSettings}
								precompileDefault={true}
							/>
						) : (
							<SnowflakeSettings
								settings={warehouseSettings}
								setSettings={setWarehouseSettings}
								precompileDefault={true}
							/>
						)}
					</div>
				)}
			</Section>
			<div className='workspace-create__buttons'>
				{hasWorkspaces && (
					<SlButton className='workspace-create__cancel-button' onClick={onCancel}>
						Cancel
					</SlButton>
				)}
				<SlButton
					className='workspace-create__check-button'
					onClick={() => onWarehouseAction('test')}
					loading={isCheckingWarehouse}
				>
					Check warehouse
				</SlButton>
				<SlButton
					className='workspace-create__create-button'
					variant='primary'
					onClick={() => onWarehouseAction('create')}
					loading={isCreatingWorkspace}
				>
					Create workspace
				</SlButton>
			</div>
		</div>
	);
};

const validateWorkspaceName = (name: string) => {
	const n = Array.from(name);
	if (n.length === 0) {
		throw new Error('Name cannot be empty');
	} else if (n.length > 100) {
		throw new Error('Name cannot be longer than 100 characters');
	}
};

export { WorkspaceCreate };

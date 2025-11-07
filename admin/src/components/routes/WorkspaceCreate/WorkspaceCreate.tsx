import React, { useState, useContext, useRef, useEffect } from 'react';
import './WorkspaceCreate.css';
import { ObjectType } from '../../../lib/api/types/types';
import { UIPreferences } from '../../../lib/api/types/workspace';
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

const postgresqlIcon = <ExternalLogo slot='prefix' code='postgresql' />;

const snowflakeIcon = <ExternalLogo slot='prefix' code='snowflake' />;

const WorkspaceCreate = () => {
	const [name, setName] = useState<string>('');
	const [selectedWarehouse, setSelectedWarehouse] = useState<string>(
		localStorage.getItem(IS_DOCKER_KEY) != null ? 'PostgreSQL-Docker' : 'PostgreSQL',
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
		}

		let uiProperties: UIPreferences = {
			userProfile: {
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
					null,
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
					null,
					uiProperties,
				);
				id = res.id;
			} catch (err) {
				setIsCreatingWorkspace(false);
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
			<div className='workspace-create__heading'>
				<div className='workspace-create__title'>Create workspace</div>
				{!hasWorkspaces && (
					<div className='workspace-create__description'>
						Currently you don't have any workspace. Create a workspace to continue.
					</div>
				)}
			</div>
			<SlInput
				className='workspace-create__name'
				maxlength={100}
				label='Name'
				value={name}
				onSlInput={onNameInput}
				ref={nameInputRef}
				placeholder='My workspace'
				helpText='A name for the workspace, so it can be easily recognized among other workspaces. It can be changed later.'
			/>
			<SlSelect
				className='workspace-create__warehouse-list'
				value={selectedWarehouse}
				onSlChange={onChangeWarehouse}
				label='Data warehouse'
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
					Since you are using Meergo with Docker you can easily create a new workspace by connecting it to the
					PostgreSQL warehouse provided directly by our image.
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

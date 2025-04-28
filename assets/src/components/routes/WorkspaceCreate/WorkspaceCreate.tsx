import React, { useState, useContext } from 'react';
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

const WorkspaceCreate = () => {
	const [name, setName] = useState<string>('');
	const [selectedWarehouse, setSelectedWarehouse] = useState<string>('PostgreSQL');
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [isCheckingWarehouse, setIsCheckingWarehouse] = useState<boolean>(false);
	const [isAddingWorkspace, setIsAddingWorkspace] = useState<boolean>(false);

	const { handleError, api, setSelectedWorkspace, setIsLoadingState, redirect, showStatus, workspaces } =
		useContext(appContext);

	const onNameInput = (e) => setName(e.target.value);

	const onChangeWarehouse = (e) => {
		setSelectedWarehouse(e.target.value);
		// reset the settings.
		setWarehouseSettings(undefined);
	};

	const onCancel = () => {
		redirect('workspaces');
	};

	const onTestWorkspaceCreation = async () => {
		try {
			validateWorkspaceName(name);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsCheckingWarehouse(true);
		let uiProperties: UIPreferences = {
			userProfile: {
				image: '',
				firstName: 'first_name',
				lastName: 'last_name',
				extra: 'email',
			},
		};
		try {
			await api.workspaces.testCreation(
				name,
				InitialSchema as ObjectType,
				selectedWarehouse,
				'Normal',
				warehouseSettings,
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
	};

	const onCreateWorkspace = async () => {
		try {
			validateWorkspaceName(name);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsAddingWorkspace(true);
		let id: number;
		let uiPreferences: UIPreferences = {
			userProfile: {
				image: '',
				firstName: 'first_name',
				lastName: 'last_name',
				extra: 'email',
			},
		};
		try {
			const res = await api.workspaces.create(
				name,
				InitialSchema as ObjectType,
				selectedWarehouse,
				'Normal',
				warehouseSettings,
				uiPreferences,
			);
			id = res.id;
		} catch (err) {
			setIsAddingWorkspace(false);
			handleError(err);
			return;
		}
		setIsAddingWorkspace(false);
		setSelectedWorkspace(id);
		setIsLoadingState(true);
		redirect('settings');
	};

	const hasWorkspaces = workspaces.length > 0;

	return (
		<div className='workspace-add'>
			<div className='workspace-add__heading'>
				<div className='workspace-add__title'>Add workspace</div>
				{!hasWorkspaces && (
					<div className='workspace-add__description'>
						Currently you don't have any workspace. Add a workspace to continue.
					</div>
				)}
			</div>
			<SlInput
				className='workspace-add__name'
				maxlength={100}
				label='Name'
				value={name}
				onSlInput={onNameInput}
			/>
			<SlSelect value={selectedWarehouse} onSlChange={onChangeWarehouse} label='Warehouse'>
				<SlOption value='PostgreSQL'>PostgreSQL</SlOption>
				<SlOption value='Snowflake'>Snowflake</SlOption>
			</SlSelect>
			<div className='workspace-add__warehouse-settings'>
				{selectedWarehouse === 'PostgreSQL' ? (
					<PostgreSQLSettings settings={warehouseSettings} setSettings={setWarehouseSettings} />
				) : (
					<SnowflakeSettings settings={warehouseSettings} setSettings={setWarehouseSettings} />
				)}
			</div>
			<div className='workspace-add__buttons'>
				{hasWorkspaces && (
					<SlButton className='workspace-add__cancel-button' onClick={onCancel}>
						Cancel
					</SlButton>
				)}
				<SlButton
					className='workspace-add__check-button'
					onClick={onTestWorkspaceCreation}
					loading={isCheckingWarehouse}
				>
					Check warehouse
				</SlButton>
				<SlButton
					className='workspace-add__add-button'
					variant='primary'
					onClick={onCreateWorkspace}
					loading={isAddingWorkspace}
				>
					Add workspace
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

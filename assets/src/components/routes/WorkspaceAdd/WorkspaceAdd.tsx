import React, { useState, useContext } from 'react';
import './WorkspaceAdd.css';
import appContext from '../../../context/AppContext';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import { PostgreSQLSettings } from '../../base/PostgreSQLSettings/PostgreSQLSettings';
import { SnowflakeSettings } from '../../base/SnowflakeSettings/SnowflakeSettings';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import * as icons from '../../../constants/icons';

const WorkspaceAdd = () => {
	const [name, setName] = useState<string>('');
	const [useEuropeRegion, setUseEuropeRegion] = useState<boolean>(false);
	const [selectedWarehouse, setSelectedWarehouse] = useState<string>('PostgreSQL');
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [isCheckingWarehouse, setIsCheckingWarehouse] = useState<boolean>(false);

	const { handleError, api, setSelectedWorkspace, setIsLoadingState, redirect, showStatus } = useContext(appContext);

	const onNameInput = (e) => setName(e.target.value);

	const onUseEuropeRegionChange = () => setUseEuropeRegion(!useEuropeRegion);

	const onChangeWarehouse = (e) => {
		setSelectedWarehouse(e.target.value);
	};

	const onCancel = () => {
		redirect('workspaces');
	};

	const onCheckWarehouse = async () => {
		setIsCheckingWarehouse(true);
		try {
			await api.canInitializeWarehouse(selectedWarehouse, 'Normal', warehouseSettings);
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

	const onAddWorkspace = async () => {
		const privacyRegion = useEuropeRegion ? 'Europe' : '';
		try {
			validateWorkspaceName(name);
		} catch (err) {
			handleError(err);
			return;
		}
		let id: number;
		try {
			const res = await api.workspaces.add(name, privacyRegion, selectedWarehouse, 'Normal', warehouseSettings);
			id = res.id;
		} catch (err) {
			handleError(err);
			return;
		}
		setSelectedWorkspace(id);
		setIsLoadingState(true);
		redirect('settings');
	};

	return (
		<div className='workspace-add'>
			<div className='workspace-add__title'>Add workspace</div>
			<SlInput
				className='workspace-add__name'
				maxlength={100}
				label='Name'
				value={name}
				onSlInput={onNameInput}
			/>
			<SlCheckbox
				className='workspace-add__europe-region'
				checked={useEuropeRegion}
				onSlChange={onUseEuropeRegionChange}
			>
				Use the European Privacy Region <span>(can be changed later)</span>
			</SlCheckbox>
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
				<SlButton className='workspace-add__cancel-button' onClick={onCancel}>
					Cancel
				</SlButton>
				<SlButton
					className='workspace-add__check-button'
					onClick={onCheckWarehouse}
					loading={isCheckingWarehouse}
				>
					Check warehouse
				</SlButton>
				<SlButton className='workspace-add__add-button' variant='primary' onClick={onAddWorkspace}>
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

export { WorkspaceAdd };

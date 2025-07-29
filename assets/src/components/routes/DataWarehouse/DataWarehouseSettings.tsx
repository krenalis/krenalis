import React, { useContext, useState, useLayoutEffect } from 'react';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { Warehouse } from './DataWarehouse.helpers';
import appContext from '../../../context/AppContext';
import * as icons from '../../../constants/icons';
import { WarehouseMode, WarehouseSettings } from '../../../lib/api/types/warehouse';
import objectKeysToLower from '../../../utils/objectKeysToLower';
import { UnprocessableError } from '../../../lib/api/errors';
import { PostgreSQLSettings } from '../../base/PostgreSQLSettings/PostgreSQLSettings';
import { SnowflakeSettings } from '../../base/SnowflakeSettings/SnowflakeSettings';

interface DataWarehouseSettingsProps {
	selectedWarehouse: Warehouse;
	setSelectedWarehouse: React.Dispatch<React.SetStateAction<Warehouse | undefined>>;
	currentMode: WarehouseMode | undefined;
	currentSettings: WarehouseSettings | undefined;
	currentMCPSettings: WarehouseSettings | undefined;
}

const DataWarehouseSettings = ({
	selectedWarehouse,
	setSelectedWarehouse,
	currentMode,
	currentSettings,
	currentMCPSettings,
}: DataWarehouseSettingsProps) => {
	const [mode, setMode] = useState<WarehouseMode>(currentMode || 'Normal');
	const [settings, setSettings] = useState<Record<string, any> | undefined>(objectKeysToLower(currentSettings));
	const [mcpSettings, setMCPSettings] = useState<Record<string, any> | undefined>(
		objectKeysToLower(currentMCPSettings),
	);
	const [isCheckLoading, setIsCheckLoading] = useState<boolean>(false);
	const [isActionButtonLoading, setIsActionButtonLoading] = useState<boolean>(false);

	const { setTitle, api, handleError, showStatus, setIsLoadingWorkspaces } = useContext(appContext);

	useLayoutEffect(() => {
		setTitle(`${selectedWarehouse.name} settings`);
	}, []);

	const onCancelClick = () => setSelectedWarehouse(null);

	const onChangeMode = (e) => {
		setMode(e.target.value);
	};

	const onCheck = async () => {
		const timeout = setTimeout(() => setIsCheckLoading(true), 300);
		try {
			await api.workspaces.testWarehouseUpdate(settings, mcpSettings);
		} catch (err) {
			handleError(err);
			clearTimeout(timeout);
			setIsCheckLoading(false);
			return;
		}
		showStatus({
			variant: 'success',
			icon: icons.OK,
			text: `The ${selectedWarehouse.name} warehouse with the specified settings is valid`,
		});
		clearTimeout(timeout);
		setIsCheckLoading(false);
	};

	const onSave = async () => {
		setIsActionButtonLoading(true);
		try {
			await api.workspaces.updateWarehouse(
				selectedWarehouse.name,
				mode,
				settings,
				mcpSettings != undefined && Object.values(mcpSettings).some((v) => v !== '' && v !== 0)
					? mcpSettings
					: null,
				false,
			);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				if (err.code === 'InvalidWarehouseType') {
					handleError(
						'The workspace has already been connected to a different type of data warehouse. Please reload to see the connected data warehouse.',
					);
					setIsActionButtonLoading(false);
					return;
				}
			}
			handleError(err);
			setIsActionButtonLoading(false);
			return;
		}
		setTimeout(() => {
			setIsActionButtonLoading(false);
			setSelectedWarehouse(null);
			setIsLoadingWorkspaces(true);
		}, 500);
	};

	return (
		<div className='warehouse-settings'>
			<div className='warehouse-settings__info'>
				<div className='warehouse-settings__icon'>
					<LittleLogo icon={selectedWarehouse.icon} />
				</div>
				<p className='warehouse-settings__name'>{selectedWarehouse.name}</p>
			</div>
			<div className='warehouse-settings__settings'>
				<SlSelect label='Mode' className='warehouse-settings__mode' value={mode} onSlChange={onChangeMode}>
					<SlOption value='Normal'>
						<div className='warehouse-settings__mode-title'>Normal</div>
						<div className='warehouse-settings__mode-description'>
							{' '}
							<span className='warehouse-settings__mode-separator'>-</span> Full read and write access
						</div>
					</SlOption>
					<SlOption value='Inspection'>
						<div className='warehouse-settings__mode-title'>Inspection</div>
						<div className='warehouse-settings__mode-description'>
							{' '}
							<span className='warehouse-settings__mode-separator'>-</span> Read-only for data inspection
						</div>
					</SlOption>
					<SlOption value='Maintenance'>
						<div className='warehouse-settings__mode-title'>Maintenance</div>
						<div className='warehouse-settings__mode-description'>
							{' '}
							<span className='warehouse-settings__mode-separator'>-</span> Init and alter schema
							operations only
						</div>
					</SlOption>
				</SlSelect>
				<h2>Warehouse settings</h2>
				{selectedWarehouse.name === 'PostgreSQL' ? (
					<PostgreSQLSettings setSettings={setSettings} settings={settings} precompileDefault={true} />
				) : (
					<SnowflakeSettings setSettings={setSettings} settings={settings} precompileDefault={true} />
				)}
				<h2>MCP settings (optional)</h2>
				<div>
					Here you can configure a {selectedWarehouse.name} user that will be used by the MCP (Model Context
					Protocol) server to access and query the data warehouse.
				</div>
				<div>
					<b>It is highly recommended that this user only have read-only access to the data warehouse</b>,
					otherwise the MCP client may run destructive operations on your data.
				</div>
				<div>
					If you leave these fields blank, the MCP user will not be configured and MCP features won't be
					available.
				</div>
				{selectedWarehouse.name === 'PostgreSQL' ? (
					<PostgreSQLSettings setSettings={setMCPSettings} settings={mcpSettings} precompileDefault={false} />
				) : (
					<SnowflakeSettings setSettings={setMCPSettings} settings={mcpSettings} precompileDefault={false} />
				)}
			</div>

			<div className='warehouse-settings__buttons'>
				<SlButton disabled={isCheckLoading || isActionButtonLoading} variant='default' onClick={onCancelClick}>
					Cancel
				</SlButton>
				<SlButton
					disabled={isCheckLoading || isActionButtonLoading}
					loading={isCheckLoading}
					variant='default'
					onClick={onCheck}
				>
					Check
				</SlButton>
				<SlButton
					disabled={isCheckLoading || isActionButtonLoading}
					loading={isActionButtonLoading}
					variant='primary'
					onClick={onSave}
				>
					Save
				</SlButton>
			</div>
		</div>
	);
};

export default DataWarehouseSettings;

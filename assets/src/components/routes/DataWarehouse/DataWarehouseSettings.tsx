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
}

const DataWarehouseSettings = ({
	selectedWarehouse,
	setSelectedWarehouse,
	currentMode,
	currentSettings,
}: DataWarehouseSettingsProps) => {
	const [mode, setMode] = useState<WarehouseMode>(currentMode || 'Normal');
	const [settings, setSettings] = useState<Record<string, any> | undefined>(objectKeysToLower(currentSettings));
	const [isPingLoading, setIsPingLoading] = useState<boolean>(false);
	const [isActionButtonLoading, setIsActionButtonLoading] = useState<boolean>(false);

	const { setTitle, api, handleError, showStatus, setIsLoadingWorkspaces } = useContext(appContext);

	useLayoutEffect(() => {
		setTitle(`${selectedWarehouse.label} settings`);
	}, []);

	const onCancelClick = () => setSelectedWarehouse(null);

	const onChangeMode = (e) => {
		setMode(e.target.value);
	};

	const onPing = async () => {
		const timeout = setTimeout(() => setIsPingLoading(true), 300);
		try {
			await api.workspaces.pingWarehouse(selectedWarehouse.label, settings);
		} catch (err) {
			handleError(err);
			clearTimeout(timeout);
			setIsPingLoading(false);
			return;
		}
		showStatus({
			variant: 'success',
			icon: icons.OK,
			text: `${selectedWarehouse.label} responded successfully`,
		});
		clearTimeout(timeout);
		setIsPingLoading(false);
	};

	const onSave = async () => {
		setIsActionButtonLoading(true);
		try {
			await api.workspaces.changeWarehouseSettings(selectedWarehouse.label, mode, settings, false);
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
				<p className='warehouse-settings__name'>{selectedWarehouse.label}</p>
			</div>
			<div className='warehouse-settings__settings'>
				{selectedWarehouse.name === 'postgresql' ? (
					<PostgreSQLSettings setSettings={setSettings} settings={settings} />
				) : (
					<SnowflakeSettings setSettings={setSettings} settings={settings} />
				)}
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
			</div>

			<div className='warehouse-settings__buttons'>
				<SlButton disabled={isPingLoading || isActionButtonLoading} variant='default' onClick={onCancelClick}>
					Cancel
				</SlButton>
				<SlButton
					disabled={isPingLoading || isActionButtonLoading}
					loading={isPingLoading}
					variant='default'
					onClick={onPing}
				>
					Ping
				</SlButton>
				<SlButton
					disabled={isPingLoading || isActionButtonLoading}
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

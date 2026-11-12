import React, { useContext, useState, useLayoutEffect } from 'react';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { Warehouse } from './DataWarehouse.helpers';
import appContext from '../../../context/AppContext';
import * as icons from '../../../constants/icons';
import { WarehouseMode, WarehouseSettings } from '../../../lib/api/types/warehouse';
import { PostgreSQLSettings } from '../../base/PostgreSQLSettings/PostgreSQLSettings';
import { SnowflakeSettings } from '../../base/SnowflakeSettings/SnowflakeSettings';
import Section from '../../base/Section/Section';
import { warehouseSectionTexts } from './DataWarehouse';
import { WAREHOUSES_ASSETS_PATH } from '../../../constants/paths';

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
	const [settings, setSettings] = useState<Record<string, any> | undefined>(currentSettings);
	const [isCheckLoading, setIsCheckLoading] = useState<boolean>(false);
	const [isPipelineButtonLoading, setIsPipelineButtonLoading] = useState<boolean>(false);

	const { setTitle, api, handleError, showStatus, setIsLoadingWorkspaces } = useContext(appContext);

	useLayoutEffect(() => {
		setTitle(`${selectedWarehouse.name} settings`);
	}, []);

	const onCancelClick = () => setSelectedWarehouse(null);

	const onChangeMode = (e) => {
		setMode(e.target.value);
	};

	const onCheck = async () => {
		setIsCheckLoading(true);
		try {
			await api.workspaces.testWarehouseUpdate(settings);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsCheckLoading(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			showStatus({
				variant: 'success',
				icon: icons.OK,
				text: `The ${selectedWarehouse.name} warehouse with the specified settings is valid`,
			});
			setIsCheckLoading(false);
		}, 300);
	};

	const onSave = async () => {
		setIsPipelineButtonLoading(true);
		try {
			await api.workspaces.updateWarehouse(
				selectedWarehouse.name,
				mode,
				settings,
				false,
			);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsPipelineButtonLoading(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			setIsPipelineButtonLoading(false);
			setSelectedWarehouse(null);
			setIsLoadingWorkspaces(true);
		}, 300);
	};

	return (
		<div className='warehouse-settings'>
			<div className='warehouse-settings__info'>
				<div className='warehouse-settings__icon'>
					<LittleLogo code={selectedWarehouse.code} path={WAREHOUSES_ASSETS_PATH} />
				</div>
				<p className='warehouse-settings__name'>{selectedWarehouse.name}</p>
			</div>
			<div className='warehouse-settings__settings'>
				<Section
					title={warehouseSectionTexts.mode.title}
					description={warehouseSectionTexts.mode.description}
					padded={true}
					annotated={true}
				>
					<SlSelect className='warehouse-settings__mode' value={mode} onSlChange={onChangeMode}>
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
								<span className='warehouse-settings__mode-separator'>-</span> Read-only for data
								inspection
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
				</Section>
				<Section
					title={warehouseSectionTexts.main.title}
					description={warehouseSectionTexts.main.description}
					padded={true}
					annotated={true}
				>
					<div className='warehouse-settings__main-form'>
						{selectedWarehouse.name === 'PostgreSQL' ? (
							<PostgreSQLSettings
								setSettings={setSettings}
								settings={settings}
								precompileDefault={true}
							/>
						) : (
							<SnowflakeSettings setSettings={setSettings} settings={settings} precompileDefault={true} />
						)}
					</div>
				</Section>
			</div>
			<div className='warehouse-settings__buttons'>
				<SlButton
					disabled={isCheckLoading || isPipelineButtonLoading}
					variant='default'
					onClick={onCancelClick}
				>
					Cancel
				</SlButton>
				<SlButton
					disabled={isCheckLoading || isPipelineButtonLoading}
					loading={isCheckLoading}
					variant='default'
					onClick={onCheck}
				>
					Check
				</SlButton>
				<SlButton
					disabled={isCheckLoading || isPipelineButtonLoading}
					loading={isPipelineButtonLoading}
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

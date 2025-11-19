import React, { useState, useContext, useEffect, ReactNode, useMemo } from 'react';
import './DataWarehouse.css';
import appContext from '../../../context/AppContext';
import { Warehouse, warehouses } from './DataWarehouse.helpers';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import PasswordToggle from '../../base/PasswordToggle/PasswordToggle';
import { WarehouseMode, WarehouseSettings } from '../../../lib/api/types/warehouse';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import Section from '../../base/Section/Section';
import DataWarehouseSettings from './DataWarehouseSettings';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { WAREHOUSES_ASSETS_PATH } from '../../../constants/paths';

const DataWarehouse = () => {
	const [connectedWarehouse, setConnectedWarehouse] = useState<string>();
	const [selectedWarehouse, setSelectedWarehouse] = useState<Warehouse>();
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [warehouseMCPSettings, setWarehouseMCPSettings] = useState<WarehouseSettings>();
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, handleError, selectedWorkspace, workspaces } = useContext(appContext);

	useEffect(() => {
		const fetchWarehouse = async () => {
			try {
				const response = await api.workspaces.warehouse();
				setConnectedWarehouse(response.name);
				setWarehouseSettings(response.settings);
				setWarehouseMCPSettings(response.mcpSettings);
			} catch (err) {
				handleError(err);
				return;
			}
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		fetchWarehouse();
	}, [selectedWorkspace, selectedWarehouse]);

	if (isLoading) {
		return (
			<SlSpinner
				className='data-warehouse__spinner'
				style={
					{
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			></SlSpinner>
		);
	}

	const warehouseMode = workspaces.find((w) => w.id === selectedWorkspace).warehouseMode;

	return (
		<div className='data-warehouse'>
			{selectedWarehouse ? (
				<DataWarehouseSettings
					selectedWarehouse={selectedWarehouse}
					setSelectedWarehouse={setSelectedWarehouse}
					currentMode={warehouseMode}
					currentSettings={warehouseSettings}
					currentMCPSettings={warehouseMCPSettings}
				/>
			) : (
				<WarehouseInfo
					warehouseName={connectedWarehouse}
					warehouseSettings={warehouseSettings!}
					warehouseMCPSettings={warehouseMCPSettings}
					warehouseMode={warehouseMode}
					setSelectedWarehouse={setSelectedWarehouse}
				/>
			)}
		</div>
	);
};

interface WarehouseInfoProps {
	warehouseName: string;
	warehouseSettings: WarehouseSettings;
	warehouseMCPSettings: WarehouseSettings;
	warehouseMode: WarehouseMode;
	setSelectedWarehouse: React.Dispatch<React.SetStateAction<Warehouse | undefined>>;
}

const warehouseModeDescriptions: Record<WarehouseMode, string> = {
	Normal: 'Full read and write access',
	Inspection: 'Read-only for data inspection',
	Maintenance: 'Init and alter schema operations only',
};

const warehouseSectionTexts = {
	mode: {
		title: 'Mode',
		description: 'The mode of accessing the data warehouse',
	},
	main: {
		title: 'Main credentials',
		description: 'Read and write credentials used by Meergo for accessing the data warehouse',
	},
	mcp: {
		title: 'MCP credentials',
		description:
			'Read-only credentials used by the built-in MCP server (Model Context Protocol server) for accessing the data warehouse',
	},
};

const warehouseInfoColumns: GridColumn[] = [
	{
		name: 'Field',
	},
	{
		name: 'Value',
	},
];

const WarehouseInfo = ({
	warehouseName,
	warehouseSettings,
	warehouseMCPSettings,
	warehouseMode,
	setSelectedWarehouse,
}: WarehouseInfoProps) => {
	const [isWarehouseModeLoading, setIsWarehouseModeLoading] = useState<boolean>(false);

	useEffect(() => {
		if (isWarehouseModeLoading) {
			setTimeout(() => {
				setIsWarehouseModeLoading(false);
			}, 300);
		}
	}, [warehouseMode]);

	const warehouse = useMemo(() => {
		return warehouses.find((w) => w.name === warehouseName)!;
	}, [warehouses, warehouseName]);

	const onChangeSettings = () => {
		setSelectedWarehouse(warehouse);
	};

	const rows: GridRow[] = [];
	for (const k in warehouseSettings) {
		let value: ReactNode;
		if (k === 'Password') {
			value = <PasswordToggle password={warehouseSettings[k]} />;
		} else {
			value = warehouseSettings[k];
		}
		const row: GridRow = { cells: [<span style={{ fontWeight: '600' }}>{k}</span>, value] };
		rows.push(row);
	}

	const mcpRows: GridRow[] = [];
	for (const k in warehouseMCPSettings) {
		let value: ReactNode;
		if (k === 'Password') {
			value = <PasswordToggle password={warehouseMCPSettings[k]} />;
		} else {
			value = warehouseMCPSettings[k];
		}
		const row: GridRow = { cells: [<span style={{ fontWeight: '600' }}>{k}</span>, value] };
		mcpRows.push(row);
	}

	return (
		<div className='warehouse-info'>
			<div className='warehouse-info__info'>
				<div className='warehouse-info__title'>
					<div className='warehouse-info__icon'>
						<LittleLogo code={warehouse.code} path={WAREHOUSES_ASSETS_PATH} />
					</div>
					<div className='warehouse-info__name'>{warehouse.name}</div>
				</div>
				<SlButton
					className='warehouse-info__change-settings-button'
					variant='primary'
					onClick={onChangeSettings}
				>
					Modify...
				</SlButton>
			</div>
			<Section
				title={warehouseSectionTexts.mode.title}
				description={warehouseSectionTexts.mode.description}
				padded={true}
				annotated={true}
			>
				{warehouseMode} - {warehouseModeDescriptions[warehouseMode]}
			</Section>
			<Section
				title={warehouseSectionTexts.main.title}
				description={warehouseSectionTexts.main.description}
				annotated={true}
			>
				<Grid rows={rows} columns={warehouseInfoColumns} />
			</Section>
			{warehouseName !== 'Snowflake' && (
				<Section
					title={warehouseSectionTexts.mcp.title}
					description={warehouseSectionTexts.mcp.description}
					padded={mcpRows.length === 0}
					annotated={true}
				>
					{mcpRows.length > 0 ? (
						<div className='warehouse-info__settings'>
							<Grid rows={mcpRows} columns={warehouseInfoColumns} />
						</div>
					) : (
						<div className='warehouse-info__mcp-not-configured'>
							No credentials have been set, so the MCP server has no access to the data warehouse.
						</div>
					)}
				</Section>
			)}
		</div>
	);
};

export default DataWarehouse;
export { warehouseSectionTexts };

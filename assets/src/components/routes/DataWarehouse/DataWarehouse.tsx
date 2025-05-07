import React, { useState, useContext, useEffect, ReactNode, useMemo } from 'react';
import './DataWarehouse.css';
import appContext from '../../../context/AppContext';
import { Warehouse, warehouses } from './DataWarehouse.helpers';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import PasswordToggle from '../../base/PasswordToggle/PasswordToggle';
import { WarehouseMode, WarehouseSettings } from '../../../lib/api/types/warehouse';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import DataWarehouseSettings from './DataWarehouseSettings';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';

const DataWarehouse = () => {
	const [connectedWarehouse, setConnectedWarehouse] = useState<string>();
	const [selectedWarehouse, setSelectedWarehouse] = useState<Warehouse>();
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [hasError, setHasError] = useState<boolean>();

	const { api, handleError, selectedWorkspace, workspaces } = useContext(appContext);

	useEffect(() => {
		const fetchWarehouse = async () => {
			setHasError(false);
			try {
				const response = await api.workspaces.warehouse();
				setConnectedWarehouse(response.name);
				setWarehouseSettings(response.settings);
			} catch (err) {
				setTimeout(() => {
					setIsLoading(false);
				}, 300);
				setHasError(true);
				handleError(err);
				return;
			}
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		fetchWarehouse();
	}, [selectedWorkspace, selectedWarehouse]);

	// TODO: handle unexpected errors.
	if (hasError) {
		return null;
	}

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
				/>
			) : (
				<WarehouseInfo
					warehouseName={connectedWarehouse}
					warehouseSettings={warehouseSettings!}
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
	warehouseMode: WarehouseMode;
	setSelectedWarehouse: React.Dispatch<React.SetStateAction<Warehouse | undefined>>;
}

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
	warehouseMode,
	setSelectedWarehouse,
}: WarehouseInfoProps) => {
	const [isWarehouseModeLoading, setIsWarehouseModeLoading] = useState<boolean>(false);
	const [warehouseModeToSet, setWarehouseModeToSet] = useState<WarehouseMode | ''>('');

	const { api, handleError, setIsLoadingWorkspaces } = useContext(appContext);

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

	const onChange = () => {
		setSelectedWarehouse(warehouse);
	};

	const onChangeMode = async (e) => {
		setWarehouseModeToSet(e.target.value);
	};

	const onConfirmChangeMode = async () => {
		setIsWarehouseModeLoading(true);
		try {
			await api.workspaces.updateWarehouseMode(warehouseModeToSet as WarehouseMode, false);
		} catch (err) {
			setIsWarehouseModeLoading(false);
			handleError(err);
			return;
		}
		setIsLoadingWorkspaces(true);
		setWarehouseModeToSet('');
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

	return (
		<div className='warehouse-info'>
			<div className='warehouse-info__info'>
				<div className='warehouse-info__title'>
					<div className='warehouse-info__icon'>
						<LittleLogo icon={warehouse.icon} />
					</div>
					<div className='warehouse-info__name'>{warehouse.name}</div>
				</div>
				<div className='warehouse-info__mode-init'>
					<SlSelect
						className='warehouse-info__mode'
						value={isWarehouseModeLoading ? '' : warehouseMode}
						onSlChange={onChangeMode}
						disabled={isWarehouseModeLoading}
					>
						{isWarehouseModeLoading && <SlSpinner slot='prefix'></SlSpinner>}
						<SlOption value='Normal'>
							<div className='warehouse-info__mode-title'>Normal</div>
							<div className='warehouse-info__mode-description'>
								{' '}
								<span className='warehouse-info__mode-separator'>-</span> Full read and write access
							</div>
						</SlOption>
						<SlOption value='Inspection'>
							<div className='warehouse-info__mode-title'>Inspection</div>
							<div className='warehouse-info__mode-description'>
								{' '}
								<span className='warehouse-info__mode-separator'>-</span> Read-only for data inspection
							</div>
						</SlOption>
						<SlOption value='Maintenance'>
							<div className='warehouse-info__mode-title'>Maintenance</div>
							<div className='warehouse-info__mode-description'>
								{' '}
								<span className='warehouse-info__mode-separator'>-</span> Init and alter schema
								operations only
							</div>
						</SlOption>
					</SlSelect>
				</div>
			</div>
			<div className='warehouse-info__settings'>
				<Grid rows={rows} columns={warehouseInfoColumns} />
			</div>
			<div className='warehouse-info__buttons'>
				<SlButton variant='default' onClick={onChange}>
					Change settings...
				</SlButton>
			</div>
			<AlertDialog
				variant='danger'
				isOpen={warehouseModeToSet !== ''}
				onClose={() => setWarehouseModeToSet('')}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={() => setWarehouseModeToSet('')} disabled={isWarehouseModeLoading}>
							Cancel
						</SlButton>
						<SlButton variant='primary' onClick={onConfirmChangeMode} loading={isWarehouseModeLoading}>
							Put in {warehouseModeToSet} mode
						</SlButton>
					</>
				}
			>
				<p>
					Changing the warehouse mode alters the types of operations that can be performed on the data
					warehouse
				</p>
			</AlertDialog>
		</div>
	);
};

export default DataWarehouse;

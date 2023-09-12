import React, { useState, useContext, useEffect, ReactNode } from 'react';
import './DataWarehouse.css';
import appContext from '../../../context/AppContext';
import { Warehouse, warehouses } from './DataWarehouse.helpers';
import ListTile from '../../shared/ListTile/ListTile';
import LittleLogo from '../../shared/LittleLogo/LittleLogo';
import PasswordToggle from '../../shared/PasswordToggle/PasswordToggle';
import { WarehouseSettings, WarehouseType } from '../../../types/external/warehouse';
import Grid from '../../shared/Grid/Grid';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import DataWarehouseSettings from './DataWarehouseSettings';
import { SlButton, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const DataWarehouse = () => {
	const [connectedWarehouse, setConnectedWarehouse] = useState<WarehouseType>();
	const [selectedWarehouse, setSelectedWarehouse] = useState<Warehouse>();
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [hasError, setHasError] = useState<boolean>();

	const { setTitle, api, showError } = useContext(appContext);

	setTitle('Data Warehouse');

	useEffect(() => {
		const fetchWarehouse = async () => {
			setHasError(false);
			try {
				const response = await api.workspace.warehouseSettings();
				setConnectedWarehouse(response.type);
				setWarehouseSettings(response.settings);
			} catch (err) {
				if (err.code === 'NotConnected') {
					return;
				}
				setHasError(true);
				showError(err);
			}
		};
		fetchWarehouse();
	}, [selectedWarehouse]);

	// TODO: handle unexpected errors.
	if (hasError) {
		return null;
	}

	return (
		<div className='data-warehouse'>
			{selectedWarehouse ? (
				<DataWarehouseSettings
					selectedWarehouse={selectedWarehouse}
					setSelectedWarehouse={setSelectedWarehouse}
					currentSettings={warehouseSettings}
				/>
			) : connectedWarehouse ? (
				<WarehouseInfo
					warehouseName={connectedWarehouse}
					warehouseSettings={warehouseSettings!}
					setConnectedWarehouse={setConnectedWarehouse}
					setSelectedWarehouse={setSelectedWarehouse}
					setWarehouseSettings={setWarehouseSettings}
				/>
			) : (
				<WarehouseList setSelectedWarehouse={setSelectedWarehouse} />
			)}
		</div>
	);
};

interface WarehouseInfoProps {
	warehouseName: string;
	warehouseSettings: WarehouseSettings;
	setConnectedWarehouse: React.Dispatch<React.SetStateAction<WarehouseType | undefined>>;
	setSelectedWarehouse: React.Dispatch<React.SetStateAction<Warehouse | undefined>>;
	setWarehouseSettings: React.Dispatch<React.SetStateAction<WarehouseSettings | undefined>>;
}

const warehouseInfoColumns: GridColumn[] = [
	{
		name: 'field',
	},
	{
		name: 'value',
	},
];

const WarehouseInfo = ({
	warehouseName,
	warehouseSettings,
	setConnectedWarehouse,
	setSelectedWarehouse,
	setWarehouseSettings,
}: WarehouseInfoProps) => {
	const [isConfirmationDialogOpen, setIsConfirmationDialogOpen] = useState<boolean>(false);
	const [isDisconnectButtonLoading, setIsDisconnectButtonLoading] = useState<boolean>(false);

	const { api, showError } = useContext(appContext);

	const warehouse = warehouses.find((w) => w.label === warehouseName)!;

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

	const onDisconnect = async () => {
		setIsConfirmationDialogOpen(true);
	};

	const onChange = () => {
		setSelectedWarehouse(warehouse);
	};

	const onDisconnectConfirmation = async () => {
		setIsDisconnectButtonLoading(true);
		try {
			await api.workspace.disconnectWarehouse();
		} catch (err) {
			setIsDisconnectButtonLoading(false);
			showError(err);
			return;
		}
		setConnectedWarehouse(undefined);
		setWarehouseSettings(undefined);
		setIsConfirmationDialogOpen(false);
		setIsDisconnectButtonLoading(false);
	};

	const onCancelDisconnection = async () => {
		setIsConfirmationDialogOpen(false);
	};

	return (
		<div className='warehouse-info'>
			<div className='warehouse-info__info'>
				<div className='warehouse-info__icon'>
					<LittleLogo icon={warehouse.icon} />
				</div>
				<div className='warehouse-info__name'>{warehouse.label}</div>
			</div>
			<div className='warehouse-info__settings'>
				<Grid rows={rows} columns={warehouseInfoColumns} />
			</div>
			<div className='warehouse-info__buttons'>
				<SlButton onClick={onDisconnect} variant='danger'>
					Disconnect
				</SlButton>
				<SlButton variant='default' onClick={onChange}>
					Change settings...
				</SlButton>
			</div>
			<SlDialog open={isConfirmationDialogOpen} label='Are you sure?' onSlAfterHide={onCancelDisconnection}>
				<p className='warehouse-info__confirmation-text'>
					If you disconnect the warehouse, you will no longer be able to import users and events
				</p>
				<div className='warehouse-info__confirmation-buttons'>
					<SlButton onClick={onCancelDisconnection} disabled={isDisconnectButtonLoading}>
						Cancel
					</SlButton>
					<SlButton variant='danger' onClick={onDisconnectConfirmation} loading={isDisconnectButtonLoading}>
						Disconnect
					</SlButton>
				</div>
			</SlDialog>
		</div>
	);
};

interface WarehouseListProps {
	setSelectedWarehouse: React.Dispatch<React.SetStateAction<Warehouse | undefined>>;
}

const WarehouseList = ({ setSelectedWarehouse }: WarehouseListProps) => {
	const onWarehouseClick = (name: string) => {
		const warehouse = warehouses.find((w) => w.name === name)!;
		setSelectedWarehouse(warehouse);
	};

	return (
		<div className='warehouse-list'>
			<p className='warehouse-list__title'>Select a warehouse</p>
			<p className='warehouse-list__description'>
				You have not connected a warehouse yet. Select one of the following warehouse and configure it to start
				storing your users and events.
			</p>
			{warehouses.map((warehouse) => {
				return (
					<ListTile
						className='warehouse-list__warehouse'
						icon={<LittleLogo icon={warehouse.icon} />}
						name={warehouse.label}
						onClick={() => onWarehouseClick(warehouse.name)}
					/>
				);
			})}
		</div>
	);
};

export default DataWarehouse;

import React, { useState, useContext, useEffect, ReactNode } from 'react';
import './DataWarehouse.css';
import appContext from '../../../context/AppContext';
import { Warehouse, warehouses } from './DataWarehouse.helpers';
import ListTile from '../../base/ListTile/ListTile';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import PasswordToggle from '../../base/PasswordToggle/PasswordToggle';
import { WarehouseMode, WarehouseSettings, WarehouseType } from '../../../lib/api/types/warehouse';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import DataWarehouseSettings from './DataWarehouseSettings';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';

const DataWarehouse = () => {
	const [connectedWarehouse, setConnectedWarehouse] = useState<WarehouseType>();
	const [selectedWarehouse, setSelectedWarehouse] = useState<Warehouse>();
	const [warehouseSettings, setWarehouseSettings] = useState<WarehouseSettings>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [hasError, setHasError] = useState<boolean>();

	const { api, handleError, selectedWorkspace, workspaces } = useContext(appContext);

	useEffect(() => {
		const fetchWarehouse = async () => {
			setHasError(false);
			try {
				const response = await api.workspaces.warehouseSettings();
				setConnectedWarehouse(response.type);
				setWarehouseSettings(response.settings);
			} catch (err) {
				setTimeout(() => {
					setIsLoading(false);
				}, 300);
				if (err.code === 'NotConnected') {
					setConnectedWarehouse(undefined);
					return;
				}
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

	const warehouseMode = workspaces.find((w) => w.ID === selectedWorkspace).WarehouseMode;

	return (
		<div className='data-warehouse'>
			{selectedWarehouse ? (
				<DataWarehouseSettings
					selectedWarehouse={selectedWarehouse}
					setSelectedWarehouse={setSelectedWarehouse}
					currentMode={warehouseMode}
					currentSettings={warehouseSettings}
				/>
			) : connectedWarehouse ? (
				<WarehouseInfo
					warehouseName={connectedWarehouse}
					warehouseSettings={warehouseSettings!}
					warehouseMode={warehouseMode}
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
	warehouseMode: WarehouseMode;
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
	warehouseMode,
	setConnectedWarehouse,
	setSelectedWarehouse,
	setWarehouseSettings,
}: WarehouseInfoProps) => {
	const [isConfirmationDialogOpen, setIsConfirmationDialogOpen] = useState<boolean>(false);
	const [isDisconnectButtonLoading, setIsDisconnectButtonLoading] = useState<boolean>(false);
	const [isWarehouseModeLoading, setIsWarehouseModeLoading] = useState<boolean>(false);
	const [warehouseModeToSet, setWarehouseModeToSet] = useState<WarehouseMode | ''>('');

	const { api, handleError, setIsLoadingState, setIsLoadingWorkspaces } = useContext(appContext);

	useEffect(() => {
		if (isWarehouseModeLoading) {
			setTimeout(() => {
				setIsWarehouseModeLoading(false);
			}, 300);
		}
	}, [warehouseMode]);

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

	const onChangeMode = async (e) => {
		setWarehouseModeToSet(e.target.value);
	};

	const onConfirmChangeMode = async () => {
		setIsWarehouseModeLoading(true);
		try {
			await api.workspaces.changeWarehouseMode(warehouseModeToSet as WarehouseMode, false);
		} catch (err) {
			setIsWarehouseModeLoading(false);
			handleError(err);
			return;
		}
		setIsLoadingWorkspaces(true);
		setWarehouseModeToSet('');
	};

	const onDisconnectConfirmation = async () => {
		setIsDisconnectButtonLoading(true);
		try {
			await api.workspaces.disconnectWarehouse();
		} catch (err) {
			setIsDisconnectButtonLoading(false);
			handleError(err);
			return;
		}
		setConnectedWarehouse(undefined);
		setWarehouseSettings(undefined);
		setIsConfirmationDialogOpen(false);
		setIsDisconnectButtonLoading(false);
		setIsLoadingState(true);
	};

	const onCancelDisconnection = async () => {
		setIsConfirmationDialogOpen(false);
	};

	return (
		<div className='warehouse-info'>
			<div className='warehouse-info__info'>
				<div className='warehouse-info__title'>
					<div className='warehouse-info__icon'>
						<LittleLogo icon={warehouse.icon} />
					</div>
					<div className='warehouse-info__name'>{warehouse.label}</div>
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
				<SlButton onClick={onDisconnect} variant='danger'>
					Disconnect
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
			<AlertDialog
				variant='danger'
				isOpen={isConfirmationDialogOpen}
				onClose={onCancelDisconnection}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={onCancelDisconnection} disabled={isDisconnectButtonLoading}>
							Cancel
						</SlButton>
						<SlButton
							variant='danger'
							onClick={onDisconnectConfirmation}
							loading={isDisconnectButtonLoading}
						>
							Disconnect
						</SlButton>
					</>
				}
			>
				<p>If you disconnect the data warehouse, you will no longer be able to import users and events.</p>
				<br />
				<p>
					It is also important to note that a data warehouse should be disconnected only when there are no
					operations currently running on it. Therefore, it is advised to disconnect it only when it is in
					maintenance mode.
				</p>
			</AlertDialog>
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
			<p className='warehouse-list__title'>Select a data warehouse</p>
			<p className='warehouse-list__description'>
				You have not connected a data warehouse yet. Select one of the following data warehouses and configure
				it to start storing your users and events.
			</p>
			{warehouses.map((warehouse) => {
				return (
					<ListTile
						key={warehouse.name}
						className='warehouse-list__warehouse'
						icon={<LittleLogo icon={warehouse.icon} />}
						name={warehouse.label}
						onClick={() => onWarehouseClick(warehouse.name)}
						action={<SlIcon name='chevron-right' />}
					/>
				);
			})}
		</div>
	);
};

export default DataWarehouse;

type WarehouseType = 'BigQuery' | 'ClickHouse' | 'PostgreSQL' | 'Redshift' | 'Snowflake';

type WarehouseMode = 'Normal' | 'Inspection' | 'Maintenance';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	type: WarehouseType;
	settings: WarehouseSettings;
}

type ConnectWarehouseBehavior = 'FailOnCheck' | 'InitializeWarehouse' | 'RepairWarehouse';

export type { ConnectWarehouseBehavior, WarehouseType, WarehouseMode, WarehouseSettings, WarehouseResponse };

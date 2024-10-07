type WarehouseType = 'PostgreSQL' | 'Snowflake';

type WarehouseMode = 'Normal' | 'Inspection' | 'Maintenance';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	type: WarehouseType;
	settings: WarehouseSettings;
}

export type { WarehouseType, WarehouseMode, WarehouseSettings, WarehouseResponse };

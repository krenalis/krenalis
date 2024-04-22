type WarehouseType = 'BigQuery' | 'ClickHouse' | 'PostgreSQL' | 'Redshift' | 'Snowflake';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	type: WarehouseType;
	settings: WarehouseSettings;
}

export type { WarehouseType, WarehouseSettings, WarehouseResponse };

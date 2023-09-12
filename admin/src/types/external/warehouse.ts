type WarehouseType = 'BigQuery' | 'ClickHouse' | 'PostgreSQL' | 'Redshift' | 'Snowflake';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	type: WarehouseType;
	settings: WarehouseSettings;
}

export { WarehouseType, WarehouseSettings, WarehouseResponse };

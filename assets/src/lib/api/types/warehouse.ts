type WarehouseMode = 'Normal' | 'Inspection' | 'Maintenance';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	name: string;
	settings: WarehouseSettings;
}

export type { WarehouseMode, WarehouseSettings, WarehouseResponse };

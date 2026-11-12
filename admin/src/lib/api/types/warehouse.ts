type WarehouseMode = 'Normal' | 'Inspection' | 'Maintenance';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	platform: string;
	settings: WarehouseSettings;
}

export type { WarehouseMode, WarehouseSettings, WarehouseResponse };

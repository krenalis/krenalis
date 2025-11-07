type WarehouseMode = 'Normal' | 'Inspection' | 'Maintenance';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	name: string;
	settings: WarehouseSettings;
	mcpSettings: WarehouseSettings;
}

export type { WarehouseMode, WarehouseSettings, WarehouseResponse };

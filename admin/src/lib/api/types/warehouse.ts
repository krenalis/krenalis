type WarehouseMode = 'Normal' | 'Inspection' | 'Maintenance';

type WarehouseSettings = Record<string, any>;

interface WarehouseResponse {
	platform: string;
	settings: WarehouseSettings;
	mcpSettings: WarehouseSettings;
}

export type { WarehouseMode, WarehouseSettings, WarehouseResponse };

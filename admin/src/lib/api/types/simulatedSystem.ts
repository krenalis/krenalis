type SimulatedSystemStatus = 'Preparing' | 'Available' | 'Failed' | 'Deleting';

interface SimulatedSystem {
	id: string;
	workspace: string;
	name: string;
	status: SimulatedSystemStatus;
	userCount: number;
	generatedRecordCount: number;
	sourceConnection: string | null;
	destinationConnection: string | null;
	duplicateRecordPercent: number;
	countries: Record<string, number>;
	createdAt: string;
	updatedAt: string;
}

interface SimulatedSystemToCreate {
	name: string;
	userCount: number;
	duplicateRecordPercent: number;
	countries: Record<string, number>;
}

export type { SimulatedSystem, SimulatedSystemToCreate };

type SimulatedAccountStatus = 'Preparing' | 'Ready' | 'Failed';

interface SimulatedAccount {
	id: string;
	workspace: string;
	name: string;
	status: SimulatedAccountStatus;
	userCount: number;
	duplicateRecordPercent: number;
	countries: Record<string, number>;
	generatedRecordCount: number;
	generationError: string;
	createdAt: string;
	updatedAt: string;
}

interface SimulatedAccountToCreate {
	name: string;
	userCount: number;
	duplicateRecordPercent: number;
	countries: Record<string, number>;
}

export type { SimulatedAccount, SimulatedAccountToCreate };

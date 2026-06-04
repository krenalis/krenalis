type AccessKeyType = 'API' | 'MCP';

interface AccessKey {
	id: string;
	workspace: string | null;
	name: string;
	type: AccessKeyType;
	token: string;
	createdAt: string;
}

export { AccessKey, AccessKeyType };

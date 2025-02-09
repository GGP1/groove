export type Zone = {
	name: string,
	required_permission_keys: string[],
}

export type UpdateZone = {
	name?: string,
	required_permission_keys?: string[]
}

export type AccessZoneResp = {
	name?: string,
	access?: boolean
}

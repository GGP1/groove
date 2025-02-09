export type Role = {
	name: string,
	permission_keys: string[]
}

export type CloneRole = {
	exporter_event_id: string
}

export type SetRoles = {
	users_ids: string[],
	role_name: string
}

export type UpdateRole = {
	name?: string,
	permission_keys?: string[]
}

export type Permission = {
	name: string,
	key: string,
	description?: string,
	created_at?: Date,
}

export type ClonePermission = {
	exporter_event_id: string
}

export type UpdatePermission = {
	name?: string,
	description?: string,
	key?: string,
}


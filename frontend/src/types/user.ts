export type User = {
	id: string,
	name: string,
	username: string,
	email: string,
	birth_date: Date,
	description?: string,
	profile_image_url?: string,
	private: boolean,
	type: Type,
	invitations: Invitations,
	created_at: Date,
	updated_at?: Date,
}

export type CreateUser = {
	name: string,
	username: string,
	password: string,
	email: string,
	birth_date: Date,
	description?: string,
	type: Type,
	profile_image_url?: string
}

export type UpdateUser = {
	name?: string,
	username?: string,
	private?: boolean,
	invitations?: Invitations
}

export type UserStatistics = {
	blocked_count: number,
	blocked_by_count: number,
	friends_count?: number, // Personal users only
	following_count?: number // Business users only
	followers_count?: number, // Business users only
	attending_events_count: number,
	hosted_events_count: number,
	invitations_count: number,
	liked_events_count: number,
}

export type Invite = {
	event_id: string,
	user_ids: string[],
}

export enum Invitations {
	Nobody = 1,
	Friend,
}

export enum Type {
	Personal = 1,
	Business,
}

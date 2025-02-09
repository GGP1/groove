export type Event = {
	id: string,
	name: string,
	description?: string,
	type: Type,
	ticket_type: TicketType,
	public: boolean,
	virtual: boolean,
	hide_roles: boolean,
	logo_url?: string,
	header_url?: string,
	url?: string,
	cron: string,
	start_date: Date,
	end_date: Date,
	min_age: number,
	slots: number,
	location: {
		address: string,
		coordinates: Coordinates,
	},
	created_at: Date,
	updated_at?: Date
}

export type CreateEvent = {
	host_id: string,
	name: string,
	description?: string,
	type: Type,
	ticket_type: TicketType,
	public: boolean,
	virtual: boolean,
	hide_roles: boolean,
	logo_url?: string,
	header_url?: string,
	url?: string,
	location?: {
		address?: string,
		coordinates?: Coordinates,
	},
	cron: string,
	start_date: Date,
	end_date: Date,
	min_age: number,
	slots: number
}

export type UpdateEvent = {
	name?: string,
	description?: string,
	type?: Type,
	hide_roles?: boolean,
	logo_url?: string,
	header_url?: string,
	location?: {
		address?: string,
		coordinates?: Coordinates
	},
	cron?: string,
	start_date?: Date,
	end_date?: Date,
	min_age?: number,
	slots?: number
}

export type Coordinates = {
	latitude: number,
	longitude: number,
}

export type LocationSearch = {
	latitude: number,
	longitude: number,
	latitude_delta: number,
	longitude_delta: number,
}

export type EventStatistics = {
	banned_count: number,
	members_count: number,
	invited_count: number,
	likes_count: number
}

export enum Type {
	Meeting = 1,
	Party,
	Conference,
	Talk,
	Show,
	Class,
	Birthday,
	Reunion,
	Match,
	League,
	Tournament,
	Trip,
	Protest,
	GrandPrix,
	Marriage,
	Concert,
	Marathon,
	Hackathon,
	Ceremony,
	Graduation,
	Tribute,
	Anniversary
}

export function typeToString(type: Type | undefined): string | undefined {
	if (type) {
		if (type === Type.GrandPrix) {
			return "Grand Prix";
		}
		// Generic alternative: return Type[type].replace(/([a-z])([A-Z])/g, "$1 $2");
		return Type[type];
	}

	return undefined;
}

export enum TicketType {
	Free = 1,
	Paid,
	Mixed,
	Donation
}

export function ticketTypeToString(type: TicketType | undefined): string | undefined {
	return type ? TicketType[type] : undefined;
}

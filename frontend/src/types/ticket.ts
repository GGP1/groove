export type Ticket = {
	name: string,
	description?: string,
	available_count: number,
	cost: number,
	linked_role: string
}

export type BuyTicket = {
	user_ids: string[]
}

export type UpdateTicket = {
	name?: string,
	description?: string,
	available_count?: number,
	cost?: number,
	linked_role?: string
}


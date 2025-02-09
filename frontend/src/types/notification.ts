export interface Notification {
	id: string,
	sender_id: string,
	receiver_id: string,
	event_id?: string,
	content: string,
	type: Type,
	seen: boolean,
	created_at: Date,
}

export enum Type {
	Invitation = 1,
	FriendRequest,
	Proposal
}

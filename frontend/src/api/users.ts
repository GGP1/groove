import {
	BannedEventsCount, BlockedByCount, BlockedCount,
	BlockedIDBody, EventsPag, FollowersCount, FollowingCount, FriendIDBody,
	FriendsCount, FriendsInCommonCount, FriendsNotInCommonCount,
	// eslint-disable-next-line comma-dangle
	HostedEventsCount, IDResponse, InvitedEventsCount, LikedEventsCount, UserIDBody, UsersPag
} from "../types/api";
import { Event } from "../types/event";
import { CreateUser, Invite, UpdateUser, User, UserStatistics } from "../types/user";
import { HTTP } from "../utils/http";
import { API_URL, buildURL, Fields, Params } from "./api";
import { APIKeyResponse } from "./key";

export class UsersEndpoints {
	/**
	 * Create creates a new user.
	 * @param body user
	 * @returns user ID and API key
	 */
	async Create(body: CreateUser): Promise<APIKeyResponse> {
		body.name = body.name.trim();
		body.email = body.email.trim();
		body.username = body.username.trim();
		body.password = body.password.trim();

		const resp = await HTTP.post({
			url: `${API_URL}/create/user`,
			body: JSON.stringify(body),
		});
		const respBody = await resp.json() as APIKeyResponse;
		return respBody;
	}

	/**
	 * Search users using Full Text Search.
	 * @param query search query
	 * @param params request parameters
	 * @returns list of users and a the next cursor
	 */
	async Search(query: string, params?: Params<User>): Promise<UsersPag> {
		const searchResp = await HTTP.get<UsersPag>({
			url: buildURL(`${API_URL}/search/users?query=${query}`, Fields.user, params),
		});
		return searchResp;
	}

	/**
	 * GetByID returns a user with the ID provided.
	 * @param id user ID
	 * @returns a user
	 */
	async GetByID(id: string): Promise<User> {
		const user = await HTTP.get<User>({ url: `${API_URL}/users/${id}` });
		return user;
	}

	/**
	 * GetStatistics returns a user's statistics.
	 * @param id user ID
	 * @returns user statistics
	 */
	async GetStatistics(id: string): Promise<UserStatistics> {
		const stats = await HTTP.get<UserStatistics>({ url: `${API_URL}/users/${id}/stats` });
		return stats;
	}

	/**
	 * InviteToEvent invites a user to an event.
	 * @param senderID authenticated user ID
	 * @param invite invitation details
	 * @returns no content
	 */
	async InviteToEvent(senderID: string, invite: Invite): Promise<Response> {
		const resp = await HTTP.post({
			url: `${API_URL}/users/${senderID}/invite`,
			body: JSON.stringify(invite),
		});
		return resp;
	}

	/**
	 * Block blocks another user.
	 * @param id authenticated user ID
	 * @param body block details
	 */
	async Block(id: string, body: BlockedIDBody) {
		await HTTP.post({
			url: `${API_URL}/users/${id}/block`,
			body: JSON.stringify(body),
		});
	}

	/**
	 * Follow follows a business.
	 * @param id user ID
	 * @param bussiness_id business ID
	 */
	async Follow(id: string, bussiness_id: string) {
		await HTTP.post({ url: `${API_URL}/users/${id}/follow/${bussiness_id}` });
	}

	/**
	 * GetBlocked returns the list of users blocked by the auth user or a count of them.
	 * @param id authenticated user ID
	 * @param params request parameters
	 * @returns a list of users or a count
	 */
	async GetBlocked<T extends UsersPag | BlockedCount>(id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/blocked`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * GetBlockedBy returns the users that blocked the auth user or a count of them.
	 * @param id authenticated user ID
	 * @param params request parameters
	 * @returns a list of users or a count
	 */
	async GetBlockedBy<T extends UsersPag | BlockedByCount>(id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/blocked_by`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * Delete removes a user from the system.
	 * @param id user ID
	 */
	async Delete(id: string) {
		await HTTP.delete({ url: `${API_URL}/users/${id}/delete` });
	}

	/**
	 * Unblock removes a block on a user.
	 * @param id authenticated user ID
	 * @param body block details
	 */
	async Unblock(id: string, body: BlockedIDBody) {
		await HTTP.post({
			url: `${API_URL}/users/${id}/unblock`,
			body: JSON.stringify(body),
		});
	}

	/**
	 * Unfollow unfollows a bussiness.
	 * @param id user ID
	 * @param bussiness_id business ID
	 */
	async Unfollow(id: string, bussiness_id: string) {
		await HTTP.post({ url: `${API_URL}/users/${id}/unfollow/${bussiness_id}` });
	}


	/**
	 * Update updates a user.
	 * @param id user ID
	 * @param body updated information
	 */
	async Update(id: string, body: UpdateUser): Promise<IDResponse> {
		const resp = await HTTP.put({
			url: `${API_URL}/users/${id}/update`,
			body: JSON.stringify(body),
		});
		return await resp.json() as IDResponse;
	}

	// Events
	/**
	 * GetAttendingEvents returns
	 * @param id user ID
	 * @param params request parameters
	 * @returns a list of events or the number of them
	 */
	async GetAttendingEvents<T extends EventsPag | HostedEventsCount>(id: string, params?: Params<Event>): Promise<T> {
		const uri = `${API_URL}/users/${id}/events/attending`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.event, params),
		});
		return resp;
	}

	/**
	 * GetBannedEvents returns the events the user passed is banned from.
	 * @param id user ID
	 * @param params request parameters
	 * @returns a list of events or the number of them
	 */
	async GetBannedEvents<T extends EventsPag | BannedEventsCount>(id: string, params?: Params<Event>): Promise<T> {
		const uri = `${API_URL}/users/${id}/events/banned`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.event, params),
		});
		return resp;
	}

	/**
	 * GetHostedEvents returns the events hosted by the user passed.
	 * @param id user ID
	 * @param params request parameters
	 * @returns a list of events or the number of them
	 */
	async GetHostedEvents<T extends EventsPag | HostedEventsCount>(id: string, params?: Params<Event>): Promise<T> {
		const uri = `${API_URL}/users/${id}/events/hosted`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.event, params),
		});
		return resp;
	}

	/**
	 * GetInvitedEvents returns the events the user passed is invited to.
	 * @param id user ID
	 * @param params request parameters
	 * @returns a list of events or the number of them
	 */
	async GetInvitedEvents<T extends EventsPag | InvitedEventsCount>(id: string, params?: Params<Event>): Promise<T> {
		const uri = `${API_URL}/users/${id}/events/invited`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.event, params),
		});
		return resp;
	}

	/**
	 * GetLikedEvents returns the events liked by the user passed.
	 * @param id user ID
	 * @param params request parameters
	 * @returns a list of events or the number of them
	 */
	async GetLikedEvents<T extends EventsPag | LikedEventsCount>(id: string, params?: Params<Event>): Promise<T> {
		const uri = `${API_URL}/users/${id}/events/liked`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.event, params),
		});
		return resp;
	}

	// Friends/Followers

	/**
	 * GetFollowers returns an organization's followers.
	 * @param id user ID
	 * @param params requrest parameters
	 * @returns a list of users or the number of them
	 */
	async GetFollowers<T extends UsersPag | FollowersCount>(id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/followers`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * GetFollowing returns the organization a user is following.
	 * @param id user ID
	 * @param params requrest parameters
	 * @returns a list of users or the number of them
	 */
	async GetFollowing<T extends UsersPag | FollowingCount>(id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/following`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * GetFriends returns a user's friends.
	 * @param id user ID
	 * @param params request parameters
	 * @returns a list of users or the number of them
	 */
	async GetFriends<T extends UsersPag | FriendsCount>(id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/friends`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * GetFriendsInCommon returns the friends in common between the authenticated user and another one.
	 * @param id authenticated user ID
	 * @param friend_id friend ID
	 * @param params request parameters
	 * @returns a list of users or a the number of them
	 */
	async GetFriendsInCommon<T extends UsersPag | FriendsInCommonCount>(id: string, friend_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/friends/common/${friend_id}`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * GetFriendsNotInCommon returns the friends not in common between the authenticated user and another one.
	 * @param id authenticated user ID
	 * @param friend_id friend ID
	 * @param params request parameters
	 * @returns a list of users or a the number of them
	 */
	async GetFriendsNotInCommon<T extends UsersPag | FriendsNotInCommonCount>(id: string, friend_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/users/${id}/friends/notcommon/${friend_id}`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	/**
	 * RemoveFriend removes a friend.
	 * @param id authenticated user ID
	 * @param body friendship details
	 */
	async RemoveFriend(id: string, body: FriendIDBody) {
		await HTTP.post({
			url: `${API_URL}/users/${id}/friends/remove`,
			body: JSON.stringify(body),
		});
	}

	async SendFriendRequest(id: string, body: UserIDBody) {
		await HTTP.post({
			url: `${API_URL}/users/${id}/friends/request`,
			body: JSON.stringify(body),
		});
	}
}

import {
	AvailableTickets, BannedCount, BannedFriendsCount,
	CommentLikes, CommentsPag, EventsPag,
	IDResponse,
	InvitedCount, InvitedFriendsCount, LikedByFriendsCount,
	LikesCount, MembersCount, MembersFriendsCount, NameResponse, PostLikes,
	// eslint-disable-next-line comma-dangle
	PostsPag, ProductsPag, UserIDBody, UsersPag
} from "../types/api";
import { Comment, CreateComment } from "../types/comment";
import { Coordinates, CreateEvent, Event, EventStatistics, LocationSearch, UpdateEvent } from "../types/event";
import { ClonePermission, Permission, UpdatePermission } from "../types/permission";
import { CreatePost, Post, UpdatePost } from "../types/post";
import { Product, UpdateProduct } from "../types/product";
import { CloneRole, Role, SetRoles, UpdateRole } from "../types/role";
import { BuyTicket, Ticket, UpdateTicket } from "../types/ticket";
import { User } from "../types/user";
import { AccessZoneResp, UpdateZone, Zone } from "../types/zone";
import { HTTP } from "../utils/http";
import { API_URL, buildURL, Fields, Params } from "./api";

export class EventsEndpoints {
	/**
	 * Create creates an event.
	 * @param body event input data
	 * @returns resource location url
	 */
	async Create(body: CreateEvent): Promise<IDResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/create/event`,
			body: JSON.stringify(body),
		});
		return await resp.json() as IDResponse;
	}

	async GetRecommended(body: Coordinates, params?: Params<Event>): Promise<EventsPag> {
		const resp = await HTTP.post({
			url: buildURL(`${API_URL}/recommended/events`, Fields.event, params),
			body: JSON.stringify(body),
		});

		const events = await resp.json() as EventsPag;
		return events;
	}

	async Search(query: string, params?: Params<Event>): Promise<EventsPag> {
		const uri = `${API_URL}/search/events?query=${query}`;
		const events = await HTTP.get<EventsPag>({
			url: buildURL(uri, Fields.event, params),
		});
		return events;
	}

	async SearchLocation(body: LocationSearch): Promise<Event[]> {
		const resp = await HTTP.post({
			url: `${API_URL}/search/events/location`,
			body: JSON.stringify(body),
		});
		const events = await resp.json() as Event[];
		return events;
	}

	async Delete(id: string) {
		await HTTP.delete({ url: `${API_URL}/events/${id}/delete` });
	}

	async GetByID(id: string): Promise<Event> {
		const event = await HTTP.get<Event>({
			url: buildURL(`${API_URL}/events/${id}`, Fields.event),
		});
		return event;
	}

	async GetHosts(id: string, params?: Params<User>): Promise<UsersPag> {
		const uri = `${API_URL}/events/${id}/hosts`;
		const resp = await HTTP.get<UsersPag>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async GetStatistics(id: string): Promise<EventStatistics> {
		const stats = await HTTP.get<EventStatistics>({ url: `${API_URL}/events/${id}/stats` });
		return stats;
	}

	async Join(id: string) {
		await HTTP.post({ url: `${API_URL}/events/${id}/join` });
	}

	async Leave(id: string) {
		await HTTP.post({ url: `${API_URL}/events/${id}/leave` });
	}

	async Update(id: string, body: UpdateEvent): Promise<IDResponse> {
		const resp = await HTTP.put({
			url: `${API_URL}/events/${id}/update`,
			body: JSON.stringify(body),
		});
		return await resp.json() as IDResponse;
	}

	// Bans
	async GetBans<T extends UsersPag | BannedCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/bans`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async Ban(event_id: string, body: UserIDBody) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/bans/ban`,
			body: JSON.stringify(body),
		});
	}

	async GetBannedFriends<T extends UsersPag | BannedFriendsCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/bans/friends`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async RemoveBanned(event_id: string, body: UserIDBody) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/bans/remove`,
			body: JSON.stringify(body),
		});
	}

	// Invited
	async GetInvited<T extends UsersPag | InvitedCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/invited`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async GetInvitedFriends<T extends UsersPag | InvitedFriendsCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/invited/friends`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async RemoveInvited(event_id: string, body: UserIDBody) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/invited/remove`,
			body: JSON.stringify(body),
		});
	}

	// Likes
	async GetLikes<T extends UsersPag | LikesCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/likes`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async GetLikedByFriends<T extends UsersPag | LikedByFriendsCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/likes/friends`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async Like(event_id: string, body: UserIDBody) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/likes/like`,
			body: JSON.stringify(body),
		});
	}

	async RemoveLike(event_id: string, body: UserIDBody) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/likes/remove`,
			body: JSON.stringify(body),
		});
	}

	// Posts
	async GetHomePosts(params?: Params<Post>): Promise<PostsPag> {
		const uri = `${API_URL}/home/posts`;
		const resp = await HTTP.get<PostsPag>({
			url: buildURL(uri, Fields.posts, params),
		});
		return resp;
	}

	async GetPosts(event_id: string, params?: Params<Post>): Promise<PostsPag> {
		const uri = `${API_URL}/events/${event_id}/posts`;
		const resp = await HTTP.get<PostsPag>({
			url: buildURL(uri, Fields.posts, params),
		});
		return resp;
	}

	async GetPost(event_id: string, post_id: string): Promise<Post> {
		const resp = await HTTP.get<Post>({
			url: `${API_URL}/events/${event_id}/posts/${post_id}`,
		});
		return resp;
	}

	async GetPostLikes<T extends UsersPag | PostLikes>(event_id: string, post_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/posts/${post_id}/likes`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async LikePost(event_id: string, post_id: string) {
		await HTTP.get({
			url: `${API_URL}/events/${event_id}/posts/${post_id}/like`,
		});
	}

	async CreatePost(event_id: string, body: CreatePost): Promise<IDResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/posts/create`,
			body: JSON.stringify(body),
		});
		return await resp.json() as IDResponse;
	}

	async DeletePost(event_id: string, post_id: string) {
		await HTTP.delete({ url: `${API_URL}/events/${event_id}/posts/delete/${post_id}` });
	}

	async UpdatePost(event_id: string, post_id: string, body: UpdatePost): Promise<IDResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/posts/update/${post_id}`,
			body: JSON.stringify(body),
		});
		return await resp.json() as IDResponse;
	}

	// Comments
	async GetComment(event_id: string, comment_id: string): Promise<Comment> {
		const resp = await HTTP.get<Comment>({
			url: `${API_URL}/events/${event_id}/comments/${comment_id}`,
		});
		return resp;
	}

	async LikeComment(event_id: string, comment_id: string) {
		await HTTP.get({
			url: `${API_URL}/events/${event_id}/comments/${comment_id}/like`,
		});
	}

	async GetCommentLikes<T extends UsersPag | CommentLikes>(event_id: string, comment_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/comments/${comment_id}/likes`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async CreateComment(event_id: string, body: CreateComment): Promise<IDResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/comments/create`,
			body: JSON.stringify(body),
		});
		return await resp.json() as IDResponse;
	}

	async DeleteComment(event_id: string, comment_id: string) {
		await HTTP.delete({ url: `${API_URL}/events/${event_id}/comments/delete/${comment_id}` });
	}

	async GetReplies(event_id: string, parent_id: string, params?: Params<Comment>): Promise<CommentsPag> {
		const uri = `${API_URL}/events/${event_id}/replies/${parent_id}`;
		const resp = await HTTP.get<CommentsPag>({
			url: buildURL(uri, Fields.comment, params),
		});
		return resp;
	}

	// Permissions
	async GetPermissions(event_id: string): Promise<Permission[]> {
		const permissions = await HTTP.get<Permission[]>({
			url: `${API_URL}/events/${event_id}/permissions`,
		}
		);
		return permissions;
	}

	async GetPermission(event_id: string, key: string): Promise<Permission> {
		const permission = await HTTP.get<Permission>({
			url: `${API_URL}/events/${event_id}/permissions/${key}`,
		}
		);
		return permission;
	}

	async ClonePermissions(event_id: string, body: ClonePermission) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/permissions/clone`,
			body: JSON.stringify(body),
		});
	}

	async CreatePermission(event_id: string, body: Permission): Promise<NameResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/permissions/create`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}

	async DeletePermission(event_id: string, key: string) {
		HTTP.delete({ url: `${API_URL}/events/${event_id}/permissions/delete/${key}` });
	}

	async UpdatePermission(event_id: string, key: string, body: UpdatePermission): Promise<NameResponse> {
		const resp = await HTTP.put({
			url: `${API_URL}/events/${event_id}/permissions/update/${key}`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}

	// Products
	async GetProducts(event_id: string, params?: Params<Product>): Promise<ProductsPag> {
		const uri = `${API_URL}/events/${event_id}/products`;
		const resp = await HTTP.get<ProductsPag>({
			url: buildURL(uri, Fields.product, params),
		});
		return resp;
	}

	async CreateProduct(event_id: string, body: Product) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/products/create`,
			body: JSON.stringify(body),
		});
	}

	async DeleteProduct(event_id: string, product_event_id: string) {
		await HTTP.delete({ url: `${API_URL}/events/${event_id}/products/delete/${product_event_id}` });
	}

	async UpdateProduct(event_id: string, product_event_id: string, body: UpdateProduct) {
		await HTTP.put({
			url: `${API_URL}/events/${event_id}/products/update/${product_event_id}`,
			body: JSON.stringify(body),
		});
	}

	// Roles
	async GetRoles(event_id: string): Promise<Role[]> {
		const roles = await HTTP.get<Role[]>({ url: `${API_URL}/events/${event_id}/roles` });
		return roles;
	}

	async GetRole(event_id: string, name: string): Promise<Role> {
		const role = await HTTP.get<Role>({ url: `${API_URL}/events/${event_id}/roles/${name}` });
		return role;
	}

	async Members<T extends UsersPag | MembersCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/roles/members`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async MembersFriends<T extends UsersPag | MembersFriendsCount>(event_id: string, params?: Params<User>): Promise<T> {
		const uri = `${API_URL}/events/${event_id}/roles/members/friends`;
		const resp = await HTTP.get<T>({
			url: buildURL(uri, Fields.user, params),
		});
		return resp;
	}

	async CloneRole(event_id: string, body: CloneRole) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/roles/clone`,
			body: JSON.stringify(body),
		});
	}

	async CreateRole(event_id: string, body: Role): Promise<NameResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/roles/create`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}

	async DeleteRole(event_id: string, name: string) {
		await HTTP.delete({ url: `${API_URL}/events/${event_id}/roles/delete/${name}` });
	}

	async SetRole(event_id: string, body: SetRoles) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/roles/set`,
			body: JSON.stringify(body),
		});
	}

	async GetUserRole(event_id: string, body: UserIDBody): Promise<Role> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/roles/user`,
			body: JSON.stringify(body),
		});
		const role = await resp.json() as Role;
		return role;
	}

	async UpdateRole(event_id: string, name: string, body: UpdateRole): Promise<NameResponse> {
		const resp = await HTTP.put({
			url: `${API_URL}/events/${event_id}/roles/update/${name}`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}

	// Tickets
	async GetTickets(event_id: string): Promise<Ticket[]> {
		const tickets = await HTTP.get<Ticket[]>({ url: `${API_URL}/events/${event_id}/tickets` });
		return tickets;
	}

	async GetAvailableTickets(event_id: string, name: string): Promise<AvailableTickets> {
		const availableTickets = await HTTP.get<AvailableTickets>({
			url: `${API_URL}/events/${event_id}/tickets/available/${name}`,
		});
		return availableTickets;
	}

	async BuyTicket(event_id: string, name: string, body: BuyTicket) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/tickets/buy/${name}`,
			body: JSON.stringify(body),
		});
	}

	async CreateTicket(event_id: string, body: Ticket): Promise<NameResponse> {
		const resp = await HTTP.post({
			url: `${API_URL}/events/${event_id}/tickets/create`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}

	async DeleteTicket(event_id: string, name: string) {
		await HTTP.delete({ url: `${API_URL}/events/${event_id}/tickets/delete/${name}` });
	}

	async RefundTicket(event_id: string, name: string) {
		await HTTP.get({ url: `${API_URL}/events/${event_id}/tickets/refund/${name}` });
	}

	async UpdateTicket(event_id: string, name: string, body: UpdateTicket): Promise<NameResponse> {
		const resp = await HTTP.put({
			url: `${API_URL}/events/${event_id}/tickets/update/${name}`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}

	// Zones
	async GetZones(event_id: string): Promise<Zone[]> {
		const zones = await HTTP.get<Zone[]>({ url: `${API_URL}/events/${event_id}/zones` });
		return zones;
	}

	async GetZone(event_id: string, zone_name: string): Promise<Zone> {
		const zone = await HTTP.get<Zone>({ url: `${API_URL}/events/${event_id}/zones/${zone_name}` });
		return zone;
	}

	async AccessZone(event_id: string, zone_name: string): Promise<AccessZoneResp> {
		const accessed = await HTTP.get<AccessZoneResp>({ url: `${API_URL}/events/${event_id}/zones/access/${zone_name}` });
		return accessed;
	}

	async CreateZone(event_id: string, body: Zone) {
		await HTTP.post({
			url: `${API_URL}/events/${event_id}/zones/create`,
			body: JSON.stringify(body),
		});
	}

	async DeleteZone(event_id: string, name: string) {
		await HTTP.delete({ url: `${API_URL}/events/${event_id}/zones/delete/${name}` });
	}

	/**
	 * UpdateZone updates an event's zone.
	 * @param event_id event ID
	 * @param zone_name zone name
	 * @param body zone updates
	 * @returns resource location url
	 */
	async UpdateZone(event_id: string, zone_name: string, body: UpdateZone): Promise<NameResponse> {
		const resp = await HTTP.put({
			url: `${API_URL}/events/${event_id}/zones/update/${zone_name}`,
			body: JSON.stringify(body),
		});
		return await resp.json() as NameResponse;
	}
}

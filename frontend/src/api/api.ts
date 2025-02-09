import { AuthEndpoints } from "./auth";
import { EventsEndpoints } from "./events";
import { NotificationsEndpoints } from "./notifications";
import { UsersEndpoints } from "./users";

// Temporary until a useful solution is found.
// react-native-dotenv does not update variables on change.
export const API_URL = "http://172.28.240.1:4000";

type key<T> = keyof T

export interface Params<T> {
	cursor?: string
	limit?: string
	count?: boolean
	fields?: key<T>[]
	lookup_id?: string
}

/** buildURL constructs a url given the parameters passed, if no params are provided it just returns the uri
 * @param uri endpoint url
 * @param fields fields search param name
 * @param params optional paramenters for constructing the url
 * @returns a url, i.e. https://groove.com/events/invited?limit=15
 */
export function buildURL<T>(uri: string, fields: Fields, params?: Params<T>): string {
	if (!params) {
		return uri;
	}

	const url = new URL(uri);
	if (params.count) {
		url.searchParams.append("count", params.count ? "t" : "f");
		return removeLastSlash(url.href);
	}
	if (params.fields) {
		url.searchParams.append(fields, params.fields.join(","));
	}
	if (params.lookup_id) {
		url.searchParams.append("lookup.id", params.lookup_id);
		return removeLastSlash(url.href);
	}
	if (params.cursor) {
		url.searchParams.append("cursor", params.cursor);
	}
	if (params.limit) {
		url.searchParams.append("limit", params.limit);
	}
	return removeLastSlash(url.href);
}

/** removeLastSlash deletes the last slash before the url parameters
 * added by the URL object to avoid the app from crashing */
function removeLastSlash(uri: string): string {
	const i = uri.lastIndexOf("/");
	return uri.substring(0, i) + uri.substring(i + 1);
}

/**
 * Note: All endpoints shall start with an uppercase and use camelCase.
 */
class APIEndpoints {
	Auth = new AuthEndpoints()
	Events = new EventsEndpoints()
	Notifications = new NotificationsEndpoints()
	Users = new UsersEndpoints()
}

export const API = new APIEndpoints();

export class Resources {
	events = {
		attending: "events_attending",
		invited: "events_invited",
		likes: "events_likes",
	}
}

/**
 * Fields contains the name of the url parameter used
 * to request specific fields of an object in a request
 */
export enum Fields {
	comment = "comment.fields",
	event = "event.fields",
	notification = "notification.fields",
	product = "product.fields",
	posts = "post.fields",
	user = "user.fields",
}

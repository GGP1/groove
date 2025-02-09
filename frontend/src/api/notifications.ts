import { NotificationsCount, NotificationsPag } from "../types/api";
import { Notification } from "../types/notification";
import { HTTP } from "../utils/http";
import { API_URL, buildURL, Fields, Params } from "./api";

export class NotificationsEndpoints {
	/**
	 * Answer responds to a notification request.
	 * @param id notification ID
	 * @param accepted true: accept, false: reject
	 */
	async Answer(id: string, accepted: boolean) {
		await HTTP.post({
			url: `${API_URL}/answer/${id}`,
			body: JSON.stringify(accepted),
		});
	}

	/**
	 * GetFromUser returns a user's notifications
	 * @param user_id user ID
	 * @param params notification parameters
	 * @returns Either notifications with pagination or a count of them
	 */
	async GetFromUser<T extends NotificationsPag | NotificationsCount>(user_id: string, params?: Params<Notification>): Promise<T> {
		const resp = await HTTP.get<T>({
			url: buildURL(`${API_URL}/notifications/user/${user_id}`, Fields.notification, params),
		});
		return resp;
	}
}

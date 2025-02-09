import { Login } from "../context/Session";
import { HTTP } from "../utils/http";
import { API_URL } from "./api";

export class AuthEndpoints {
	/**
	 * Login logs a user into the system.
	 * @param login authentication details
	 * @returns the user session, in this case we return
	 * a response to have a little more control
	 */
	async Login(login: Login): Promise<Response> {
		const resp = await HTTP.post({
			url: `${API_URL}/login`,
			body: JSON.stringify(login),
		});
		return resp;
	}

	/**
	 * Logout logs a user out from the system.
	 */
	async Logout() {
		await HTTP.get({ url: `${API_URL}/logout` });
	}
}

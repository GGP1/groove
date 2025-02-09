import AsyncStorage from "@react-native-async-storage/async-storage";
import * as Keychain from "react-native-keychain";
import { loginFunc, UserSession } from "../context/Session";
import { Cache } from "./cache";

class SessionStore {
	private readonly key = "user_session"

	/**
	* Save stores the user object to async storage and credentials to keychain.
	  *
	  * @param u user session object
	  * @param username username used for login
	  * @param password password used for login
	  */
	async save(u: UserSession, username: string, password: string) {
		try {
			await Keychain.setGenericPassword(username, password);
			await AsyncStorage.setItem(this.key, JSON.stringify(u));
		} catch (err) {
			console.error("failed saving credentials:", err);
		}
	}

	/**
	* Get returns the user session or undefined if it doesn't exist yet.
	*
	* @returns userSession or undefined if the user is not logged in
	*/
	async get(): Promise<UserSession | undefined> {
		const cachedSession = Cache.get<UserSession>(this.key);
		if (cachedSession) {
			return cachedSession;
		}

		try {
			const storedSession = await AsyncStorage.getItem(this.key);
			if (storedSession) {
				const session: UserSession = JSON.parse(storedSession) as UserSession;
				Cache.set(this.key, session);
				return session;
			}
		} catch (err) {
			console.error(err);
		}
	}

	/**
	* Restore attempts to log the user in by using the previously stored credentials.
	*
	* @param login login function to set the user to the context
	*/
	async restore(login: loginFunc) {
		try {
			// Get the credentials from the keychain and login the user
			const credentials = await Keychain.getGenericPassword();
			if (credentials) {
				login({
					username: credentials.username,
					password: credentials.password,
				});
			}
		} catch (err) {
			console.error("failed loading credentials:", err);
		}
	}

	/**
	* Remove deletes all the user credentials from the filesystem.
	*/
	async remove() {
		try {
			await AsyncStorage.removeItem(this.key);
			await Keychain.resetGenericPassword();
			Cache.clear();
		} catch (err) {
			console.log(err);
		}
	}
}

export const Session = new SessionStore();

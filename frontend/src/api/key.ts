import AsyncStorage from "@react-native-async-storage/async-storage";
import { Cache, CacheKey } from "../utils/cache";
import { Session } from "../utils/session";

export type APIKeyResponse = {
	id: string,
	api_key: string
}

const STORAGE_KEY = "api_key_";

export class APIKey {
	/**
	 * Get returns the user's API key.
	 * @returns API key
	 */
	static async get(): Promise<string | undefined> {
		const user = await Session.get();

		if (user) {
			const cachedAPIKey = Cache.get<string>(CacheKey.APIKey + user.id);
			if (cachedAPIKey) {
				return cachedAPIKey;
			}

			try {
				const apiKey = await AsyncStorage.getItem(STORAGE_KEY + user.id);
				if (apiKey) {
					Cache.set(CacheKey.APIKey, apiKey);
					return apiKey;
				}
			} catch (err) {
				console.log(err);
			}
		}
	}

	/**
	 * Store saves the user's API key to disk.
	 * @param userData represents the API key response
	 * @returns void
	 */
	static async store(userData: APIKeyResponse): Promise<void> {
		try {
			// Store the user's api key (use the id to differenciate several account)
			await AsyncStorage.setItem(STORAGE_KEY + userData.id, userData.api_key);
		} catch (err) {
			console.log("couldn't save the users's API key:", err);
		}

		return;
	}
}

/** A minute in milliseconds */
export const MINUTE = 60000;
/** An hour in milliseconds */
export const HOUR = MINUTE * 60;
/** A day in milliseconds */
export const DAY = HOUR * 24;

export enum CacheKey {
	APIKey = "api_key_",
	Events = "events_",
	Users = "users_",
	Roles = "roles_",
	Permissions = "permissions_",
	Zones = "zones_",
	UserLastRegion = "user_last_region",
	Query = "q_"
}

type Record = {
	value: any,
	expires: number,
}

/**
 * Cache policy
 *
 * Store information that won't get invalidated too often, if so, use a short TTL.
 *
 * Configuration and preferences, session, API key, location, etc. may be cached until logout.
 * For other information use a short TTL.
 * Do not store resources from the server as it has its own cache that is updated
 * as the records mutate.
 */
class CacheStore {
	private map: Map<string, Record> = new Map();

	/**
	 * clear resets the cache
	 */
	clear(): void {
		this.map.clear();
	}

	/**
	 * delete removes an item from the cache
	 * @param key record identifier
	 * @returns if a record was deleted or not
	 */
	delete(key: string): boolean {
		return this.map.delete(key);
	}

	/**
	 * get returns a record from the cache
	 * @param key record identifier
	 * @returns a value of type T or undefined
	 */
	get<T>(key: string): T | undefined {
		const record = this.map.get(key);
		if (!record) {
			return;
		}
		if (record.expires !== 0) {
			const now = new Date().getTime();
			if (record.expires < now) {
				this.map.delete(key);
				return;
			}
		}
		return record.value;
	}

	/**
	 * set saves a record inside the cache
	 * @param key record identifier
	 * @param value record content
	 * @param ttl time to live in milliseconds, by default it does not expire
	 */
	set(key: string, value: any, ttl?: number): void {
		if (ttl) {
			const now = new Date().getTime();
			ttl += now;
		}
		this.map.set(key, { value: value, expires: ttl ? ttl : 0 });
	}
}

/** Cache instance for storing records in-memory */
export const Cache = new CacheStore();

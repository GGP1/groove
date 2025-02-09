import { APIKey } from "../api/key";
import { ErrResponse } from "../types/api";
import { panic } from "./errors";
import { HTTPStatus } from "./httpStatus";

export interface Req {
	url: string,
	keepalive?: boolean,
	headers?: { [key: string]: string },
	signal?: AbortSignal,
}

export interface ReqWithBody extends Req {
	body?: BodyInit_
}

// TODO:
// - abort requests when a component is unmounted (abort signal)
// - retry failed requests after a certain delay
// - group calls based on the resource/service they work on and make posts to invalidate get queries
// - try to cache queries and update them if any data changes
// - handle requests abortion: https://stackoverflow.com/questions/31061838/how-do-i-cancel-an-http-fetch-request
export class HTTP {
	static async delete(req: Req): Promise<void> {
		setAPIKeyHeader(req);
		const res = await fetch(req.url, {
			method: "DELETE",
			credentials: "include",
			keepalive: req.keepalive,
			signal: req.signal,
			headers: {
				...(req.headers || {}),
			},
		});
		if (!res.ok) {
			panic(await parseErr(res));
		}

		return;
	}

	static async get<T>(req: Req): Promise<T> {
		setAPIKeyHeader(req);
		const res = await fetch(req.url, {
			method: "GET",
			credentials: "include",
			keepalive: req.keepalive,
			signal: req.signal,
			headers: {
				"Accept": "application/json",
				...(req.headers || {}),
			},
		});
		if (!res.ok) {
			panic(await parseErr(res));
		}
		if (res.status === HTTPStatus.NO_CONTENT) {
			return new Promise(() => { });
		}

		const json = await res.json();
		return <T>json;
	}

	static async post(req: ReqWithBody): Promise<Response> {
		setAPIKeyHeader(req);
		const res = await fetch(req.url, {
			method: "POST",
			credentials: "include",
			body: req.body,
			keepalive: req.keepalive,
			signal: req.signal,
			headers: {
				"Accept": "application/json",
				"Content-Type": "application/json; charset=UTF-8",
				...(req.headers || {}),
			},
		});
		if (!res.ok) {
			panic(await parseErr(res));
		}

		return res;
	}

	static async put(req: ReqWithBody): Promise<Response> {
		setAPIKeyHeader(req);
		const res = await fetch(req.url, {
			method: "PUT",
			body: req.body,
			credentials: "include",
			keepalive: req.keepalive,
			signal: req.signal,
			headers: {
				"Content-Type": "application/json; charset=UTF-8",
				...(req.headers || {}),
			},
		});
		if (!res.ok) {
			panic(await parseErr(res));
		}

		return res;
	}

	/**
	 * paralell executes multiple promises in parallel.
	 * Calling too many promises simultaneously may overload the device's memory.
	 * @param requests array of promises
	 * @returns an array of the responses received, only if all of the requests succeeded
	 */
	static async paralell(requests: PromiseLike<unknown>[]): Promise<unknown[]> {
		return await Promise.all(requests);
	}
}

// TODO: if we receive a 401 status (Unauthorized) show a login modal.
// The request body used could also be logged as well as the headers.
const parseErr = async (res: Response): Promise<string> => {
	if (res.status === HTTPStatus.NOT_FOUND) {
		return "404 - Not found";
	}
	const err = await res.json() as ErrResponse;
	return JSON.stringify({ url: res.url, message: err }, null, 4);
};

const setAPIKeyHeader = async (req: Req | ReqWithBody) => {
	const apiKey = await APIKey.get();
	if (apiKey) {
		if (req.headers) {
			req.headers["X-Api-Key"] = apiKey;
		} else {
			req.headers = { "X-Api-Key": apiKey };
		}
	}
};

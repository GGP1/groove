export function panic(message: string) {
	const e = new Error(message);
	// TODO: Send error log with JSON format to the server
	throw e;
}

import parser from "cron-parser";

// Format:
//  ┌───────────────> minutes (0-59)
//  │ ┌───────────> hours (0-23)
//  │ │ ┌─────────> month day (0-31)
//  │ │ │  ┌──────> year month (0-12)
//  │ │ │  │  ┌───> week day (0-6, 0=sunday)
//  │ │ │  │  │  ┌> duration (1-∞)
//  m h md ym wd d
// Values may contain (*) meaning any, (-) for ranges, (L) for last times or (,) for concatenations.
//
// Non-numeric characters shouldn't be allowed in minutes and hours
//
// const [date, d] = Cron.parse("14 15 1-15 11 * 91");
// const listItems = date.iterate(1).map((cron, i) => (
// 	<Text key={i}>
// 		{new Date(cron.toDate().getTime() + Cron.minToMilli(d)).toJSON()}
// 	</Text>
// ));

export class Cron {
	/**
	 * @param cron cron-formatted string
	 * @returns a cron parser expression and the duration
	 *
	 * TODO: create a personalized and efficient parser meeting our requirements.
	 * Format: m h md ym wd d
	 */
	static parse(startDate: Date, endDate: Date, cron: string): [parser.CronExpression, number] {
		const parts = cron.split(" ");
		if (parts.length !== 6) {
			throw new Error("invalid cron");
		}

		const d: number = Number(parts[5]);
		const expr: string = parts.slice(0, 5).join(" ");
		const opts: parser.ParserOptions = {
			utc: false,
			startDate: startDate,
			currentDate: Date.now(),
			endDate: endDate,
		};
		return [parser.parseExpression(expr, opts), d];
	}

	/**
	 * @param d duration in minutes
	 * @returns duration in milliseconds
	 */
	static minToMilli(d: number): number {
		return d * 60000;
	}

	// TODO: make a build function or "add methods" (add minutes, add hours, etc) and
	// build the cron that way

	/**
	 * TODO: describe the cron with words to display it to the user.
	 * Describes what the cron represents in plain english.
	 * @param cron cron-formatted string
	 */
	// static Description(cron: string): string {

	// }
}

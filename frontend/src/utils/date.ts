const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
];
const WEEK_DAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

type CustomDate = {
	minutes: string,
	hours: string,
	day: number,
	month: number,
	year: number,
}

function formatDate(date: Date): CustomDate {
	let d = new Date(date),
		month = d.getMonth() + 1,
		day = d.getDate(),
		year = d.getFullYear(),
		hours = "" + d.getHours(),
		minutes = "" + d.getMinutes();

	if (minutes.length === 1) {
		minutes = "0" + minutes;
	}
	if (hours.length === 1) {
		hours = "0" + hours;
	}

	return {
		minutes: minutes,
		hours: hours,
		day: day,
		month: month,
		year: year,
	};
}

/**
 * Format: `16:00 · 16 Dec 2022`
 * @param date Date
 * @returns Date string representation
 */
export function formatPostDate(date: Date): string {
	const d = formatDate(date);
	// const ante_meridiem = d.hours < "12" ? "AM" : "PM";
	const first = [d.hours, d.minutes].join(":");
	const second = [d.day, MONTHS[d.month - 1], d.year].join(" ");

	return [first, second].join(" · ");
}

/**
 * Format: `16/12/2022, 16:00`
 * @param date Date
 * @returns Date string representation
 */
export function formatDateNumbers(date: Date): string {
	const d = formatDate(date);
	const first = [d.day, d.month, d.year].join("/");
	const second = [d.hours, d.minutes].join(":");

	return [first, second].join(", ");
}

/**
 * Format: `Thu 16 Dec, 16:00`
 * @param date Date
 * @returns Date string representation
 */
export function formatDateNames(date: Date): string {
	const d = formatDate(date);
	const first = [WEEK_DAYS[date.getDay() - 1], d.day, MONTHS[d.month - 1]].join(" ");
	const second = [d.hours, d.minutes].join(":");

	return [first, second].join(", ");
}

export function dayAndMonth(date: Date): string {
	return `${date.getDate()} ${MONTHS[date.getMonth()]}`;
}

import { Comment } from "../types/comment";
import { Event, TicketType, Type as EventType } from "../types/event";
import { Notification, Type as NotifType } from "../types/notification";
import { Permission } from "../types/permission";
import { Post } from "../types/post";
import { Role } from "../types/role";
import { Ticket } from "../types/ticket";
import { Invitations, Type, User } from "../types/user";
import { Zone } from "../types/zone";

export const MOCK_EVENTS: Event[] = [
	{
		id: "01FEBSN6W3GNYXDY2T4831Q7GQ",
		name: "city inauguration",
		type: EventType.Conference,
		ticket_type: TicketType.Free,
		public: false,
		logo_url: "https://pbs.twimg.com/profile_images/1172477872826703872/CoPRGFVH_400x400.jpg",
		header_url: "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fdevelopns.ca%2Fwp-content%2Fuploads%2F2020%2F06%2FFB-event-header-s1-1024x576.jpg&f=1&nofb=1",
		min_age: 1,
		virtual: false,
		cron: "4 22 * * * 654",
		start_date: new Date(),
		end_date: new Date("2050-05-27"),
		slots: 12,
		location: {
			address: "Tubulu, Mali",
			coordinates: {
				latitude: 36.720465,
				longitude: -4.369885,
			},
		},
		hide_roles: true,
		created_at: new Date("2021-11-10T10:09:15Z"),
	},
	{
		id: "01FEBSN6W3GNYXDI2T4801C7LQ",
		name: "2019 photo contest",
		type: EventType.Graduation,
		ticket_type: TicketType.Paid,
		public: false,
		header_url: "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fhistoricdenver.org%2Fwp-content%2Fuploads%2F2019%2F03%2FFB-event-header.jpg&f=1&nofb=1",
		min_age: 0,
		virtual: false,
		cron: "4 22 * * * 654",
		start_date: new Date(),
		end_date: new Date("2050-05-27"),
		slots: 300,
		location: {
			address: "Frankfurt, Germany",
			coordinates: {
				latitude: 4.54978,
				longitude: -15.14895,
			},
		},
		hide_roles: false,
		created_at: new Date("2021-11-10T10:09:15Z"),
	},
	{
		id: "01FDV0J3ZE6B2ER1PGS69GDN8N",
		name: "Glow night",
		type: EventType.Anniversary,
		ticket_type: TicketType.Paid,
		public: true,
		header_url: "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fwww.psdmarket.net%2Fwp-content%2Fuploads%2F2019%2F06%2Fglow_night_party_flyer_psd_psdmarket_1.jpg&f=1&nofb=1",
		min_age: 16,
		virtual: false,
		cron: "47 11 8 6 1 147",
		start_date: new Date(),
		end_date: new Date("2050-04-27"),
		slots: 22,
		location: {
			address: "Málaga, Spain",
			coordinates: {
				latitude: 36.726171,
				longitude: -4.410045,
			},
		},
		hide_roles: false,
		created_at: new Date("2021-11-10T10:09:15Z"),
	},
	{
		id: "01FDV0J3ZE6B2ER1PGS69GDR6Y",
		name: "Gastby Party",
		type: EventType.GrandPrix,
		ticket_type: TicketType.Donation,
		public: true,
		header_url: "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fwww.psdmarket.net%2Fwp-content%2Fuploads%2F2019%2F10%2Fgatsby_party_event_flyer_psd_psdmarket_1.jpg&f=1&nofb=1",
		min_age: 0,
		virtual: false,
		cron: "30 14 1 1 * 60",
		start_date: new Date(),
		end_date: new Date("2050-08-27"),
		slots: 25000,
		location: {
			address: "Mónaco",
			coordinates: {
				latitude: 43.740110,
				longitude: 7.429155,
			},
		},
		hide_roles: false,
		created_at: new Date("2021-11-10T10:09:15Z"),
	},
];

export const MOCK_USERS: User[] = [
	{
		id: "01FM4TY5W95MJ13C4Q10T106MT",
		birth_date: new Date(),
		created_at: new Date(),
		description: "asd",
		email: "dsa",
		invitations: Invitations.Nobody,
		name: "Esteban",
		private: true,
		profile_image_url: "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Ftse1.mm.bing.net%2Fth%3Fid%3DOIP.WTRBk5aF6HCwJaBW88eLYwHaLC%26pid%3DApi&f=1",
		username: "cage",
		type: Type.Personal,
	},
	{
		id: "01FM4V0BSCXNEE9XGVYFKRY7Y8",
		birth_date: new Date(),
		created_at: new Date(),
		description: "asd",
		email: "dsa",
		invitations: Invitations.Nobody,
		name: "Tomas",
		private: true,
		profile_image_url: "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fwallpapercave.com%2Fwp%2Fwp2890235.jpg&f=1&nofb=1",
		username: "box",
		type: Type.Business,
	},
];

export const MOCK_POSTS: Post[] = [
	{
		id: "01FM4TW8BEC0B8XF4J7V9J5ZAK",
		likes_count: 5,
		content: "Post content",
		event_id: "01FK8MNJGDF2FR8JR5E9PB364Q",
		comments_count: 1,
		created_at: new Date(),
		media: ["", ""],
	},
	{
		id: "01FM4TVEF60J2CJCG2DTPWJABN",
		likes_count: 15,
		content: "Post content 2",
		event_id: "01FK8MGAQ63MXA2P35AXBK1VPX",
		comments_count: 0,
		created_at: new Date(),
		media: [""],
	},
	{
		id: "01FM4TVB1AD56YT2XTJYRS735B",
		likes_count: 313,
		content: "Hello friends, my name is Jess and I'm a DJ living in Vancouver, Canada. I fell in love with DJing a few years ago and decided to share my mixes through streaming. My favourite genre to play is house music but I hope to broaden my musical horizons in the future.",
		event_id: "01FK8MNJGDF2FR8JR5E9PB364Q",
		comments_count: 21,
		created_at: new Date(),
		media: [""],
	},
	{
		id: "01FM4TV71XXHYH2S730XWQ9R8D",
		likes_count: 313,
		content: "Hello friends, my name is Jess and I'm a DJ living in Vancouver, Canada. I fell in love with DJing a few years ago and decided to share my mixes through streaming. My favourite genre to play is house music but I hope to broaden my musical horizons in the future.",
		event_id: "01FK8MNJGDF2FR8JR5E9PB364Q",
		comments_count: 21,
		created_at: new Date(),
		media: [""],
	},
	{
		id: "01FM4TTYY2K3APTC0VJYHKADB9",
		likes_count: 313,
		content: "Hello friends, my name is Jess and I'm a DJ living in Vancouver, Canada. I fell in love with DJing a few years ago and decided to share my mixes through streaming. My favourite genre to play is house music but I hope to broaden my musical horizons in the future.",
		event_id: "01FK8MNJGDF2FR8JR5E9PB364Q",
		comments_count: 21,
		created_at: new Date(),
		media: [""],
	},
];

export const MOCK_COMMENTS: Comment[] = [
	{
		id: "01FM4TTR92EHSZJYDYE90SY1HT",
		post_id: "01FM4TTYY2K3APTC0VJYHKADB9",
		content: "Absolutely lovely Jess!",
		user_id: "01FM4TY5W95MJ13C4Q10T106MT",
		replies_count: 1,
		likes_count: 15,
		replies: [
			{
				id: "01FM4TZN95GREFRR0J6DPNHNQ2",
				parent_comment_id: "01FM4TTR92EHSZJYDYE90SY1HT",
				content: "Agreed!",
				user_id: "01FM4V0BSCXNEE9XGVYFKRY7Y8",
				replies_count: 0,
				likes_count: 1,
				replies: [],
				created_at: new Date(),
			},
		],
		created_at: new Date(),
	},
	{
		id: "01FM4V8KWA5RJE5R2SF3Y9G52E",
		post_id: "01FM4TTYY2K3APTC0VJYHKADB9",
		content: "Let's groove!",
		user_id: "01FM4TY5W95MJ13C4Q10T106MT",
		replies_count: 0,
		likes_count: 8,
		replies: [],
		created_at: new Date(),
	},
];

export const MOCK_ROLES: Role[] = [
	{
		name: "host",
		permission_keys: ["*"],
	},
	{
		name: "viewer",
		permission_keys: ["view_event"],
	},
	{
		name: "mechanic",
		permission_keys: ["access_box"],
	},
];

export const MOCK_PERMISSIONS: Permission[] = [
	{
		key: "host",
		name: "Host",
		description: "Event host",
		created_at: new Date(),
	},
	{
		key: "view_event",
		name: "Viewer",
		description: "Event viewer",
		created_at: new Date(),
	},
	{
		key: "access_box",
		name: "Mechanic",
		description: "Team mechanic",
		created_at: new Date(),
	},
];

export const MOCK_ZONES: Zone[] = [
	{
		name: "Box",
		required_permission_keys: ["access_box"],
	},
	{
		name: "FIA department",
		required_permission_keys: ["fia"],
	},
];

export const MOCK_TICKETS: Ticket[] = [
	{
		name: "Standard",
		available_count: 40000,
		cost: 2000,
		linked_role: "attendant",
	},
	{
		name: "VIP",
		available_count: 4000,
		cost: 5000,
		linked_role: "attendant_vip",
	},
];

export const MOCK_NOTIFICATIONS: Notification[] = [
	{
		id: "01FM4V8ZFGCMCFDV6BJVZ1228S",
		content: "Hi box I want to invite you to a Grand Prix",
		seen: false,
		type: NotifType.Invitation,
		sender_id: "01FM4TY5W95MJ13C4Q10T106MT",
		receiver_id: "01FM4V0BSCXNEE9XGVYFKRY7Y8",
		created_at: new Date(),
		event_id: "01FDV0J3ZE6B2ER1PGS69GDR6Y",
	},
];

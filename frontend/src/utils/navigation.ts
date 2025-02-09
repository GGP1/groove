import { RouteProp } from "@react-navigation/native";
import { StackNavigationProp } from "@react-navigation/stack";
import { Props } from "../components/List";
import { Event } from "../types/event";
import { Permission } from "../types/permission";
import { Role } from "../types/role";
import { Ticket } from "../types/ticket";
import { User } from "../types/user";
import { Zone } from "../types/zone";

interface ID {
	id: string
}
interface Name {
	name: string
}

// Identifier contains either an id or a name, it's used for typing the objects that
// are sent to the "List" screen through navigation.
type Identifier = ID | Name

/**
 * CommonNavList contains the properties of a screen list that are shared among many stacks.
 */
type CommonNavList<T extends Identifier> = {
	Comment: { comment_id: string, event_id: string, user_id: string },
	Event: { event_id: string, item?: Event },
	List: { title: string, props: Props<T> },
	Post: { post_id: string, event_id: string, event_name: string, event_logo_url?: string },
	User: { user_id: string, item?: User },
	CreatePermission: { event_id: string },
	CreatePost: { event_id: string },
	CreateRole: { event_id: string },
	CreateTicket: { event_id: string },
	CreateZone: { event_id: string },
	UpdatePermission: { event_id: string, permission: Permission },
	UpdateRole: { event_id: string, role: Role },
	UpdateTicket: { event_id: string, ticket: Ticket },
	UpdateZone: { event_id: string, zone: Zone },
	UpdateEvent: { event: Event },
}

export type CommonNavProps<T extends keyof CommonNavList<any>> =
	EventsNavProps<T> | ExploreNavProps<T> | HomeNavProps<T> | NotificationsNavProps<T>

// Auth
export type AuthNavList = {
	Login: undefined
	Register: undefined
}

export type AuthNavProps<T extends keyof AuthNavList> = {
	navigation: StackNavigationProp<AuthNavList, T>
	route: RouteProp<AuthNavList, T>
}

// Events
export type EventsNavList<T extends Identifier> = {
	Events: undefined,
	CreateEvent: undefined,
} & CommonNavList<T>

export type EventsNavProps<T extends keyof EventsNavList<any>> = {
	navigation: StackNavigationProp<EventsNavList<any>, T>
	route: RouteProp<EventsNavList<any>, T>
}

// Explore
export type ExploreNavList<T extends Identifier> = {
	Explore: undefined,
	Map: undefined,
} & CommonNavList<T>

export type ExploreNavProps<T extends keyof ExploreNavList<any>> = {
	navigation: StackNavigationProp<ExploreNavList<any>, T>
	route: RouteProp<ExploreNavList<any>, T>
}

// Home
export type HomeNavList<T extends Identifier> = {
	Home: undefined,
} & CommonNavList<T>

export type HomeNavProps<T extends keyof HomeNavList<any>> = {
	navigation: StackNavigationProp<HomeNavList<any>, T>
	route: RouteProp<HomeNavList<any>, T>
}

// MyProfile
export type MyProfileNavList<T extends Identifier> = {
	MyProfile: undefined,
	UpdateUser: { user: User },
} & CommonNavList<T>

export type MyProfileNavProps<T extends keyof MyProfileNavList<any>> = {
	navigation: StackNavigationProp<MyProfileNavList<any>, T>
	route: RouteProp<MyProfileNavList<any>, T>
}

// Notifications
export type NotificationsNavList<T extends Identifier> = {
	Notifications: undefined,
} & CommonNavList<T>

export type NotificationsNavProps<T extends keyof NotificationsNavList<any>> = {
	navigation: StackNavigationProp<NotificationsNavList<any>, T>
	route: RouteProp<NotificationsNavList<any>, T>
}

export type EventsNavigationProps = StackNavigationProp<EventsNavList<any>>
export type ExploreNavigationProps = StackNavigationProp<ExploreNavList<any>>
export type HomeNavigationProps = StackNavigationProp<HomeNavList<any>>
export type THomeNavigationProps<T extends Identifier> = StackNavigationProp<HomeNavList<T>>
export type MyProfileNavigationProps = StackNavigationProp<MyProfileNavList<any>>
export type NotificationsNavigationProps = StackNavigationProp<NotificationsNavList<any>>

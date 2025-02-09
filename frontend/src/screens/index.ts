import { App as Login } from "./auth/Login";
import { App as Register } from "./auth/Register";
import { App as CreateEvent } from "./event/CreateEvent";
import { App as Event } from "./event/Event";
import { App as Events } from "./event/Events";
import { App as Comment } from "./event/post/Comment";
import { App as CreatePost } from "./event/post/CreatePost";
import { App as Post } from "./event/post/Post";
import { App as CreatePermission } from "./event/role/CreatePermission";
import { App as CreateRole } from "./event/role/CreateRole";
import { App as UpdatePermission } from "./event/role/UpdatePermission";
import { App as UpdateRole } from "./event/role/UpdateRole";
import { App as CreateTicket } from "./event/ticket/CreateTicket";
import { App as UpdateTicket } from "./event/ticket/UpdateTicket";
import { App as UpdateEvent } from "./event/UpdateEvent";
import { App as CreateZone } from "./event/zone/CreateZone";
import { App as UpdateZone } from "./event/zone/UpdateZone";
import { App as Explore } from "./Explore";
import { App as Home } from "./Home";
import { App as List } from "./List";
import { App as Notifications } from "./notification/Notifications";
import { App as UpdateUser } from "./user/UpdateUser";
import { App as User } from "./user/User";

export {
	Register,
	Login,
	Home,
	Explore,
	Event,
	User,
	Post,
	Notifications,
	Events,
	List,
	Comment,
	CreateEvent,
	UpdateEvent,
	CreateRole,
	UpdateRole,
	CreatePermission,
	UpdatePermission,
	CreateZone,
	UpdateZone,
	UpdateUser,
	CreatePost,
	CreateTicket,
	UpdateTicket,
};

